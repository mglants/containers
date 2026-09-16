# Google Workspace → Authentik lifecycle sync

Five-minute, one-way reconciliation of all users in an explicitly configured
Google Workspace customer (including secondary domains). No Google writes,
password synchronization, group synchronization, or database access.

## Behavior

- First adoption requires a unique case-insensitive primary-email match. Later
  runs use the immutable Google user ID stored in `attributes.google_workspace_sync`.
- Existing usernames, groups, roles, and unrelated attributes are preserved.
  New usernames use `USERNAME_FORMAT=email` (default, full email address) or
  `USERNAME_FORMAT=local_part` (`user@example.com` becomes `user`). Existing
  usernames remain unchanged, including after email renames or format changes.
  `USERNAME_COLLISION_POLICY=error` (default) aborts on username collisions.
  Set it to `email` to use full email addresses for new accounts whose local
  parts collide across eligible new users or with an existing Authentik username.
  All new members of a collision use email, independent of listing order.
  Existing usernames remain unchanged; a collision on the fallback email still
  aborts. Identities are never merged or automatically suffixed.
  New accounts default to internal users with no passwords, groups, or roles.
  `AUTHENTIK_USER_TYPE` selects `internal` or `external` and `AUTHENTIK_USER_PATH`
  chooses the Authentik folder for new accounts. Existing account types and paths
  are preserved; changing these settings is not a migration of earlier imports.
- Suspension, archival, or confirmed deletion disables a managed Authentik user.
  Missing users are individually checked: only HTTP 404 confirms deletion.
- New suspended or archived Google users are skipped by default. Set
  `CREATE_DISABLED_USERS=true` to create them as inactive Authentik users instead.
  Existing matched users are still adopted and disabled regardless of this setting.
  A skipped user is created normally if Google later restores the account.
- All Google and Authentik pages and deletion confirmations are read before any
  writes. Empty Google snapshots, failed reads, and identity conflicts abort the
  run. Never treat a failed request as an empty directory.
- A recycled email cannot take over an account already associated with another
  Google ID. Duplicate identities and username collisions require manual review.
- Only accounts disabled by this job are automatically reactivated. Accounts
  disabled independently before the sync are left disabled.
- Service accounts, superusers, and `EXCLUDED_USERS` (Authentik PKs or usernames)
  are protected. The default explicit exclusion is `akadmin`. Review all
  break-glass accounts before rollout. Excluding a previously managed user
  stops all subsequent lifecycle changes for that account.
- `EXCLUDED_GOOGLE_USERS` skips Google accounts by comma-separated primary emails
  (case-insensitive) or immutable Google user IDs. It skips creation **and all
  updates/offboarding** of existing matched users, without deleting them. Use IDs
  for exclusions that must survive email renames. For already-deleted Google users,
  email exclusions match the last synchronized Authentik email. Aliases, local
  parts, and wildcard patterns are not supported. Skips are logged with reason
  `excluded_google_account` and do not reserve a new local-part username.
- Set `attributes.google_workspace_sync.manual_hold: true` on a managed user
  to keep it disabled, including after Google restores it. If an administrator
  wants to keep an already sync-disabled user inactive, they **must set this
  hold**: setting `is_active=false` again cannot record separate intent.
- Account changes between planning and execution abort that update. Authentik
  does not offer a compare-and-swap transaction here: avoid concurrent bulk user
  edits and never run multiple copies of the sync simultaneously.
- Writes are not automatically retried after uncertain responses. The next run
  re-reads state, making partial failures recoverable without blind POST replay.

Disabling through Authentik's user API triggers its session cleanup. It does
**not** guarantee termination of independently issued OpenUnison sessions,
downstream app cookies, or already-issued access tokens. Validate deactivation
against your deployed Authentik version with a test user before production use.

## Credentials and configuration

1. Create a dedicated Google Cloud service account and enable Admin SDK API in
   its project. Do not reuse the Google login OAuth client.
2. In Google Admin, create a custom administrative role with **Users → Read**
   across the entire customer, and assign it directly to the service account.
   This implementation uses direct role assignment, **not domain-wide delegation
   or administrator impersonation**. Do not grant Google write scopes or Super Admin.
3. Create a service-account JSON key, protecting it as a credential. Obtain the
   explicit customer ID (`C...`), not a domain or the `my_customer` alias.
