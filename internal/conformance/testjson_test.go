package conformance

import (
	"strings"
	"testing"
)

func TestParseTestJSONPass(t *testing.T) {
	input := strings.Join([]string{
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestA"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"pkg","Test":"TestA","Elapsed":0.01}`,
		`{"Time":"2026-01-01T00:00:02Z","Action":"run","Package":"pkg","Test":"TestB"}`,
		`{"Time":"2026-01-01T00:00:03Z","Action":"pass","Package":"pkg","Test":"TestB","Elapsed":0.02}`,
		`{"Time":"2026-01-01T00:00:04Z","Action":"pass","Package":"pkg","Elapsed":0.03}`,
	}, "\n")
	sum, err := ParseTestJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	if sum.TestsMatched != 2 || sum.Passed != 2 || sum.Failed != 0 || sum.Skipped != 0 {
		t.Fatalf("summary=%+v", sum)
	}
	suite := Suite{Required: true}
	if reason := sum.FailReason(suite, 0, false); reason != "" {
		t.Fatalf("unexpected fail: %q", reason)
	}
}

func TestParseTestJSONAllSkipped(t *testing.T) {
	input := strings.Join([]string{
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestA"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"skip","Package":"pkg","Test":"TestA","Elapsed":0}`,
	}, "\n")
	sum, err := ParseTestJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	suite := Suite{Required: true}
	if got := sum.FailReason(suite, 0, false); got != "all matched tests skipped" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTestJSONZeroMatch(t *testing.T) {
	input := `{"Time":"2026-01-01T00:00:00Z","Action":"pass","Package":"pkg","Elapsed":0}`
	sum, err := ParseTestJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	suite := Suite{Required: true}
	if got := sum.FailReason(suite, 0, false); got != "zero tests matched run filter" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTestJSONMalformed(t *testing.T) {
	_, err := ParseTestJSON([]byte("not json\n"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseTestJSONPackageFail(t *testing.T) {
	input := `{"Time":"2026-01-01T00:00:00Z","Action":"fail","Package":"pkg","Elapsed":0}`
	sum, err := ParseTestJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	suite := Suite{Required: true}
	if got := sum.FailReason(suite, 1, false); got != "package build or setup failed" {
		t.Fatalf("got %q", got)
	}
}

func TestParseTestJSONRequiredSkip(t *testing.T) {
	input := strings.Join([]string{
		`{"Time":"2026-01-01T00:00:00Z","Action":"run","Package":"pkg","Test":"TestA"}`,
		`{"Time":"2026-01-01T00:00:01Z","Action":"pass","Package":"pkg","Test":"TestA","Elapsed":0.01}`,
		`{"Time":"2026-01-01T00:00:02Z","Action":"run","Package":"pkg","Test":"TestB"}`,
		`{"Time":"2026-01-01T00:00:03Z","Action":"skip","Package":"pkg","Test":"TestB","Elapsed":0}`,
	}, "\n")
	sum, err := ParseTestJSON([]byte(input))
	if err != nil {
		t.Fatal(err)
	}
	suite := Suite{Required: true}
	if got := sum.FailReason(suite, 0, false); got != "1 required test(s) skipped" {
		t.Fatalf("got %q", got)
	}
}
