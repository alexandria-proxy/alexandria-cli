package cli

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/route"
)

type confkey struct {
	name    string
	help    string
	choices []string
	conn    bool
	get     func(c *conf.Config) string
	set     func(c *conf.Config, v string) error
}

func onoff(b bool) string {
	if b {
		return "on"
	}
	return "off"
}

func parsebool(v string) (bool, error) {
	switch strings.ToLower(v) {
	case "on", "true", "yes", "1":
		return true, nil
	case "off", "false", "no", "0":
		return false, nil
	}
	return false, errors.New("expected on or off")
}

func boolkey(name, help string, conn bool, ptr func(*conf.Config) *bool) confkey {
	return confkey{
		name: name, help: help, conn: conn, choices: []string{"on", "off"},
		get: func(c *conf.Config) string { return onoff(*ptr(c)) },
		set: func(c *conf.Config, v string) error {
			b, err := parsebool(v)
			if err == nil {
				*ptr(c) = b
			}
			return err
		},
	}
}

func choicekey(name string, conn bool, choices []string, ptr func(*conf.Config) *string) confkey {
	return confkey{
		name: name, help: strings.Join(choices, "|"), conn: conn, choices: choices,
		get: func(c *conf.Config) string { return *ptr(c) },
		set: func(c *conf.Config, v string) error {
			v = strings.ToLower(v)
			for _, ch := range choices {
				if v == ch {
					*ptr(c) = v
					return nil
				}
			}
			return errors.New("expected " + strings.Join(choices, ", "))
		},
	}
}

func intkey(name string, conn bool, lo, hi int, unit string, ptr func(*conf.Config) *int) confkey {
	return confkey{
		name: name, help: fmt.Sprintf("%d-%d%s", lo, hi, unit), conn: conn,
		get: func(c *conf.Config) string { return strconv.Itoa(*ptr(c)) + unit },
		set: func(c *conf.Config, v string) error {
			n, err := strconv.Atoi(strings.TrimSuffix(strings.ToLower(v), unit))
			if err != nil || n < lo || n > hi {
				return fmt.Errorf("expected %d-%d", lo, hi)
			}
			*ptr(c) = n
			return nil
		},
	}
}

var rangeword = regexp.MustCompile(`^\d+(-\d+)?$`)

func rangekey(name string, ptr func(*conf.Config) *string) confkey {
	return confkey{
		name: name, help: "N or N-M", conn: true,
		get: func(c *conf.Config) string { return *ptr(c) },
		set: func(c *conf.Config, v string) error {
			if !rangeword.MatchString(v) {
				return errors.New("expected N or N-M")
			}
			*ptr(c) = v
			return nil
		},
	}
}

