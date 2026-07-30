package binresolve

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// TrustedNodePath returns an absolute path to stock Node.js for bin launch.
// It never executes /usr/bin/env and does not use project PATH for discovery.
func TrustedNodePath(hostEnv []string) (string, error) {
	if p, ok := lookupEnv(hostEnv, "MEW_NODE"); ok && p != "" {
		if abs, err := validateNodeExecutable(p); err == nil {
			return abs, nil
		}
	}
	if p, ok := lookupEnv(hostEnv, "MEW_TRUSTED_NODE"); ok && p != "" {
		if abs, err := validateNodeExecutable(p); err == nil {
			return abs, nil
		}
	}
	candidates := trustedNodeCandidates()
	for _, c := range candidates {
		if abs, err := validateNodeExecutable(c); err == nil {
			return abs, nil
		}
	}
	// ponytail: last resort LookPath for dev/test hosts; upgrade = dedicated node provisioner (0044+).
	path, err := exec.LookPath("node")
	if err != nil {
		return "", apperr.New(apperr.Exec, "binresolve.node", "node", "trusted Node.js not found")
	}
	return validateNodeExecutable(path)
}

func trustedNodeCandidates() []string {
	var out []string
	if runtime.GOOS == "windows" {
		for _, root := range []string{os.Getenv("ProgramFiles"), os.Getenv("ProgramFiles(x86)")} {
			if root != "" {
				out = append(out, filepath.Join(root, "nodejs", "node.exe"))
			}
		}
		return out
	}
	out = append(out, "/usr/local/bin/node", "/opt/homebrew/bin/node")
	return out
}

func validateNodeExecutable(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	st, err := os.Stat(abs)
	if err != nil {
		return "", err
	}
	if st.IsDir() {
		return "", apperr.New(apperr.Exec, "binresolve.node", abs, "not a file")
	}
	base := strings.ToLower(filepath.Base(abs))
	if base != "node" && base != "node.exe" {
		return "", apperr.New(apperr.Exec, "binresolve.node", abs, "not a node binary")
	}
	return abs, nil
}

func lookupEnv(env []string, key string) (string, bool) {
	for _, kv := range env {
		if i := strings.IndexByte(kv, '='); i >= 0 {
			if strings.EqualFold(kv[:i], key) {
				return kv[i+1:], true
			}
		}
	}
	return "", false
}
