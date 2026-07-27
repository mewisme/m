package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/mewisme/m/internal/app"
	"github.com/mewisme/m/internal/testkit"
)

const mutationConfigWaitEnv = "MEW_MUTATION_CONFIG_WAIT_PROC"

func TestAddWaitsOnLockAndUsesReloadedScopedRegistry(t *testing.T) {
	if role := os.Getenv(mutationConfigWaitEnv); role != "" {
		runMutationConfigWaitChild(t, role)
		return
	}
	if testing.Short() {
		t.Skip("config-wait proc test")
	}

	testkit.CleanEnv(t)
	t.Setenv("NO_PROXY", "*")
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	defer srv.Close()

	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "config-wait",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	deadRegistry := `{"registry":"http://127.0.0.1:1"}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(deadRegistry), 0o644); err != nil {
		t.Fatal(err)
	}

	ready := filepath.Join(projDir, ".holder-ready")
	release := filepath.Join(projDir, ".holder-release")
	_ = os.Remove(ready)
	_ = os.Remove(release)

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command(exe, "-test.run=^TestAddWaitsOnLockAndUsesReloadedScopedRegistry$", "-test.count=1")
	holder.Env = append(os.Environ(),
		mutationConfigWaitEnv+"=holder",
		"MEW_CONFIG_WAIT_PROJ="+projDir,
		"MEW_CONFIG_WAIT_CFG="+cfgPath,
		"MEW_CONFIG_WAIT_READY="+ready,
		"MEW_CONFIG_WAIT_RELEASE="+release,
	)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitForFile(ready, 10*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}

	liveCfg := `{
  "registry": "` + srv.URL + `",
  "registries": { "@scope": "` + srv.URL + `" }
}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(liveCfg), 0o644); err != nil {
		t.Fatal(err)
	}

	adderDone := make(chan error, 1)
	go func() {
		cmd := exec.Command(exe, "-test.run=^TestAddWaitsOnLockAndUsesReloadedScopedRegistry$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			mutationConfigWaitEnv+"=adder",
			"MEW_CONFIG_WAIT_PROJ="+projDir,
			"MEW_CONFIG_WAIT_CFG="+cfgPath,
		)
		adderDone <- cmd.Run()
	}()

	if err := waitForFile(filepath.Join(projDir, ".adder-started"), 10*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}
	if err := os.WriteFile(release, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := holder.Wait(); err != nil {
		t.Fatalf("holder exit: %v", err)
	}
	if err := <-adderDone; err != nil {
		t.Fatalf("adder exit: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "@scope", "pkg", "package.json")); err != nil {
		t.Fatalf("scoped package not linked: %v", err)
	}
}

func runMutationConfigWaitChild(t *testing.T, role string) {
	t.Helper()
	projDir := os.Getenv("MEW_CONFIG_WAIT_PROJ")
	cfgPath := os.Getenv("MEW_CONFIG_WAIT_CFG")
	if projDir == "" || cfgPath == "" {
		t.Fatal("missing project env")
	}
	switch role {
	case "holder":
		ready := os.Getenv("MEW_CONFIG_WAIT_READY")
		release := os.Getenv("MEW_CONFIG_WAIT_RELEASE")
		if ready == "" || release == "" {
			t.Fatal("missing holder sync env")
		}
		ctx := context.Background()
		ac, err := app.New(ctx, app.Options{CWD: projDir, ConfigPath: cfgPath})
		if err != nil {
			t.Fatal(err)
		}
		sess, err := app.BeginMutationSession(ctx, ac, projDir)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := waitForFile(release, 30*time.Second); err != nil {
			_, _ = sess.Abort(ctx)
			t.Fatal(err)
		}
		if _, err := sess.Abort(ctx); err != nil {
			t.Fatal(err)
		}
	case "adder":
		started := filepath.Join(projDir, ".adder-started")
		if err := os.WriteFile(started, []byte("go"), 0o644); err != nil {
			t.Fatal(err)
		}
		code, out := runM(t, projDir, cfgPath, "add", "@scope/pkg")
		if code != 0 {
			t.Fatalf("add @scope/pkg exit=%d out=%s", code, out)
		}
	default:
		t.Fatalf("unknown role %q", role)
	}
}

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return os.ErrNotExist
}
