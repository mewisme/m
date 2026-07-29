package conformance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

// TestSummary aggregates go test -json events for one invocation.
type TestSummary struct {
	TestsMatched int
	Passed       int
	Failed       int
	Skipped      int
	PackageFail  bool
	ParseError   string
}

// TotalOutcomes returns pass+fail+skip at test granularity.
func (s TestSummary) TotalOutcomes() int {
	return s.Passed + s.Failed + s.Skipped
}

// FailReason returns a non-empty reason when the summary should fail certification.
func (s TestSummary) FailReason(suite Suite, exitCode int, requireTools bool) string {
	if s.ParseError != "" {
		return s.ParseError
	}
	if s.PackageFail {
		return "package build or setup failed"
	}
	if exitCode != 0 && s.TestsMatched == 0 && s.TotalOutcomes() == 0 {
		return "go test exited non-zero"
	}
	if s.TestsMatched == 0 {
		return "zero tests matched run filter"
	}
	if s.TotalOutcomes() == 0 {
		return "no test outcomes recorded"
	}
	if s.Skipped > 0 && s.Skipped == s.TestsMatched {
		return "all matched tests skipped"
	}
	if suite.Required && s.Skipped > 0 {
		return fmt.Sprintf("%d required test(s) skipped", s.Skipped)
	}
	if requireTools && suite.RequireTools && s.Skipped > 0 {
		return fmt.Sprintf("%d test(s) skipped with requireTools", s.Skipped)
	}
	if s.Failed > 0 {
		return fmt.Sprintf("%d test(s) failed", s.Failed)
	}
	if exitCode != 0 {
		return "go test exited non-zero"
	}
	return ""
}

type testJSONEvent struct {
	Action  string `json:"Action"`
	Package string `json:"Package"`
	Test    string `json:"Test"`
	Output  string `json:"Output"`
}

// ParseTestJSONLines parses newline-delimited go test -json output.
func ParseTestJSONLines(r io.Reader) (TestSummary, error) {
	var summary TestSummary
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	lines := 0
	jsonLines := 0
	running := map[string]struct{}{}

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		lines++
		var ev testJSONEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			summary.ParseError = fmt.Sprintf("malformed testjson line %d: %v", lines, err)
			return summary, fmt.Errorf("%s", summary.ParseError)
		}
		jsonLines++
		if ev.Action == "output" && strings.Contains(ev.Output, "no tests to run") {
			continue
		}
		if ev.Test == "" {
			switch ev.Action {
			case "fail":
				summary.PackageFail = true
			}
			continue
		}
		switch ev.Action {
		case "run":
			running[ev.Test] = struct{}{}
			summary.TestsMatched++
		case "pass":
			summary.Passed++
			delete(running, ev.Test)
		case "fail":
			summary.Failed++
			delete(running, ev.Test)
		case "skip":
			summary.Skipped++
			delete(running, ev.Test)
		}
	}
	if err := scanner.Err(); err != nil {
		return summary, err
	}
	if jsonLines == 0 {
		summary.ParseError = "no go test -json events"
		return summary, fmt.Errorf("%s", summary.ParseError)
	}
	return summary, nil
}

// ParseTestJSON parses a byte buffer of go test -json output.
func ParseTestJSON(data []byte) (TestSummary, error) {
	return ParseTestJSONLines(bytes.NewReader(data))
}
