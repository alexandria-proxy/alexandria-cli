package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

func conncmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "conn [OPTION]... [SERVER]",
		Aliases:           []string{"connect", "c"},
		Short:             "connect; SERVER is a number from ls or a part of the name",
		GroupID:           "conn",
		Args:              maxargs(1),
		ValidArgsFunction: completeservers,
		RunE:              runconn,
	}
	f := c.Flags()
	f.BoolP("tun", "t", false, "TUN mode")
	f.BoolP("proxy", "p", false, "proxy mode")
	f.BoolP("best", "b", false, "fastest working server, SERVER names a subscription")
	addverbose(c)
	return c
}

func runconn(c *cobra.Command, args []string) error {
	tun, _ := c.Flags().GetBool("tun")
	proxy, _ := c.Flags().GetBool("proxy")
	best, _ := c.Flags().GetBool("best")
	if tun && proxy {
		return usagef("--tun and --proxy are mutually exclusive")
	}
	cfg, err := loadcfg()
	if err != nil {
		return err
	}
	mode := cfg.Mode
	switch {
	case tun:
		mode = "tun"
	case proxy:
		mode = "proxy"
	}
	if mode != "tun" {
		mode = "proxy"
	}

	if err := ensurefor(mode); err != nil {
		return err
	}
	list, err := ctl.Livesubs()
	if err != nil {
		return err
	}

	var resp ipc.Response
	if best {
		si, err := bestsub(list, cfg, args)
		if err != nil {
			return err
		}
		resp, err = ctl.Calltimeout(ipc.Request{Cmd: ipc.Cmdbest, Url: list[si].Url, Mode: mode}, besttimeout)
		if err != nil {
			return connerr(resp, err)
		}
	} else {
		e, err := target(list, cfg, args)
		if err != nil {
			return err
		}
		resp, err = ctl.Call(ipc.Request{Cmd: ipc.Cmdconnect, Url: list[e.sub].Url, Srvidx: e.idx, Mode: mode})
		if err != nil {
			return connerr(resp, err)
		}
	}

	if cfg, err = conf.Load(); err == nil {
		cfg.Mode = mode
		cfg.Lasturl, cfg.Lastsrv = resp.Activeurl, resp.Activesrv
		if err := conf.Save(cfg); err != nil {
			warnf("can't save the server: %v", err)
		}
	}
	if verbose(c) {
		say("conn  %s  %s", mode, servername(list, resp.Activeurl, resp.Activesrv))
	}
	say("%s", proxyline(cfg.Settings.Socksport))
	return nil
}

func bestsub(list []sub.Subscription, cfg conf.Config, args []string) (int, error) {
	if len(args) == 1 {
		return findsub(list, args[0])
	}
	if len(list) == 0 {
		return findsub(list, "")
	}
	for i, s := range list {
		if s.Url == cfg.Lasturl {
			return i, nil
		}
	}
	return 0, nil
}

func target(list []sub.Subscription, cfg conf.Config, args []string) (entry, error) {
	if len(args) == 1 {
		return findserver(list, args[0])
	}
	all := number(list)
	if len(all) == 0 {
		return findserver(list, "")
	}
	for _, e := range all {
		if list[e.sub].Url == cfg.Lasturl && e.idx == cfg.Lastsrv {
			return e, nil
		}
	}
	return all[0], nil
}

func connerr(resp ipc.Response, err error) error {
	if resp.Code == "needelevate" {
		return errors.New("tun mode needs the daemon to run as root: alexandria-cli svc on")
	}
	return err
}

func servername(list []sub.Subscription, url string, idx int) string {
	for _, e := range number(list) {
		if list[e.sub].Url == url && e.idx == idx {
			return fmt.Sprintf("{%d} %s", e.num, e.srv.Name)
		}
	}
	return "?"
}

func proxyline(port int) string {
	return "socks 127.0.0.1:" + strconv.Itoa(port) + "  http 127.0.0.1:" + strconv.Itoa(port+1)
}

func dconncmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "dconn [OPTION]...",
		Aliases: []string{"disconnect", "d"},
		Short:   "disconnect",
		GroupID: "conn",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			if !ctl.Up() {
				return nil
			}
			if _, err := ctl.Call(ipc.Request{Cmd: ipc.Cmddisconnect}); err != nil {
				return err
			}
			if verbose(c) {
				say("dconn")
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

type statjson struct {
	Connected bool   `json:"connected"`
	Mode      string `json:"mode,omitempty"`
	Server    int    `json:"server,omitempty"`
	Name      string `json:"name,omitempty"`
	Protocol  string `json:"protocol,omitempty"`
	Sub       string `json:"sub,omitempty"`
	Since     int64  `json:"since,omitempty"`
	Socks     string `json:"socks,omitempty"`
	Http      string `json:"http,omitempty"`
	Uprate    int64  `json:"up_rate,omitempty"`
	Downrate  int64  `json:"down_rate,omitempty"`
	Uptotal   int64  `json:"up_total,omitempty"`
	Downtotal int64  `json:"down_total,omitempty"`
	Error     string `json:"error,omitempty"`
}

func statcmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "stat [OPTION]...",
		Aliases: []string{"status", "st"},
		Short:   "connection status, exit 3 if disconnected",
		GroupID: "conn",
		Args:    nargs,
		RunE:    runstat,
	}
	f := c.Flags()
	f.BoolP("json", "j", false, "print JSON")
	f.BoolP("quiet", "q", false, "exit status only")
	f.BoolP("watch", "w", false, "redraw every second")
	return c
}

func runstat(c *cobra.Command, args []string) error {
	asjson, _ := c.Flags().GetBool("json")
	quiet, _ := c.Flags().GetBool("quiet")
	watch, _ := c.Flags().GetBool("watch")
	if watch && (asjson || quiet) {
		return usagef("--watch can't be combined with --json or --quiet")
	}
	if !watch {
		lines, js, live := statnow()
		switch {
		case quiet:
		case asjson:
			if err := printjson(js); err != nil {
				return err
			}
		default:
			for _, l := range lines {
				say("%s", l)
			}
			if js.Error != "" {
				errf("%s", js.Error)
			}
		}
		if !live {
			return exiterr{3}
		}
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	tty := isterm(os.Stdout)
	drawn := 0
	for {
		lines, js, _ := statnow()
		if js.Error != "" {
			lines = append(lines, "err: "+js.Error)
		}
		if tty && drawn > 0 {
			fmt.Fprintf(stdout, "\x1b[%dA\x1b[J", drawn)
		}
		fmt.Fprintln(stdout, strings.Join(lines, "\n"))
		drawn = len(lines)
		select {
		case <-ctx.Done():
			return exiterr{130}
		case <-time.After(time.Second):
		}
	}
}

func statnow() ([]string, statjson, bool) {
	st, ok := status()
	if !ok || !st.Connected {
		js := statjson{Error: st.Error}
		return []string{paint(dim, "dconn")}, js, false
	}
	cfg, _ := conf.Load()
	port := cfg.Settings.Socksport
	list, _ := ctl.Subs()
	js := statjson{
		Connected: true,
		Mode:      st.Mode,
		Since:     st.Since,
		Socks:     "127.0.0.1:" + strconv.Itoa(port),
		Http:      "127.0.0.1:" + strconv.Itoa(port+1),
	}
	lines := []string{paint(green, "conn") + strings.Repeat(" ", 7) + st.Mode + "  " + uptime(st.Since)}
	for _, e := range number(list) {
		if list[e.sub].Url == st.Activeurl && e.idx == st.Activesrv {
			js.Server, js.Name, js.Protocol, js.Sub = e.num, e.srv.Name, protolabel(e.srv), list[e.sub].Name
			lines = append(lines,
				row("server", fmt.Sprintf("{%d} %s  %s", e.num, e.srv.Name, paint(dim, protolabel(e.srv)))),
				row("sub", list[e.sub].Name))
			break
		}
	}
	lines = append(lines, row("proxy", proxyline(port)))
	if st.Hasstats {
		js.Uprate, js.Downrate, js.Uptotal, js.Downtotal = st.Uprate, st.Downrate, st.Uptotal, st.Downtotal
		lines = append(lines, row("traffic", fmt.Sprintf("↓ %s/s  ↑ %s/s  %s total",
			humanbytes(st.Downrate), humanbytes(st.Uprate), humanbytes(st.Uptotal+st.Downtotal))))
	}
	return lines, js, true
}

func row(key, val string) string {
	return pad(key, 9) + "  " + val
}
