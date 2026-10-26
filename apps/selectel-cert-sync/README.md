# cert-manager → Selectel certificate stores

A Python sync job that uploads cert-manager TLS Secrets to **Selectel CDN certificates**, **Selectel Certificate Manager**, or both, and updates them after renewal. Both destinations share one project-scoped IAM token per run. The default destination is Certificate Manager; CDN syncing requires an explicit `cdn` destination.

## Choosing destinations

Set the `selectel-cert-sync.glants.xyz/stores` annotation on each opted-in TLS Secret (normally through `Certificate.spec.secretTemplate.annotations`):

```yaml
spec:
  secretTemplate:
    annotations:
      selectel-cert-sync.glants.xyz/enabled: "true"
      selectel-cert-sync.glants.xyz/name: production-example-certificate
      selectel-cert-sync.glants.xyz/stores: "cdn,certificate-manager"
      selectel-cert-sync.glants.xyz/resource-ids: "<cdn-resource-uuid>"
```

| `stores` value | Destination |
| --- | --- |
| Omitted or `certificate-manager` | Certificate Manager only (default); omit CDN `resource-ids` |
| `cdn` | CDN certificate store only |
| `cdn,certificate-manager` | Both stores; resource IDs apply only to CDN |

Certificate Manager uses `https://cloud.api.selcloud.ru/certificate-manager/v1`. Managed records have the stable name `cert-sync:<sync-name>`. The app finds that exact name, compares the normalized stored **public certificate chain** against the source, and creates a new version on renewal using `POST /cert/{id}`. The UUID is preserved; the app does not rename records or change consumer associations. It never downloads private keys. A failed destination is retried independently, and does not prevent attempts to sync to the other destination. The Job exits nonzero if any destination fails.

Certificate Manager and CDN use separate records and separate IDs. Uploading to Certificate Manager does not bind cloud load balancers, and does not update the CDN store unless `cdn` is also selected. Removing a destination annotation stops future updates to that store and retains the remote certificate. Existing manual records are not adopted; duplicate managed names cause a failure before writing that record.

When upgrading from `0.1.0`, explicitly add `stores: "cdn"` or `stores: "cdn,certificate-manager"` to existing CDN sources before upgrading. A source that still has CDN resource IDs but omits the CDN destination is rejected rather than silently changing its targets.

For single-mounted-Secret mode, set `SYNC_STORES=certificate-manager` or `SYNC_STORES=cdn,certificate-manager`. Its default is `certificate-manager`. Supplying CDN resource IDs without selecting the CDN store is rejected.

One CronJob can run in `kube-system`, mount the existing Selectel `cloud-config` Secret, and sync annotation-selected TLS Secrets from explicitly allowed namespaces. Authentication is shared across all sources in a run. A simpler single-mounted-Secret mode is also available. No custom resources, database, or persistent volume is needed.

## Annotation-based selection

Put annotations in the source Certificate's `spec.secretTemplate.annotations` so cert-manager maintains them on its generated TLS Secret. Top-level annotations on the Certificate alone do not select the Secret.

```yaml
apiVersion: cert-manager.io/v1
kind: Certificate
metadata:
  name: example-com-wildcard-tls
  namespace: network
spec:
  # Keep the existing issuer, DNS names, renewal and private-key settings.
  secretName: example-com-wildcard-tls
  secretTemplate:
    annotations:
      selectel-cert-sync.glants.xyz/enabled: "true"
      selectel-cert-sync.glants.xyz/name: production-example-cdn
      selectel-cert-sync.glants.xyz/stores: "cdn"
      selectel-cert-sync.glants.xyz/resource-ids: "00000000-0000-4000-8000-000000000001,00000000-0000-4000-8000-000000000002,00000000-0000-4000-8000-000000000003"
```

This is an annotation fragment for the existing Certificate, not a complete replacement manifest: preserve its other spec fields. Replace the three placeholder resource UUIDs with your own Selectel CDN resource IDs.

| Annotation suffix (prefix `selectel-cert-sync.glants.xyz/`) | Meaning |
| --- | --- |
| `enabled` | Only the exact string `"true"` opts a Secret in |
| `name` | Optional stable remote ownership name; defaults to a hash of cluster/namespace/Secret name |
| `stores` | `certificate-manager` (default), `cdn`, or both, comma-separated |
| `resource-ids` | Optional comma-separated CDN resource UUIDs; omit for upload/update only |

