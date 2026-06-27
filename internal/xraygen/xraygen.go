package xraygen

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

const (
	socksport = 10808
	httpport  = 10809
)

var errunsupported = errors.New("unsupported share-link protocol")

func Build(link string) (string, error) {
	ob, err := outbound(strings.TrimSpace(link))
	if err != nil {
		return "", err
	}
	cfg := map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{
			map[string]any{
				"tag":      "socks",
				"listen":   "127.0.0.1",
				"port":     socksport,
				"protocol": "socks",
				"settings": map[string]any{"udp": true, "auth": "noauth"},
				"sniffing": map[string]any{"enabled": true, "destOverride": []any{"http", "tls", "quic"}},
			},
			map[string]any{
				"tag":      "http",
				"listen":   "127.0.0.1",
				"port":     httpport,
				"protocol": "http",
			},
		},
		"outbounds": []any{
			ob,
			map[string]any{"tag": "direct", "protocol": "freedom"},
			map[string]any{"tag": "block", "protocol": "blackhole"},
		},
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func Supported(link string) bool {
	link = strings.TrimSpace(link)
	for _, p := range []string{"vless://", "vmess://", "trojan://", "ss://"} {
		if strings.HasPrefix(link, p) {
			return true
		}
	}
	return false
}

func outbound(link string) (map[string]any, error) {
	switch {
	case strings.HasPrefix(link, "vless://"):
		return vless(link)
	case strings.HasPrefix(link, "vmess://"):
		return vmess(link)
	case strings.HasPrefix(link, "trojan://"):
		return trojan(link)
	case strings.HasPrefix(link, "ss://"):
		return shadowsocks(link)
	}
	return nil, errunsupported
}

func vless(link string) (map[string]any, error) {
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("bad vless link")
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	user := map[string]any{
		"id":         u.User.Username(),
		"encryption": deflt(q.Get("encryption"), "none"),
	}
	if f := q.Get("flow"); f != "" {
		user["flow"] = f
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []any{map[string]any{
				"address": u.Hostname(),
				"port":    port,
				"users":   []any{user},
			}},
		},
		"streamSettings": streamsettings(q, u.Hostname(), "none"),
	}, nil
}

func trojan(link string) (map[string]any, error) {
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return nil, errors.New("bad trojan link")
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	srv := map[string]any{
		"address":  u.Hostname(),
		"port":     port,
		"password": u.User.Username(),
	}
	if f := q.Get("flow"); f != "" {
		srv["flow"] = f
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "trojan",
		"settings": map[string]any{
			"servers": []any{srv},
		},
		"streamSettings": streamsettings(q, u.Hostname(), "tls"),
	}, nil
}

func vmess(link string) (map[string]any, error) {
	dec, ok := decodeb64(strings.TrimPrefix(link, "vmess://"))
	if !ok {
		return nil, errors.New("bad vmess link")
	}
	var v struct {
		Add  string `json:"add"`
		Port any    `json:"port"`
		ID   string `json:"id"`
		Aid  any    `json:"aid"`
		Scy  string `json:"scy"`
		Net  string `json:"net"`
		Type string `json:"type"`
		Host string `json:"host"`
		Path string `json:"path"`
		TLS  string `json:"tls"`
		SNI  string `json:"sni"`
		ALPN string `json:"alpn"`
		FP   string `json:"fp"`
	}
	if json.Unmarshal([]byte(dec), &v) != nil || v.Add == "" {
		return nil, errors.New("bad vmess json")
	}

	q := url.Values{}
	q.Set("type", v.Net)
	q.Set("security", v.TLS)
	q.Set("headerType", v.Type)
	q.Set("host", v.Host)
	q.Set("path", v.Path)
	q.Set("sni", v.SNI)
	q.Set("alpn", v.ALPN)
	q.Set("fp", v.FP)

	return map[string]any{
		"tag":      "proxy",
		"protocol": "vmess",
		"settings": map[string]any{
			"vnext": []any{map[string]any{
				"address": v.Add,
				"port":    anyint(v.Port),
				"users": []any{map[string]any{
					"id":       v.ID,
					"alterId":  anyint(v.Aid),
					"security": deflt(v.Scy, "auto"),
				}},
			}},
		},
		"streamSettings": streamsettings(q, v.Add, "none"),
	}, nil
}

