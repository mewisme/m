package pnpm

import "testing"

func TestResolveDependencyTargetGolden(t *testing.T) {
	idx := NewPackageIndex([]string{
		"b@2.0.0",
		"@scope/pkg@1.0.0",
		"react-dom@18.2.0(react@18.2.0)",
	})
	cases := []struct {
		name, ref, want string
	}{
		{"b", "2.0.0", "b@2.0.0"},
		{"b", "b@2.0.0", "b@2.0.0"},
		{"@scope/pkg", "1.0.0", "@scope/pkg@1.0.0"},
		{"react-dom", "react-dom@18.2.0(react@18.2.0)", "react-dom@18.2.0(react@18.2.0)"},
		{"local", "link:../pkg", "link:../pkg"},
	}
	for _, tc := range cases {
		target, err := ResolveDependencyTarget(tc.name, tc.ref, idx)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.name, tc.ref, err)
		}
		if target.Key != tc.want {
			t.Fatalf("%s %s: got %q want %q", tc.name, tc.ref, target.Key, tc.want)
		}
	}
}

func TestResolveDependencyTargetDangling(t *testing.T) {
	idx := NewPackageIndex([]string{"a@1.0.0"})
	_, err := ResolveDependencyTarget("missing", "9.9.9", idx)
	if err == nil {
		t.Fatal("expected dangling error")
	}
}

func FuzzResolveDependencyTarget(f *testing.F) {
	f.Add("b", "2.0.0")
	f.Add("@scope/x", "1.0.0")
	f.Fuzz(func(t *testing.T, name, ref string) {
		if len(name)+len(ref) > maxPackageKeyLen {
			return
		}
		idx := NewPackageIndex([]string{name + "@" + ref, ref})
		_, _ = ResolveDependencyTarget(name, ref, idx)
	})
}
