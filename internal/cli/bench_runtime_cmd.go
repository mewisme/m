package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/apperr"
)

func newBenchRuntimeCmd() *cobra.Command {
	var (
		cold   bool
		asJSON bool
	)
	cmd := &cobra.Command{
		Use:   "runtime",
		Short: "Benchmark runtime hot paths (transform, cache, launch)",
		Long:  "Run Go benchmarks for internal/runtime and internal/transform packages.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			repoRoot, err := findModuleRoot()
			if err != nil {
				return apperr.Wrap(apperr.Internal, "bench runtime", "", err)
			}

			if cold {
				cacheDir := filepath.Join(repoRoot, ".cache", "mew", "transform")
				if err := os.RemoveAll(cacheDir); err != nil {
					return apperr.Wrap(apperr.IO, "bench runtime", cacheDir, err)
				}
			}

			benchPkgs := []string{
				"./internal/runtime",
				"./internal/transform",
			}

			var allOut strings.Builder
			var errs []string
			for _, pkg := range benchPkgs {
				out, err := runBenchPkg(cmd.Context(), repoRoot, pkg)
				if err != nil {
					errs = append(errs, pkg+": "+err.Error())
					continue
				}
				allOut.Write(out)
				allOut.WriteByte('\n')
			}

			if asJSON {
				return writeStaticOut(cmd,
					fmt.Sprintf(`{"packages":["internal/runtime","internal/transform"],"cold":%v}`, cold))
			}

			for _, e := range errs {
				if err := writeStaticErr(cmd, e); err != nil {
					return err
				}
			}

			return writeStaticOut(cmd, allOut.String())
		},
	}
	cmd.Flags().BoolVar(&cold, "cold", false, "clear transform cache before benchmarking")
	cmd.Flags().BoolVar(&asJSON, "json", false, "emit JSON summary")
	return cmd
}

func runBenchPkg(ctx context.Context, repoRoot, pkg string) ([]byte, error) {
	args := []string{"test", pkg, "-bench=.", "-benchmem", "-count=1"}
	c := exec.CommandContext(ctx, "go", args...)
	c.Dir = repoRoot
	c.Env = append(os.Environ(), "CGO_ENABLED=0")
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		return nil, fmt.Errorf("%v: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}
