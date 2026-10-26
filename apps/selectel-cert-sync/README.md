# Selectel certificate sync

Upload and rotate cert-manager TLS certificates in Selectel Certificate Manager, CDN certificates, or both. The app manages certificate storage and versions only. Resource assignment belongs to the consuming CDN resource, load balancer, or service configuration.

Certificate Manager is the default destination. Explicitly select `cdn` when a CDN copy is needed. Both stores share one project-scoped IAM token per run.

## Annotation-based selection

One CronJob runs in `kube-system`, mounts the existing Selectel `cloud-config` Secret, and discovers opted-in TLS Secrets in explicitly allowed namespaces. Put annotations in the source Certificate's `spec.secretTemplate.annotations` so cert-manager maintains them on its generated Secret:

```yaml
spec:
  secretTemplate:
    annotations:
      selectel-cert-sync.glants.xyz/enabled: "true"
      selectel-cert-sync.glants.xyz/name: production-example-certificate
      selectel-cert-sync.glants.xyz/stores: "cdn,certificate-manager"
```

Keep the source Certificate's other spec fields. For Gateway-generated certificates, an independently managed metadata-only Secret manifest can add the annotations without owning its TLS data.

| Annotation suffix (prefix `selectel-cert-sync.glants.xyz/`) | Meaning |
| --- | --- |
| `enabled` | Only the exact string `"true"` opts a TLS Secret in |
| `name` | Optional stable ownership name, 1–80 letters/digits/`._-`; defaults to a hash of cluster/namespace/Secret name |
| `stores` | `certificate-manager` (default), `cdn`, or `cdn,certificate-manager` |

Disable or remove `enabled` to stop syncing. Remote certificates and existing consumer assignments are retained. Keep sync names and cluster identity stable, and use one writer for each name.

## Rotation and ownership

cert-manager handles issuance and renewal. Every five minutes a new Job validates the source certificate/key, authenticates once, and reconciles the selected stores. Discovery reads both fields from one Kubernetes object; mounted mode resolves one projected-volume generation. Private keys stay in memory.

- **CDN:** creates a managed certificate initially and PATCHes the same certificate ID on renewal. Its comment is `cert-sync:<name>:sha256:<normalized-full-chain-hash>`. Matching comments skip uploads. Do not edit managed comments or change these certificates outside the app: comments are its reconciliation record.
- **Certificate Manager:** finds the exact managed name `cert-sync:<name>` and compares its stored public chain with the source. Renewal creates a new version under the same UUID with `POST /cert/{id}`. The app never downloads private keys or changes consumer associations.

The stores have separate records and IDs. Existing manual certificates are not adopted or deleted. Duplicate managed names fail for investigation. Retries reread remote state; one failed destination does not prevent attempts for the other. The Job exits nonzero if any destination fails. The app retries three times and the Job retries twice. Avoid overlapping manual Jobs; CronJob concurrency policy covers scheduled Jobs only.

Validation checks certificate/key matching, current validity with at least 24 hours remaining, and adjacent chain signatures. Keys are normalized to unencrypted PKCS#8 in memory. Public trust-store validation is not performed. Logs include IDs, hashes, expiry and outcomes, excluding private keys, passwords, tokens and API response bodies.

## Resource assignment

Assign an uploaded certificate to its consumer once using Selectel, Terraform, or the consumer's configuration. Renewals preserve its managed ID. This app never reads or writes CDN resources, load-balancer listeners, or Certificate Manager consumer associations.

Version `0.3.0` removes automatic resource attachment. Remove old resource-ID annotations and environment settings when upgrading. Existing bindings remain in place, and certificate rotation continues under the same IDs.

Successful upload means the API accepted the certificate. Verify public TLS separately to confirm propagation to consumers and CDN edges.

## Kubernetes deployment

Use `deploy/discovery-rbac.yaml` and `deploy/discovery-cronjob.yaml`. Publish the image and pin its digest first. Configuration:

| Environment variable | Purpose |
| --- | --- |
| `CLUSTER_NAME` | Required stable cluster identity for discovery |
| `SYNC_NAMESPACES` | Required comma-separated namespace allowlist, e.g. `network` |
| `SELECTEL_CLOUD_CONFIG_FILE` | Mounted cloud.conf path, e.g. `/selectel/cloud.conf` |

