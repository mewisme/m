package plan_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/m/internal/plan"
)

func TestPlanEncodeStable(t *testing.T) {
	p := &plan.Plan{
		SchemaVersion: plan.SchemaVersion,
		Desired: []plan.DesiredState{
			{PackageKey: "ms@2.1.3"},
			{PackageKey: "left-pad@1.0.0", Integrity: "sha512-aaa"},
		},
		Operations: []plan.Operation{
			{Op: "fetch", Subject: "ms@2.1.3"},
			{Op: "fetch", Subject: "left-pad@1.0.0"},
		},
		Commits: []plan.CommitAction{
			{Op: "write-lock", Subject: "m.lock"},
		},
	}
	first, err := plan.EncodeJSON(p)
	if err != nil {
		t.Fatal(err)
	}
	again, err := plan.DecodeJSON(first)
	if err != nil {
		t.Fatal(err)
	}
	second, err := plan.EncodeJSON(again)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatalf("unstable\n%s\n%s", first, second)
	}
}
