package isolated_test

import (
	"strings"
	"testing"

	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/linker/isolated"
)

func TestStoreIDScoped(t *testing.T) {
	id := graph.PackageID{Name: "@scope/pkg", Version: "1.0.0"}
	got := isolated.StoreID(id)
	want := "@scope+pkg@1.0.0"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestStoreIDWithPeers(t *testing.T) {
	id := graph.PackageID{
		Name: "react-dom", Version: "18.0.0",
		PeerProviderContext: graph.PeerProviderContext{{Name: "react", Version: "18.0.0", Key: "react@18.0.0"}},
	}
	got := isolated.StoreID(id)
	if !strings.HasPrefix(got, "react-dom@18.0.0@") {
		t.Fatalf("got %q", got)
	}
	if len(got) > 120 {
		t.Fatalf("store id too long: %d", len(got))
	}
	for _, c := range got {
		if strings.ContainsRune(`<>:"/\|?*`, c) {
			t.Fatalf("forbidden char %q in %q", c, got)
		}
	}
}

func TestStoreIDFromKey(t *testing.T) {
	got := isolated.StoreIDFromKey("lodash@4.17.21")
	if got != "lodash@4.17.21" {
		t.Fatalf("got %q", got)
	}
}

func TestStoreIDLongScopedName(t *testing.T) {
	longName := "@" + strings.Repeat("a", 80) + "/pkg"
	id := graph.PackageID{Name: longName, Version: "1.0.0"}
	got := isolated.StoreID(id)
	if len(got) > 120 {
		t.Fatalf("len=%d", len(got))
	}
}

func TestStoreIDLongPeerContext(t *testing.T) {
	ppc := make(graph.PeerProviderContext, 0, 12)
	for i := 0; i < 12; i++ {
		ppc = append(ppc, graph.PeerProvider{
			Name:    "peer-package-with-long-name",
			Version: "99.0.0",
			Key:     "peer-package-with-long-name@99.0.0",
		})
	}
	id := graph.PackageID{Name: "consumer", Version: "1.0.0", PeerProviderContext: ppc}
	got := isolated.StoreID(id)
	if len(got) > 120 {
		t.Fatalf("len=%d", len(got))
	}
	if !strings.Contains(got, "@") {
		t.Fatalf("missing digest suffix: %q", got)
	}
}

func TestStoreIDDeterministicPeerDigest(t *testing.T) {
	ppc := graph.PeerProviderContext{
		{Name: "react", Version: "18.0.0", Key: "react@18.0.0"},
		{Name: "scheduler", Version: "0.23.0", Key: "scheduler@0.23.0"},
	}
	a := isolated.StoreID(graph.PackageID{Name: "pkg", Version: "1.0.0", PeerProviderContext: ppc})
	b := isolated.StoreID(graph.PackageID{Name: "pkg", Version: "1.0.0", PeerProviderContext: ppc})
	if a != b {
		t.Fatalf("digest not stable: %q vs %q", a, b)
	}
}

func TestStoreIDForbiddenCharsStripped(t *testing.T) {
	id := graph.PackageID{Name: `weird<>:"/\|?*pkg`, Version: "1.0.0"}
	got := isolated.StoreID(id)
	for _, c := range `<>:"/\|?*` {
		if strings.ContainsRune(got, c) {
			t.Fatalf("forbidden %q in %q", c, got)
		}
	}
}
