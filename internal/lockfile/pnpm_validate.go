package lockfile

import (
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
		if _, err := ParsePMDeclaration(field.name, field.value); err != nil {
			return apperr.New(apperr.Usage, "pnpm.major", field.name, err.Error())
		}
	}
	return nil
}

// parsePnpmMajorField extracts the pnpm major from a packageManager field value.
func parsePnpmMajorField(pm string) (int, error) {
	decl, err := ParsePMDeclaration("packageManager", pm)
	if err != nil {
		return 0, err
	}
	return decl.ProducerMajor, nil
}

// DetectPnpmForProject runs detection with manifest hints and explicit major validation.
func DetectPnpmForProject(data []byte, hints ProjectHints, explicitMajor int) (Detection, error) {
	if err := ValidatePnpmHints(hints, explicitMajor); err != nil {
		return Detection{}, err
	}
	return DetectPnpmWithContext(data, hints, explicitMajor)
}
