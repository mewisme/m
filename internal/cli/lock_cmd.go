package cli

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/lockfile/mlock"
)

func newLockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "lock",
		Short: "Lockfile utilities",
	}
	cmd.AddCommand(newLockFormatCmd())
	cmd.AddCommand(newLockValidateCmd())
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
			path := app.LockPath(proj.Root)
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
		frozen bool
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate m.lock parse, checksum, and graph",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "lock.validate", "", "missing app context")
			}
			proj, err := app.OpenProject(cmd.Context(), ac)
			if err != nil {
				return err
			}
			path := app.LockPath(proj.Root)
			data, err := os.ReadFile(path)
			if err != nil {
				return apperr.Wrap(apperr.IO, "lock.validate", path, err)
			}
			doc, err := mlock.Decode(data)
			if err != nil {
				return err
			}
			if _, err := mlock.ToGraph(doc); err != nil {
				return err
			}
			if frozen {
				if err := app.ValidateFrozenLock(cmd.Context(), ac); err != nil {
					return err
				}
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				return enc.Encode(map[string]any{"ok": true, "path": path, "frozen": frozen})
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), "ok", path)
			return err
		},
	}
	cmd.Flags().BoolVar(&frozen, "frozen", false, "also check manifest specifiers match lock")
	cmd.Flags().BoolVar(&asJSON, "json", false, "report result as JSON")
	return cmd
}
