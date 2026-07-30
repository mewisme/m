// Package project discovers project roots and package-manager identity.
package project

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

// Identity is the package-manager identity for a project.
type Identity string

const (
	IdentityMew  Identity = "mew"
	IdentityNub  Identity = "nub"
	IdentityNPM  Identity = "npm"
	IdentityPNPM Identity = "pnpm"
	IdentityYarn Identity = "yarn"
	IdentityBun  Identity = "bun"
)

// Signal records why an identity was chosen.
type Signal struct {
	Kind   string // packageManager|devEngines|lockfile|default
	Detail string
	Path   string
}

// Project is a discovered project root with identity.
type Project struct {
	Root       string
	Rel        string // importer path relative to Root ("." for root)
	Identity   Identity
	Signals    []Signal
	Doc        *manifest.Document
	Normalized *manifest.Manifest
}

// FindRoot walks up from cwd looking for package.json.
func FindRoot(cwd string) (string, error) {
	dir, err := filepath.Abs(cwd)
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "project.find", cwd, err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "package.json")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", apperr.New(apperr.NotFound, "project.find", cwd, "package.json not found")
		}
		dir = parent
	}
}

// DetectIdentity applies AGENTS.md detection order.
func DetectIdentity(root string) (*Project, error) {
	pkgPath := filepath.Join(root, "package.json")
	fieldID, fieldSig, fieldKind, err := readPackageManagerField(pkgPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	found := listLockfiles(root)
	p := &Project{Root: root}

	if fieldID != "" {
		p.Signals = append(p.Signals, Signal{Kind: fieldKind, Detail: fieldSig, Path: pkgPath})
		for _, f := range found {
			sig := Signal{Kind: "lockfile", Detail: f.file, Path: filepath.Join(root, f.file)}
			p.Signals = append(p.Signals, sig)
			if f.id != fieldID {
				return nil, apperr.New(apperr.Config, "identity", root,
					"conflicting signals: package field is "+string(fieldID)+" but lockfile is "+string(f.id))
			}
		}
		p.Identity = fieldID
		return p, nil
	}

	if len(found) == 0 {
		p.Identity = IdentityMew
		p.Signals = append(p.Signals, Signal{Kind: "default", Detail: "mew native", Path: root})
		return p, nil
	}

	idSet := map[Identity]struct{}{}
	for _, f := range found {
		idSet[f.id] = struct{}{}
		p.Signals = append(p.Signals, Signal{Kind: "lockfile", Detail: f.file, Path: filepath.Join(root, f.file)})
	}
	if len(idSet) > 1 {
		// #region agent log
		files := make([]string, 0, len(found))
		ids := make([]string, 0, len(idSet))
		for _, f := range found {
			files = append(files, f.file+"="+string(f.id))
		}
		for id := range idSet {
			ids = append(ids, string(id))
		}
		agentIdentityDebugLog("identity.go:conflict", "A", "conflicting lockfiles", map[string]any{
			"root": root, "fieldID": string(fieldID), "locks": files, "identities": ids,
		})
		// #endregion
		return nil, apperr.New(apperr.Config, "identity", root, "conflicting lockfiles present")
	}
	p.Identity = found[0].id
	return p, nil
}

type packageJSON struct {
	PackageManager string          `json:"packageManager"`
	DevEngines     json.RawMessage `json:"devEngines"`
}

func readPackageManagerField(path string) (id Identity, detail, kind string, err error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return "", "", "", err
	}
	var pkg packageJSON
	if err := json.Unmarshal(b, &pkg); err != nil {
		return "", "", "", apperr.Wrap(apperr.Config, "project.package.json", path, err)
	}
	if pkg.PackageManager != "" {
		id, ok := parsePMName(pkg.PackageManager)
		if !ok {
			return "", "", "", apperr.New(apperr.Config, "identity", path, "unrecognized packageManager "+pkg.PackageManager)
		}
		return id, pkg.PackageManager, "packageManager", nil
	}
	if len(pkg.DevEngines) > 0 {
		var de struct {
			PackageManager string `json:"packageManager"`
		}
		if err := json.Unmarshal(pkg.DevEngines, &de); err == nil && de.PackageManager != "" {
			id, ok := parsePMName(de.PackageManager)
			if !ok {
				return "", "", "", apperr.New(apperr.Config, "identity", path, "unrecognized devEngines.packageManager")
			}
			return id, de.PackageManager, "devEngines", nil
		}
	}
	return "", "", "", nil
}

func parsePMName(s string) (Identity, bool) {
	name := s
	if i := strings.IndexByte(s, '@'); i >= 0 {
		name = s[:i]
	}
	name = strings.ToLower(strings.TrimSpace(name))
	switch name {
	case "npm":
		return IdentityNPM, true
	case "pnpm":
		return IdentityPNPM, true
	case "yarn":
		return IdentityYarn, true
	case "bun":
		return IdentityBun, true
	case "nub":
		return IdentityNub, true
	case "mew", "m":
		return IdentityMew, true
	default:
		return "", false
	}
}

type lockCand struct {
	file string
	id   Identity
}

func listLockfiles(root string) []lockCand {
	cands := []lockCand{
		{"nub.lock", IdentityNub},
		{"m.lock", IdentityMew},
		{"pnpm-lock.yaml", IdentityPNPM},
		{"package-lock.json", IdentityNPM},
		{"npm-shrinkwrap.json", IdentityNPM},
		{"yarn.lock", IdentityYarn},
		{"bun.lock", IdentityBun},
		{"bun.lockb", IdentityBun},
	}
	var found []lockCand
	for _, c := range cands {
		if _, err := os.Stat(filepath.Join(root, c.file)); err == nil {
			found = append(found, c)
		}
	}
	return found
}

// ReadsBrandedConfig reports whether identity may treat branded PM files as authority.
func ReadsBrandedConfig(id Identity) bool {
	return id != IdentityMew
}

// #region agent log
func agentIdentityDebugLog(location, hypothesisId, message string, data map[string]any) {
	payload := map[string]any{
		"sessionId": "d57042", "timestamp": time.Now().UnixMilli(),
		"location": location, "message": message, "data": data,
		"hypothesisId": hypothesisId, "runId": "post-fix",
	}
	b, err := json.Marshal(payload)
	if err != nil {
		return
	}
	f, err := os.OpenFile(`f:\Project\package-managers\mew\debug-d57042.log`, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	_, _ = f.Write(append(b, '\n'))
}

// #endregion
