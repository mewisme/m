package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/testkit"
)

func TestTxnCrashAtJournalCreated(t *testing.T) {
	runTxnCrashAt(t, "journal_created")
}

func TestTxnCrashAtBackup(t *testing.T) {
	runTxnCrashAt(t, "backup:0")
}

func TestTxnCrashAtPublish(t *testing.T) {
	runTxnCrashAt(t, "publish:0")
}

func TestTxnCrashAtCommitted(t *testing.T) {
	runTxnCrashAt(t, "committed")
}

func TestTxnCrashAtRollback(t *testing.T) {
	runTxnCrashAt(t, "rollback:0")
}

func TestTxnCrashAtRecovery(t *testing.T) {
	runTxnCrashAt(t, "recovery")
}

func runTxnCrashAt(t *testing.T, crashAt string) {
	t.Helper()
	if testing.Short() {
		t.Skip("txn crash subprocess test")
	}
	projDir := t.TempDir()
	testkit.CleanEnv(t)
	reg := testkit.LoadRegistry(t, "registry/v1")
	srv := reg.Start(t)
	t.Setenv("NO_PROXY", "*")
	pkgJSON := `{"name":"crash-at","version":"1.0.0","dependencies":{"lodash":"4.17.21"}}`
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
  "github.com/mewisme/m/internal/app"
)
func main() {
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
	cmd.Env = append(os.Environ(),
		"MEW_CWD="+projDir,
		"MEW_CFG="+cfgPath,
		"MEW_TXN_CRASH_AT="+crashAt,
	)
	_ = cmd.Run()

	for pass := 0; pass < 2; pass++ {
		code, out := runM(t, projDir, cfgPath, "recover")
		if code != 0 {
			t.Fatalf("recover pass %d exit=%d out=%s crashAt=%s", pass, code, out, crashAt)
		}
	}
	if _, err := os.Stat(transactionLockPath(projDir)); err == nil {
		t.Fatalf("lock leaked after recover for crashAt=%s", crashAt)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("retry install exit=%d out=%s crashAt=%s", code, out, crashAt)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatalf("expected lodash after retry crashAt=%s", crashAt)
	}
}

func transactionLockPath(projDir string) string {
	return filepath.Join(projDir, ".mew", "txn", "lock")
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
	exe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in path")
	}
	crashProg := `
package main
import (
  "context"
  "os"
  "github.com/mewisme/m/internal/app"
)
func main() {
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
	cmd.Env = append(os.Environ(),
		"MEW_CWD="+projDir,
		"MEW_CFG="+cfgPath,
		"MEW_TXN_CRASH_AT=publish:0",
	)
	if err := cmd.Run(); err == nil {
		t.Fatal("expected crash exit")
	}
	code, out := runM(t, projDir, cfgPath, "recover")
	if code != 0 {
		t.Fatalf("recover exit=%d out=%s", code, out)
	}
	if strings.Contains(strings.ToLower(out), "error") && !strings.Contains(strings.ToLower(out), "rolled") {
		t.Logf("recover output: %s", out)
	}
}
