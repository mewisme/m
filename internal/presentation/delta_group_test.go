package presentation_test

import (
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/presentation"
)

func AS(deltas ...presentation.PackageDelta) []presentation.PackageDelta { return deltas }
func A(name, ver string) presentation.PackageDelta {
	return presentation.PackageDelta{Kind: presentation.DeltaAdded, Name: name, Version: ver}
}
func U(name, from, to string) presentation.PackageDelta {
	return presentation.PackageDelta{Kind: presentation.DeltaUpdated, Name: name, From: from, To: to}
}
func R(name, ver string) presentation.PackageDelta {
	return presentation.PackageDelta{Kind: presentation.DeltaRemoved, Name: name, Version: ver}
}

func TestGroupedPackageDeltasAllGroups(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(
		A("zod", "4.0.14"),
		A("vite", "7.0.4"),
		U("react", "19.1.0", "19.1.1"),
		R("lodash", "4.17.20"),
	))
	if !strings.Contains(out, "Added") {
		t.Fatalf("missing Added heading:\n%s", out)
	}
	if !strings.Contains(out, "Updated") {
		t.Fatalf("missing Updated heading:\n%s", out)
	}
	if !strings.Contains(out, "Removed") {
		t.Fatalf("missing Removed heading:\n%s", out)
	}
	if strings.ContainsAny(out, "\x1b") {
		t.Fatalf("ANSI in plain output:\n%s", out)
	}
}

func TestGroupedPackageDeltasEmptyGroupsOmitted(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(A("zod", "4.0.14")))
	if !strings.Contains(out, "Added") {
		t.Fatalf("missing Added:\n%s", out)
	}
	if strings.Contains(out, "Updated") {
		t.Fatalf("empty Updated should be omitted:\n%s", out)
	}
	if strings.Contains(out, "Removed") {
		t.Fatalf("empty Removed should be omitted:\n%s", out)
	}
}

func TestGroupedPackageDeltasOnlyUpdated(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(U("react", "19.1.0", "19.1.1")))
	if strings.Contains(out, "Added") || strings.Contains(out, "Removed") {
		t.Fatalf("unexpected group:\n%s", out)
	}
	if !strings.Contains(out, "Updated") {
		t.Fatalf("missing Updated:\n%s", out)
	}
}

func TestGroupedPackageDeltasOnlyRemoved(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(R("lodash", "4.17.20")))
	if strings.Contains(out, "Added") || strings.Contains(out, "Updated") {
		t.Fatalf("unexpected group:\n%s", out)
	}
	if !strings.Contains(out, "Removed") {
		t.Fatalf("missing Removed:\n%s", out)
	}
}

func TestGroupedPackageDeltasDeterministicGroupOrder(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	for i := 0; i < 10; i++ {
		out := r.PackageDeltas(AS(
			R("z", "1.0.0"),
			A("a", "1.0.0"),
			U("m", "1.0.0", "2.0.0"),
		))
		addedIdx := strings.Index(out, "Added")
		updatedIdx := strings.Index(out, "Updated")
		removedIdx := strings.Index(out, "Removed")
		if addedIdx < 0 || updatedIdx < 0 || removedIdx < 0 {
			t.Fatalf("run %d: missing group:\n%s", i, out)
		}
		if !(addedIdx < updatedIdx && updatedIdx < removedIdx) {
			t.Fatalf("run %d: group order wrong: added=%d updated=%d removed=%d\n%s", i, addedIdx, updatedIdx, removedIdx, out)
		}
	}
}

func TestGroupedPackageDeltasUnicodeSymbols(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: true,
		Width:      80,
		Symbols:    presentation.UnicodeSymbols,
	})
	out := r.PackageDeltas(AS(
		A("zod", "4.0.14"),
		U("react", "19.1.0", "19.1.1"),
		R("lodash", "4.17.20"),
	))
	if !strings.Contains(out, "+ zod") {
		t.Fatalf("missing addition:\n%s", out)
	}
	if !strings.Contains(out, "~ react") {
		t.Fatalf("missing update:\n%s", out)
	}
	if !strings.Contains(out, "- lodash") {
		t.Fatalf("missing removal:\n%s", out)
	}
	if strings.ContainsAny(out, "\x1b") {
		t.Fatalf("ANSI in plain unicode output:\n%s", out)
	}
}

func TestGroupedPackageDeltasASCIIFallback(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(
		A("zod", "4.0.14"),
		R("lodash", "4.17.20"),
	))
	if !strings.Contains(out, "+ zod") {
		t.Fatalf("missing ascii addition:\n%s", out)
	}
	if !strings.Contains(out, "- lodash") {
		t.Fatalf("missing ascii removal:\n%s", out)
	}
	if strings.ContainsAny(out, "\x1b") {
		t.Fatalf("ANSI in ascii output:\n%s", out)
	}
}

func TestGroupedPackageDeltasNoTrailingWhitespace(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(AS(A("zod", "4.0.14")))
	for _, line := range strings.Split(out, "\n") {
		if strings.TrimSpace(line) != line {
			t.Fatalf("line %q has leading/trailing whitespace", line)
		}
	}
	if strings.HasPrefix(out, "\n") || strings.HasSuffix(out, "\n") {
		t.Fatalf("leading or trailing blank line:\n%q", out)
	}
}

func TestGroupedPackageDeltasRichColor(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		UseColor:   true,
		UseUnicode: true,
		ThemeMode:  presentation.ThemeDark,
		Width:      80,
		Symbols:    presentation.UnicodeSymbols,
	})
	out := r.PackageDeltas(AS(A("zod", "4.0.14")))
	if !strings.Contains(out, "\x1b") {
		t.Fatalf("expected ANSI in rich output:\n%s", out)
	}
}

func TestGroupedPackageDeltasTruncationNotice(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	deltas := make([]presentation.PackageDelta, 60)
	for i := 0; i < 60; i++ {
		deltas[i] = A("pkg", "1.0.0")
	}
	out := r.PackageDeltas(deltas)
	if !strings.Contains(out, "10 additional package changes are not shown") {
		t.Fatalf("missing truncation notice:\n%s", out)
	}
	if !strings.Contains(out, "m plan") {
		t.Fatalf("missing plan hint:\n%s", out)
	}
	// Verify at most 50 rows rendered (plus headings and notice).
	if strings.Count(out, "\n") > 60 {
		t.Fatalf("too many lines:\n%s", out)
	}
}

func TestGroupedPackageDeltasNoTruncationUnderLimit(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	deltas := make([]presentation.PackageDelta, 50)
	for i := 0; i < 50; i++ {
		deltas[i] = A("pkg", "1.0.0")
	}
	out := r.PackageDeltas(deltas)
	if strings.Contains(out, "not shown") {
		t.Fatalf("truncation notice at limit:\n%s", out)
	}
}

func TestGroupedPackageDeltasEmpty(t *testing.T) {
	r := presentation.NewStaticRenderer(presentation.EffectiveSettings{
		ThemeMode:  presentation.ThemeNone,
		UseUnicode: false,
		Width:      80,
		Symbols:    presentation.ASCIISymbols,
	})
	out := r.PackageDeltas(nil)
	if out != "" {
		t.Fatalf("expected empty string, got %q", out)
	}
}
