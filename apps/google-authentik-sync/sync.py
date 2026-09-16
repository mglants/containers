"""One-way Workspace lifecycle reconciliation. No Google writes or database access."""

import argparse
import copy
import json
import os
import time
from collections import Counter, defaultdict
from dataclasses import dataclass
from urllib.parse import quote, urlparse

MARKER = "google_workspace_sync"
OWNER = "google-authentik-sync/v1"
SCOPE = "https://www.googleapis.com/auth/admin.directory.user.readonly"


class SyncError(Exception):
    """Safe-to-log error; never include HTTP bodies or credential contents."""


def emit(event, **fields):
    print(json.dumps({"event": event, **fields}, sort_keys=True), flush=True)


class HTTP:
    def __init__(self, session, label):
        self.session, self.label = session, label

    def request(self, method, url, *, missing_ok=False, **kwargs):
        # Never replay writes: a lost response might already have committed.
        for attempt in range(4 if method == "GET" else 1):
            try:
                response = self.session.request(
                    method, url, timeout=20, allow_redirects=False, **kwargs
                )
            except Exception as exc:
                if method == "GET" and attempt < 3:
                    time.sleep(2 ** attempt)
                    continue
                raise SyncError(f"{self.label}: transport failure ({type(exc).__name__})") from None
            if missing_ok and response.status_code == 404:
                return None
            if response.status_code in (429, 500, 502, 503, 504) and method == "GET" and attempt < 3:
                time.sleep(2 ** attempt)
                continue
            if not 200 <= response.status_code < 300:
                raise SyncError(f"{self.label}: HTTP {response.status_code}")
            try:
                data = response.json()
            except ValueError:
                raise SyncError(f"{self.label}: invalid JSON") from None
            if not isinstance(data, dict):
                raise SyncError(f"{self.label}: invalid response")
            return data


class Google:
    def __init__(self, http, customer):
        self.http, self.customer = http, customer
        self.url = "https://admin.googleapis.com/admin/directory/v1/users"

    def validate(self, user):
        if (not user.get("id") or not user.get("primaryEmail")
                or user.get("customerId") != self.customer
                or not isinstance(user.get("suspended"), bool)
                or not isinstance(user.get("archived", False), bool)):
            raise SyncError("Google: incomplete user or wrong customer")
        return user

    def users(self):
        result, seen, token = [], set(), None
        while True:
            params = {"customer": self.customer, "maxResults": 500,
                      "projection": "full", "viewType": "admin_view"}
            if token:
                params["pageToken"] = token
            data = self.http.request("GET", self.url, params=params)
            if not isinstance(data.get("users", []), list):
                raise SyncError("Google: invalid user listing")
            result.extend(self.validate(u) for u in data.get("users", []))
            token = data.get("nextPageToken")
            if not token:
                break
            if token in seen:
                raise SyncError("Google: repeated page token")
            seen.add(token)
        if not result:
            raise SyncError("Google: refusing empty customer snapshot")
        return result

    def get(self, identifier):
        data = self.http.request("GET", self.url + "/" + quote(identifier, safe=""),
                                 params={"projection": "full", "viewType": "admin_view"},
                                 missing_ok=True)
        return self.validate(data) if data is not None else None


class Authentik:
    def __init__(self, http, base):
        if urlparse(base).scheme != "https" or urlparse(base).username:
            raise SyncError("Authentik: HTTPS URL without credentials required")
        self.http = http
        self.url = base.rstrip("/") + "/api/v3/core/users/"

    def users(self):
        result, page = [], 1
        while True:
            data = self.http.request("GET", self.url, params={"page": page, "page_size": 100})
            if (not isinstance(data.get("results"), list)
                    or not isinstance(data.get("pagination"), dict)
                    or "next" not in data["pagination"]):
                raise SyncError("Authentik: invalid paginated listing")
            result.extend(data["results"])
            next_page = data["pagination"].get("next")
            if not next_page:
                return result
            if not isinstance(next_page, int) or next_page != page + 1:
                raise SyncError("Authentik: invalid next page")
            page = next_page

    def get(self, pk):
        return self.http.request("GET", f"{self.url}{pk}/")

    def create(self, body):
        return self.http.request("POST", self.url, json=body)

    def patch(self, pk, body):
        return self.http.request("PATCH", f"{self.url}{pk}/", json=body)


@dataclass
class Action:
    kind: str
    google_id: str
    body: dict
    before: dict | None = None


def marker(user):
    attrs = user.get("attributes", {})
    if not isinstance(attrs, dict):
        raise SyncError("Authentik: malformed attributes")
    value = attrs.get(MARKER, {})
    if not isinstance(value, dict):
        raise SyncError("Authentik: malformed sync marker")
    if value and (value.get("owner") != OWNER or not value.get("user_id") or not value.get("customer_id")):
        raise SyncError("Authentik: unknown or incomplete sync marker")
    return value


def protected(user, exclusions):
    return (user.get("type") not in ("internal", "external")
            or user.get("is_superuser", False)
            or str(user["pk"]) in exclusions
            or user.get("username", "").casefold() in exclusions)


