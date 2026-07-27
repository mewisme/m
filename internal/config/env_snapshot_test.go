package config_test

import (
	"testing"

	"github.com/mewisme/m/internal/config"
)

func TestEnvSnapshotLookupUnixCaseSensitive(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"HOME=/home/a", "Home=/home/b"}, "linux")
	if v, ok := snap.Lookup("HOME"); !ok || v != "/home/a" {
		t.Fatalf("HOME=%q ok=%v", v, ok)
	}
	if v, ok := snap.Lookup("Home"); !ok || v != "/home/b" {
		t.Fatalf("Home=%q ok=%v", v, ok)
	}
}

func TestEnvSnapshotLookupWindowsCaseInsensitive(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{
		"AppData=C:\\Users\\me\\AppData\\Roaming",
		"appdata=C:\\should-not-win",
		"APPDATA=C:\\Users\\me\\AppData\\Roaming-final",
	}, "windows")
	if v, ok := snap.Lookup("appdata"); !ok || v != "C:\\Users\\me\\AppData\\Roaming-final" {
		t.Fatalf("appdata=%q ok=%v", v, ok)
	}
	if v, ok := snap.Lookup("APPDATA"); !ok || v != "C:\\Users\\me\\AppData\\Roaming-final" {
		t.Fatalf("APPDATA=%q ok=%v", v, ok)
	}
}

func TestEnvSnapshotValueWithEquals(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"TOKEN=a=b=c"}, "linux")
	v, ok := snap.Lookup("TOKEN")
	if !ok || v != "a=b=c" {
		t.Fatalf("TOKEN=%q ok=%v", v, ok)
	}
}

func TestEnvSnapshotMalformedNoEquals(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"BARE"}, "linux")
	v, ok := snap.Lookup("BARE")
	if !ok || v != "" {
		t.Fatalf("BARE=%q ok=%v", v, ok)
	}
}

func TestEnvSnapshotCloneIndependent(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"MEW_HOME=/a"}, "linux")
	clone := snap.Clone()
	if v, _ := clone.Lookup("MEW_HOME"); v != "/a" {
		t.Fatalf("clone=%q", v)
	}
}

func TestEnvSnapshotEnvironRoundTrip(t *testing.T) {
	in := []string{"A=1", "B=two=three"}
	snap := config.NewEnvSnapshot(in, "linux")
	out := snap.Environ()
	round := config.NewEnvSnapshot(out, "linux")
	for _, key := range []string{"A", "B"} {
		want, _ := snap.Lookup(key)
		got, ok := round.Lookup(key)
		if !ok || got != want {
			t.Fatalf("%s: got %q want %q", key, got, want)
		}
	}
}

func TestEnvSnapshotMEWMixedCaseWindows(t *testing.T) {
	snap := config.NewEnvSnapshot([]string{"mew_cache_dir=C:\\cache"}, "windows")
	v, ok := snap.Lookup("MEW_CACHE_DIR")
	if !ok || v != "C:\\cache" {
		t.Fatalf("MEW_CACHE_DIR=%q ok=%v", v, ok)
	}
}
