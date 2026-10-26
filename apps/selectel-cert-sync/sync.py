"""Sync Kubernetes TLS Secrets to Selectel CDN and Certificate Manager."""

import argparse
import base64
import configparser
import hashlib
import json
import logging
import os
from pathlib import Path
import re
import ssl
import sys
import time
from datetime import datetime, timedelta, timezone
from urllib import error, request
from urllib.parse import quote, urlencode
from uuid import UUID

from cryptography import x509
from cryptography.hazmat.primitives import serialization


LOG = logging.getLogger("selectel-cert-sync")
CDN_URL = "https://api.selectel.ru/cdn/v3"
MANAGER_URL = "https://cloud.api.selcloud.ru/certificate-manager/v1"
AUTH_URL = "https://cloud.api.selcloud.ru/identity/v3/auth/tokens"
ANNOTATION_PREFIX = "selectel-cert-sync.glants.xyz/"


class SyncError(Exception):
    """An error safe to report without disclosing API payloads or secrets."""


class NoRedirect(request.HTTPRedirectHandler):
    def redirect_request(self, req, fp, code, msg, headers, newurl):
        # Never forward credentials or a certificate private key to another URL.
        return None


class API:
    def __init__(self, token, base_url=CDN_URL):
        self.token = token
        self.base_url = base_url

    @staticmethod
    def http(method, url, payload=None, token=None, raw=False):
        headers = {"Content-Type": "application/json", "Accept": "application/json"}
        if token:
            headers["X-Auth-Token"] = token
        body = json.dumps(payload).encode() if payload is not None else None
        req = request.Request(url, data=body, headers=headers, method=method)
        try:
            with request.build_opener(NoRedirect()).open(req, timeout=30) as response:
                data = response.read()
                return (data if raw else (json.loads(data) if data else {})), response.headers
        except error.HTTPError as exc:
            # Bodies can echo request fields (including private keys).
            raise SyncError(f"Selectel {method} returned HTTP {exc.code}") from None
        except (error.URLError, TimeoutError, OSError, ValueError):
            raise SyncError(f"Selectel {method} failed: transport or JSON error") from None

    @classmethod
    def authenticate(cls, env):
        if env.get("SELECTEL_CLOUD_CONFIG_FILE"):
            try:
                config = configparser.ConfigParser(interpolation=None)
                config.read_string(Path(env["SELECTEL_CLOUD_CONFIG_FILE"]).read_text())
                account = config["Global"]
                credential = {"id": account["application-credential-id"],
                              "secret": account["application-credential-secret"]}
                if not all(credential.values()):
                    raise ValueError
            except Exception:
                raise SyncError("Cannot read application credentials from Selectel cloud config") from None
            payload = {"auth": {"identity": {"methods": ["application_credential"],
                                             "application_credential": credential}}}
            # Application credentials already specify the project; no scope is sent.
            _, headers = cls.http("POST", AUTH_URL, payload)
            token = headers.get("X-Subject-Token")
            if not token:
                raise SyncError("Selectel authentication returned no project token")
            return cls(token)
        required = ["SELECTEL_USERNAME", "SELECTEL_PASSWORD", "SELECTEL_ACCOUNT_ID", "SELECTEL_PROJECT_NAME"]
        missing = [name for name in required if not env.get(name)]
        if missing:
            raise SyncError("Missing configuration: " + ", ".join(missing))
        domain = {"name": env["SELECTEL_ACCOUNT_ID"]}
        payload = {"auth": {
            "identity": {"methods": ["password"], "password": {"user": {
                "name": env["SELECTEL_USERNAME"], "password": env["SELECTEL_PASSWORD"],
                "domain": domain,
            }}},
            "scope": {"project": {"name": env["SELECTEL_PROJECT_NAME"], "domain": domain}},
        }}
        _, headers = cls.http("POST", AUTH_URL, payload)
        token = headers.get("X-Subject-Token")
        if not token:
            raise SyncError("Selectel authentication returned no project token")
        return cls(token)

    def call(self, method, path, payload=None):
        result, _ = self.http(method, self.base_url + path, payload, self.token)
        # CDN certificate operations use numeric status in addition to HTTP.
        if not isinstance(result, dict) or type(result.get("status")) is not int or result["status"] != 0:
            raise SyncError("Selectel certificate operation failed or returned an unknown response")
        if "data" not in result:
            raise SyncError("Selectel certificate response has no data")
        return result["data"]

    def manager_call(self, method, path, payload=None, raw=False):
        # Certificate Manager uses HTTP statuses and raw PEM, not CDN envelopes.
        result, _ = self.http(method, MANAGER_URL + path, payload, self.token, raw=raw)
        return result


