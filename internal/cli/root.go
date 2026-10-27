package cli

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/alexandria-proxy/alexandria-cli/internal/conf"
	"github.com/alexandria-proxy/alexandria-cli/internal/dmn"
)

const (
	prog     = "alexandria-cli"
	homepage = "https://github.com/alexandria-proxy/alexandria-cli"
)

const exitdefault = `  0  if OK,
  1  if a problem occurred,
  2  if the command line was wrong.`

func Execute(version, cores string) int {
	if os.Getenv("ALEXANDRIA_DAEMON") == "1" {
		return rundaemon(slices.Contains(os.Args[1:], "--autoconnect"), false)
	}
	root := newroot(version, cores)
	c, err := root.ExecuteC()
	return finish(c, err)
}

func finish(c *cobra.Command, err error) int {
	var ue usageerr
	var ee exiterr
	switch {
	case err == nil:
	case errors.As(err, &ee):
		return ee.code
	case errors.As(err, &ue):
		errf("%s", ue.msg)
		fmt.Fprintf(stderr, "Try '%s --help' for more information.\n", c.CommandPath())
		return 2
	default:
		errf("%v", err)
		return 1
	}
	if failed {
		return 1
	}
	return 0
}

func rundaemon(autoconnect, service bool) int {
	if err := dmn.Run(autoconnect, service); err != nil {
		errf("daemon: %v", err)
		return 1
	}
	return 0
}

func newroot(version, cores string) *cobra.Command {
	cobra.EnableCommandSorting = false
	root := &cobra.Command{
		Use:   prog + " [OPTION]... COMMAND [ARG]...",
		Short: "control the alexandria proxy daemon",
		Long: "Control the Alexandria proxy daemon: connect through Xray-core servers from your\n" +
			"subscriptions, check status and tweak settings. The proxy keeps running in the\n" +
			"background after the command returns.",
		Version:       version,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args: func(c *cobra.Command, args []string) error {
			if len(args) > 0 {
				return usagef("unknown command '%s'", args[0])
			}
			return nil
		},
		RunE: func(c *cobra.Command, args []string) error {
			if on, _ := c.Flags().GetBool("daemon"); on {
				auto, _ := c.Flags().GetBool("autoconnect")
				svc, _ := c.Flags().GetBool("service")
				if code := rundaemon(auto, svc); code != 0 {
					return exiterr{code}
				}
				return nil
			}
			return c.Help()
		},
		PersistentPreRunE: func(c *cobra.Command, args []string) error {
			if dir, _ := c.Flags().GetString("dir"); dir != "" {
				_ = os.Setenv(conf.Direnv, dir)
			}
			return setcolor(colorwhen)
		},
	}

	root.SetVersionTemplate(prog + " {{.Version}}\n" + cores + "\n" +
		"License MPL-2.0 <https://mozilla.org/MPL/2.0/>.\n" +
		"This is free software: you are free to change and redistribute it.\n" +
		"There is NO WARRANTY, to the extent permitted by law.\n")

	pf := root.PersistentFlags()
	pf.StringVar(&colorwhen, "color", "auto", "colorize the output; `WHEN` is 'always', 'never', or 'auto'")
	pf.Lookup("color").NoOptDefVal = "always"
	pf.BoolP("help", "h", false, "display this help and exit")
	root.Flags().Bool("version", false, "output version information and exit")
	root.Flags().Bool("daemon", false, "run the background daemon")
	root.Flags().Bool("autoconnect", false, "connect to the last used server on start")
	root.Flags().Bool("service", false, "run as the system service")
	pf.String("dir", "", "use `DIR` as the config dir")
	_ = root.Flags().MarkHidden("daemon")
	_ = root.Flags().MarkHidden("autoconnect")
	_ = root.Flags().MarkHidden("service")
	_ = pf.MarkHidden("dir")

	root.AddGroup(
		&cobra.Group{ID: "conn", Title: "Connection:"},
		&cobra.Group{ID: "servers", Title: "Servers:"},
		&cobra.Group{ID: "subs", Title: "Subscriptions:"},
		&cobra.Group{ID: "logs", Title: "Logs:"},
		&cobra.Group{ID: "settings", Title: "Settings:"},
		&cobra.Group{ID: "other", Title: "Other:"},
	)
	root.AddCommand(
		conncmd(), dconncmd(), statcmd(),
		lscmd(), pingcmd(),
		subcmd(),
		logscmd(),
		confcmd(), routecmd(), autostartcmd(), svccmd(), resetcmd(),
		completioncmd(),
	)
	root.SetHelpCommand(helpcmd())
	root.SetHelpCommandGroupID("other")
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetHelpFunc(help)
	root.SetFlagErrorFunc(flagerr)

	walk(root, func(c *cobra.Command) {
		c.Flags().SortFlags = false
		c.PersistentFlags().SortFlags = false
		c.SilenceErrors, c.SilenceUsage = true, true
	})
	return root
}

