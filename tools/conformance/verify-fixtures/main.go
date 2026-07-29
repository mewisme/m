// Command verify-fixtures validates committed lock bridge conformance fixtures.
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/tools/conformance/fixturemeta"
)

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
		family := ent.Name()
		familyDir := filepath.Join(root, family)
		metaPath := filepath.Join(familyDir, "metadata.json")
		meta, err := fixturemeta.ReadMeta(metaPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s metadata: %v", major, family, err))
			continue
		}
		opts := fixturemeta.ValidateOptions{
			WantProducer:        "pnpm",
			WantProducerVersion: wantVersion,
			WantProducerMajor:   major,
			WantFamily:          family,
		}
		for _, e := range meta.Validate(opts) {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: %s", major, family, e))
		}
		if meta.LockfileVersion != "9.0" {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: lockfileVersion=%q", major, family, meta.LockfileVersion))
		}
		lockPath := filepath.Join(familyDir, "pnpm-lock.yaml")
		lockData, err := os.ReadFile(lockPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: %v", major, family, err))
			continue
		}
		if !strings.Contains(string(lockData), "lockfileVersion: '9.0'") {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: lock missing v9 marker", major, family))
		}
		for _, e := range fixturemeta.VerifyFixtureDir(familyDir, meta, "pnpm-lock.yaml") {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: %s", major, family, e))
		}
		src := filepath.Join(moduleRootMust(), "fixtures", "locks", "sources", "pnpm", family)
		if digest, err := fixturemeta.SourceTreeDigest(src); err == nil && meta.SourceTreeDigest != digest {
			errs = append(errs, fmt.Sprintf("pnpm-%d/%s: sourceTreeDigest mismatch meta=%s calc=%s", major, family, meta.SourceTreeDigest, digest))
		}
	}
	return errs
}

func verifyNub(dir, family string) []string {
	var errs []string
	metaPath := filepath.Join(dir, "metadata.json")
	meta, err := fixturemeta.ReadMeta(metaPath)
	if err != nil {
		return []string{fmt.Sprintf("%s metadata: %v", family, err)}
	}
	opts := fixturemeta.ValidateOptions{
		WantProducer: "nub",
		WantFamily:   family,
	}
	for _, e := range meta.Validate(opts) {
		errs = append(errs, fmt.Sprintf("%s: %s", family, e))
	}
	if meta.Classification != fixturemeta.ClassDerived {
		errs = append(errs, fmt.Sprintf("%s: classification=%q want derived", family, meta.Classification))
	}
	lockPath := filepath.Join(dir, "nub.lock")
	lockData, err := os.ReadFile(lockPath)
	if err != nil {
		return append(errs, fmt.Sprintf("%s: %v", family, err))
	}
	if !strings.Contains(string(lockData), "nubVersion:") {
		errs = append(errs, fmt.Sprintf("%s: missing nubVersion marker", family))
	}
	for _, e := range fixturemeta.VerifyFixtureDir(dir, meta, "nub.lock") {
		errs = append(errs, fmt.Sprintf("%s: %s", family, e))
	}
	return errs
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

func moduleRootMust() string {
	root, err := moduleRoot()
	if err != nil {
		panic(err)
	}
	return root
}

func fail(err error) {
	fmt.Fprintf(os.Stderr, "verify-fixtures: %v\n", err)
	os.Exit(1)
}
