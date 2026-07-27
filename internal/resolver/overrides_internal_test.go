package resolver

import "testing"

func TestMatchOverride(t *testing.T) {
	t.Parallel()
	overrides := map[string]string{
		"lodash":         "4.17.21",
		"webpack.loader": "1.0.0",
	}
	if spec, ok := matchOverride(overrides, []string{"webpack"}, "loader"); !ok || spec != "1.0.0" {
		t.Fatalf("nested override got %q ok=%v", spec, ok)
	}
	if spec, ok := matchOverride(overrides, []string{"other"}, "lodash"); !ok || spec != "4.17.21" {
		t.Fatalf("global override got %q ok=%v", spec, ok)
	}
}

func TestParsePackageKey(t *testing.T) {
	t.Parallel()
	id := parsePackageKey("react@18.2.0#react-dom@18.2.0")
	if id.Name != "react" || id.Version != "18.2.0" {
		t.Fatalf("base id=%#v", id)
	}
	if len(id.PeerProviderContext) != 1 || id.PeerProviderContext[0].Name != "react-dom" {
		t.Fatalf("peer providers=%#v", id.PeerProviderContext)
	}
}
