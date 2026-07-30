package envexec

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
)

// ErrReporterFail is a sentinel error for reporter failure tests.
var ErrReporterFail = errors.New("reporter fail")

// BuildEnvironmentPreparedEventForTest is an alias for conformance tests in-package.
func BuildEnvironmentPreparedEventForTest(req ExecutionRequest, env PreparedEnvironment) (diagnostics.EnvironmentPreparedEvent, error) {
	return BuildEnvironmentPreparedEvent(req, env)
}

type failReporter struct{}

func (failReporter) Progress(diagnostics.Event)                   {}
func (failReporter) Error(error)                                  {}
func (failReporter) Debug(string, ...diagnostics.Attr)            {}
func (failReporter) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (failReporter) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (failReporter) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (failReporter) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return ErrReporterFail
}
func (failReporter) OperationStarted(diagnostics.OperationStartedEvent)     {}
func (failReporter) OperationProgress(diagnostics.OperationProgressEvent)   {}
func (failReporter) OperationCompleted(diagnostics.OperationCompletedEvent) {}
func (failReporter) Notice(diagnostics.NoticeEvent)                         {}

type trackingLease struct {
	released bool
}

func (t *trackingLease) Acquire(context.Context, EnvironmentIdentity, string, int, int64) (func(), error) {
	return func() { t.released = true }, nil
}

func samplePreparedEnv() PreparedEnvironment {
	return PreparedEnvironment{
		Source: SourceProject,
		Identity: EnvironmentIdentity{
			SchemaVersion:  IdentitySchemaVersion,
			Source:         SourceProject,
			GraphDigest:    strings.Repeat("a", 64),
			MaterialDigest: strings.Repeat("b", 64),
			SourceDigest:   strings.Repeat("c", 64),
			Platform:       CurrentPlatform(),
			LinkerMode:     "isolated",
		},
		Root:        "/tmp/root",
		NodeModules: "/tmp/root/node_modules",
		CacheState:  CacheCold,
	}
}

func TestBuildEnvironmentPreparedEventRequiresDigests(t *testing.T) {
	_, err := buildEnvironmentPreparedEvent(ExecutionRequest{}, PreparedEnvironment{Source: SourceProject}, 0)
	if err == nil {
		t.Fatal("expected digest error")
	}
}

func TestEnvironmentPreparedReporterFailureReleasesLease(t *testing.T) {
	leases := &trackingLease{}
	env := samplePreparedEnv()
	orch := Orchestrator{Reporter: failReporter{}, Leases: leases}
	release, err := orch.acquireLease(t.Context(), env)
	if err != nil {
		t.Fatal(err)
	}
	if err := orch.emitPrepared(ExecutionRequest{Policy: LockedProviderPolicy(SourceProject)}, env); err == nil {
		t.Fatal("expected reporter error")
	} else if !errors.Is(err, ErrReporterFail) {
		t.Fatalf("err=%v", err)
	}
	if release != nil {
		release()
	}
	if !leases.released {
		t.Fatal("lease not released")
	}
}

func TestEnvironmentPreparedBrokenPipeReleasesLease(t *testing.T) {
	TestEnvironmentPreparedReporterFailureReleasesLease(t)
}

func TestEnvironmentPreparedReporterFailureRunsCleanup(t *testing.T) {
	calls := 0
	env := samplePreparedEnv()
	env.Cleanup = func(context.Context) error {
		calls++
		return nil
	}
	orch := Orchestrator{Reporter: failReporter{}}
	if err := orch.emitPrepared(ExecutionRequest{Policy: LockedProviderPolicy(SourceProject)}, env); err == nil {
		t.Fatal("expected error")
	}
	orch.runCleanup(env)
	if calls != 1 {
		t.Fatalf("cleanup calls=%d", calls)
	}
}

// TestRunCleanupIdempotent verifies cleanup can run multiple times.
func TestRunCleanupIdempotent(t *testing.T) {
	calls := 0
	env := PreparedEnvironment{
		Cleanup: func(context.Context) error {
			calls++
			return nil
		},
	}
	orch := Orchestrator{}
	orch.runCleanup(env)
	orch.runCleanup(env)
	if calls != 2 {
		t.Fatalf("calls=%d", calls)
	}
}
