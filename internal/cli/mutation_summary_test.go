package cli

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
)

func TestMutationSummaryInstallWithAdditions(t *testing.T) {
	result := app.InstallResult{
		Added:    2,
		Packages: 2,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeAdded, Name: "zod", ToVersion: "4.0.14", ToKey: "zod@4.0.14"},
			{Kind: app.PackageChangeAdded, Name: "vite", ToVersion: "7.0.4", ToKey: "vite@7.0.4"},
		},
	}
	s := mutationSummary(result, false)
	if s.Status != presentation.StatusSuccess {
		t.Fatalf("status=%v", s.Status)
	}
	if !strings.Contains(s.Title, "Installed") {
		t.Fatalf("title=%q", s.Title)
	}
	if len(s.Deltas) != 2 {
		t.Fatalf("deltas=%d want 2", len(s.Deltas))
	}
	for _, d := range s.Deltas {
		if d.Kind != presentation.DeltaAdded {
			t.Fatalf("delta kind=%v want added", d.Kind)
		}
	}
	if len(s.Metrics) < 3 {
		t.Fatalf("metrics=%d", len(s.Metrics))
	}
}

func TestMutationSummaryRemove(t *testing.T) {
	result := app.InstallResult{
		Removed:  1,
		Packages: 1,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeRemoved, Name: "lodash", FromVersion: "4.17.20", FromKey: "lodash@4.17.20"},
		},
	}
	s := mutationSummary(result, false)
	if s.Status != presentation.StatusSuccess {
		t.Fatalf("status=%v", s.Status)
	}
	if len(s.Deltas) != 1 {
		t.Fatalf("deltas=%d", len(s.Deltas))
	}
	d := s.Deltas[0]
	if d.Kind != presentation.DeltaRemoved || d.Name != "lodash" || d.Version != "4.17.20" {
		t.Fatalf("delta=%+v", d)
	}
}

func TestMutationSummaryUpdate(t *testing.T) {
	result := app.InstallResult{
		Changed:  1,
		Packages: 1,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeUpdated, Name: "react", FromVersion: "19.1.0", ToVersion: "19.1.1", FromKey: "react@19.1.0", ToKey: "react@19.1.1"},
		},
	}
	s := mutationSummary(result, false)
	if len(s.Deltas) != 1 {
		t.Fatalf("deltas=%d", len(s.Deltas))
	}
	d := s.Deltas[0]
	if d.Kind != presentation.DeltaUpdated || d.Name != "react" || d.From != "19.1.0" || d.To != "19.1.1" {
		t.Fatalf("delta=%+v", d)
	}
}

func TestMutationSummaryMixedAddUpdateRemove(t *testing.T) {
	result := app.InstallResult{
		Added:   1,
		Changed: 1,
		Removed: 1,
		Packages: 3,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeAdded, Name: "new-pkg", ToVersion: "1.0.0", ToKey: "new-pkg@1.0.0"},
			{Kind: app.PackageChangeUpdated, Name: "react", FromVersion: "19.1.0", ToVersion: "19.1.1", FromKey: "react@19.1.0", ToKey: "react@19.1.1"},
			{Kind: app.PackageChangeRemoved, Name: "old-pkg", FromVersion: "2.0.0", FromKey: "old-pkg@2.0.0"},
		},
	}
	s := mutationSummary(result, false)
	if len(s.Deltas) != 3 {
		t.Fatalf("deltas=%d", len(s.Deltas))
	}
}

func TestMutationSummaryDryRun(t *testing.T) {
	result := app.InstallResult{
		Added:    2,
		Packages: 2,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeAdded, Name: "zod", ToVersion: "4.0.14", ToKey: "zod@4.0.14"},
		},
	}
	s := mutationSummary(result, true)
	if s.Status != presentation.StatusInfo {
		t.Fatalf("status=%v want info", s.Status)
	}
	if s.Title != "Planned changes" {
		t.Fatalf("title=%q", s.Title)
	}
	if len(s.Deltas) != 1 {
		t.Fatalf("deltas=%d want 1", len(s.Deltas))
	}
	if len(s.Metrics) == 0 {
		t.Fatalf("dry-run should include metrics")
	}
}

func TestMutationSummaryAlreadyUpToDate(t *testing.T) {
	result := app.InstallResult{Added: 0, Removed: 0, Changed: 0, Packages: 126}
	s := mutationSummary(result, false)
	if s.Title != "Dependencies are already up to date" {
		t.Fatalf("title=%q", s.Title)
	}
	if len(s.Deltas) != 0 {
		t.Fatalf("deltas=%d want 0", len(s.Deltas))
	}
}

func TestMutationSummaryScriptsBlocked(t *testing.T) {
	result := app.InstallResult{
		Added:          1,
		Packages:       1,
		ScriptsBlocked: 3,
		PackageChanges: []app.PackageChange{
			{Kind: app.PackageChangeAdded, Name: "pkg", ToVersion: "1.0.0", ToKey: "pkg@1.0.0"},
		},
	}
	s := mutationSummary(result, false)
	if len(s.Notices) == 0 {
		t.Fatalf("expected scripts blocked notice")
	}
	found := false
	for _, n := range s.Notices {
		if strings.Contains(n.Message, "lifecycle scripts were blocked") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("notices=%v", s.Notices)
	}
	if len(s.Hints) == 0 {
		t.Fatalf("expected hint")
	}
}

func TestMutationSummaryEmptyChanges(t *testing.T) {
	changes := []app.PackageChange{}
	deltas := mutationPackageDeltas(changes)
	if len(deltas) != 0 {
		t.Fatalf("deltas=%d", len(deltas))
	}
}

func TestMutationPackageDeltasMapping(t *testing.T) {
	changes := []app.PackageChange{
		{Kind: app.PackageChangeAdded, Name: "a", ToVersion: "1.0.0", ToKey: "a@1.0.0"},
		{Kind: app.PackageChangeUpdated, Name: "b", FromVersion: "1.0.0", ToVersion: "2.0.0", FromKey: "b@1.0.0", ToKey: "b@2.0.0"},
		{Kind: app.PackageChangeRemoved, Name: "c", FromVersion: "3.0.0", FromKey: "c@3.0.0"},
	}
	deltas := mutationPackageDeltas(changes)
	if len(deltas) != 3 {
		t.Fatalf("deltas=%d", len(deltas))
	}
	if deltas[0].Kind != presentation.DeltaAdded || deltas[0].Name != "a" || deltas[0].Version != "1.0.0" {
		t.Fatalf("delta[0]=%+v", deltas[0])
	}
	if deltas[1].Kind != presentation.DeltaUpdated || deltas[1].From != "1.0.0" || deltas[1].To != "2.0.0" {
		t.Fatalf("delta[1]=%+v", deltas[1])
	}
	if deltas[2].Kind != presentation.DeltaRemoved || deltas[2].Name != "c" || deltas[2].Version != "3.0.0" {
		t.Fatalf("delta[2]=%+v", deltas[2])
	}
}
