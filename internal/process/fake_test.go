package process_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/process"
)

type fakeSupervisor struct{}

func (fakeSupervisor) Start(context.Context, process.Spec) (*process.Handle, error) {
	return &process.Handle{}, nil
}
func (fakeSupervisor) Wait(context.Context, *process.Handle) error { return nil }

var _ process.ProcessSupervisor = fakeSupervisor{}

func TestFakeProcessSupervisorSatisfiesInterface(t *testing.T) {
	var s process.ProcessSupervisor = fakeSupervisor{}
	h, err := s.Start(context.Background(), process.Spec{Path: "echo"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Wait(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}
