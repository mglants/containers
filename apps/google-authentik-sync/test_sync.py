import unittest
from unittest.mock import Mock, patch

from sync import (Action, Authentik, Google, HTTP, MARKER, OWNER, SyncError,
                  execute, plan, run)


def google(identifier="1", email="user@example.com", **kw):
    return dict(id=identifier, primaryEmail=email, customerId="C123",
                name={"fullName": "Test User"}, suspended=False, **kw)


def user(managed=True, **kw):
    result = dict(pk=1, username="existing", email="user@example.com", name="Old",
                  is_active=True, is_superuser=False, type="internal", groups=["ops"],
                  roles=["role"], attributes={"other": {"keep": True}})
    if managed:
        result["attributes"][MARKER] = dict(owner=OWNER, customer_id="C123", user_id="1")
    result.update(kw)
    return result


class PlannerTests(unittest.TestCase):
    def planning(self, gs=None, users=None, lookup=None, exclusions=frozenset()):
        return plan(gs if gs is not None else [google()], users or [], "C123",
                    lookup or Mock(return_value=None), exclusions)

    def test_create(self):
        actions, errors, _ = self.planning()
        self.assertFalse(errors)
        body = actions[0].body
        self.assertEqual(body["username"], "user@example.com")
        self.assertEqual(body["groups"], [])
        self.assertNotIn("password", body)

    def test_adopt_preserves_privileges_username_and_attributes(self):
        before = user(False)
        actions, errors, _ = self.planning(users=[before])
        self.assertFalse(errors)
        body = actions[0].body
        for key in ("username", "groups", "roles", "type"):
            self.assertNotIn(key, body)
        self.assertEqual(body["attributes"]["other"], before["attributes"]["other"])

    def test_rename_matches_id(self):
        actions, errors, _ = self.planning([google(email="new@example.com")], [user()])
        self.assertFalse(errors)
        self.assertEqual(actions[0].body["email"], "new@example.com")
        self.assertNotIn("username", actions[0].body)

    def test_suspend_archive_and_delete(self):
        for flag in ("suspended", "archived", "deleted"):
            with self.subTest(flag=flag):
                g = google()
                g[flag] = True
                actions, errors, _ = self.planning([] if flag == "deleted" else [g], [user()])
                self.assertFalse(errors)
                self.assertFalse(actions[0].body["is_active"])
                self.assertTrue(actions[0].body["attributes"][MARKER]["disabled_by_sync"])

    def test_reactivate_only_sync_disabled(self):
        for sync_disabled, hold, expected in ((True, False, True), (False, False, False), (True, True, False)):
            with self.subTest(sync_disabled=sync_disabled, hold=hold):
                u = user(is_active=False)
                u["attributes"][MARKER].update(disabled_by_sync=sync_disabled, manual_hold=hold)
                actions, _, _ = self.planning(users=[u])
                self.assertEqual(actions[0].body["is_active"], expected)

    def test_manual_disabled_not_claimed(self):
        u, g = user(is_active=False), google()
        g["suspended"] = True
        actions, _, _ = self.planning([g], [u])
        self.assertNotIn("disabled_by_sync", actions[0].body["attributes"][MARKER])

    def test_conflicts(self):
        for gs, users in [
            ([google()], [user(False), user(False, pk=2, username="second")]),
            ([google("new-id")], [user()]),
            ([google()], [user(False, email="other@example.com", username="user@example.com")]),
            ([google(), google("2")], []),
            ([google()], [user(), user(pk=2)]),
            ([google(email="taken@example.com")], [user(), user(False, pk=2, email="taken@example.com")]),
        ]:
            with self.subTest(gs=gs, users=users):
                _, issues, _ = self.planning(gs, users)
                self.assertTrue(issues)

    def test_protected_users(self):
        for u, exclusions in [(user(is_superuser=True), set()), (user(type="service_account"), set()),
                              (user(), {"1"}), (user(), {"existing"})]:
            actions, _, skipped = self.planning(users=[u], exclusions=exclusions)
            self.assertFalse(actions)
            self.assertTrue(skipped)

    def test_deletion_lookup_failure_aborts(self):
        with self.assertRaises(SyncError):
            self.planning([], [user()], Mock(side_effect=SyncError("Google: HTTP 403")))

    def test_missing_but_exists_is_conflict(self):
        actions, issues, _ = self.planning([], [user()], Mock(return_value=google()))
        self.assertFalse(actions)
        self.assertTrue(issues)

    def test_idempotent(self):
        actions, _, _ = self.planning(users=[user()])
        updated = {**user(), **actions[0].body}
        self.assertEqual(self.planning(users=[updated])[0], [])

    def test_new_disabled_user_can_be_restored(self):
        g = google(); g["suspended"] = True
        created = self.planning([g])[0][0].body
        updated = dict(created, pk=9, is_superuser=False)
        self.assertTrue(self.planning(users=[updated])[0][0].body["is_active"])


