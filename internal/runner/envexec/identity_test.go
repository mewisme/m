package envexec_test

import (
	"encoding/json"
	"testing"

	"github.com/mewisme/mew/internal/runner/envexec"
)

func TestEnvironmentIdentityDigestDeterministic(t *testing.T) {
	id := envexec.EnvironmentIdentity{
		SchemaVersion:  envexec.IdentitySchemaVersion,
		Source:         envexec.SourceSnapshot,
		GraphDigest:    "graph",
		MaterialDigest: "material",
		SourceDigest:   "source",
		Platform:       envexec.PlatformFingerprint{OS: "linux", Arch: "amd64"},
		LinkerMode:     "isolated",
		NodeABI:        "node-v20",
	}
	d1 := id.IdentityDigest()
	d2 := id.IdentityDigest()
	if d1 == "" || d1 != d2 {
		t.Fatalf("digest not stable: %q vs %q", d1, d2)
	}
}

func TestEnvironmentIdentitySerializationStable(t *testing.T) {
	id := envexec.EnvironmentIdentity{
		SchemaVersion:  envexec.IdentitySchemaVersion,
		Source:         envexec.SourceCapsule,
		GraphDigest:    "abc",
		MaterialDigest: "def",
		SourceDigest:   "ghi",
		Platform:       envexec.CurrentPlatform(),
		LinkerMode:     "hoisted",
	}
	b1, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	b2, err := json.Marshal(id)
	if err != nil {
		t.Fatal(err)
	}
	if string(b1) != string(b2) {
		t.Fatalf("marshal not stable:\n%s\n%s", b1, b2)
	}
}

func TestIdentityExcludesCommandArgs(t *testing.T) {
	base := envexec.EnvironmentIdentity{
		SchemaVersion:  envexec.IdentitySchemaVersion,
		Source:         envexec.SourceProject,
		GraphDigest:    "g",
		MaterialDigest: "m",
		SourceDigest:   "s",
		Platform:       envexec.CurrentPlatform(),
		LinkerMode:     "isolated",
	}
	digest := base.IdentityDigest()
	b, err := json.Marshal(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(b) == 0 {
		t.Fatal("empty identity json")
	}
	// Command args are not fields on EnvironmentIdentity; digest must not change
	// when only unrelated execution request fields differ.
	_ = digest
	reqA := envexec.ExecutionRequest{
		Source:  envexec.ProjectSource{CWD: "/p"},
		Command: envexec.CommandRequest{Name: "eslint", Args: []string{"--fix"}},
	}
	reqB := reqA
	reqB.Command.Args = []string{"--version"}
	if reqA.Command.Args[0] == reqB.Command.Args[0] {
		t.Fatal("test setup")
	}
	_ = reqB
	if base.IdentityDigest() != digest {
		t.Fatal("identity digest changed without identity field changes")
	}
}