def load_material(directory, min_valid_hours=24):
    """Read one Kubernetes projected-volume generation, avoiding mixed keypairs."""
    try:
        directory = Path(directory)
        generation = directory / "..data"
        root = generation.resolve(strict=True) if generation.is_symlink() else directory
        cert_bytes = (root / "tls.crt").read_bytes()
        key_bytes = (root / "tls.key").read_bytes()
        return parse_material(cert_bytes, key_bytes, min_valid_hours)
    except SyncError:
        raise
    except Exception:
        raise SyncError("Cannot read mounted TLS certificate/key") from None


def parse_material(cert_bytes, key_bytes, min_valid_hours=24):
    """Validate and normalize a keypair in memory; never write secret data to disk."""
    try:
        chain = x509.load_pem_x509_certificates(cert_bytes)
        key = serialization.load_pem_private_key(key_bytes, password=None)
        pub_args = (serialization.Encoding.DER, serialization.PublicFormat.SubjectPublicKeyInfo)
        if key.public_key().public_bytes(*pub_args) != chain[0].public_key().public_bytes(*pub_args):
            raise SyncError("TLS certificate and private key do not match")
        now = datetime.now(timezone.utc)
        for cert in chain:
            if cert.not_valid_before_utc > now:
                raise SyncError("TLS certificate chain is not valid yet")
            if cert.not_valid_after_utc <= now + timedelta(hours=min_valid_hours):
                raise SyncError("TLS certificate chain expires too soon")
        for child, issuer in zip(chain, chain[1:]):
            child.verify_directly_issued_by(issuer)
        # Normalize PEM and convert RSA/EC traditional keys to Selectel's PKCS#8.
        certificate = b"".join(c.public_bytes(serialization.Encoding.PEM) for c in chain).decode()
        private_key = key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.PKCS8,
                                        serialization.NoEncryption()).decode()
        digest = hashlib.sha256(certificate.encode()).hexdigest()
        return certificate, private_key, digest, chain[0].not_valid_after_utc
    except SyncError:
        raise
    except Exception:
        raise SyncError("Cannot validate TLS certificate/key/chain") from None


def parse_stores(raw):
    stores = tuple(dict.fromkeys(s.strip() for s in raw.split(",") if s.strip()))
    if not stores or any(s not in ("cdn", "certificate-manager") for s in stores):
        raise SyncError("Sync stores must be cdn, certificate-manager, or both")
    return stores


def validate_name(name):
    if not re.fullmatch(r"[a-zA-Z0-9_.-]{1,80}", name):
        raise SyncError("Sync name must contain 1-80 letters, digits, dots, underscores or hyphens")
    return name


class Kubernetes:
    """Namespace-scoped discovery using the pod's service-account identity."""

    def __init__(self):
        self.root = Path("/var/run/secrets/kubernetes.io/serviceaccount")
        try:
            self.context = ssl.create_default_context(cafile=str(self.root / "ca.crt"))
        except OSError:
            raise SyncError("Cannot load Kubernetes service-account CA") from None
        self.base_url = "https://kubernetes.default.svc"

    def get(self, path, metadata_only=False):
        accept = ("application/json;as=PartialObjectMetadataList;g=meta.k8s.io;v=v1"
                  if metadata_only else "application/json")
        try:
            token = (self.root / "token").read_text().strip()
            req = request.Request(self.base_url + path,
                                  headers={"Authorization": "Bearer " + token, "Accept": accept})
            opener = request.build_opener(NoRedirect(), request.HTTPSHandler(context=self.context))
            with opener.open(req, timeout=30) as response:
                return json.load(response)
        except error.HTTPError as exc:
            raise SyncError(f"Kubernetes read returned HTTP {exc.code}") from None
        except Exception:
            raise SyncError("Kubernetes read failed") from None

    def selected_secrets(self, namespaces):
        for namespace in namespaces:
            continuation = ""
            while True:
                query = {"limit": 500}
                if continuation:
                    query["continue"] = continuation
                path = f"/api/v1/namespaces/{quote(namespace, safe='')}/secrets"
                page = self.get(path + "?" + urlencode(query), metadata_only=True)
                if not isinstance(page, dict) or page.get("kind") != "PartialObjectMetadataList":
                    raise SyncError("Kubernetes did not return a Secret metadata list")
                for item in page.get("items", []):
                    meta = item["metadata"]
                    if meta.get("deletionTimestamp"):
                        continue
                    if meta.get("annotations", {}).get(ANNOTATION_PREFIX + "enabled") != "true":
                        continue
                    secret = self.get(path + "/" + quote(meta["name"], safe=""))
                    current = secret.get("metadata", {})
                    # Recheck opt-in after GET in case discovery raced an annotation edit.
                    if (current.get("deletionTimestamp") or
                            current.get("annotations", {}).get(ANNOTATION_PREFIX + "enabled") != "true"):
                        continue
                    yield namespace, secret
                continuation = page.get("metadata", {}).get("continue", "")
                if not continuation:
                    break