class ClientTests(unittest.TestCase):
    def test_google_pages(self):
        http = Mock()
        http.request.side_effect = [{"users": [google()], "nextPageToken": "next"},
                                    {"users": [google("2", "two@example.com")]}]
        self.assertEqual(len(Google(http, "C123").users()), 2)
        self.assertEqual(http.request.call_args.kwargs["params"]["pageToken"], "next")

    def test_google_snapshot_guards(self):
        for responses in [[{}], [{"users": [dict(google(), customerId="C_OTHER")]}],
                          [{"users": [google()], "nextPageToken": "x"}] * 2]:
            http = Mock(); http.request.side_effect = responses
            with self.assertRaises(SyncError):
                Google(http, "C123").users()

    def test_authentik_pages(self):
        http = Mock()
        http.request.side_effect = [{"results": [user()], "pagination": {"next": 2}},
                                    {"results": [], "pagination": {"next": 0}}]
        self.assertEqual(len(Authentik(http, "https://sso.example.com").users()), 1)

    def test_http_errors_and_retry(self):
        for status in (401, 403, 404, 429, 503):
            session = Mock(); session.request.return_value.status_code = status
            with patch("sync.time.sleep"), self.assertRaises(SyncError):
                HTTP(session, "test").request("GET", "https://example.com")
            self.assertEqual(session.request.call_count, 4 if status in (429, 503) else 1)

    def test_http_writes_not_retried(self):
        session = Mock(); session.request.side_effect = TimeoutError("secret")
        with self.assertRaises(SyncError) as error:
            HTTP(session, "test").request("POST", "https://example.com")
        self.assertNotIn("secret", str(error.exception))
        self.assertEqual(session.request.call_count, 1)

    def test_confirmed_404_only(self):
        session = Mock(); session.request.return_value.status_code = 404
        self.assertIsNone(HTTP(session, "test").request("GET", "https://example.com", missing_ok=True))

    def test_partial_scan_no_writes(self):
        g, ak = Mock(), Mock()
        g.users.side_effect = SyncError("partial page")
        with self.assertRaises(SyncError):
            run(g, ak, "C123", set(), True)
        ak.create.assert_not_called(); ak.patch.assert_not_called()

    def test_conflict_prevents_all_writes(self):
        g, ak = Mock(), Mock()
        g.users.return_value = [google(), google("2", "fresh@example.com")]
        ak.users.return_value = [user(False), user(False, pk=2)]
        with patch("sync.emit"), self.assertRaises(SyncError):
            run(g, ak, "C123", set(), True)
        ak.create.assert_not_called(); ak.patch.assert_not_called()

    def test_later_google_page_failure_no_writes(self):
        http, ak = Mock(), Mock()
        http.request.side_effect = [{"users": [google()], "nextPageToken": "next"},
                                    SyncError("Google: HTTP 503")]
        with self.assertRaises(SyncError):
            run(Google(http, "C123"), ak, "C123", set(), True)
        ak.create.assert_not_called(); ak.patch.assert_not_called()

    def test_missing_authentik_pagination_fails(self):
        http = Mock(); http.request.return_value = {"results": [user()], "pagination": {}}
        with self.assertRaises(SyncError):
            Authentik(http, "https://sso.example.com").users()

    def test_dry_run_no_writes(self):
        g, ak = Mock(), Mock(); g.users.return_value = [google()]; ak.users.return_value = []
        with patch("sync.emit"):
            run(g, ak, "C123", set())
        ak.create.assert_not_called(); ak.patch.assert_not_called()

    def test_concurrent_change_stops_write(self):
        ak = Mock(); ak.get.return_value = user(is_active=False)
        with self.assertRaises(SyncError):
            execute(ak, [Action("update", "1", {"name": "new"}, user())], set())
        ak.patch.assert_not_called()

    def test_partial_write_failure_is_reported(self):
        ak = Mock(); ak.create.side_effect = [{}, SyncError("failure")]
        with patch("sync.emit"), self.assertRaises(SyncError):
            execute(ak, [Action("create", "1", {}), Action("create", "2", {})], set())
        self.assertEqual(ak.create.call_count, 2)


if __name__ == "__main__":
    unittest.main()
