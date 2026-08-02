package cli

import (
	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/darkmode"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/presentation"
)

func (g *globalFlags) bindPresentation(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&g.output, "output", "", "output mode: rich|plain|json|ndjson|silent")
	cmd.PersistentFlags().StringVar(&g.logLevel, "log-level", "", "log level: error|warn|info|debug")
	cmd.PersistentFlags().BoolVar(&g.noColor, "no-color", false, "disable ANSI color")
	cmd.PersistentFlags().BoolVar(&g.noProgress, "no-progress", false, "disable progress output")
	cmd.PersistentFlags().BoolVar(&g.ascii, "ascii", false, "use ASCII instead of Unicode symbols")
	cmd.PersistentFlags().BoolVar(&g.noSummary, "no-summary", false, "suppress command summary output")
	cmd.PersistentFlags().BoolVar(&g.accessible, "accessible", false, "accessible append-only output")
}

func (g *globalFlags) presentationInput() presentation.Input {
	theme := g.loadUITheme()
	return presentation.Input{
		OutputFlag:   g.output,
		NoColor:      g.noColor,
		ASCII:        g.ascii,
		NoProgress:   g.noProgress,
		Accessible:   g.accessible,
		NoSummary:    g.noSummary,
		LogLevelFlag: g.logLevel,
		Debug:        g.resolveDebug(),
		Unsafe:       g.unsafe,
		Theme:        theme,
		BinaryName:   g.invokedBinary,
	}
}

// loadUITheme reads ui.theme from config (lightweight, skips project config if missing).
func (g *globalFlags) loadUITheme() string {
	eff, err := config.Load(nil, config.LoadOptions{
		CWD:         g.cwd,
		GlobalPath:  g.configPath,
		ProjectRoot: g.cwd,
	})
	if err != nil {
		return ""
	}
	return config.String(eff, "ui.theme", "")
}

func (g *globalFlags) controller(cmd *cobra.Command) (presentation.Controller, error) {
	if g.ctrl != nil {
		return g.ctrl, nil
	}
	streams := presentation.WriterPair(cmd.OutOrStdout(), cmd.ErrOrStderr())
	caps := presentation.DetectCapabilities(cmd.InOrStdin(), streams.Out, streams.Err, nil)
	resolved, err := presentation.Resolve(g.presentationInput())
	if err != nil {
		return nil, err
	}
	ctrl, err := presentation.NewController(resolved, caps, streams, darkmodeDetector{})
	if err != nil {
		return nil, err
	}
	g.ctrl = ctrl
	return ctrl, nil
}

// darkmodeDetector adapts the internal darkmode package to presentation.DarkModeDetector.
type darkmodeDetector struct{}

func (darkmodeDetector) IsDarkMode() (bool, error) {
	return darkmode.IsDarkMode()
}

// staticRenderer returns the design-system renderer for this invocation.
func (g *globalFlags) staticRenderer(cmd *cobra.Command) (presentation.StaticRenderer, error) {
	ctrl, err := g.controller(cmd)
	if err != nil {
		return nil, err
	}
	settings := ctrl.Settings()
	settings.BinaryName = g.invokedBinary
	return presentation.NewStaticRenderer(settings), nil
}

// mustStaticRenderer returns a plain renderer when controller setup fails.
func (g *globalFlags) mustStaticRenderer(cmd *cobra.Command) presentation.StaticRenderer {
	r, err := g.staticRenderer(cmd)
	if err != nil {
		return presentation.NewStaticRenderer(presentation.EffectiveSettings{
			ThemeMode:  presentation.ThemeNone,
			Width:      80,
			UseUnicode: false,
			Symbols:    presentation.ASCIISymbols,
			BinaryName: g.invokedBinary,
		})
	}
	return r
}

func (g *globalFlags) resolveReporter(cmd *cobra.Command) (diagnostics.Reporter, error) {
	ctrl, err := g.controller(cmd)
	if err != nil {
		return nil, err
	}
	return ctrl.Reporter(), nil
}

func (g *globalFlags) mustReporter(cmd *cobra.Command) diagnostics.Reporter {
	rep, err := g.resolveReporter(cmd)
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
	case *presentation.ConflictError, *presentation.InvalidModeError:
		return apperr.Wrap(apperr.Usage, "cli.presentation", "", err)
	default:
		return err
	}
}

func (g *globalFlags) validateStructuredConflict(cmd *cobra.Command) error {
	ctrl, err := g.controller(cmd)
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
