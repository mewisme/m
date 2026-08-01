package presentation_test

import (
	"bytes"
	"context"
	"io"
	"testing"

	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

func pipeCaps() presentation.Capabilities {
	return presentation.Capabilities{}
}

func TestResolveOutputDefaults(t *testing.T) {
	cases := []struct {
		name string
		in   presentation.Input
		want presentation.OutputMode
	}{
		{
			name: "no flags defaults to rich",
			in:   presentation.Input{},
			want: presentation.OutputRich,
		},
		{
			name: "explicit rich",
			in:   presentation.Input{OutputFlag: "rich"},
			want: presentation.OutputRich,
		},
		{
			name: "explicit plain",
			in:   presentation.Input{OutputFlag: "plain"},
			want: presentation.OutputPlain,
		},
		{
			name: "explicit json",
			in:   presentation.Input{OutputFlag: "json"},
			want: presentation.OutputJSON,
		},
		{
			name: "explicit ndjson",
			in:   presentation.Input{OutputFlag: "ndjson"},
			want: presentation.OutputNDJSON,
		},
		{
			name: "explicit silent",
			in:   presentation.Input{OutputFlag: "silent"},
			want: presentation.OutputSilent,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := presentation.Resolve(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			if got.Output != tc.want {
				t.Fatalf("output=%q want %q", got.Output, tc.want)
			}
		})
	}
}

func TestResolveRejectsAutoDefaultHuman(t *testing.T) {
	for _, v := range []string{"auto", "default", "human"} {
		t.Run(v, func(t *testing.T) {
			_, err := presentation.Resolve(presentation.Input{OutputFlag: v})
			if err == nil {
				t.Fatalf("expected error for --output=%s", v)
			}
			var ie *presentation.InvalidModeError
			if !asInvalid(err, &ie) {
				t.Fatalf("expected InvalidModeError, got %T: %v", err, err)
			}
		})
	}
}

func TestResolveBoolFlags(t *testing.T) {
	cases := []struct {
		name  string
		in    presentation.Input
		check func(t *testing.T, opts presentation.ResolvedOptions)
	}{
		{
			name: "--no-color disables color",
			in:   presentation.Input{NoColor: true},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if opts.Color {
					t.Fatal("Color should be false")
				}
			},
		},
		{
			name: "--ascii disables unicode",
			in:   presentation.Input{ASCII: true},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if opts.Unicode {
					t.Fatal("Unicode should be false")
				}
			},
		},
		{
			name: "--no-progress disables progress",
			in:   presentation.Input{NoProgress: true},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if opts.Progress {
					t.Fatal("Progress should be false")
				}
			},
		},
		{
			name: "--accessible keeps rich output",
			in:   presentation.Input{Accessible: true},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if opts.Output != presentation.OutputRich {
					t.Fatalf("Output=%q want rich", opts.Output)
				}
				if !opts.Accessible {
					t.Fatal("Accessible should be true")
				}
			},
		},
		{
			name: "--no-summary sets Summary=false",
			in:   presentation.Input{NoSummary: true},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if opts.Summary {
					t.Fatal("Summary should be false")
				}
			},
		},
		{
			name: "defaults: color, unicode, progress all true",
			in:   presentation.Input{},
			check: func(t *testing.T, opts presentation.ResolvedOptions) {
				if !opts.Color {
					t.Fatal("Color should default to true for rich")
				}
				if !opts.Unicode {
					t.Fatal("Unicode should default to true")
				}
				if !opts.Progress {
					t.Fatal("Progress should default to true for rich")
				}
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := presentation.Resolve(tc.in)
			if err != nil {
				t.Fatal(err)
			}
			tc.check(t, got)
		})
	}
}

func TestResolvePlainOutputDisablesRichFeatures(t *testing.T) {
	got, err := presentation.Resolve(presentation.Input{OutputFlag: "plain"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Color {
		t.Fatal("Color should be false for plain output")
	}
	if got.Progress {
		t.Fatal("Progress should be false for plain output")
	}
}

func TestStructuredCommandJSONConflict(t *testing.T) {
	opts := presentation.ResolvedOptions{Output: presentation.OutputJSON}
	if err := presentation.StructuredConflictsWithCommandJSON(opts, true); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestControllerCloseIdempotent(t *testing.T) {
	var out, errb bytes.Buffer
	resolved, err := presentation.Resolve(presentation.Input{})
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
	resolved, err := presentation.Resolve(presentation.Input{OutputFlag: "plain"})
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

func TestEnvMapNoLongerIncludesPresentation(t *testing.T) {
	t.Setenv("MEW_OUTPUT", "plain")
	t.Setenv("MEW_COLOR", "never")
	m := presentation.EnvMap()
	if m["MEW_OUTPUT"] != "" {
		t.Fatal("MEW_OUTPUT should not be in EnvMap")
	}
	if m["MEW_COLOR"] != "" {
		t.Fatal("MEW_COLOR should not be in EnvMap")
	}
}

func TestReporterFormatMapping(t *testing.T) {
	opts := presentation.ResolvedOptions{Output: presentation.OutputNDJSON}
	if opts.ReporterFormat() != "ndjson" {
		t.Fatalf("%q", opts.ReporterFormat())
	}
}

func TestColorModeMapping(t *testing.T) {
	opts := presentation.ResolvedOptions{Color: true}
	if opts.ColorMode() != diagnostics.ColorAlways {
		t.Fatalf("ColorMode=%d want ColorAlways (%d)", opts.ColorMode(), diagnostics.ColorAlways)
	}
	opts = presentation.ResolvedOptions{Color: false}
	if opts.ColorMode() != diagnostics.ColorNever {
		t.Fatalf("ColorMode=%d want ColorNever (%d)", opts.ColorMode(), diagnostics.ColorNever)
	}
}

func asInvalid(err error, target **presentation.InvalidModeError) bool {
	if err == nil {
		return false
	}
	ie, ok := err.(*presentation.InvalidModeError)
	if !ok {
		return false
	}
	*target = ie
	return true
}
