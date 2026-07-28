//go:build crash

package integration_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/mewisme/mew/internal/testkit"
)

// installCrashBoundaries are subprocess crash injection points for install-family flows.
var installCrashBoundaries = []string{
	"journal_created",
	"post_resolve",
	"post_fetch",
	"post_link",
	"post_lockfile",
	"post_validate",
	"backup:0",
	"backup:1",
	"backup:2",
	"publish:0",
	"publish:1",
	"publish:2",
	"commit:0",
	"commit:1",
	"pre_committed",
	"committed",
	"finish",
	"recovery",
}

type crashFlow string

const (
	crashFlowInstall crashFlow = "install"
	crashFlowUpdate  crashFlow = "update"
	crashFlowRestore crashFlow = "restore"
)

type crashScenario struct {
	flow        crashFlow
	crashAt     string
	prepare     func(t *testing.T, projDir, cfgPath string)
	skipRecover bool
	assertOK    func(t *testing.T, projDir, cfgPath string)
}

func runCrashScenario(t *testing.T, sc crashScenario) {
	t.Helper()
	if testing.Short() {
		t.Skip("subprocess crash test")
	}
	projDir, cfgPath := prepareCrashProject(t, sc.flow, sc.prepare)
	runCrashSubprocess(t, projDir, cfgPath, sc.flow, sc.crashAt)

	if !sc.skipRecover {
		for pass := 0; pass < 2; pass++ {
			code, out := runM(t, projDir, cfgPath, "recover")
			if code != 0 {
				t.Fatalf("recover pass %d exit=%d out=%s crashAt=%s flow=%s", pass, code, out, sc.crashAt, sc.flow)
			}
		}
		assertNoTxnLock(t, projDir, sc.crashAt, sc.flow)
	}

	switch sc.flow {
	case crashFlowRestore:
		code, out := runM(t, projDir, cfgPath, "snapshot", "restore", "000001")
		if code != 0 {
			t.Fatalf("retry restore exit=%d out=%s crashAt=%s", code, out, sc.crashAt)
		}
	default:
		code, out := runM(t, projDir, cfgPath, string(sc.flow))
		if code != 0 {
			t.Fatalf("retry %s exit=%d out=%s crashAt=%s", sc.flow, code, out, sc.crashAt)
		}
	}

	if sc.skipRecover {
		assertNoTxnLock(t, projDir, sc.crashAt, sc.flow)
	}

	if sc.assertOK != nil {
		sc.assertOK(t, projDir, cfgPath)
	} else {
		assertDefaultCrashOutcome(t, projDir, sc.flow)
	}
}

func prepareCrashProject(t *testing.T, flow crashFlow, prepare func(t *testing.T, projDir, cfgPath string)) (projDir, cfgPath string) {
	t.Helper()
	projDir, cfgPath, _ = setupRegistryProject(t, `{
  "name": "crash-matrix",
  "version": "1.0.0",
  "dependencies": { "lodash": "4.17.21", "pkg-a": "^1.0.0" }
}`)
	if code, out := runM(t, projDir, cfgPath, "install"); code != 0 {
		t.Fatalf("seed install exit=%d out=%s", code, out)
	}
	switch flow {
	case crashFlowUpdate:
		if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
			t.Fatalf("seed add exit=%d out=%s", code, out)
		}
	case crashFlowRestore:
		if code, out := runM(t, projDir, cfgPath, "add", "pkg-c"); code != 0 {
			t.Fatalf("seed add exit=%d out=%s", code, out)
		}
	}
	if prepare != nil {
		prepare(t, projDir, cfgPath)
	}
	return projDir, cfgPath
}

func runCrashSubprocess(t *testing.T, projDir, cfgPath string, flow crashFlow, crashAt string) {
	t.Helper()
	exe, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go not in path")
	}
	crashProg := crashSubprocessSource(flow)
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
}

func crashSubprocessSource(flow crashFlow) string {
	switch flow {
	case crashFlowUpdate:
		return `
package main
import (
  "context"
  "os"
  "github.com/mewisme/mew/internal/app"
)
func main() {
  ac, err := app.New(context.Background(), app.Options{CWD: os.Getenv("MEW_CWD"), ConfigPath: os.Getenv("MEW_CFG")})
  if err != nil { os.Exit(1) }
  _, err = app.Update(context.Background(), ac, app.UpdateOptions{Targets: []string{"pkg-a"}})
  if err != nil { os.Exit(2) }
}
`
	case crashFlowRestore:
		return `
package main
import (
  "context"
  "os"
  "github.com/mewisme/mew/internal/app"
)
func main() {
  ac, err := app.New(context.Background(), app.Options{CWD: os.Getenv("MEW_CWD"), ConfigPath: os.Getenv("MEW_CFG")})
  if err != nil { os.Exit(1) }
  _, err = app.RestoreSnapshot(context.Background(), ac, "000001")
  if err != nil { os.Exit(2) }
}
`
	default:
		return `
package main
import (
  "context"
  "os"
  "github.com/mewisme/mew/internal/app"
)
func main() {
  ac, err := app.New(context.Background(), app.Options{CWD: os.Getenv("MEW_CWD"), ConfigPath: os.Getenv("MEW_CFG")})
  if err != nil { os.Exit(1) }
  _, err = app.Install(context.Background(), ac, app.InstallOptions{})
  if err != nil { os.Exit(2) }
}
`
	}
}

func assertNoTxnLock(t *testing.T, projDir, crashAt string, flow crashFlow) {
	t.Helper()
	if _, err := os.Stat(transactionLockPath(projDir)); err == nil {
		t.Fatalf("lock leaked after recover crashAt=%s flow=%s", crashAt, flow)
	}
}

func assertDefaultCrashOutcome(t *testing.T, projDir string, flow crashFlow) {
	t.Helper()
	switch flow {
	case crashFlowRestore:
		if hasDirectDep(t, projDir, "pkg-c") {
			t.Fatal("pkg-c should be removed after restore")
		}
		if _, err := os.Stat(filepath.Join(projDir, "node_modules", "pkg-a", "package.json")); err != nil {
			t.Fatal("pkg-a should be linked after restore retry")
		}
	case crashFlowUpdate:
		if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
			t.Fatal("expected lodash after update retry")
		}
	default:
		if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
			t.Fatal("expected lodash after install retry")
		}
	}
}

func runGreenfieldCrashAt(t *testing.T, crashAt string, skipRecover bool) {
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
	runCrashSubprocess(t, projDir, cfgPath, crashFlowInstall, crashAt)
	if !skipRecover {
		for pass := 0; pass < 2; pass++ {
			code, out := runM(t, projDir, cfgPath, "recover")
			if code != 0 {
				t.Fatalf("recover pass %d exit=%d out=%s crashAt=%s", pass, code, out, crashAt)
			}
		}
		assertNoTxnLock(t, projDir, crashAt, crashFlowInstall)
	}
	code, out := runM(t, projDir, cfgPath, "install")
	if code != 0 {
		t.Fatalf("retry install exit=%d out=%s crashAt=%s", code, out, crashAt)
	}
	if skipRecover {
		assertNoTxnLock(t, projDir, crashAt, crashFlowInstall)
	}
	if _, err := os.Stat(filepath.Join(projDir, "node_modules", "lodash", "package.json")); err != nil {
		t.Fatalf("expected lodash after retry crashAt=%s", crashAt)
	}
}

func transactionLockPath(projDir string) string {
	return filepath.Join(projDir, ".mew", "txn", "lock")
}
