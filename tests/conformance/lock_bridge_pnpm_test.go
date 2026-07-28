package conformance_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"os/exec"
	"path/filepath"
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
	injectPackageManager(t, proj, major)
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

func injectPackageManager(t *testing.T, projDir string, major int) {
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
	doc["packageManager"] = "pnpm@" + strconv.Itoa(major) + ".0.0"
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pkgPath, append(out, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runPnpmFrozen(t *testing.T, projDir string, major int, registryURL string, strictBytes bool) {
	t.Helper()
	pnpm, err := exec.LookPath("pnpm")
	if err != nil {
		t.Skip("pnpm not on PATH")
	}
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
	cmd := exec.Command(pnpm, "install", "--frozen-lockfile", "--lockfile-only", "--ignore-scripts")
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

func setupTestRegistry(t *testing.T, projDir string) string {
	t.Helper()
	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	home := t.TempDir()
	t.Setenv("MEW_HOME", home)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Cleanup(srv.Close)
	cfgDir := filepath.Join(home, ".config", "mew")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := `{"registry":"` + srv.URL + `"}`
	if err := os.WriteFile(filepath.Join(cfgDir, "mew.json"), []byte(cfg+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".npmrc"), []byte("registry="+srv.URL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return srv.URL
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

func runMewAdd(t *testing.T, projDir string, major int, name, version string) {
	t.Helper()
	mutateAddDependency(t, projDir, name, version)
	runMewInstall(t, projDir, major)
}

func testPnpmMutation(t *testing.T, rel string, major int) {
	t.Helper()
	dir, meta := loadGeneratedFixture(t, rel)
	validateFixtureLock(t, dir, major)
	proj := copyFixtureProject(t, dir, major)
	registryURL := setupTestRegistry(t, proj)

	before, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	runMewAdd(t, proj, major, "pkg-a", "1.0.0")
	after, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(before, after) {
		t.Fatal("mutation must change lock bytes")
	}
	runLockValidateCLI(t, proj, major)
	runPnpmFrozen(t, proj, major, registryURL, false)

	runMewInstall(t, proj, major, "--frozen-lockfile")
	after2, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(after, after2) {
		t.Fatal("repeat frozen install must preserve lock bytes")
	}

	transaction.SetTestHook(func(phase string, _ int) error {
		if phase == "commit" {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })
	snap := append([]byte(nil), after2...)
	cliRoot := cli.NewMRoot(cli.BuildInfo{Version: "0.0.0-test"})
	cliRoot.SetOut(io.Discard)
	cliRoot.SetErr(io.Discard)
	cliRoot.SetArgs([]string{"--cwd", proj, "install", "--frozen-lockfile", "--pnpm-major", strconv.Itoa(major)})
	if code := cli.ExecuteWithContext(cliRoot, context.Background()); code == 0 {
		t.Fatal("expected commit failure from test hook")
	}
	restored, err := os.ReadFile(filepath.Join(proj, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snap, restored) {
		t.Fatal("failed txn must restore incumbent lock bytes")
	}
	_ = meta
}

func TestLockBridgePnpm9(t *testing.T) {
	dir, _ := loadGeneratedFixture(t, "pnpm-9/basic")
	validateFixtureLock(t, dir, 9)
	proj := copyFixtureProject(t, dir, 9)
	runLockValidateCLI(t, proj, 9)
	runPnpmFrozen(t, proj, 9, "", true)
}

func TestLockBridgePnpm10(t *testing.T) {
	dir, _ := loadGeneratedFixture(t, "pnpm-10/basic")
	validateFixtureLock(t, dir, 10)
	proj := copyFixtureProject(t, dir, 10)
	runLockValidateCLI(t, proj, 10)
	runPnpmFrozen(t, proj, 10, "", true)
}

func TestLockBridgePnpm11(t *testing.T) {
	dir, _ := loadGeneratedFixture(t, "pnpm-11/basic")
	validateFixtureLock(t, dir, 11)
	proj := copyFixtureProject(t, dir, 11)
	runLockValidateCLI(t, proj, 11)
	runPnpmFrozen(t, proj, 11, "", true)
}

func TestLockBridgePnpm9Mutation(t *testing.T) {
	testPnpmMutation(t, "pnpm-9/basic", 9)
}

func TestLockBridgePnpm10Mutation(t *testing.T) {
	testPnpmMutation(t, "pnpm-10/basic", 10)
}

func TestLockBridgePnpm11Mutation(t *testing.T) {
	testPnpmMutation(t, "pnpm-11/basic", 11)
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
	families := []string{"nub-basic"}
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
			if _, _, err := ext.ReadWithExtensions(context.Background(), lockPath); err != nil {
				t.Fatal(err)
			}
			if meta.LockfileSha256 == "" {
				t.Fatal("metadata missing lockfileSha256")
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
			if !strings.Contains(string(data), "lockfileVersion") {
				t.Fatal("expected lock content")
			}
		})
	}
}
