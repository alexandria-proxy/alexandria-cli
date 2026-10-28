package cli

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

type lsopts struct {
	sort    string
	reverse bool
	unique  bool
	json    bool
}

type srvjson struct {
	Num      int    `json:"num"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Pingms   int    `json:"ping_ms"`
	Json     bool   `json:"json"`
	Current  bool   `json:"current"`
}

type subjson struct {
	Num     int       `json:"num"`
	Name    string    `json:"name"`
	Url     string    `json:"url"`
	Used    int64     `json:"used_bytes"`
	Total   int64     `json:"total_bytes"`
	Expires string    `json:"expires,omitempty"`
	Updated string    `json:"updated,omitempty"`
	Servers []srvjson `json:"servers"`
}

func lscmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "ls [OPTION]... [SUB]...",
		Aliases:           []string{"list"},
		Short:             "list servers",
		GroupID:           "servers",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completesubs,
		RunE: func(c *cobra.Command, args []string) error {
			o, err := readls(c)
			if err != nil {
				return err
			}
			list, err := ctl.Subs()
			if err != nil {
				return err
			}
			if len(list) == 0 && !o.json {
				infof("no subscriptions")
				return nil
			}
			return printservers(list, findsubs(list, args), o)
		},
	}
	f := c.Flags()
	f.StringP("sort", "s", "none", "sort servers by `WORD`: none, ping or name")
	f.BoolP("reverse", "r", false, "reverse the sort order")
	f.BoolP("unique", "u", false, "hide duplicates")
	f.BoolP("json", "j", false, "print JSON")
	_ = c.RegisterFlagCompletionFunc("sort", cobra.FixedCompletions([]cobra.Completion{"none", "ping", "name"}, cobra.ShellCompDirectiveNoFileComp))
	return c
}

func readls(c *cobra.Command) (lsopts, error) {
	var o lsopts
	o.sort, _ = c.Flags().GetString("sort")
	o.reverse, _ = c.Flags().GetBool("reverse")
	o.unique, _ = c.Flags().GetBool("unique")
	o.json, _ = c.Flags().GetBool("json")
	switch o.sort {
	case "none", "ping", "name":
	default:
		return o, usagef("invalid argument '%s' for '--sort'\nValid arguments are: 'none', 'ping', 'name'", o.sort)
	}
	return o, nil
}

func pingcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "ping [OPTION]... [SUB]...",
		Short:             "measure latency",
		GroupID:           "servers",
		Args:              cobra.ArbitraryArgs,
		ValidArgsFunction: completesubs,
		RunE: func(c *cobra.Command, args []string) error {
			asjson, _ := c.Flags().GetBool("json")
			list, err := ctl.Livesubs()
			if err != nil {
				return err
			}
			idxs := findsubs(list, args)
			var latest []sub.Subscription
			for _, i := range idxs {
				if len(list[i].Servers) == 0 {
					continue
				}
				resp, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdprobe, Url: list[i].Url})
				if err != nil {
					errf("%s: %v", list[i].Name, err)
					continue
				}
				latest = resp.Subscriptions
			}
			if len(latest) == len(list) {
				list = latest
			}
			return printservers(list, idxs, lsopts{sort: "ping", json: asjson})
		},
	}
	c.Flags().BoolP("json", "j", false, "print JSON")
	return c
}

func printservers(list []sub.Subscription, idxs []int, o lsopts) error {
	curl, cidx := ctl.Current()
	all := number(list)
	seen := map[string]bool{}
	groups := make(map[int][]entry)
	for _, e := range all {
		if o.unique && e.srv.Host != "" {
			key := strings.ToLower(e.srv.Protocol) + "|" + e.srv.Host + "|" + strconv.Itoa(e.srv.Port)
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		groups[e.sub] = append(groups[e.sub], e)
	}
	for si, g := range groups {
		switch o.sort {
		case "ping":
			slices.SortStableFunc(g, func(a, b entry) int { return pingrank(a.srv.Pingms) - pingrank(b.srv.Pingms) })
		case "name":
			slices.SortStableFunc(g, func(a, b entry) int {
				return strings.Compare(strings.ToLower(a.srv.Name), strings.ToLower(b.srv.Name))
			})
		}
		if o.reverse {
			slices.Reverse(g)
		}
		groups[si] = g
	}
	iscur := func(e entry) bool { return list[e.sub].Url == curl && e.idx == cidx }

	if o.json {
		out := []subjson{}
		for _, si := range idxs {
			s := list[si]
			sj := subjson{Num: si + 1, Name: s.Name, Url: s.Url, Used: s.Usedbytes, Total: s.Totalbytes,
				Expires: day(s.Expires), Updated: stamp(s.Updatedat), Servers: []srvjson{}}
			for _, e := range groups[si] {
				sj.Servers = append(sj.Servers, srvjson{Num: e.num, Name: e.srv.Name, Protocol: protolabel(e.srv),
					Host: e.srv.Host, Port: e.srv.Port, Pingms: e.srv.Pingms, Json: isjson(e.srv.Raw), Current: iscur(e)})
			}
			out = append(out, sj)
		}
		return printjson(out)
	}

	numw, namew, protow := 0, 0, 0
	for _, si := range idxs {
		for _, e := range groups[si] {
			numw = max(numw, len(strconv.Itoa(e.num))+2)
			namew = max(namew, width(e.srv.Name))
			protow = max(protow, width(protolabel(e.srv)))
		}
	}
	for n, si := range idxs {
		if n > 0 {
			say("")
		}
		say("%s", subheader(list[si]))
		if len(groups[si]) == 0 {
			say("  %s", paint(dim, "(no servers)"))
		}
		for _, e := range groups[si] {
			mark := " "
			if iscur(e) {
				mark = paint(green+";"+bold, "*")
			}
			num := fmt.Sprintf("{%d}", e.num)
			gap := strings.Repeat(" ", numw-len(num)+1)
			ms := e.srv.Pingms
			say("%s%s%s%s  %s  %s", num, mark, gap, pad(e.srv.Name, namew), pad(protolabel(e.srv), protow),
				paint(pingcolor(ms), padleft(pingtext(ms), 5)))
		}
	}
	return nil
}

func subheader(s sub.Subscription) string {
	parts := []string{paint(bold, s.Name)}
	if u := usage(s); u != "" {
		parts = append(parts, u)
	}
	if d := day(s.Expires); d != "" {
		parts = append(parts, "expires "+d)
	}
	return strings.Join(parts, " ")
}
