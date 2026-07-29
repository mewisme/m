package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/diagnostics"
)

// globalFlags holds persistent CLI presentation options.
type globalFlags struct {
	reporter      string
	debug         bool
	color         string
	noColor       bool
	unsafe        bool
	cwd           string
	configPath    string
	offline       bool
	preferOffline bool
	filter        []string
}

var flagOwners sync.Map // *cobra.Command -> *globalFlags

func (g *globalFlags) bind(cmd *cobra.Command) {
	cmd.PersistentFlags().StringVar(&g.reporter, "reporter", "", "output reporter: default|ndjson|json|silent (env MEW_LOG_FORMAT)")
	cmd.PersistentFlags().BoolVar(&g.debug, "debug", false, "verbose diagnostics (env MEW_DEBUG or M_LOG=debug)")
	cmd.PersistentFlags().StringVar(&g.color, "color", "auto", "color: auto|always|never")
	cmd.PersistentFlags().BoolVar(&g.noColor, "no-color", false, "disable ANSI color")
	cmd.PersistentFlags().BoolVar(&g.unsafe, "unsafe-diagnostics", false, "disable secret redaction (dangerous)")
	_ = cmd.PersistentFlags().MarkHidden("unsafe-diagnostics")
	cmd.PersistentFlags().StringVar(&g.cwd, "cwd", "", "project working directory")
	cmd.PersistentFlags().StringVar(&g.configPath, "config", "", "JSONC config file overlay path")
	cmd.PersistentFlags().BoolVar(&g.offline, "offline", false, "force offline mode")
	cmd.PersistentFlags().BoolVar(&g.preferOffline, "prefer-offline", false, "prefer cached artifacts")
	cmd.PersistentFlags().StringArrayVar(&g.filter, "filter", nil, "workspace package filter (pnpm-style)")
}

func (g *globalFlags) resolveFormat() string {
	if g.reporter != "" {
		return g.reporter
	}
	// intentional: pre-app.New presentation flags read ambient env before snapshot exists.
	if v := os.Getenv("MEW_LOG_FORMAT"); v != "" {
		return v
	}
	return "default"
}

func (g *globalFlags) resolveDebug() bool {
	if g.debug {
		return true
	}
	// intentional: pre-app.New debug flags read ambient env before snapshot exists.
	if os.Getenv("MEW_DEBUG") != "" {
		return true
	}
	return strings.EqualFold(os.Getenv("M_LOG"), "debug")
}

func (g *globalFlags) resolveColor() diagnostics.ColorMode {
	if g.noColor || strings.EqualFold(g.color, "never") {
		return diagnostics.ColorNever
	}
	if strings.EqualFold(g.color, "always") {
		return diagnostics.ColorAlways
	}
	return diagnostics.ColorAuto
}

func (g *globalFlags) newReporter(cmd *cobra.Command) diagnostics.Reporter {
	opts := diagnostics.Options{
		Format: g.resolveFormat(),
		Debug:  g.resolveDebug(),
		Color:  g.resolveColor(),
		Unsafe: g.unsafe,
	}
	if cmd != nil {
		opts.Out = cmd.OutOrStdout()
		opts.Err = cmd.ErrOrStderr()
	}
	return diagnostics.NewReporter(opts)
}

func attachGlobals(root *cobra.Command) *globalFlags {
	g := &globalFlags{}
	g.bind(root)
	flagOwners.Store(root, g)
	return g
}

func workspaceFilters(cmd *cobra.Command) []string {
	g := ownerFlags(cmd.Root())
	if g == nil || len(g.filter) == 0 {
		return nil
	}
	return append([]string(nil), g.filter...)
}

func installOptsFromGlobals(cmd *cobra.Command, base app.InstallOptions) app.InstallOptions {
	base.Filter = workspaceFilters(cmd)
	return base
}

func ownerFlags(root *cobra.Command) *globalFlags {
	if v, ok := flagOwners.Load(root); ok {
		return v.(*globalFlags)
	}
	return &globalFlags{}
}

func execute(root *cobra.Command) (exit int) {
	g := ownerFlags(root)
	rep := g.newReporter(root)
	defer func() {
		if rec := recover(); rec != nil {
			err := apperr.New(apperr.InternalPanic, "cli", newCrashID(), fmt.Sprintf("panic: %v", rec))
			rep.Error(err)
			exit = apperr.ExitCode(err)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	root.SetContext(ctx)

	err := root.ExecuteContext(ctx)
	rep = g.newReporter(root)
	if err == nil {
		return 0
	}
	err = classifyCLIError(err)
	rep.Error(err)
	return apperr.ExitCode(err)
}

func classifyCLIError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, context.Canceled) {
		return apperr.Wrap(apperr.Cancelled, "cli", "", err)
	}
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return err
	}
	msg := err.Error()
	if strings.Contains(msg, "unknown command") ||
		strings.Contains(msg, "unknown flag") ||
		strings.Contains(msg, "invalid argument") ||
		strings.Contains(msg, "required flag") ||
		(strings.Contains(msg, "accepts") && strings.Contains(msg, "arg")) {
		return apperr.Wrap(apperr.Usage, "cli", "", err)
	}
	return apperr.Wrap(apperr.Internal, "cli", "", err)
}

// RecoverPanic is exported for tests to exercise panic recovery formatting.
func RecoverPanic(rep diagnostics.Reporter, fn func()) (exit int) {
	defer func() {
		if rec := recover(); rec != nil {
			err := apperr.New(apperr.InternalPanic, "cli", newCrashID(), fmt.Sprintf("panic: %v", rec))
			rep.Error(err)
			exit = apperr.ExitCode(err)
		}
	}()
	fn()
	return 0
}

// ExecuteWithContext runs root with an explicit context (tests).
func ExecuteWithContext(root *cobra.Command, ctx context.Context) int {
	g := ownerFlags(root)
	root.SetContext(ctx)
	err := root.ExecuteContext(ctx)
	rep := g.newReporter(root)
	if err == nil {
		return 0
	}
	err = classifyCLIError(err)
	rep.Error(err)
	return apperr.ExitCode(err)
}
