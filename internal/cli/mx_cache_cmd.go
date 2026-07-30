package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
)

func newMXCacheCmd() *cobra.Command {
	cache := &cobra.Command{
		Use:   "cache",
		Short: "Manage mx execution cache",
	}
	cache.AddCommand(newMXCachePruneCmd())
	return cache
}

func newMXCachePruneCmd() *cobra.Command {
	var dryRun bool
	var olderThan string
	cmd := &cobra.Command{
		Use:   "prune",
		Short: "Remove stale mx execution environments",
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil || ac.Config == nil {
				return apperr.New(apperr.Internal, "mx.cache", "", "missing app context")
			}
			opts := app.MXCachePruneOptions{DryRun: dryRun}
			if olderThan != "" {
				d, err := parseDurationFlag(olderThan)
				if err != nil {
					return apperr.Wrap(apperr.Usage, "mx.cache", olderThan, err)
				}
				opts.OlderThan = d
			}
			removed, err := app.PruneMXCache(cmd.Context(), ac, opts)
			if err != nil {
				return err
			}
			for _, c := range removed {
				line := fmt.Sprintf("prune %s last=%s", c.Digest, c.LastUsed.Format(time.RFC3339))
				if dryRun {
					line = "dry-run " + line
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "report candidates without deleting")
	cmd.Flags().StringVar(&olderThan, "older-than", "", "retention override (e.g. 7d)")
	return cmd
}

func parseDurationFlag(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}
	if strings.HasSuffix(s, "d") {
		n := strings.TrimSuffix(s, "d")
		days, err := time.ParseDuration(n + "h")
		if err != nil {
			return 0, err
		}
		return days * 24, nil
	}
	return time.ParseDuration(s)
}

// MXCacheRetentionDays reads runner.mx.cache.retention_days from config.
func MXCacheRetentionDays(eff *config.Effective) int {
	d := config.Int(eff, "runner.mx.cache.retention_days", 7)
	if d <= 0 {
		return 7
	}
	return d
}
