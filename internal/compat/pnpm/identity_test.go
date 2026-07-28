package pnpm

import (
	"testing"
)

func TestParsePackageIdentityScopedPeer(t *testing.T) {
	cases := []struct {
		key     string
		name    string
		version string
		peer    string
	}{
		{"lodash@4.17.21", "lodash", "4.17.21", ""},
		{"@types/node@20.1.0", "@types/node", "20.1.0", ""},
		{"react-dom@18.2.0(react@18.2.0)", "react-dom", "18.2.0", "(react@18.2.0)"},
		{"@scope/pkg@1.0.0(peer@2.0.0)", "@scope/pkg", "1.0.0", "(peer@2.0.0)"},
	}
	for _, tc := range cases {
		id, err := ParsePackageIdentity(tc.key)
		if err != nil {
			t.Fatalf("%q: %v", tc.key, err)
		}
		if id.Name != tc.name || id.BaseVersion != tc.version || id.PeerSuffix != tc.peer {
			t.Fatalf("%q: got %+v", tc.key, id)
		}
	}
}

func TestKeyToNameVersionMatchesGolden(t *testing.T) {
	name, ver := KeyToNameVersion("@scope/pkg@1.0.0(peer@2.0.0)")
	if name != "@scope/pkg" || ver != "1.0.0(peer@2.0.0)" {
		t.Fatalf("got %q %q", name, ver)
	}
}

func FuzzParsePackageIdentity(f *testing.F) {
	seeds := []string{
		"lodash@4.17.21",
		"@types/node@20.1.0",
		"react-dom@18.2.0(react@18.2.0)",
		"link:../local-pkg",
		"workspace:foo@1.0.0",
	}
	for _, s := range seeds {
		f.Add(s)
	}
	f.Fuzz(func(t *testing.T, key string) {
		if len(key) > maxPackageKeyLen {
			return
		}
		id, err := ParsePackageIdentity(key)
		if err != nil {
			return
		}
		if id.CanonicalKey != key {
			t.Fatalf("canonical key mismatch")
		}
	})
}
