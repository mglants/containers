// bbb-runtime renders native BBB configuration without a shell or root privileges.
package main

import (
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

//go:embed frontend.conf.tmpl
var frontendTemplate string

type object = map[string]any
type configFile struct{ path, name string }

var runtimeDir = "/tmp/bbb-config"
var defaultsDir = "/usr/share/bbb-container/defaults"
var files = map[string][]configFile{
	"web":                {{"/etc/bigbluebutton/bbb-web.properties", "web.properties"}, {"/etc/bigbluebutton/turn-stun-servers.xml", "turn.xml"}},
	"apps":               {{"/etc/bigbluebutton/bbb-apps-akka.conf", "apps.conf"}, {"/etc/bigbluebutton/bbb-html5.yml", "client.json"}},
	"notes":              {{"/etc/bigbluebutton/bbb-shared-notes-server.yml", "notes.json"}},
	"annotations":        {{"/usr/local/bigbluebutton/bbb-export-annotations/config/settings.json", "annotations.json"}},
	"graphql-middleware": {{"/etc/bigbluebutton/bbb-graphql-middleware.yml", "middleware.json"}},
	"signaling":          {{"/etc/bigbluebutton/bbb-webrtc-sfu/production.json", "signaling.json"}},
	"recorder":           {{"/etc/bbb-webrtc-recorder/bbb-webrtc-recorder.yml", "recorder.json"}},
	"recordings":         {{"/etc/bigbluebutton/recording/recording.yml", "recording.json"}, {"/etc/bigbluebutton/bbb-web.properties", "web.properties"}},
	"sip":                {{"/etc/bigbluebutton/livekit-sip.yaml", "sip.json"}},
	"frontend":           {{"/etc/nginx/nginx.conf", "nginx.conf"}},
	"plugins":            {{"/usr/share/nginx/html/live-transcription/manifest.json", "plugin.json"}},
}
var secretNames = []string{"SHARED_SECRET", "POSTGRES_PASSWORD", "REDIS_PASSWORD", "LIVEKIT_SECRET", "LIVEKIT_KEY", "LIVEKIT_API_SECRET", "LIVEKIT_API_KEY", "GLADIA_API_KEY", "TURN_SECRET", "TURN_EXT_SECRET", "PGPASSWORD", "HASURA_SECRET", "HASURA_GRAPHQL_ADMIN_SECRET", "DATABASE_URL", "SECRET_KEY_BASE", "SMTP_PASSWORD", "OPENID_CONNECT_CLIENT_SECRET", "BBB_CONFIG_JSON", "BBB_CLIENT_CONFIG_JSON"}

func env(name, fallback string) string {
	if s, ok := os.LookupEnv(name); ok {
		return s
	}
	return fallback
}
func required(name string) string {
	s := env(name, "")
	if s == "" {
		panic(fmt.Errorf("%s is required", name))
	}
	return s
}
func boolean(name string, fallback bool) bool {
	s, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	switch strings.ToLower(s) {
	case "true", "1":
		return true
	case "false", "0":
		return false
	}
	panic(fmt.Errorf("%s must be true or false", name))
}
func integer(name string, fallback, min, max int) int {
	s, ok := os.LookupEnv(name)
	if !ok {
		return fallback
	}
	n, e := strconv.Atoi(s)
	if e != nil || n < min || n > max {
		panic(fmt.Errorf("%s must be an integer from %d to %d", name, min, max))
	}
	return n
}
func resolveSecrets() {
	for _, name := range secretNames {
		if file, ok := os.LookupEnv(name + "_FILE"); ok {
			if _, exists := os.LookupEnv(name); exists {
				panic(fmt.Errorf("set only one of %s and %s_FILE", name, name))
			}
			b, err := os.ReadFile(file)
			if err != nil {
				panic(fmt.Errorf("cannot read %s_FILE", name))
			}
			os.Setenv(name, strings.TrimSuffix(strings.TrimSuffix(string(b), "\n"), "\r"))
		}
	}
}
func endpoint(raw, name string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Scheme != "https" && u.Scheme != "http" && u.Scheme != "ws" && u.Scheme != "wss") || strings.ContainsAny(u.Host, "\r\n\t $;{}\\\"") {
		panic(fmt.Errorf("%s must be an absolute HTTP(S) or WS(S) URL", name))
	}
	return u
}
func publicURL() *url.URL {
	raw := env("BBB_PUBLIC_URL", "")
	if raw == "" {
		raw = env("BBB_PUBLIC_SCHEME", "https") + "://" + required("DOMAIN")
	}
	u := endpoint(raw, "BBB_PUBLIC_URL")
	if u.Scheme != "http" && u.Scheme != "https" {
		panic(errors.New("BBB_PUBLIC_URL must use http or https"))
	}
	if u.Path != "" && u.Path != "/" {
		panic(errors.New("BBB_PUBLIC_URL must not have a path prefix"))
	}
	u.Path = ""
	return u
}
func wsURL(raw string) string {
	u := endpoint(raw, "WebSocket URL")
	if u.Scheme == "https" {
		u.Scheme = "wss"
	}
	if u.Scheme == "http" {
		u.Scheme = "ws"
	}
	return strings.TrimRight(u.String(), "/")
}
func httpURL(raw string) string {
	u := endpoint(raw, "HTTP URL")
	if u.Scheme == "wss" {
		u.Scheme = "https"
	}
	if u.Scheme == "ws" {
		u.Scheme = "http"
	}
	return strings.TrimRight(u.String(), "/")
}
func host(name, fallback string) string {
	s := env(name, fallback)
	if s == "" || strings.ContainsAny(s, "\r\n\t $;{}\\\"/") {
		panic(fmt.Errorf("%s must be a hostname or IP", name))
	}
	return s
}
func address(hostName string, port int) string { return net.JoinHostPort(hostName, strconv.Itoa(port)) }
func webURL() string {
	return httpURL(env("BBB_WEB_URL", "http://"+address(host("BBB_WEB_HOST", "web"), integer("BBB_WEB_PORT", 8090, 1, 65535))))
}
func redis() object {
	return object{"host": host("REDIS_HOST", "redis"), "port": integer("REDIS_PORT", 6379, 1, 65535), "password": env("REDIS_PASSWORD", "")}
}
func database(defaultDB string) object {
	return object{"host": host("POSTGRES_HOST", "postgres"), "port": integer("POSTGRES_PORT", 5432, 1, 65535), "database": env("POSTGRES_DATABASE", defaultDB), "user": env("POSTGRES_USER", "bbb"), "password": required("POSTGRES_PASSWORD")}
}
func jsonObject(raw, name string) object {
	var m object
	if json.Unmarshal([]byte(raw), &m) != nil || m == nil {
		panic(fmt.Errorf("%s must contain a JSON object", name))
	}
	return m
}
func merge(a, b object) object {
	for k, v := range b {
		if m, ok := v.(map[string]any); ok {
			if old, ok := a[k].(map[string]any); ok {
				v = merge(old, m)
			}
		}
		a[k] = v
	}
	return a
}
func patch(m object) object {
	if raw := env("BBB_CONFIG_JSON", ""); raw != "" {
		merge(m, jsonObject(raw, "BBB_CONFIG_JSON"))
	}
	return m
}
func bytesJSON(m object) []byte {
	b, e := json.MarshalIndent(m, "", "  ")
	if e != nil {
		panic(e)
	}
	return append(b, '\n')
}
func loadDefault(name string) object {
	b, e := os.ReadFile(filepath.Join(defaultsDir, name))
	if e != nil {
		panic(fmt.Errorf("missing packaged defaults: %s", name))
	}
	return jsonObject(string(b), "packaged defaults")
}
func propertyValue(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\':
			b.WriteString(`\\`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		case ' ':
			b.WriteString(`\ `)
		default:
			if r > 127 {
				if r > 65535 {
					r -= 65536
					fmt.Fprintf(&b, `\u%04x\u%04x`, 0xd800+(r>>10), 0xdc00+(r&1023))
				} else {
					fmt.Fprintf(&b, `\u%04x`, r)
				}
			} else {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
func properties(m object) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		if strings.ContainsAny(k, "\r\n= :\\\t") {
			panic(errors.New("invalid property key in BBB_CONFIG_JSON"))
		}
		var v string
		switch x := m[k].(type) {
		case string:
			v = x
		case bool:
			v = strconv.FormatBool(x)
		case int:
			v = strconv.Itoa(x)
		case float64:
			v = strconv.FormatFloat(x, 'f', -1, 64)
		default:
			panic(errors.New("web property values must be strings, numbers or booleans"))
		}
		b.WriteString(k + "=" + propertyValue(v) + "\n")
	}
	return []byte(b.String())
}
func hocon(m object) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		key, _ := json.Marshal(k)
		value, e := json.Marshal(m[k])
		if e != nil {
			panic(e)
		}
		b.Write(key)
		b.WriteString(" = ")
		b.Write(value)
		b.WriteByte('\n')
	}
	return []byte(b.String())
}
func webProperties() object {
	u := publicURL()
	record := boolean("ENABLE_RECORDING", true)
	m := object{"bigbluebutton.web.serverURL": u.String(), "securitySalt": required("SHARED_SECRET"), "redisHost": host("REDIS_HOST", "redis"), "redisPort": integer("REDIS_PORT", 6379, 1, 65535), "redisPassword": env("REDIS_PASSWORD", ""), "disableRecordingDefault": !record, "cameraBridge": "livekit", "screenShareBridge": "livekit", "audioBridge": "livekit", "learningDashboardEnabled": boolean("ENABLE_LEARNING_DASHBOARD", true)}
	for name, key := range map[string]string{"WELCOME_MESSAGE": "defaultWelcomeMessage", "WELCOME_FOOTER": "defaultWelcomeMessageFooter", "DEFAULT_PRESENTATION_URL": "beans.presentationService.defaultUploadedPresentation", "PLUGIN_MANIFESTS": "pluginManifests"} {
		if v, ok := os.LookupEnv(name); ok {
			m[key] = v
		}
	}
	return m
}
func clientConfig() object {
	u := publicURL()
	m := object{"public": object{"kurento": object{"wsUrl": wsURL(u.String()) + "/bbb-webrtc-sfu"}, "media": object{"livekit": object{"url": env("LIVEKIT_PUBLIC_URL", wsURL(u.String())+"/livekit")}}}}
	if raw := env("BBB_CLIENT_CONFIG_JSON", ""); raw != "" {
		merge(m, jsonObject(raw, "BBB_CLIENT_CONFIG_JSON"))
	}
	return m
}
func turnXML() []byte {
	const ns = "http://www.springframework.org/schema/beans"
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?><beans xmlns="` + ns + `" xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" xsi:schemaLocation="` + ns + ` http://www.springframework.org/schema/beans/spring-beans.xsd">`)
	var stuns, turns []string
	if stun := env("STUN_SERVER", ""); stun != "" {
		b.WriteString(`<bean id="stun0" class="org.bigbluebutton.web.services.turn.StunServer"><constructor-arg index="0" value="` + html.EscapeString(stun) + `"/></bean>`)
		stuns = append(stuns, "stun0")
	}
	servers := []struct{ uri, secret string }{}
	if secret := env("TURN_SECRET", ""); secret != "" {
		uri := env("TURN_SERVER", "turn:"+address(publicURL().Hostname(), 3478))
		servers = append(servers, struct{ uri, secret string }{uri, secret})
	}
	if uri := env("TURN_EXT_SERVER", ""); uri != "" {
		servers = append(servers, struct{ uri, secret string }{uri, required("TURN_EXT_SECRET")})
	}
	for i, s := range servers {
		if !strings.HasPrefix(s.uri, "turn:") && !strings.HasPrefix(s.uri, "turns:") {
			panic(errors.New("TURN_SERVER/TURN_EXT_SERVER must be turn: or turns: URIs"))
		}
		id := fmt.Sprintf("turn%d", i)
		turns = append(turns, id)
		fmt.Fprintf(&b, `<bean id="%s" class="org.bigbluebutton.web.services.turn.TurnServer"><constructor-arg index="0" value="%s"/><constructor-arg index="1" value="%s"/><constructor-arg index="2" value="%d"/></bean>`, id, html.EscapeString(s.secret), html.EscapeString(s.uri), integer("TURN_TTL", 86400, 1, 604800))
	}
	b.WriteString(`<bean id="stunTurnService" class="org.bigbluebutton.web.services.turn.StunTurnService">`)
	for name, ids := range map[string][]string{"stunServers": stuns, "turnServers": turns, "remoteIceCandidates": {}} {
		b.WriteString(`<property name="` + name + `"><set>`)
		for _, id := range ids {
			b.WriteString(`<ref bean="` + id + `"/>`)
		}
		b.WriteString(`</set></property>`)
	}
	b.WriteString(`</bean></beans>`)
	return []byte(b.String())
}
func render(role string) map[string][]byte {
	out := map[string][]byte{}
	var m object
	switch role {
	case "web":
		out["web.properties"] = properties(patch(webProperties()))
		out["turn.xml"] = turnXML()
	case "apps":
		db := database("bbb_graphql")
		m = object{"redis": redis(), "postgres": object{"properties": object{"serverName": db["host"], "portNumber": fmt.Sprint(db["port"]), "databaseName": db["database"], "user": db["user"], "password": db["password"]}}, "services": object{"bbbWebAPI": webURL() + "/bigbluebutton/api", "sharedSecret": required("SHARED_SECRET"), "graphqlMiddlewareAPI": env("BBB_MIDDLEWARE_URL", "http://middleware:8378")}, "http": object{"interface": "0.0.0.0"}}
		out["apps.conf"] = append([]byte("include \"/usr/share/bbb-apps-akka/conf/application.conf\"\n"), hocon(patch(m))...)
		out["client.json"] = bytesJSON(clientConfig())
	case "notes":
		w := endpoint(webURL(), "BBB_WEB_URL")
		if w.Scheme != "http" {
			panic(errors.New("notes requires an HTTP BBB_WEB_URL (upstream client limitation)"))
		}
		port := 80
		if w.Port() != "" {
			port, _ = strconv.Atoi(w.Port())
		}
		m = object{"expressServer": object{"host": "0.0.0.0"}, "hocuspocusServer": object{"host": "0.0.0.0"}, "bbbWeb": object{"host": w.Hostname(), "port": port}, "redis": redis(), "postgres": database("blocknote_app")}
		out["notes.json"] = bytesJSON(patch(m))
	case "annotations":
		m = loadDefault("annotations.json")
		m["redis"] = merge(m["redis"].(map[string]any), redis())
		m["bbbWebAPI"] = webURL()
		out["annotations.json"] = bytesJSON(patch(m))
	case "graphql-middleware":
		m = object{"server": object{"listen_host": "0.0.0.0"}, "redis": redis(), "hasura": object{"url": env("HASURA_WS_URL", "ws://hasura:8085/v1/graphql")}, "graphql-actions": object{"url": env("BBB_ACTIONS_URL", "http://actions:8093")}, "auth_hook": object{"url": webURL() + "/bigbluebutton/connection/checkGraphqlAuthorization"}, "session_vars_hook": object{"url": env("BBB_APPS_URL", "http://apps:8901") + "/userInfo"}}
		out["middleware.json"] = bytesJSON(patch(m))
	case "signaling":
		m = object{"redisHost": host("REDIS_HOST", "redis"), "redisPort": integer("REDIS_PORT", 6379, 1, 65535), "redisPassword": env("REDIS_PASSWORD", ""), "clientHost": "0.0.0.0", "log": object{"file": false}, "media-server-adapters": []any{}, "modules": []any{object{"path": "./lib/mcs-core/process.js", "name": "core", "dedicated": true, "ipc": object{"mode": "native", "options": object{}}}, object{"path": "./lib/livekit/livekit-process.js", "name": "livekit", "dedicated": true, "ipc": object{"mode": "native", "options": object{}}}}, "livekit": object{"enabled": true, "host": httpURL(env("LIVEKIT_HOST", "http://livekit:7880")), "bbbDomain": publicURL().Host, "key": required("LIVEKIT_KEY"), "secret": required("LIVEKIT_SECRET"), "recordingAdapter": "bbb-webrtc-recorder", "rtcAgent": object{"enabled": false}, "egress": object{"enabled": false}, "sip": object{"enabled": boolean("LIVEKIT_SIP_ENABLED", false), "requirePin": boolean("LIVEKIT_SIP_REQUIRE_PIN", true)}}}
		out["signaling.json"] = bytesJSON(patch(m))
	case "recorder":
		m = loadDefault("recorder.json")
		r := m["recorder"].(map[string]any)
		r["directory"] = env("RECORDING_MEDIA_DIR", "/var/lib/bbb-webrtc-recorder")
		r["dirFileMode"] = "0755"
		r["fileMode"] = "0644"
		adapter := m["pubsub"].(map[string]any)["adapters"].(map[string]any)["redis"].(map[string]any)
		adapter["address"] = address(host("REDIS_HOST", "redis"), integer("REDIS_PORT", 6379, 1, 65535))
		adapter["password"] = env("REDIS_PASSWORD", "")
		lk := m["livekit"].(map[string]any)
		lk["host"] = wsURL(env("LIVEKIT_HOST", "ws://livekit:7880"))
		lk["apiKey"] = required("LIVEKIT_KEY")
		lk["apiSecret"] = required("LIVEKIT_SECRET")
		lk["healthCheck"].(map[string]any)["enable"] = true
		lk["healthCheck"].(map[string]any)["abortBootOnFailure"] = true
		out["recorder.json"] = bytesJSON(patch(m))
	case "recordings":
		u := publicURL()
		formats := strings.Split(env("RECORDING_FORMATS", "presentation,video"), ",")
		var next []string
		steps := object{"archive": "sanity", "sanity": "captions"}
		for _, f := range formats {
			switch f {
			case "presentation", "video", "notes", "podcast":
			default:
				panic(errors.New("RECORDING_FORMATS supports presentation,video,notes,podcast"))
			}
			next = append(next, "process:"+f)
			steps["process:"+f] = "publish:" + f
		}
		steps["captions"] = next
		m = object{"redis_host": host("REDIS_HOST", "redis"), "redis_port": integer("REDIS_PORT", 6379, 1, 65535), "redis_password": env("REDIS_PASSWORD", ""), "redis_workers_host": host("REDIS_HOST", "redis"), "redis_workers_port": integer("REDIS_PORT", 6379, 1, 65535), "redis_workers_password": env("REDIS_PASSWORD", ""), "bn_notes_endpoint": env("BBB_NOTES_URL", "http://notes:8787") + "/loopback/api/documents", "playback_host": u.Host, "playback_protocol": u.Scheme, "video_compositing_parallel_workers": integer("RECORDING_WORKERS", 1, 1, 64), "steps": steps}
		if env("REDIS_PASSWORD", "") == "" {
			// Ruby's Redis client treats an empty string as an AUTH attempt.
			m["redis_password"] = nil
			m["redis_workers_password"] = nil
		}
		out["recording.json"] = bytesJSON(patch(m))
		out["web.properties"] = properties(webProperties())
	case "sip":
		m = object{"api_key": required("LIVEKIT_KEY"), "api_secret": required("LIVEKIT_SECRET"), "ws_url": wsURL(env("LIVEKIT_HOST", "ws://livekit:7880")), "redis": object{"address": address(host("REDIS_HOST", "redis"), integer("REDIS_PORT", 6379, 1, 65535)), "password": env("REDIS_PASSWORD", "")}, "sip_port": integer("SIP_PORT", 5062, 1, 65535), "sip_port_listen": integer("SIP_LISTEN_PORT", 5062, 1, 65535), "rtp_port": env("SIP_RTP_PORT_RANGE", "18000-18100"), "use_external_ip": boolean("SIP_USE_EXTERNAL_IP", false), "no_pin_wait_for_subscribe": false}
		if ip := env("SIP_PUBLIC_IP", ""); ip != "" {
			if net.ParseIP(ip) == nil {
				panic(errors.New("SIP_PUBLIC_IP must be an IP address"))
			}
			m["nat_1_to_1_ip"] = ip
		}
		out["sip.json"] = bytesJSON(patch(m))
	case "plugins":
		u := publicURL()
		out["plugin.json"] = bytesJSON(object{"requiredSdkVersion": "^1.0.0-beta.2", "name": "LiveTranscriptionPlugin", "version": "0.1.2", "javascriptEntrypointUrl": u.String() + "/plugins/live-transcription/LiveTranscriptionPlugin.js", "localesBaseUrl": u.String() + "/plugins/live-transcription/locales"})
	case "frontend":
		out["nginx.conf"] = renderFrontend()
	case "webhooks":
		required("SHARED_SECRET")
	case "stt":
		required("LIVEKIT_API_KEY")
		required("LIVEKIT_API_SECRET")
		required("GLADIA_API_KEY")
	case "graphql-actions", "bootstrap": // These components consume native environment variables.
	default:
		panic(errors.New("unknown runtime component"))
	}
	return out
}
func nativeEnvironment(role string) {
	alias := func(dst, src string) {
		if _, ok := os.LookupEnv(dst); !ok {
			if s, ok := os.LookupEnv(src); ok {
				os.Setenv(dst, s)
			}
		}
	}
	switch role {
	case "graphql-actions":
		alias("BBB_REDIS_HOST", "REDIS_HOST")
		alias("BBB_REDIS_PORT", "REDIS_PORT")
		alias("BBB_REDIS_PASSWORD", "REDIS_PASSWORD")
	case "webhooks":
		if env("SERVER_DOMAIN", "") == "" {
			if env("DOMAIN", "") != "" || env("BBB_PUBLIC_URL", "") != "" {
				os.Setenv("SERVER_DOMAIN", publicURL().Host)
			}
		}
		if env("SERVER_BIND_IP", "") == "" {
			os.Setenv("SERVER_BIND_IP", "0.0.0.0")
		}
	case "stt":
		alias("LIVEKIT_API_KEY", "LIVEKIT_KEY")
		alias("LIVEKIT_API_SECRET", "LIVEKIT_SECRET")
		if env("LIVEKIT_URL", "") == "" {
			os.Setenv("LIVEKIT_URL", wsURL(env("LIVEKIT_HOST", "ws://livekit:7880")))
		}
	}
}
func writeConfig(role string, contents map[string][]byte) error {
	dir := runtimeDir
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	for _, f := range files[role] {
		info, err := os.Lstat(f.path)
		if err != nil || info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("%s uses mounted configuration; select BBB_CONFIG_MODE=file", role)
		}
		target, err := os.Readlink(f.path)
		if err != nil || target != filepath.Join(dir, f.name) {
			return errors.New("unexpected configuration link")
		}
		if b, ok := contents[f.name]; ok {
			tmp := target + ".new"
			if err := os.WriteFile(tmp, b, 0600); err != nil {
				return err
			}
			if err := os.Rename(tmp, target); err != nil {
				return err
			}
		}
	}
	return nil
}
func restoreDefaults(role string) error {
	for _, f := range files[role] {
		if info, e := os.Lstat(f.path); e == nil && info.Mode()&os.ModeSymlink != 0 {
			if _, e := os.Stat(f.path); os.IsNotExist(e) {
				if b, e := os.ReadFile("/usr/share/bbb-container/originals/" + f.name); e == nil {
					if e = writeConfigFile(f.name, b); e != nil {
						return e
					}
				}
			}
		}
	}
	return nil
}
func writeConfigFile(name string, b []byte) error {
	if e := os.MkdirAll(runtimeDir, 0700); e != nil {
		return e
	}
	return os.WriteFile(filepath.Join(runtimeDir, name), b, 0600)
}
func run(args []string) error {
	if len(args) < 1 {
		return errors.New("usage: bbb-runtime COMPONENT [--render-only] COMMAND...")
	}
	role := args[0]
	args = args[1:]
	resolveSecrets()
	nativeEnvironment(role)
	mode := env("BBB_CONFIG_MODE", "auto")
	if mode != "env" && mode != "file" && mode != "auto" {
		return errors.New("BBB_CONFIG_MODE must be auto, env or file")
	}
	active := mode == "env" || mode == "auto" && (role == "frontend" || env("DOMAIN", "") != "" || env("BBB_PUBLIC_URL", "") != "")
	if active {
		if e := writeConfig(role, render(role)); e != nil {
			return e
		}
	} else if e := restoreDefaults(role); e != nil {
		return e
	}
	if len(args) > 0 && args[0] == "--render-only" {
		return nil
	}
	if len(args) == 0 {
		return errors.New("missing application command")
	}
	cmd, e := exec.LookPath(args[0])
	if e != nil {
		return errors.New("application command not found")
	}
	return syscall.Exec(cmd, args, os.Environ())
}
func main() {
	// Do not echo config values or panic stacks: errors may concern credentials.
	defer func() {
		if e := recover(); e != nil {
			fmt.Fprintln(os.Stderr, "bbb-runtime:", e)
			os.Exit(2)
		}
	}()
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "bbb-runtime:", err)
		os.Exit(2)
	}
}
