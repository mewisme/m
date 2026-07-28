package pnpm

import (
	"testing"

	"go.yaml.in/yaml/v3"
)

func TestScalarTypingRoundTrip(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
settings:
  autoInstallPeers: true
  excludeLinksFromLockfile: false
  quotedTrue: "true"
  quotedZero: "0"
  count: 3
  ratio: 1.5
  empty: null
`
	doc, err := Decode([]byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if doc.Settings["autoInstallPeers"] != true {
		t.Fatalf("bool: %#v", doc.Settings["autoInstallPeers"])
	}
	if doc.Settings["excludeLinksFromLockfile"] != false {
		t.Fatalf("bool false: %#v", doc.Settings["excludeLinksFromLockfile"])
	}
	if doc.Settings["quotedTrue"] != "true" {
		t.Fatalf("quoted: %#v", doc.Settings["quotedTrue"])
	}
	if doc.Settings["quotedZero"] != "0" {
		t.Fatalf("quoted zero: %#v", doc.Settings["quotedZero"])
	}
	if doc.Settings["count"] != int64(3) {
		t.Fatalf("int: %#v", doc.Settings["count"])
	}
	if doc.Settings["ratio"] != 1.5 {
		t.Fatalf("float: %#v", doc.Settings["ratio"])
	}
	if doc.Settings["empty"] != nil {
		t.Fatalf("null: %#v", doc.Settings["empty"])
	}
	out, err := Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	doc2, err := Decode(out)
	if err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"autoInstallPeers", "excludeLinksFromLockfile", "quotedTrue", "quotedZero", "count", "ratio", "empty"} {
		if doc.Settings[k] != doc2.Settings[k] {
			t.Fatalf("%s: %#v vs %#v", k, doc.Settings[k], doc2.Settings[k])
		}
	}
}

func TestDuplicateKeyRejected(t *testing.T) {
	const src = `
lockfileVersion: '9.0'
settings:
  autoInstallPeers: true
  autoInstallPeers: false
`
	_, err := Decode([]byte(src))
	if err == nil {
		t.Fatal("expected duplicate key error")
	}
}

func TestAliasRejected(t *testing.T) {
	var root yaml.Node
	const src = `
lockfileVersion: &v '9.0'
settings: *v
`
	if err := yaml.Unmarshal([]byte(src), &root); err != nil {
		t.Fatal(err)
	}
	_, err := Decode([]byte(src))
	if err == nil {
		t.Fatal("expected alias rejection")
	}
}
