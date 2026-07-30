package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

func newRunCmd() *cobra.Command {
	var (
		ifPresent     bool
		wsConcurrency int
		wsOrder       string
		wsOutput      string
		wsBail        = true
		noWsBail      bool
		wsOnlyTouched bool
	)
	cmd := &cobra.Command{
		Use:   "run <script>",
		Short: "Run a package script",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "run", "", "missing app context")
			}
			leading := leadingDispatchFlags{
				ifPresent:     ifPresent,
				wsConcurrency: wsConcurrency,
				wsOrder:       wsOrder,
				wsOutput:      wsOutput,
				wsBail:        wsBail,
				noWsBail:      noWsBail,
				wsOnlyTouched: wsOnlyTouched,
				recursive:     workspaceRecursive(cmd),
			}
			if g := ownerFlags(cmd.Root()); g != nil {
				leading.filter = workspaceFilters(cmd)
			}
			inv, err := BuildScriptInvocation(args[0], args, cmd.ArgsLenAtDash(), leading)
			if err != nil {
				return err
			}
			_, err = app.Run(cmd.Context(), ac, inv.ToRunOptions())
			return err
		},
	}
	cmd.Flags().BoolVar(&ifPresent, "if-present", false, "exit with code 0 when the script is missing")
	cmd.Flags().IntVar(&wsConcurrency, "workspace-concurrency", 0, "max parallel workspace tasks (0 = GOMAXPROCS)")
	cmd.Flags().StringVar(&wsOrder, "workspace-order", "topological", "workspace order: topological|reverse-topological|parallel|sequential")
	cmd.Flags().StringVar(&wsOutput, "workspace-output", "stream", "workspace output: stream|aggregate")
	cmd.Flags().BoolVar(&wsBail, "workspace-bail", true, "stop workspace run on first failure")
	cmd.Flags().BoolVar(&noWsBail, "no-workspace-bail", false, "continue workspace run after failures")
	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		wsOnlyTouched = false
		for _, name := range []string{"workspace-concurrency", "workspace-order", "workspace-output", "workspace-bail", "no-workspace-bail"} {
			if f := cmd.Flags().Lookup(name); f != nil && f.Changed {
				wsOnlyTouched = true
			}
		}
		return nil
	}
	cmd.ValidArgsFunction = scriptNameCompletion
	return cmd
}
