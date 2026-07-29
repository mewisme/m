package fetch_test

import (
	"runtime"
	"testing"

	"github.com/mewisme/mew/internal/fetch"
)

func TestDefaultWorkersUsesNumCPUCap(t *testing.T) {
	w := fetch.DefaultWorkers()
	if w < 1 {
		t.Fatalf("workers=%d", w)
	}
	if w > 16 {
		t.Fatalf("workers=%d exceeds cap", w)
	}
	if runtime.NumCPU() <= 16 && w != runtime.NumCPU() {
		t.Fatalf("workers=%d want %d", w, runtime.NumCPU())
	}
}
