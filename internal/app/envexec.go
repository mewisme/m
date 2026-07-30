package app

import (
	"context"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/diagnostics"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/runner"
	"github.com/mewisme/mew/internal/runner/dlx"
	"github.com/mewisme/mew/internal/runner/envexec"
	"github.com/mewisme/mew/internal/snapshot"
)

type execConfigView struct {
	ac *Context
}

func (v execConfigView) CacheRoot() string  { return config.CacheRoot(v.ac.Config) }
func (v execConfigView) MXCacheDir() string { return config.MXCacheDir(v.ac.Config) }
func (v execConfigView) LinkerMode() string {
	return config.String(v.ac.Config, "install.linker", "auto")
}

type execProjectView struct{ root string }

func (p execProjectView) Root() string { return p.root }

type execImporterView struct {
	ExecImporter
}

func (i execImporterView) ProjectRoot() string  { return i.ExecImporter.ProjectRoot }
func (i execImporterView) PackageDir() string   { return i.ExecImporter.PackageDir }
func (i execImporterView) RelativePath() string { return i.Rel }

func defaultOrchestrator(ac *Context) *envexec.Orchestrator {
	return &envexec.Orchestrator{
		Providers: envexec.DefaultProviderRegistry(),
		Leases:    envexec.DLXLeaseManager{MXCacheRoot: MXCacheRoot(ac)},
		Reporter:  ac.Reporter,
	}
}

func providerDeps(ac *Context) envexec.ProviderDeps {
	return envexec.ProviderDeps{
		Config: execConfigView{ac: ac},
		HostEnv: func() []string {
			if ac != nil && ac.Config != nil {
				return ac.Config.Env.Environ()
			}
			return nil
		},
		DiscoverProject: func(ctx context.Context, cwd string) (envexec.ProjectView, error) {
			prev := ac.CWD
			ac.CWD = cwd
			defer func() { ac.CWD = prev }()
			proj, err := OpenProject(ctx, ac)
			if err != nil {
				return nil, err
			}
			return execProjectView{root: proj.Root}, nil
		},
		SelectImporter: func(ctx context.Context, proj envexec.ProjectView, filters []string) (envexec.ImporterView, error) {
			imp, err := SelectExecImporter(ctx, ac, ExecImporterOptions{Filters: filters})
			if err != nil {
				return nil, err
			}
			return execImporterView{imp}, nil
		},
		LoadBinding: func(ctx context.Context, root string) (binmeta.GenerationBinding, error) {
			b, err := LoadGenerationBinding(root)
			return binmeta.GenerationBinding{GenerationID: b.GenerationID, Fingerprint: b.Fingerprint}, err
		},
		Materializer: appFrozenMaterializer{ac: ac},
		SnapshotLoad: loadSnapshotForExec,
		CapsuleOpen:  openCapsuleForExec,
		PrepStage: func(label string) {
			if ac == nil || ac.Reporter == nil {
				return
			}
			ac.Reporter.Progress(diagnostics.Event{V: 1, Type: "prep-stage", Phase: label})
		},
		DLX: envexec.DLXHooks{
			ResolveMetadata: func(ctx context.Context, req envexec.ExecutionRequest) (envexec.DLXResolveResult, error) {
				src := req.Source.(envexec.DLXSource)
				opts := dlxOptionsFromSource(src, req)
				resolved, reqID, err := resolveDLXMetadata(ctx, ac, opts)
				if err != nil {
					return envexec.DLXResolveResult{}, err
				}
				return envexec.DLXResolveResult{
					Identity:   resolved.Identity,
					RequestID:  reqID,
					DirectKeys: resolved.DirectKeys,
					DirectBins: resolved.DirectBins,
					TxnID:      resolved.TxnID,
					AppState:   resolved,
				}, nil
			},
			BuildEnvironment: func(ctx context.Context, req envexec.ExecutionRequest, resolved envexec.DLXResolveResult, envDir string) error {
				src := req.Source.(envexec.DLXSource)
				opts := dlxOptionsFromSource(src, req)
				legacy, _ := resolved.AppState.(*dlxResolveResult)
				if legacy == nil {
					return apperr.New(apperr.Internal, "app.dlx", "", "missing resolve state")
				}
				return buildDLXEnvironment(ctx, ac, opts, legacy, envDir)
			},
			TryLocal: func(ctx context.Context, req envexec.ExecutionRequest) (envexec.PreparedEnvironment, bool, error) {
				src := req.Source.(envexec.DLXSource)
				opts := dlxOptionsFromSource(src, req)
				cwd := ac.CWD
				if cwd == "" {
					cwd, _ = os.Getwd()
				}
				res, ok, err := tryDLXLocal(ctx, ac, opts, cwd)
				if err != nil || !ok {
					return envexec.PreparedEnvironment{}, ok, err
				}
				imp, err := SelectExecImporter(ctx, ac, ExecImporterOptions{})
				if err != nil {
					return envexec.PreparedEnvironment{}, true, err
				}
				bind, err := LoadGenerationBinding(imp.ProjectRoot)
				if err != nil {
					return envexec.PreparedEnvironment{}, true, err
				}
				binNames, err := localDeclaredBins(cwd, src.Packages[0].Name)
				if err != nil {
					return envexec.PreparedEnvironment{}, true, err
				}
				command, err := dlx.InferModeABin(src.Packages[0].Name, binNames)
				if err != nil {
					return envexec.PreparedEnvironment{}, true, err
				}
				env := envexec.PreparedEnvironment{
					SchemaVersion:    envexec.PreparedSchemaVersion,
					Source:           envexec.SourceDLX,
					Identity:         dlxLocalIdentity(bind, imp),
					Root:             imp.ProjectRoot,
					ImporterRoot:     imp.PackageDir,
					ImporterRel:      imp.Rel,
					NodeModules:      filepath.Join(imp.PackageDir, "node_modules"),
					WorkingDirectory: imp.PackageDir,
					Binding:          binmeta.GenerationBinding{GenerationID: bind.GenerationID, Fingerprint: bind.Fingerprint},
					RequireVerified:  false,
					AllowUnowned:     false,
					HostEnv:          ac.Config.Env.Environ(),
					CommandOwner:     src.Packages[0].Name,
					InferredCommand:  command,
					LeasePolicy:      envexec.LeaseNone,
					CacheState:       envexec.CacheWarm,
				}
				_ = res
				return env, true, nil
			},
			Interactive: func() bool {
				return ac != nil && ac.CanPrompt
			},
		},
		Stdin:    os.Stdin,
		Stdout:   os.Stdout,
		Stderr:   os.Stderr,
		Suspend:  ac.SuspendUI,
		Resume:   ac.ResumeUI,
		Prompter: acPrompter(ac),
	}
}

