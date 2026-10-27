package cli

import (
	"strconv"

	"github.com/spf13/cobra"

	"github.com/alexandria-proxy/alexandria-cli/internal/route"
	"github.com/alexandria-proxy/alexandria-cli/internal/sub"
)

const nofile = cobra.ShellCompDirectiveNoFileComp

func completeservers(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	list, _ := sub.Load()
	if best, _ := c.Flags().GetBool("best"); best {
		return subcompletions(list), nofile
	}
	if len(args) > 0 {
		return nil, nofile
	}
	var out []cobra.Completion
	for _, e := range number(list) {
		out = append(out, cobra.CompletionWithDesc(strconv.Itoa(e.num), e.srv.Name))
	}
	return out, nofile
}

func completesubs(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	list, _ := sub.Load()
	return subcompletions(list), nofile
}

func subcompletions(list []sub.Subscription) []cobra.Completion {
	var out []cobra.Completion
	for i, s := range list {
		out = append(out, cobra.CompletionWithDesc(strconv.Itoa(i+1), s.Name))
	}
	return out
}

func completekeys(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	var out []cobra.Completion
	for _, k := range confkeys {
		out = append(out, cobra.CompletionWithDesc(k.name, k.help))
	}
	return out, nofile
}

func completeset(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return completekeys(c, args, "")
	case 1:
		if k := findkey(args[0]); k != nil {
			return k.choices, nofile
		}
	}
	return nil, nofile
}

func completeroutes(c *cobra.Command, args []string, _ string) ([]cobra.Completion, cobra.ShellCompDirective) {
	entries, _, _ := route.Load()
	return entries, nofile
}
