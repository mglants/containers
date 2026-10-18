package main

import (
	"encoding/json"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func isolated(t *testing.T) {
	t.Helper()
	for _, name := range append(secretNames, "DOMAIN", "BBB_PUBLIC_URL", "BBB_CONFIG_MODE", "REDIS_PORT", "ENABLE_RECORDING", "POSTGRES_HOST", "POSTGRES_USER", "LIVEKIT_HOST", "BBB_TLS_CERT_FILE", "BBB_TLS_KEY_FILE", "GREENLIGHT_PUBLIC_URL", "BBB_CLIENT_CONFIG_JSON") {
		for _, n := range []string{name, name + "_FILE"} {
			old, ok := os.LookupEnv(n)
			os.Unsetenv(n)
			t.Cleanup(func() {
				if ok {
					os.Setenv(n, old)
				} else {
					os.Unsetenv(n)
				}
			})
		}
	}
	t.Setenv("DOMAIN", "meet.example.test:8443")
	t.Setenv("SHARED_SECRET", " local\\secret=&\nрусский😀")
	t.Setenv("POSTGRES_PASSWORD", "DB\"\\${not-a-substitution}\npassword")
	t.Setenv("LIVEKIT_KEY", "test-key")
	t.Setenv("LIVEKIT_SECRET", "test-secret")
}
func panics(t *testing.T, fn func(), label string) {
	t.Helper()
	defer func() {
		if recover() == nil {
			t.Fatalf("%s should fail", label)
		}
	}()
	fn()
}
func TestURLPortsAndIPv6(t *testing.T) {
	isolated(t)
	for _, raw := range []string{"https://meet.example.test:8443", "http://[::1]:8080"} {
		t.Setenv("BBB_PUBLIC_URL", raw)
		m := clientConfig()
		got := m["public"].(map[string]any)["media"].(map[string]any)["livekit"].(map[string]any)["url"]
		want := strings.Replace(raw, "http", "ws", 1) + "/livekit"
		if got != want {
			t.Fatalf("URL/port lost: %s", got)
		}
	}
	for _, raw := range []string{"https://u:p@example.test", "https://example.test/prefix", "https://example.test/?token=secret", "https://example.test/#fragment", "https://evil$var", "ws://example.test"} {
		t.Setenv("BBB_PUBLIC_URL", raw)
		panics(t, func() { publicURL() }, "invalid public URL")
	}
}
func TestEscapingAndTypedConfig(t *testing.T) {
	isolated(t)
	props := string(render("web")["web.properties"])
	if !strings.Contains(props, `securitySalt=\ local\\secret=&\n\u0440`) || strings.Contains(props, "русский") || strings.Contains(props, "\nрусский") {
		t.Fatal("Java properties escaping failed")
	}
	app := string(render("apps")["apps.conf"])
	if !strings.HasPrefix(app, "include ") || !strings.Contains(app, `DB\"\\${not-a-substitution}\npassword`) {
		t.Fatal("HOCON string escaping failed")
	}
	var db object
	if err := json.Unmarshal(render("notes")["notes.json"], &db); err != nil {
		t.Fatal(err)
	}
	if db["postgres"].(map[string]any)["password"] != os.Getenv("POSTGRES_PASSWORD") {
		t.Fatal("DB password changed during serialization")
	}
	t.Setenv("ENABLE_RECORDING", "false")
	if !strings.Contains(string(render("web")["web.properties"]), "disableRecordingDefault=true") {
		t.Fatal("boolean inverted")
	}
	t.Setenv("ENABLE_RECORDING", "tru")
	panics(t, func() { render("web") }, "misspelled boolean")
	t.Setenv("REDIS_PORT", "0")
	panics(t, func() { redis() }, "invalid port")
}
func TestSecretFilesAndConflicts(t *testing.T) {
	isolated(t)
	path := filepath.Join(t.TempDir(), "secret")
	os.WriteFile(path, []byte(" leading spaces stay \r\n"), 0600)
	os.Unsetenv("SHARED_SECRET")
	t.Setenv("SHARED_SECRET_FILE", path)
	resolveSecrets()
	if os.Getenv("SHARED_SECRET") != " leading spaces stay " {
		t.Fatal("secret-file whitespace corrupted")
	}
	panics(t, resolveSecrets, "direct/file conflict")
}
func TestTURNXMLAndNoImplicitExternalSTUN(t *testing.T) {
	isolated(t)
	t.Setenv("STUN_SERVER", "")
	t.Setenv("TURN_SECRET", "")
	t.Setenv("TURN_EXT_SERVER", "")
	if strings.Contains(string(turnXML()), "google") || strings.Contains(string(turnXML()), `id="stun0"`) {
		t.Fatal("unexpected STUN service")
	}
	t.Setenv("STUN_SERVER", "stun:turn.example.test:3480")
	t.Setenv("TURN_EXT_SERVER", "turns:turn.example.test:5349?transport=tcp")
	t.Setenv("TURN_EXT_SECRET", `quote"<&secret`)
	var data any // XML tokenization proves escaping and document validity.
	_ = data
	d := xml.NewDecoder(strings.NewReader(string(turnXML())))
	for {
		_, e := d.Token()
		if e != nil {
			if e.Error() != "EOF" {
				t.Fatal(e)
			}
			break
		}
	}
	if !strings.Contains(string(turnXML()), "&lt;&amp;") {
		t.Fatal("XML secret not escaped")
	}
}
func TestMergePreservesNestedDefaults(t *testing.T) {
	m := merge(object{"x": object{"a": 1, "b": 2}}, object{"x": object{"a": 3}})
	if m["x"].(map[string]any)["b"] != 2 {
		t.Fatal("nested default lost")
	}
}
func TestNotesRecordingWorkflow(t *testing.T) {
	isolated(t)
	t.Setenv("RECORDING_FORMATS", "presentation,video,notes")
	var config struct {
		Steps map[string]json.RawMessage `json:"steps"`
	}
	if err := json.Unmarshal(render("recordings")["recording.json"], &config); err != nil {
		t.Fatal(err)
	}
	var formats []string
	if err := json.Unmarshal(config.Steps["captions"], &formats); err != nil {
		t.Fatal(err)
	}
	if strings.Join(formats, ",") != "process:presentation,process:video,process:notes" || string(config.Steps["process:notes"]) != `"publish:notes"` {
		t.Fatal("notes must be processed and published alongside the existing recording formats")
	}
}
func TestPrivateRuntimeFilesAndMountedConfig(t *testing.T) {
	isolated(t)
	old := runtimeDir
	runtimeDir = t.TempDir() + "/config"
	t.Cleanup(func() { runtimeDir = old })
	native := filepath.Join(t.TempDir(), "web.properties")
	os.Symlink(filepath.Join(runtimeDir, "test.properties"), native)
	files["test"] = []configFile{{native, "test.properties"}}
	t.Cleanup(func() { delete(files, "test") })
	if e := writeConfig("test", map[string][]byte{"test.properties": []byte("private")}); e != nil {
		t.Fatal(e)
	}
	info, _ := os.Stat(filepath.Join(runtimeDir, "test.properties"))
	dir, _ := os.Stat(runtimeDir)
	if info.Mode().Perm() != 0600 || dir.Mode().Perm() != 0700 {
		t.Fatal("private config permissions lost")
	}
	os.Remove(native)
	os.WriteFile(native, []byte("mounted"), 0444)
	if e := writeConfig("test", map[string][]byte{"test.properties": []byte("overwritten")}); e == nil {
		t.Fatal("mounted config was accepted in env mode")
	}
	b, _ := os.ReadFile(native)
	if string(b) != "mounted" {
		t.Fatal("mounted file changed")
	}
}
func TestFrontendBehindTLSProxyAndOptionalTLS(t *testing.T) {
	isolated(t)
	t.Setenv("DNS_RESOLVER", "127.0.0.11")
	config := string(renderFrontend())
	if strings.Contains(config, " ssl;") || !strings.Contains(config, "proxy_set_header X-Forwarded-Proto https;") {
		t.Fatal("HTTP behind HTTPS proxy rendering failed")
	}
	t.Setenv("BBB_TLS_CERT_FILE", "/cert/tls.crt")
	panics(t, func() { renderFrontend() }, "partial TLS config")
	t.Setenv("BBB_TLS_KEY_FILE", "/cert/tls.key")
	if !strings.Contains(string(renderFrontend()), "listen 8443 ssl;") {
		t.Fatal("TLS listener missing")
	}
}
func TestNativeAliases(t *testing.T) {
	isolated(t)
	t.Setenv("REDIS_HOST", "cache")
	os.Unsetenv("BBB_REDIS_HOST")
	nativeEnvironment("graphql-actions")
	if os.Getenv("BBB_REDIS_HOST") != "cache" {
		t.Fatal("native actions alias missing")
	}
	os.Unsetenv("LIVEKIT_API_SECRET")
	nativeEnvironment("stt")
	if os.Getenv("LIVEKIT_API_SECRET") != "test-secret" {
		t.Fatal("STT key alias missing")
	}
}

func TestRecordingRedisNoAuthAndAuth(t *testing.T) {
	isolated(t)
	for _, password := range []string{"", "auth-secret"} {
		t.Setenv("REDIS_PASSWORD", password)
		var m object
		if err := json.Unmarshal(render("recordings")["recording.json"], &m); err != nil {
			t.Fatal(err)
		}
		if password == "" {
			if m["redis_password"] != nil || m["redis_workers_password"] != nil {
				t.Fatal("empty password would trigger Ruby Redis AUTH")
			}
		} else if m["redis_password"] != password || m["redis_workers_password"] != password {
			t.Fatal("Redis authentication secret missing")
		}
	}
}

func TestFrontendLandingUsesAFileNotDirectoryAlias(t *testing.T) {
	isolated(t)
	t.Setenv("DNS_RESOLVER", "127.0.0.11")
	t.Setenv("BBB_LANDING_FILE", "/local/landing.html")
	config := string(renderFrontend())
	if !strings.Contains(config, `root "/local"; try_files "/landing.html" =404;`) {
		t.Fatal("root launcher must serve a file without invoking the directory index")
	}
}

func TestNativeServicesRejectMissingSecretsInEnvMode(t *testing.T) {
	isolated(t)
	os.Unsetenv("SHARED_SECRET")
	panics(t, func() { render("webhooks") }, "missing webhook API secret")
	t.Setenv("LIVEKIT_API_KEY", "test-key")
	t.Setenv("LIVEKIT_API_SECRET", "test-secret")
	os.Unsetenv("GLADIA_API_KEY")
	panics(t, func() { render("stt") }, "missing transcription provider key")
	t.Setenv("GLADIA_API_KEY", "test-only-gladia")
	render("stt")
}

func TestFrontendAutoModePreservesStandaloneAssets(t *testing.T) {
	isolated(t)
	t.Setenv("DOMAIN", "")
	t.Setenv("BBB_PUBLIC_URL", "")
	t.Setenv("BBB_CONFIG_MODE", "auto")
	oldDir, oldFiles := runtimeDir, files["frontend"]
	runtimeDir = filepath.Join(t.TempDir(), "config")
	native := filepath.Join(t.TempDir(), "nginx.conf")
	if e := os.Symlink(filepath.Join(runtimeDir, "nginx.conf"), native); e != nil {
		t.Fatal(e)
	}
	files["frontend"] = []configFile{{native, "nginx.conf"}}
	t.Cleanup(func() { runtimeDir = oldDir; files["frontend"] = oldFiles })
	if e := run([]string{"frontend", "--render-only"}); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile(native)
	if e != nil {
		t.Fatal(e)
	}
	config := string(b)
	if !strings.Contains(config, "listen 8080;") || !strings.Contains(config, "location /html5client/") || strings.Contains(config, "include /usr/share/bbb-container/nginx/") {
		t.Fatal("standalone frontend must serve assets without requiring backends")
	}
	t.Setenv("BBB_CONFIG_MODE", "env")
	panics(t, func() { renderFrontend() }, "explicit env mode without public URL")
}
