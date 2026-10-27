package sub

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/link"
)

const (
	maxbody    = 4 << 20
	devicename = "alexandria fuck you"
)

var Version = "dev"

type Fetchopts struct {
	Useragent string
	Timeout   time.Duration
	Sendhwid  bool
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
	req.Header.Set("X-Hwid", hwid())
	req.Header.Set("X-Device-Os", devos())
	req.Header.Set("X-Device-Model", devicename)
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
	Pingms   int    `json:"ping_ms"`
	Raw      string `json:"raw"`
}

type Subscription struct {
	Name        string        `json:"name"`
	Url         string        `json:"url"`
	Updatedat   time.Time     `json:"updated_at"`
	Autoupdate  time.Duration `json:"auto_update"`
	Usedbytes   int64         `json:"used_bytes"`
	Totalbytes  int64         `json:"total_bytes"`
	Expires     time.Time     `json:"expires"`
	Note        string        `json:"note"`
	Supporturl  string        `json:"support_url,omitempty"`
	Announceurl string        `json:"announce_url,omitempty"`
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
	if resp.StatusCode != http.StatusOK {
		return Subscription{}, errors.New("server answered " + resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxbody))
	if err != nil {
		return Subscription{}, err
	}

	sub := Subscription{Url: raw, Updatedat: time.Now()}
	parseheaders(resp.Header, &sub)
	sub.Servers = parsebody(body)
	if len(sub.Servers) == 0 {
		return Subscription{}, errors.New("no servers found in the response")
	}
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
				sub.Totalbytes = n
			case "expire":
				if n > 0 {
					sub.Expires = time.Unix(n, 0)
				}
			}
		}
		sub.Usedbytes = up + down
	}
	if iv := h.Get("Profile-Update-Interval"); iv != "" {
		if n, err := strconv.Atoi(strings.TrimSpace(iv)); err == nil {
			sub.Autoupdate = time.Duration(n) * time.Hour
		}
	}
	if title := h.Get("Profile-Title"); title != "" {
		sub.Name = unb64prefix(title)
	}
	if ann := h.Get("Announce"); ann != "" {
		sub.Note = unb64prefix(ann)
	}
	sub.Supporturl = strings.TrimSpace(h.Get("Support-Url"))
	sub.Announceurl = strings.TrimSpace(h.Get("Announce-Url"))
}

func unb64prefix(s string) string {
	if strings.HasPrefix(s, "base64:") {
		if dec, ok := link.Decodeb64(strings.TrimPrefix(s, "base64:")); ok {
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

	if dec, ok := link.Decodeb64(text); ok && strings.Contains(dec, "://") {
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
			Streamsettings struct {
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
			Protocol: link.Label(ob.Protocol, ob.Streamsettings.Network, ob.Streamsettings.Security),
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

func parselink(raw string) (Server, bool) {
	l, err := link.Parse(raw)
	if err != nil {
		return Server{}, false
	}
	return Server{Name: l.Name, Host: l.Host, Port: l.Port, Protocol: l.Label(), Raw: raw}, true
}

func Merge(prev, cur Subscription) Subscription {
	pings := make(map[string]int, len(prev.Servers))
	for _, s := range prev.Servers {
		if s.Pingms != 0 {
			pings[s.Raw] = s.Pingms
		}
	}
	for i := range cur.Servers {
		if cur.Servers[i].Pingms == 0 {
			if p, ok := pings[cur.Servers[i].Raw]; ok {
				cur.Servers[i].Pingms = p
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
			sub.Servers[i].Pingms = pingone(ctx, sub.Servers[i].Host, sub.Servers[i].Port, proto)
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

func storepath() (string, error) {
	return conf.File("subscriptions.json")
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

func Saveall(subs []Subscription) error {
	if _, err := conf.Mkdir(); err != nil {
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
	return conf.Write(p, data, 0600)
}
