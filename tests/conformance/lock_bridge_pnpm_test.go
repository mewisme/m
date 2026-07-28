package conformance_test

import (
	"bytes"
	"context"
	"encoding/json"
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
)

type fixtureMeta struct {
	ProducerMajor   int    `json:"producerMajor"`
	ProducerVersion string `json:"producerVersion"`
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
	if major != 0 && det.ProducerMajor != major && det.Format != "pnpm-v6" {
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

func copyFixtureProject(t *testing.T, fixtureDir string) string {
	t.Helper()
	proj := t.TempDir()
	for _, name := range []string{"package.json", "pnpm-lock.yaml", "nub.lock"} {
		src := filepath.Join(fixtureDir, name)
		if _, err := os.Stat(src); err != nil {
			continue
		}
		data, err := os.ReadFile(src)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(proj, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return proj
}

func runPnpmFrozen(t *testing.T, projDir string, major int) {
	t.Helper()
	pnpm, err := exec.LookPath("pnpm")
	if err != nil {
		t.Skip("pnpm not on PATH")
	}
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Cleanup(srv.Close)
	if err := os.WriteFile(filepath.Join(projDir, ".npmrc"), []byte("registry="+srv.URL+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(pnpm, "install", "--frozen-lockfile")
	cmd.Dir = projDir
	cmd.Env = append(os.Environ(), "CI=true")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pnpm frozen install: %v\n%s", err, out)
	}
	after, err := os.ReadFile(filepath.Join(projDir, "pnpm-lock.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Fatalf("pnpm mutated lockfile on frozen install (major=%d)", major)
	}
}

func TestLockBridgePnpm9(t *testing.T) {
	dir, meta := loadGeneratedFixture(t, "pnpm-9/basic")
	validateFixtureLock(t, dir, 9)
	proj := copyFixtureProject(t, dir)
	runLockValidateCLI(t, proj, 9)
	runPnpmFrozen(t, proj, 9)
	_ = meta
}

func TestLockBridgePnpm10(t *testing.T) {
	dir, _ := loadGeneratedFixture(t, "pnpm-10/basic")
	validateFixtureLock(t, dir, 10)
	proj := copyFixtureProject(t, dir)
	runLockValidateCLI(t, proj, 10)
	runPnpmFrozen(t, proj, 10)
}

func TestLockBridgePnpm11(t *testing.T) {
	dir, _ := loadGeneratedFixture(t, "pnpm-11/basic")
	validateFixtureLock(t, dir, 11)
	proj := copyFixtureProject(t, dir)
	runLockValidateCLI(t, proj, 11)
	runPnpmFrozen(t, proj, 11)
}

func TestLockBridgeNubFixtures(t *testing.T) {
	dir, meta := loadGeneratedFixture(t, "nub-basic")
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
	proj := copyFixtureProject(t, dir)
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
}
