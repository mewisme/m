// Command verify-fixtures validates committed lock bridge conformance fixtures.
package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type fixtureMeta struct {
	Producer        string `json:"producer"`
	ProducerVersion string `json:"producerVersion"`
	ProducerMajor   int    `json:"producerMajor"`
	Family          string `json:"family"`
	LockfileVersion string `json:"lockfileVersion"`
	LockfileSha256  string `json:"lockfileSha256"`
	Command         string `json:"command"`
	Classification  string `json:"classification"`
}

type pinEnv struct {
	Pnpm9  string
	Pnpm10 string
	Pnpm11 string
}

func main() {
	root, err := moduleRoot()
	if err != nil {
		fail(err)
	}
	pins, err := loadPins(filepath.Join(root, "tools", "conformance", "pnpm-versions.env"))
	if err != nil {
		fail(err)
	}
	gen := filepath.Join(root, "fixtures", "locks", "generated")
	var errs []string
	for _, major := range []int{9, 10, 11} {
		want := pins.forMajor(major)
		dir := filepath.Join(gen, fmt.Sprintf("pnpm-%d", major))
		errs = append(errs, verifyTree(dir, want, major)...)
	}
	nubFamilies := []string{
		"nub-basic", "nub-transitive", "nub-workspace",
		"nub-catalog", "nub-peer", "nub-optional",
	}
	for _, family := range nubFamilies {
		errs = append(errs, verifyNub(filepath.Join(gen, family), family)...)
	}
	if len(errs) > 0 {
		fail(fmt.Errorf("%d fixture error(s):\n  %s", len(errs), strings.Join(errs, "\n  ")))
	}
	fmt.Println("ok: lock bridge fixtures verified")
}

func verifyTree(root, wantVersion string, major int) []string {
	var errs []string
	entries, err := os.ReadDir(root)
	if err != nil {
		return []string{fmt.Sprintf("pnpm-%d: %v", major, err)}
	}
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		familyDir := filepath.Join(root, ent.Name())
		metaPath := filepath.Join(familyDir, "metadata.json")
		lockPath := filepath.Join(familyDir, "pnpm-lock.yaml")
		meta, err := readMeta(metaPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s metadata: %v", major, ent.Name(), err))
			continue
		}
		if meta.Producer != "pnpm" {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: producer=%q", major, ent.Name(), meta.Producer))
		}
		if meta.ProducerMajor != major {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: producerMajor=%d", major, ent.Name(), meta.ProducerMajor))
		}
		if meta.ProducerVersion != wantVersion {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: producerVersion=%q want %q", major, ent.Name(), meta.ProducerVersion, wantVersion))
		}
		if meta.Family != ent.Name() {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: family=%q", major, ent.Name(), meta.Family))
		}
		if meta.LockfileVersion != "9.0" {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: lockfileVersion=%q", major, ent.Name(), meta.LockfileVersion))
		}
		if meta.Command == "" {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: missing command", major, ent.Name()))
		}
		if isPlaceholderCommand(meta.Command) {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: placeholder command %q (run generate-lock-fixtures.ps1 -Generate)", major, ent.Name(), meta.Command))
		}
		lockData, err := os.ReadFile(lockPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: %v", major, ent.Name(), err))
			continue
		}
		if !strings.Contains(string(lockData), "lockfileVersion: '9.0'") {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: lock missing v9 marker", major, ent.Name()))
		}
		hash := sha256.Sum256(lockData)
		got := hex.EncodeToString(hash[:])
		if meta.LockfileSha256 != got {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: lockfileSha256 mismatch (meta=%s calc=%s)", major, ent.Name(), meta.LockfileSha256, got))
		}
	}
	return errs
}

func verifyNub(dir, family string) []string {
	var errs []string
	metaPath := filepath.Join(dir, "metadata.json")
	lockPath := filepath.Join(dir, "nub.lock")
	meta, err := readMeta(metaPath)
	if err != nil {
		return []string{fmt.Sprintf("%s metadata: %v", family, err)}
	}
	if meta.Producer != "nub" {
		errs = append(errs, fmt.Sprintf("%s: producer=%q", family, meta.Producer))
	}
	if meta.Family != family {
		errs = append(errs, fmt.Sprintf("%s: family=%q", family, meta.Family))
	}
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return append(errs, fmt.Sprintf("%s: %v", family, err))
	}
	if !strings.Contains(string(lockData), "nubVersion:") {
		errs = append(errs, fmt.Sprintf("%s: missing nubVersion marker", family))
	}
	hash := sha256.Sum256(lockData)
	got := hex.EncodeToString(hash[:])
	if meta.LockfileSha256 != got {
		errs = append(errs, fmt.Sprintf("%s: lockfileSha256 mismatch", family))
	}
	if meta.Command != "" && isPlaceholderCommand(meta.Command) {
		errs = append(errs, fmt.Sprintf("%s: placeholder command %q", family, meta.Command))
	}
	return errs
}

func isPlaceholderCommand(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return false
	}
	placeholders := []string{
		"committed generated fixture",
		"placeholder",
		"TODO",
	}
	lower := strings.ToLower(cmd)
	for _, p := range placeholders {
		if strings.Contains(lower, strings.ToLower(p)) {
			return true
		}
	}
	return false
}

func readMeta(path string) (fixtureMeta, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fixtureMeta{}, err
	}
	var meta fixtureMeta
	if err := json.Unmarshal(raw, &meta); err != nil {
		return fixtureMeta{}, err
	}
	return meta, nil
}

func loadPins(path string) (pinEnv, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return pinEnv{}, err
	}
	var pins pinEnv
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		switch parts[0] {
		case "PNPM9_VERSION":
			pins.Pnpm9 = strings.TrimSpace(parts[1])
		case "PNPM10_VERSION":
			pins.Pnpm10 = strings.TrimSpace(parts[1])
		case "PNPM11_VERSION":
			pins.Pnpm11 = strings.TrimSpace(parts[1])
		}
	}
	if pins.Pnpm9 == "" || pins.Pnpm10 == "" || pins.Pnpm11 == "" {
		return pinEnv{}, fmt.Errorf("incomplete pins in %s", path)
	}
	return pins, nil
}

func (p pinEnv) forMajor(major int) string {
	switch major {
	case 9:
		return p.Pnpm9
	case 10:
		return p.Pnpm10
	case 11:
		return p.Pnpm11
	default:
		return ""
	}
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
	fmt.Fprintf(os.Stderr, "verify-fixtures: %v\n", err)
	os.Exit(1)
}
