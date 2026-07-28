package runner_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/runner"
)

type fakeRunner struct{}

func (fakeRunner) Run(context.Context, string, runner.ScriptEnv) error { return nil }

var _ runner.ScriptRunner = fakeRunner{}

func TestFakeScriptRunnerSatisfiesInterface(t *testing.T) {
	var r runner.ScriptRunner = fakeRunner{}
	if err := r.Run(context.Background(), "test", runner.ScriptEnv{}); err != nil {
		t.Fatal(err)
	}
}
