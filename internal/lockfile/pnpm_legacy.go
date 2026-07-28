package lockfile

import "fmt"

// SupportedPnpmMajors are the only pnpm producer majors Mew supports for lock bridge.
var SupportedPnpmMajors = []int{9, 10, 11}

// PnpmLegacyUnsupportedError reports pnpm 5–8 lockfiles with remediation guidance.
type PnpmLegacyUnsupportedError struct {
	DetectedVersion string
	Layout          string
	SupportedMajors []int
	Remediation     string
}

func (e *PnpmLegacyUnsupportedError) Error() string {
	if e == nil {
		return "unsupported legacy pnpm lockfile"
	}
	return fmt.Sprintf(
		"pnpm lockfile version %q (layout=%s) is not supported; supported majors: %v; %s",
		e.DetectedVersion, e.Layout, e.SupportedMajors, e.Remediation,
	)
}

// NewPnpmLegacyUnsupported builds a typed legacy rejection payload.
func NewPnpmLegacyUnsupported(detectedVersion, layout string) *PnpmLegacyUnsupportedError {
	return &PnpmLegacyUnsupportedError{
		DetectedVersion: detectedVersion,
		Layout:          layout,
		SupportedMajors: append([]int(nil), SupportedPnpmMajors...),
		Remediation:     "regenerate pnpm-lock.yaml with pnpm 9, 10, or 11",
	}
}
