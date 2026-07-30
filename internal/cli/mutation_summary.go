package cli

import (
	"fmt"
	"strconv"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
)

// mutationSummary builds the install-family StaticRenderer Summary model.
func mutationSummary(result app.InstallResult, dryRun bool) presentation.Summary {
	if dryRun {
		s := presentation.Summary{
			Status: presentation.StatusInfo,
			Title:  "Planned changes",
			Notices: []presentation.Notice{{
				Status:  presentation.StatusInfo,
				Message: "No project files were changed.",
			}},
		}
		s.Metrics = installMetrics(result)
		return s
	}

	if result.Added == 0 && result.Removed == 0 && result.Changed == 0 {
		s := presentation.Summary{
			Status: presentation.StatusSuccess,
			Title:  "Dependencies are already up to date",
		}
		appendCleanupNotices(&s, result)
		return s
	}

	title := formatInstalledTitle(result)
	st := presentation.StatusSuccess
	if result.RecoveryRequired || result.TransactionCleanupIncomplete || result.StoreMaintenanceRequired {
		st = presentation.StatusWarning
	}
	s := presentation.Summary{
		Status:  st,
		Title:   title,
		Metrics: installMetrics(result),
	}
	appendCleanupNotices(&s, result)
	if result.ScriptsBlocked > 0 {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: fmt.Sprintf("%d lifecycle scripts were blocked", result.ScriptsBlocked),
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: "Run `m builds` to review them"})
	}
	return s
}

func formatInstalledTitle(result app.InstallResult) string {
	n := result.Packages
	if n <= 0 {
		n = result.Added + result.Changed
	}
	if result.DurationMs > 0 {
		return fmt.Sprintf("Installed %d packages in %s", n, formatSummaryDuration(result.DurationMs))
	}
	return fmt.Sprintf("Installed %d packages", n)
}

func formatSummaryDuration(ms int64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	sec := float64(ms) / 1000
	if sec < 10 {
		return fmt.Sprintf("%.1fs", sec)
	}
	return fmt.Sprintf("%.0fs", sec)
}

func installMetrics(result app.InstallResult) []presentation.KeyValue {
	rows := []presentation.KeyValue{
		{Key: "Added", Value: strconv.Itoa(result.Added), Style: presentation.ValueNumber},
		{Key: "Updated", Value: strconv.Itoa(result.Changed), Style: presentation.ValueNumber},
		{Key: "Removed", Value: strconv.Itoa(result.Removed), Style: presentation.ValueNumber},
	}
	if result.Reused > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Reused", Value: strconv.Itoa(result.Reused), Style: presentation.ValueNumber})
	}
	if result.Downloaded > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Downloaded", Value: strconv.Itoa(result.Downloaded), Style: presentation.ValueNumber})
	}
	if result.ScriptsRun > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Scripts", Value: strconv.Itoa(result.ScriptsRun), Style: presentation.ValueNumber})
	}
	if result.DurationMs > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Duration", Value: formatSummaryDuration(result.DurationMs), Style: presentation.ValueMuted})
	}
	return rows
}

func appendCleanupNotices(s *presentation.Summary, result app.InstallResult) {
	if result.Committed && (result.TransactionCleanupIncomplete || result.RecoveryRequired) {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Installation committed, but transaction cleanup is incomplete.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: "Run `m recover` to clear stale transaction metadata."})
	}
	if result.RolledBack && (result.TransactionCleanupIncomplete || result.RecoveryRequired) {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Rollback completed with cleanup warnings.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: "Run `m recover` if stale transaction metadata remains."})
	}
	if result.StoreCleanupIncomplete || result.StoreMaintenanceRequired {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Store cleanup is incomplete.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: "Run `m store status` for details."})
	}
}

func rollbackSummary(result app.InstallResult) presentation.Summary {
	s := presentation.Summary{
		Status: presentation.StatusWarning,
		Title:  "Installation failed; project changes were rolled back",
	}
	appendCleanupNotices(&s, result)
	return s
}

func recoveryRequiredSummary() presentation.Summary {
	return presentation.Summary{
		Status: presentation.StatusError,
		Title:  "Installation failed",
		Notices: []presentation.Notice{{
			Status:  presentation.StatusError,
			Message: "Recovery is required before the next mutation.",
		}},
		Hints: []presentation.Hint{{Message: "Run `m recover`"}},
	}
}
