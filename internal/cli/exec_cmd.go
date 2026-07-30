package cli

import (
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
)

type execParsed struct {
	snapshotID  string
	capsulePath string
	packageName string
	command     string
	forwarded   []string
	filters     []string
	recursive   bool
}

func parseExecArgs(cmd *cobra.Command, args []string) (execParsed, error) {
	var out execParsed
	if g := ownerFlags(cmd.Root()); g != nil {
		out.filters = workspaceFilters(cmd)
		out.recursive = workspaceRecursive(cmd)
	}
	i := 0
	for i < len(args) {
		arg := args[i]
		switch {
		case arg == "--":
			out.forwarded = append(out.forwarded, args[i+1:]...)
			return out, nil
		case arg == "--snapshot":
			if out.capsulePath != "" {
				return out, apperr.New(apperr.Usage, "exec", "", "snapshot and capsule are mutually exclusive")
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return out, apperr.New(apperr.Usage, "exec", "", "missing snapshot id")
			}
			out.snapshotID = args[i+1]
			i += 2
		case arg == "--capsule":
			if out.snapshotID != "" {
				return out, apperr.New(apperr.Usage, "exec", "", "snapshot and capsule are mutually exclusive")
			}
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") {
				return out, apperr.New(apperr.Usage, "exec", "", "missing capsule path")
			}
			out.capsulePath = args[i+1]
			i += 2
		case arg == "--package" || strings.HasPrefix(arg, "--package="):
			if out.snapshotID != "" || out.capsulePath != "" {
				return out, apperr.New(apperr.Usage, "exec", "", "source flags cannot combine with --package")
			}
			if strings.HasPrefix(arg, "--package=") {
				out.packageName = strings.TrimPrefix(arg, "--package=")
				i++
				continue
			}
			if i+1 >= len(args) {
				return out, apperr.New(apperr.Usage, "exec", "", "missing --package value")
			}
			out.packageName = args[i+1]
			i += 2
		case strings.HasPrefix(arg, "-"):
			return out, apperr.New(apperr.Usage, "exec", arg, "unknown exec flag after source selection")
		default:
			out.command = arg
			if cmd.ArgsLenAtDash() >= 0 && i >= cmd.ArgsLenAtDash() {
				out.forwarded = append(out.forwarded, args[i+1:]...)
			} else if dash := indexArgsDash(args, i); dash >= 0 {
				out.forwarded = append(out.forwarded, args[dash+1:]...)
			}
			if out.command == "" {
				return out, apperr.New(apperr.Usage, "exec", "", "missing command selector")
			}
			return out, nil
		}
	}
	if out.command == "" {
		return out, apperr.New(apperr.Usage, "exec", "", "missing command selector")
	}
	return out, nil
}

func indexArgsDash(args []string, from int) int {
	for i := from; i < len(args); i++ {
		if args[i] == "--" {
			return i
		}
	}
	return -1
}

func newExecCmd() *cobra.Command {
	var pkg string
	cmd := &cobra.Command{
		Use:   "exec <binary>",
		Short: "Execute a local package binary",
		Args:  cobra.MinimumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "exec", "", "missing app context")
			}
			parsed, err := parseExecArgs(cmd, args)
			if err != nil {
				return err
			}
			if parsed.packageName == "" {
				parsed.packageName = pkg
			}
			if g := ownerFlags(cmd.Root()); g != nil && g.ctrl != nil {
				g.ctrl.SetRunnerCommand(parsed.command)
			}
			_, err = app.Exec(cmd.Context(), ac, app.ExecOptions{
				Command:       parsed.command,
				PackageFilter: parsed.packageName,
				ForwardedArgs: parsed.forwarded,
				Filters:       parsed.filters,
				Recursive:     parsed.recursive,
				SnapshotID:    parsed.snapshotID,
				CapsulePath:   parsed.capsulePath,
			})
			return err
		},
	}
	cmd.Flags().StringVar(&pkg, "package", "", "importer-visible dependency providing the binary")
	cmd.ValidArgsFunction = execCompletion
	return cmd
}