def desired(user, google_user, customer, identifier):
    attrs = copy.deepcopy(user.get("attributes", {})) if user else {}
    state = copy.deepcopy(marker(user)) if user else {}
    state.update(owner=OWNER, customer_id=customer, user_id=identifier)
    inactive = google_user is None or google_user["suspended"] or google_user.get("archived", False)
    active = user["is_active"] if user else True
    reason = "deleted" if google_user is None else "suspended" if google_user["suspended"] else "archived"
    if inactive:
        if active or not user:
            state["disabled_by_sync"] = True
        active = False
        state["reason"] = reason
    elif state.get("disabled_by_sync") is True and not state.get("manual_hold", False):
        active = True
        state.pop("disabled_by_sync", None)
        state.pop("reason", None)
    if state.get("manual_hold", False):
        active = False
    attrs[MARKER] = state
    body = {"attributes": attrs, "is_active": active}
    if google_user:
        body.update(email=google_user["primaryEmail"].lower(),
                    name=google_user.get("name", {}).get("fullName") or google_user["primaryEmail"])
    return body


def plan(google_users, users, customer, lookup, exclusions=frozenset(), create_disabled_users=False,
         username_format="email", google_exclusions=frozenset(), username_collision_policy="error"):
    """Complete read-only planning, including deletion confirmations, before writes."""
    if username_format not in ("email", "local_part"):
        raise SyncError("USERNAME_FORMAT must be email or local_part")
    if username_collision_policy not in ("error", "email"):
        raise SyncError("USERNAME_COLLISION_POLICY must be error or email")
    actions, issues, skipped = [], [], []
    planned_usernames = set()
    by_id, by_email, by_name = defaultdict(list), defaultdict(list), defaultdict(list)
    for user in users:
        state = marker(user)
        if state.get("customer_id") == customer:
            by_id[state["user_id"]].append(user)
        by_email[user.get("email", "").casefold()].append(user)
        by_name[user["username"].casefold()].append(user)
    gids = Counter(g["id"] for g in google_users)
    emails = Counter(g["primaryEmail"].casefold() for g in google_users)
    # Count only eligible new identities, so all members of a collision receive
    # email usernames regardless of API listing order. Existing names are fixed.
    new_local_parts = Counter()
    for g in google_users:
        email = g["primaryEmail"].casefold()
        local_part = email.split("@", 1)[0]
        if (g["id"] not in google_exclusions and email not in google_exclusions
                and email not in exclusions and local_part not in exclusions
                and not by_id[g["id"]] and not by_email[email]
                and (create_disabled_users or not (g["suspended"] or g.get("archived", False)))):
            new_local_parts[local_part] += 1
    assigned = set()
    for g in google_users:
        gid, email = g["id"], g["primaryEmail"].casefold()
        if gid in google_exclusions or email in google_exclusions:
            skipped.append({"google_id": gid, "reason": "excluded_google_account"})
            continue
        username = email.split("@", 1)[0] if username_format == "local_part" else email
        if gids[gid] != 1 or emails[email] != 1:
            issues.append({"google_id": gid, "reason": "duplicate_google_identity"})
            continue
        matches = by_id[gid] or by_email[email]
        if len(matches) > 1:
            issues.append({"google_id": gid, "reason": "ambiguous_authentik_match"})
            continue
        user = matches[0] if matches else None
        if not user and not create_disabled_users and (g["suspended"] or g.get("archived", False)):
            skipped.append({"google_id": gid, "reason": "disabled_user_creation_disabled"})
            continue
        if user and protected(user, exclusions):
            skipped.append({"pk": user["pk"], "reason": "protected"})
            continue
        if not user and (email in exclusions or username in exclusions):
            skipped.append({"google_id": gid, "reason": "excluded_username"})
            continue
        if user:
            state = marker(user)
            if state and (state["user_id"] != gid or state["customer_id"] != customer):
                issues.append({"google_id": gid, "reason": "email_reused_or_foreign_owner"})
                continue
            if user["pk"] in assigned or any(u["pk"] != user["pk"] for u in by_email[email]):
                issues.append({"google_id": gid, "reason": "identity_collision"})
                continue
            assigned.add(user["pk"])
        else:
            if (username_format == "local_part" and username_collision_policy == "email"
                    and (by_name[username] or new_local_parts[username] > 1)):
                username = email
            # A full-email collision remains an error: never merge identities.
            if by_name[username] or username in planned_usernames or not username or len(username) > 150:
                issues.append({"google_id": gid, "reason": "username_collision_or_length"})
                continue
        body = desired(user, g, customer, gid)
        if not user:
            planned_usernames.add(username)
            body.update(username=username, type="external", path="goauthentik.io/sources/google",
                        groups=[], roles=[])
            actions.append(Action("create", gid, body))
        elif any(user.get(k) != v for k, v in body.items()):
            actions.append(Action("update", gid, body, user))
    for gid, matches in by_id.items():
        if gid in gids:
            continue
        # For deleted Google users, the last synchronized primary email is all
        # that remains. Immutable IDs are the reliable choice across renames.
        if gid in google_exclusions or any(u.get("email", "").casefold() in google_exclusions for u in matches):
            skipped.append({"google_id": gid, "reason": "excluded_google_account"})
            continue
        if len(matches) != 1:
            issues.append({"google_id": gid, "reason": "duplicate_stored_google_id"})
            continue
        user = matches[0]
        if protected(user, exclusions):
            skipped.append({"pk": user["pk"], "reason": "protected"})
            continue
        # A GET failure aborts the entire plan. Only an explicit 404 means deleted.
        if lookup(gid) is not None:
            issues.append({"google_id": gid, "reason": "inconsistent_google_snapshot"})
            continue
        body = desired(user, None, customer, gid)
        if any(user.get(k) != v for k, v in body.items()):
            actions.append(Action("update", gid, body, user))
    return actions, issues, skipped


