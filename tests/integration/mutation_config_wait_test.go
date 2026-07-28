package integration_test

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/testkit"
)

const mutationConfigWaitEnv = "MEW_MUTATION_CONFIG_WAIT_PROC"

func TestAddWaitsOnLockAndUsesReloadedScopedRegistry(t *testing.T) {
	if role := os.Getenv(mutationConfigWaitEnv); role != "" {
		runMutationConfigWaitChild(t, role)
		return
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
	cfgPath := filepath.Join(projDir, "custom.jsonc")
	deadRegistry := `{"registry":"http://127.0.0.1:1"}` + "\n"
	if err := os.WriteFile(cfgPath, []byte(deadRegistry), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(projDir, "m.jsonc")); err == nil {
		t.Fatal("default m.jsonc should not exist")
	}

	holderReady := filepath.Join(projDir, ".holder-ready")
	adderContextReady := filepath.Join(projDir, ".adder-context-ready")
	adderLockWait := filepath.Join(projDir, ".adder-lock-wait")
	holderRelease := filepath.Join(projDir, ".holder-release")
	for _, p := range []string{holderReady, adderContextReady, adderLockWait, holderRelease} {
		_ = os.Remove(p)
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command(exe, "-test.run=^TestAddWaitsOnLockAndUsesReloadedScopedRegistry$", "-test.count=1")
	holder.Env = append(os.Environ(),
		mutationConfigWaitEnv+"=holder",
		"MEW_CONFIG_WAIT_PROJ="+projDir,
		"MEW_CONFIG_WAIT_CFG="+cfgPath,
		"MEW_CONFIG_WAIT_READY="+holderReady,
		"MEW_CONFIG_WAIT_RELEASE="+holderRelease,
	)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitForFile(holderReady, 15*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
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

	if err := waitForFile(adderContextReady, 15*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}
	if err := waitForFile(adderLockWait, 15*time.Second); err != nil {
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
	if err := os.WriteFile(holderRelease, []byte("go"), 0o644); err != nil {
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
	if _, err := os.Stat(filepath.Join(projDir, "m.jsonc")); err == nil {
		t.Fatal("default m.jsonc should remain absent")
	}
}

func TestAddFailsOnMalformedConfigDuringLockWait(t *testing.T) {
	if role := os.Getenv(mutationConfigWaitEnv); role != "" {
		runMutationConfigWaitChild(t, role)
		return
	}

	testkit.CleanEnv(t)
	projDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{
  "name": "config-wait-bad",
  "version": "1.0.0"
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "custom.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"http://127.0.0.1:1"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	holderReady := filepath.Join(projDir, ".holder-ready")
	adderContextReady := filepath.Join(projDir, ".adder-context-ready")
	adderLockWait := filepath.Join(projDir, ".adder-lock-wait")
	holderRelease := filepath.Join(projDir, ".holder-release")
	for _, p := range []string{holderReady, adderContextReady, adderLockWait, holderRelease} {
		_ = os.Remove(p)
	}

	exe, err := exec.LookPath(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	holder := exec.Command(exe, "-test.run=^TestAddFailsOnMalformedConfigDuringLockWait$", "-test.count=1")
	holder.Env = append(os.Environ(),
		mutationConfigWaitEnv+"=holder",
		"MEW_CONFIG_WAIT_PROJ="+projDir,
		"MEW_CONFIG_WAIT_CFG="+cfgPath,
		"MEW_CONFIG_WAIT_READY="+holderReady,
		"MEW_CONFIG_WAIT_RELEASE="+holderRelease,
	)
	if err := holder.Start(); err != nil {
		t.Fatal(err)
	}
	if err := waitForFile(holderReady, 15*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}

	adderDone := make(chan error, 1)
	go func() {
		cmd := exec.Command(exe, "-test.run=^TestAddFailsOnMalformedConfigDuringLockWait$", "-test.count=1")
		cmd.Env = append(os.Environ(),
			mutationConfigWaitEnv+"=adder-bad",
			"MEW_CONFIG_WAIT_PROJ="+projDir,
			"MEW_CONFIG_WAIT_CFG="+cfgPath,
		)
		adderDone <- cmd.Run()
	}()

	if err := waitForFile(adderContextReady, 15*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}
	if err := waitForFile(adderLockWait, 15*time.Second); err != nil {
		_ = holder.Process.Kill()
		_ = holder.Wait()
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte(`{not-json`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(holderRelease, []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := holder.Wait(); err != nil {
		t.Fatalf("holder exit: %v", err)
	}
	if err := <-adderDone; err != nil {
		t.Fatalf("adder subprocess failed: %v", err)
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
		runConfigWaitAdder(t, projDir, cfgPath, false)
	case "adder-bad":
		runConfigWaitAdder(t, projDir, cfgPath, true)
	default:
		t.Fatalf("unknown role %q", role)
	}
}

func runConfigWaitAdder(t *testing.T, projDir, cfgPath string, expectFail bool) {
	t.Helper()
	ctx := context.Background()
	ac, err := app.New(ctx, app.Options{CWD: projDir, ConfigPath: cfgPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".adder-context-ready"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projDir, ".adder-lock-wait"), []byte("go"), 0o644); err != nil {
		t.Fatal(err)
	}
	sess, err := app.BeginMutationSession(ctx, ac, projDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	if expectFail {
		_, err := sess.ReopenProject(ctx)
		if err == nil {
			t.Fatal("expected config error")
		}
		if apperr.CodeOf(err) != apperr.Config {
			t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
		}
		if strings.Contains(err.Error(), "127.0.0.1:1") {
			t.Fatalf("should fail before registry fetch: %v", err)
		}
		return
	}

	_, err = app.AddInSession(ctx, sess, "@scope/pkg", app.AddOptions{})
	if err != nil {
		t.Fatalf("add failed: %v", err)
	}
	if _, err := sess.Finish(ctx, false); err != nil {
		t.Fatalf("finish: %v", err)
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