4. Create a dedicated `google-authentik-sync` service account in Authentik,
   using your infrastructure configuration or the Authentik admin interface,
   with only `authentik_core.view_user`, `authentik_core.add_user`, and
   `authentik_core.change_user`. No superuser or deletion
   permissions. These permissions are user-model-wide: code exclusions are not
   an Authentik authorization boundary. Protect this token accordingly.
5. Generate an API token for that service account and transfer it directly to
   the secret editor; never paste it into a terminal transcript or commit it
   unencrypted. If Terraform manages the token, it also stores it in state.
6. In the Flux deployment repository, edit
   `apps/authentik/authentik/google-authentik-sync/app/base/secret.sops.yaml` using
   `sops`, replacing both encrypted placeholders. The keys are
   `google-service-account.json` (the complete JSON) and `authentik-token`.
   No Vault is required.

Runtime inputs:

| Variable | Meaning |
| --- | --- |
| `GOOGLE_CUSTOMER_ID` | Explicit Workspace customer ID |
| `GOOGLE_APPLICATION_CREDENTIALS` | Mounted service-account JSON filename |
| `AUTHENTIK_URL` | Required Authentik HTTPS origin, e.g. `https://authentik.example.com`; no application default |
| `AUTHENTIK_TOKEN` | Dedicated API token, injected from the Secret via `secretKeyRef` |
| `AUTHENTIK_USER_TYPE` | New-account type: `internal` (default) or `external` |
| `AUTHENTIK_USER_PATH` | Folder for new accounts; default `goauthentik.io/sources/google`, e.g. `employees/google` |
| `EXCLUDED_USERS` | Comma-separated protected usernames or Authentik PKs |
| `EXCLUDED_GOOGLE_USERS` | Comma-separated Google primary emails or immutable IDs; empty by default |
| `CREATE_DISABLED_USERS` | `false` (default) skips new suspended/archived users; `true` creates them inactive |
| `USERNAME_FORMAT` | `email` (default) or `local_part`; only affects newly created usernames |
| `USERNAME_COLLISION_POLICY` | `error` (default) or `email`; email fallback applies to new local-part usernames |

The Google scope is solely
`https://www.googleapis.com/auth/admin.directory.user.readonly`.
All requests verify TLS; neither redirects nor insecure HTTPS are accepted.
Dry-run makes API reads only, though Google authentication obtains a short-lived
access token. Logs contain identity IDs/usernames and actions, never tokens or
HTTP response bodies; treat logs as personnel data.

## Build and test

This container integrates with the repository's standard GitHub Actions pipeline:
`metadata.yaml` declares the stable channel and tested `linux/amd64` platform;
`ci/latest.py` reads the locally maintained `VERSION` file. Bump `VERSION` for
releases. CI supplies OCI version/revision labels and publishes through the
repository owner's GHCR namespace. PR builds do not publish; main-branch builds
publish changed apps. Scheduled builds skip an already-published version unless forced.

From the containers repository root:

```sh
task container:build app=google-authentik-sync
task container:test app=google-authentik-sync
```

On hosts without `/bin/bash` (such as NixOS), the downloaded `dgoss` helper's
shebang prevents the test task from launching it. After building, run the same
checks explicitly through Bash from the repository root:

```sh
GOSS_PATH="$PWD/.goss/goss" \
GOSS_FILES_PATH="$PWD/apps/google-authentik-sync/ci" \
GOSS_FILE=goss.yaml GOSS_FILES_STRATEGY=cp \
bash .goss/dgoss run YOUR_BUILT_IMAGE tail -f /dev/null
```

Unit tests run during the image build and again in `ci/goss.yaml`, alongside
non-root, CLI-help, and missing-configuration checks. No live credentials are
needed. The entrypoint forwards option arguments to the sync command, while
explicit commands such as `tail -f /dev/null` are executed directly for dgoss.
The default remains `--dry-run`; the Kubernetes `args: ["--apply"]` interface
is unchanged. Additional architectures can be enabled after validating them.

From this container directory (`apps/google-authentik-sync` in the containers repository):

```sh
python3 -m unittest discover -s . -v
docker build -t YOUR_REGISTRY/google-authentik-sync:0.1.5 .
```

Deployment manifests remain in the separate Flux repository. From its root,
validate them with `kustomize build apps/authentik/authentik/google-authentik-sync/app/base`.

