package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

func subcmd() *cobra.Command {
	return parent("sub COMMAND", "manage subscriptions",
		"SUB is a number from 'sub ls' or a part of the name.",
		"subs", subaddcmd(), sublscmd(), subupcmd(), subrmcmd(), subinfocmd())
}

func subaddcmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "add [OPTION]... URL...",
		Aliases: []string{"a"},
		Short:   "add subscriptions",
		Args:    minargs(1),
		RunE: func(c *cobra.Command, args []string) error {
			urls, err := operands(args)
			if err != nil {
				return err
			}
			if len(urls) == 0 {
				return usagef("missing operand")
			}
			if err := ctl.Ensure(); err != nil {
				return err
			}
			for _, u := range urls {
				resp, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdadd, Url: u})
				if err != nil {
					errf("%s: %v", u, err)
					continue
				}
				if verbose(c) {
					say("added %s", subdesc(resp.Subscriptions, u))
				}
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

func subdesc(list []sub.Subscription, url string) string {
	url = strings.TrimSpace(url)
	for _, s := range list {
		if s.Url == url {
			return fmt.Sprintf("'%s' (%d servers)", s.Name, len(s.Servers))
		}
	}
	return "'" + url + "'"
}

func sublscmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "ls [OPTION]...",
		Aliases: []string{"list"},
		Short:   "list subscriptions",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			asjson, _ := c.Flags().GetBool("json")
			list, err := ctl.Subs()
			if err != nil {
				return err
			}
			if asjson {
				out := []subjson{}
				for i, s := range list {
					out = append(out, subjson{Num: i + 1, Name: s.Name, Url: s.Url, Used: s.Usedbytes, Total: s.Totalbytes,
						Expires: day(s.Expires), Updated: stamp(s.Updatedat), Servers: []srvjson{}})
				}
				return printjson(out)
			}
			if len(list) == 0 {
				infof("no subscriptions")
				return nil
			}
			numw, namew, cntw := 0, 0, 0
			for i, s := range list {
				numw = max(numw, len(strconv.Itoa(i+1))+2)
				namew = max(namew, width(s.Name))
				cntw = max(cntw, len(strconv.Itoa(len(s.Servers))))
			}
			for i, s := range list {
				parts := []string{
					pad(fmt.Sprintf("{%d}", i+1), numw),
					padpaint(bold, s.Name, namew),
					padleft(strconv.Itoa(len(s.Servers)), cntw) + " servers",
				}
				if u := usage(s); u != "" {
					parts = append(parts, u)
				}
				if d := day(s.Expires); d != "" {
					parts = append(parts, "expires "+d)
				}
				say("%s", strings.TrimRight(strings.Join(parts, "  "), " "))
			}
			return nil
		},
	}
	c.Flags().BoolP("json", "j", false, "print JSON")
	return c
}

func subupcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "up [OPTION]... [SUB]...",
		Aliases:           []string{"update"},
		Short:             "update subscriptions, all by default",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completesubs,
		RunE: func(c *cobra.Command, args []string) error {
			list, err := ctl.Livesubs()
			if err != nil {
				return err
			}
			for _, i := range findsubs(list, args) {
				resp, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdadd, Url: list[i].Url})
				if err != nil {
					errf("%s: %v", list[i].Name, err)
					continue
				}
				if verbose(c) {
					say("updated %s", subdesc(resp.Subscriptions, list[i].Url))
				}
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

func subrmcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "rm [OPTION]... SUB...",
		Aliases:           []string{"remove"},
		Short:             "remove subscriptions",
		Args:              minargs(1),
		ValidArgsFunction: completesubs,
		RunE: func(c *cobra.Command, args []string) error {
			list, err := ctl.Livesubs()
			if err != nil {
				return err
			}
			cfg, _ := conf.Load()
			dropped := false
			for _, i := range findsubs(list, args) {
				if _, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdremove, Url: list[i].Url}); err != nil {
					errf("%s: %v", list[i].Name, err)
					continue
				}
				if list[i].Url == cfg.Lasturl {
					cfg.Lasturl, cfg.Lastsrv, dropped = "", 0, true
				}
				if verbose(c) {
					say("removed '%s'", list[i].Name)
				}
			}
			if dropped {
				_ = conf.Save(cfg)
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

func subinfocmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "info [OPTION]... SUB...",
		Short:             "show traffic, expiry and provider links",
		Args:              minargs(1),
		ValidArgsFunction: completesubs,
		RunE: func(c *cobra.Command, args []string) error {
			asjson, _ := c.Flags().GetBool("json")
			list, err := ctl.Subs()
			if err != nil {
				return err
			}
			idxs := findsubs(list, args)
			if asjson {
				type infojson struct {
					subjson
					Autoupdate string `json:"autoupdate,omitempty"`
					Note       string `json:"note,omitempty"`
					Support    string `json:"support,omitempty"`
					Announce   string `json:"announce,omitempty"`
				}
				out := []infojson{}
				for _, i := range idxs {
					s := list[i]
					out = append(out, infojson{
						subjson: subjson{Num: i + 1, Name: s.Name, Url: s.Url, Used: s.Usedbytes, Total: s.Totalbytes,
							Expires: day(s.Expires), Updated: stamp(s.Updatedat), Servers: []srvjson{}},
						Autoupdate: hours(s.Autoupdate), Note: s.Note, Support: s.Supporturl, Announce: s.Announceurl,
					})
				}
				return printjson(out)
			}
			for n, i := range idxs {
				if n > 0 {
					say("")
				}
				s := list[i]
				rows := [][2]string{
					{"name", s.Name}, {"url", s.Url}, {"servers", strconv.Itoa(len(s.Servers))},
					{"used", usage(s)}, {"expires", day(s.Expires)}, {"updated", stamp(s.Updatedat)},
					{"autoupdate", hours(s.Autoupdate)}, {"note", s.Note},
					{"support", s.Supporturl}, {"announce", s.Announceurl},
				}
				for _, r := range rows {
					if r[1] != "" {
						say("%s  %s", pad(r[0], 10), r[1])
					}
				}
			}
			return nil
		},
	}
	c.Flags().BoolP("json", "j", false, "print JSON")
	return c
}
