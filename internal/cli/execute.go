package cli

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"reflect"
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
	recursive     bool
}

var flagOwners sync.Map     // *cobra.Command -> *globalFlags
var rootBuildInfos sync.Map // *cobra.Command -> BuildInfo

func storeRootBuildInfo(root *cobra.Command, info BuildInfo) {
	if root != nil {
		rootBuildInfos.Store(root, info)
	}
}

func loadRootBuildInfo(root *cobra.Command) BuildInfo {
	if v, ok := rootBuildInfos.Load(root); ok {
		return v.(BuildInfo)
	}
	return BuildInfo{}
}

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

func (g *globalFlags) bindRecursive(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&g.recursive, "recursive", "r", false, "workspace recursive mode (consumed by m run)")
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

func workspaceRecursive(cmd *cobra.Command) bool {
	g := ownerFlags(cmd.Root())
	return g != nil && g.recursive
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

func buildAppContext(ctx context.Context, cmd *cobra.Command, g *globalFlags, info BuildInfo) (*app.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cwd := g.cwd
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return nil, err
		}
	} else {
		abs, err := filepath.Abs(cwd)
		if err != nil {
			return nil, err
		}
		cwd = abs
	}
	ac, err := app.New(ctx, app.Options{
		CWD:           cwd,
		ConfigPath:    g.configPath,
		Offline:       g.offline,
		PreferOffline: g.preferOffline,
		Reporter:      g.newReporter(cmd),
		Version:       info.Version,
		Commit:        info.Commit,
		BuildDate:     info.BuildDate,
	})
	if err != nil {
		return nil, err
	}
	return ac, nil
}

func execute(root *cobra.Command, info BuildInfo, argv []string) (exit int) {
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

	if len(argv) == 0 {
		argv = os.Args[1:]
	}

	if dispatchEnabledForRoot(root) {
		if code, handled := tryDirectDispatch(ctx, root, g, info, argv); handled {
			return code
		}
	}
	if isMXRoot(root) {
		if code, handled := tryMXDispatch(ctx, root, g, info, argv); handled {
			return code
		}
	}

	root.SetArgs(argv)
	err := root.ExecuteContext(ctx)
	rep = g.newReporter(root)
	if err == nil {
		return 0
	}
	err = classifyCLIError(err)
	rep.Error(err)
	return apperr.ExitCode(err)
}

func dispatchEnabledForRoot(root *cobra.Command) bool {
	if root == nil {
		return false
	}
	switch root.Name() {
	case "m", "mew":
		return true
	default:
		return false
	}
}

func isMXRoot(root *cobra.Command) bool {
	if root == nil {
		return false
	}
	switch root.Name() {
	case "mx", "mewx":
		return true
	default:
		return false
	}
}

func tryDirectDispatch(ctx context.Context, root *cobra.Command, g *globalFlags, info BuildInfo, argv []string) (int, bool) {
	if len(argv) == 0 {
		return handleBareM(ctx, root, g, info, PhaseAResult{})
	}
	if isRootMetaInvocation(argv) {
		return 0, false
	}

	phase, err := ParsePhaseA(argv)
	if err != nil {
		rep := g.newReporter(root)
		rep.Error(classifyCLIError(err))
		return apperr.ExitCode(err), true
	}

	if phase.BareM {
		applyLeadingToGlobalFlags(g, phase.Leading)
		return handleBareM(ctx, root, g, info, phase)
	}

	if kind, _ := lookupBuiltin(root, phase.Selector); kind != "" {
		return 0, false
	}

	applyLeadingToGlobalFlags(g, phase.Leading)
	ac, err := buildAppContext(ctx, root, g, info)
	if err != nil {
		rep := g.newReporter(root)
		rep.Error(classifyCLIError(err))
		return apperr.ExitCode(err), true
	}

	res := ResolveDispatch(root, phase, ac.CWD, ac.Config)
	switch res.Kind {
	case OutcomeScript:
		if res.Invocation == nil {
			err := apperr.New(apperr.Internal, "dispatch", res.Canonical, "missing script invocation")
			rep := g.newReporter(root)
			rep.Error(err)
			return apperr.ExitCode(err), true
		}
		_, err = app.Run(ctx, ac, res.Invocation.ToRunOptions())
		rep := g.newReporter(root)
		if err == nil {
			return 0, true
		}
		rep.Error(classifyCLIError(err))
		return apperr.ExitCode(err), true
	case OutcomeBin:
		if res.Bin == nil {
			err := apperr.New(apperr.Internal, "dispatch", res.Canonical, "missing bin invocation")
			rep := g.newReporter(root)
			rep.Error(err)
			return apperr.ExitCode(err), true
		}
		_, err = app.Exec(ctx, ac, res.Bin.ToExecOptions())
		rep := g.newReporter(root)
		if err == nil {
			return 0, true
		}
		rep.Error(classifyCLIError(err))
		return apperr.ExitCode(err), true
	case OutcomeSuggest, OutcomeUnknown:
		return emitDispatchFailure(root, g, res)
	default:
		return 0, false
	}
}

