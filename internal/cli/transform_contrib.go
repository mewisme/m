package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/runtime"
	"github.com/mewisme/mew/internal/transform"
)

// isTypeScriptFile reports whether path has a .ts/.mts/.cts extension.
func isTypeScriptFile(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".ts", ".mts", ".cts":
		return true
	}
	return false
}

// buildTransformContribution creates a transform session and returns
// a LaunchContribution with the service endpoint, token, loader preload,
// and cleanup hook.
func buildTransformContribution(ctx context.Context, cwd, entrypoint string, eff *config.Effective) (*runtime.LaunchContribution, error) {
	cacheDir := transform.TransformCacheDir(eff)
	sess, err := transform.NewSession(transform.ServiceOptions{
		Engine:   transform.NewEsbuildEngine(),
		CacheDir: cacheDir,
		Workers:  4,
	})
	if err != nil {
		return nil, apperr.Wrap(apperr.Internal, "cli.transform", entrypoint,
			fmt.Errorf("starting transform service: %w", err))
	}

	// Start the listener using the command context for proper cancellation.
	if ctx == nil {
		ctx = context.Background()
	}
	if err := sess.Start(ctx); err != nil {
		_ = sess.Close()
		return nil, apperr.Wrap(apperr.RuntimeNodeStart, "cli.transform", entrypoint,
			fmt.Errorf("transform service health check: %w", err))
	}

	// Discover tsconfig chain for the entrypoint directory.
	entryDir := filepath.Dir(entrypoint)
	configPath, tsconfigErr := transform.DiscoverTsconfig(entryDir)
	var opts transform.NormalizedOptions
	var optsDigest string
	if tsconfigErr == nil && configPath != "" {
		chain, loadErr := transform.LoadTsconfigChain(configPath)
		if loadErr == nil && len(chain) > 0 {
			opts = transform.NormalizeOptions(chain)
			optsDigest = transform.TsconfigChainDigest(chain)
		}
	}
	// tsconfig errors are non-fatal; transforms proceed with default options.
	optsJSON, _ := json.Marshal(opts)

	// Pass tsconfig options through environment for the Node loader.
	extraEnv := sess.EnvOverlay()
	extraEnv = append(extraEnv,
		"MEW_TRANSFORM_OPTIONS="+string(optsJSON),
		"MEW_TRANSFORM_OPTS_DIGEST="+optsDigest,
	)

	return &runtime.LaunchContribution{
		ExtraEnv:    extraEnv,
		CleanupHook: func() error { return sess.Close() },
	}, nil
}
