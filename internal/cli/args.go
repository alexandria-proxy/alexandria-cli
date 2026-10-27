package cli

import (
	"bufio"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func nargs(c *cobra.Command, args []string) error {
	if len(args) > 0 {
		return usagef("extra operand '%s'", args[0])
	}
	return nil
}

func minargs(n int) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if len(args) < n {
			if len(args) == 0 {
				return usagef("missing operand")
			}
			return usagef("missing operand after '%s'", args[len(args)-1])
		}
		return nil
	}
}

func maxargs(n int) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if len(args) > n {
			return usagef("extra operand '%s'", args[n])
		}
		return nil
	}
}

func rangeargs(lo, hi int) cobra.PositionalArgs {
	return func(c *cobra.Command, args []string) error {
		if err := minargs(lo)(c, args); err != nil {
			return err
		}
		return maxargs(hi)(c, args)
	}
}

func operands(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		if a != "-" {
			out = append(out, a)
			continue
		}
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line != "" && !strings.HasPrefix(line, "#") {
				out = append(out, line)
			}
		}
		if err := sc.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func verbose(c *cobra.Command) bool {
	v, _ := c.Flags().GetBool("verbose")
	return v
}

func addverbose(c *cobra.Command) {
	c.Flags().BoolP("verbose", "v", false, "explain what is being done")
}
