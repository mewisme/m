package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
)

func newLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lockfile utilities",
	}
	cmd.AddCommand(newLockFormatCmd())
	cmd.AddCommand(newLockValidateCmd())
	cmd.AddCommand(newLockDiffCmd())
	cmd.AddCommand(newLockMigrateCmd())
	return cmd
}

func newLockFormatCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "format",
		Short: "Canonicalize m.lock in the current project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "lock.format", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			if proj.Identity != project.IdentityMew {
				return apperr.New(apperr.Usage, "lock.format", string(proj.Identity), "format applies to m.lock projects only")
			}
			path := app.LockPath(proj)
			before, err := os.ReadFile(path)
			if err != nil {
				return apperr.Wrap(apperr.IO, "lock.format", path, err)
			}
			doc, err := mlock.Decode(before)
			if err != nil {
				return err
			}
			after, err := mlock.Encode(doc)
			if err != nil {
				return err
			}
			changed := string(before) != string(after)
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{"changed": changed, "path": path})
			}
			if changed {
				if err := mlock.WriteAtomic(path, doc); err != nil {
					return err
				}
				_, err = fmt.Fprintln(cmd.OutOrStdout(), "formatted", path)
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "unchanged", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "report changed/unchanged as JSON")
	return cmd
}

func newLockValidateCmd() *cobra.Command {
	var (
		frozen    bool
		asJSON    bool
		pnpmMajor int
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate incumbent lock parse and graph",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "lock.validate", "", "missing app context")
			}
			result, err := app.ValidateIncumbentLock(cmd.Context(), ac, app.ValidateLockOptions{
				Frozen: frozen, PnpmMajor: pnpmMajor,
			})
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{
					"ok": true, "path": result.Path, "frozen": frozen, "detection": result.Detection,
				})
			}
			if result.Detection.Format != "" {
				_, err = fmt.Fprintf(cmd.OutOrStdout(), "ok %s format=%s major=%d confidence=%s\n",
					result.Path, result.Detection.Format, result.Detection.ProducerMajor, result.Detection.Confidence)
				if len(result.Detection.Evidence) > 0 {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "evidence: %s\n", strings.Join(result.Detection.Evidence, ", "))
				}
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok", result.Path)
			return err
		},
	}
	cmd.Flags().BoolVar(&frozen, "frozen", false, "also check manifest specifiers match lock")
	cmd.Flags().BoolVar(&asJSON, "json", false, "report result as JSON")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}

func newLockDiffCmd() *cobra.Command {
	var (
		asJSON    bool
		pnpmMajor int
	)
	cmd := &cobra.Command{
		Use:   "diff [other]",
		Short: "Diff incumbent lock graph against another lockfile",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "lock.diff", "", "missing app context")
			}
			other := ""
			if len(args) > 0 {
				other = args[0]
			}
			diff, err := app.LockDiff(cmd.Context(), ac, app.LockDiffOptions{OtherPath: other, PnpmMajor: pnpmMajor})
			if err != nil {
				return err
			}
			data, err := lockfile.EncodeDiffJSON(diff)
			if err != nil {
				return err
			}
			if asJSON {
				_, err = cmd.OutOrStdout().Write(data)
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return err
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "alias for JSON diff output")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}

func newLockMigrateCmd() *cobra.Command {
	var (
		from      string
		to        string
		dryRun    bool
		pnpmMajor int
		asJSON    bool
	)
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Migrate nub, pnpm, or npm lock to m.lock",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "lock.migrate", "", "missing app context")
			}
			result, err := app.MigrateLock(cmd.Context(), ac, app.MigrateLockOptions{
				From: from, To: to, DryRun: dryRun, PnpmMajor: pnpmMajor,
			})
			if err != nil {
				return err
			}
			if dryRun || asJSON {
				payload := map[string]any{
					"dryRun":           result.DryRun,
					"path":             result.Path,
					"sourceIdentity":   result.SourceIdentity,
					"sourceLockPath":   result.SourceLockPath,
					"detection":        result.Detection,
					"preservedUnknown": result.PreservedUnknown,
					"lossReport":       result.LossReport,
				}
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(payload)
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "migrated", result.Path)
			return err
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source identity: nub, pnpm, or npm (default: project identity)")
	cmd.Flags().StringVar(&to, "to", "m", "target format (only m is supported)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "emit loss report without writing m.lock")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit migration report JSON on success")
	return cmd
}