func dispatchCWD(g *globalFlags, phase PhaseAResult) string {
	if phase.Leading.cwd != "" {
		return phase.Leading.cwd
	}
	if g != nil && g.cwd != "" {
		return g.cwd
	}
	cwd, _ := os.Getwd()
	return cwd
}

func handleBareM(ctx context.Context, root *cobra.Command, g *globalFlags, info BuildInfo, phase PhaseAResult) (int, bool) {
	_ = ctx
	_ = info
	cwd := dispatchCWD(g, phase)
	msg := bareMUsageMessage(cwd)
	err := apperr.New(apperr.Usage, "cli", "", msg)
	rep := g.newReporter(root)
	rep.Error(err)
	return apperr.ExitCode(err), true
}

func emitDispatchFailure(root *cobra.Command, g *globalFlags, res DispatchResult) (int, bool) {
	rep := g.newReporter(root)
	if res.Err != nil {
		rep.Error(classifyCLIError(res.Err))
		return apperr.ExitCode(res.Err), true
	}
	msg := res.Message
	if msg == "" {
		msg = fmt.Sprintf("unknown command %q", res.Canonical)
	}
	err := apperr.New(apperr.Usage, "dispatch", res.Canonical, msg)
	rep.Error(err)
	return apperr.ExitCode(err), true
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
		(strings.Contains(msg, "accepts") && strings.Contains(msg, "arg")) ||
		(strings.Contains(msg, "requires") && strings.Contains(msg, "arg")) {
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
	argv := cobraPendingArgs(root)
	if len(argv) == 0 {
		argv = os.Args[1:]
	}
	return executeWithArgv(root, ctx, argv)
}

// ExecuteWithArgv runs root with explicit argv (integration tests).
func ExecuteWithArgv(root *cobra.Command, ctx context.Context, argv []string) int {
	return executeWithArgv(root, ctx, argv)
}

func executeWithArgv(root *cobra.Command, ctx context.Context, argv []string) int {
	g := ownerFlags(root)
	rep := g.newReporter(root)

	if dispatchEnabledForRoot(root) {
		if code, handled := tryDirectDispatch(ctx, root, g, loadRootBuildInfo(root), argv); handled {
			return code
		}
	}
	if isMXRoot(root) {
		if code, handled := tryMXDispatch(ctx, root, g, loadRootBuildInfo(root), argv); handled {
			return code
		}
	}

	root.SetArgs(argv)
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

func cobraPendingArgs(cmd *cobra.Command) []string {
	if cmd == nil {
		return nil
	}
	rv := reflect.ValueOf(cmd).Elem()
	f := rv.FieldByName("args")
	if !f.IsValid() || f.Kind() != reflect.Slice || f.Len() == 0 {
		return nil
	}
	out := make([]string, f.Len())
	for i := 0; i < f.Len(); i++ {
		out[i] = f.Index(i).String()
	}
	return out
}

// dispatchJSON is the __dispatch introspection schema (schemaVersion 1).
type dispatchJSON struct {
	SchemaVersion int                      `json:"schemaVersion"`
	Kind          string                   `json:"kind"`
	Selector      string                   `json:"selector"`
	Enabled       bool                     `json:"enabled"`
	Path          string                   `json:"path,omitempty"`
	Suggestions   []dispatchSuggestionJSON `json:"suggestions"`
}

type dispatchSuggestionJSON struct {
	Name       string `json:"name"`
	Kind       string `json:"kind"`
	Invocation string `json:"invocation"`
	Distance   int    `json:"distance"`
}

func encodeDispatchJSON(res DispatchResult, selector string) ([]byte, error) {
	doc := dispatchJSON{
		SchemaVersion: 1,
		Selector:      selector,
		Enabled:       res.DirectGateOn,
		Suggestions:   []dispatchSuggestionJSON{},
	}
	switch res.Kind {
	case OutcomeBuiltin:
		doc.Kind = "builtin"
		doc.Path = res.Canonical
	case OutcomeAlias:
		doc.Kind = "alias"
		doc.Path = res.Canonical
	case OutcomeScript:
		doc.Kind = "script"
	case OutcomeSuggest:
		doc.Kind = "suggest"
	case OutcomeUnknown:
		doc.Kind = "unknown"
	default:
		doc.Kind = string(res.Kind)
	}
	for _, s := range res.Suggestions {
		doc.Suggestions = append(doc.Suggestions, dispatchSuggestionJSON{
			Name:       s.Name,
			Kind:       string(s.Kind),
			Invocation: s.Invocation,
			Distance:   s.Distance,
		})
	}
	enc, err := json.Marshal(doc)
	if err != nil {
		return nil, err
	}
	return enc, nil
}
