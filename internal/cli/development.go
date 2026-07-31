// Developer tooling contracts (doctor and related stubs).
package cli

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/presentation"
)

func newDevelopmentCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "development",
		Short: "Developer tooling",
		Long:  "Commands for contributors and CI (doctor, future local gates).",
	}
	cmd.AddCommand(newDoctorCmd())
	return cmd
}

func newDoctorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Check local development prerequisites (stub)",
		Long:  "Reports Go version and pinned tool expectations. Full diagnostics are documented in docs/development-doctor.md.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected arguments: %v", args)
			}
			return runDoctor(cmd)
		},
	}
	cmd.AddCommand(newDoctorFilesystemCmd())
	return cmd
}

func newDoctorFilesystemCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "filesystem",
		Short: "Probe filesystem link capabilities for store and node_modules",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return fmt.Errorf("missing app context")
			}
			rep, err := app.DoctorFilesystem(cmd.Context(), ac)
			if err != nil {
				return err
			}
			_, err = fmt.Fprint(cmd.OutOrStdout(), app.FormatFilesystemProbe(rep))
			return err
		},
	}
}

type devDoctorCheck struct {
	Name   string
	Status presentation.Status
	Value  string
	Detail string
}

func runDoctor(cmd *cobra.Command) error {
	versions := loadVersionsEnv()
	goVer := runtime.Version()

	hasStub := false

	goCheck := devDoctorCheck{
		Name:   "Go",
		Value:  goVer,
		Detail: "(>= go1.26.5)",
		Status: presentation.StatusSuccess,
	}

	golangciLintValue := "unknown"
	if v, ok := versions["GOLANGCI_LINT_VERSION"]; ok {
		golangciLintValue = v
	}
	hasStub = true
	golangciLintCheck := devDoctorCheck{
		Name:   "golangci-lint",
		Value:  golangciLintValue,
		Detail: "(stub)",
		Status: presentation.StatusWarning,
	}

	govulncheckValue := "unknown"
	if v, ok := versions["GOVULNCHECK_VERSION"]; ok {
		govulncheckValue = v
	}
	govulncheckCheck := devDoctorCheck{
		Name:   "govulncheck",
		Value:  govulncheckValue,
		Detail: "(stub)",
		Status: presentation.StatusWarning,
	}

	makeCheck := devDoctorCheck{
		Name:   "make",
		Value:  "stub",
		Detail: "(optional)",
		Status: presentation.StatusWarning,
	}

	checks := []devDoctorCheck{goCheck, golangciLintCheck, govulncheckCheck, makeCheck}

	overallStatus := presentation.StatusSuccess
	if hasStub {
		overallStatus = presentation.StatusWarning
	}

	metrics := make([]presentation.KeyValue, 0, len(checks))
	for _, c := range checks {
		metrics = append(metrics, presentation.KeyValue{
			Key:   c.Name,
			Value: c.Value + "  " + c.Detail,
		})
	}

	summary := presentation.Summary{
		Status:  overallStatus,
		Title:   "Development prerequisites checked",
		Metrics: metrics,
	}

	g := ownerFlags(cmd.Root())
	r := g.mustStaticRenderer(cmd)
	return writeStaticOut(cmd, r.Summary(summary))
}

func loadVersionsEnv() map[string]string {
	root, err := findModuleRoot()
	if err != nil {
		return nil
	}
	path := filepath.Join(root, "tools", "versions.env")
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer func() { _ = f.Close() }()
	out := make(map[string]string)
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		out[strings.TrimSpace(k)] = strings.TrimSpace(v)
	}
	return out
}

func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("module root not found")
		}
		dir = parent
	}
}
