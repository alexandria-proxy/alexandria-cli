package subscription

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/alexandria-proxy/alexandria-cli/internal/hwid"
)

const (
	maxbody    = 4 << 20
	devicename = "Alexandria"
)

var Version = "dev"

type Fetchopts struct {
	Useragent string
	Timeout   time.Duration
	Sendhwid  bool
}

func Defaultopts() Fetchopts {
	return Fetchopts{Timeout: 12 * time.Second, Sendhwid: true}
}

func Useragent() string {
	return "Alexandria/" + Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
}

func (o Fetchopts) apply(req *http.Request) {
	ua := strings.TrimSpace(o.Useragent)
	if ua == "" {
		ua = Useragent()
	}
	req.Header.Set("User-Agent", ua)
	if !o.Sendhwid {
		return
	}
	req.Header.Set("X-Device-Name", devicename)
	req.Header.Set("X-Device-Os", hwid.OS())
	req.Header.Set("X-Hwid", hwid.ID())
}

func (o Fetchopts) client() *http.Client {
	t := o.Timeout
	if t <= 0 {
		t = 12 * time.Second
	}
	return &http.Client{Timeout: t}
}

type Server struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	PingMs   int    `json:"ping_ms"`
	Raw      string `json:"raw"`
}

type Subscription struct {
	Name        string        `json:"name"`
	URL         string        `json:"url"`
	UpdatedAt   time.Time     `json:"updated_at"`
	AutoUpdate  time.Duration `json:"auto_update"`
	UsedBytes   int64         `json:"used_bytes"`
	TotalBytes  int64         `json:"total_bytes"`
	Expires     time.Time     `json:"expires"`
	Note        string        `json:"note"`
	SupportURL  string        `json:"support_url,omitempty"`
	AnnounceURL string        `json:"announce_url,omitempty"`
	Pinned      bool          `json:"pinned,omitempty"`
	Servers     []Server      `json:"servers"`
}

func sanitizeurl(raw string) string {
	cleaned := strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, raw)
	return strings.TrimSpace(cleaned)
}

func Fetch(ctx context.Context, raw string, o Fetchopts) (Subscription, error) {
	raw = sanitizeurl(raw)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, raw, nil)
	if err != nil {
		return Subscription{}, err
	}
	o.apply(req)

	resp, err := o.client().Do(req)
	if err != nil {
		return Subscription{}, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxbody))
	if err != nil {
		return Subscription{}, err
	}

	sub := Subscription{URL: raw, UpdatedAt: time.Now()}
	parseheaders(resp.Header, &sub)
	sub.Servers = parsebody(body)
	if sub.Name == "" {
		if u, err := url.Parse(raw); err == nil {
			sub.Name = u.Hostname()
		}
	}
	return sub, nil
}

func parseheaders(h http.Header, sub *Subscription) {
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
		sub.Name = unb64prefix(title)
	}
	if ann := h.Get("Announce"); ann != "" {
		sub.Note = unb64prefix(ann)
	}
	sub.SupportURL = strings.TrimSpace(h.Get("Support-Url"))
	sub.AnnounceURL = strings.TrimSpace(h.Get("Announce-Url"))
}

func unb64prefix(s string) string {
	if strings.HasPrefix(s, "base64:") {
		if dec, ok := decodeb64(strings.TrimPrefix(s, "base64:")); ok {
			return dec
		}
	}
	return s
}

func parsebody(b []byte) []Server {
	text := strings.TrimSpace(string(b))

	if strings.HasPrefix(text, "[") || strings.HasPrefix(text, "{") {
		if servers := parsejsonconfigs(text); len(servers) > 0 {
			return servers
		}
	}

	if dec, ok := decodeb64(text); ok && strings.Contains(dec, "://") {
		text = dec
	}

	var servers []Server
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if s, ok := parselink(line); ok {
			servers = append(servers, s)
		}
	}
	return servers
}

func parsejsonconfigs(text string) []Server {
	var arr []json.RawMessage
	if json.Unmarshal([]byte(text), &arr) == nil {
		var out []Server
		for _, c := range arr {
			if s, ok := parsexrayconfig(c); ok {
				out = append(out, s)
			}
		}
		return out
	}
	if s, ok := parsexrayconfig(json.RawMessage(text)); ok {
		return []Server{s}
	}
	return nil
}

func parsexrayconfig(raw json.RawMessage) (Server, bool) {
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
		if !isproxyproto(ob.Protocol) {
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
			Protocol: chainlabel(strings.ToUpper(ob.Protocol), ob.StreamSettings.Network, ob.StreamSettings.Security),
			Raw:      string(raw),
		}, true
	}
	return Server{}, false
}

type endpoint struct {
	Address string `json:"address"`
	Port    int    `json:"port"`
}

func isproxyproto(p string) bool {
	switch p {
	case "vless", "vmess", "trojan", "shadowsocks":
		return true
	}
	return false
}

func parselink(link string) (Server, bool) {
	switch {
	case strings.HasPrefix(link, "vmess://"):
		return parsevmess(link)
	case strings.HasPrefix(link, "vless://"), strings.HasPrefix(link, "trojan://"):
		return parseurllink(link)
	case strings.HasPrefix(link, "ss://"):
		return parsess(link)
	}
	return Server{}, false
}