var confkeys = []confkey{
	choicekey("mode", false, []string{"proxy", "tun"}, func(c *conf.Config) *string { return &c.Mode }),
	intkey("socksport", true, 1, 65534, "", func(c *conf.Config) *int { return &c.Settings.Socksport }),
	choicekey("preferip", true, []string{"auto", "ipv4", "ipv6"}, func(c *conf.Config) *string { return &c.Settings.Preferip }),
	choicekey("pingproto", false, []string{"tcp", "icmp"}, func(c *conf.Config) *string { return &c.Settings.Pingproto }),
	boolkey("frag", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Fragment.On }),
	choicekey("frag.packets", true, []string{"tlshello", "1-1", "1-3"}, func(c *conf.Config) *string { return &c.Settings.Fragment.Packets }),
	rangekey("frag.length", func(c *conf.Config) *string { return &c.Settings.Fragment.Length }),
	rangekey("frag.interval", func(c *conf.Config) *string { return &c.Settings.Fragment.Interval }),
	boolkey("mux", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Mux.On }),
	intkey("mux.conc", true, 1, 1024, "", func(c *conf.Config) *int { return &c.Settings.Mux.Concurrency }),
	boolkey("sniff", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Advanced.Sniffing }),
	boolkey("sysproxy", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Advanced.Sysproxy }),
	boolkey("dns.local", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Advanced.Localdns }),
	boolkey("dns.json", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Advanced.Jsondns }),
	boolkey("dns.resolve", "on|off", true, func(c *conf.Config) *bool { return &c.Settings.Advanced.Resolvesrv }),
	boolkey("subs.auto", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Subs.Auto }),
	intkey("subs.interval", false, 1, 168, "h", func(c *conf.Config) *int { return &c.Settings.Subs.Intervalh }),
	intkey("subs.timeout", false, 1, 60, "s", func(c *conf.Config) *int { return &c.Settings.Subs.Timeoutsec }),
	boolkey("subs.hwid", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Subs.Sendhwid }),
	{
		name: "subs.ua", help: "TEXT, empty for the default",
		get: func(c *conf.Config) string { return c.Settings.Subs.Useragent },
		set: func(c *conf.Config, v string) error { c.Settings.Subs.Useragent = strings.TrimSpace(v); return nil },
	},
	choicekey("tun.stack", true, []string{"mixed", "system", "gvisor"}, func(c *conf.Config) *string { return &c.Settings.Advanced.Tunstack }),
	{
		name: "tun.dns", help: "off|IP", conn: true, choices: []string{"off", "1.1.1.1", "8.8.8.8", "9.9.9.9"},
		get: func(c *conf.Config) string {
			if !c.Settings.Advanced.Tundnson {
				return "off"
			}
			return c.Settings.Advanced.Tundns
		},
		set: func(c *conf.Config, v string) error {
			if b, err := parsebool(v); err == nil && !b {
				c.Settings.Advanced.Tundnson = false
				return nil
			}
			if net.ParseIP(v) == nil {
				return errors.New("expected off or an IP address")
			}
			c.Settings.Advanced.Tundnson, c.Settings.Advanced.Tundns = true, v
			return nil
		},
	},
	choicekey("tun.config", true, []string{"default", "custom"}, func(c *conf.Config) *string { return &c.Settings.Advanced.Tunconfig }),
	boolkey("logs", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Logs.On }),
	boolkey("logs.daemon", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Logs.Daemon }),
	boolkey("logs.xray", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Logs.Xray }),
	boolkey("logs.singbox", "on|off", false, func(c *conf.Config) *bool { return &c.Settings.Logs.Singbox }),
	{
		name: "logs.max", help: "SIZE like 200mb, 0 for unlimited",
		get: func(c *conf.Config) string { return formatsize(c.Settings.Logs.Max) },
		set: func(c *conf.Config, v string) error {
			n, ok := parsesize(v)
			if !ok {
				return errors.New("expected a size like 200mb, or 0")
			}
			c.Settings.Logs.Max = n
			return nil
		},
	},
}

func findkey(name string) *confkey {
	for i := range confkeys {
		if confkeys[i].name == strings.ToLower(name) {
			return &confkeys[i]
		}
	}
	return nil
}

func keytable() string {
	w := 0
	for _, k := range confkeys {
		w = max(w, len(k.name))
	}
	var b strings.Builder
	for _, k := range confkeys {
		b.WriteString("  " + pad(k.name, w) + "  " + k.help + "\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func confcmd() *cobra.Command {
	c := parent("conf COMMAND", "get and set options",
		"Keys:\n"+keytable(),
		"settings", conflscmd(), confgetcmd(), confsetcmd(), confunsetcmd(), confeditcmd(), confpathcmd())
	c.Aliases = []string{"config", "cfg"}
	return c
}

func conflscmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "ls [OPTION]...",
		Aliases: []string{"list"},
		Short:   "list options",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			asjson, _ := c.Flags().GetBool("json")
			cfg, err := loadcfg()
			if err != nil {
				return err
			}
			if asjson {
				out := map[string]string{}
				for _, k := range confkeys {
					out[k.name] = k.get(&cfg)
				}
				return printjson(out)
			}
			w := 0
			for _, k := range confkeys {
				w = max(w, len(k.name))
			}
			for _, k := range confkeys {
				v := k.get(&cfg)
				if v == "" {
					v = paint(dim, "(default)")
				}
				say("%s  %s", pad(k.name, w), v)
			}
			return nil
		},
	}
	c.Flags().BoolP("json", "j", false, "print JSON")
	return c
}

