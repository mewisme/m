package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

func (g *globalFlags) bindPresentation(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&g.output, "output", "", "output mode: auto|rich|plain|json|ndjson|silent")
	cmd.PersistentFlags().StringVar(&g.progress, "progress", "", "progress: auto|always|never")
	cmd.PersistentFlags().StringVar(&g.unicode, "unicode", "", "unicode symbols: auto|always|never")
	cmd.PersistentFlags().StringVar(&g.interactive, "interactive", "", "interactive UI: auto|always|never")
	cmd.PersistentFlags().StringVar(&g.logLevel, "log-level", "", "log level: error|warn|info|debug")
	cmd.PersistentFlags().BoolVar(&g.noSummary, "no-summary", false, "suppress command summary output")
	cmd.PersistentFlags().BoolVar(&g.accessible, "accessible", false, "accessible append-only output")
	cmd.PersistentFlags().BoolVar(&g.presentationLegacy, "presentation-legacy", false, "force legacy human presentation (hidden rollout switch)")
	_ = cmd.PersistentFlags().MarkHidden("presentation-legacy")
}

func (g *globalFlags) presentationInput(cfg *config.Effective) presentation.Input {
	in := presentation.Input{
		OutputFlag:      g.output,
		ReporterFlag:    g.reporter,
		ColorFlag:       g.color,
		NoColor:         g.noColor,
		ProgressFlag:    g.progress,
		UnicodeFlag:     g.unicode,
		InteractiveFlag: g.interactive,
		LogLevelFlag:    g.logLevel,
		NoSummary:       g.noSummary,
		Accessible:      g.accessible,
		LegacyFlag:      g.presentationLegacy,
		Debug:           g.resolveDebug(),
		Unsafe:          g.unsafe,
		Env:             presentation.EnvMap(),
		Config:          presentationConfig(cfg),
	}
	return in
}

func presentationConfig(cfg *config.Effective) map[string]string {
	if cfg == nil || len(cfg.Values) == 0 {
		return nil
	}
	flat := make(map[string]string, len(cfg.Values))
	for k, v := range cfg.Values {
		switch raw := v.Raw.(type) {
		case string:
			flat[k] = raw
		case bool:
			flat[k] = strconv.FormatBool(raw)
		case float64:
			flat[k] = strconv.FormatFloat(raw, 'f', -1, 64)
		case int:
			flat[k] = strconv.Itoa(raw)
		}
	}
	return presentation.ConfigMap(flat)
}

func (g *globalFlags) controller(cmd *cobra.Command, cfg *config.Effective) (presentation.Controller, error) {
	if g.ctrl != nil {
		return g.ctrl, nil
	}
	streams := presentation.WriterPair(cmd.OutOrStdout(), cmd.ErrOrStderr())
	caps := presentation.DetectCapabilities(cmd.InOrStdin(), streams.Out, streams.Err, nil)
	resolved, err := presentation.Resolve(g.presentationInput(cfg), caps)
	if err != nil {
		return nil, err
	}
	ctrl, err := presentation.NewController(resolved, caps, streams)
	if err != nil {
		return nil, err
	}
	g.ctrl = ctrl
	return ctrl, nil
}

// staticRenderer returns the design-system renderer for this invocation.
func (g *globalFlags) staticRenderer(cmd *cobra.Command, cfg *config.Effective) (presentation.StaticRenderer, error) {
	ctrl, err := g.controller(cmd, cfg)
	if err != nil {
		return nil, err
	}
	settings := presentation.Effective(ctrl.Options(), ctrl.Capabilities())
	return presentation.NewStaticRenderer(settings), nil
}

// mustStaticRenderer returns a plain renderer when controller setup fails.
func (g *globalFlags) mustStaticRenderer(cmd *cobra.Command, cfg *config.Effective) presentation.StaticRenderer {
	r, err := g.staticRenderer(cmd, cfg)
	if err != nil {
		return presentation.NewStaticRenderer(presentation.EffectiveSettings{
			ThemeMode:  presentation.ThemeNone,
			Width:      80,
			UseUnicode: false,
			Symbols:    presentation.ASCIISymbols,
		})
	}
	return r
}

func (g *globalFlags) resolveReporter(cmd *cobra.Command, cfg *config.Effective) (diagnostics.Reporter, error) {
	ctrl, err := g.controller(cmd, cfg)
	if err != nil {
		return nil, err
	}
	return ctrl.Reporter(), nil
}

func (g *globalFlags) mustReporter(cmd *cobra.Command, cfg *config.Effective) diagnostics.Reporter {
	rep, err := g.resolveReporter(cmd, cfg)
	if err != nil {
		return diagnostics.NewReporter(diagnostics.Options{
			Out: cmd.OutOrStdout(),
			Err: cmd.ErrOrStderr(),
		})
	}
	return rep
}

func wrapPresentationErr(err error) error {
	if err == nil {
		return nil
	}
	switch err.(type) {
	case *presentation.ConflictError, *presentation.InvalidModeError, *presentation.RichUnsupportedError:
		return apperr.Wrap(apperr.Usage, "cli.presentation", "", err)
	default:
		return err
	}
}

func (g *globalFlags) validateStructuredConflict(cmd *cobra.Command) error {
	ctrl, err := g.controller(cmd, nil)
	if err != nil {
		return wrapPresentationErr(err)
	}
	if cmd == nil {
		return nil
	}
	if f := cmd.Flags().Lookup("json"); f != nil && f.Changed {
		val, _ := cmd.Flags().GetBool("json")
		if val {
			if err := presentation.StructuredConflictsWithCommandJSON(ctrl.Options(), true); err != nil {
				return wrapPresentationErr(err)
			}
		}
	}
	return nil
}

func closePresentation(cmd *cobra.Command, g *globalFlags, outcome presentation.Outcome) {
	if g == nil || g.ctrl == nil {
		return
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = cmd.Root().Context()
	}
	_ = g.ctrl.Close(ctx, outcome)
}

// ValidateCommandPresentation is exported for tests that need structured/json conflict checks.
func ValidateCommandPresentation(cmd *cobra.Command, g *globalFlags) error {
	if g == nil {
		return nil
	}
	return g.validateStructuredConflict(cmd)
}

func presentationOutcome(err error) presentation.Outcome {
	return presentation.Outcome{Err: err}
}

func reportCLIError(cmd *cobra.Command, g *globalFlags, err error) {
	if err == nil {
		return
	}
	err = classifyCLIError(err)
	rep := g.mustReporter(cmd, nil)
	rep.Error(err)
}

func reportCLIErrorWithExit(cmd *cobra.Command, g *globalFlags, err error) int {
	reportCLIError(cmd, g, err)
	return apperr.ExitCode(classifyCLIError(err))
}

func formatPresentationHelp(mode string) string {
	return fmt.Sprintf("see --output (%s)", mode)
}
