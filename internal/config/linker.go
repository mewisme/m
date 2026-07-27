package config

import (
	"os"

	"github.com/mewisme/m/internal/apperr"
)

// ResolveLinkerMode returns hoisted or isolated for install.
// Priority: frozen lock linker > CLI/config install.linker. auto maps to hoisted.
func ResolveLinkerMode(eff *Effective, lockLinker string, frozen bool) (string, error) {
	raw := String(eff, "install.linker", "auto")
	if frozen && lockLinker != "" {
		raw = lockLinker
	}
	switch raw {
	case "", "auto":
		return "hoisted", nil
	case "hoisted":
		return "hoisted", nil
	case "isolated":
		if os.Getenv("MEW_EXPERIMENTAL_ISOLATED_LINKER") != "1" {
			return "", apperr.New(apperr.Usage, "config.linker", "isolated",
				"isolated linker requires MEW_EXPERIMENTAL_ISOLATED_LINKER=1")
		}
		return "isolated", nil
	default:
		return "", apperr.New(apperr.Usage, "config.linker", raw, "want auto|hoisted|isolated")
	}
}

// UseIsolatedLinker reports whether install should use the isolated virtual store layout.
func UseIsolatedLinker(eff *Effective, lockLinker string, frozen bool) bool {
	mode, err := ResolveLinkerMode(eff, lockLinker, frozen)
	return err == nil && mode == "isolated"
}
