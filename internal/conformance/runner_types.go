package conformance

import "time"

const (
	RunnerMatrix              = "runner"
	RunnerManifestSchema      = 1
	RunnerReportSchemaVersion = 1
	WaiverSchemaVersion       = 1

	RunnerGracefulTerminationGrace = 5 * time.Second
	RunnerForceKillGrace           = 5 * time.Second
	RunnerExternalDeadlineGrace    = 10 * time.Second

	RunnerMaxStdoutBytes   = 4 * 1024 * 1024
	RunnerMaxStderrBytes   = 4 * 1024 * 1024
	RunnerDiagnosticTail   = 64 * 1024
	RunnerMaxJSONEventSize = 1024 * 1024
	RunnerMaxSuiteCount    = 256
	RunnerMaxExpectedTests = 256
)

var runnerGroupOrder = []string{
	"execution",
	"security",
	"dispatch",
	"workspace",
	"schema",
	"stress",
	"probe",
}

var validRunnerGroups = map[string]struct{}{
	"execution": {},
	"security":  {},
	"dispatch":  {},
	"workspace": {},
	"schema":    {},
	"stress":    {},
	"probe":     {},
}

var validIsolationPolicies = map[string]struct{}{
	"fresh": {},
}

var validNetworkPolicies = map[string]struct{}{
	"forbidden":     {},
	"loopback-only": {},
	"local-fixture": {},
}

var validWaiverPolicies = map[string]struct{}{
	"forbidden": {},
	"allowed":   {},
}

const (
	RunnerResultPass           = "pass"
	RunnerResultFail           = "fail"
	RunnerResultTimeout        = "timeout"
	RunnerResultSkip           = "skip"
	RunnerResultNotApplicable  = "not-applicable"
	RunnerResultPassWithWaiver = "pass-with-waiver"
	RunnerResultProbePass      = "probe-pass"
	RunnerResultProbeSkip      = "probe-skip"
	RunnerResultProbeFail      = "probe-fail"
	RunnerResultNotRun         = "not-run"
)
