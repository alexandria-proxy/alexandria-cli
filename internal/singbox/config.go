package singbox

import (
	"encoding/json"
	"fmt"
	"runtime"
)

const TunName = "alexat"

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

func Config(socksport int) string {
	base := fmt.Sprintf(`{
  "log": { "level": "warn" },
  "dns": {
    "servers": [
      { "type": "udp", "tag": "dns-proxy", "server": "8.8.8.8", "detour": "direct" }
    ]
  },
  "inbounds": [
    {
      "type": "tun",
      "tag": "tun-in",
      "address": ["172.19.0.1/30"],
      "mtu": 1500,
      "auto_route": true,
      "stack": "mixed"
    }
  ],
  "outbounds": [
    {
      "type": "socks",
      "tag": "proxy",
      "server": "127.0.0.1",
      "server_port": %d,
      "udp_fragment": true,
      "domain_resolver": { "server": "dns-proxy", "strategy": "prefer_ipv4" }
    },
    {
      "type": "direct",
      "tag": "direct",
      "domain_resolver": { "server": "dns-proxy", "strategy": "prefer_ipv4" }
    }
  ],
  "route": {
    "auto_detect_interface": true,
    "final": "proxy",
    "rules": [
      { "outbound": "direct", "process_name": ["xray", "sing-box", "xray.exe", "sing-box.exe"] },
      { "action": "sniff" },
      { "action": "hijack-dns", "protocol": "dns" }
    ]
  }
}`, socksport)
	return forcetunname(base)
}

func forcetunname(cfg string) string {
	if runtime.GOOS != "linux" {
		return cfg
	}
	var m map[string]any
	if json.Unmarshal([]byte(cfg), &m) != nil {
		return cfg
	}
	ins, _ := m["inbounds"].([]any)
	for _, it := range ins {
		in, _ := it.(map[string]any)
		if in != nil && in["type"] == "tun" {
			in["interface_name"] = TunName
		}
	}
	out, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return cfg
	}
	return string(out)
}
