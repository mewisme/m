package resolver_test

import (
	"context"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/policy"
	"github.com/mewisme/mew/internal/resolver"
)

func TestBuildPeerConflictTree(t *testing.T) {
	conf := resolver.PeerConflict{Package: "react", Peer: "react-dom", Range: "^18.0.0", Importer: "root"}
	tree := resolver.BuildPeerConflictTree(conf, nil)
	if tree.Peer != "react-dom" {
		t.Fatalf("peer=%q", tree.Peer)
	}
	if tree.Root.RequiringPackage != "react" {
		t.Fatalf("requiring=%q", tree.Root.RequiringPackage)
	}
	if tree.Root.Importer != "root" {
		t.Fatalf("importer=%q", tree.Root.Importer)
	}
	if !strings.Contains(tree.Root.Constraint, "react-dom") {
		t.Fatalf("constraint=%q", tree.Root.Constraint)
	}
}

func TestExplainPeerMissing(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": { "react": "^18.0.0" }
}`)
	tree, err := eng.ExplainPeer(context.Background(), root, "react-dom", resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if tree == nil {
		t.Fatal("expected conflict tree")
	}
	if tree.Peer != "react-dom" || tree.Root.RequiringPackage != "react" {
		t.Fatalf("%#v", tree)
	}
}

func TestPeerConflictFromError(t *testing.T) {
	eng, _ := engineWithPackuments(t, reactPackuments())
	root := writeProject(t, `{
  "name": "app",
  "version": "1.0.0",
  "dependencies": { "react": "^18.0.0" }
}`)
	_, err := eng.Resolve(context.Background(), root, resolver.ResolveOptions{
		Policy: &policy.Policy{StrictPeerDependencies: true},
	})
	if err == nil {
		t.Fatal("expected error")
	}
	conf, ok := resolver.PeerConflictFromError(err)
	if !ok || conf.Peer != "react-dom" {
		t.Fatalf("conf=%#v ok=%v err=%v", conf, ok, err)
	}
}
