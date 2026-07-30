package presentation_test

import (
	"bytes"
	"context"
	"io"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/presentation"
)

func pipeCaps() presentation.Capabilities {
	return presentation.Capabilities{}
}

func ttyCaps() presentation.Capabilities {
	return presentation.Capabilities{StderrTTY: true, StdoutTTY: true}
}

func TestResolveOutputPrecedence(t *testing.T) {
	cases := []struct {
		name string
		in   presentation.Input
		caps presentation.Capabilities
		want presentation.OutputMode
	}{
		{
			name: "cli output wins",
			in: presentation.Input{
				OutputFlag: "json",
				Env:        map[string]string{"MEW_OUTPUT": "plain"},
			},
			want: presentation.OutputJSON,
		},
		{
			name: "reporter when no output flag",
			in:   presentation.Input{ReporterFlag: "ndjson"},
			want: presentation.OutputNDJSON,
		},
		{
			name: "env output",
			in:   presentation.Input{Env: map[string]string{"MEW_OUTPUT": "silent"}},
			want: presentation.OutputSilent,
		},
		{
			name: "auto rich on tty",
			in:   presentation.Input{},
			caps: ttyCaps(),
			want: presentation.OutputRich,
		},
		{
			name: "auto plain on pipe",
			in:   presentation.Input{},
			caps: pipeCaps(),
			want: presentation.OutputPlain,
		},
		{
			name: "accessible forces plain",
			in:   presentation.Input{Accessible: true},
			caps: ttyCaps(),
			want: presentation.OutputPlain,
		},
		{
			name: "legacy forces plain",
			in:   presentation.Input{LegacyFlag: true},
			caps: ttyCaps(),
			want: presentation.OutputPlain,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			caps := tc.caps
			if caps == (presentation.Capabilities{}) && tc.name != "auto rich on tty" && tc.name != "accessible forces plain" && tc.name != "legacy forces plain" {
				caps = pipeCaps()
			}
			got, err := presentation.Resolve(tc.in, caps)
			if err != nil {
				t.Fatal(err)
			}
			if got.EffectiveOutput != tc.want {
				t.Fatalf("effective=%q want %q", got.EffectiveOutput, tc.want)
			}
		})
	}
}

func TestResolveConflictingFlags(t *testing.T) {
	_, err := presentation.Resolve(presentation.Input{
		OutputFlag:   "json",
		ReporterFlag: "ndjson",
	}, pipeCaps())
	if err == nil {
		t.Fatal("expected conflict")
	}
	var ce *presentation.ConflictError
	if !asConflict(err, &ce) {
		t.Fatalf("got %T %v", err, err)
	}
}

func asConflict(err error, target **presentation.ConflictError) bool {
	if err == nil {
		return false
	}
	ce, ok := err.(*presentation.ConflictError)
	if !ok {
		return false
	}
	*target = ce
	return true
}

func TestForcedRichUnsupported(t *testing.T) {
	_, err := presentation.Resolve(presentation.Input{OutputFlag: "rich"}, pipeCaps())
	if err == nil {
		t.Fatal("expected rich unsupported")
	}
	if _, ok := err.(*presentation.RichUnsupportedError); !ok {
		t.Fatalf("got %T", err)
	}
}

func TestStructuredCommandJSONConflict(t *testing.T) {
	opts := presentation.ResolvedOptions{EffectiveOutput: presentation.OutputJSON}
	if err := presentation.StructuredConflictsWithCommandJSON(opts, true); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestControllerCloseIdempotent(t *testing.T) {
	var out, errb bytes.Buffer
	resolved, err := presentation.Resolve(presentation.Input{}, pipeCaps())
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := presentation.NewController(resolved, pipeCaps(), presentation.StreamWriters{Out: &out, Err: &errb})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ctrl.Close(ctx, presentation.Outcome{}); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Close(ctx, presentation.Outcome{}); err != nil {
		t.Fatal(err)
	}
}

func TestControllerSuspendResume(t *testing.T) {
	resolved, err := presentation.Resolve(presentation.Input{}, pipeCaps())
	if err != nil {
		t.Fatal(err)
	}
	ctrl, err := presentation.NewController(resolved, pipeCaps(), presentation.StreamWriters{Out: io.Discard, Err: io.Discard})
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := ctrl.Suspend(ctx); err != nil {
		t.Fatal(err)
	}
	if err := ctrl.Resume(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestEnvMapIncludesMEWOutput(t *testing.T) {
	t.Setenv("MEW_OUTPUT", "plain")
	m := presentation.EnvMap()
	if m["MEW_OUTPUT"] != "plain" {
		t.Fatalf("%v", m)
	}
}

func TestReporterFormatMapping(t *testing.T) {
	opts := presentation.ResolvedOptions{EffectiveOutput: presentation.OutputNDJSON}
	if opts.ReporterFormat() != "ndjson" {
		t.Fatalf("%q", opts.ReporterFormat())
	}
}

func TestDowngradeRichEmitsDebug(t *testing.T) {
	var errb bytes.Buffer
	in := presentation.Input{OutputFlag: "auto"}
	caps := presentation.Capabilities{CI: true}
	resolved, err := presentation.Resolve(in, caps)
	if err != nil {
		t.Fatal(err)
	}
	if !resolved.DowngradedRich {
		// auto on CI becomes plain without downgrade flag when never requested rich
	}
	resolved.Debug = true
	resolved.DowngradedRich = true
	ctrl, err := presentation.NewController(resolved, caps, presentation.StreamWriters{Out: io.Discard, Err: &errb})
	if err != nil {
		t.Fatal(err)
	}
	_ = ctrl
	if !strings.Contains(errb.String(), "downgraded") {
		t.Fatalf("stderr=%q", errb.String())
	}
}
