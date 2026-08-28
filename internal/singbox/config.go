package singbox

import (
	"encoding/json"
	"strings"
)

const (
	TunName    = "tun0"
	TableIndex = 2022
	RuleIndex  = 9000
)

type Opts struct {
	Socksport int
	Stack     string
	Dnson     bool
	Dns       string
	Preferip  string
	Excluded  []string
}

func SocksPort(xrayconfig string) int {
	var c struct {
		Inbounds []struct {
			Port     int    `json:"port"`
			Protocol string `json:"protocol"`
		} `json:"inbounds"`
	}
	if json.Unmarshal([]byte(xrayconfig), &c) == nil {
		for _, in := range c.Inbounds {
			if in.Protocol == "socks" && in.Port > 0 {
				return in.Port
			}
		}
		for _, in := range c.Inbounds {
			if in.Port > 0 {
				return in.Port
			}
		}
	}
	return 10808
}

func (o Opts) stack() string {
	switch o.Stack {
	case "system", "gvisor":
		return o.Stack
	}
	return "mixed"
}

func (o Opts) strategy() string {
	switch o.Preferip {
	case "ipv4":
		return "ipv4_only"
	case "ipv6":
		return "ipv6_only"
	}
	return "prefer_ipv4"
}

func (o Opts) dnsaddr() string {
	if s := strings.TrimSpace(o.Dns); s != "" {
		return s
	}
	return "1.1.1.1"
}

func cidrs(list []string) ([]any, []any) {
	var ip, domain []any
	for _, raw := range list {
		v := strings.TrimSpace(raw)
		if v == "" {
			continue
		}
		if strings.ContainsAny(v, "/:") || isnumeric(v) {
			ip = append(ip, v)
			continue
		}
		domain = append(domain, v)
	}
	return ip, domain
}

func isnumeric(s string) bool {
	for _, r := range s {
		if r != '.' && (r < '0' || r > '9') {
			return false
		}
	}
	return s != ""
}

func Config(o Opts) string {
	resolver := map[string]any{"server": "dns-proxy", "strategy": o.strategy()}

	proxy := map[string]any{
		"type":            "socks",
		"tag":             "proxy",
		"server":          "127.0.0.1",
		"server_port":     o.Socksport,
		"udp_fragment":    true,
		"domain_resolver": resolver,
	}
	direct := map[string]any{
		"type":            "direct",
		"tag":             "direct",
		"domain_resolver": resolver,
	}

	tun := map[string]any{
		"type":         "tun",
		"tag":          "tun-in",
		"address":      []any{"172.19.0.1/30"},
		"mtu":          1500,
		"auto_route":   true,
		"strict_route": true,
		"stack":        o.stack(),
	}

	rules := []any{
		map[string]any{"outbound": "direct", "process_name": []any{"xray", "sing-box", "xray.exe", "sing-box.exe"}},
	}
	ip, domain := cidrs(o.Excluded)
	if len(ip) > 0 {
		rules = append(rules, map[string]any{"outbound": "direct", "ip_cidr": ip})
	}
	if len(domain) > 0 {
		rules = append(rules, map[string]any{"outbound": "direct", "domain_suffix": domain})
	}
	rules = append(rules, map[string]any{"action": "sniff"})
	if o.Dnson {
		rules = append(rules, map[string]any{"action": "hijack-dns", "protocol": "dns"})
	}

	cfg := map[string]any{
		"log":       map[string]any{"level": "warn"},
		"inbounds":  []any{tun},
		"outbounds": []any{proxy, direct},
		"route": map[string]any{
			"auto_detect_interface": true,
			"final":                 "proxy",
			"rules":                 rules,
		},
	}
	if o.Dnson {
		cfg["dns"] = map[string]any{
			"servers": []any{map[string]any{
				"type":   "udp",
				"tag":    "dns-proxy",
				"server": o.dnsaddr(),
				"detour": "direct",
			}},
		}
	} else {
		cfg["dns"] = map[string]any{
			"servers": []any{map[string]any{
				"type":   "local",
				"tag":    "dns-proxy",
				"detour": "direct",
			}},
		}
	}

	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return ""
	}
	return string(b)
}
