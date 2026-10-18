package main

import (
	"bytes"
	"errors"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"text/template"
)

func nginxPath(name string) string {
	p := env(name, "")
	if p != "" && (!strings.HasPrefix(p, "/") || strings.ContainsAny(p, "\r\n$\\\";{}")) {
		panic(errors.New(name + " must be an absolute file path"))
	}
	return strconv.Quote(p)
}
func resolver() string {
	s := env("DNS_RESOLVER", "")
	if s == "" {
		b, e := os.ReadFile("/etc/resolv.conf")
		if e != nil {
			panic(errors.New("set DNS_RESOLVER"))
		}
		for _, line := range strings.Split(string(b), "\n") {
			fields := strings.Fields(line)
			if len(fields) > 1 && fields[0] == "nameserver" {
				s = fields[1]
				break
			}
		}
	}
	ip := net.ParseIP(s)
	if ip == nil {
		panic(errors.New("DNS_RESOLVER must be an IP address"))
	}
	if ip.To4() == nil {
		return "[" + s + "]"
	}
	return s
}
func renderFrontend() []byte {
	configured := env("BBB_PUBLIC_URL", "") != "" || env("DOMAIN", "") != ""
	u := endpoint("http://localhost", "frontend default")
	if configured || env("BBB_CONFIG_MODE", "auto") == "env" {
		u = publicURL()
	}
	type upstream struct{ Name, Address string }
	var upstreams []upstream
	for _, s := range []struct {
		name, host, variable string
		port                 int
	}{{"web", "web", "WEB", 8090}, {"middleware", "middleware", "MIDDLEWARE", 8378}, {"hasura", "hasura", "HASURA", 8085}, {"notes", "notes", "NOTES", 8787}, {"signaling", "signaling", "SIGNALING", 3008}, {"livekit", "livekit", "LIVEKIT", 7880}, {"plugins", "plugins", "PLUGINS", 8080}, {"webhooks", "webhooks", "WEBHOOKS", 3005}} {
		upstreams = append(upstreams, upstream{"bbb-" + s.name, address(host("BBB_"+s.variable+"_HOST", s.host), integer("BBB_"+s.variable+"_PORT", s.port, 1, 65535))})
	}
	tls := env("BBB_TLS_CERT_FILE", "") != ""
	if tls != (env("BBB_TLS_KEY_FILE", "") != "") {
		panic(errors.New("set both BBB_TLS_CERT_FILE and BBB_TLS_KEY_FILE"))
	}
	gl := env("GREENLIGHT_PUBLIC_URL", "") != ""
	glPublicPort := ""
	if gl {
		if !tls {
			panic(errors.New("local Greenlight TLS listener requires a certificate"))
		}
		g := endpoint(env("GREENLIGHT_PUBLIC_URL", ""), "GREENLIGHT_PUBLIC_URL")
		glPublicPort = g.Port()
		if glPublicPort == "" {
			glPublicPort = "443"
		}
		upstreams = append(upstreams, upstream{"bbb-greenlight", address(host("BBB_GREENLIGHT_HOST", "greenlight"), integer("BBB_GREENLIGHT_PORT", 3000, 1, 65535))})
	}
	values := object{"Configured": configured, "Resolver": resolver(), "Upstreams": upstreams, "HTTPPort": integer("BBB_HTTP_PORT", 8080, 1, 65535), "TLS": tls, "TLSPort": integer("BBB_TLS_PORT", 8443, 1, 65535), "Cert": nginxPath("BBB_TLS_CERT_FILE"), "Key": nginxPath("BBB_TLS_KEY_FILE"), "ServerName": u.Hostname(), "Scheme": u.Scheme, "Landing": "", "Greenlight": gl, "GreenlightPort": integer("GREENLIGHT_TLS_PORT", 8444, 1, 65535), "GreenlightPublicPort": glPublicPort}
	if env("BBB_LANDING_FILE", "") != "" {
		values["Landing"] = nginxPath("BBB_LANDING_FILE")
		values["LandingDir"] = strconv.Quote(filepath.Dir(env("BBB_LANDING_FILE", "")))
		values["LandingURI"] = strconv.Quote("/" + filepath.Base(env("BBB_LANDING_FILE", "")))
	}
	t, e := template.New("nginx").Parse(frontendTemplate)
	if e != nil {
		panic(e)
	}
	var b bytes.Buffer
	if e = t.Execute(&b, values); e != nil {
		panic(e)
	}
	return b.Bytes()
}
