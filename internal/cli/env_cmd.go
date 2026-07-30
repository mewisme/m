package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
)

func newEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "env",
		Short: "Inspect execution environments",
	}
	cmd.AddCommand(newEnvInspectCmd())
	return cmd
}

func newEnvInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Plan-only environment inspection",
	}
	cmd.AddCommand(newEnvInspectProjectCmd())
	cmd.AddCommand(newEnvInspectDLXCmd())
	cmd.AddCommand(newEnvInspectSnapshotCmd())
	cmd.AddCommand(newEnvInspectCapsuleCmd())
	return cmd
}

func newEnvInspectProjectCmd() *cobra.Command {
	var projectDir, pkg string
	cmd := &cobra.Command{
		Use:   "project [<binary>]",
		Short: "Inspect the local project environment",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "env.inspect", "", "missing app context")
			}
			cwd := projectDir
			if cwd == "" {
				cwd = ac.CWD
			}
			command := ""
			if len(args) > 0 {
				command = args[0]
			}
			b, err := app.InspectEnvironmentJSON(cmd.Context(), ac, app.EnvInspectOptions{
				SourceKind: envexec.SourceProject,
				CWD:        cwd,
				Package:    pkg,
				Command:    command,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(b))
			return err
		},
	}
	cmd.Flags().StringVar(&projectDir, "project-dir", "", "project working directory")
	cmd.Flags().StringVar(&pkg, "package", "", "dependency owner filter")
	return cmd
}

func newEnvInspectDLXCmd() *cobra.Command {
	var packages []string
	cmd := &cobra.Command{
		Use:   "dlx <binary>",
		Short: "Inspect a DLX environment plan",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "env.inspect", "", "missing app context")
			}
			specs, err := parseDLXSpecs(packages)
			if err != nil {
				return err
			}
			b, err := app.InspectEnvironmentJSON(cmd.Context(), ac, app.EnvInspectOptions{
				SourceKind: envexec.SourceDLX,
				DLXSpecs:   specs,
				DLXMode:    envexec.DLXModeExplicitPackages,
				Command:    args[0],
				Offline:    true,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(b))
			return err
		},
	}
	cmd.Flags().StringArrayVarP(&packages, "package", "p", nil, "package specs")
	return cmd
}

func newEnvInspectSnapshotCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "snapshot <id> [<binary>]",
		Short: "Inspect a snapshot execution environment",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "env.inspect", "", "missing app context")
			}
			command := ""
			if len(args) > 1 {
				command = args[1]
			}
			b, err := app.InspectEnvironmentJSON(cmd.Context(), ac, app.EnvInspectOptions{
				SourceKind: envexec.SourceSnapshot,
				SnapshotID: args[0],
				Command:    command,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(b))
			return err
		},
	}
	return cmd
}

func newEnvInspectCapsuleCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "capsule <path> [<binary>]",
		Short: "Inspect a capsule execution environment",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "env.inspect", "", "missing app context")
			}
			command := ""
			if len(args) > 1 {
				command = args[1]
			}
			b, err := app.InspectEnvironmentJSON(cmd.Context(), ac, app.EnvInspectOptions{
				SourceKind:  envexec.SourceCapsule,
				CapsulePath: args[0],
				Command:     command,
			})
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), string(b))
			return err
		},
	}
	return cmd
}

func parseDLXSpecs(raw []string) ([]dlx.PackageSpec, error) {
	if len(raw) == 0 {
		return nil, apperr.New(apperr.Usage, "env.inspect", "", "missing package spec")
	}
	out := make([]dlx.PackageSpec, 0, len(raw))
	for _, s := range raw {
		spec, err := dlx.ParsePackageSpec(s)
		if err != nil {
			return nil, err
		}
		out = append(out, spec)
	}
	return out, nil
}

var _ = json.RawMessage{}
