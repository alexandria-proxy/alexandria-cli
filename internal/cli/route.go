package cli

import (
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/route"
)

func routecmd() *cobra.Command {
	return parent("route COMMAND", "TUN bypass list",
		"One CIDR, IP or domain per line, '#' comments. Reloaded live.",
		"settings", routelscmd(), routeaddcmd(), routermcmd(), routeeditcmd(), routepathcmd())
}

func warnbad(bad []route.Bad) {
	for _, b := range bad {
		warnf("line %d: '%s' is not a CIDR, IP or domain", b.Line, b.Text)
	}
}

func routelscmd() *cobra.Command {
	c := &cobra.Command{
		Use:     "ls [OPTION]...",
		Aliases: []string{"list"},
		Short:   "list entries",
		Args:    nargs,
		RunE: func(c *cobra.Command, args []string) error {
			asjson, _ := c.Flags().GetBool("json")
			entries, bad, err := route.Load()
			if err != nil {
				return err
			}
			if asjson {
				type badjson struct {
					Line int    `json:"line"`
					Text string `json:"text"`
				}
				out := struct {
					Entries []string  `json:"entries"`
					Bad     []badjson `json:"bad"`
				}{Entries: append([]string{}, entries...), Bad: []badjson{}}
				for _, b := range bad {
					out.Bad = append(out.Bad, badjson{b.Line, b.Text})
				}
				return printjson(out)
			}
			for _, e := range entries {
				say("%s", e)
			}
			warnbad(bad)
			return nil
		},
	}
	c.Flags().BoolP("json", "j", false, "print JSON")
	return c
}

func routeaddcmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "add [OPTION]... ENTRY...",
		Short: "add entries",
		Args:  minargs(1),
		RunE: func(c *cobra.Command, args []string) error {
			list, err := operands(args)
			if err != nil {
				return err
			}
			for _, v := range list {
				added, err := route.Add(v)
				switch {
				case errors.Is(err, route.Errinvalid):
					errf("'%s': %v", v, err)
				case err != nil:
					return err
				case added && verbose(c):
					say("added '%s'", v)
				case !added && verbose(c):
					say("'%s' is already listed", v)
				}
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

func routermcmd() *cobra.Command {
	c := &cobra.Command{
		Use:               "rm [OPTION]... ENTRY...",
		Aliases:           []string{"remove"},
		Short:             "remove entries",
		Args:              minargs(1),
		ValidArgsFunction: completeroutes,
		RunE: func(c *cobra.Command, args []string) error {
			list, err := operands(args)
			if err != nil {
				return err
			}
			for _, v := range list {
				removed, err := route.Remove(v)
				switch {
				case err != nil:
					return err
				case !removed:
					errf("'%s' is not listed", v)
				case verbose(c):
					say("removed '%s'", v)
				}
			}
			return nil
		},
	}
	addverbose(c)
	return c
}

func routeeditcmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "edit in $EDITOR",
		Args:  nargs,
		RunE: func(c *cobra.Command, args []string) error {
			p, err := route.Path()
			if err != nil {
				return err
			}
			if _, err := os.Stat(p); os.IsNotExist(err) {
				if _, err := conf.Mkdir(); err != nil {
					return err
				}
				if err := conf.Write(p, []byte(route.Header), 0600); err != nil {
					return err
				}
			}
			if err := editfile(p); err != nil {
				return err
			}
			_, bad, err := route.Load()
			if err != nil {
				return err
			}
			warnbad(bad)
			return nil
		},
	}
}

func routepathcmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "print the file path",
		Args:  nargs,
		RunE: func(c *cobra.Command, args []string) error {
			p, err := route.Path()
			if err != nil {
				return err
			}
			say("%s", p)
			return nil
		},
	}
}
