// Command check-license asserts the repository root LICENSE exists and is MIT.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root, err := moduleRoot()
	if err != nil {
		fail(err)
	}
	path := filepath.Join(root, "LICENSE")
	b, err := os.ReadFile(path)
	if err != nil {
		fail(fmt.Errorf("LICENSE: %w", err))
	}
	text := string(b)
	if !strings.Contains(text, "MIT") {
		fail(fmt.Errorf("%s does not mention MIT", path))
	}
	fmt.Println("ok: LICENSE is MIT")
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
	fmt.Fprintf(os.Stderr, "check-license: %v\n", err)
	os.Exit(1)
}
