package runner_test

import (
	"context"
	"testing"

	"github.com/mewisme/mew/internal/runner"
)

type fakeRunner struct{}

func (fakeRunner) Run(context.Context, runner.RunOptions) (runner.RunResult, error) {
	return runner.RunResult{}, nil
}

var _ runner.ScriptRunner = fakeRunner{}

func TestFakeScriptRunnerSatisfiesInterface(t *testing.T) {
	var r runner.ScriptRunner = fakeRunner{}
	if _, err := r.Run(context.Background(), runner.RunOptions{Selector: "test"}); err != nil {
		t.Fatal(err)
	}
}