Disable or remove `enabled` to stop future synchronization. The remote certificate and bindings are retained. A duplicate sync name or two selected Secrets targeting the same CDN resource fails the run before remote writes. Each source's certificate must cover **all custom domains** on every selected resource. A wildcard covers one DNS label: `*.example.com` covers `cdn.example.com`, but not `example.com` or `apk.s3.example.com`. Only the resource's exact provider `cdn_domain` is excluded from this check.

Use `deploy/discovery-rbac.yaml` and `deploy/discovery-cronjob.yaml` for this mode. The CronJob:

- Runs in `kube-system` and mounts `cloud-config/cloud.conf` read-only, with no credential copy or per-certificate auth settings.
- Uses `SELECTEL_CLOUD_CONFIG_FILE=/selectel/cloud.conf` to authenticate with its `[Global] application-credential-id` and `application-credential-secret`. It obtains a fresh project-scoped token once per run.
- Uses `CLUSTER_NAME=example-cluster` for stable default remote names and `SYNC_NAMESPACES=network` as the explicit namespace allowlist. Keep the cluster name stable after first upload.
- Lists Secret **metadata only**, follows pagination, and fetches full certificate/key data only for opted-in Secrets. It rechecks opt-in after fetching the Secret.
- Processes each source independently; a material/upload failure for one source does not prevent attempts for other sources, but the overall Job exits nonzero.

Annotation discovery requires Kubernetes `list` and `get` permission on Secrets in each allowed namespace. RBAC cannot restrict a namespace-wide list by annotation; the supplied Role grants read access to all Secrets in `network`. Annotations are selection controls, not an RBAC boundary. There is no ClusterRole and no Kubernetes write permission. Add a namespace-scoped Role/RoleBinding for each additional allowed namespace. The account Secret is mounted by Kubernetes; the app does not list `kube-system` Secrets.

After releasing the image through this repository and setting its digest in the discovery CronJob:

```sh
kubectl --context YOUR_CONTEXT apply -f deploy/discovery-rbac.yaml
kubectl --context YOUR_CONTEXT apply -f deploy/discovery-cronjob.yaml
kubectl --context YOUR_CONTEXT -n kube-system create job \
  --from=cronjob/selectel-cert-sync selectel-cert-sync-preview
kubectl --context YOUR_CONTEXT -n kube-system logs -f job/selectel-cert-sync-preview
```

The supplied discovery CronJob starts with `--discover --dry-run`. Review the actions for annotated sources, then remove `--dry-run` to enable writes. Run only one CronJob for any set of sync names, and avoid overlapping manual Jobs. Secrets are scanned every five minutes; annotations select sources without turning the app into a continuously watching controller.

Existing manually uploaded Selectel certificates are retained. The first sync creates a managed certificate and can switch the configured resource bindings to it; later renewals update that managed ID. It does not adopt an arbitrary old certificate ID. A dry run reports the planned creation without changing existing bindings.

The remaining deployment instructions describe the **single-mounted-Secret mode**. Use one deployment mode for each source.

## Rotation

1. cert-manager issues and renews the source certificate. The example requests 90 days and renews with 33% of the actual certificate lifetime remaining, approximately 30 days before expiry for a 90-day certificate. `rotationPolicy: Always` rotates the private key with each issuance.
2. Every five minutes a new Job reads the latest Secret. Discovery mode fetches certificate and key in one Kubernetes object read. Mounted mode resolves the projected volume generation once so the certificate and key come from the same Secret version.
3. The first run creates a CDN certificate. Later runs PATCH the same certificate ID when the normalized full-chain SHA-256 changes. Certificate and key are sent together. Unchanged certificates produce no upload.
4. Optionally, `CDN_RESOURCE_IDS` binds that ID to existing CDN resources. Only their top-level `certificate` field is patched. If a binding fails, the next reconciliation retries it without uploading the certificate again.
5. API failures exit nonzero. The app retries reconciliation three times with delays; Kubernetes retries failed Jobs twice. Subsequent scheduled runs also retry. `concurrencyPolicy: Forbid` prevents overlapping scheduled Jobs.

