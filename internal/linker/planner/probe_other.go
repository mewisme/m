//go:build !linux && !darwin && !windows

package planner

func platformSameVolume(a, b string) bool {
	return a == b
}

func platformProbeReflink(_, _ string) bool {
	return false
}

func platformProbeJunction(_ string) bool {
	return false
}
