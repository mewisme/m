// Package testkit provides fixtures for presentation resolver tests.
package testkit

import (
	"io"
	"os"

	"github.com/mewisme/mew/internal/presentation"
)

// PipeCapabilities returns non-TTY capabilities.
func PipeCapabilities() presentation.Capabilities {
	return presentation.Capabilities{Width: 80, Background: presentation.BackgroundLight}
}

// TTYCapabilities returns interactive-terminal capabilities.
func TTYCapabilities() presentation.Capabilities {
	return presentation.Capabilities{
		StdoutTTY: true, StderrTTY: true, StdinTTY: true,
		Width: 80, Unicode: true, Interactive: true,
		ColorProfile: presentation.ColorProfileTrueColor,
		Background:   presentation.BackgroundLight,
	}
}

// CICapabilities returns CI environment capabilities.
func CICapabilities() presentation.Capabilities {
	return presentation.Capabilities{CI: true}
}

// DiscardStreams returns writers that discard output.
func DiscardStreams() presentation.StreamWriters {
	return presentation.StreamWriters{Out: io.Discard, Err: io.Discard}
}

// EnvWith sets one variable on a copy of the base env map.
func EnvWith(base map[string]string, key, value string) map[string]string {
	out := make(map[string]string, len(base)+1)
	for k, v := range base {
		out[k] = v
	}
	out[key] = value
	return out
}

// MinimalEnv returns an empty env map (not nil).
func MinimalEnv() map[string]string {
	return map[string]string{}
}

// RealStreams returns process stdout/stderr.
func RealStreams() presentation.StreamWriters {
	return presentation.WriterPair(os.Stdout, os.Stderr)
}