func walk(c *cobra.Command, fn func(*cobra.Command)) {
	fn(c)
	for _, sub := range c.Commands() {
		walk(sub, fn)
	}
}

func setcolor(v string) error {
	switch v {
	case "always", "yes", "force":
		colorwhen = "always"
	case "never", "no", "none":
		colorwhen = "never"
	case "auto", "tty", "if-tty":
		colorwhen = "auto"
	default:
		return usagef("invalid argument '%s' for '--color'\nValid arguments are: 'always', 'never', 'auto'", v)
	}
	return nil
}

func parent(use, short, long, group string, kids ...*cobra.Command) *cobra.Command {
	c := &cobra.Command{
		Use:     use,
		Short:   short,
		Long:    long,
		GroupID: group,
		Args:    cobra.ArbitraryArgs,
		RunE: func(c *cobra.Command, args []string) error {
			if len(args) == 0 {
				return c.Help()
			}
			return usagef("unknown command '%s'", args[0])
		},
	}
	c.AddCommand(kids...)
	return c
}

func helpcmd() *cobra.Command {
	return &cobra.Command{
		Use:   "help [COMMAND]...",
		Short: "display help for COMMAND",
		Long:  "Display help for COMMAND, or the overview with no COMMAND.",
		ValidArgsFunction: func(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
			target, _, err := c.Root().Find(args)
			if err != nil {
				return nil, cobra.ShellCompDirectiveNoFileComp
			}
			var out []cobra.Completion
			for _, sub := range target.Commands() {
				if sub.IsAvailableCommand() {
					out = append(out, cobra.CompletionWithDesc(sub.Name(), sub.Short))
				}
			}
			return out, cobra.ShellCompDirectiveNoFileComp
		},
		RunE: func(c *cobra.Command, args []string) error {
			target, rest, err := c.Root().Find(args)
			if err != nil || len(rest) > 0 {
				return usagef("no help for '%s'", strings.Join(args, " "))
			}
			target.InitDefaultHelpFlag()
			return target.Help()
		},
	}
}

func completioncmd() *cobra.Command {
	return &cobra.Command{
		Use:   "completion SHELL",
		Short: "print a shell completion script",
		Long: "Print the completion script for SHELL: bash, zsh, fish or powershell.\n" +
			"Server numbers, subscriptions, settings keys and routes complete too.",
		Example: "  alexandria-cli completion bash > ~/.local/share/bash-completion/completions/alexandria-cli\n" +
			"  alexandria-cli completion zsh > \"${fpath[1]}/_alexandria-cli\"\n" +
			"  alexandria-cli completion fish > ~/.config/fish/completions/alexandria-cli.fish\n" +
			"  alexandria-cli completion powershell >> $PROFILE",
		GroupID:   "other",
		Args:      rangeargs(1, 1),
		ValidArgs: []cobra.Completion{"bash", "zsh", "fish", "powershell"},
		RunE: func(c *cobra.Command, args []string) error {
			root := c.Root()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(stdout, true)
			case "zsh":
				return root.GenZshCompletion(stdout)
			case "fish":
				return root.GenFishCompletion(stdout, true)
			case "powershell", "pwsh":
				return root.GenPowerShellCompletionWithDesc(stdout)
			}
			return usagef("unsupported shell '%s'", args[0])
		},
	}
}

var badarg = regexp.MustCompile(`^invalid argument "(.*)" for "(?:-\w, )?(--[\w.-]+)" flag`)

func flagerr(c *cobra.Command, err error) error {
	msg := err.Error()
	switch {
	case strings.HasPrefix(msg, "unknown flag: "):
		return usagef("unrecognized option '%s'", strings.TrimPrefix(msg, "unknown flag: "))
	case strings.HasPrefix(msg, "unknown shorthand flag: "):
		r := strings.TrimPrefix(msg, "unknown shorthand flag: ")
		if i := strings.Index(r, " in "); i >= 0 {
			r = r[:i]
		}
		return usagef("invalid option -- %s", r)
	case strings.HasPrefix(msg, "flag needs an argument: "):
		r := strings.TrimPrefix(msg, "flag needs an argument: ")
		if strings.HasPrefix(r, "--") {
			return usagef("option '%s' requires an argument", r)
		}
		if i := strings.Index(r, " in "); i >= 0 {
			r = r[:i]
		}
		return usagef("option requires an argument -- %s", r)
	}
	if m := badarg.FindStringSubmatch(msg); m != nil {
		return usagef("invalid argument '%s' for '%s'", m[1], m[2])
	}
	return usageerr{msg}
}