func dlxOptionsFromSource(src envexec.DLXSource, req envexec.ExecutionRequest) DLXOptions {
	modeA := src.Mode == envexec.DLXModePackageCommand
	return DLXOptions{
		ModeA:         modeA,
		PackageSpecs:  src.Packages,
		Command:       req.Command.Name,
		ForwardedArgs: req.Command.Args,
		AssumeYes:     src.Yes,
		Offline:       src.Offline,
		Stdin:         req.IO.Stdin,
		Stdout:        req.IO.Stdout,
		Stderr:        req.IO.Stderr,
	}
}

func loadSnapshotForExec(ctx context.Context, projectRoot, id string) (envexec.SnapshotLoadResult, error) {
	_ = ctx
	var empty envexec.SnapshotLoadResult
	if projectRoot == "" {
		return empty, apperr.New(apperr.Usage, "app.snapshot.exec", id, "missing project root")
	}
	store := snapshot.NewStore(projectRoot)
	rec, err := store.Load(id)
	if err != nil {
		return empty, err
	}
	g, _, err := snapshot.ValidateRestorePair(*rec)
	if err != nil {
		return empty, err
	}
	return envexec.SnapshotLoadResult{
		ID:              id,
		GraphDigest:     rec.Meta.GraphDigest,
		Lock:            rec.Lock,
		Manifest:        rec.Manifest,
		MemberManifests: rec.MemberManifests,
		Graph:           g,
	}, nil
}

func execViaOrchestrator(ctx context.Context, ac *Context, req envexec.ExecutionRequest) (runner.ExecResult, error) {
	if ac == nil || ac.Config == nil {
		return runner.ExecResult{}, apperr.New(apperr.Internal, "app.envexec", "", "missing app context")
	}
	if req.Policy.Network == "" {
		req.Policy = envexec.LockedProviderPolicy(req.Source.Kind())
	}
	if req.Suspend == nil {
		req.Suspend = ac.SuspendUI
	}
	if req.Resume == nil {
		req.Resume = ac.ResumeUI
	}
	started := time.Now()
	result, err := defaultOrchestrator(ac).Execute(ctx, providerDeps(ac), req)
	name := req.Command.Name
	if name == "" {
		name = "exec"
	}
	emitExecCompletion(ac, name, time.Since(started), result.ExitCode, err)
	return result, err
}

func dlxLocalIdentity(bind GenerationBinding, imp ExecImporter) envexec.EnvironmentIdentity {
	mat := bind.Fingerprint
	if len(mat) != 64 {
		mat = envexec.HashHex(bind.GenerationID + "|" + bind.Fingerprint)
	}
	graph := envexec.HashHex(imp.Rel + "|" + imp.ProjectRoot)
	return envexec.EnvironmentIdentity{
		SchemaVersion:  envexec.IdentitySchemaVersion,
		Source:         envexec.SourceDLX,
		GraphDigest:    graph,
		MaterialDigest: mat,
		SourceDigest:   envexec.HashHex("local|" + imp.Rel),
		Platform:       envexec.CurrentPlatform(),
		LinkerMode:     "isolated",
	}
}

func discoverProjectRoot(ac *Context) (string, error) {
	if ac == nil {
		return "", apperr.New(apperr.Internal, "app.envexec", "", "missing app context")
	}
	cwd := ac.CWD
	if cwd == "" {
		var err error
		cwd, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	if root, err := project.FindRoot(cwd); err == nil {
		return root, nil
	}
	return cwd, nil
}
