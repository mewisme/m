// Command check-license asserts the repository root LICENSE is Apache-2.0 and NOTICE exists.
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
	licensePath := filepath.Join(root, "LICENSE")
	license, err := os.ReadFile(licensePath)
	if err != nil {
		fail(fmt.Errorf("LICENSE: %w", err))
	}
	licenseText := string(license)
	if !strings.Contains(licenseText, "Apache License") || !strings.Contains(licenseText, "Version 2.0") {
		fail(fmt.Errorf("%s is not Apache License 2.0", licensePath))
	}

	noticePath := filepath.Join(root, "NOTICE")
	notice, err := os.ReadFile(noticePath)
	if err != nil {
		fail(fmt.Errorf("NOTICE: %w", err))
	}
	noticeText := string(notice)
	if !strings.Contains(noticeText, "Copyright 2026 Nguyễn Mậu Minh") {
		fail(fmt.Errorf("%s missing copyright attribution", noticePath))
	}
	if !strings.Contains(noticeText, "Apache License, Version 2.0") {
		fail(fmt.Errorf("%s missing Apache-2.0 declaration", noticePath))
	}

	fmt.Println("ok: LICENSE is Apache-2.0")
	fmt.Println("ok: NOTICE present")
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