func parseurllink(link string) (Server, bool) {
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
		Protocol: chainlabel(strings.ToUpper(u.Scheme), q.Get("type"), q.Get("security")),
		Raw:      link,
	}, true
}

func parsevmess(link string) (Server, bool) {
	dec, ok := decodeb64(strings.TrimPrefix(link, "vmess://"))
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
		Port:     anyint(v.Port),
		Protocol: chainlabel("VMESS", v.Net, v.TLS),
		Raw:      link,
	}, true
}

func parsess(link string) (Server, bool) {
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
		Protocol: "shadowsocks",
		Raw:      link,
	}, true
}

func chainlabel(proto, network, security string) string {
	parts := []string{strings.ToLower(proto)}
	if network != "" {
		parts = append(parts, netlabel(network))
	}
	if security != "" && security != "none" {
		parts = append(parts, strings.ToLower(security))
	}
	return strings.Join(parts, " / ")
}

func netlabel(network string) string {
	switch strings.ToLower(network) {
	case "splithttp":
		return "xhttp"
	case "h2", "h3":
		return "http"
	case "raw":
		return "tcp"
	}
	return strings.ToLower(network)
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

func anyint(v any) int {
	switch x := v.(type) {
	case float64:
		return int(x)
	case string:
		n, _ := strconv.Atoi(x)
		return n
	}
	return 0
}

func Sort(subs []Subscription) {
	sort.SliceStable(subs, func(i, j int) bool {
		return subs[i].Pinned && !subs[j].Pinned
	})
}

func Merge(prev, cur Subscription) Subscription {
	cur.Pinned = prev.Pinned
	pings := make(map[string]int, len(prev.Servers))
	for _, s := range prev.Servers {
		if s.PingMs != 0 {
			pings[s.Raw] = s.PingMs
		}
	}
	for i := range cur.Servers {
		if cur.Servers[i].PingMs == 0 {
			if p, ok := pings[cur.Servers[i].Raw]; ok {
				cur.Servers[i].PingMs = p
			}
		}
	}
	return cur
}

func Ping(ctx context.Context, sub *Subscription, proto string) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, 8)
	for i := range sub.Servers {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sub.Servers[i].PingMs = pingone(ctx, sub.Servers[i].Host, sub.Servers[i].Port, proto)
		}(i)
	}
	wg.Wait()
}

func pingone(ctx context.Context, host string, port int, proto string) int {
	if host == "" {
		return -1
	}
	if proto == "icmp" {
		return pingicmp(ctx, host)
	}
	if port == 0 {
		return -1
	}
	d := net.Dialer{Timeout: 3 * time.Second}
	start := time.Now()
	conn, err := d.DialContext(ctx, "tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return -1
	}
	conn.Close()
	return clampms(time.Since(start))
}

var icmptime = regexp.MustCompile(`(?i)[=<]\s*([0-9]+(?:[.,][0-9]+)?)\s*(?:ms|мс)`)

func icmpargs(host string) []string {
	switch runtime.GOOS {
	case "windows":
		return []string{"-n", "1", "-w", "3000", host}
	case "darwin":
		return []string{"-n", "-c", "1", "-t", "3", host}
	}
	return []string{"-n", "-c", "1", "-W", "3", host}
}

func pingicmp(ctx context.Context, host string) int {
	bin, err := exec.LookPath("ping")
	if err != nil {
		return -1
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	start := time.Now()
	out, err := exec.CommandContext(ctx, bin, icmpargs(host)...).CombinedOutput()
	if err != nil {
		return -1
	}
	elapsed := time.Since(start)
	if m := icmptime.FindSubmatch(out); len(m) == 2 {
		v, err := strconv.ParseFloat(strings.Replace(string(m[1]), ",", ".", 1), 64)
		if err == nil {
			return clampms(time.Duration(v * float64(time.Millisecond)))
		}
	}
	return clampms(elapsed)
}

func clampms(d time.Duration) int {
	ms := int(d.Milliseconds())
	if ms < 1 {
		ms = 1
	}
	return ms
}

func Sortservers(servers []Server, mode string) []Server {
	if mode != "ping" && mode != "alpha" {
		return servers
	}
	out := append([]Server(nil), servers...)
	if mode == "alpha" {
		sort.SliceStable(out, func(i, j int) bool {
			return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
		})
		return out
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i].PingMs, out[j].PingMs
		if a <= 0 {
			a = 1 << 30
		}
		if b <= 0 {
			b = 1 << 30
		}
		return a < b
	})
	return out
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria"), nil
}

func storepath() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "subscriptions.json"), nil
}

func Load() ([]Subscription, error) {
	p, err := storepath()
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
	p, err := storepath()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(subs, "", "  ")
	if err != nil {
		return err
	}
	return atomicwrite(p, data, 0600)
}

func atomicwrite(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".subs-*.tmp")
	if err != nil {
		return err
	}
	tmpname := tmp.Name()
	defer os.Remove(tmpname)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpname, path)
}