Renewal delivery normally takes up to five minutes plus scheduling and Selectel propagation time. A successful Job means the API accepted the operation; it does **not** prove that every CDN edge is serving the new certificate. Validate propagation with external TLS monitoring before relying on unattended rotation.

## Container build integration

This app is built and released by the repository's shared workflows. `docker-bake.hcl` declares version `0.2.0`, image targets, and `linux/amd64` / `linux/arm64` platforms. Builds run the Python tests, and `container_test.go` repeats them inside the finished image and checks the entrypoint. The image runs as UID/GID 65532. Dependencies are pinned with hashes in `requirements.txt`; regenerate from `requirements.in` when updating them.

From the repository root, use `mise run local-build selectel-cert-sync`. Released images use `ghcr.io/mglants/selectel-cert-sync`; this change does not publish one.

## Build and deploy

Requirements: cert-manager already installed, an existing issuer, a Selectel service user permitted to manage the selected certificate stores in the target project, and an existing CDN resource if automatic binding is wanted. The certificate SANs must cover every custom CDN domain. Prefer a DNS-01 issuer because the public domain points at CDN, which can prevent HTTP-01 validation from reaching Kubernetes.

```sh
cd apps/selectel-cert-sync
docker buildx bake --load
TEST_IMAGE=selectel-cert-sync:0.2.0 go test -count=1 -v ./...
```

Edit `deploy/certificate.example.yaml` with your DNS names and issuer, or use an existing Certificate. Edit `deploy/cronjob.yaml` with your published image (prefer an immutable digest), TLS Secret name, and a stable `SYNC_NAME`. Set `CDN_RESOURCE_IDS` to comma-separated resource UUIDs to enable automatic binding, or leave it empty and select the uploaded certificate in Selectel once.

Create a Secret named `selectel-cert-sync-credentials` using your existing secret-management workflow, with these keys:

| Secret key | Value |
| --- | --- |
| `username` | Selectel service-user name |
| `password` | Service-user password |
| `account-id` | Selectel account number |
| `project-name` | Target project name, not its UUID |

The source Certificate, TLS Secret, credentials Secret, and CronJob must be in the **same namespace**. The pod runs as a non-root user, mounts the TLS Secret read-only, and has no service-account token. It needs outbound DNS and HTTPS to `cloud.api.selcloud.ru` and `api.selectel.ru`.

```sh
# Only if you need to create the source Certificate:
kubectl -n YOUR_NAMESPACE apply -f deploy/certificate.example.yaml
kubectl -n YOUR_NAMESPACE wait --for=condition=Ready certificate/cdn-example-com --timeout=300s

# After configuring the image, Secret name, sync name, and credentials:
kubectl -n YOUR_NAMESPACE apply -f deploy/cronjob.yaml
kubectl -n YOUR_NAMESPACE create job --from=cronjob/selectel-cert-sync selectel-cert-sync-initial
kubectl -n YOUR_NAMESPACE logs -f job/selectel-cert-sync-initial
```

Ensure HTTPS delivery through CNAME is enabled on the CDN resource. This app manages the uploaded certificate and optional certificate binding; it does not manage resource domains, origins, or redirect settings.

For multiple certificates, use one CronJob per source Secret with a distinct CronJob name and sync name. Use only one writer for each sync name. Do not run manual Jobs while a scheduled Job is active; CronJob concurrency policy does not cover manually created Jobs.

## Configuration and dry run

| Environment variable | Purpose |
| --- | --- |
| `SYNC_NAME` | Required stable identifier, 1–80 letters/digits/`._-` |
| `SYNC_STORES` | Mounted mode destinations: `certificate-manager` (default), `cdn`, or both |
| `TLS_DIRECTORY` | Directory containing `tls.crt` and `tls.key`; default `/tls` |
| `SELECTEL_USERNAME` | Required service-user name |
| `SELECTEL_PASSWORD` | Required service-user password |
| `SELECTEL_ACCOUNT_ID` | Required account number |
| `SELECTEL_PROJECT_NAME` | Required project name |
| `SELECTEL_CLOUD_CONFIG_FILE` | Optional mounted cloud.conf path; application credentials take precedence over password authentication |
| `CDN_RESOURCE_IDS` | Optional comma-separated existing CDN resource UUIDs |

