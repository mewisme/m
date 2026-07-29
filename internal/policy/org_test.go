package policy_test

import (
	"testing"
	"time"

	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/policy"
)

func TestEvaluateDenyLicense(t *testing.T) {
	g, err := graph.NewBuilder().
		Importer(graph.RootImporter, "root").
		Package(graph.PackageID{Name: "gpl-pkg", Version: "1.0.0"}, "", "").
		Edge(string(graph.RootImporter), "gpl-pkg@1.0.0", graph.DepProd, "1.0.0").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	org := &policy.OrgPolicy{
		DenyLicenses:      []string{"GPL"},
		SeverityThreshold: policy.SeverityError,
	}
	result := policy.Evaluate(g, map[string]string{"gpl-pkg@1.0.0": "GPL-3.0"}, org)
	if result.Passed {
		t.Fatal("expected failure")
	}
	if len(result.Violations) != 1 || result.Violations[0].Rule != "denied_license" {
		t.Fatalf("violations=%+v", result.Violations)
	}
}

func TestEvaluateWaiverAllows(t *testing.T) {
	g, err := graph.NewBuilder().
		Package(graph.PackageID{Name: "gpl-pkg", Version: "1.0.0"}, "", "").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	org := &policy.OrgPolicy{
		DenyLicenses: []string{"GPL"},
		Waivers: []policy.Waiver{{
			Package: "gpl-pkg",
			Reason:  "approved exception",
			Expires: time.Now().Add(24 * time.Hour).Format(time.RFC3339),
		}},
	}
	result := policy.Evaluate(g, map[string]string{"gpl-pkg@1.0.0": "GPL-3.0"}, org)
	if !result.Passed || len(result.Violations) != 0 {
		t.Fatalf("expected pass, got %+v", result)
	}
}

func TestEvaluateDenyPackage(t *testing.T) {
	g, err := graph.NewBuilder().
		Package(graph.PackageID{Name: "blocked", Version: "1.0.0"}, "", "").
		Build()
	if err != nil {
		t.Fatal(err)
	}
	org := &policy.OrgPolicy{
		DenyPackages:      []string{"blocked"},
		SeverityThreshold: policy.SeverityError,
	}
	result := policy.Evaluate(g, nil, org)
	if result.Passed || len(result.Violations) != 1 {
		t.Fatalf("violations=%+v", result.Violations)
	}
}
