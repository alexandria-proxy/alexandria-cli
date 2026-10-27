package route

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"os"
	"strings"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
)

const (
	Name   = "route"
	Custom = "tun.custom.json"
)

const Header = "# alexandria TUN bypass list: one CIDR, IP or domain per line\n" +
	"# traffic to these goes direct, not through the tunnel. '#' starts a comment\n"

var Errinvalid = errors.New("not a CIDR, IP or domain")

type Bad struct {
	Line int
	Text string
}

func Path() (string, error) { return conf.File(Name) }

func Custompath() (string, error) { return conf.File(Custom) }

func Valid(v string) bool {
	if _, _, err := net.ParseCIDR(v); err == nil {
		return true
	}
	if net.ParseIP(v) != nil {
		return true
	}
	return strings.Contains(v, ".") && !strings.ContainsAny(v, " \t/\\#")
}

func clean(line string) string {
	if i := strings.IndexByte(line, '#'); i >= 0 {
		line = line[:i]
	}
	return strings.TrimSpace(line)
}

func Parse(data []byte) ([]string, []Bad) {
	var entries []string
	var bad []Bad
	seen := map[string]bool{}
	sc := bufio.NewScanner(bytes.NewReader(data))
	for n := 1; sc.Scan(); n++ {
		v := clean(sc.Text())
		switch {
		case v == "":
		case !Valid(v):
			bad = append(bad, Bad{Line: n, Text: v})
		case !seen[strings.ToLower(v)]:
			seen[strings.ToLower(v)] = true
			entries = append(entries, v)
		}
	}
	return entries, bad
}

func read() ([]byte, error) {
	p, err := Path()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if os.IsNotExist(err) {
		return nil, nil
	}
	return data, err
}

func write(data []byte) error {
	if _, err := conf.Mkdir(); err != nil {
		return err
	}
	p, err := Path()
	if err != nil {
		return err
	}
	return conf.Write(p, data, 0600)
}

func Load() ([]string, []Bad, error) {
	data, err := read()
	if err != nil {
		return nil, nil, err
	}
	entries, bad := Parse(data)
	return entries, bad, nil
}

func Add(v string) (bool, error) {
	v = strings.TrimSpace(v)
	if !Valid(v) {
		return false, Errinvalid
	}
	data, err := read()
	if err != nil {
		return false, err
	}
	entries, _ := Parse(data)
	for _, e := range entries {
		if strings.EqualFold(e, v) {
			return false, nil
		}
	}
	if len(data) == 0 {
		data = []byte(Header)
	}
	if !bytes.HasSuffix(data, []byte("\n")) {
		data = append(data, '\n')
	}
	return true, write(append(data, v+"\n"...))
}

func Remove(v string) (bool, error) {
	v = strings.TrimSpace(v)
	data, err := read()
	if err != nil {
		return false, err
	}
	var out []string
	found := false
	for _, line := range strings.SplitAfter(string(data), "\n") {
		if line != "" && strings.EqualFold(clean(line), v) {
			found = true
			continue
		}
		out = append(out, line)
	}
	if !found {
		return false, nil
	}
	return true, write([]byte(strings.Join(out, "")))
}

func Migrate() error {
	cfg, err := conf.Load()
	if err != nil {
		return err
	}
	adv := &cfg.Settings.Advanced
	if len(adv.Excluded) == 0 && strings.TrimSpace(adv.Tuncustom) == "" {
		return nil
	}
	if len(adv.Excluded) > 0 {
		if data, err := read(); err == nil && len(data) == 0 {
			if err := write([]byte(Header + strings.Join(adv.Excluded, "\n") + "\n")); err != nil {
				return err
			}
		}
		adv.Excluded = nil
	}
	if strings.TrimSpace(adv.Tuncustom) != "" {
		p, err := Custompath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if _, err := conf.Mkdir(); err != nil {
				return err
			}
			if err := conf.Write(p, []byte(adv.Tuncustom), 0600); err != nil {
				return err
			}
		}
		adv.Tuncustom = ""
	}
	return conf.Save(cfg)
}
