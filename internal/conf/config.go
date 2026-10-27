package conf

import (
	"encoding/json"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
)

type Fragment struct {
	On       bool   `json:"on"`
	Packets  string `json:"packets"`
	Length   string `json:"length"`
	Interval string `json:"interval"`
}

type Mux struct {
	On          bool `json:"on"`
	Concurrency int  `json:"concurrency"`
}

type Subs struct {
	Auto       bool   `json:"auto"`
	Intervalh  int    `json:"intervalh"`
	Timeoutsec int    `json:"timeoutsec"`
	Sendhwid   bool   `json:"sendhwid"`
	Useragent  string `json:"useragent"`
}

type Advanced struct {
	Localdns   bool   `json:"localdns"`
	Jsondns    bool   `json:"jsondns"`
	Resolvesrv bool   `json:"resolvesrv"`
	Sniffing   bool   `json:"sniffing"`
	Sysproxy   bool   `json:"sysproxy"`
	Tunstack   string `json:"tunstack"`
	Tunconfig  string `json:"tunconfig"`
	Tundnson   bool   `json:"tundnson"`
	Tundns     string `json:"tundns"`

	Tuncustom string   `json:"tuncustom,omitempty"`
	Excluded  []string `json:"excluded,omitempty"`
}

type Logs struct {
	On      bool  `json:"on"`
	Daemon  bool  `json:"daemon"`
	Xray    bool  `json:"xray"`
	Singbox bool  `json:"singbox"`
	Max     int64 `json:"maxbytes"`
}

type Settings struct {
	Pingproto string   `json:"pingproto"`
	Preferip  string   `json:"preferip"`
	Socksport int      `json:"socksport"`
	Fragment  Fragment `json:"fragment"`
	Mux       Mux      `json:"mux"`
	Subs      Subs     `json:"subs"`
	Advanced  Advanced `json:"advanced"`
	Logs      Logs     `json:"logs"`
}

type Config struct {
	Mode      string   `json:"mode"`
	Autostart bool     `json:"autostart"`
	Lasturl   string   `json:"lasturl,omitempty"`
	Lastsrv   int      `json:"lastsrv,omitempty"`
	Settings  Settings `json:"settings"`
}

func Defaults() Config {
	return Config{
		Mode: "proxy",
		Settings: Settings{
			Pingproto: "tcp",
			Preferip:  "auto",
			Socksport: 10808,
			Fragment:  Fragment{Packets: "tlshello", Length: "10-20", Interval: "10-20"},
			Mux:       Mux{Concurrency: 8},
			Subs: Subs{
				Intervalh:  1,
				Timeoutsec: 9,
				Sendhwid:   true,
			},
			Logs: Logs{
				On:      true,
				Daemon:  true,
				Xray:    true,
				Singbox: true,
				Max:     200 << 20,
			},
			Advanced: Advanced{
				Sniffing:  true,
				Tunstack:  "mixed",
				Tunconfig: "default",
				Tundnson:  true,
				Tundns:    "1.1.1.1",
			},
		},
	}
}

const Direnv = "ALEXANDRIA_DIR"

func Dir() (string, error) {
	if d := os.Getenv(Direnv); d != "" {
		return filepath.Clean(d), nil
	}
	base, err := basedir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria"), nil
}

func basedir() (string, error) {
	home := sudohome()
	if home == "" || os.Getenv("XDG_CONFIG_HOME") != "" {
		return os.UserConfigDir()
	}
	if runtime.GOOS == "darwin" {
		return filepath.Join(home, "Library", "Application Support"), nil
	}
	return filepath.Join(home, ".config"), nil
}

func sudohome() string {
	if runtime.GOOS == "windows" || os.Geteuid() != 0 {
		return ""
	}
	name := os.Getenv("SUDO_USER")
	if name == "" || name == "root" {
		return ""
	}
	u, err := user.Lookup(name)
	if err != nil {
		return ""
	}
	return u.HomeDir
}

func Home() (string, error) {
	if h := sudohome(); h != "" {
		return h, nil
	}
	if uid, _, ok := Owner(); ok && os.Geteuid() == 0 {
		if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
			return u.HomeDir, nil
		}
	}
	return os.UserHomeDir()
}

func File(name string) (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, name), nil
}

func Mkdir() (string, error) {
	d, err := Dir()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
		return "", err
	}
	Own(d)
	return d, nil
}

func Path() (string, error) {
	return File("config.json")
}

func Load() (Config, error) {
	c := Defaults()
	p, err := Path()
	if err != nil {
		return c, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return c, nil
		}
		return c, err
	}
	if err := json.Unmarshal(data, &c); err != nil {
		return Defaults(), err
	}
	return Normalize(c), nil
}

func Normalize(c Config) Config {
	d := Defaults()
	if c.Mode != "tun" {
		c.Mode = "proxy"
	}
	s := &c.Settings
	if s.Pingproto != "icmp" {
		s.Pingproto = "tcp"
	}
	switch s.Preferip {
	case "ipv4", "ipv6":
	default:
		s.Preferip = "auto"
	}
	if s.Socksport < 1 || s.Socksport > 65534 {
		s.Socksport = d.Settings.Socksport
	}
	if s.Mux.Concurrency < 1 || s.Mux.Concurrency > 1024 {
		s.Mux.Concurrency = d.Settings.Mux.Concurrency
	}
	if s.Fragment.Packets == "" {
		s.Fragment.Packets = d.Settings.Fragment.Packets
	}
	if s.Fragment.Length == "" {
		s.Fragment.Length = d.Settings.Fragment.Length
	}
	if s.Fragment.Interval == "" {
		s.Fragment.Interval = d.Settings.Fragment.Interval
	}
	if s.Subs.Intervalh < 1 || s.Subs.Intervalh > 168 {
		s.Subs.Intervalh = d.Settings.Subs.Intervalh
	}
	if s.Subs.Timeoutsec < 1 || s.Subs.Timeoutsec > 60 {
		s.Subs.Timeoutsec = d.Settings.Subs.Timeoutsec
	}
	if s.Logs.Max < 0 {
		s.Logs.Max = 0
	}
	switch s.Advanced.Tunstack {
	case "system", "gvisor":
	default:
		s.Advanced.Tunstack = d.Settings.Advanced.Tunstack
	}
	if s.Advanced.Tunconfig != "custom" {
		s.Advanced.Tunconfig = "default"
	}
	if s.Advanced.Tundns == "" {
		s.Advanced.Tundns = d.Settings.Advanced.Tundns
	}
	return c
}

func Save(c Config) error {
	d, err := Mkdir()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	return Write(filepath.Join(d, "config.json"), data, 0600)
}

func Write(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+"-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)

	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		tmp.Close()
		return err
	}
	Own(name)
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
