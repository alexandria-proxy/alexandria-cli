package config

import (
	"encoding/json"
	"os"
	"path/filepath"
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
	Auto        bool   `json:"auto"`
	IntervalH   int    `json:"intervalh"`
	TimeoutSec  int    `json:"timeoutsec"`
	UpdateOpen  bool   `json:"updateopen"`
	PingOpen    bool   `json:"pingopen"`
	ConnectOpen bool   `json:"connectopen"`
	NoDupes     bool   `json:"nodupes"`
	SendHWID    bool   `json:"sendhwid"`
	UserAgent   string `json:"useragent"`
	SortBy      string `json:"sortby"`
}

type Advanced struct {
	LocalDNS    bool     `json:"localdns"`
	JSONDNS     bool     `json:"jsondns"`
	ResolveSrv  bool     `json:"resolvesrv"`
	Sniffing    bool     `json:"sniffing"`
	SysProxy    bool     `json:"sysproxy"`
	TUN         bool     `json:"tun"`
	TunProvider string   `json:"tunprovider"`
	TunStack    string   `json:"tunstack"`
	TunConfig   string   `json:"tunconfig"`
	TunCustom   string   `json:"tuncustom,omitempty"`
	TunDNSOn    bool     `json:"tundnson"`
	TunDNS      string   `json:"tundns"`
	Excluded    []string `json:"excluded,omitempty"`
}

type Logs struct {
	On      bool  `json:"on"`
	Daemon  bool  `json:"daemon"`
	Xray    bool  `json:"xray"`
	Singbox bool  `json:"singbox"`
	Max     int64 `json:"maxbytes"`
}

type Settings struct {
	PingProto string   `json:"pingproto"`
	PreferIP  string   `json:"preferip"`
	SocksPort int      `json:"socksport"`
	LAN       bool     `json:"lan"`
	Fragment  Fragment `json:"fragment"`
	Mux       Mux      `json:"mux"`
	Subs      Subs     `json:"subs"`
	Advanced  Advanced `json:"advanced"`
	Logs      Logs     `json:"logs"`
}

type Config struct {
	Lang     string   `json:"lang"`
	Mode     string   `json:"mode"`
	LastURL  string   `json:"lasturl,omitempty"`
	LastSrv  int      `json:"lastsrv,omitempty"`
	Settings Settings `json:"settings"`
}

func Defaults() Config {
	return Config{
		Settings: Settings{
			PingProto: "tcp",
			PreferIP:  "auto",
			SocksPort: 10808,
			Fragment:  Fragment{Packets: "tlshello", Length: "10-20", Interval: "10-20"},
			Mux:       Mux{Concurrency: 8},
			Subs: Subs{
				IntervalH:  1,
				TimeoutSec: 9,
				NoDupes:    true,
				SendHWID:   true,
				SortBy:     "none",
			},
			Logs: Logs{
				On:      true,
				Daemon:  true,
				Xray:    true,
				Singbox: true,
				Max:     200 << 20,
			},
			Advanced: Advanced{
				Sniffing:    true,
				TUN:         true,
				TunProvider: "singbox",
				TunStack:    "mixed",
				TunConfig:   "default",
				TunDNSOn:    true,
				TunDNS:      "1.1.1.1",
			},
		},
	}
}

func dir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "alexandria"), nil
}

func Path() (string, error) {
	d, err := dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(d, "config.json"), nil
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
	s := &c.Settings
	if s.PingProto != "icmp" {
		s.PingProto = "tcp"
	}
	switch s.PreferIP {
	case "ipv4", "ipv6":
	default:
		s.PreferIP = "auto"
	}
	if s.SocksPort < 1 || s.SocksPort > 65534 {
		s.SocksPort = d.Settings.SocksPort
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
	if s.Subs.IntervalH < 1 || s.Subs.IntervalH > 168 {
		s.Subs.IntervalH = d.Settings.Subs.IntervalH
	}
	if s.Subs.TimeoutSec < 1 || s.Subs.TimeoutSec > 60 {
		s.Subs.TimeoutSec = d.Settings.Subs.TimeoutSec
	}
	switch s.Subs.SortBy {
	case "ping", "alpha":
	default:
		s.Subs.SortBy = "none"
	}
	if s.Logs.Max < 0 {
		s.Logs.Max = 0
	}
	s.Advanced.TunProvider = "singbox"
	switch s.Advanced.TunStack {
	case "system", "gvisor":
	default:
		s.Advanced.TunStack = d.Settings.Advanced.TunStack
	}
	if s.Advanced.TunConfig != "custom" {
		s.Advanced.TunConfig = "default"
	}
	if s.Advanced.TunDNS == "" {
		s.Advanced.TunDNS = d.Settings.Advanced.TunDNS
	}
	return c
}

func Save(c Config) error {
	d, err := dir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(d, 0700); err != nil {
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
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
