package process_test

import (
	"context"
	"os"
	"testing"

	"github.com/mewisme/m/internal/process"
)

func TestExecSupervisorEcho(t *testing.T) {
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	spec := process.Spec{
		Path: "echo hello",
		Dir:  dir,
		Env:  process.RestrictedEnv(os.Environ(), dir),
	}
	h, err := sup.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := sup.Wait(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}
