package runner_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestInspectSchemaProjectGolden(t *testing.T) {
	report := envexec.InspectReport{
		V:                envexec.InspectSchemaVersion,
		Source:           envexec.SourceProject,
		IdentityDigest:   strings.Repeat("a", 64),
		GraphDigest:      strings.Repeat("b", 64),
		CacheState:       envexec.CacheCold,
		Materialized:     false,
		WouldMaterialize: true,
		NetworkPolicy:    envexec.NetworkForbidden,
		Verified:         true,
		Warnings:         []envexec.Diagnostic{},
	}
	data, err := envexec.EncodeInspectReport(report)
	if err != nil {
		t.Fatal(err)
	}
	if warningsFieldPresentNil(data) {
		t.Fatal("must not emit null warnings field")
	}
}

func warningsFieldPresentNil(data []byte) bool {
	return strings.Contains(string(data), `"warnings":null`)
}

func TestInspectSchemaDLXGolden(t *testing.T) {
	report := envexec.InspectReport{
		V:              envexec.InspectSchemaVersion,
		Source:         envexec.SourceDLX,
		IdentityDigest: strings.Repeat("c", 64),
		GraphDigest:    strings.Repeat("d", 64),
		CacheState:     envexec.CacheWarm,
		NetworkPolicy:  envexec.NetworkMetadataOnly,
		Verified:       true,
		Command:        &envexec.InspectCommand{Name: "eslint", Owner: "eslint", Available: true},
		Warnings:       []envexec.Diagnostic{{Code: "warn", Message: "note"}},
	}
	data, err := envexec.EncodeInspectReport(report)
	if err != nil {
		t.Fatal(err)
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["command"]; !ok {
		t.Fatal("missing command")
	}
}

func TestInspectSchemaNoAbsolutePaths(t *testing.T) {
	report := envexec.InspectReport{
		V:              envexec.InspectSchemaVersion,
		Source:         envexec.SourceProject,
		IdentityDigest: strings.Repeat("e", 64),
		GraphDigest:    strings.Repeat("f", 64),
		CacheState:     envexec.CacheProject,
		NetworkPolicy:  envexec.NetworkForbidden,
		Verified:       true,
		Warnings:       []envexec.Diagnostic{{Code: "relative", Message: "packages/app"}},
	}
	data, err := envexec.EncodeInspectReport(report)
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if strings.Contains(s, ":\\") || strings.Contains(s, "C:/") || strings.Contains(s, "/home/") {
		t.Fatalf("absolute path leaked: %s", s)
	}
}
