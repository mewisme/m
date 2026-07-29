package app

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
	"github.com/mewisme/mew/internal/policy"
)

// CheckPolicy evaluates org policy against the project lock graph and installed licenses.
func CheckPolicy(ctx context.Context, ac *Context) (policy.PolicyResult, error) {
	var empty policy.PolicyResult
	if ac == nil {
		return empty, apperr.New(apperr.Internal, "app.policy", "", "missing app context")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return empty, err
	}
	org, err := policy.LoadOrgPolicy(proj.Root)
	if err != nil {
		return empty, err
	}
	if org == nil {
		return policy.PolicyResult{Passed: true}, nil
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return empty, err
	}
	if g == nil {
		return policy.PolicyResult{Passed: true}, nil
	}
	nmRoot := filepath.Join(proj.Root, "node_modules")
	licenses := policy.LicensesFromNodeModules(nmRoot, g)
	return policy.Evaluate(g, licenses, org), nil
}

// EnforceInstallPolicy blocks install when org policy violations exceed the severity threshold.
func EnforceInstallPolicy(ctx context.Context, ac *Context, g *graph.Graph, linkPlan *linker.Plan) error {
	if ac == nil || g == nil {
		return apperr.New(apperr.Internal, "app.policy", "", "missing context or graph")
	}
	proj, err := OpenProject(ctx, ac)
	if err != nil {
		return err
	}
	org, err := policy.LoadOrgPolicy(proj.Root)
	if err != nil {
		return err
	}
	if org == nil {
		return nil
	}
	var licenses map[string]string
	if linkPlan != nil && len(linkPlan.ExtractDirs) > 0 {
		licenses = policy.LicensesFromExtractDirs(linkPlan.ExtractDirs)
	} else {
		nmRoot := filepath.Join(proj.Root, "node_modules")
		if linkPlan != nil && linkPlan.NodeModules != "" {
			nmRoot = linkPlan.NodeModules
		}
		licenses = policy.LicensesFromNodeModules(nmRoot, g)
	}
	result := policy.Evaluate(g, licenses, org)
	if result.Passed {
		return nil
	}
	var msgs []string
	for _, v := range result.Violations {
		if v.Severity == policy.SeverityError {
			msgs = append(msgs, v.Message+" ("+v.Package+")")
		}
	}
	if len(msgs) == 0 {
		return nil
	}
	return apperr.New(apperr.Policy, "app.policy", "install",
		fmt.Sprintf("org policy violation: %s", strings.Join(msgs, "; ")))
}
