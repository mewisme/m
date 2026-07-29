package fetch

import "runtime"

// DefaultWorkers returns runtime.NumCPU capped at 16 (minimum 1).
func DefaultWorkers() int {
	// ponytail: cap at 16; NumCPU when unset
	w := runtime.NumCPU()
	if w > 16 {
		w = 16
	}
	if w < 1 {
		w = 1
	}
	return w
}
