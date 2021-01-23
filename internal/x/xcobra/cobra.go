// Package xcobra wraps the cobra package to provide richer functionality.
package xcobra

import (
	"fmt"
	"strings"

	"cdr.dev/coder-cli/pkg/clog"
	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

// ExactArgs returns an error if there are not exactly n args.
func ExactArgs(n int, names ...string) cobra.PositionalArgs {
	wrap := func(name string) string { return fmt.Sprintf("[%s]", name) }
	return func(cmd *cobra.Command, args []string) error {
		wrappedNames := make([]string, 0, len(names))
		if len(args) != n {
			for _, name := range names {
				wrappedNames = append(wrappedNames, wrap(name))
			}
			baseMsg := fmt.Sprintf("accepts %d arg(s), received %d", n, len(args))
			if len(names) == 0 {
				return clog.Error(baseMsg)
			}
			return clog.Error(
				baseMsg,
				color.New(color.Bold).Sprintf("args: ")+strings.Join(wrappedNames, " "),
				clog.BlankLine,
				clog.Tipf("use \"--help\" for more info"),
			)
		}
		return nil
	}
}
