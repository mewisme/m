package app

import (
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

const (
	benchDefaultWarmup  = 1
	benchDefaultSamples = 7
)

func benchWarmupCount(n int) int {
	if n > 0 {
		return n
	}
	return benchDefaultWarmup
}

func benchSampleCount(n int) int {
	if n > 0 {
		return n
	}
	return benchDefaultSamples
}

func benchMedian(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	mid := len(cp) / 2
	if len(cp)%2 == 1 {
		return cp[mid]
	}
	return (cp[mid-1] + cp[mid]) / 2
}

func benchP95(samples []int64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(float64(len(cp))*0.95+0.999999) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func benchRuntimeMetadata(commit string) (goVersion, goos, goarch string) {
	return runtime.Version(), runtime.GOOS, runtime.GOARCH
}

func benchRunnerClass() string {
	if v := strings.TrimSpace(os.Getenv("MEW_BENCH_RUNNER_CLASS")); v != "" {
		return v
	}
	if strings.EqualFold(os.Getenv("GITHUB_ACTIONS"), "true") {
		runner := strings.TrimSpace(os.Getenv("RUNNER_OS"))
		if runner == "" {
			runner = runtime.GOOS
		}
		return "github-actions-" + strings.ToLower(runner)
	}
	return "local-" + runtime.GOOS
}

func fixtureTreeDigest(root string) (string, error) {
	var entries []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		entries = append(entries, rel+"\x00"+hex.EncodeToString(sum[:]))
		return nil
	})
	if err != nil {
		return "", err
	}
	sort.Strings(entries)
	h := sha256.New()
	for _, line := range entries {
		if _, err := io.WriteString(h, line+"\n"); err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
