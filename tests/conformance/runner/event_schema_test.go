package runner_test

import (
	"context"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/runner/envexec"
)

var eventHex64 = regexp.MustCompile(`^[0-9a-f]{64}$`)

func TestEnvironmentPreparedEventGolden(t *testing.T) {
	ev, err := envexec.BuildEnvironmentPreparedEvent(envexec.ExecutionRequest{
		Policy: envexec.ExecutionPolicy{Network: envexec.NetworkForbidden},
	}, sampleEventEnv())
	if err != nil {
		t.Fatal(err)
	}
	if ev.Type != "environment-prepared" || ev.V != 1 {
		t.Fatalf("ev=%+v", ev)
	}
	data, _ := json.Marshal(ev)
	if !strings.Contains(string(data), `"cacheState":"cold-built"`) {
		t.Fatalf("data=%s", data)
	}
}

func TestEnvironmentPreparedDigestFormat(t *testing.T) {
	ev, err := envexec.BuildEnvironmentPreparedEvent(envexec.ExecutionRequest{}, sampleEventEnv())
	if err != nil {
		t.Fatal(err)
	}
	if !eventHex64.MatchString(ev.IdentityDigest) || !eventHex64.MatchString(ev.GraphDigest) {
		t.Fatalf("digests=%s %s", ev.IdentityDigest, ev.GraphDigest)
	}
}

func TestEnvironmentPreparedReporterFailureReleasesLease(t *testing.T) {
	leases := &leaseTracker{}
	env := sampleEventEnv()
	orch := envexec.Orchestrator{Reporter: failRep{}, Leases: leases}
	release, err := orch.AcquireLeaseForTest(t.Context(), env)
	if err != nil {
		t.Fatal(err)
	}
	if err := orch.EmitPreparedForTest(envexec.ExecutionRequest{Policy: envexec.LockedProviderPolicy(envexec.SourceProject)}, env); err == nil {
		t.Fatal("expected reporter error")
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
	env := sampleEventEnv()
	env.Cleanup = func(context.Context) error {
		calls++
		return nil
	}
	orch := envexec.Orchestrator{Reporter: failRep{}}
	if err := orch.EmitPreparedForTest(envexec.ExecutionRequest{Policy: envexec.LockedProviderPolicy(envexec.SourceProject)}, env); err == nil {
		t.Fatal("expected error")
	}
	orch.RunCleanupForTest(env)
	if calls != 1 {
		t.Fatalf("cleanup calls=%d", calls)
	}
}

func sampleEventEnv() envexec.PreparedEnvironment {
	return envexec.PreparedEnvironment{
		Source: envexec.SourceProject,
		Identity: envexec.EnvironmentIdentity{
			SchemaVersion:  envexec.IdentitySchemaVersion,
			Source:         envexec.SourceProject,
			GraphDigest:    strings.Repeat("a", 64),
			MaterialDigest: strings.Repeat("b", 64),
			SourceDigest:   strings.Repeat("c", 64),
			Platform:       envexec.CurrentPlatform(),
			LinkerMode:     "isolated",
		},
		Root:        "/tmp/root",
		NodeModules: "/tmp/root/node_modules",
		CacheState:  envexec.CacheCold,
	}
}

type failRep struct{}

func (failRep) Progress(diagnostics.Event)                   {}
func (failRep) Error(error)                                  {}
func (failRep) Debug(string, ...diagnostics.Attr)            {}
func (failRep) WorkspaceTask(diagnostics.WorkspaceTaskEvent) {}
func (failRep) ChildOutput(diagnostics.ChildOutputEvent, diagnostics.WorkspaceOutputMode) {
}
func (failRep) WorkspaceSummary(diagnostics.WorkspaceSummaryEvent) {}
func (failRep) EnvironmentPrepared(diagnostics.EnvironmentPreparedEvent) error {
	return errRepFail
}
func (failRep) OperationStarted(diagnostics.OperationStartedEvent)     {}
func (failRep) OperationProgress(diagnostics.OperationProgressEvent)   {}
func (failRep) OperationCompleted(diagnostics.OperationCompletedEvent) {}
func (failRep) Notice(diagnostics.NoticeEvent)                         {}

var errRepFail = errors.New("reporter fail")

type leaseTracker struct {
	released bool
}

func (l *leaseTracker) Acquire(context.Context, envexec.EnvironmentIdentity, string, int, int64) (func(), error) {
	return func() { l.released = true }, nil
}