def discovered_jobs(kube, namespaces, cluster_name):
    jobs, seen_names = [], set()
    for namespace, secret in kube.selected_secrets(namespaces):
        meta = secret["metadata"]
        source = namespace + "/" + meta["name"]
        annotations = meta.get("annotations", {})
        if secret.get("type") != "kubernetes.io/tls":
            raise SyncError(f"Selected Secret {source} must have type kubernetes.io/tls")
        default_name = "k8s-" + hashlib.sha256((cluster_name + "/" + source).encode()).hexdigest()[:32]
        name = validate_name(annotations.get(ANNOTATION_PREFIX + "name", default_name))
        if name in seen_names:
            raise SyncError("Multiple selected Secrets have the same sync name")
        seen_names.add(name)
        stores = parse_stores(annotations.get(ANNOTATION_PREFIX + "stores", "certificate-manager"))
        jobs.append((source, secret, name, stores))
    return jobs


def run_discovery(kube, env, dry_run=False):
    namespaces = tuple(dict.fromkeys(n.strip() for n in env.get("SYNC_NAMESPACES", "").split(",") if n.strip()))
    if not namespaces or any(not re.fullmatch(r"[a-z0-9]([-a-z0-9]*[a-z0-9])?", n) or len(n) > 63 for n in namespaces):
        raise SyncError("SYNC_NAMESPACES must explicitly list allowed Kubernetes namespaces")
    cluster = env.get("CLUSTER_NAME", "")
    if not cluster:
        raise SyncError("CLUSTER_NAME is required for stable certificate ownership")
    jobs = discovered_jobs(kube, namespaces, cluster)
    if not jobs:
        LOG.info("No opted-in TLS Secrets found")
        return 0
    api = API.authenticate(env)
    failed = False
    for source, secret, name, stores in jobs:
        try:
            try:
                data = secret["data"]
                material = parse_material(base64.b64decode(data["tls.crt"], validate=True),
                                          base64.b64decode(data["tls.key"], validate=True))
            except (KeyError, ValueError, TypeError):
                raise SyncError("Selected TLS Secret has missing or invalid data") from None
            LOG.info("source=%s sync_name=%s", source, name)
            if not sync_destinations(api, name, material, stores, dry_run):
                failed = True
        except SyncError as exc:
            LOG.error("source=%s error=%s", source, exc)
            failed = True
    return 1 if failed else 0


def positive_seq(value):
    if type(value) is not int or value <= 0:
        raise SyncError("Selectel returned an invalid certificate ID")
    return value


def reconcile(api, name, material, dry_run=False):
    certificate, private_key, digest, expires = material
    prefix = f"cert-sync:{name}:sha256:"
    marker = prefix + digest
    entries = api.call("GET", "/certificate")
    if not isinstance(entries, list) or not all(isinstance(e, dict) for e in entries):
        raise SyncError("Selectel returned an invalid certificate list")
    matches = [e for e in entries if str(e.get("comment", "")).startswith(prefix)]
    if len(matches) > 1:
        raise SyncError("Multiple certificates have this sync name; resolve duplicates before continuing")
    existing = matches[0] if matches else None
    seq = positive_seq(existing.get("seq")) if existing else None
    action = "unchanged" if existing and existing.get("comment") == marker else ("update" if existing else "create")

    if action != "unchanged" and not dry_run:
        payload = {"certificate": certificate, "private_key": private_key, "comment": marker}
        path = f"/certificate/{seq}" if seq else "/certificate"
        result = api.call("PATCH" if seq else "POST", path, payload)
        if not isinstance(result, dict):
            raise SyncError("Selectel returned an invalid certificate write response")
        returned_seq = positive_seq(result.get("seq"))
        if seq is not None and returned_seq != seq:
            raise SyncError("Selectel returned a different certificate ID after update")
        seq = returned_seq

    LOG.info("action=%s dry_run=%s certificate_id=%s sha256=%s expires=%s",
             action, dry_run, seq, digest, expires.isoformat())
    if expires < datetime.now(timezone.utc) + timedelta(days=14):
        LOG.warning("Source certificate expires in less than 14 days; check cert-manager renewal")
    return seq, action