func shadowsocks(link string) (map[string]any, error) {
	rest := strings.TrimPrefix(link, "ss://")
	if i := strings.IndexByte(rest, '#'); i >= 0 {
		rest = rest[:i]
	}

	var method, password, hostport string
	if at := strings.LastIndexByte(rest, '@'); at >= 0 {
		userinfo := rest[:at]
		hostport = rest[at+1:]
		if dec, ok := decodeb64(userinfo); ok && strings.Contains(dec, ":") {
			userinfo = dec
		}
		method, password = splitpair(userinfo)
	} else {
		dec, ok := decodeb64(rest)
		if !ok {
			return nil, errors.New("bad ss link")
		}
		at := strings.LastIndexByte(dec, '@')
		if at < 0 {
			return nil, errors.New("bad ss link")
		}
		method, password = splitpair(dec[:at])
		hostport = dec[at+1:]
	}
	if i := strings.IndexByte(hostport, '?'); i >= 0 {
		hostport = hostport[:i]
	}
	host, port := splithostport(hostport)
	if host == "" {
		return nil, errors.New("bad ss link")
	}

	return map[string]any{
		"tag":      "proxy",
		"protocol": "shadowsocks",
		"settings": map[string]any{
			"servers": []any{map[string]any{
				"address":  host,
				"port":     port,
				"method":   method,
				"password": password,
			}},
		},
	}, nil
}

func streamsettings(q url.Values, host, defsec string) map[string]any {
	net := normnet(deflt(q.Get("type"), "tcp"))
	sec := deflt(q.Get("security"), defsec)
	if sec == "" {
		sec = "none"
	}
	ss := map[string]any{"network": net, "security": sec}

	switch net {
	case "ws":
		ss["wsSettings"] = pathhost(q)
	case "httpupgrade":
		ss["httpupgradeSettings"] = pathhost(q)
	case "xhttp":
		x := pathhost(q)
		if m := q.Get("mode"); m != "" {
			x["mode"] = m
		}
		ss["xhttpSettings"] = x
	case "grpc":
		g := map[string]any{"serviceName": deflt(q.Get("serviceName"), q.Get("path"))}
		if strings.Contains(q.Get("mode"), "multi") {
			g["multiMode"] = true
		}
		ss["grpcSettings"] = g
	case "http":
		h := map[string]any{}
		if p := q.Get("path"); p != "" {
			h["path"] = p
		}
		if hv := q.Get("host"); hv != "" {
			h["host"] = splitcsv(hv)
		}
		ss["httpSettings"] = h
	case "kcp":
		k := map[string]any{"header": map[string]any{"type": deflt(q.Get("headerType"), "none")}}
		if s := q.Get("seed"); s != "" {
			k["seed"] = s
		}
		ss["kcpSettings"] = k
	case "quic":
		ss["quicSettings"] = map[string]any{
			"security": deflt(q.Get("quicSecurity"), "none"),
			"key":      q.Get("key"),
			"header":   map[string]any{"type": deflt(q.Get("headerType"), "none")},
		}
	case "tcp":
		if q.Get("headerType") == "http" {
			req := map[string]any{}
			if hv := q.Get("host"); hv != "" {
				req["headers"] = map[string]any{"Host": splitcsv(hv)}
			}
			if p := q.Get("path"); p != "" {
				req["path"] = splitcsv(p)
			}
			ss["tcpSettings"] = map[string]any{"header": map[string]any{"type": "http", "request": req}}
		}
	}

	switch sec {
	case "tls":
		t := map[string]any{"serverName": firstnonempty(q.Get("sni"), q.Get("host"), host)}
		if truthy(q.Get("allowInsecure")) {
			t["allowInsecure"] = true
		}
		if a := q.Get("alpn"); a != "" {
			t["alpn"] = splitcsv(a)
		}
		if fp := q.Get("fp"); fp != "" {
			t["fingerprint"] = fp
		}
		ss["tlsSettings"] = t
	case "reality":
		r := map[string]any{
			"serverName":  q.Get("sni"),
			"fingerprint": deflt(q.Get("fp"), "chrome"),
			"publicKey":   q.Get("pbk"),
			"shortId":     q.Get("sid"),
		}
		if spx := q.Get("spx"); spx != "" {
			r["spiderX"] = spx
		}
		ss["realitySettings"] = r
	}
	return ss
}

func pathhost(q url.Values) map[string]any {
	m := map[string]any{}
	if p := q.Get("path"); p != "" {
		m["path"] = p
	}
	if h := q.Get("host"); h != "" {
		m["host"] = h
	}
	return m
}

func normnet(n string) string {
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

func deflt(v, d string) string {
	if v == "" {
		return d
	}
	return v
}

func firstnonempty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

func truthy(v string) bool {
	return v == "1" || strings.EqualFold(v, "true")
}

func splitcsv(s string) []any {
	parts := strings.Split(s, ",")
	out := make([]any, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
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

func anyint(v any) int {
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

func decodeb64(s string) (string, bool) {
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
