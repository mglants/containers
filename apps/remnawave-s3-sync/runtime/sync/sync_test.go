package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Synthetic fixtures only; no real subscription, marker or key.
const (
	fixtureMarker = "fixture-marker"
	fixtureHeader = "x-fixture-backup"
)

var fixtureFormat = fileFormat{Magic: "FIXT1", NameInfo: "fixture-name", KeyInfo: "fixture-key-v1"}

func token(n int) string { return fmt.Sprintf("fixture-token-%06d", n) }

type memoryStore struct {
	mu       sync.Mutex
	files    map[string][]byte
	failPut  bool
	failList bool
}

func (m *memoryStore) Put(_ context.Context, name string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failPut {
		return fmt.Errorf("upload refused")
	}
	m.files[name] = data
	return nil
}

func (m *memoryStore) List(context.Context) ([]string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.failList {
		return nil, fmt.Errorf("list refused")
	}
	var names []string
	for name := range m.files {
		names = append(names, name)
	}
	return names, nil
}

func (m *memoryStore) Delete(_ context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, name)
	return nil
}

type panel struct {
	users   []panelUser
	answer  func(w http.ResponseWriter, r *http.Request, token string)
	listErr int
	seen    []http.Header
	mu      sync.Mutex
}

func (p *panel) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Like the real panel, which closes the connection without answering.
	if r.Header.Get("X-Forwarded-For") == "" || r.Header.Get("X-Forwarded-Proto") != "https" {
		panic(http.ErrAbortHandler)
	}
	if strings.HasPrefix(r.URL.Path, "/api/users") {
		if r.Header.Get("Authorization") != "Bearer fixture-api-token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if p.listErr != 0 {
			w.WriteHeader(p.listErr)
			return
		}
		var start, size int
		fmt.Sscan(r.URL.Query().Get("start"), &start)
		fmt.Sscan(r.URL.Query().Get("size"), &size)
		end := min(start+size, len(p.users))
		page := []panelUser{}
		if start < len(p.users) {
			page = p.users[start:end]
		}
		var body struct {
			Response struct {
				Users []panelUser `json:"users"`
				Total int         `json:"total"`
			} `json:"response"`
		}
		body.Response.Users, body.Response.Total = page, len(p.users)
		json.NewEncoder(w).Encode(body)
		return
	}
	p.mu.Lock()
	p.seen = append(p.seen, r.Header.Clone())
	p.mu.Unlock()
	name := strings.TrimPrefix(r.URL.Path, "/api/sub/")
	if p.answer != nil {
		p.answer(w, r, name)
		return
	}
	goodConfig(w)
}

func goodConfig(w http.ResponseWriter) {
	w.Header().Set("Profile-Title", "Fixture")
	w.Header().Set("Subscription-Userinfo", "upload=1; download=2; total=3; expire=0")
	w.Header().Set("Profile-Update-Interval", "6")
	w.Header().Set("X-Hwid-Active", "true")
	fmt.Fprintf(w, "x-fixture-marker: %s\nmode: rule\n", fixtureMarker)
}

func setup(t *testing.T, p *panel, files map[string][]byte) (*syncer, *memoryStore, *[]string) {
	t.Helper()
	server := httptest.NewServer(p)
	t.Cleanup(server.Close)
	sum := sha256.Sum256([]byte(fixtureMarker))
	if files == nil {
		files = map[string][]byte{}
	}
	store := &memoryStore{files: files}
	logged := &[]string{}
	var mu sync.Mutex
	return &syncer{
		cfg: config{
			APIURL: server.URL, APIToken: "fixture-api-token", SubBaseURL: server.URL + "/api/sub",
			UserAgent: defaultUserAgent, Format: fixtureFormat, BackupHeader: fixtureHeader,
			MarkerKey:        "x-fixture-marker",
			MarkerSHA256:     map[string]bool{hex.EncodeToString(sum[:]): true},
			MaxFailedPercent: 10, MaxDeletePercent: 50, Concurrency: 4,
		},
		client: newHTTPClient(),
		store:  store,
		now:    func() time.Time { return time.UnixMilli(1000) },
		logf: func(format string, args ...any) {
			mu.Lock()
			defer mu.Unlock()
			*logged = append(*logged, fmt.Sprintf(format, args...))
		},
	}, store, logged
}

func active(ids ...int) []panelUser {
	var users []panelUser
	for _, id := range ids {
		users = append(users, panelUser{ShortUUID: token(id), Status: "ACTIVE"})
	}
	return users
}

func nameOf(t *testing.T, tok string) string {
	t.Helper()
	name, err := fixtureFormat.fileName(tok)
	if err != nil {
		t.Fatal(err)
	}
	return name
}

