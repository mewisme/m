package lifecycle

import (
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker"
	"github.com/mewisme/m/internal/process"
)

// Script is one lifecycle script to run for an installed package.
type Script struct {
	PackageName string
	PackageKey  string
	PackageDir  string
	Name        string
	Command     string
	Integrity   string
}

// Plan is the ordered set of lifecycle scripts for one install.
type Plan struct {
	Scripts []Script
}

// InstallInput carries staged install context for lifecycle execution.
type InstallInput struct {
	ProjectRoot string
	NodeModules string
	Graph       *graph.Graph
	LinkPlan    *linker.Plan
	Config      *config.Effective
	Env         []string
	Trusted     *TrustStore
	Interactive bool
	Supervisor  process.ProcessSupervisor
	AuditPath   string
	CacheDir    string
}

// Result summarizes lifecycle execution for one install.
type Result struct {
	Ran     int
	Cached  int
	Skipped int
}
