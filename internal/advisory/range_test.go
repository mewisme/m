package advisory

import "testing"

func TestVersionMatchesRangeSingleInterval(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "1.0.0"},
		{Fixed: "2.0.0"},
	}}
	cases := []struct {
		version string
		want    bool
	}{
		{"0.9.0", false},
		{"1.0.0", true},
		{"1.9.9", true},
		{"2.0.0", false},
		{"2.1.0", false},
	}
	for _, tc := range cases {
		if got := versionMatchesRange(r, tc.version); got != tc.want {
			t.Fatalf("version %s got %v want %v", tc.version, got, tc.want)
		}
	}
}

func TestVersionMatchesRangeLastAffected(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "1.0.0"},
		{LastAffected: "1.5.0"},
	}}
	if !versionMatchesRange(r, "1.5.0") {
		t.Fatal("last_affected boundary should be vulnerable")
	}
	if versionMatchesRange(r, "1.6.0") {
		t.Fatal("above last_affected should not match")
	}
}

func TestVersionMatchesRangeMultiInterval(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "0"},
		{Fixed: "1.0.0"},
		{Introduced: "2.0.0"},
		{Fixed: "3.0.0"},
	}}
	cases := []struct {
		version string
		want    bool
	}{
		{"0.5.0", true},
		{"1.0.0", false},
		{"1.5.0", false},
		{"2.5.0", true},
		{"3.0.0", false},
	}
	for _, tc := range cases {
		if got := versionMatchesRange(r, tc.version); got != tc.want {
			t.Fatalf("version %s got %v want %v", tc.version, got, tc.want)
		}
	}
}

func TestVersionMatchesRangeOpenEnded(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "2.0.0"},
	}}
	if !versionMatchesRange(r, "9.9.9") {
		t.Fatal("open-ended range should match high versions")
	}
	if versionMatchesRange(r, "1.9.9") {
		t.Fatal("below introduced should not match")
	}
}

func TestVersionMatchesRangeOpenEndedConcreteIntroduced(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "2.4.0"},
	}}
	cases := []struct {
		version string
		want    bool
	}{
		{"2.3.9", false},
		{"2.4.0", true},
		{"2.4.1", true},
		{"99.0.0", true},
	}
	for _, tc := range cases {
		if got := versionMatchesRange(r, tc.version); got != tc.want {
			t.Fatalf("version %s got %v want %v", tc.version, got, tc.want)
		}
	}
}

func TestVersionMatchesRangeEqualToIntroducedBoundary(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "1.2.3"},
		{Fixed: "2.0.0"},
	}}
	if !versionMatchesRange(r, "1.2.3") {
		t.Fatal("version equal to introduced should match")
	}
}

func TestVersionMatchesRangeIntroducedZeroOpenEnded(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "0"},
	}}
	if !versionMatchesRange(r, "99.0.0") {
		t.Fatal("introduced 0 with no upper bound should match")
	}
}

func TestNormalizeAuditVersionStripsPeerSuffix(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: "1.0.0"},
		{Fixed: "2.0.0"},
	}}
	if !versionMatchesRange(r, "1.2.0#peer@1.0.0") {
		t.Fatal("peer suffix should be stripped before matching")
	}
}

func TestVersionMatchesRangeMalformedExpanded(t *testing.T) {
	r := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Fixed: "1.0.0"},
	}}
	if versionMatchesRange(r, "0.5.0") {
		t.Fatal("fixed without introduced should not match")
	}
	_, warnings := buildSemverIntervals(r.Events)
	if len(warnings) != 1 {
		t.Fatalf("warnings=%v", warnings)
	}

	emptyIntro := VersionRange{Type: "SEMVER", Events: []RangeEvent{
		{Introduced: ""},
		{Fixed: "2.0.0"},
	}}
	if versionMatchesRange(emptyIntro, "1.0.0") {
		t.Fatal("empty introduced should not match")
	}
}

func TestVersionMatchesRangeSkipsNonSemver(t *testing.T) {
	r := VersionRange{Type: "GIT", Events: []RangeEvent{{Introduced: "0"}}}
	if versionMatchesRange(r, "1.0.0") {
		t.Fatal("non-SEMVER range type should not match")
	}
}

func TestSeverityRankAndFailOnThreshold(t *testing.T) {
	cases := []struct {
		severity string
		want     int
	}{
		{"critical", 4},
		{"high", 3},
		{"moderate", 2},
		{"low", 1},
		{"", 1},
		{"9.8", 4},
		{"7.1", 3},
		{"5.0", 2},
		{"2.0", 1},
	}
	for _, tc := range cases {
		if got := SeverityRank(tc.severity); got != tc.want {
			t.Fatalf("severity %q rank=%d want %d", tc.severity, got, tc.want)
		}
	}

	report := AuditReport{Vulnerabilities: []Vulnerability{{Severity: "critical"}}}
	if !ReportExceedsThreshold(report, FailOnCritical) {
		t.Fatal("critical finding should exceed critical threshold")
	}
	if ReportExceedsThreshold(report, FailOnNone) {
		t.Fatal("none threshold should never fail")
	}
	if ReportExceedsThreshold(AuditReport{}, FailOnLow) {
		t.Fatal("empty report should not fail")
	}

	highOnly := AuditReport{Vulnerabilities: []Vulnerability{{Severity: "high"}}}
	if ReportExceedsThreshold(highOnly, FailOnCritical) {
		t.Fatal("high should not exceed critical threshold")
	}
	if !ReportExceedsThreshold(highOnly, FailOnHigh) {
		t.Fatal("high should exceed high threshold")
	}
}

func TestParseFailOn(t *testing.T) {
	if _, err := ParseFailOn("invalid"); err == nil {
		t.Fatal("expected error for invalid fail-on")
	}
	level, err := ParseFailOn("HIGH")
	if err != nil || level != FailOnHigh {
		t.Fatalf("level=%q err=%v", level, err)
	}
}

func TestEntryMatchesMultiIntervalRange(t *testing.T) {
	db := &AdvisoryDB{Entries: []OSVEntry{{
		ID: "OSV-MULTI",
		Affected: []Affected{{
			Package: struct {
				Ecosystem string `json:"ecosystem"`
				Name      string `json:"name"`
			}{Ecosystem: "npm", Name: "gap-pkg"},
			Ranges: []VersionRange{{
				Type: "SEMVER",
				Events: []RangeEvent{
					{Introduced: "0"},
					{Fixed: "1.0.0"},
					{Introduced: "2.0.0"},
					{Fixed: "3.0.0"},
				},
			}},
		}},
	}}}
	if !db.IsVulnerable("gap-pkg", "0.5.0") {
		t.Fatal("expected vulnerable in first interval")
	}
	if db.IsVulnerable("gap-pkg", "1.5.0") {
		t.Fatal("expected safe in gap interval")
	}
	if !db.IsVulnerable("gap-pkg", "2.5.0") {
		t.Fatal("expected vulnerable in second interval")
	}
}

func TestLoadCollectsRangeWarnings(t *testing.T) {
	raw := []byte(`[{"id":"BAD","affected":[{"package":{"ecosystem":"npm","name":"x"},"ranges":[{"type":"SEMVER","events":[{"fixed":"1.0.0"}]}]}]}]`)
	db, err := Load(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(db.Warnings) != 1 {
		t.Fatalf("warnings=%+v", db.Warnings)
	}
	if db.Warnings[0].EntryID != "BAD" {
		t.Fatalf("warning=%+v", db.Warnings[0])
	}
}
