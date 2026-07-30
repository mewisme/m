package prompt

import (
	"fmt"
	"strings"
)

// InteractivePolicy controls when prompts may run.
type InteractivePolicy string

const (
	InteractiveAuto   InteractivePolicy = "auto"
	InteractiveAlways InteractivePolicy = "always"
	InteractiveNever  InteractivePolicy = "never"
)

// ParseInteractivePolicy maps flag/config/env values.
func ParseInteractivePolicy(s string) (InteractivePolicy, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "auto":
		return InteractiveAuto, nil
	case "always":
		return InteractiveAlways, nil
	case "never":
		return InteractiveNever, nil
	default:
		return "", fmt.Errorf("invalid interactive policy %q", s)
	}
}

// Caps is the capability snapshot used by ResolveInteractive.
type Caps struct {
	StdinTTY          bool
	HumanMode         bool // false for JSON/NDJSON/silent
	CI                bool
	ChildOwnsTerminal bool
	Accessible        bool
	AccessibleOK      bool // accessible adapter available
	RichOK            bool // rich Huh adapter available
}

// Decision is the resolved prompt permission for one invocation.
type Decision struct {
	CanPrompt      bool
	UseAccessible  bool
	NeedTTY        bool
	DeniedByPolicy bool
	Reason         string
}

// ResolveInteractive applies auto/always/never against terminal capabilities.
// Always still requires a usable stdin TTY (fail closed, never hang).
func ResolveInteractive(policy InteractivePolicy, caps Caps) Decision {
	adapterOK := caps.AccessibleOK || caps.RichOK
	useAccessible := caps.Accessible || !caps.RichOK
	if useAccessible && !caps.AccessibleOK {
		adapterOK = false
	}
	if !useAccessible && !caps.RichOK {
		if caps.AccessibleOK {
			useAccessible = true
			adapterOK = true
		} else {
			adapterOK = false
		}
	}

	base := Decision{UseAccessible: useAccessible}

	switch policy {
	case InteractiveNever:
		base.DeniedByPolicy = true
		base.Reason = "interactive=never"
		return base
	case InteractiveAlways, InteractiveAuto:
		if !caps.StdinTTY {
			base.NeedTTY = true
			base.Reason = "stdin is not a TTY"
			return base
		}
		if policy == InteractiveAuto {
			if !caps.HumanMode {
				base.DeniedByPolicy = true
				base.Reason = "structured or silent output mode"
				return base
			}
			if caps.CI {
				base.DeniedByPolicy = true
				base.Reason = "CI environment"
				return base
			}
			if caps.ChildOwnsTerminal {
				base.DeniedByPolicy = true
				base.Reason = "child owns the terminal"
				return base
			}
		}
		if !adapterOK {
			base.DeniedByPolicy = true
			base.Reason = "no prompt adapter available"
			return base
		}
		base.CanPrompt = true
		return base
	default:
		base.DeniedByPolicy = true
		base.Reason = "unknown interactive policy"
		return base
	}
}

// SafeDefaultID returns DefaultID for empty Enter / EOF when it is fail-closed safe.
// Dangerous prompts never accept a non-deny default on EOF.
func SafeDefaultID(req PromptRequest) string {
	id := strings.TrimSpace(req.DefaultID)
	if id == "" {
		return ""
	}
	if !req.Dangerous {
		return id
	}
	switch id {
	case OptionDeny, OptionReject:
		return id
	default:
		return ""
	}
}