Publish the image to your registry, then set the container `image` in that repository's
`apps/authentik/authentik/google-authentik-sync/app/base/cronjob.yaml` to its immutable
`repository@sha256:...` reference. Add `imagePullSecrets` to the Pod spec if
required. No registry or publication permission is assumed.
Dependencies are fully pinned with hashes in `requirements.txt`; regenerate
using `pip-compile --generate-hashes --strip-extras requirements.in` in this directory.

For a local read-only preview, install the lock into an isolated Python 3.13
virtual environment, supply the required variables above, and run:

```sh
python sync.py --dry-run
```

`--dry-run` is also the CLI default. `--apply` explicitly enables writes.

## Rollout

The native `cronjob.yaml` resource is managed directly by Flux/Kustomize. Include
the application's `app/base/ks.yaml` in your target cluster's Kustomization.
No Helm resources are used. The CronJob is **suspended** and configured for
**dry-run** by default. Set `KUBE_CONTEXT` to your target kubectl context for the
commands below. The example Authentik hostname is a placeholder, not a live endpoint.
Set the Authentik URL, customer ID, and a digest-pinned container image
before unsuspending or manually starting it. There is no Helm-time validation;
the sync command still rejects an unconfigured customer ID or token at runtime.

1. Publish the image, populate SOPS credentials, configure the Authentik URL,
   customer ID and exclusions, commit the changes, and reconcile Flux on your target cluster.
2. Keep `spec.suspend: true` and container `args: ["--dry-run"]`. Create one preview Job:

   ```sh
   kubectl --context "${KUBE_CONTEXT:?Set your target context}" -n authentik create job google-authentik-sync-preview-1 --from=cronjob/google-authentik-sync
   kubectl --context "${KUBE_CONTEXT:?Set your target context}" -n authentik logs job/google-authentik-sync-preview-1
   ```

3. Review the **entire** adoption/create/exclusion report. Resolve conflicts
   before enabling writes, especially duplicate emails and reused identities.
4. Change container `args` to `["--apply"]`, keeping the CronJob suspended, reconcile, and run one
   explicitly reviewed Job. Never overlap manual Jobs with scheduled runs.
5. With a designated test user, verify login, Google suspension → Authentik
   disablement/session removal, and restoration. Verify manual holds and unchanged
   groups/roles. Creating or suspending the Google test account is an administrator
   action, not something the read-only sync can do.
6. Set `suspend: false` through Git. Successful syncs start every five minutes;
   the four-minute deadline bounds execution. An API outage extends this delay
   and must be handled operationally, not by disabling unconfirmed users.

## Monitoring, rotation, and rollback

Every complete run emits `event=success` with its mode and counts; any conflict
or API failure exits nonzero. Jobs do not retry writes in-process or via Kubernetes;
the next scheduled run reconciles fresh state. Inspect failed Jobs and their logs.

```sh
kubectl --context "${KUBE_CONTEXT:?Set your target context}" -n authentik get cronjob google-authentik-sync -o jsonpath='{.status.lastSuccessfulTime}'
kubectl --context "${KUBE_CONTEXT:?Set your target context}" -n authentik get jobs
```

When wiring your monitoring backend, alert on failed Jobs and, while unsuspended,
no successful run for 15 minutes. With kube-state-metrics, use
`kube_cronjob_status_last_successful_time` for this CronJob, including a missing
timestamp condition after initial scheduling. No monitoring CRD is assumed here.
Check `mode=apply` in logs after rollout; dry-run success is not a lifecycle update.

Rotate the Google key by issuing a replacement, updating SOPS, verifying a run,
and revoking the old key. Rotate the Authentik token through your credential
management process, update SOPS, and verify; token replacement can briefly interrupt sync.
New Jobs receive the updated token environment variable and mounted Google key automatically.

Rollback: set `suspend: true` in Git and reconcile. If a Job is already running,
stop that specific Job too; CronJob suspension only prevents future Jobs.
Suspension never reactivates users. Review any account remediation separately.

References: [Google service-account setup](https://developers.google.com/workspace/guides/create-credentials),
[Directory users API](https://developers.google.com/workspace/admin/directory/reference/rest/v1/users),
[Authentik user API](https://api.goauthentik.io/reference/core-users-partial-update/).
