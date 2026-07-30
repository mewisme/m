package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
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
			selector := args[0]
			g := ownerFlags(root)
			phase := PhaseAResult{Selector: selector}
			cwd := dispatchCWD(g, phase)
			var eff *config.Effective
			if ac := app.FromContext(cmd.Context()); ac != nil {
				cwd = ac.CWD
				eff = ac.Config
			}
			res := ResolveDispatch(root, phase, cwd, eff)
			raw, err := encodeDispatchJSON(res, selector)
			if err != nil {
				return apperr.Wrap(apperr.Internal, "__dispatch", selector, err)
			}
			_, err = cmd.OutOrStdout().Write(append(raw, '\n'))
			return err
		},
	}
}