def execute(authentik, actions, exclusions):
    for action in actions:
        if action.before is None:
            authentik.create(action.body)
        else:
            current = authentik.get(action.before["pk"])
            # Do not overwrite changes made since the snapshot, including admin holds.
            keys = ("username", "email", "name", "is_active", "attributes", "type", "is_superuser")
            if protected(current, exclusions) or any(current.get(k) != action.before.get(k) for k in keys):
                raise SyncError("Authentik: user changed during reconciliation; rerun required")
            authentik.patch(current["pk"], action.body)
        emit("applied", action=action.kind, google_id=action.google_id)


def run(google, authentik, customer, exclusions, apply=False, create_disabled_users=False,
        username_format="email", google_exclusions=frozenset(), username_collision_policy="error"):
    actions, issues, skipped = plan(google.users(), authentik.users(), customer, google.get,
                                    exclusions, create_disabled_users, username_format, google_exclusions,
                                    username_collision_policy)
    for item in skipped:
        emit("excluded", **item)
    for item in issues:
        emit("conflict", **item)
    for action in actions:
        emit("planned", action=action.kind, google_id=action.google_id,
             pk=action.before["pk"] if action.before else None,
             username=action.before["username"] if action.before else action.body["username"],
             is_active=action.body["is_active"], fields=sorted(action.body))
    if issues:
        raise SyncError("Reconciliation conflicts; no changes applied")
    if apply:
        execute(authentik, actions, exclusions)
    emit("success", mode="apply" if apply else "dry-run", actions=len(actions), excluded=len(skipped))


def env_bool(name, default=False):
    value = os.environ.get(name, str(default)).strip().lower()
    if value not in ("true", "false"):
        raise SyncError(f"{name} must be true or false")
    return value == "true"


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    mode = parser.add_mutually_exclusive_group()
    mode.add_argument("--apply", action="store_true")
    mode.add_argument("--dry-run", action="store_true")
    args = parser.parse_args()
    try:
        import requests
        from google.oauth2 import service_account
        from google.auth.transport.requests import AuthorizedSession

        create_disabled_users = env_bool("CREATE_DISABLED_USERS")
        username_format = os.environ.get("USERNAME_FORMAT", "email").strip().lower()
        if username_format not in ("email", "local_part"):
            raise SyncError("USERNAME_FORMAT must be email or local_part")
        username_collision_policy = os.environ.get("USERNAME_COLLISION_POLICY", "error").strip().lower()
        if username_collision_policy not in ("error", "email"):
            raise SyncError("USERNAME_COLLISION_POLICY must be error or email")
        customer = os.environ["GOOGLE_CUSTOMER_ID"]
        if not customer.startswith("C") or customer == "CHANGE_ME":
            raise SyncError("An explicit Google customer ID (C...) is required")
        credentials = service_account.Credentials.from_service_account_file(
            os.environ["GOOGLE_APPLICATION_CREDENTIALS"], scopes=[SCOPE])
        # Direct Workspace role assignment; no impersonation or domain-wide delegation.
        google = Google(HTTP(AuthorizedSession(credentials), "Google"), customer)
        token = os.environ.get("AUTHENTIK_TOKEN", "").strip()
        if not token or token.startswith("REPLACE_"):
            raise SyncError("Authentik API token has not been configured")
        session = requests.Session()
        session.headers["Authorization"] = "Bearer " + token
        authentik = Authentik(HTTP(session, "Authentik"), os.environ["AUTHENTIK_URL"])
        exclusions = frozenset(v.strip().casefold() for v in os.environ.get("EXCLUDED_USERS", "akadmin").split(",") if v.strip())
        google_exclusions = frozenset(v.strip().casefold() for v in os.environ.get("EXCLUDED_GOOGLE_USERS", "").split(",") if v.strip())
        run(google, authentik, customer, exclusions, args.apply, create_disabled_users,
            username_format, google_exclusions, username_collision_policy)
        return 0
    except SyncError as exc:
        emit("failure", error=str(exc))
    except Exception as exc:
        # Third-party exceptions can contain secrets, request bodies or tokens.
        emit("failure", error="Unexpected error", error_type=type(exc).__name__)
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
