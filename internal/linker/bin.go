package linker

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

// BinCommandsFromDir reads package.json bin entries from an extracted package dir.
func BinCommandsFromDir(pkgDir string) (map[string]string, error) {
	return BinCommandsFromDirNamed(pkgDir, "")
}

// BinCommandsFromDirNamed reads bin entries when package.json name matches wantName.
// An empty wantName skips the name check.
func BinCommandsFromDirNamed(pkgDir, wantName string) (map[string]string, error) {
	doc, err := manifest.Load(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return nil, err
	}
	if wantName != "" && doc.Name != wantName {
		return nil, nil
	}
	return BinCommands(doc)
}

// BinCommands parses bin entries from a manifest document.
func BinCommands(doc *manifest.Document) (map[string]string, error) {
	if doc == nil || len(doc.Bin) == 0 {
		return nil, nil
	}
	if err := manifest.ValidateBin(doc.Bin); err != nil {
		return nil, err
	}
	var s string
	if err := json.Unmarshal(doc.Bin, &s); err == nil {
		name := doc.Name
		if name == "" {
			return nil, apperr.New(apperr.Manifest, "linker.bin", "bin", "package name required for string bin")
		}
		return map[string]string{name: s}, nil
	}
	var m map[string]string
	if err := json.Unmarshal(doc.Bin, &m); err != nil {
		return nil, apperr.Wrap(apperr.Manifest, "linker.bin", "bin", err)
	}
	return m, nil
}

// WriteBins creates node_modules/.bin shims for the given sources.
func WriteBins(nodeModules string, sources []BinSource) error {
	if len(sources) == 0 {
		return nil
	}
	binDir := filepath.Join(nodeModules, ".bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "linker.bin", binDir, err)
	}
	seen := make(map[string]struct{}, len(sources))
	for _, src := range sources {
		if src.Cmd == "" || src.Target == "" || src.PackageDir == "" {
			return apperr.New(apperr.Internal, "linker.bin", src.Cmd, "incomplete bin source")
		}
		if _, dup := seen[src.Cmd]; dup {
			return apperr.New(apperr.Internal, "linker.bin", src.Cmd, "duplicate bin command")
		}
		seen[src.Cmd] = struct{}{}
		script := filepath.Join(src.PackageDir, filepath.FromSlash(strings.TrimPrefix(src.Target, "./")))
		rel, err := filepath.Rel(binDir, script)
		if err != nil {
			return apperr.Wrap(apperr.IO, "linker.bin", src.Cmd, err)
		}
		rel = filepath.ToSlash(rel)
		if runtime.GOOS == "windows" {
			if err := writeWindowsBin(binDir, src.Cmd, rel); err != nil {
				return err
			}
			continue
		}
		if err := writeUnixBin(binDir, src.Cmd, rel); err != nil {
			return err
		}
	}
	return nil
}

func writeUnixBin(binDir, cmd, relScript string) error {
	path := filepath.Join(binDir, cmd)
	body := fmt.Sprintf("#!/bin/sh\nbasedir=$(dirname \"$(echo \"$0\" | sed -e 's,\\\\,/,g')\")\nexec node \"$basedir/%s\" \"$@\"\n", relScript)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "linker.bin", path, err)
	}
	return nil
}

func writeWindowsBin(binDir, cmd, relScript string) error {
	path := filepath.Join(binDir, cmd+".cmd")
	winRel := strings.ReplaceAll(relScript, "/", "\\")
	body := fmt.Sprintf("@ECHO off\r\nGOTO start\r\n:find_dp0\r\nSET dp0=%%~dp0\r\nEXIT /b\r\n:start\r\nSETLOCAL\r\nCALL :find_dp0\r\nnode \"%%dp0%%\\%s\" %%*\r\n", winRel)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "linker.bin", path, err)
	}
	return nil
}
