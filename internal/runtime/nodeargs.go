package runtime

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// nodeFlagsTakingValue is a small table of Node/V8 flags that consume a next-arg value.
var nodeFlagsTakingValue = map[string]bool{
	"--eval":                              true,
	"--print":                             true,
	"--require":                           true,
	"--import":                            true,
	"--experimental-loader":               true,
	"--experimental-specifier-resolution": true,
	"--title":                             true,
	"--max-old-space-size":                true,
	"--max-semi-space-size":               true,
	"--max-heap-size":                     true,
	"--heap-prof":                         true,
	"--diagnostic-dir":                    true,
	"--icu-data-dir":                      true,
	"--openssl-config":                    true,
	"--policy-integrity":                  true,
	"--redirect-warnings":                 true,
	"--report-directory":                  true,
	"--report-filename":                   true,
	"--snapshot-blob":                     true,
	"--test-reporter":                     true,
	"--watch-path":                        true,
	"--dns-result-order":                  true,
	"--env-file":                          true,
	"--node-memory-debug":                 true,
	"--security-revert":                   true,
	"--experimental-sea-config":           true,
	"--experimental-policy":               true,
	"--http-parser":                       true,
}

// ParseNodeArgs partitions node-args output:
//
//	m node-args -- <node/v8 flags> <entrypoint> [-- app-args]
//
// Returns nodeV8Args, entrypoint, appArgs.
func ParseNodeArgs(rawArgs []string) (v8Args []string, entrypoint string, appArgs []string, err error) {
	args := stripLeadingDashDash(rawArgs)
	if len(args) == 0 {
		return nil, "", nil, apperr.New(apperr.Usage, "runtime.node-args", "", "no arguments after --")
	}

	i := 0
	for i < len(args) {
		arg := args[i]
		// Check if this looks like an entrypoint (not a flag, has supported extension)
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			// This is the entrypoint
			entrypoint = arg
			i++
			break
		}
		// It's a flag. Determine if it takes a value.
		name, hasInlineValue := splitNodeFlag(arg)
		v8Args = append(v8Args, arg)
		i++
		if !hasInlineValue && nodeFlagsTakingValue[name] {
			if i >= len(args) {
				return nil, "", nil, apperr.New(apperr.Usage, "runtime.node-args", name,
					"flag requires a value")
			}
			v8Args = append(v8Args, args[i])
			i++
		}
	}

	if entrypoint == "" {
		return nil, "", nil, apperr.New(apperr.RuntimeEntrypoint, "runtime.node-args", "", "no entrypoint found in arguments")
	}

	// Remaining args after entrypoint become app-level args
	// Check for -- separator
	rest := args[i:]
	if len(rest) > 0 && rest[0] == "--" {
		appArgs = append([]string(nil), rest[1:]...)
	} else {
		appArgs = append([]string(nil), rest...)
	}

	return v8Args, entrypoint, appArgs, nil
}

func stripLeadingDashDash(args []string) []string {
	if len(args) > 0 && args[0] == "--" {
		return args[1:]
	}
	return args
}

func splitNodeFlag(arg string) (name string, hasInlineValue bool) {
	if !strings.HasPrefix(arg, "--") {
		if strings.HasPrefix(arg, "-") && len(arg) == 2 {
			return arg, false
		}
		return arg, false
	}
	body := arg[2:]
	if idx := strings.IndexByte(body, '='); idx >= 0 {
		return "--" + body[:idx], true
	}
	return arg, false
}
