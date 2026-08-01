package cli

import (
	"fmt"
	"strings"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
)

// fixBin replaces backtick-quoted `m ` with the actual binary name in hint messages.
func fixBin(msg, bin string) string {
	if bin == "" || bin == "m" {
		return msg
	}
	msg = strings.ReplaceAll(msg, "`m ", "`"+bin+" ")
	msg = strings.ReplaceAll(msg, "`m<", "`"+bin+" <")
	return msg
}

// mutationSummary builds the install-family StaticRenderer Summary model.
func mutationSummary(result app.InstallResult, dryRun bool, binName string) presentation.Summary {
	if binName == "" {
		binName = "m"
	}
	if dryRun {
		s := presentation.Summary{
			Status: presentation.StatusInfo,
			Title:  "Planned changes",
			Notices: []presentation.Notice{{
				Status:  presentation.StatusInfo,
				Message: "No project files were changed.",
			}},
		}
		s.Deltas = mutationPackageDeltas(result.PackageChanges)
		s.Metrics = mutationCompletionMetrics(result)
		return s
	}

	if result.Added == 0 && result.Removed == 0 && result.Changed == 0 {
		s := presentation.Summary{
			Status: presentation.StatusSuccess,
			Title:  "Already up to date",
		}
		if result.DurationMs > 0 {
			s.Title += " [" + presentation.FormatDuration(result.DurationMs) + "]"
		}
		appendCleanupNotices(&s, result, binName)
		return s
	}

	title := presentation.FormatMutationCompletion(result.Added, result.Changed, result.Removed, result.DurationMs)
	st := presentation.StatusSuccess
	if result.RecoveryRequired || result.TransactionCleanupIncomplete || result.StoreMaintenanceRequired {
		st = presentation.StatusWarning
	}
	deltas := result.DirectPackageChanges
	if len(deltas) == 0 {
		deltas = result.PackageChanges
	}
	s := presentation.Summary{
		Status:  st,
		Title:   title,
		Deltas:  mutationPackageDeltas(deltas),
		Metrics: mutationCompletionMetrics(result),
	}
	appendCleanupNotices(&s, result, binName)
	if result.ScriptsBlocked > 0 {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: fmt.Sprintf("%d lifecycle scripts were blocked", result.ScriptsBlocked),
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: fixBin("Run `m builds` to review them", binName)})
	}
	return s
}

// mutationCompletionMetrics returns secondary metrics for the compact summary.
func mutationCompletionMetrics(result app.InstallResult) []presentation.KeyValue {
	var rows []presentation.KeyValue
	if result.Reused > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Reused", Value: fmt.Sprintf("%d", result.Reused), Style: presentation.ValueNumber})
	}
	if result.Downloaded > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Downloaded", Value: fmt.Sprintf("%d", result.Downloaded), Style: presentation.ValueNumber})
	}
	if result.ScriptsRun > 0 {
		rows = append(rows, presentation.KeyValue{Key: "Scripts", Value: fmt.Sprintf("%d", result.ScriptsRun), Style: presentation.ValueNumber})
	}
	return rows
}

func appendCleanupNotices(s *presentation.Summary, result app.InstallResult, binName string) {
	if result.Committed && (result.TransactionCleanupIncomplete || result.RecoveryRequired) {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Installation committed, but transaction cleanup is incomplete.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: fixBin("Run `m recover` to clear stale transaction metadata.", binName)})
	}
	if result.RolledBack && (result.TransactionCleanupIncomplete || result.RecoveryRequired) {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Rollback completed with cleanup warnings.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: fixBin("Run `m recover` if stale transaction metadata remains.", binName)})
	}
	if result.StoreCleanupIncomplete || result.StoreMaintenanceRequired {
		s.Notices = append(s.Notices, presentation.Notice{
			Status:  presentation.StatusWarning,
			Message: "Store cleanup is incomplete.",
		})
		s.Hints = append(s.Hints, presentation.Hint{Message: fixBin("Run `m store status` for details.", binName)})
	}
}

func rollbackSummary(result app.InstallResult, binName string) presentation.Summary {
	s := presentation.Summary{
		Status: presentation.StatusWarning,
		Title:  "Installation failed; project changes were rolled back",
	}
	appendCleanupNotices(&s, result, binName)
	return s
}

func recoveryRequiredSummary(binName string) presentation.Summary {
	return presentation.Summary{
		Status: presentation.StatusError,
		Title:  "Installation failed",
		Notices: []presentation.Notice{{
			Status:  presentation.StatusError,
			Message: "Recovery is required before the next mutation.",
		}},
		Hints: []presentation.Hint{{Message: fixBin("Run `m recover`", binName)}},
	}
}

// mutationPackageDeltas maps app.PackageChange values into presentation.PackageDelta rows.
func mutationPackageDeltas(changes []app.PackageChange) []presentation.PackageDelta {
	if len(changes) == 0 {
		return nil
	}
	out := make([]presentation.PackageDelta, 0, len(changes))
	for _, c := range changes {
		d := presentation.PackageDelta{
			Name: c.Name,
		}
		switch c.Kind {
		case app.PackageChangeAdded:
			d.Kind = presentation.DeltaAdded
			d.Version = c.ToVersion
		case app.PackageChangeUpdated:
			d.Kind = presentation.DeltaUpdated
			d.From = c.FromVersion
			d.To = c.ToVersion
		case app.PackageChangeRemoved:
			d.Kind = presentation.DeltaRemoved
			d.Version = c.FromVersion
		}
		out = append(out, d)
	}
	return out
}