The account file uses `[Global] application-credential-id` and `application-credential-secret`. It must permit management of the selected certificate stores. The app needs outbound DNS/HTTPS to `cloud.api.selcloud.ru` and `api.selectel.ru`.

Discovery requests metadata-only lists and fetches full data only for opted-in Secrets. Kubernetes RBAC cannot restrict lists by annotation: the supplied Role grants `get` and `list` for all Secrets in `network`. There is no ClusterRole or Kubernetes write permission. Add a namespace-scoped Role/RoleBinding for each additional allowed namespace.

```sh
kubectl --context YOUR_CONTEXT apply -f deploy/discovery-rbac.yaml
kubectl --context YOUR_CONTEXT apply -f deploy/discovery-cronjob.yaml
kubectl --context YOUR_CONTEXT -n kube-system create job \
  --from=cronjob/selectel-cert-sync selectel-cert-sync-preview
kubectl --context YOUR_CONTEXT -n kube-system logs -f job/selectel-cert-sync-preview
```

The discovery example starts with `--discover --dry-run`. Review the planned uploads, then remove `--dry-run` to enable them.

## Single-mounted-Secret mode

`deploy/cronjob.yaml` mounts a TLS Secret in the Job's namespace and needs no Kubernetes API token. Configure:

| Environment variable | Purpose |
| --- | --- |
| `SYNC_NAME` | Required stable ownership name |
| `SYNC_STORES` | `certificate-manager` (default), `cdn`, or both, comma-separated |
| `TLS_DIRECTORY` | Contains `tls.crt` and `tls.key`; defaults to `/tls` |
| `SELECTEL_CLOUD_CONFIG_FILE` | Optional mounted account file; takes precedence over password authentication |
| `SELECTEL_USERNAME` | Service-user name for password authentication |
| `SELECTEL_PASSWORD` | Service-user password |
| `SELECTEL_ACCOUNT_ID` | Account number |
| `SELECTEL_PROJECT_NAME` | Project name, not UUID |

The password example references `selectel-cert-sync-credentials` keys `username`, `password`, `account-id`, and `project-name`. Create this Secret through your existing secret-management workflow. `deploy/certificate.example.yaml` demonstrates cert-manager renewal and private-key rotation.

`python sync.py --dry-run` validates material and reads remote state without uploading. Add `--discover` for annotation-based discovery.

## Build and tests

The repository workflows discover `docker-bake.hcl`, declaring version `0.3.0` and amd64/arm64 builds. Images run as UID/GID 65532. Dependencies are hash-pinned; regenerate `requirements.txt` from `requirements.in` when changing them.

From the repository root: `mise run local-build selectel-cert-sync`. Alternatively, from this app directory:

```sh
docker buildx bake --load
TEST_IMAGE=selectel-cert-sync:0.3.0 go test -count=1 -v ./...
```

Builds run the Python tests; the Go container test repeats them in the finished image and checks the entrypoint. Unit tests generate ephemeral certificates and use fake APIs without accessing your cluster or account.

## Monitoring and recovery

Alert on failed Jobs, no successful scheduled run for 30 minutes (including never-succeeded CronJobs), cert-manager renewal failures, and source/public expiry under 14 days. The app warns below 14 days; a log warning alone is not an alert. Monitor store delivery and public TLS separately from issuance.

Fix credentials/connectivity or issuance and rerun; reconciliation rereads remote state. A create timeout may leave a record whose response was lost. Inspect Selectel before manual retries if listing is delayed. Resolve duplicate managed names before resuming.

For a bad renewal, suspend syncing and restore known-good material or select a known-good certificate through the consumer's configuration. The app does not archive private keys, delete remote certificates or roll back automatically. Keep rollback material in your existing encrypted backup workflow.

## References

- [Selectel CDN certificate API](https://docs.selectel.ru/en/api/cdn/)
- [Selectel Certificate Manager API](https://docs.selectel.ru/en/api/certificates-manager/)
- [Selectel IAM authentication](https://docs.selectel.ru/en/api/authorization/)
- [cert-manager Certificate lifecycle](https://cert-manager.io/docs/usage/certificate/)
- [Kubernetes metadata-only reads](https://kubernetes.io/docs/reference/using-api/api-concepts/#metadata-only-fetches)