func synopsis(c *cobra.Command) string {
	return strings.Join(strings.Fields(strings.ReplaceAll(c.Use, "[OPTION]...", "")), " ")
}

func cmdwidth(cmds []*cobra.Command) int {
	w := 0
	for _, c := range cmds {
		w = max(w, width(synopsis(c)))
	}
	return w
}

func cmdlist(b *strings.Builder, cmds []*cobra.Command, w int) {
	for _, c := range cmds {
		fmt.Fprintf(b, "  %s  %s\n", pad(synopsis(c), w), c.Short)
	}
}

func help(c *cobra.Command, _ []string) {
	var b strings.Builder
	use := c.Use
	if c.HasParent() {
		use = c.Parent().CommandPath() + " " + c.Use
	}
	b.WriteString("Usage: " + use + "\n")
	desc := c.Long
	if desc == "" {
		desc = strings.ToUpper(c.Short[:1]) + c.Short[1:] + "."
	}
	b.WriteString(desc + "\n")
	if len(c.Aliases) > 0 {
		b.WriteString("\nAliases: " + strings.Join(c.Aliases, ", ") + "\n")
	}

	var kids []*cobra.Command
	for _, sub := range c.Commands() {
		if sub.IsAvailableCommand() || sub.Name() == "help" && !sub.Hidden {
			kids = append(kids, sub)
		}
	}
	w := cmdwidth(kids)
	if !c.HasParent() {
		for _, g := range c.Groups() {
			var in []*cobra.Command
			for _, sub := range kids {
				if sub.GroupID == g.ID {
					in = append(in, sub)
				}
			}
			if len(in) > 0 {
				b.WriteString("\n" + g.Title + "\n")
				cmdlist(&b, in, w)
			}
		}
	} else if len(kids) > 0 {
		b.WriteString("\nCommands:\n")
		cmdlist(&b, kids, w)
	}

	b.WriteString("\nOptions:\n" + flaglines(c))
	if c.Example != "" {
		b.WriteString("\nExamples:\n" + strings.TrimRight(c.Example, "\n") + "\n")
	}
	if len(kids) == 0 || !c.HasParent() {
		exit := c.Annotations["exit"]
		if exit == "" {
			exit = exitdefault
		}
		b.WriteString("\nExit status:\n" + exit + "\n")
	}
	if len(kids) > 0 {
		fmt.Fprintf(&b, "\nRun '%s COMMAND --help' for more information on a command.\n", c.CommandPath())
	}
	if !c.HasParent() {
		b.WriteString("Home page: <" + homepage + ">\n")
	}
	fmt.Fprint(c.OutOrStdout(), b.String())
}

func flaglines(c *cobra.Command) string {
	var list []*pflag.Flag
	add := func(fs *pflag.FlagSet) {
		fs.VisitAll(func(f *pflag.Flag) {
			if !f.Hidden && f.Name != "help" && f.Name != "version" {
				list = append(list, f)
			}
		})
	}
	add(c.LocalFlags())
	add(c.InheritedFlags())
	if f := c.Root().PersistentFlags().Lookup("help"); f != nil {
		list = append(list, f)
	}
	if f := c.Flags().Lookup("version"); f != nil && !c.HasParent() {
		list = append(list, f)
	}

	lefts := make([]string, len(list))
	w := 0
	for i, f := range list {
		left := "      --" + f.Name
		if f.Shorthand != "" {
			left = "  -" + f.Shorthand + ", --" + f.Name
		}
		name, _ := pflag.UnquoteUsage(f)
		if f.Value.Type() != "bool" && name != "" {
			if f.NoOptDefVal != "" {
				left += "[=" + strings.ToUpper(name) + "]"
			} else {
				left += "=" + strings.ToUpper(name)
			}
		}
		lefts[i] = left
		if width(left) <= 28 {
			w = max(w, width(left))
		}
	}
	var b strings.Builder
	for i, f := range list {
		_, usage := pflag.UnquoteUsage(f)
		if width(lefts[i]) > w {
			b.WriteString(lefts[i] + "\n" + strings.Repeat(" ", w+2) + usage + "\n")
			continue
		}
		b.WriteString(pad(lefts[i], w) + "  " + usage + "\n")
	}
	return b.String()
}
