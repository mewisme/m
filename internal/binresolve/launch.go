package binresolve

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
)

// BuildLaunch converts a candidate into a platform launch plan (test/API alias).
func BuildLaunch(cand binmeta.BinCandidate, childArgs []string, workDir string, hostEnv []string) (LaunchSpec, error) {
	return BuildLaunchSpec(cand, childArgs, hostEnv, workDir)
}

// BuildLaunchSpec converts a validated candidate into a platform launch plan.
func BuildLaunchSpec(cand binmeta.BinCandidate, childArgs, hostEnv []string, workDir string) (LaunchSpec, error) {
	if workDir == "" {
		workDir = filepath.Dir(cand.ShimPath)
	}
	if runtime.GOOS == "windows" {
		return buildWindowsLaunch(cand, childArgs, hostEnv, workDir)
	}
	return buildUnixLaunch(cand, childArgs, hostEnv, workDir)
}

func buildWindowsLaunch(cand binmeta.BinCandidate, childArgs, hostEnv []string, workDir string) (LaunchSpec, error) {
	shim := cand.ShimPath
	if !strings.EqualFold(filepath.Ext(shim), ".cmd") {
		if st, err := os.Stat(shim + ".cmd"); err == nil && !st.IsDir() {
			shim = shim + ".cmd"
		}
	}
	comspec, err := ResolveComSpec(hostEnv)
	if err != nil {
		return LaunchSpec{}, err
	}
	quoted := quoteWindows(shim)
	for _, arg := range childArgs {
		quoted += " " + quoteWindows(arg)
	}
	return LaunchSpec{
		Program: comspec,
		Args:    []string{"/d", "/s", "/c", quoted},
		Dir:     workDir,
		Kind:    LaunchCmd,
	}, nil
}

func buildUnixLaunch(cand binmeta.BinCandidate, childArgs, hostEnv []string, workDir string) (LaunchSpec, error) {
	target := cand.TargetPath
	if target == "" {
		target = cand.ShimPath
	}
	if kind, node, script, err := inspectUnixTarget(target, hostEnv); err != nil {
		return LaunchSpec{}, err
	} else if kind == LaunchNode {
		args := append([]string{script}, childArgs...)
		return LaunchSpec{Program: node, Args: args, Dir: workDir, Kind: LaunchNode}, nil
	} else if kind == LaunchDirect {
		args := append([]string{}, childArgs...)
		return LaunchSpec{Program: script, Args: args, Dir: workDir, Kind: LaunchDirect}, nil
	}
	return LaunchSpec{}, apperr.New(apperr.Exec, "binresolve.launch", cand.Command, "unable to determine launch program")
}

func inspectUnixTarget(path string, hostEnv []string) (LaunchKind, string, string, error) {
	st, err := os.Stat(path)
	if err != nil {
		return "", "", "", apperr.Wrap(apperr.IO, "binresolve.launch", path, err)
	}
	f, err := os.Open(path)
	if err != nil {
		return "", "", "", apperr.Wrap(apperr.IO, "binresolve.launch", path, err)
	}
	defer func() { _ = f.Close() }()
	line, err := readShebangLine(f)
	if err != nil {
		return "", "", "", err
	}
	if line != "" {
		interp := shebangInterpreter(line)
		if isNodeInterpreter(interp) {
			node, err := TrustedNodePath(hostEnv)
			if err != nil {
				return "", "", "", err
			}
			return LaunchNode, node, path, nil
		}
	}
	if !st.IsDir() && st.Mode()&0o111 != 0 {
		return LaunchDirect, path, path, nil
	}
	if line == "" {
		return LaunchDirect, path, path, nil
	}
	interp := shebangInterpreter(line)
	return "", "", "", apperr.New(apperr.Unsupported, "binresolve.launch", interp, "unsupported interpreter shebang")
}

func shebangInterpreter(line string) string {
	parts := strings.Fields(strings.TrimSpace(line))
	if len(parts) == 0 {
		return ""
	}
	if len(parts) >= 2 && (parts[0] == "env" || strings.HasSuffix(parts[0], "/env")) {
		return parts[1]
	}
	if len(parts) >= 2 {
		return parts[1]
	}
	return parts[0]
}

func isNodeInterpreter(interp string) bool {
	return interp == "node" || strings.HasSuffix(interp, "/node") || strings.HasSuffix(interp, "/node.exe")
}
func readShebangLine(r io.Reader) (string, error) {
	br := bufio.NewReader(r)
	b, err := br.ReadBytes('\n')
	if err != nil && err != io.EOF {
		return "", apperr.Wrap(apperr.IO, "binresolve.launch", "shebang", err)
	}
	line := strings.TrimSpace(string(b))
	if !strings.HasPrefix(line, "#!") {
		return "", nil
	}
	return strings.TrimSpace(strings.TrimPrefix(line, "#!")), nil
}

func quoteWindows(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"&|<>^") {
		return s
	}
	var b bytes.Buffer
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// LaunchToProcessSpec converts LaunchSpec to process.Spec fields.
func LaunchToProcessSpec(spec LaunchSpec) (path string, args []string) {
	return spec.Program, append([]string(nil), spec.Args...)
}

// FormatLaunchGolden is a test helper returning stable launch description.
func FormatLaunchGolden(spec LaunchSpec) string {
	return fmt.Sprintf("kind=%s program=%s args=%q dir=%s", spec.Kind, spec.Program, spec.Args, spec.Dir)
}
