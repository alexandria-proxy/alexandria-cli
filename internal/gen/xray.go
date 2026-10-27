package gen

import (
	"context"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/link"
)

const (
	socksport  = 10808
	metricslo  = 10810
	metricshi  = 10819
	metricsin  = "metrics_in"
	metricsout = "metrics_out"
	userlevel  = 8
	fragtag    = "fragment"
	listenaddr = "127.0.0.1"
)

type Opts struct {
	Metrics    int
	Socksport  int
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

func (o Opts) strategy() string {
	switch o.Preferip {
	case "ipv4":
		return "UseIPv4"
	case "ipv6":
		return "UseIPv6"
	}
	return "UseIP"
}

func sniffblock(on bool) map[string]any {
	return map[string]any{"enabled": on, "destOverride": []any{"http", "tls", "quic"}}
}

func (o Opts) inbounds() []any {
	socks := o.socks()
	in := []any{
		map[string]any{
			"tag":      "socks",
			"listen":   listenaddr,
			"port":     socks,
			"protocol": "socks",
			"settings": map[string]any{"udp": true, "auth": "noauth"},
			"sniffing": sniffblock(o.Sniffing),
		},
		map[string]any{
			"tag":      "http",
			"listen":   listenaddr,
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

func Xray(raw string, o Opts) (string, error) {
	if strings.HasPrefix(strings.TrimSpace(raw), "{") {
		return retune(raw, o)
	}
	return build(raw, o)
}

func build(raw string, o Opts) (string, error) {
	ob, err := outbound(raw)
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

func retune(raw string, o Opts) (string, error) {
	var cfg map[string]any
	if err := json.Unmarshal([]byte(raw), &cfg); err != nil {
		return raw, err
	}

	if lg, ok := cfg["log"].(map[string]any); ok {
		delete(lg, "access")
		delete(lg, "error")
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
			in["listen"] = listenaddr
			in["port"] = o.socks()
			in["sniffing"] = sniffblock(o.Sniffing)
		case "http":
			in["listen"] = listenaddr
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

func Metricsport(cfg string) int {
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

func outbound(raw string) (map[string]any, error) {
	l, err := link.Parse(raw)
	if err != nil {
		return nil, err
	}
	switch l.Scheme {
	case "vless":
		return vless(l), nil
	case "vmess":
		return vmess(l), nil
	case "trojan":
		return trojan(l), nil
	case "ss":
		return shadowsocks(l), nil
	}
	return nil, link.Errunsupported
}

func vless(l link.Link) map[string]any {
	user := map[string]any{
		"id":         l.User,
		"encryption": deflt(l.Query.Get("encryption"), "none"),
		"level":      userlevel,
	}
	if f := l.Query.Get("flow"); f != "" {
		user["flow"] = f
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "vless",
		"settings": map[string]any{
			"vnext": []any{map[string]any{
				"address": l.Host,
				"port":    l.Port,
				"users":   []any{user},
			}},
		},
		"streamSettings": streamsettings(l.Query, l.Host, "none"),
	}
}

func trojan(l link.Link) map[string]any {
	srv := map[string]any{
		"address":  l.Host,
		"port":     l.Port,
		"password": l.User,
	}
	if f := l.Query.Get("flow"); f != "" {
		srv["flow"] = f
	}
	return map[string]any{
		"tag":      "proxy",
		"protocol": "trojan",
		"settings": map[string]any{
			"servers": []any{srv},
		},
		"streamSettings": streamsettings(l.Query, l.Host, "tls"),
	}
}

func vmess(l link.Link) map[string]any {
	return map[string]any{
		"tag":      "proxy",
		"protocol": "vmess",
		"settings": map[string]any{
			"vnext": []any{map[string]any{
				"address": l.Host,
				"port":    l.Port,
				"users": []any{map[string]any{
					"id":       l.User,
					"alterId":  l.Alterid,
					"security": deflt(l.Scy, "auto"),
					"level":    userlevel,
				}},
			}},
		},
		"streamSettings": streamsettings(l.Query, l.Host, "none"),
	}
}

func shadowsocks(l link.Link) map[string]any {
	return map[string]any{
		"tag":      "proxy",
		"protocol": "shadowsocks",
		"settings": map[string]any{
			"servers": []any{map[string]any{
				"address":  l.Host,
				"port":     l.Port,
				"method":   l.Method,
				"password": l.Pass,
			}},
		},
	}
}

func streamsettings(q url.Values, host, defsec string) map[string]any {
	net := link.Normnet(deflt(q.Get("type"), "tcp"))
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