func confgetcmd() *cobra.Command {
	return &cobra.Command{
		Use:               "get KEY...",
		Short:             "print values",
		Args:              minargs(1),
		ValidArgsFunction: completekeys,
		RunE: func(c *cobra.Command, args []string) error {
			cfg, err := loadcfg()
			if err != nil {
				return err
			}
			for _, a := range args {
				k := findkey(a)
				if k == nil {
					errf("unknown key '%s'", a)
					continue
				}
				say("%s", k.get(&cfg))
			}
			return nil
		},
	}
}

func confsetcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "set [OPTION]... KEY VALUE | KEY=VALUE...",
		Short:             "set options",
		Args:              minargs(1),
		ValidArgsFunction: completeset,
		RunE: func(c *cobra.Command, args []string) error {
			var pairs [][2]string
			if len(args) == 2 && !strings.Contains(args[0], "=") {
				pairs = append(pairs, [2]string{args[0], args[1]})
			} else {
				for _, a := range args {
					k, v, ok := strings.Cut(a, "=")
					if !ok {
						return usagef("missing value for '%s'", a)
					}
					pairs = append(pairs, [2]string{k, v})
				}
			}
			return applyconf(c, pairs, false)
		},
	}
	addverbose(c)
	return c
}

func confunsetcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "unset [OPTION]... KEY...",
		Short:             "reset to defaults",
		Args:              minargs(1),
		ValidArgsFunction: completekeys,
		RunE: func(c *cobra.Command, args []string) error {
			pairs := make([][2]string, len(args))
			for i, a := range args {
				pairs[i] = [2]string{a, ""}
			}
			return applyconf(c, pairs, true)
		},
	}
	addverbose(c)
	return c
}

func applyconf(c *cobra.Command, pairs [][2]string, todefault bool) error {
	cfg, err := loadcfg()
	if err != nil {
		return err
	}
	def := conf.Defaults()
	changed, reconnect := false, false
	for _, p := range pairs {
		k := findkey(p[0])
		if k == nil {
			errf("unknown key '%s'", p[0])
			continue
		}
		v := p[1]
		if todefault {
			v = k.get(&def)
		}
		if err := k.set(&cfg, v); err != nil {
			errf("invalid value '%s' for '%s': %v", v, k.name, err)
			continue
		}
		changed = true
		reconnect = reconnect || k.conn
		if verbose(c) {
			say("%s = %s", k.name, k.get(&cfg))
		}
		if k.name == "tun.config" && cfg.Settings.Advanced.Tunconfig == "custom" {
			if p, err := route.Custompath(); err == nil {
				if _, err := os.Stat(p); err != nil {
					warnf("%s missing, using the default", p)
				}
			}
		}
	}
	if !changed {
		return nil
	}
	if err := conf.Save(conf.Normalize(cfg)); err != nil {
		return err
	}
	if st, ok := status(); reconnect && ok && st.Connected {
		infof("applies on the next conn")
	}
	return nil
}

func confeditcmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "edit in $EDITOR",
		Args:  nargs,
		RunE: func(c *cobra.Command, args []string) error {
			p, err := conf.Path()
			if err != nil {
				return err
			}
			if _, err := os.Stat(p); os.IsNotExist(err) {
				if err := conf.Save(conf.Defaults()); err != nil {
					return err
				}
			}
			before, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			if err := editfile(p); err != nil {
				return err
			}
			after, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			cfg := conf.Defaults()
			if err := json.Unmarshal(after, &cfg); err != nil {
				_ = conf.Write(p, before, 0600)
				return fmt.Errorf("config.json is not valid JSON, changes rolled back: %v", err)
			}
			return conf.Save(conf.Normalize(cfg))
		},
	}
}

func confpathcmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "print the file path",
		Args:  nargs,
		RunE: func(c *cobra.Command, args []string) error {
			p, err := conf.Path()
			if err != nil {
				return err
			}
			say("%s", p)
			return nil
		},
	}
}
