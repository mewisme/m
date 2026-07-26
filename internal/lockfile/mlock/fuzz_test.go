package mlock

import (
	"testing"
)

func FuzzDecode(f *testing.F) {
	f.Add([]byte(`{"lockfileVersion":1}`))
	f.Add([]byte(`not json`))
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = Decode(data)
	})
}
