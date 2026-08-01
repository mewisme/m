package transform

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"runtime"
	"strings"
	"time"

	"github.com/evanw/esbuild/pkg/api"
)

// Engine transforms TypeScript source in memory.
type Engine interface {
	Transform(ctx context.Context, req TransformRequest) (TransformResult, error)
	Identity() EngineIdentity
}

// esbuildEngine implements Engine using the esbuild Go API.
type esbuildEngine struct{}

// NewEsbuildEngine returns a new esbuild-backed transform engine.
func NewEsbuildEngine() Engine {
	return &esbuildEngine{}
}

func (e *esbuildEngine) Identity() EngineIdentity {
	return EngineIdentity{
		Name:    "esbuild",
		Version: "0.28.1",
	}
}

func (e *esbuildEngine) Transform(ctx context.Context, req TransformRequest) (TransformResult, error) {
	var zero TransformResult
	start := time.Now()

	if err := ctx.Err(); err != nil {
		return zero, err
	}

	loader := mapLoader(req.Loader)
	esbuildTarget := mapTarget(req.NormalizedOpts.Target, req.TargetNodeMajor)

	sourceMap := api.SourceMapNone
	if req.SourceMapMode == SourceMapInline {
		sourceMap = api.SourceMapInline
	} else if req.SourceMapMode == SourceMapExternal {
		sourceMap = api.SourceMapExternal
	}

	result := api.Transform(string(req.SourceBytes), api.TransformOptions{
		Loader:            loader,
		Target:            esbuildTarget,
		Format:            mapFormat(req.Format),
		Sourcemap:         sourceMap,
		SourcesContent:    api.SourcesContentInclude,
		Sourcefile:        req.SourcePath,
		Define:            nil,
		Pure:              nil,
		MinifyWhitespace:  false,
		MinifyIdentifiers: false,
		MinifySyntax:      false,
		TreeShaking:       api.TreeShakingFalse,
		Platform:          api.PlatformNode,
		Charset:           api.CharsetUTF8,
	})

	if len(result.Errors) > 0 {
		return zero, transformErrors(result.Errors)
	}

	code := result.Code
	sourceMapBytes := result.Map

	// Compute output digest
	h := sha256.New()
	h.Write(code)
	h.Write(sourceMapBytes)
	outputDigest := hex.EncodeToString(h.Sum(nil))

	diags := convertDiagnostics(result.Warnings, SeverityWarning)

	return TransformResult{
		Code:         code,
		SourceMap:    sourceMapBytes,
		OutputDigest: outputDigest,
		Diagnostics:  diags,
		CacheStatus:  CacheStatusBypass,
		Transformer:  e.Identity(),
		Elapsed:      time.Since(start),
	}, nil
}

func mapLoader(l LoaderKind) api.Loader {
	switch l {
	case LoaderTS:
		return api.LoaderTS
	case LoaderMTS:
		return api.LoaderTS // esbuild handles MTS same as TS
	case LoaderCTS:
		return api.LoaderTS // esbuild handles CTS same as TS
	default:
		return api.LoaderTS
	}
}

func mapFormat(f ModuleFormat) api.Format {
	switch f {
	case FormatESM:
		return api.FormatESModule
	case FormatCJS:
		return api.FormatCommonJS
	default:
		return api.FormatESModule
	}
}

func mapTarget(tsconfigTarget string, nodeMajor int) api.Target {
	// Translate tsconfig target + Node major to esbuild target.
	// Default to current Node LTS-level target.
	switch {
	case nodeMajor >= 22:
		return api.ESNext
	case nodeMajor >= 20:
		return api.ES2023
	case nodeMajor >= 18:
		return api.ES2022
	default:
		return api.ES2020
	}
}

func transformErrors(errs []api.Message) error {
	var msgs []string
	for _, err := range errs {
		loc := ""
		if err.Location != nil {
			loc = fmt.Sprintf(" at %s:%d:%d", err.Location.File, err.Location.Line, err.Location.Column)
		}
		msgs = append(msgs, fmt.Sprintf("%s%s: %s", err.PluginName, loc, err.Text))
	}
	return fmt.Errorf("transform errors: %s", strings.Join(msgs, "; "))
}

func convertDiagnostics(msgs []api.Message, sev Severity) []Diagnostic {
	if len(msgs) == 0 {
		return nil
	}
	out := make([]Diagnostic, len(msgs))
	for i, m := range msgs {
		d := Diagnostic{
			Severity: sev,
			Message:  m.Text,
		}
		if m.Location != nil {
			d.Source = m.Location.File
			d.Line = m.Location.Line
			d.Column = m.Location.Column
			d.Length = m.Location.Length
			d.Snippet = m.Location.LineText
		}
		out[i] = d
	}
	return out
}

// SystemInfo returns the host metadata for benchmarks and diagnostics.
func SystemInfo() map[string]string {
	return map[string]string{
		"goos":   runtime.GOOS,
		"goarch": runtime.GOARCH,
		"engine": "esbuild",
	}
}
