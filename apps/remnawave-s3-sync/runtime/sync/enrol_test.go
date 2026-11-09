package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"os"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// The value was computed separately from this code. A client using the same settings
// must get the same name.
func TestRequestNameFormatIsFixed(t *testing.T) {
	got, err := fixtureEnrol().requestName("fixture-token-0123456789", "fixture-hwid-0001")
	if err != nil {
		t.Fatal(err)
	}
	if want := "uploads/4ff16d9d97d48b175f611b41f3022a79/014038352fd158f0df0a709bb915ae8b"; got != want {
		t.Fatalf("name format drifted: %s", got)
	}
}

func fixtureEnrol() *enrolConfig {
	return &enrolConfig{
		Endpoint: "storage.fixture", Bucket: "fixture", Region: "fixture-region", AccessKey: "FIXTUREID", SecretKey: "fixture-secret-value", Lifetime: 48 * time.Hour,
		Prefix: "uploads/", UserLabel: "fixture-user", RequestLabel: "fixture-request",
	}
}

func TestPolicyLimitsOneObjectAndHoldsNoSecret(t *testing.T) {
	p, err := fixtureEnrol().policy("uploads/u/d", time.Unix(1000, 0))
	if err != nil {
		t.Fatal(err)
	}
	doc, _ := base64.StdEncoding.DecodeString(p.Fields["policy"])
	for _, want := range []string{`["eq","$key","uploads/u/d"]`, `["content-length-range",0,8192]`} {
		if !strings.Contains(string(doc), want) {
			t.Fatalf("policy lacks %s: %s", want, doc)
		}
	}
	all, _ := json.Marshal(p)
	if strings.Contains(string(all)+string(doc), "fixture-secret-value") {
		t.Fatal("the secret key is in what the device receives")
	}
}

func openFile(t *testing.T, store *memoryStore, tok string) backupContent {
	t.Helper()
	store.mu.Lock()
	data := store.files[nameOf(t, tok)]
	store.mu.Unlock()
	if data == nil {
		t.Fatal("no file written")
	}
	content, err := fixtureFormat.open(tok, data)
	if err != nil {
		t.Fatal(err)
	}
	return content
}

func enrolledPanel() *panel {
	return &panel{
		users: []panelUser{
			{UUID: "u1", ShortUUID: token(1), Status: "ACTIVE"},
			{UUID: "u2", ShortUUID: token(2), Status: "ACTIVE"},
			{UUID: "u3", ShortUUID: token(3), Status: "LIMITED"},
		},
		devices: []panelDevice{{HWID: "phone", User: "u1"}, {HWID: "laptop", User: "u1"}, {HWID: "limited-device", User: "u3"}},
	}
}

func TestPoliciesPerDeviceInTheEncryptedFile(t *testing.T) {
	s, store, _ := setup(t, enrolledPanel(), nil)
	s.cfg.Enrol = fixtureEnrol()
	if _, err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	content := openFile(t, store, token(1))
	var names []string
	for name, p := range content.UploadPolicies {
		names = append(names, name)
		if p.Fields["key"] != name {
			t.Fatalf("policy for %s names %s", name, p.Fields["key"])
		}
		if strings.Contains(name, "phone") || strings.Contains(name, "laptop") {
			t.Fatal("a raw HWID in the file")
		}
	}
	sort.Strings(names)
	want := []string{}
	for _, h := range []string{"phone", "laptop"} {
		n, _ := fixtureEnrol().requestName(token(1), h)
		want = append(want, n)
	}
	sort.Strings(want)
	if strings.Join(names, ",") != strings.Join(want, ",") {
		t.Fatalf("%v, want %v", names, want)
	}
	if got := openFile(t, store, token(2)).UploadPolicies; len(got) != 0 {
		t.Fatalf("a user with no devices got %d policies", len(got))
	}
	store.mu.Lock()
	_, limited := store.files[nameOf(t, token(3))]
	store.mu.Unlock()
	if limited {
		t.Fatal("a limited user got a file")
	}
}

// The running panel links devices by numeric id, unlike its schema.
func TestDevicesLinkedByNumericID(t *testing.T) {
	p := &panel{
		users:   []panelUser{{ID: 7, UUID: "u7", ShortUUID: token(1), Status: "ACTIVE"}},
		devices: []panelDevice{{HWID: "phone", ID: 7}},
	}
	s, store, _ := setup(t, p, nil)
	s.cfg.Enrol = fixtureEnrol()
	if _, err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := openFile(t, store, token(1)).UploadPolicies; len(got) != 1 {
		t.Fatalf("%d policies", len(got))
	}
}