`python sync.py --dry-run` validates local material, authenticates, and reads Selectel state, then logs planned actions without writing. To test in Kubernetes, create a suspended copy of the CronJob and add `args: ["--dry-run"]` before creating a manual Job from it. Keep its real namespace, volume mounts, and credentials configuration.

Remote ownership and change detection use a comment of the form `cert-sync:<SYNC_NAME>:sha256:<full-chain-hash>`. Do not edit this comment or modify this certificate outside the app: metadata is the reconciliation record, not an independent cryptographic readback of Selectel's stored key. Changing `SYNC_NAME` creates a new certificate. Pre-existing manually uploaded certificates are not adopted or deleted. Duplicate matching comments cause a failure for manual investigation. No remote certificates are automatically deleted.

The client rejects expired/not-yet-valid certificates, chains with less than 24 hours remaining, mismatched keys, invalid adjacent chain signatures, and certificates missing a custom domain on a selected CDN resource. It normalizes private keys to unencrypted PKCS#8 in memory. It does not perform public trust-store validation; cert-manager must supply a trusted chain. Logs contain source names, IDs, fingerprints, expiry dates and outcomes; private keys, tokens, passwords and API response bodies are excluded.

## Monitoring and recovery

Configure these alerts in your existing monitoring system before production use:

- No successful run for 30 minutes (`kube_cronjob_status_last_successful_time` from kube-state-metrics), including an alert for a CronJob that has never succeeded.
- Failed Jobs or pods unable to start, including missing Secrets and image-pull failures.
- cert-manager source expiry under 14 days and Certificate readiness/renewal failures.
- External HTTPS certificate expiry under 14 days and a leaf fingerprint mismatch persisting beyond your measured CDN propagation interval.

The app also logs a warning below 14 days remaining. A log warning alone is not an alert. A valid source certificate does not prove successful CDN delivery, so monitor source and public endpoint separately.

On failure, existing Selectel certificates remain in place unless Selectel already accepted the update. Fix credentials/connectivity or cert-manager issuance and rerun the Job; it rereads remote state. A create timeout can leave an uploaded certificate whose response was lost; inspect Selectel before manually retrying, especially if its list endpoint is delayed. If duplicate records exist, identify the bound ID and resolve duplicates before resuming.

For a bad renewal, suspend the CronJob first and restore a known-good certificate/key in Selectel or bind a known-good retained certificate, then verify the public TLS endpoint. This app does not archive private keys or automatically roll back. Use your existing encrypted backup workflow if rollback material is required.

## Tests

```sh
python3 -m venv .venv
.venv/bin/pip install --require-hashes -r requirements.txt
.venv/bin/python -m unittest discover -v
```

Tests generate ephemeral certificates and use fake/mock API responses; they do not access your cluster or Selectel account. Before production, run a staging dry run, initial upload/binding, and a forced cert-manager renewal (`cmctl renew CERTIFICATE_NAME -n NAMESPACE`). Confirm the Selectel certificate ID remains unchanged, the next Job updates it, the CDN serves the new leaf certificate, and a subsequent Job reports `unchanged`.

## API references

- [Selectel CDN v3 API: certificate create/list/update and partial resource updates](https://docs.selectel.ru/en/api/cdn/)
- [Selectel Certificate Manager API: upload and version updates](https://docs.selectel.ru/en/api/certificates-manager/)
- [Selectel API URLs](https://docs.selectel.ru/en/api/urls/)
- [Selectel project-scoped IAM authentication](https://docs.selectel.ru/en/api/authorization/)
- [Selectel CDN certificate configuration](https://docs.selectel.ru/en/cdn/resources/tls-ssl-certificates/)
- [cert-manager Certificate lifecycle and renewal](https://cert-manager.io/docs/usage/certificate/)
- [cert-manager-sync annotation selection pattern](https://github.com/robertlestak/cert-manager-sync)
- [Kubernetes metadata-only API reads](https://kubernetes.io/docs/reference/using-api/api-concepts/#metadata-only-fetches)
