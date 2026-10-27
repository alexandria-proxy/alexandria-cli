package link

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

var Errunsupported = errors.New("unsupported share-link protocol")

type Link struct {
	Scheme  string
	Name    string
	Host    string
	Port    int
	User    string
	Method  string
	Pass    string
	Alterid int
	Scy     string
	Query   url.Values
}

func Parse(raw string) (Link, error) {
	raw = strings.TrimSpace(raw)
	switch {
	case strings.HasPrefix(raw, "vless://"), strings.HasPrefix(raw, "trojan://"):
		return parseurl(raw)
	case strings.HasPrefix(raw, "vmess://"):
		return parsevmess(raw)
	case strings.HasPrefix(raw, "ss://"):
		return parsess(raw)
	}
	return Link{}, Errunsupported
}

func (l Link) Label() string {
	if l.Scheme == "ss" {
		return "shadowsocks"
	}
	return Label(l.Scheme, l.Query.Get("type"), l.Query.Get("security"))
}

func Label(proto, network, security string) string {
	parts := []string{strings.ToLower(proto)}
	if network != "" {
		parts = append(parts, Normnet(network))
	}
	if security != "" && security != "none" {
		parts = append(parts, strings.ToLower(security))
	}
	return strings.Join(parts, " / ")
}

func named(name, host string) string {
	if name == "" {
		return host
	}
	return name
}

func parseurl(raw string) (Link, error) {
	u, err := url.Parse(raw)
	if err != nil || u.Hostname() == "" {
		return Link{}, errors.New("bad " + strings.SplitN(raw, ":", 2)[0] + " link")
	}
	port, _ := strconv.Atoi(u.Port())
	return Link{
		Scheme: strings.ToLower(u.Scheme),
		Name:   named(u.Fragment, u.Hostname()),
		Host:   u.Hostname(),
		Port:   port,
		User:   u.User.Username(),
		Query:  u.Query(),
	}, nil
}

func parsevmess(raw string) (Link, error) {
	dec, ok := Decodeb64(strings.TrimPrefix(raw, "vmess://"))
	if !ok {
		return Link{}, errors.New("bad vmess link")
	}
	var v struct {
		Ps   string `json:"ps"`
		Add  string `json:"add"`
		Port any    `json:"port"`
		Id   string `json:"id"`
		Aid  any    `json:"aid"`
		Scy  string `json:"scy"`
		Net  string `json:"net"`
		Type string `json:"type"`
		Host string `json:"host"`
		Path string `json:"path"`
		Tls  string `json:"tls"`
		Sni  string `json:"sni"`
		Alpn string `json:"alpn"`
		Fp   string `json:"fp"`
	}
	if json.Unmarshal([]byte(dec), &v) != nil || v.Add == "" {
		return Link{}, errors.New("bad vmess json")
	}
	q := url.Values{}
	q.Set("type", v.Net)
	q.Set("security", v.Tls)
	q.Set("headerType", v.Type)
	q.Set("host", v.Host)
	q.Set("path", v.Path)
	q.Set("sni", v.Sni)
	q.Set("alpn", v.Alpn)
	q.Set("fp", v.Fp)
	return Link{
		Scheme:  "vmess",
		Name:    named(v.Ps, v.Add),
		Host:    v.Add,
		Port:    Anyint(v.Port),
		User:    v.Id,
		Alterid: Anyint(v.Aid),
		Scy:     v.Scy,
		Query:   q,
	}, nil
}

func parsess(raw string) (Link, error) {
	bad := errors.New("bad ss link")
	rest := strings.TrimPrefix(raw, "ss://")
	name := ""
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		name, _ = url.PathUnescape(rest[i+1:])
		rest = rest[:i]
	}
	var method, pass, hostport string
	if at := strings.LastIndexByte(rest, '@'); at >= 0 {
		userinfo := rest[:at]
		hostport = rest[at+1:]
		if dec, ok := Decodeb64(userinfo); ok && strings.Contains(dec, ":") {
			userinfo = dec
		}
		method, pass = splitpair(userinfo)
	} else {
		dec, ok := Decodeb64(rest)
		if !ok {
			return Link{}, bad
		}
		at := strings.LastIndexByte(dec, '@')
		if at < 0 {
			return Link{}, bad
		}
		method, pass = splitpair(dec[:at])
		hostport = dec[at+1:]
	}
	if i := strings.IndexByte(hostport, '?'); i >= 0 {
		hostport = hostport[:i]
	}
	host, port := splithostport(hostport)
	if host == "" {
		return Link{}, bad
	}
	return Link{Scheme: "ss", Name: named(name, host), Host: host, Port: port, Method: method, Pass: pass, Query: url.Values{}}, nil
}

func Normnet(n string) string {
	switch strings.ToLower(n) {
	case "h2", "h3":
		return "http"
	case "splithttp":
		return "xhttp"
	case "raw":
		return "tcp"
	}
	return strings.ToLower(n)
}

func splitpair(s string) (string, string) {
	if i := strings.IndexByte(s, ':'); i >= 0 {
		return s[:i], s[i+1:]
	}
	return s, ""
}

func splithostport(s string) (string, int) {
	host, portstr, err := net.SplitHostPort(s)
	if err != nil {
		return strings.Trim(s, "[]"), 0
	}
	port, _ := strconv.Atoi(portstr)
	return host, port
}

func Anyint(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case int:
		return x
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func Decodeb64(s string) (string, bool) {
	s = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == ' ' || r == '\t' {
			return -1
		}
		return r
	}, s)
	for _, enc := range []*base64.Encoding{
		base64.StdEncoding, base64.RawStdEncoding,
		base64.URLEncoding, base64.RawURLEncoding,
	} {
		if dec, err := enc.DecodeString(s); err == nil && utf8.Valid(dec) {
			return string(dec), true
		}
	}
	return "", false
}
