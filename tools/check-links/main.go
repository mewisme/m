// Command check-links validates relative links in tracked Markdown files.
package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

var linkRE = regexp.MustCompile(`\[[^\]]*\]\(([^)]+)\)`)

func main() {
	root, err := findRepoRoot()
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-links: %v\n", err)
		os.Exit(1)
	}

	files, err := trackedMarkdown(root)
	if err != nil {
		fmt.Fprintf(os.Stderr, "check-links: %v\n", err)
		os.Exit(1)
	}

	var failures int
	for _, rel := range files {
		abs := filepath.Join(root, filepath.FromSlash(rel))
		fails := checkFile(root, abs, rel)
		failures += fails
	}
	if failures > 0 {
		fmt.Fprintf(os.Stderr, "check-links: %d broken relative link(s)\n", failures)
		os.Exit(1)
	}
}

func findRepoRoot() (string, error) {
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

func trackedMarkdown(root string) ([]string, error) {
	cmd := exec.Command("git", "ls-files", "*.md")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var files []string
	sc := bufio.NewScanner(strings.NewReader(string(out)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line != "" {
			files = append(files, line)
		}
	}
	return files, sc.Err()
}

func checkFile(root, abs, rel string) int {
	data, err := os.ReadFile(abs)
	if err != nil {
		fmt.Fprintf(os.Stderr, "%s: read: %v\n", rel, err)
		return 1
	}
	dir := filepath.Dir(abs)
	failures := 0
	for _, m := range linkRE.FindAllStringSubmatch(string(data), -1) {
		target := strings.TrimSpace(m[1])
		if target == "" {
			continue
		}
		if strings.HasPrefix(target, "http://") || strings.HasPrefix(target, "https://") {
			continue
		}
		if strings.HasPrefix(target, "mailto:") {
			continue
		}
		pathPart := target
		if i := strings.IndexAny(pathPart, "#?"); i >= 0 {
			pathPart = pathPart[:i]
		}
		if pathPart == "" {
			continue
		}
		if strings.HasPrefix(pathPart, "/") {
			continue
		}
		resolved := filepath.Clean(filepath.Join(dir, filepath.FromSlash(pathPart)))
		if _, err := os.Stat(resolved); err != nil {
			fmt.Fprintf(os.Stderr, "%s: broken link %q -> %s\n", rel, target, pathPart)
			failures++
		}
	}
	return failures
}
