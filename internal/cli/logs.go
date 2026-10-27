package cli

import (
	"context"
	"os"
	"os/signal"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/ctl"
	"github.com/alexandria-proxy/alexandria-cli/internal/ipc"
)

const logtick = 700 * time.Millisecond

var (
	logsources = []string{"daemon", "xray", "sing-box"}
	loglevels  = []string{"info", "warn", "error"}
)

type logquery struct {
	srcs, lvls, terms, nots []string
}

func parselogquery(words []string) logquery {
	var q logquery
	for _, word := range words {
		word = strings.ToLower(word)
		switch word {
		case "singbox":
			word = "sing-box"
		case "err":
			word = "error"
		}
		if neg := strings.TrimLeft(word, "-!"); neg != word {
			if neg != "" {
				q.nots = append(q.nots, neg)
			}
			continue
		}
		switch {
		case slices.Contains(logsources, word):
			q.srcs = append(q.srcs, word)
		case slices.Contains(loglevels, word):
			q.lvls = append(q.lvls, word)
		default:
			q.terms = append(q.terms, word)
		}
	}
	return q
}

func (q logquery) match(l ipc.Logline) bool {
	if len(q.srcs) > 0 && !slices.Contains(q.srcs, strings.ToLower(l.Src)) {
		return false
	}
	if len(q.lvls) > 0 && !slices.Contains(q.lvls, strings.ToLower(l.Lvl)) {
		return false
	}
	hay := strings.ToLower(l.Src + " " + l.Lvl + " " + l.Text)
	for _, t := range q.terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	for _, n := range q.nots {
		if strings.Contains(hay, n) {
			return false
		}
	}
	return true
}

func logscmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "logs [OPTION]... [FILTER]...",
		Aliases: []string{"log"},
		Short:   "print daemon, xray and sing-box logs",
		Long: "Print the last lines the daemon, xray and sing-box logged.\n" +
			"A FILTER is a source (daemon, xray, sing-box), a level (info, warn, err) or any\n" +
			"other word that has to appear; -WORD or !WORD hides lines with WORD.",
		Example: "  alexandria-cli logs -n 100\n" +
			"  alexandria-cli logs -f xray err\n" +
			"  alexandria-cli logs -- -sniff",
		GroupID: "logs",
		Args:    cobra.ArbitraryArgs,
		ValidArgsFunction: func(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			return []cobra.Completion{"daemon", "xray", "sing-box", "info", "warn", "err"}, cobra.ShellCompDirectiveNoFileComp
		},
		Annotations: map[string]string{"exit": exitdefault + "\n  130  if --follow was interrupted."},
		RunE:        runlogs,
	}
	f := c.Flags()
	f.BoolP("follow", "f", false, "keep printing new lines as they come")
	f.IntP("lines", "n", 50, "print the last `NUM` lines, 0 for all")
	return c
}

func runlogs(c *cobra.Command, args []string) error {
	follow, _ := c.Flags().GetBool("follow")
	n, _ := c.Flags().GetInt("lines")
	if n < 0 {
		return usagef("invalid number of lines: '%d'", n)
	}
	q := parselogquery(args)
	if !ctl.Up() {
		if !follow {
			return nil
		}
		if err := ctl.Ensure(); err != nil {
			return err
		}
	}
	resp, err := ctl.Call(ipc.Request{Cmd: ipc.Cmdlogs})
	if err != nil {
		return err
	}
	var lines []ipc.Logline
	for _, l := range resp.Logs {
		if q.match(l) {
			lines = append(lines, l)
		}
	}
	if n > 0 && len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	for _, l := range lines {
		printlog(l)
	}
	if !follow {
		return nil
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	since := resp.Seq
	for {
		select {
		case <-ctx.Done():
			return exiterr{130}
		case <-time.After(logtick):
		}
		resp, err := ipc.Send(ipc.Request{Cmd: ipc.Cmdlogs, Since: since})
		if err != nil || resp.Error != "" {
			continue
		}
		if resp.Seq < since {
			infof("the daemon restarted")
			since = 0
			continue
		}
		for _, l := range resp.Logs {
			if q.match(l) {
				printlog(l)
			}
		}
		since = resp.Seq
	}
}

func printlog(l ipc.Logline) {
	lvl, code := "info", ""
	switch l.Lvl {
	case "warn":
		lvl, code = "warn", yellow
	case "error":
		lvl, code = "err", red
	}
	at := time.Unix(l.At, 0).Format("15:04:05")
	say("%s  %s  %s %s", paint(dim, at), padpaint(dim, l.Src, 8), paint(code, lvl+":"), l.Text)
}
