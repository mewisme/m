package envexec

import (
	"context"
	"encoding/json"
	"sort"
)

// InspectSchemaVersion is the JSON schema version for m env inspect output.
const InspectSchemaVersion = 1

// InspectReport is the v1 inspect output schema.
type InspectReport struct {
	V                int             `json:"v"`
	Source           SourceKind      `json:"source"`
	IdentityDigest   string          `json:"identityDigest"`
	GraphDigest      string          `json:"graphDigest"`
	CacheState       CacheState      `json:"cacheState"`
	Materialized     bool            `json:"materialized"`
	WouldMaterialize bool            `json:"wouldMaterialize"`
	NetworkPolicy    NetworkPolicy   `json:"networkPolicy"`
	Verified         bool            `json:"verified"`
	Command          *InspectCommand `json:"command,omitempty"`
	Warnings         []Diagnostic    `json:"warnings"`
}

// InspectCommand describes one command in inspect output.
type InspectCommand struct {
	Name      string `json:"name"`
	Owner     string `json:"owner,omitempty"`
	Available bool   `json:"available"`
}

// InspectEnvironment plans an environment without materializing or executing.
func InspectEnvironment(ctx context.Context, deps ProviderDeps, reg ProviderRegistry, req ExecutionRequest) (InspectReport, error) {
	var empty InspectReport
	if err := ValidateInspectRequest(req); err != nil {
		return empty, err
	}
	if req.Policy.Network == "" {
		req.Policy = LockedProviderPolicy(req.Source.Kind())
	}
	provider, err := reg.providerFor(req)
	if err != nil {
		return empty, err
	}
	plan, err := provider.Plan(ctx, deps, req)
	if err != nil {
		return empty, err
	}
	report := InspectReport{
		V:                InspectSchemaVersion,
		Source:           plan.Source,
		IdentityDigest:   BareDigestHex(plan.Identity.IdentityDigest()),
		GraphDigest:      BareDigestHex(plan.GraphDigest),
		CacheState:       plan.CacheState,
		Materialized:     plan.CacheState == CacheWarm || plan.CacheState == CacheProject,
		WouldMaterialize: plan.Materialization == MaterializationRequired,
		NetworkPolicy:    req.Policy.Network,
		Verified:         true,
		Warnings:         []Diagnostic{},
	}
	sort.Slice(report.Warnings, func(i, j int) bool {
		if report.Warnings[i].Code != report.Warnings[j].Code {
			return report.Warnings[i].Code < report.Warnings[j].Code
		}
		return report.Warnings[i].Message < report.Warnings[j].Message
	})
	if req.Command.Name != "" {
		report.Command = &InspectCommand{Name: req.Command.Name, Owner: req.Command.OwnerDependency, Available: true}
	}
	return report, nil
}

// EncodeInspectReport JSON-encodes an inspect report deterministically.
func EncodeInspectReport(r InspectReport) ([]byte, error) {
	if r.Warnings == nil {
		r.Warnings = []Diagnostic{}
	}
	return json.MarshalIndent(r, "", "  ")
}
