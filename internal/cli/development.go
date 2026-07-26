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
	return &cobra.Command{
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
}

func runDoctor(cmd *cobra.Command) error {
	out := cmd.OutOrStdout()
	versions := loadVersionsEnv()

	goVer := runtime.Version()
	fmt.Fprintf(out, "status=ok check=go value=%s want>=go1.26.5\n", goVer)

	if v, ok := versions["GOLANGCI_LINT_VERSION"]; ok {
		fmt.Fprintf(out, "status=stub check=golangci-lint want=%s note=install_via_tools/install\n", v)
	} else {
		fmt.Fprintln(out, "status=missing check=golangci-lint note=tools/versions.env_unreadable")
	}
	if v, ok := versions["GOVULNCHECK_VERSION"]; ok {
		fmt.Fprintf(out, "status=stub check=govulncheck want=%s note=install_via_tools/install\n", v)
	} else {
		fmt.Fprintln(out, "status=missing check=govulncheck note=tools/versions.env_unreadable")
	}

	fmt.Fprintln(out, "status=stub check=make note=optional_see_CONTRIBUTING")
	fmt.Fprintln(out, "doctor=stub see=docs/development-doctor.md")
	return nil
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
	defer f.Close()
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
