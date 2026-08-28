package xraygen

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	socksport  = 10808
	metricslo  = 10810
	metricshi  = 10819
	metricsin  = "metrics_in"
	metricsout = "metrics_out"
	userlevel  = 8
	fragtag    = "fragment"
)

var errunsupported = errors.New("unsupported share-link protocol")

type Opts struct {
	Metrics    int
	Socksport  int
	LAN        bool
	Sniffing   bool
	Preferip   string
	Frag       bool
	Fragpkt    string
	Fraglen    string
	Fragint    string
	Mux        bool
	Muxconc    int
	Localdns   bool
	Jsondns    bool
	Resolvesrv bool
}

func Defaults() Opts {
	return Opts{
		Metrics:   freeport(metricslo, metricshi),
		Socksport: socksport,
		Sniffing:  true,
		Preferip:  "auto",
		Muxconc:   8,
	}
}

func (o Opts) socks() int {
	if o.Socksport < 1 || o.Socksport > 65534 {
		return socksport
	}
	return o.Socksport
}

func (o Opts) listen() string {
	if o.LAN {
		return "0.0.0.0"
	}
	return "127.0.0.1"
}

func (o Opts) strategy() string {
	switch o.Preferip {
	case "ipv4":
		return "UseIPv4"
	case "ipv6":
		return "UseIPv6"
	}
	return "UseIP"
}

func Build(link string) (string, error) {
	return BuildOpts(link, Defaults())
}

func sniffblock(on bool) map[string]any {
	return map[string]any{"enabled": on, "destOverride": []any{"http", "tls", "quic"}}
}

func (o Opts) inbounds() []any {
	socks := o.socks()
	in := []any{
		map[string]any{
			"tag":      "socks",
			"listen":   o.listen(),
			"port":     socks,
			"protocol": "socks",
			"settings": map[string]any{"udp": true, "auth": "noauth"},
			"sniffing": sniffblock(o.Sniffing),
		},
		map[string]any{
			"tag":      "http",
			"listen":   o.listen(),
			"port":     socks + 1,
			"protocol": "http",
			"sniffing": sniffblock(o.Sniffing),
		},
	}
	return in
}

func (o Opts) fragoutbound() map[string]any {
	return map[string]any{
		"tag":      fragtag,
		"protocol": "freedom",
		"settings": map[string]any{
			"domainStrategy": "AsIs",
			"fragment": map[string]any{
				"packets":  deflt(o.Fragpkt, "tlshello"),
				"length":   deflt(o.Fraglen, "10-20"),
				"interval": deflt(o.Fragint, "10-20"),
			},
		},
		"streamSettings": map[string]any{
			"sockopt": map[string]any{"tcpKeepAliveIdle": 100, "tcpNoDelay": true},
		},
	}
}

func submap(m map[string]any, key string) map[string]any {
	v, ok := m[key].(map[string]any)
	if !ok {
		v = map[string]any{}
		m[key] = v
	}
	return v
}

func (o Opts) decorate(ob map[string]any) {
	if o.Mux {
		conc := o.Muxconc
		if conc < 1 || conc > 1024 {
			conc = 8
		}
		ob["mux"] = map[string]any{"enabled": true, "concurrency": conc, "xudpConcurrency": conc}
	} else {
		delete(ob, "mux")
	}

	stream := submap(ob, "streamSettings")
	sock := submap(stream, "sockopt")
	if o.Frag {
		sock["dialerProxy"] = fragtag
	} else {
		delete(sock, "dialerProxy")
	}
	if len(sock) == 0 {
		delete(stream, "sockopt")
	}
	if len(stream) == 0 {
		delete(ob, "streamSettings")
	}
}

func serverfield(ob map[string]any) (map[string]any, bool) {
	settings, ok := ob["settings"].(map[string]any)
	if !ok {
		return nil, false
	}
	for _, key := range []string{"vnext", "servers"} {
		list, ok := settings[key].([]any)
		if !ok || len(list) == 0 {
			continue
		}
		if first, ok := list[0].(map[string]any); ok {
			return first, true
		}
	}
	return nil, false
}

