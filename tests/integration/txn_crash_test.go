//go:build crash

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

func TestTxnCrashMatrixGreenfield(t *testing.T) {
	for _, crashAt := range installCrashBoundaries {
		crashAt := crashAt
		t.Run(crashAt, func(t *testing.T) {
			runGreenfieldCrashAt(t, crashAt, false)
		})
	}
}

func TestTxnCrashInstallWithoutManualRecover(t *testing.T) {
	runGreenfieldCrashAt(t, "publish:0", true)
}

func TestTxnCrashSubprocessUsesEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess crash test")
	}
	projDir := t.TempDir()
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"env-crash","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCrashSubprocess(t, projDir, cfgPath, crashFlowInstall, "publish:0")
	code, out := runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover exit=%d out=%s", code, out)
	}
	if strings.Contains(strings.ToLower(out), "error") && !strings.Contains(strings.ToLower(out), "rolled") {
		t.Logf("recover output: %s", out)
	}
}

func TestTxnCrashRollbackDuringRecover(t *testing.T) {
	if testing.Short() {
		t.Skip("subprocess crash test")
	}
	projDir := t.TempDir()
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	if err := os.WriteFile(filepath.Join(projDir, "package.json"), []byte(`{"name":"rollback","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`), 0o644); err != nil {
		t.Fatal(err)
	}
	cfgPath := filepath.Join(projDir, "m.jsonc")
	if err := os.WriteFile(cfgPath, []byte(`{"registry":"`+srv.URL+`"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runCrashSubprocess(t, projDir, cfgPath, crashFlowInstall, "publish:0")
	exe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in path")
	}
	recoverProg := `
package main
import (
  "context"
  "os"
  "github.com/mewisme/mew/internal/app"
)
func main() {
  ac, err := app.New(context.Background(), app.Options{CWD: os.Getenv("MEW_CWD"), ConfigPath: os.Getenv("MEW_CFG")})
  if err != nil { os.Exit(1) }
  _, err = app.Recover(context.Background(), ac)
  if err != nil { os.Exit(2) }
}
`
	crashDir := filepath.Join(projDir, "crash-probe")
	crashFile := filepath.Join(crashDir, "recover.go")
	if err := os.WriteFile(crashFile, []byte(recoverProg), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(exe, "run", crashFile)
	cmd.Env = append(os.Environ(), "MEW_CWD="+projDir, "MEW_CFG="+cfgPath, "MEW_TXN_CRASH_AT=rollback:0")
	_ = cmd.Run()
	code, out := runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover after rollback crash exit=%d out=%s", code, out)
	}
	assertNoTxnLock(t, projDir, "rollback:0", crashFlowInstall)
	code, out = runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("retry install exit=%d out=%s", code, out)
	}
}