func TestUploadsEncryptedCopies(t *testing.T) {
	s, store, _ := setup(t, &panel{users: append(active(1, 2), panelUser{ShortUUID: token(3), Status: "DISABLED"})}, nil)
	result, err := s.run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Users != 2 || result.Uploaded != 2 || len(store.files) != 2 {
		t.Fatalf("result %+v, files %d", result, len(store.files))
	}
	data := store.files[nameOf(t, token(1))]
	if bytes.Contains(data, []byte(fixtureMarker)) || bytes.Contains(data, []byte(token(1))) {
		t.Fatal("file exposes plaintext")
	}
	content, err := fixtureFormat.open(token(1), data)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(content.Config, fixtureMarker) || content.CreatedAt != 1000 {
		t.Fatal("wrong contents")
	}
	if content.Headers["profile-title"] != "Fixture" || content.Headers["profile-update-interval"] != "6" {
		t.Fatalf("headers %v", content.Headers)
	}
	if _, copied := content.Headers["x-hwid-active"]; copied {
		t.Fatal("device-limit header copied")
	}
	if store.files[nameOf(t, token(3))] != nil {
		t.Fatal("disabled user uploaded")
	}
}

func TestPagesThroughEveryUser(t *testing.T) {
	var ids []int
	for i := range usersPageSize + 3 {
		ids = append(ids, i)
	}
	s, store, _ := setup(t, &panel{users: active(ids...)}, nil)
	result, err := s.run(context.Background())
	if err != nil || result.Users != usersPageSize+3 || len(store.files) != usersPageSize+3 {
		t.Fatalf("result %+v: %v", result, err)
	}
}

func TestSendsTheUserAgentAndSyncToken(t *testing.T) {
	p := &panel{users: active(1)}
	s, _, _ := setup(t, p, nil)
	s.cfg.SyncToken = "fixture-sync"
	s.cfg.UserAgent = "fixture-agent"
	if _, err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	header := p.seen[0]
	if header.Get("User-Agent") != "fixture-agent" || header.Get(syncHeader) != "fixture-sync" || header.Get("x-hwid") != "" {
		t.Fatalf("request headers %v", header)
	}
}

func TestDeletesOnlyRemovedUsersOwnFiles(t *testing.T) {
	files := map[string][]byte{
		nameOf(t, token(1)):   []byte("old"),
		nameOf(t, token(2)):   []byte("old"),
		nameOf(t, token(3)):   []byte("old"),
		nameOf(t, token(9)):   []byte("removed user"),
		"not-one-of-ours.txt": []byte("keep"),
	}
	s, store, _ := setup(t, &panel{users: active(1, 2, 3)}, files)
	result, err := s.run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if result.Deleted != 1 || store.files[nameOf(t, token(9))] != nil {
		t.Fatalf("removed user's file kept: %+v", result)
	}
	if store.files["not-one-of-ours.txt"] == nil {
		t.Fatal("deleted a file this job did not write")
	}
}

func TestSafetyStopsKeepFiles(t *testing.T) {
	old := func(t *testing.T) map[string][]byte {
		return map[string][]byte{nameOf(t, token(8)): []byte("old"), nameOf(t, token(9)): []byte("old")}
	}
	for name, tc := range map[string]struct {
		users  []panelUser
		listE  int
		answer func(http.ResponseWriter, *http.Request, string)
		want   string
	}{
		"panel API down":    {users: active(1), listE: 500, want: "panel API answered 500"},
		"no active users":   {want: "no active users"},
		"too many failures": {users: active(1, 2), answer: func(w http.ResponseWriter, _ *http.Request, _ string) { w.WriteHeader(503) }, want: "nothing deleted"},
		"would delete most": {users: active(1), want: "would delete"},
		"marker missing":    {users: active(1), answer: func(w http.ResponseWriter, _ *http.Request, _ string) { fmt.Fprint(w, "mode: rule") }, want: "nothing deleted"},
		"device limit": {users: active(1), answer: func(w http.ResponseWriter, _ *http.Request, _ string) {
			w.Header().Set("X-Hwid-Not-Supported", "true")
			goodConfig(w)
		}, want: "nothing deleted"},
		"html answer": {users: active(1), answer: func(w http.ResponseWriter, _ *http.Request, _ string) {
			w.Header().Set("Content-Type", "text/html")
			fmt.Fprintf(w, "x-fixture-marker: %s", fixtureMarker)
		}, want: "nothing deleted"},
		"redirect not taken": {users: active(1), answer: func(w http.ResponseWriter, r *http.Request, _ string) {
			http.Redirect(w, r, "/elsewhere", http.StatusFound)
		}, want: "nothing deleted"},
	} {
		t.Run(name, func(t *testing.T) {
			files := old(t)
			s, store, _ := setup(t, &panel{users: tc.users, listErr: tc.listE, answer: tc.answer}, files)
			_, err := s.run(context.Background())
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error %v", err)
			}
			if store.files[nameOf(t, token(8))] == nil || store.files[nameOf(t, token(9))] == nil {
				t.Fatal("a safety stop deleted files")
			}
		})
	}
}

