package manifest_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/manifest"
)

func FuzzParseJSON(f *testing.F) {
	f.Add([]byte(`{"schemaVersion":1,"name":"app","dependencies":[]}`))
	f.Add([]byte(`{`))
	f.Add([]byte(`null`))
	f.Add([]byte(`{"schemaVersion":1,"dependencies":[{"name":"a","range":"1","kind":"prod"},{"name":"a","range":"2","kind":"prod"}]}`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, err := manifest.ParseJSON(data)
		if err == nil {
			return
		}
		_ = apperr.CodeOf(err) // must not panic
	})
}

func TestKnownBadManifest(t *testing.T) {
	_, err := manifest.ParseJSON([]byte(`{not json`))
	if err == nil {
		t.Fatal("expected error")
	}
	if apperr.CodeOf(err) == apperr.OK {
		t.Fatal("unexpected OK")
	}
}
