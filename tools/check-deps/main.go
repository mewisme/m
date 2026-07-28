// Command check-deps fails if go.mod modules are outside tools/allowlist/modules.txt.
package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	root, err := moduleRoot()
	if err != nil {
		fail(err)
	}
	allow, err := loadAllowlist(filepath.Join(root, "tools", "allowlist", "modules.txt"))
	if err != nil {
		fail(err)
	}
	mods, err := listModules(root)
	if err != nil {
		fail(err)
	}
	var banned []string
	for _, m := range mods {
		if m == "" || m == "github.com/mewisme/mew" {
			continue
		}
		if !allow[m] {
			banned = append(banned, m)
		}
	}
	if len(banned) > 0 {
		fail(fmt.Errorf("modules not in allowlist:\n  %s", strings.Join(banned, "\n  ")))
	}
	fmt.Printf("ok: %d modules allowlisted\n", len(mods)-1)
}

func loadAllowlist(path string) (map[string]bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	out := make(map[string]bool)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out[line] = true
	}
	return out, sc.Err()
}

func listModules(root string) ([]string, error) {
	cmd := exec.Command("go", "list", "-m", "-f", "{{.Path}}", "all")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("go list: %v\n%s", err, ee.Stderr)
		}
		return nil, err
	}
	var mods []string
	for _, line := range strings.Split(string(bytes.TrimSpace(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			mods = append(mods, line)
		}
	}
	return mods, nil
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

func fail(err error) {
	fmt.Fprintf(os.Stderr, "check-deps: %v\n", err)
	os.Exit(1)
}
