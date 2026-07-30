package cli

import (
	"testing"

	"github.com/mewisme/mew/internal/config"
)

func TestDirectDispatchBinsGate(t *testing.T) {
	t.Setenv("MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH", "")
	eff := &config.Effective{}
	if DirectDispatchBinsEnabled(eff) {
		t.Fatal("expected disabled")
	}
	t.Setenv("MEW_EXPERIMENTAL_EXEC_DIRECT_DISPATCH", "1")
	if !DirectDispatchBinsEnabled(eff) {
		t.Fatal("expected enabled")
	}
}

func TestExecParserState(t *testing.T) {
	state, _ := execParserState([]string{"bin"})
	if state != execStateDone {
		t.Fatalf("state=%v", state)
	}
	state, _ = execParserState(nil)
	if state != execStateFlags {
		t.Fatalf("state=%v", state)
	}
	state, _ = execParserState([]string{"--package"})
	if state != execStatePackageValue {
		t.Fatalf("state=%v", state)
	}
}
