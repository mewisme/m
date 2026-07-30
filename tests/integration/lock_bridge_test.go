package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
)

const lodashLockYAML = `lockfileVersion: '9.0'

settings:
  autoInstallPeers: true
  excludeLinksFromLockfile: false

importers:
  .:
    dependencies:
      lodash:
        specifier: 4.17.21
        version: 4.17.21

packages:
  lodash@4.17.21:
    resolution: {integrity: sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63}
    engines: {node: '>=0.10.0'}

snapshots:
  lodash@4.17.21: {}
`

const lockBridgePackageJSON = `{
  "name": "lock-bridge-test",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "lodash": "4.17.21"
  }
}`

func setupLockBridgeProject(t *testing.T, lockName, lockBody string) (projDir, cfgPath string) {
	t.Helper()
	projDir, cfgPath, _ = setupRegistryProject(t, lockBridgePackageJSON)
	if err := os.WriteFile(filepath.Join(projDir, lockName), []byte(lockBody), 0o644); err != nil {
		t.Fatal(err)
	}
	return projDir, cfgPath
}

const npmLockBridgePackageJSON = `{
  "name": "lock-bridge-test",
  "version": "1.0.0",
  "private": true,
  "packageManager": "npm@10.9.2",
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`

func setupNpmLockBridgeProject(t *testing.T) (projDir, cfgPath string) {
	t.Helper()
	projDir, cfgPath, _ = setupRegistryProject(t, npmLockBridgePackageJSON)
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("bootstrap install exit=%d out=%s", code, out)
	}
	return projDir, cfgPath
}

func TestLockBridgeNpmInstallPreservesIncumbent(t *testing.T) {
	projDir, cfgPath := setupNpmLockBridgeProject(t)
	before, err := os.ReadFile(filepath.Join(projDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--frozen-lockfile")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); err == nil {
		t.Fatal("m.lock must not be created on npm install")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("unchanged graph must preserve package-lock.json bytes")
	}
}

func TestLockBridgeNpmAddRejected(t *testing.T) {
	projDir, cfgPath := setupNpmLockBridgeProject(t)
	before, err := os.ReadFile(filepath.Join(projDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "add", "pkg-a@1.0.0")
	if code == 0 {
		t.Fatalf("expected npm add rejection, out=%s", out)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "package-lock.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("rejected add must not mutate incumbent package-lock.json")
	}
}

func TestLockBridgeNubInstallPreservesIncumbent(t *testing.T) {
	projDir, cfgPath := setupLockBridgeProject(t, "nub.lock", lodashLockYAML)
	before, err := os.ReadFile(filepath.Join(projDir, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--frozen-lockfile", "--pnpm-major", "9")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); err == nil {
		t.Fatal("m.lock must not be created on nub install")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "nub.lock"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("unchanged graph must preserve nub.lock bytes")
	}
}

func TestLockBridgePnpmInstallPreservesIncumbent(t *testing.T) {
	projDir, cfgPath := setupLockBridgeProject(t, "pnpm-lock.yaml", lodashLockYAML)
	before, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install", "--frozen-lockfile", "--pnpm-major", "9")
	if code != 0 {
		t.Fatalf("install exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.lock")); err == nil {
		t.Fatal("m.lock must not be created on pnpm install")
	}
	after, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("unchanged graph must preserve pnpm-lock.yaml bytes")
	}
}

func TestLockBridgeAmbiguousPnpmWithoutMajorFailsClosed(t *testing.T) {
	projDir, cfgPath := setupLockBridgeProject(t, "pnpm-lock.yaml", lodashLockYAML)
	before, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "lock-bridge-test",
  "version": "1.0.0",
  "private": true,
  "dependencies": {
    "lodash": "^4.17.21"
  }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("expected ambiguous pnpm write failure, out=%s", out)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatal("ambiguous install must not mutate incumbent lock")
	}
}

func TestLockBridgeMigrateDryRunReportsLoss(t *testing.T) {
	projDir, cfgPath := setupLockBridgeProject(t, "nub.lock", lodashLockYAML)
	code, out := runM(t, projDir, cfgPath, "lock", "migrate", "--from", "nub", "--dry-run")
	if code != 0 {
		t.Fatalf("migrate dry-run exit=%d out=%s", code, out)
	}
	var report struct {
		SourceIdentity string `json:"sourceIdentity"`
		LossReport     struct {
			SchemaVersion int `json:"schemaVersion"`
		} `json:"lossReport"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(out)), &report); err != nil {
		t.Fatalf("migration report JSON: %v out=%q", err, out)
	}
	if report.LossReport.SchemaVersion == 0 {
		t.Fatal("expected loss report schemaVersion")
	}
}

func TestLockBridgeValidateIncumbent(t *testing.T) {
	projDir, cfgPath := setupLockBridgeProject(t, "nub.lock", lodashLockYAML)
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	cliRoot.SetOut(buf)
	cliRoot.SetErr(buf)
	cliRoot.SetArgs([]string{"--cwd", projDir, "--config", cfgPath, "lock", "validate", "--frozen", "--json"})
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("validate exit=%d out=%s", code, buf.String())
	}
}
