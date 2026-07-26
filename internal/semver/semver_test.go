package semver_test

import (
	"testing"

	"github.com/mewisme/m/internal/semver"
)

func TestSatisfies(t *testing.T) {
	cases := []struct {
		ver, spec string
		want      bool
	}{
		{"1.2.3", "^1.0.0", true},
		{"2.0.0", "^1.0.0", false},
		{"1.2.3", "~1.2.0", true},
		{"1.3.0", "~1.2.0", false},
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "*", true},
		{"1.2.3", "x", true},
		{"1.2.3", "1.0.0 - 1.3.0", true},
		{"1.4.0", "1.0.0 - 1.3.0", false},
		{"1.2.3", "^1.0.0 || ^2.0.0", true},
		{"2.1.0", "^1.0.0 || ^2.0.0", true},
		{"3.0.0", "^1.0.0 || ^2.0.0", false},
		{"1.2.3+build.1", "^1.2.0", true},
		{"1.2.4-beta.1", "^1.2.0", false}, // prerelease excluded under ^
		{"1.2.4-beta.1", "^1.2.4-beta.0", true},
		{"v1.2.3", "^1.0.0", true},
	}
	for _, tc := range cases {
		got, err := semver.Satisfies(tc.ver, tc.spec)
		if err != nil {
			t.Fatalf("%s %s: %v", tc.ver, tc.spec, err)
		}
		if got != tc.want {
			t.Fatalf("%s satisfies %s: got %v want %v", tc.ver, tc.spec, got, tc.want)
		}
	}
}

func TestSatisfiesInvalid(t *testing.T) {
	if _, err := semver.Satisfies("not-a-version", "^1.0.0"); err == nil {
		t.Fatal("expected error")
	}
	if _, err := semver.Satisfies("1.0.0", ">>>"); err == nil {
		t.Fatal("expected error")
	}
}

func TestMaxSatisfying(t *testing.T) {
	vers := []string{"1.0.0", "1.2.0", "1.2.3", "1.3.0-beta.1", "2.0.0"}
	got, err := semver.MaxSatisfying(vers, "^1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.3" {
		t.Fatalf("got %q want 1.2.3", got)
	}
	got, err = semver.MaxSatisfying(vers, "~1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if got != "1.2.3" {
		t.Fatalf("got %q", got)
	}
	if _, err := semver.MaxSatisfying(vers, "^3.0.0"); err == nil {
		t.Fatal("expected unsatisfiable")
	}
}

func BenchmarkMaxSatisfying(b *testing.B) {
	vers := []string{}
	for maj := 0; maj < 5; maj++ {
		for min := 0; min < 20; min++ {
			for pat := 0; pat < 5; pat++ {
				vers = append(vers, formatVer(maj, min, pat))
			}
		}
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = semver.MaxSatisfying(vers, "^2.5.0")
	}
}

func formatVer(maj, min, pat int) string {
	return itoa(maj) + "." + itoa(min) + "." + itoa(pat)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
