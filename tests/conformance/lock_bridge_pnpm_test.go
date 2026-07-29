package conformance_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/cli"
	_ "github.com/mewisme/mew/internal/compat/nub"
	_ "github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/lockfile"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

type fixtureMeta struct {
	ProducerMajor   int    `json:"producerMajor"`
	ProducerVersion string `json:"producerVersion"`
	Family          string `json:"family"`
	LockfileSha256  string `json:"lockfileSha256"`
	Command         string `json:"command"`
}

var mutationFamilies = []string{
	"basic", "transitive", "optional", "peer-context", "workspace", "alias", "patch",
}

func moduleRoot(t testing.TB) string {
	t.Helper()
	return testkit.ModuleRoot(t)
}

func loadGeneratedFixture(t *testing.T, rel string) (dir string, meta fixtureMeta) {
	t.Helper()
	dir = filepath.Join(moduleRoot(t), "fixtures", "locks", "generated", rel)
	raw, err := os.ReadFile(filepath.Join(dir, "metadata.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	return dir, meta
}

func validateFixtureLock(t *testing.T, fixtureDir string, major int) {
	t.Helper()
	lockPath := filepath.Join(fixtureDir, "pnpm-lock.yaml")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	det, err := lockfile.DetectPnpmWithMajor(data, major)
	if err != nil {
		t.Fatal(err)
	}
	if major != 0 {
		det.ExplicitMajor = true
	}
	if major != 0 && det.ProducerMajor != major {
		t.Fatalf("major=%d det=%+v", major, det)
	}
	ext, ok := lockfile.ExtAdapterFor(project.IdentityPNPM)
	if !ok {
		t.Fatal("missing pnpm adapter")
	}
	if _, _, err := ext.ReadWithExtensions(context.Background(), lockPath); err != nil {
		t.Fatal(err)
	}
	before := append([]byte(nil), data...)
	res, err := ext.(lockfile.PreservingEncoder).EncodePreserving(context.Background(), lockPath, mustGraph(t, lockPath), data, nil, det)
	if err != nil {
		t.Fatal(err)
	}
	out := before
	if !res.Unchanged {
		out = res.Bytes
	}
	if !bytes.Equal(before, out) {
		t.Fatal("graph-equal no-op must preserve lock bytes")
	}
}

func mustGraph(t *testing.T, lockPath string) *lockfile.Graph {
	t.Helper()
	ext, _ := lockfile.ExtAdapterFor(project.IdentityPNPM)
	g, err := ext.Read(context.Background(), lockPath)
	if err != nil {
		t.Fatal(err)
	}
	return g
}

func runLockValidateCLI(t *testing.T, projDir string, major int) {
	t.Helper()
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	cliRoot.SetOut(buf)
	cliRoot.SetErr(buf)
	args := []string{"--cwd", projDir, "lock", "validate", "--frozen", "--json"}
	if major != 0 {
		args = append(args, "--pnpm-major", strconv.Itoa(major))
	}
	cliRoot.SetArgs(args)
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("validate exit=%d out=%s", code, buf.String())
	}
}

func copyFixtureProject(t *testing.T, fixtureDir string, major int) string {
	t.Helper()
	proj := t.TempDir()
	copyTree(t, fixtureDir, proj)
	injectPackageManager(t, proj, major, "")
	return proj
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if strings.HasSuffix(rel, "metadata.json") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func injectPackageManager(t *testing.T, projDir string, major int, exactVersion string) {
	t.Helper()
	pkgPath := filepath.Join(projDir, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if exactVersion == "" {
		exactVersion = strconv.Itoa(major) + ".0.0"
	}
	doc["packageManager"] = "pnpm@" + exactVersion
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func setupIsolatedPnpmHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("PNPM_HOME", filepath.Join(home, "pnpm"))
	t.Setenv("XDG_DATA_HOME", filepath.Join(home, ".local", "share"))
	t.Setenv("XDG_CACHE_HOME", filepath.Join(home, ".cache"))
	return home
}

func runPnpmFrozen(t *testing.T, projDir string, major int, registryURL string, strictBytes bool) {
	t.Helper()
	pnpm, err := exec.LookPath("pnpm")
	if err != nil {
		t.Skip("pnpm not on PATH")
	}
	setupIsolatedPnpmHome(t)
	before, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	npmrc := ""
	if registryURL != "" {
		npmrc = "registry=" + registryURL + "\n"
	}
	if err := os.WriteFile(filepath.Join(projDir, ".npmrc"), []byte(npmrc), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pnpm, "install", "--frozen-lockfile", "--ignore-scripts")
	cmd.Dir = projDir
	cmd.Env = append(os.Environ(), "CI=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pnpm frozen install (major=%d): %v\n%s", major, err, out)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if strictBytes && !bytes.Equal(before, after) {
		t.Fatalf("pnpm mutated lockfile on frozen install (major=%d)", major)
	}
}

func runMewInstall(t *testing.T, projDir string, major int, extraArgs ...string) {
	t.Helper()
	args := []string{"--cwd", projDir, "install", "--pnpm-major", strconv.Itoa(major)}
	args = append(args, extraArgs...)
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	buf := new(bytes.Buffer)
	cliRoot.SetOut(buf)
	cliRoot.SetErr(buf)
	cliRoot.SetArgs(args)
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code != 0 {
		t.Fatalf("m install exit=%d out=%s", code, buf.String())
	}
}

func mutateAddDependency(t *testing.T, projDir, name, version string) {
	t.Helper()
	pkgPath := filepath.Join(projDir, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	deps, _ := doc["dependencies"].(map[string]any)
	if deps == nil {
		deps = map[string]any{}
	}
	deps[name] = version
	doc["dependencies"] = deps
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mutateRemoveDependency(t *testing.T, projDir, name string) {
	t.Helper()
	pkgPath := filepath.Join(projDir, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	if deps, ok := doc["dependencies"].(map[string]any); ok {
		delete(deps, name)
		doc["dependencies"] = deps
	}
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mutateUpdateDependency(t *testing.T, projDir, name, version string) {
	t.Helper()
	mutateAddDependency(t, projDir, name, version)
}

func runMewAdd(t *testing.T, projDir string, major int, name, version string) {
	t.Helper()
	mutateAddDependency(t, projDir, name, version)
	runMewInstall(t, projDir, major)
}

func stripPackageManager(t *testing.T, projDir string) {
	t.Helper()
	pkgPath := filepath.Join(projDir, "package.json")
	raw, err := os.ReadFile(pkgPath)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	delete(doc, "packageManager")
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func lockHash(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func assertPeerContextGraph(t *testing.T, proj string, major int) {
	t.Helper()
	lockPath := filepath.Join(proj, "pnpm-lock.yaml")
	g := mustGraph(t, lockPath)
	acornVer := "8.18.0"
	if major == 11 {
		acornVer = "8.17.0"
	}
	peerInstance := "acorn-jsx@5.3.2#acorn@" + acornVer
	acornKey := "acorn@" + acornVer
	if !graphHasPackage(g, peerInstance) {
		t.Fatalf("missing peer-context package %q", peerInstance)
	}
	if !graphHasPackage(g, acornKey) {
		t.Fatalf("missing acorn package %q", acornKey)
	}
}

func graphHasPackage(g *lockfile.Graph, id string) bool {
	for _, p := range g.Packages {
		if p.ID.Key() == id {
			return true
		}
	}
	return false
}

func verifyNodeModulesGraph(t *testing.T, proj, family string) {
	t.Helper()
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node not on PATH")
	}
	scripts := importScriptsForFamily(family)
	for _, script := range scripts {
		cmd := exec.Command(node, "-e", script)
		cmd.Dir = proj
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("node import (%s): %v\n%s", family, err, out)
		}
	}
	nm := filepath.Join(proj, "node_modules")
	if _, err := os.Stat(nm); err != nil {
		t.Fatalf("missing node_modules: %v", err)
	}
}

func importScriptsForFamily(family string) []string {
	switch family {
	case "basic":
		return []string{"require('lodash'); console.log('ok')"}
	case "transitive":
		return []string{
			"require('chalk'); require('ansi-styles'); console.log('ok')",
		}
	case "optional":
		return []string{"require('left-pad'); console.log('ok')"}
	case "peer-context":
		return []string{"require('acorn-jsx'); console.log('ok')"}
	case "workspace":
		return []string{"require('pkg-a'); console.log('ok')"}
	case "alias":
		return []string{"require('my-lodash'); console.log('ok')"}
	case "patch":
		return []string{"require('ms'); console.log('ok')"}
	default:
		return []string{"console.log('ok')"}
	}
}

func setupMutationEnv(t *testing.T) {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	t.Setenv("MEW_RESOLVE_AUTO_INSTALL_PEERS", "1")
	home := t.TempDir()
	t.Setenv("MEW_HOME", home)
	setupIsolatedPnpmHome(t)
}

func mutationAddDep(family string) (name, version string) {
	switch family {
	case "patch":
		return "chalk", "4.1.2"
	default:
		return "ms", "2.1.3"
	}
}

func mutationUpdateVersion(name, version string) string {
	switch name {
	case "ms":
		return "2.1.2"
	case "chalk":
		return "4.1.1"
	default:
		return version
	}
}

func testPnpmMutationFamily(t *testing.T, rel string, major int, family string) {
	t.Helper()
	dir, _ := loadGeneratedFixture(t, rel)
	validateFixtureLock(t, dir, major)
	proj := copyFixtureProject(t, dir, major)
	setupMutationEnv(t)

	before := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	addName, addVer := mutationAddDep(family)
	runMewAdd(t, proj, major, addName, addVer)
	afterAdd := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	if afterAdd == before {
		t.Fatal("add mutation must change lock bytes")
	}
	runLockValidateCLI(t, proj, major)
	stripPackageManager(t, proj)
	runPnpmFrozen(t, proj, major, "", true)
	verifyNodeModulesGraph(t, proj, family)

	// update mutation
	mutateUpdateDependency(t, proj, addName, mutationUpdateVersion(addName, addVer))
	runMewInstall(t, proj, major)
	afterUpdate := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	if afterUpdate == afterAdd {
		t.Fatal("update mutation must change lock bytes")
	}
	runLockValidateCLI(t, proj, major)
	stripPackageManager(t, proj)
	runPnpmFrozen(t, proj, major, "", true)

	// remove mutation
	mutateRemoveDependency(t, proj, addName)
	runMewInstall(t, proj, major)
	afterRemove := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	if afterRemove == afterUpdate {
		t.Fatal("remove mutation must change lock bytes")
	}
	runLockValidateCLI(t, proj, major)
	stripPackageManager(t, proj)
	runPnpmFrozen(t, proj, major, "", true)
	verifyNodeModulesGraph(t, proj, family)

	if family == "peer-context" {
		assertPeerContextGraph(t, proj, major)
	}

	// deterministic repeat
	runMewInstall(t, proj, major, "--frozen-lockfile")
	afterRepeat := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	if afterRepeat != afterRemove {
		t.Fatal("repeat frozen install must preserve lock bytes")
	}

	// commit-interrupt restore
	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "commit" {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	snap := afterRepeat
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	cliRoot.SetOut(io.Discard)
	cliRoot.SetErr(io.Discard)
	cliRoot.SetArgs([]string{"--cwd", proj, "install", "--frozen-lockfile", "--pnpm-major", strconv.Itoa(major)})
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code == 0 {
		t.Fatal("expected commit failure from test hook")
	}
	restored := lockHash(t, filepath.Join(proj, "pnpm-lock.yaml"))
	if restored != snap {
		t.Fatal("failed txn must restore incumbent lock bytes")
	}
}

func testPnpmParseFamily(t *testing.T, rel string, major int) {
	t.Helper()
	dir, _ := loadGeneratedFixture(t, rel)
	validateFixtureLock(t, dir, major)
	proj := copyFixtureProject(t, dir, major)
	runLockValidateCLI(t, proj, major)
	setupIsolatedPnpmHome(t)
	runPnpmFrozen(t, proj, major, "", true)
}

func TestLockBridgePnpm9(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-9/basic", 9)
}

func TestLockBridgePnpm9PeerContext(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-9/peer-context", 9)
	dir, _ := loadGeneratedFixture(t, "pnpm-9/peer-context")
	proj := copyFixtureProject(t, dir, 9)
	assertPeerContextGraph(t, proj, 9)
}

func TestLockBridgePnpm10(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-10/basic", 10)
}

func TestLockBridgePnpm10PeerContext(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-10/peer-context", 10)
	dir, _ := loadGeneratedFixture(t, "pnpm-10/peer-context")
	proj := copyFixtureProject(t, dir, 10)
	assertPeerContextGraph(t, proj, 10)
}

func TestLockBridgePnpm11(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-11/basic", 11)
}

func TestLockBridgePnpm11PeerContext(t *testing.T) {
	testPnpmParseFamily(t, "pnpm-11/peer-context", 11)
	dir, _ := loadGeneratedFixture(t, "pnpm-11/peer-context")
	proj := copyFixtureProject(t, dir, 11)
	assertPeerContextGraph(t, proj, 11)
}

func TestLockBridgePnpm9MutationSuite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mutation suite uses isolated pnpm store; run on Linux CI")
	}
	for _, family := range mutationFamilies {
		t.Run(family, func(t *testing.T) {
			testPnpmMutationFamily(t, fmt.Sprintf("pnpm-9/%s", family), 9, family)
		})
	}
}

func TestLockBridgePnpm10MutationSuite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mutation suite uses isolated pnpm store; run on Linux CI")
	}
	for _, family := range mutationFamilies {
		t.Run(family, func(t *testing.T) {
			testPnpmMutationFamily(t, fmt.Sprintf("pnpm-10/%s", family), 10, family)
		})
	}
}

func TestLockBridgePnpm11MutationSuite(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("mutation suite uses isolated pnpm store; run on Linux CI")
	}
	for _, family := range mutationFamilies {
		t.Run(family, func(t *testing.T) {
			testPnpmMutationFamily(t, fmt.Sprintf("pnpm-11/%s", family), 11, family)
		})
	}
}

func TestLockBridgePnpmUnsupportedLegacy(t *testing.T) {
	lockPath := filepath.Join(moduleRoot(t), "fixtures", "locks", "pnpm", "unsupported", "v6", "pnpm-lock.yaml")
	data, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	_, err = lockfile.DetectPnpm(data)
	if err == nil {
		t.Fatal("expected legacy rejection")
	}
	ext, ok := lockfile.ExtAdapterFor(project.IdentityPNPM)
	if !ok {
		t.Fatal("missing adapter")
	}
	proj := t.TempDir()
	if err := os.WriteFile(filepath.Join(proj, "package.json"), []byte(`{"name":"x","version":"1.0.0"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proj, "pnpm-lock.yaml"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ext.Read(context.Background(), filepath.Join(proj, "pnpm-lock.yaml")); err == nil {
		t.Fatal("expected adapter read rejection for v6 lock")
	}
}

func copyNubFixtureProject(t *testing.T, fixtureDir string) string {
	t.Helper()
	proj := t.TempDir()
	copyTree(t, fixtureDir, proj)
	return proj
}

func TestLockBridgeNubFixtures(t *testing.T) {
	families := []string{
		"nub-basic", "nub-transitive", "nub-workspace",
		"nub-catalog", "nub-peer", "nub-optional",
	}
	// ponytail: workspace derived fixture validated after workspace graph support.
	skipValidate := map[string]bool{}
	for _, family := range families {
		t.Run(family, func(t *testing.T) {
			dir, meta := loadGeneratedFixture(t, family)
			lockPath := filepath.Join(dir, "nub.lock")
			data, err := os.ReadFile(lockPath)
			if err != nil {
				t.Fatal(err)
			}
			ext, ok := lockfile.ExtAdapterFor(project.IdentityNub)
			if !ok {
				t.Fatal("missing nub adapter")
			}
			if meta.LockfileSha256 == "" {
				t.Fatal("metadata missing lockfileSha256")
			}
			if !strings.Contains(string(data), "lockfileVersion") {
				t.Fatal("expected lock content")
			}
			if skipValidate[family] {
				t.Log("derived-format evidence: metadata + lock bytes only (workspace link edges)")
				return
			}
			if _, _, err := ext.ReadWithExtensions(context.Background(), lockPath); err != nil {
				t.Fatal(err)
			}
			proj := copyNubFixtureProject(t, dir)
			cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
			buf := new(bytes.Buffer)
			cliRoot.SetOut(buf)
			cliRoot.SetErr(buf)
			cliRoot.SetArgs([]string{"--cwd", proj, "lock", "validate", "--json"})
			if code := cli.ExecuteWithContext(cliRoot, context.Background()); code != 0 {
				t.Fatalf("validate exit=%d out=%s", code, buf.String())
			}
		})
	}
}
