//go:build crash

package integration_test

import (
	"testing"
)

func TestSnapshotCrashMatrix(t *testing.T) {
	for _, crashAt := range installCrashBoundaries {
		crashAt := crashAt
		t.Run(crashAt, func(t *testing.T) {
			runCrashScenario(t, crashScenario{
				flow:    crashFlowRestore,
				crashAt: crashAt,
			})
		})
	}
}

func TestSnapshotCrashInstallWithoutManualRecover(t *testing.T) {
	runCrashScenario(t, crashScenario{
		flow:        crashFlowRestore,
		crashAt:     "publish:0",
		skipRecover: true,
	})
}
