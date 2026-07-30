package cli

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/presentation"
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
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd, nil)
			if changed {
				if err := mlock.WriteAtomic(path, doc); err != nil {
					return err
				}
				return writeStaticOut(cmd, r.Status(presentation.StatusLine{
					Status: presentation.StatusSuccess,
					Text:   "formatted",
					Detail: path,
				}))
			}
			return writeStaticOut(cmd, r.Status(presentation.StatusLine{
				Status: presentation.StatusInfo,
				Text:   "unchanged",
				Detail: path,
			}))
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
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd, nil)
			return writeStaticOut(cmd, lockValidateView(r, result))
		},
	}
	cmd.Flags().BoolVar(&frozen, "frozen", false, "also check manifest specifiers match lock")
	cmd.Flags().BoolVar(&asJSON, "json", false, "report result as JSON")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}

func lockValidateView(r presentation.StaticRenderer, result app.ValidateLockResult) string {
	if result.Detection.Format == "" {
		return r.Status(presentation.StatusLine{
			Status: presentation.StatusSuccess,
			Text:   "ok",
			Detail: result.Path,
		})
	}
	s := presentation.Summary{
		Status: presentation.StatusSuccess,
		Title:  "ok",
		Metrics: []presentation.KeyValue{
			{Key: "path", Value: result.Path, Style: presentation.ValuePath},
			{Key: "format", Value: result.Detection.Format},
			{Key: "major", Value: strconv.Itoa(result.Detection.ProducerMajor), Style: presentation.ValueNumber},
			{Key: "confidence", Value: string(result.Detection.Confidence)},
		},
	}
	if len(result.Detection.Evidence) > 0 {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusInfo,
			Message: "evidence: " + strings.Join(result.Detection.Evidence, ", "),
		})
	}
	return r.Summary(s)
}

func newLockDiffCmd() *cobra.Command {
	var (
		fromPath  string
		toPath    string
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
			opts := app.LockDiffOptions{FromPath: fromPath, ToPath: toPath, PnpmMajor: pnpmMajor}
			if len(args) > 0 {
				opts.OtherPath = args[0]
			}
			return runLockDiff(cmd, ac, opts, asJSON)
		},
	}
	cmd.Flags().StringVar(&fromPath, "from", "", "left lockfile path (requires --to)")
	cmd.Flags().StringVar(&toPath, "to", "", "right lockfile path (requires --from)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit diff as JSON")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	return cmd
}

func newLockMigrateCmd() *cobra.Command {
	var (
		from      string
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
				From: from, DryRun: dryRun, PnpmMajor: pnpmMajor,
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
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd, nil)
			return writeStaticOut(cmd, r.Summary(presentation.Summary{
				Status: presentation.StatusSuccess,
				Title:  "migrated",
				Metrics: []presentation.KeyValue{
					{Key: "path", Value: result.Path, Style: presentation.ValuePath},
					{Key: "from", Value: string(result.SourceIdentity)},
				},
			}))
		},
	}
	cmd.Flags().StringVar(&from, "from", "", "source identity: nub, pnpm, npm, bun, or yarn (optional; auto-detect from packageManager/devEngines or sole lockfile)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "emit loss report without writing m.lock")
	cmd.Flags().IntVar(&pnpmMajor, "pnpm-major", 0, "disambiguate v9-shaped pnpm locks (9, 10, or 11)")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit migration report JSON on success")
	return cmd
}
