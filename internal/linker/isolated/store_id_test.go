package isolated_test

import (
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
		PeerContext: graph.PeerContext{{Name: "react", Range: "^18.0.0"}},
	}
	got := isolated.StoreID(id)
	if got != "react-dom@18.0.0(react@^18.0.0)" {
		t.Fatalf("got %q", got)
	}
}

func TestStoreIDFromKey(t *testing.T) {
	got := isolated.StoreIDFromKey("lodash@4.17.21")
	if got != "lodash@4.17.21" {
		t.Fatalf("got %q", got)
	}
}
