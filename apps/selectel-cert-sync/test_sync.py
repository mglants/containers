import contextlib
import base64
from datetime import datetime, timedelta, timezone
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import Mock, patch
from urllib.error import HTTPError

from cryptography import x509
from cryptography.hazmat.primitives import hashes, serialization
from cryptography.hazmat.primitives.asymmetric import rsa
from cryptography.x509.oid import NameOID

import sync


def issue(days=90, starts=-1, issuer=None):
    key = rsa.generate_private_key(public_exponent=65537, key_size=2048)
    subject = x509.Name([x509.NameAttribute(NameOID.COMMON_NAME, "cdn.example.com")])
    now = datetime.now(timezone.utc)
    builder = (x509.CertificateBuilder().subject_name(subject)
               .issuer_name(issuer[0].subject if issuer else subject)
               .public_key(key.public_key()).serial_number(x509.random_serial_number())
               .not_valid_before(now + timedelta(days=starts))
               .not_valid_after(now + timedelta(days=days))
               .add_extension(x509.SubjectAlternativeName([x509.DNSName("cdn.example.com")]), False))
    cert = builder.sign(issuer[1] if issuer else key, hashes.SHA256())
    return cert, key


def pem(cert, key):
    return (cert.public_bytes(serialization.Encoding.PEM),
            key.private_bytes(serialization.Encoding.PEM, serialization.PrivateFormat.TraditionalOpenSSL,
                              serialization.NoEncryption()))


class FakeAPI:
    def __init__(self):
        self.entries = []
        self.writes = []
        self.resource = {"certificate": None, "names": ["example.selcdn.net", "cdn.example.com"],
                         "cdn_domain": "example.selcdn.net"}
        self.fail_binding = False

    def call(self, method, path, payload=None, envelope=True):
        if method == "GET":
            return self.entries if path == "/certificate" else dict(self.resource)
        self.writes.append((method, path, payload))
        if path.startswith("/resources/"):
            if self.fail_binding:
                raise sync.SyncError("resource busy")
            self.resource.update(payload)
            return {"status": "accept"}
        if method == "POST":
            self.entries = [{"seq": 7, "comment": payload["comment"]}]
        else:
            self.entries[0]["comment"] = payload["comment"]
        return {"seq": 7}


class SyncTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        cls.cert, cls.key = issue()

    def material(self, cert=None, key=None, chain=b""):
        cert_bytes, key_bytes = pem(cert or self.cert, key or self.key)
        with tempfile.TemporaryDirectory() as directory:
            Path(directory, "tls.crt").write_bytes(cert_bytes + chain)
            Path(directory, "tls.key").write_bytes(key_bytes)
            return sync.load_material(directory)

    def test_validation_normalizes_key_to_pkcs8(self):
        material = self.material()
        self.assertTrue(material[1].startswith("-----BEGIN PRIVATE KEY-----"))
        self.assertEqual(len(material[2]), 64)

    def test_mismatched_key_rejected(self):
        _, key = issue()
        with self.assertRaisesRegex(sync.SyncError, "do not match"):
            self.material(key=key)

    def test_expired_near_expiry_and_future_rejected(self):
        for days, starts in [(-1, -90), (0.5, -1), (90, 1)]:
            with self.subTest(days=days, starts=starts):
                cert, key = issue(days, starts)
                with self.assertRaises(sync.SyncError):
                    self.material(cert, key)

    def test_chain_signature_checked_and_full_chain_affects_digest(self):
        issuer = issue()
        cert, key = issue(issuer=issuer)
        leaf_only = self.material(cert, key)
        chain = issuer[0].public_bytes(serialization.Encoding.PEM)
        with_chain = self.material(cert, key, chain)
        self.assertNotEqual(leaf_only[2], with_chain[2])
        # Same issuer subject but a different signing key must fail verification.
        wrong_issuer = issue()[0].public_bytes(serialization.Encoding.PEM)
        with self.assertRaises(sync.SyncError):
            self.material(cert, key, wrong_issuer)

    def test_projected_secret_generation_is_read_consistently(self):
        cert_bytes, key_bytes = pem(self.cert, self.key)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            generation = root / "generation"
            generation.mkdir()
            (generation / "tls.crt").write_bytes(cert_bytes)
            (generation / "tls.key").write_bytes(key_bytes)
            (root / "..data").symlink_to(generation)
            # Direct file paths would fail; the app must resolve ..data once.
            (root / "tls.key").write_bytes(b"invalid")
            self.assertEqual(sync.load_material(root)[2], self.material()[2])

    def test_create_noop_and_renewal_keep_same_id(self):
        api = FakeAPI()
        material = self.material()
        self.assertEqual(sync.reconcile(api, "prod", material), (7, "create"))
        self.assertEqual(sync.reconcile(api, "prod", material), (7, "unchanged"))
        cert, key = issue()
        self.assertEqual(sync.reconcile(api, "prod", self.material(cert, key)), (7, "update"))
        self.assertEqual([w[:2] for w in api.writes], [("POST", "/certificate"), ("PATCH", "/certificate/7")])

    def test_dry_run_does_not_write(self):
        api = FakeAPI()
        sync.reconcile(api, "prod", self.material(), ["resource"], dry_run=True)
        self.assertEqual(api.writes, [])

    def test_duplicate_names_fail_without_writes(self):
        api = FakeAPI()
        api.entries = [{"seq": i, "comment": "cert-sync:prod:sha256:any"} for i in [1, 2]]
        with self.assertRaisesRegex(sync.SyncError, "Multiple"):
            sync.reconcile(api, "prod", self.material())
        self.assertEqual(api.writes, [])

    def test_partial_failure_recovers_binding_without_second_upload(self):
        api = FakeAPI()
        api.fail_binding = True
        material = self.material()
        with self.assertRaises(sync.SyncError):
            sync.reconcile(api, "prod", material, ["resource"])
        api.fail_binding = False
        sync.reconcile(api, "prod", material, ["resource"])
        sync.reconcile(api, "prod", material, ["resource"])
        self.assertEqual(sum(path == "/certificate" for _, path, _ in api.writes), 1)
        self.assertEqual(api.resource["certificate"], 7)

    def test_other_sync_name_is_not_overwritten(self):
        api = FakeAPI()
        api.entries = [{"seq": 9, "comment": "cert-sync:other:sha256:any"}]
        sync.reconcile(api, "prod", self.material())
        self.assertEqual(api.writes[0][0], "POST")

    def test_api_rejects_http_200_with_failed_status(self):
        api = sync.API("secret-token")
        for response in [{"status": 1, "message": "SECRET"}, {"status": "0"}, {}, {"status": 0}]:
            with patch.object(api, "http", return_value=(response, {})):
                with self.assertRaises(sync.SyncError) as caught:
                    api.call("POST", "/certificate", {"private_key": "SECRET"})
                self.assertNotIn("SECRET", str(caught.exception))

    def test_http_error_body_and_token_not_exposed(self):
        failure = HTTPError("https://api.selectel.ru", 400, "SECRET", {}, io.BytesIO(b"PRIVATE KEY"))
        with patch("sync.request.OpenerDirector.open", side_effect=failure):
            with self.assertRaisesRegex(sync.SyncError, "HTTP 400") as caught:
                sync.API.http("POST", sync.CDN_URL, {"private_key": "PRIVATE KEY"}, "TOKEN")
        self.assertNotIn("PRIVATE KEY", str(caught.exception))
        self.assertNotIn("TOKEN", str(caught.exception))

    def test_auth_project_scope_and_token_header(self):
        env = {"SELECTEL_USERNAME": "user", "SELECTEL_PASSWORD": "password",
               "SELECTEL_ACCOUNT_ID": "123", "SELECTEL_PROJECT_NAME": "cdn"}
        with patch.object(sync.API, "http", return_value=({}, {"X-Subject-Token": "token"})) as http:
            api = sync.API.authenticate(env)
        self.assertEqual(api.token, "token")
        payload = http.call_args.args[2]
        self.assertEqual(payload["auth"]["scope"]["project"]["name"], "cdn")
        self.assertEqual(payload["auth"]["identity"]["methods"], ["password"])

    def test_invalid_local_material_prevents_authentication(self):
        with patch.dict("os.environ", {"SYNC_NAME": "prod", "TLS_DIRECTORY": "/nonexistent"}, clear=True):
            with patch.object(sync.API, "authenticate") as authenticate:
                with contextlib.redirect_stderr(io.StringIO()):
                    with patch("sys.argv", ["sync.py"]):
                        self.assertEqual(sync.main(), 1)
                authenticate.assert_not_called()

    def test_application_credentials_from_existing_cloud_config(self):
        with tempfile.TemporaryDirectory() as directory:
            path = Path(directory, "cloud.conf")
            path.write_text("[Global]\napplication-credential-id = credential-id\n"
                            "application-credential-secret = credential-secret\n")
            with patch.object(sync.API, "http", return_value=({}, {"X-Subject-Token": "token"})) as http:
                api = sync.API.authenticate({"SELECTEL_CLOUD_CONFIG_FILE": str(path)})
        self.assertEqual(api.token, "token")
        payload = http.call_args.args[2]
        self.assertNotIn("scope", payload["auth"])
        self.assertEqual(payload["auth"]["identity"]["methods"], ["application_credential"])
        self.assertEqual(payload["auth"]["identity"]["application_credential"]["id"], "credential-id")

    def test_unbound_resource_without_certificate_field_can_be_bound(self):
        api = FakeAPI()
        del api.resource["certificate"]
        sync.reconcile(api, "prod", self.material(), ["resource"])
        self.assertEqual(api.resource["certificate"], 7)

    def test_domain_gap_prevents_certificate_upload_and_binding(self):
        api = FakeAPI()
        api.resource["names"].append("uncovered.example.com")
        with self.assertRaisesRegex(sync.SyncError, "does not cover"):
            sync.reconcile(api, "prod", self.material(), ["resource"])
        self.assertEqual(api.writes, [])

    def test_wildcard_matches_one_label_only(self):
        for name, expected in [("cdn.example.com", True), ("example.com", False),
                               ("apk.s3.example.com", False), ("cdn.example.com.attacker.test", False),
                               ("CDN.EXAMPLE.COM.", True)]:
            with self.subTest(name=name):
                self.assertEqual(sync.covers_hostname("*.example.com", name), expected)

    def secret(self, name="cdn-tls", annotations=None):
        cert, key = pem(self.cert, self.key)
        return {"metadata": {"name": name, "annotations": annotations or {
            sync.ANNOTATION_PREFIX + "enabled": "true"}}, "type": "kubernetes.io/tls",
            "data": {"tls.crt": base64.b64encode(cert).decode(),
                     "tls.key": base64.b64encode(key).decode()}}

    def test_discovery_ignores_unannotated_metadata_and_follows_pagination(self):
        secret = self.secret()
        kube = sync.Kubernetes.__new__(sync.Kubernetes)
        kube.get = Mock(side_effect=[
            {"kind": "PartialObjectMetadataList", "metadata": {"continue": "next/page"},
             "items": [{"metadata": {"name": "ignored"}}]},
            {"kind": "PartialObjectMetadataList", "metadata": {},
             "items": [{"metadata": secret["metadata"]}]}, secret])
        self.assertEqual(list(kube.selected_secrets(["network"])), [("network", secret)])
        self.assertEqual(kube.get.call_count, 3)
        self.assertIn("continue=next%2Fpage", kube.get.call_args_list[1].args[0])
        self.assertTrue(kube.get.call_args_list[0].kwargs["metadata_only"])

    def test_discovery_rechecks_disabled_or_deleting_secrets(self):
        for changed in [self.secret(annotations={sync.ANNOTATION_PREFIX + "enabled": "false"}), self.secret()]:
            if changed["metadata"]["annotations"][sync.ANNOTATION_PREFIX + "enabled"] == "true":
                changed["metadata"]["deletionTimestamp"] = "2026-10-26T00:00:00Z"
            kube = sync.Kubernetes.__new__(sync.Kubernetes)
            kube.get = Mock(side_effect=[{"kind": "PartialObjectMetadataList", "items": [
                {"metadata": self.secret()["metadata"]}]}, changed])
            self.assertEqual(list(kube.selected_secrets(["network"])), [])

    def test_no_secrets_selected_does_not_authenticate(self):
        kube = Mock()
        kube.selected_secrets.return_value = []
        with patch.object(sync.API, "authenticate") as auth:
            self.assertEqual(sync.run_discovery(kube, {"CLUSTER_NAME": "example-cluster", "SYNC_NAMESPACES": "network"}), 0)
        auth.assert_not_called()

    def test_multiple_sources_share_one_authentication_and_dry_run(self):
        kube = Mock()
        kube.selected_secrets.return_value = [("network", self.secret()), ("network", self.secret("another"))]
        with patch.object(sync.API, "authenticate", return_value=FakeAPI()) as auth:
            with patch.object(sync, "reconcile_with_retry") as reconcile:
                result = sync.run_discovery(kube, {"CLUSTER_NAME": "example-cluster", "SYNC_NAMESPACES": "network"}, True)
        self.assertEqual(result, 0)
        auth.assert_called_once()
        self.assertEqual(reconcile.call_count, 2)
        self.assertTrue(all(c.args[-1] for c in reconcile.call_args_list))
        self.assertNotEqual(reconcile.call_args_list[0].args[1], reconcile.call_args_list[1].args[1])

    def test_bad_source_does_not_block_other_sources_but_job_fails(self):
        bad = self.secret("broken")
        bad["data"]["tls.key"] = "invalid-base64!"
        kube = Mock()
        kube.selected_secrets.return_value = [("network", bad), ("network", self.secret())]
        with patch.object(sync.API, "authenticate", return_value=FakeAPI()):
            with patch.object(sync, "reconcile_with_retry") as reconcile:
                with self.assertLogs(sync.LOG, level="ERROR"):
                    result = sync.run_discovery(kube, {"CLUSTER_NAME": "example-cluster", "SYNC_NAMESPACES": "network"})
        self.assertEqual(result, 1)
        reconcile.assert_called_once()

    def test_conflicting_annotations_prevent_all_remote_changes(self):
        for field, value in [("name", "same-name"), ("resource-ids", "00000000-0000-4000-8000-000000000001")]:
            with self.subTest(field=field):
                annotations = {sync.ANNOTATION_PREFIX + "enabled": "true", sync.ANNOTATION_PREFIX + field: value}
                kube = Mock()
                kube.selected_secrets.return_value = [("network", self.secret(annotations=annotations)),
                                                      ("network", self.secret("other", annotations))]
                with patch.object(sync.API, "authenticate") as auth:
                    with self.assertRaisesRegex(sync.SyncError, "Multiple"):
                        sync.run_discovery(kube, {"CLUSTER_NAME": "example-cluster", "SYNC_NAMESPACES": "network"})
                auth.assert_not_called()

    def test_discovery_requires_namespace_allowlist(self):
        with self.assertRaisesRegex(sync.SyncError, "SYNC_NAMESPACES"):
            sync.run_discovery(Mock(), {"CLUSTER_NAME": "example-cluster"})


if __name__ == "__main__":
    unittest.main()
