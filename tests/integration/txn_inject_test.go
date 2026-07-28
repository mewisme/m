package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
	"github.com/mewisme/mew/internal/transaction"
)

func TestTxnInjectMidCommitPreservesPriorState(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "txn-inject",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	lockBefore, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
	if err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "publish" && opIndex == 0 {
			return os.ErrPermission
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("expected install failure, out=%s", out)
	}
	if lockBefore != nil {
		after, err := os.ReadFile(filepath.Join(projDir, "m.lock"))
		if err != nil {
			t.Fatal(err)
		}
		if string(after) != string(lockBefore) {
			t.Fatal("lock changed after failed commit")
		}
	}
	code, out = runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover exit=%d out=%s", code, out)
	}
	code, out = runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("second recover exit=%d out=%s", code, out)
	}
}

func TestTxnInjectRetrySucceeds(t *testing.T) {
	projDir, cfgPath, _ := setupRegistryProject(t, `{
  "name": "txn-retry",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21" }
}`)
	var attempts int
	transaction.SetTestHook(func(phase string, opIndex int) error {
		if phase == "publish" && opIndex == 0 {
			attempts++
			if attempts == 1 {
				return os.ErrPermission
			}
		}
		return nil
	})
	t.Cleanup(func() { transaction.SetTestHook(nil) })

	code, out := runM(t, projDir, cfgPath, "install")
	if code == 0 {
		t.Fatalf("first install should fail, out=%s", out)
	}
	_, _ = runM(t, projDir, cfgPath, "recover")
	code, out = runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("retry install exit=%d out=%s", code, out)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatal("expected lodash linked after retry")
	}
}

func TestTxnCrashSubprocessMidCommit(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess crash test")
	}
	projDir := t.TempDir()
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	pkgJSON := `{"name":"crash","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(pkgJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	exe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in path")
	}
	crashProg := `
package main
import (
  "context"
  "os"
  "github.com/mewisme/mew/internal/app"
  "github.com/mewisme/mew/internal/transaction"
)
func main() {
  transaction.SetTestHook(func(phase string, opIndex int) error {
    if phase == "publish" && opIndex == 0 { os.Exit(3) }
    return nil
  })
  ac, err := app.New(context.Background(), app.Options{CWD: os.Getenv("MEW_CWD"), ConfigPath: os.Getenv("MEW_CFG")})
  if err != nil { os.Exit(1) }
  _, err = app.Install(context.Background(), ac, app.InstallOptions{})
  if err != nil { os.Exit(2) }
}
`
	crashDir := filepath.Join(projDir, "crash-probe")
	if err := os.MkdirAll(crashDir, 0o755); err != nil {
		t.Fatal(err)
	}
	crashFile := filepath.Join(crashDir, "main.go")
	if err := os.WriteFile(crashFile, []byte(crashProg), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "run", crashFile)
	cmd.Env = append(os.Environ(), "MEW_CWD="+projDir, "MEW_CFG="+cfgPath)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected crash exit")
	}
	code, out := runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover after crash exit=%d out=%s", code, out)
	}
	if strings.Contains(out, "error") && !strings.Contains(strings.ToLower(out), "rolled") {
		t.Logf("recover output: %s", out)
	}
}
