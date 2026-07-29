package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/pack"
	"github.com/mewisme/mew/internal/project"
)

// PackOptions configures m pack.
type PackOptions struct {
	PackDestination string
	PackageDir      string // default project root
}

// PackResult is the app-level pack summary.
type PackResult struct {
	TarballPath string   `json:"tarball"`
	Files       []string `json:"files"`
}

// Pack creates a deterministic package tarball.
func Pack(ctx context.Context, ac *Context, opts PackOptions) (PackResult, error) {
	if ac == nil {
		return PackResult{}, apperr.New(apperr.Internal, "app.pack", "", "missing app context")
	}
	root := opts.PackageDir
	if root == "" {
		var err error
		root, err = project.FindRoot(ac.CWD)
		if err != nil {
			return PackResult{}, err
		}
	}
	dest := opts.PackDestination
	if dest == "" {
		dest = ac.CWD
	}
	res, err := pack.Pack(ctx, pack.Options{Root: root, PackDestination: dest})
	if err != nil {
		return PackResult{}, err
	}
	return PackResult{TarballPath: res.TarballPath, Files: res.Files}, nil
}

// FormatPackLine returns human-readable pack output.
func FormatPackLine(r PackResult) string {
	return fmt.Sprintf("%s\n", filepath.Base(r.TarballPath))
}

// ResolvePackTarball returns an absolute tarball path from CLI args or packs first.
func ResolvePackTarball(ctx context.Context, ac *Context, tarballArg string, packOpts PackOptions) (string, error) {
	if tarballArg != "" {
		abs, err := filepath.Abs(tarballArg)
		if err != nil {
			return "", apperr.Wrap(apperr.IO, "app.publish", tarballArg, err)
		}
		if st, err := os.Stat(abs); err != nil || st.IsDir() {
			if err != nil {
				return "", apperr.Wrap(apperr.IO, "app.publish", abs, err)
			}
			return "", apperr.New(apperr.Usage, "app.publish", abs, "tarball path must be a file")
		}
		if !strings.HasSuffix(strings.ToLower(abs), ".tgz") {
			return "", apperr.New(apperr.Usage, "app.publish", abs, "tarball must end with .tgz")
		}
		return abs, nil
	}
	res, err := Pack(ctx, ac, packOpts)
	if err != nil {
		return "", err
	}
	return res.TarballPath, nil
}
