package cli

import (
	"fmt"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/binmeta"
	"github.com/mewisme/mew/internal/binresolve"
	"github.com/mewisme/mew/internal/project"
)

func tryDirectBinDispatch(phase PhaseAResult, cwd string, directOn bool) (DispatchResult, bool) {
	if phase.BareM || phase.Selector == "" {
		return DispatchResult{}, false
	}
	if phase.Leading.recursive {
		return DispatchResult{
			Kind:      OutcomeUnknown,
			Err:       apperr.New(apperr.Usage, "dispatch", phase.Selector, "direct bin dispatch does not support -r/--recursive"),
			Canonical: phase.Selector,
		}, true
	}
	if len(phase.Leading.filter) > 1 {
		return DispatchResult{
			Kind:      OutcomeUnknown,
			Err:       apperr.New(apperr.Usage, "dispatch", phase.Selector, "direct bin dispatch requires --filter to match exactly one workspace member"),
			Canonical: phase.Selector,
		}, true
	}

	root, err := project.FindRoot(cwd)
	if err != nil {
		return DispatchResult{}, false
	}
	pkgDir := root
	if phase.Leading.cwd != "" {
		if abs, err := filepath.Abs(phase.Leading.cwd); err == nil {
			pkgDir = abs
		}
	}

	if directOn {
		opts := binresolve.Options{
			ProjectRoot:     root,
			PackageDir:      pkgDir,
			Command:         phase.Selector,
			RequireVerified: true,
			RequestCache:    map[string]*binmeta.Document{},
		}
		_, err := binresolve.Resolve(opts)
		if err == nil {
			inv, ierr := BuildBinInvocation(phase)
			if ierr != nil {
				return DispatchResult{Kind: OutcomeUnknown, Err: ierr, Canonical: phase.Selector}, true
			}
			return DispatchResult{Kind: OutcomeBin, Canonical: phase.Selector, Bin: &inv, DirectGateOn: true}, true
		}
		code := apperr.CodeOf(err)
		switch code {
		case apperr.NotFound:
			// fall through to suggestions
		case apperr.Usage, apperr.Integrity, apperr.Exec, apperr.Timeout, apperr.Policy, apperr.Unsupported, apperr.PNPUnsupported, apperr.Cancelled:
			return DispatchResult{Kind: OutcomeUnknown, Err: err, Canonical: phase.Selector, DirectGateOn: true}, true
		default:
			return DispatchResult{Kind: OutcomeUnknown, Err: err, Canonical: phase.Selector, DirectGateOn: true}, true
		}
	} else {
		hintOpts := binresolve.Options{
			ProjectRoot:  root,
			PackageDir:   pkgDir,
			Command:      phase.Selector,
			RequestCache: map[string]*binmeta.Document{},
		}
		if _, ok, err := binresolve.CheapVerifiedHint(hintOpts); err != nil {
			return DispatchResult{Kind: OutcomeUnknown, Err: err, Canonical: phase.Selector}, true
		} else if ok {
			return DispatchResult{
				Kind:         OutcomeSuggest,
				Canonical:    phase.Selector,
				DirectGateOn: false,
				Err:          apperr.New(apperr.Usage, "dispatch", phase.Selector, gateOffExactBinMessage(phase.Selector)),
			}, true
		}
	}
	return DispatchResult{}, false
}

func BuildBinInvocation(phase PhaseAResult) (BinInvocation, error) {
	if phase.Leading.recursive {
		return BinInvocation{}, apperr.New(apperr.Usage, "dispatch", phase.Selector, "direct bin dispatch does not support -r/--recursive")
	}
	return BinInvocation{
		Selector:      phase.Selector,
		ForwardedArgs: append([]string(nil), phase.ForwardedArgs...),
		Filters:       append([]string(nil), phase.Leading.filter...),
	}, nil
}

func gateOffExactBinMessage(bin string) string {
	return fmt.Sprintf("Direct bin dispatch is disabled.\n\nRun it explicitly with:\n  m exec %s", bin)
}
