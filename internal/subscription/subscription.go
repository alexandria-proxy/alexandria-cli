package subscription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

const (
	userAgent = "Alexandria"
	maxBody   = 4 << 20
)

type Server struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	PingMs   int    `json:"ping_ms"`
	Raw      string `json:"raw"`
}

type Subscription struct {
	Name       string        `json:"name"`
	URL        string        `json:"url"`
	UpdatedAt  time.Time     `json:"updated_at"`
	AutoUpdate time.Duration `json:"auto_update"`
	UsedBytes  int64         `json:"used_bytes"`
	TotalBytes int64         `json:"total_bytes"`
	Expires    time.Time     `json:"expires"`
	Note       string        `json:"note"`
	Servers    []Server      `json:"servers"`
}

var httpClient = &http.Client{Timeout: 12 * time.Second}

func Fetch(ctx context.Context, raw string) (Subscription, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return Subscription{}, err
	}
	req.Header.Set("User-Agent", userAgent)

	resp, err := httpClient.Do(req)
	if err != nil {
		return Subscription{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxBody))
	if err != nil {
		return Subscription{}, err
	}

	sub := Subscription{URL: raw, UpdatedAt: time.Now()}
	parseHeaders(resp.Header, &sub)
	sub.Servers = parseBody(body)
	if sub.Name == "" {
		if u, err := url.Parse(raw); err == nil {
			sub.Name = u.Hostname()
		}
	}
	return sub, nil
}

func parseHeaders(h http.Header, sub *Subscription) {
	if info := h.Get("Subscription-Userinfo"); info != "" {
		var up, down int64
		for _, part := range strings.Split(info, ";") {
			kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
			if len(kv) != 2 {
				continue
			}
			n, _ := strconv.ParseInt(kv[1], 10, 64)
			switch kv[0] {
			case "upload":
				up = n
			case "download":
				down = n
			case "total":
				sub.TotalBytes = n
			case "expire":
				if n > 0 {
					sub.Expires = time.Unix(n, 0)
				}
			}
		}
		sub.UsedBytes = up + down
	}
	if iv := h.Get("Profile-Update-Interval"); iv != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(iv)); err == nil {
			sub.AutoUpdate = time.Duration(n) * time.Hour
		}
	}
	if title := h.Get("Profile-Title"); title != "" {
		if strings.HasPrefix(title, "base64:") {
			if dec, ok := decodeB64(strings.TrimPrefix(title, "base64:")); ok {
				title = dec
			}
		}
		sub.Name = title
	}
}

func parseBody(b []byte) []Server {
	text := strings.TrimSpace(string(b))

	if strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{") {
		if servers := parseJSONConfigs(text); len(servers) > 0 {
			return servers
		}
	}

	if dec, ok := decodeB64(text); ok && strings.Contains(dec, "://") {
		text = dec
	}

	var servers []Server
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if s, ok := parseLink(line); ok {
			servers = append(servers, s)
		}
	}
	return servers
}

func parseJSONConfigs(text string) []Server {
	var arr []json.RawMessage
	if json.Unmarshal([]byte(text), &arr) == nil {
		var out []Server
		for _, c := range arr {
			if s, ok := parseXrayConfig(c); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if s, ok := parseXrayConfig(json.RawMessage(text)); ok {
		return []Server{s}
	}
	return nil
}

func parseXrayConfig(raw json.RawMessage) (Server, bool) {
	var cfg struct {
		Remarks   string `json:"remarks"`
		Outbounds []struct {
			Protocol       string `json:"protocol"`
			StreamSettings struct {
				Network  string `json:"network"`
				Security string `json:"security"`
			} `json:"streamSettings"`
			Settings struct {
				Vnext   []endpoint `json:"vnext"`
				Servers []endpoint `json:"servers"`
			} `json:"settings"`
		} `json:"outbounds"`
	}
	if json.Unmarshal(raw, &cfg) != nil {
		return Server{}, false
	}

	for _, ob := range cfg.Outbounds {
		if !isProxyProto(ob.Protocol) {
			continue
		}
		host, port := "", 0
		if len(ob.Settings.Vnext) > 0 {
			host, port = ob.Settings.Vnext[0].Address, ob.Settings.Vnext[0].Port
		} else if len(ob.Settings.Servers) > 0 {
			host, port = ob.Settings.Servers[0].Address, ob.Settings.Servers[0].Port
		}
		name := cfg.Remarks
		if name == "" {
			name = host
		}
		return Server{
			Name:     name,
			Host:     host,
			Port:     port,
			Protocol: chainLabel(strings.ToUpper(ob.Protocol), ob.StreamSettings.Network, ob.StreamSettings.Security),
			Raw:      string(raw),
		}, true
	}
	return Server{}, false
}

type endpoint struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func isProxyProto(p string) bool {
	switch p {
	case "vless", "vmess", "trojan", "shadowsocks":
		return true
	}
	return false
}

func parseLink(link string) (Server, bool) {
	switch {
	case strings.HasPrefix(link, "vmess://"):
		return parseVmess(link)
	case strings.HasPrefix(link, "vless://"), strings.HasPrefix(link, "trojan://"):
		return parseURLLink(link)
	case strings.HasPrefix(link, "ss://"):
		return parseSS(link)
	}
	return Server{}, false
}

func parseURLLink(link string) (Server, bool) {
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return Server{}, false
	}
	port, _ := strconv.Atoi(u.Port())
	q := u.Query()
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	return Server{
		Name:     name,
		Host:     u.Hostname(),
		Port:     port,
		Protocol: chainLabel(strings.ToUpper(u.Scheme), q.Get("type"), q.Get("security")),
		Raw:      link,
	}, true
}

func parseVmess(link string) (Server, bool) {
	dec, ok := decodeB64(strings.TrimPrefix(link, "vmess://"))
	if !ok {
		return Server{}, false
	}
	var v struct {
		Ps   string `json:"ps"`
		Add  string `json:"add"`
		Port any    `json:"port"`
		Net  string `json:"net"`
		TLS  string `json:"tls"`
	}
	if json.Unmarshal([]byte(dec), &v) != nil || v.Add == "" {
		return Server{}, false
	}
	name := v.Ps
	if name == "" {
		name = v.Add
	}
	return Server{
		Name:     name,
		Host:     v.Add,
		Port:     anyInt(v.Port),
		Protocol: chainLabel("VMESS", v.Net, v.TLS),
		Raw:      link,
	}, true
}

func parseSS(link string) (Server, bool) {
	u, err := url.Parse(link)
	if err != nil || u.Hostname() == "" {
		return Server{}, false
	}
	port, _ := strconv.Atoi(u.Port())
	name := u.Fragment
	if name == "" {
		name = u.Hostname()
	}
	return Server{
		Name:     name,
		Host:     u.Hostname(),
		Port:     port,
		Protocol: "SHADOWSOCKS",
		Raw:      link,
	}, true
}

func chainLabel(proto, network, security string) string {
	parts := []string{proto}
	if network != "" {
		parts = append(parts, strings.ToUpper(network))
	}
	if security != "" && security != "none" {
		parts = append(parts, strings.ToUpper(security))
	}
	return strings.Join(parts, " / ")
}

func decodeB64(s string) (string, bool) {
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

func anyInt(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria"), nil
}

func storePath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "subscriptions.json"), nil
}

func Load() ([]Subscription, error) {
	p, err := storePath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var subs []Subscription
	if err := json.Unmarshal(data, &subs); err != nil {
		return nil, err
	}
	return subs, nil
}

func SaveAll(subs []Subscription) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return err
	}
	p, err := storePath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(p, data, 0600)
}
