# Remnawave S3 sync

Keeps encrypted copies of Remnawave subscriptions in S3-compatible storage, for clients to download and decrypt when the panel can't be reached. Run it on a schedule; it runs once and exits.

Each run lists users from the panel API, fetches each `ACTIVE` user's subscription, checks its marker, encrypts it and uploads it. It then deletes files of users who are gone or no longer active.

The file's name and AES-256-GCM key derive from the user's subscription token with HKDF, so storage learns neither the user nor the contents.

## File format

A file is the magic, a 12-byte nonce, then the ciphertext with its tag; the magic is also the additional authenticated data. The plaintext is JSON with `created-at`, `headers` and `config`. The name is 16 bytes of HKDF-SHA256 output in hex, and the key is 32 bytes.

The client must use the same `FORMAT_*` values, or it won't find or open the files.

## Settings

| Variable                                | Required  | Meaning                                                                                                   |
| --------------------------------------- | --------- | --------------------------------------------------------------------------------------------------------- |
| `RW_API_URL`                            | yes       | Panel address, such as `http://remnawave:3000`                                                            |
| `RW_API_TOKEN`                          | yes       | Panel API token, which can read every user. Treat it like the panel.                                      |
| `RW_SUB_BASE_URL`                       | no        | Base of subscription links; defaults to `RW_API_URL/api/sub`                                              |
| `RW_SYNC_TOKEN`                         | see below | Sent as the `x-sync-token` header                                                                         |
| `USER_AGENT`                            | no        | Sent when fetching subscriptions; defaults to `mihomo`                                                    |
| `FORMAT_MAGIC`                          | yes       | Bytes that start each file                                                                                |
| `FORMAT_NAME_LABEL`, `FORMAT_KEY_LABEL` | yes       | HKDF info values for the file name and key                                                                |
| `MARKER_KEY`                            | yes       | Top-level config key that marks a subscription as yours                                                   |
| `MARKER_SHA256`                         | yes       | SHA-256 of that key's value, comma-separated if several                                                   |
| `BACKUP_HEADER`                         | yes       | Header that carries storage addresses, read from the panel and written to each file                       |
| `S3_ENDPOINT`, `S3_BUCKET`              | yes       | Storage host, without scheme, and bucket                                                                  |
| `S3_ACCESS_KEY`, `S3_SECRET_KEY`        | yes       | Must allow listing, writing and deleting in the bucket                                                    |
| `S3_REGION`, `S3_PREFIX`                | no        | Region, and folder inside the bucket                                                                      |
| `BACKUP_ADDRESSES`                      | no        | Comma-separated addresses to put in each file; defaults to the panel's `BACKUP_HEADER` header             |
| `NEW_DOMAIN`                            | no        | Tells clients to move to this domain. Test the domain first: a wrong one moves users to a broken address. |
| `MAX_FAILED_PERCENT`                    | no        | Default 10. Above this share of failed fetches, nothing is deleted and the run fails.                     |
| `MAX_DELETE_PERCENT`                    | no        | Default 50. Refuses to delete more than this share of files.                                              |
| `CONCURRENCY`                           | no        | Default 4                                                                                                 |

## Device limits

With Remnawave's HWID limit on, a request without `x-hwid` gets no subscription, and sending one would use a device slot for every user. Add a response rule that matches this job's secret header and turns the check off for it, so no slots are used. The job treats a device-limit answer as a failure and stores nothing.

```json
{
  "name": "Remnawave S3 sync",
  "enabled": true,
  "operator": "AND",
  "conditions": [
    {
      "headerName": "x-sync-token",
      "operator": "EQUALS",
      "value": "<long random value>",
      "caseSensitive": true
    }
  ],
  "responseType": "MIHOMO",
  "responseModifications": { "disableHwidCheck": true }
}
```

Check the exact fields against your panel's version. Anyone holding that value can skip the limit for a link they already have, so keep it secret, and don't send it through anything that logs headers.

Point `RW_API_URL` at the panel's internal address where you can, so that the token doesn't cross the public internet.

## Safety

- If the panel lists no active users, or too many fetches fail, or too many files would go, nothing is deleted.
- A user whose fetch fails keeps their previous copy.
- Only files named like this job's output are ever deleted, so a folder can hold other files.
- A subscription without the marker isn't stored, because a client that checks the marker would refuse it.
- Logs hold counts and fixed failure reasons, never links, tokens or configs.

The storage bucket must allow public reads of files and must not allow public listing.

## Build and tests

The repository workflows discover `docker-bake.hcl`, declaring version `0.1.0` and amd64/arm64 builds. The image runs as UID/GID 65532.

From the repository root: `mise run local-build remnawave-s3-sync`. Alternatively, from this app directory:

```sh
docker buildx bake --load
TEST_IMAGE=remnawave-s3-sync:0.1.0 go test -count=1 -v ./...
```

Builds run the Go tests in `runtime/sync`; the container test checks the binary in the finished image.