func TestEnrolSettingsAreRequiredNotAssumed(t *testing.T) {
	all := map[string]string{
		"ENROL_BUCKET": "b", "ENROL_ACCESS_KEY": "a", "ENROL_SECRET_KEY": "s", "ENROL_REGION": "r",
		"ENROL_PREFIX": "p/", "ENROL_USER_LABEL": "u", "ENROL_REQUEST_LABEL": "q",
	}
	set := func(skip string) {
		t.Setenv("S3_REGION", "")
		t.Setenv("ENROL_ENDPOINT", "")
		for k, v := range all {
			if k == skip {
				v = ""
			}
			t.Setenv(k, v)
		}
	}
	set("")
	e, err := enrolSettings("storage.fixture")
	if err != nil || e.Endpoint != "storage.fixture" || e.Region != "r" || e.Prefix != "p/" {
		t.Fatalf("%+v %v", e, err)
	}
	for k := range all {
		if k == "ENROL_BUCKET" {
			continue
		}
		set(k)
		if _, err := enrolSettings("storage.fixture"); err == nil {
			t.Fatalf("started without %s", k)
		}
	}
	set("ENROL_REGION")
	t.Setenv("S3_REGION", "from-s3")
	if e, err := enrolSettings("storage.fixture"); err != nil || e.Region != "from-s3" {
		t.Fatalf("%+v %v", e, err)
	}
	set("ENROL_BUCKET")
	if e, err := enrolSettings("storage.fixture"); err != nil || e != nil {
		t.Fatalf("no bucket should mean off: %+v %v", e, err)
	}
}

func TestOffByDefaultNoDeviceRequests(t *testing.T) {
	p := enrolledPanel()
	p.devErr = http.StatusTeapot
	s, store, logged := setup(t, p, nil)
	if _, err := s.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if got := openFile(t, store, token(1)); got.UploadPolicies != nil {
		t.Fatal("policies with enrolment off")
	}
	for _, line := range *logged {
		if strings.Contains(line, "upload policies") {
			t.Fatal("asked for devices with enrolment off")
		}
	}
	raw, _ := json.Marshal(openFile(t, store, token(1)))
	if strings.Contains(string(raw), "upload-policies") {
		t.Fatal("the file format changed with enrolment off")
	}
}

func TestDeviceListFailureDoesNotStopBackups(t *testing.T) {
	p := enrolledPanel()
	p.devErr = http.StatusBadGateway
	s, store, logged := setup(t, p, nil)
	s.cfg.Enrol = fixtureEnrol()
	result, err := s.run(context.Background())
	if err != nil || result.Uploaded != 2 {
		t.Fatalf("%+v %v", result, err)
	}
	if got := openFile(t, store, token(1)); got.UploadPolicies != nil || got.Config == "" {
		t.Fatalf("%+v", got)
	}
	found := false
	for _, line := range *logged {
		found = found || strings.Contains(line, "no upload policies this run")
		if strings.Contains(line, "phone") {
			t.Fatal("a HWID in the log")
		}
	}
	if !found {
		t.Fatal("not logged")
	}
}

// TestLivePolicyAcceptedByStorage uses a real test bucket, so it runs only when
// asked. Set ENROL_TEST_ENDPOINT, ENROL_TEST_REGION, ENROL_TEST_BUCKET, S3_ACCESS_KEY
// and S3_SECRET_KEY.
func TestLivePolicyAcceptedByStorage(t *testing.T) {
	bucket := os.Getenv("ENROL_TEST_BUCKET")
	if bucket == "" {
		t.Skip("set ENROL_TEST_BUCKET to use real storage")
	}
	e := fixtureEnrol()
	e.Endpoint, e.Region, e.Bucket = os.Getenv("ENROL_TEST_ENDPOINT"), os.Getenv("ENROL_TEST_REGION"), bucket
	e.AccessKey, e.SecretKey, e.Lifetime = os.Getenv("S3_ACCESS_KEY"), os.Getenv("S3_SECRET_KEY"), time.Hour
	name, _ := e.requestName("fixture-token-0123456789", "fixture-hwid-live")
	post := func(p uploadPolicy, size int) int {
		var body bytes.Buffer
		form := multipart.NewWriter(&body)
		for k, v := range p.Fields {
			form.WriteField(k, v)
		}
		file, _ := form.CreateFormFile("file", "request")
		file.Write(make([]byte, size))
		form.Close()
		resp, err := http.Post(p.URL, form.FormDataContentType(), &body)
		if err != nil {
			t.Fatal("upload failed")
		}
		resp.Body.Close()
		return resp.StatusCode
	}
	client, err := minio.New(e.Endpoint, &minio.Options{Creds: credentials.NewStaticV4(e.AccessKey, e.SecretKey, ""), Secure: true, Region: e.Region})
	if err != nil {
		t.Fatal("client failed")
	}
	defer client.RemoveObject(context.Background(), bucket, name, minio.RemoveObjectOptions{})
	p, _ := e.policy(name, time.Now())
	if st := post(p, 512); st/100 != 2 {
		t.Fatalf("a valid request refused: %d", st)
	}
	if st := post(p, 64<<10); st/100 == 2 {
		t.Fatalf("a 64 KiB request accepted: %d", st)
	}
	info, err := client.StatObject(context.Background(), bucket, name, minio.StatObjectOptions{})
	if err != nil || info.Size != 512 {
		t.Fatalf("stored size %d", info.Size)
	}
	t.Log("real storage accepted the job's policy and refused the oversized upload")
}