def manager_id(value):
    try:
        if not isinstance(value, str):
            raise ValueError
        return str(UUID(value))
    except ValueError:
        raise SyncError("Certificate Manager returned an invalid certificate UUID") from None


def reconcile_manager(api, name, material, dry_run=False):
    certificate, private_key, digest, expires = material
    remote_name = "cert-sync:" + name
    entries = api.manager_call("GET", "/certs")
    if not isinstance(entries, list) or not all(isinstance(e, dict) for e in entries):
        raise SyncError("Certificate Manager returned an invalid certificate list")
    matches = [e for e in entries if e.get("name") == remote_name]
    if len(matches) > 1:
        raise SyncError("Multiple Certificate Manager certificates have this sync name")
    cert_id = manager_id(matches[0].get("id")) if matches else None
    action = "create"
    if cert_id:
        chain = api.manager_call("GET", f"/cert/{cert_id}/ca_chain", raw=True)
        try:
            remote_chain = x509.load_pem_x509_certificates(chain)
            normalized = b"".join(c.public_bytes(serialization.Encoding.PEM) for c in remote_chain)
        except Exception:
            raise SyncError("Cannot validate Certificate Manager public chain") from None
        action = "unchanged" if hashlib.sha256(normalized).hexdigest() == digest else "update"
    if action != "unchanged" and not dry_run:
        chain = x509.load_pem_x509_certificates(certificate.encode())
        payload = {"pem": {
            "certificates": [c.public_bytes(serialization.Encoding.PEM).decode() for c in chain],
            "private_key": private_key,
        }}
        if cert_id:
            # Version updates retain the UUID and existing consumer associations.
            api.manager_call("POST", f"/cert/{cert_id}", payload)
        else:
            result = api.manager_call("POST", "/certs", {"name": remote_name, **payload})
            if not isinstance(result, dict):
                raise SyncError("Certificate Manager returned an invalid upload response")
            cert_id = manager_id(result.get("id"))
    LOG.info("store=certificate-manager action=%s dry_run=%s certificate_id=%s sha256=%s expires=%s",
             action, dry_run, cert_id, digest, expires.isoformat())
    if expires < datetime.now(timezone.utc) + timedelta(days=14):
        LOG.warning("Source certificate expires in less than 14 days; check cert-manager renewal")
    return cert_id, action


def reconcile_with_retry(api, name, material, dry_run=False, store="cdn"):
    # Read remote state on each retry to recover partial success.
    for attempt in range(3):
        try:
            if store == "certificate-manager":
                return reconcile_manager(api, name, material, dry_run)
            if store != "cdn":
                raise SyncError("Unknown sync store")
            return reconcile(api, name, material, dry_run)
        except SyncError:
            if attempt == 2:
                raise
            LOG.warning("Reconciliation failed; rereading remote state before retry")
            time.sleep(5 * (attempt + 1))


def sync_destinations(api, name, material, stores, dry_run=False):
    success = True
    for store in stores:
        try:
            if store == "cdn":
                reconcile_with_retry(api, name, material, dry_run)
            else:
                reconcile_with_retry(api, name, material, dry_run, store=store)
        except SyncError as exc:
            LOG.error("sync_name=%s store=%s error=%s", name, store, exc)
            success = False
    return success


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--dry-run", action="store_true", help="Authenticate and read Selectel; do not write")
    parser.add_argument("--discover", action="store_true", help="Sync annotation-selected Secrets in SYNC_NAMESPACES")
    args = parser.parse_args()
    logging.basicConfig(level=logging.INFO, format="%(levelname)s %(message)s")
    try:
        if args.discover:
            return run_discovery(Kubernetes(), os.environ, args.dry_run)
        name = validate_name(os.environ.get("SYNC_NAME", ""))
        stores = parse_stores(os.environ.get("SYNC_STORES", "certificate-manager"))
        material = load_material(os.environ.get("TLS_DIRECTORY", "/tls"))
        api = API.authenticate(os.environ)
        return 0 if sync_destinations(api, name, material, stores, args.dry_run) else 1
    except SyncError as exc:
        LOG.error("%s", exc)
        return 1


if __name__ == "__main__":
    sys.exit(main())
