package cli

import (
	"context"
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
		sess.Close()
		return nil, apperr.Wrap(apperr.RuntimeNodeStart, "cli.transform", entrypoint,
			fmt.Errorf("transform service health check: %w", err))
	}

	// The loader-register.mjs asset is extracted to the runtime cache by
	// Plan() -> EnsureAssets(). Since its role is "loader-registration"
	// (Injected()=true), it is automatically added to Node argv via --import.
	// We only need to pass env overlay (endpoint + token) and cleanup hook.

	return &runtime.LaunchContribution{
		ExtraEnv:    sess.EnvOverlay(),
		CleanupHook: func() error { return sess.Close() },
	}, nil
}
