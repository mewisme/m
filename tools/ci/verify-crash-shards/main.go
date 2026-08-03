// Command verify-crash-shards proves every crash integration test belongs to
// exactly one Windows CI shard. Shard regexes must stay in sync with
// .github/workflows/full.yml crash-integration matrix run expressions.
package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type shard struct {
	name    string
	pattern *regexp.Regexp
}

var windowsShards = []shard{
	{name: "snapshot", pattern: regexp.MustCompile(`^Test(SnapshotCrash|WorkspaceSnapshotRestoreCrash)`)},
	{name: "install", pattern: regexp.MustCompile(`^TestTxnCrash`)},
	{name: "update", pattern: regexp.MustCompile(`^TestUpdateCrash`)},
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "verify-crash-shards: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	root, err := moduleRoot()
	if err != nil {
		return err
	}
	tests, err := listCrashTests(root)
	if err != nil {
		return err
	}
	if len(tests) == 0 {
		return fmt.Errorf("no crash tests found")
	}

	var problems []string
	for _, test := range tests {
		var matched []string
		for _, s := range windowsShards {
			if s.pattern.MatchString(test) {
				matched = append(matched, s.name)
			}
		}
		switch len(matched) {
		case 0:
			problems = append(problems, fmt.Sprintf("%s: matches no shard", test))
		case 1:
			fmt.Printf("%s -> %s\n", test, matched[0])
		default:
			problems = append(problems, fmt.Sprintf("%s: matches multiple shards: %s", test, strings.Join(matched, ", ")))
		}
	}
	if len(problems) > 0 {
		return fmt.Errorf("%s", strings.Join(problems, "; "))
	}
	fmt.Printf("ok: %d crash tests each assigned to exactly one Windows shard\n", len(tests))
	return nil
}

func listCrashTests(root string) ([]string, error) {
	cmd := exec.Command("go", "test", "-tags", "crash", "./tests/integration/...", "-list", "Crash")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("go test -list: %w\n%s", err, out)
	}
	var tests []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Test") {
			tests = append(tests, line)
		}
	}
	return tests, nil
}

func moduleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}
