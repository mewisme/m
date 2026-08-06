package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/dotenv"
	"github.com/mewisme/mew/internal/runtime"
	"github.com/mewisme/mew/internal/watch"
)

func newWatchCmd() *cobra.Command {
	var (
		clearScreen   bool
		noClearScreen bool
		envFile       []string
		noEnvFile     bool
		mode          string
		debounceMS    int
	)
	cmd := &cobra.Command{
		Use:   "watch <entrypoint>",
		Short: "Watch files and restart on changes",
		Long: `Watch source files, configuration, and environment files for changes,
then restart the application automatically.

Watches the entrypoint directory, tsconfig.json, package.json, and
.env files. Restarts happen after a short debounce period (200ms default)
to coalesce rapid saves.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "watch", "", "missing app context")
			}

			entrypoint := args[0]
			appArgs := args[1:]
			cwd := ac.CWD

			epAbs := entrypoint
			if !filepath.IsAbs(epAbs) {
				epAbs = filepath.Join(cwd, epAbs)
			}

			// Determine clear-screen policy.
			clear := clearScreen
			if noClearScreen {
				clear = false
			} else if !clearScreen && !noClearScreen {
				if fi, err := os.Stdout.Stat(); err == nil && (fi.Mode()&os.ModeCharDevice) != 0 {
					clear = true
				}
			}

			// Build restart function that launches Node on each restart.
			restart := func(ctx context.Context) (int, error) {
				currentEnv := buildWatchEnvOverlay(cwd, envFile, noEnvFile, mode)

				plan, err := runtime.Plan(ctx, runtime.LaunchRequest{
					Entrypoint: epAbs,
					AppArgs:    appArgs,
					WorkingDir: cwd,
					EnvOverlay: currentEnv,
				}, ac.Config)
				if err != nil {
					fmt.Fprintf(os.Stderr, "watch: plan error: %v\n", err)
					return 1, err
				}

				req := runtime.LaunchRequest{
					Entrypoint: epAbs,
					AppArgs:    appArgs,
					WorkingDir: cwd,
					EnvOverlay: currentEnv,
					Stdio: runtime.LaunchStdio{
						Stdin:  os.Stdin,
						Stdout: os.Stdout,
						Stderr: os.Stderr,
					},
				}

				if err := runtime.Launch(ctx, plan, req); err != nil {
					code := apperr.ExitCode(err)
					if apperr.CodeOf(err) == apperr.Cancelled {
						return code, nil
					}
					return code, err
				}
				return 0, nil
			}

			// Collect watch paths.
			watchPaths, err := watch.CollectPaths(entrypoint, cwd)
			if err != nil {
				return err
			}
			for _, f := range envFile {
				p := f
				if !filepath.IsAbs(p) {
					p = filepath.Join(cwd, p)
				}
				watchPaths = append(watchPaths, p)
			}

			w, err := watch.NewWatcher()
			if err != nil {
				return apperr.Wrap(apperr.IO, "watch", entrypoint, err)
			}
			defer w.Close()

			debounce := watch.DefaultDebounceInterval
			if debounceMS > 0 {
				debounce = time.Duration(debounceMS) * time.Millisecond
			}

			sup := watch.NewSupervisor(watch.SupervisorOptions{
				Watcher:          w,
				WatchPaths:       watchPaths,
				Restart:          restart,
				ClearScreen:      clear,
				DebounceInterval: debounce,
				OnRestart: func(reason string) {
					fmt.Fprintf(os.Stderr, "\n[watch] restarting: %s\n\n", reason)
				},
			})

			code, err := sup.Run(cmd.Context())
			if err != nil && err != context.Canceled {
				return err
			}
			_ = code
			return nil
		},
	}

	cmd.Flags().BoolVar(&clearScreen, "clear-screen", false, "clear terminal before each restart")
	cmd.Flags().BoolVar(&noClearScreen, "no-clear-screen", false, "never clear terminal")
	cmd.Flags().StringArrayVar(&envFile, "env-file", nil, "path to .env file (repeatable)")
	cmd.Flags().BoolVar(&noEnvFile, "no-env-file", false, "skip .env file auto-discovery")
	cmd.Flags().StringVar(&mode, "mode", "", "mode for .env file discovery (sets NODE_ENV)")
	cmd.Flags().IntVar(&debounceMS, "debounce", 0, "debounce interval in milliseconds")
	return cmd
}

func buildWatchEnvOverlay(cwd string, envFile []string, noEnvFile bool, mode string) []string {
	if noEnvFile && len(envFile) == 0 {
		if mode != "" {
			return []string{"NODE_ENV=" + mode}
		}
		return nil
	}

	var files []string
	if len(envFile) > 0 {
		for _, f := range envFile {
			if filepath.IsAbs(f) {
				files = append(files, f)
			} else {
				files = append(files, filepath.Join(cwd, f))
			}
		}
	} else {
		files = dotenv.Discover(cwd, mode)
	}

	envVars, err := dotenv.Load(files)
	if err != nil {
		envVars = nil
	}

	if mode != "" {
		envVars = append(envVars, "NODE_ENV="+mode)
	}
	return envVars
}