func TestSmallFailureKeepsThatUsersFile(t *testing.T) {
	var ids []int
	for i := range 20 {
		ids = append(ids, i)
	}
	bad := token(3)
	previous := []byte("previous copy")
	p := &panel{users: active(ids...), answer: func(w http.ResponseWriter, _ *http.Request, tok string) {
		if tok == bad {
			w.WriteHeader(503)
			return
		}
		goodConfig(w)
	}}
	s, store, _ := setup(t, p, map[string][]byte{nameOf(t, bad): previous})
	result, err := s.run(context.Background())
	if err != nil || result.Failed != 1 || result.Uploaded != 19 {
		t.Fatalf("result %+v: %v", result, err)
	}
	if !bytes.Equal(store.files[nameOf(t, bad)], previous) {
		t.Fatal("failing user's copy was removed")
	}
}

func TestMoveAndAddressesReachTheFile(t *testing.T) {
	s, store, _ := setup(t, &panel{users: active(1)}, nil)
	s.cfg.NewDomain = "repaired.example.invalid"
	s.cfg.BackupAddresses = []string{"https://a.example.invalid/x", "https://b.example.invalid/y"}
	if _, err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	content, err := fixtureFormat.open(token(1), store.files[nameOf(t, token(1))])
	if err != nil {
		t.Fatal(err)
	}
	if content.Headers["new-domain"] != "repaired.example.invalid" {
		t.Fatal("move missing")
	}
	if content.Headers[fixtureHeader] != "https://a.example.invalid/x,https://b.example.invalid/y" {
		t.Fatalf("addresses %q", content.Headers[fixtureHeader])
	}
}

func TestLogsExposeNoSecrets(t *testing.T) {
	p := &panel{users: active(1, 2), answer: func(w http.ResponseWriter, _ *http.Request, _ string) { w.WriteHeader(503) }}
	s, _, logged := setup(t, p, nil)
	_, err := s.run(context.Background())
	all := strings.Join(*logged, "\n") + fmt.Sprint(err)
	for _, secret := range []string{token(1), token(2), "fixture-api-token", "http://"} {
		if strings.Contains(all, secret) {
			t.Fatalf("output exposes %q", secret)
		}
	}
	if !strings.Contains(all, "status 503") {
		t.Fatalf("no reason given: %s", all)
	}
}

func TestBackupRejectsWrongKeyAndTampering(t *testing.T) {
	data, err := fixtureFormat.seal(token(1), backupContent{CreatedAt: 1, Config: "x"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fixtureFormat.open(token(2), data); err == nil {
		t.Fatal("another token opened the file")
	}
	changed := bytes.Clone(data)
	changed[len(changed)-1] ^= 1
	if _, err := fixtureFormat.open(token(1), changed); err == nil {
		t.Fatal("tampered file accepted")
	}
	if _, err := fixtureFormat.seal("short", backupContent{}); err == nil {
		t.Fatal("short token accepted")
	}
}

// A fixed file: if this fails, the job's files have stopped being readable by
// clients that already use this format.
const (
	formatToken = "fixture-token-format-0001"
	formatName  = "8070a7a870f62a52e309fff53c267758"
	formatFile  = "RklYVDHY/EMHAMcqXRX6peLc/tIECNLxfVTKzTpF0Usa1OdR0ybvuXUBUetfvTNpWFWIgo3rgTLy3OGhG3M3b5tl4LgvRSQqv19QBm4IUDtskbWx9KIx/rcBJ64SMIu9Hz7iUdQwcCdi5jQGbVGeRnw="
)

func TestOpensFileInTheFixedFormat(t *testing.T) {
	data, err := base64.StdEncoding.DecodeString(formatFile)
	if err != nil {
		t.Fatal(err)
	}
	content, err := fixtureFormat.open(formatToken, data)
	if err != nil {
		t.Fatal(err)
	}
	if content.Config != "mode: rule\n" || content.CreatedAt != 1234 || content.Headers["profile-title"] != "Format" {
		t.Fatalf("wrong contents: %+v", content)
	}
	if name := nameOf(t, formatToken); name != formatName {
		t.Fatalf("name %q", name)
	}
}
