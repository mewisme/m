package manifest_test

import (
	"testing"

	"github.com/mewisme/mew/internal/manifest"
)

func TestParseSpecifier(t *testing.T) {
	cases := []struct {
		key  string
		spec string
		want manifest.Specifier
	}{
		{
			key:  "lodash",
			spec: "^4.17.21",
			want: manifest.Specifier{DisplayName: "lodash", TargetName: "lodash", Range: "^4.17.21", Protocol: manifest.ProtocolRegistry},
		},
		{
			key:  "foo",
			spec: "npm:bar@^1.0.0",
			want: manifest.Specifier{DisplayName: "foo", TargetName: "bar", Range: "^1.0.0", Protocol: manifest.ProtocolNpm},
		},
		{
			key:  "alias",
			spec: "npm:foo@npm:bar@^1",
			want: manifest.Specifier{DisplayName: "foo", TargetName: "bar", Range: "^1", Protocol: manifest.ProtocolNpm},
		},
		{
			key:  "pkg",
			spec: "workspace:*",
			want: manifest.Specifier{DisplayName: "pkg", TargetName: "pkg", Range: "*", Protocol: manifest.ProtocolWorkspace},
		},
		{
			key:  "pkg",
			spec: "workspace:^",
			want: manifest.Specifier{DisplayName: "pkg", TargetName: "pkg", Range: "^", Protocol: manifest.ProtocolWorkspace},
		},
		{
			key:  "pkg",
			spec: "workspace:~",
			want: manifest.Specifier{DisplayName: "pkg", TargetName: "pkg", Range: "~", Protocol: manifest.ProtocolWorkspace},
		},
		{
			key:  "local",
			spec: "file:./vendor/pkg",
			want: manifest.Specifier{DisplayName: "local", TargetName: "local", Range: "./vendor/pkg", Protocol: manifest.ProtocolFile},
		},
		{
			key:  "linked",
			spec: "link:../other",
			want: manifest.Specifier{DisplayName: "linked", TargetName: "linked", Range: "../other", Protocol: manifest.ProtocolLink},
		},
		{
			key:  "portal",
			spec: "portal:../portal-pkg",
			want: manifest.Specifier{DisplayName: "portal", TargetName: "portal", Range: "../portal-pkg", Protocol: manifest.ProtocolPortal},
		},
		{
			key:  "@scope/pkg",
			spec: "^1.0.0",
			want: manifest.Specifier{DisplayName: "@scope/pkg", TargetName: "@scope/pkg", Range: "^1.0.0", Protocol: manifest.ProtocolRegistry},
		},
		{
			key:  "scoped",
			spec: "npm:@scope/pkg@^2.0.0",
			want: manifest.Specifier{DisplayName: "scoped", TargetName: "@scope/pkg", Range: "^2.0.0", Protocol: manifest.ProtocolNpm},
		},
	}
	for _, tc := range cases {
		t.Run(tc.key+"="+tc.spec, func(t *testing.T) {
			got, err := manifest.ParseSpecifier(tc.key, tc.spec)
			if err != nil {
				t.Fatal(err)
			}
			if got != tc.want {
				t.Fatalf("got %+v want %+v", got, tc.want)
			}
		})
	}
}

func TestFlattenOverrides(t *testing.T) {
	doc, err := manifest.Parse([]byte(`{
  "name": "x",
  "version": "1.0.0",
  "overrides": {
    "lodash": "4.17.21",
    "foo": {
      ".": "1.0.0",
      "bar": "2.0.0"
    }
  }
}`))
	if err != nil {
		t.Fatal(err)
	}
	m, err := manifest.ToNormalized(doc)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"lodash":  "4.17.21",
		"foo":     "1.0.0",
		"foo.bar": "2.0.0",
	}
	for k, v := range want {
		if m.Overrides[k] != v {
			t.Fatalf("override %q=%q got %q", k, v, m.Overrides[k])
		}
	}
}