func (o Opts) resolve(ob map[string]any) {
	node, ok := serverfield(ob)
	if !ok {
		return
	}
	host, _ := node["address"].(string)
	if host == "" || net.ParseIP(host) != nil {
		return
	}
	network := "ip"
	switch o.Preferip {
	case "ipv4":
		network = "ip4"
	case "ipv6":
		network = "ip6"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()
	addrs, err := net.DefaultResolver.LookupIP(ctx, network, host)
	if err != nil || len(addrs) == 0 {
		return
	}
	node["address"] = addrs[0].String()

	stream := submap(ob, "streamSettings")
	security, _ := stream["security"].(string)
	switch security {
	case "tls", "reality":
		tls := submap(stream, security+"Settings")
		if name, _ := tls["serverName"].(string); name == "" {
			tls["serverName"] = host
		}
	}
}

func BuildOpts(link string, o Opts) (string, error) {
	ob, err := outbound(strings.TrimSpace(link))
	if err != nil {
		return "", err
	}
	o.decorate(ob)
	if o.Resolvesrv {
		o.resolve(ob)
	}

	outs := []any{ob}
	if o.Frag {
		outs = append(outs, o.fragoutbound())
	}
	outs = append(outs,
		map[string]any{
			"tag":      "direct",
			"protocol": "freedom",
			"settings": map[string]any{"domainStrategy": o.strategy()},
		},
		map[string]any{"tag": "block", "protocol": "blackhole"},
	)

	cfg := map[string]any{
		"log":       map[string]any{"loglevel": "warning"},
		"outbounds": outs,
	}
	if o.Localdns {
		cfg["dns"] = localdns()
	}

	inbounds := o.inbounds()
	rules := []any{}
	if o.Metrics > 0 {
		inbounds = append(inbounds, map[string]any{
			"tag":      metricsin,
			"listen":   "127.0.0.1",
			"port":     o.Metrics,
			"protocol": "dokodemo-door",
			"settings": map[string]any{"address": "127.0.0.1"},
		})
		rules = append(rules, map[string]any{
			"inboundTag":  []any{metricsin},
			"outboundTag": metricsout,
		})
		cfg["stats"] = map[string]any{}
		cfg["metrics"] = map[string]any{"tag": metricsout}
		cfg["policy"] = policy()
	}

	cfg["inbounds"] = inbounds
	cfg["routing"] = map[string]any{"domainStrategy": "IPIfNonMatch", "rules": rules}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func localdns() map[string]any {
	return map[string]any{
		"servers":       []any{"localhost"},
		"queryStrategy": "UseIP",
	}
}

func Retune(raw string, o Opts) (string, error) {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return raw, err
	}

	if o.Localdns {
		cfg["dns"] = localdns()
	} else if !o.Jsondns {
		delete(cfg, "dns")
	}

	outs, _ := cfg["outbounds"].([]any)
	filtered := make([]any, 0, len(outs)+1)
	for _, item := range outs {
		ob, ok := item.(map[string]any)
		if !ok {
			filtered = append(filtered, item)
			continue
		}
		if tag, _ := ob["tag"].(string); tag == fragtag {
			continue
		}
		filtered = append(filtered, ob)
	}
	if len(filtered) > 0 {
		if ob, ok := filtered[0].(map[string]any); ok {
			o.decorate(ob)
			if o.Resolvesrv {
				o.resolve(ob)
			}
		}
	}
	if o.Frag {
		filtered = append(filtered, o.fragoutbound())
	}
	cfg["outbounds"] = filtered

	ins, _ := cfg["inbounds"].([]any)
	kept := make([]any, 0, len(ins))
	for _, item := range ins {
		in, ok := item.(map[string]any)
		if !ok {
			kept = append(kept, item)
			continue
		}
		switch proto, _ := in["protocol"].(string); proto {
		case "socks":
			in["listen"] = o.listen()
			in["port"] = o.socks()
			in["sniffing"] = sniffblock(o.Sniffing)
		case "http":
			in["listen"] = o.listen()
			in["port"] = o.socks() + 1
			in["sniffing"] = sniffblock(o.Sniffing)
		}
		kept = append(kept, in)
	}
	cfg["inbounds"] = kept

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return raw, err
	}
	return string(b), nil
}

func MetricsPort(cfg string) int {
	var c struct {
		Inbounds []struct {
			Tag  string `json:"tag"`
			Port int    `json:"port"`
		} `json:"inbounds"`
	}
	if json.Unmarshal([]byte(cfg), &c) != nil {
		return 0
	}
	for _, in := range c.Inbounds {
		if in.Tag == metricsin && in.Port > 0 {
			return in.Port
		}
	}
	return 0
}

func policy() map[string]any {
	return map[string]any{
		"levels": map[string]any{
			"0": map[string]any{
				"statsUserUplink":   true,
				"statsUserDownlink": true,
			},
			strconv.Itoa(userlevel): map[string]any{
				"connIdle":     300,
				"downlinkOnly": 1,
				"handshake":    4,
				"uplinkOnly":   1,
			},
		},
		"system": map[string]any{
			"statsInboundUplink":    true,
			"statsInboundDownlink":  true,
			"statsOutboundUplink":   true,
			"statsOutboundDownlink": true,
		},
	}
}

func freeport(lo, hi int) int {
	for p := lo; p <= hi; p++ {
		l, err := net.Listen("tcp", "127.0.0.1:"+strconv.Itoa(p))
		if err != nil {
			continue
		}
		_ = l.Close()
		return p
	}
	return 0
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
		"level":      userlevel,
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
					"level":    userlevel,
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
