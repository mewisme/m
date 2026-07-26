package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/apperr"
)

func registerStubs(root *cobra.Command) {
	for _, st := range stubCommands {
		st := st
		cmd := &cobra.Command{
			Use:     st.Use,
			Short:   st.Short,
			Aliases: st.Aliases,
			Args:    cobra.ArbitraryArgs,
			RunE: func(cmd *cobra.Command, args []string) error {
				return apperr.New(apperr.Unimplemented, "cli", st.Use,
					fmt.Sprintf("not implemented (MVP %s)", st.MVP))
			},
		}
		root.AddCommand(cmd)
	}
}

func newDispatchCmd(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:    "__dispatch",
		Short:  "Show effective command resolution (diagnostic)",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			kind, path := resolveDispatch(root, name)
			_, err := fmt.Fprintf(cmd.OutOrStdout(), "kind=%s path=%s\n", kind, path)
			return err
		},
	}
}

func resolveDispatch(root *cobra.Command, name string) (kind, path string) {
	for _, c := range root.Commands() {
		if c.Name() == name {
			return "builtin", c.Name()
		}
		for _, a := range c.Aliases {
			if a == name {
				return "alias", c.Name()
			}
		}
	}
	if IsReserved(name) {
		return "builtin", name
	}
	return "unknown", name
}
