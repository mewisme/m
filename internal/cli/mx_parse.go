package cli

import (
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/runner/dlx"
)

// MXInvocation is parsed mx argv after global flags.
type MXInvocation struct {
	ModeA           bool
	PackageSpecs    []dlx.PackageSpec
	Command         string
	ForwardedArgs   []string
	AssumeYes       bool
	Offline         bool
	CWD             string
	PackageFlagUsed bool
}

// ParseMXInvocation parses mx DLX argv. Unknown leading flags return ERR_M_USAGE.
func ParseMXInvocation(argv []string) (MXInvocation, error) {
	var inv MXInvocation
	i := skipMXLeadingArgs(argv)
	if i >= len(argv) {
		return inv, apperr.New(apperr.Usage, "mx.parse", "", "missing package selector or command")
	}
	if argv[i] == "--" {
		i++
	}
	if i >= len(argv) {
		return inv, apperr.New(apperr.Usage, "mx.parse", "", "missing package selector or command")
	}
	// Re-parse leading mx-only flags for invocation fields.
	for j := 0; j < len(argv); {
		arg := argv[j]
		if !strings.HasPrefix(arg, "-") {
			break
		}
		switch {
		case arg == "--":
			i = j + 1
			goto doneFlags
		case arg == "--yes":
			inv.AssumeYes = true
			j++
		case arg == "--offline":
			inv.Offline = true
			j++
		case arg == "-p", arg == "--package":
			if j+1 >= len(argv) {
				return inv, apperr.New(apperr.Usage, "mx.parse", arg, "missing package value")
			}
			spec, err := dlx.ParsePackageSpec(argv[j+1])
			if err != nil {
				return inv, err
			}
			inv.PackageSpecs = append(inv.PackageSpecs, spec)
			inv.PackageFlagUsed = true
			j += 2
		case arg == "--cwd":
			if j+1 >= len(argv) {
				return inv, apperr.New(apperr.Usage, "mx.parse", arg, "missing cwd value")
			}
			inv.CWD = argv[j+1]
			j += 2
		case arg == "--config", arg == "--debug", arg == "--no-color", arg == "--prefer-offline", arg == "--unsafe-diagnostics":
			if strings.Contains(arg, "=") {
				j++
			} else if j+1 < len(argv) && !strings.HasPrefix(argv[j+1], "-") {
				j += 2
			} else {
				j++
			}
		case strings.HasPrefix(arg, "--filter"):
			if strings.Contains(arg, "=") {
				j++
			} else if j+1 < len(argv) {
				j += 2
			} else {
				j++
			}
		case strings.HasPrefix(arg, "-"):
			return inv, apperr.New(apperr.Usage, "mx.parse", arg, "unknown mx flag")
		default:
			j++
		}
	}
doneFlags:
	if inv.PackageFlagUsed {
		inv.ModeA = false
		inv.Command = argv[i]
		inv.ForwardedArgs = append([]string(nil), argv[i+1:]...)
		if strings.TrimSpace(inv.Command) == "" {
			return inv, apperr.New(apperr.Usage, "mx.parse", "", "missing explicit command for -p mode")
		}
	} else {
		inv.ModeA = true
		spec, err := dlx.ParsePackageSpec(argv[i])
		if err != nil {
			return inv, err
		}
		inv.PackageSpecs = []dlx.PackageSpec{spec}
		inv.ForwardedArgs = append([]string(nil), argv[i+1:]...)
	}
	if len(inv.PackageSpecs) == 0 {
		return inv, apperr.New(apperr.Usage, "mx.parse", "", "missing package spec")
	}
	return inv, nil
}
