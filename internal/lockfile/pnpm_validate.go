package lockfile

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// ValidatePnpmProducerMajor rejects unsupported pnpm producer majors.
func ValidatePnpmProducerMajor(major int) error {
	if major == 0 {
		return nil
	}
	if major < 9 || major > 11 {
		return apperr.New(apperr.Usage, "pnpm.major", strconv.Itoa(major),
			"pnpm producer major must be 9, 10, or 11")
	}
	return nil
}

// ValidatePnpmHints checks packageManager/devEngines fields and explicit major flag.
func ValidatePnpmHints(hints ProjectHints, explicitMajor int) error {
	if err := ValidatePnpmProducerMajor(explicitMajor); err != nil {
		return err
	}
	for _, field := range []struct {
		name, value string
	}{
		{"packageManager", hints.PackageManager},
		{"devEngines.packageManager", hints.DevEnginesPM},
	} {
		if strings.TrimSpace(field.value) == "" {
			continue
		}
		major, err := parsePnpmMajorField(field.value)
		if err != nil {
			return apperr.New(apperr.Usage, "pnpm.major", field.name, err.Error())
		}
		if major != 0 && (major < 9 || major > 11) {
			return apperr.New(apperr.Usage, "pnpm.major", field.name,
				fmt.Sprintf("pnpm@%d is unsupported; regenerate with pnpm 9, 10, or 11", major))
		}
	}
	return nil
}

// parsePnpmMajorField extracts the pnpm major from a packageManager field value.
func parsePnpmMajorField(pm string) (int, error) {
	pm = strings.TrimSpace(pm)
	if pm == "" {
		return 0, nil
	}
	name := pm
	if i := strings.IndexByte(pm, '@'); i >= 0 {
		name = pm[:i]
	}
	if strings.ToLower(strings.TrimSpace(name)) != "pnpm" {
		return 0, fmt.Errorf("expected pnpm@<major>, got %q", pm)
	}
	at := strings.LastIndexByte(pm, '@')
	if at < 0 || at >= len(pm)-1 {
		return 0, nil
	}
	ver := strings.TrimSpace(pm[at+1:])
	if ver == "" {
		return 0, nil
	}
	if strings.ContainsAny(ver, "^~>=< *xX") {
		return 0, fmt.Errorf("version ranges and tags are not supported in %q", pm)
	}
	dot := strings.IndexByte(ver, '.')
	if dot > 0 {
		ver = ver[:dot]
	}
	n, err := strconv.Atoi(ver)
	if err != nil {
		return 0, fmt.Errorf("unrecognized pnpm version %q", pm)
	}
	return n, nil
}

// DetectPnpmForProject runs detection with manifest hints and explicit major validation.
func DetectPnpmForProject(data []byte, hints ProjectHints, explicitMajor int) (Detection, error) {
	if err := ValidatePnpmHints(hints, explicitMajor); err != nil {
		return Detection{}, err
	}
	return DetectPnpmWithContext(data, hints, explicitMajor)
}
