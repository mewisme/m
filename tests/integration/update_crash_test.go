//go:build crash

package integration_test

import (
	"testing"
)

func TestUpdateCrashMatrix(t *testing.T) {
	for _, crashAt := range installCrashBoundaries {
		crashAt := crashAt
		t.Run(crashAt, func(t *testing.T) {
			runCrashScenario(t, crashScenario{
				flow:    crashFlowUpdate,
				crashAt: crashAt,
			})
		})
	}
}

func TestUpdateCrashInstallWithoutManualRecover(t *testing.T) {
	runCrashScenario(t, crashScenario{
		flow:        crashFlowUpdate,
		crashAt:     "publish:0",
		skipRecover: true,
	})
}
