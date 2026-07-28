package mlock_test

import (
	"testing"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/lockfile/mlock"
)

func TestMigrateRejectsV1PeerRangeLock(t *testing.T) {
	raw := []byte(`{
  "lockfileVersion": 1,
  "packages": [{
    "id": {
      "name": "react",
      "version": "18.2.0",
      "peerContext": [{"name": "react-dom", "range": "^18.0.0"}]
    }
  }]
}`)
	got, err := mlock.Decode(raw)
	if err == nil {
		t.Fatalf("expected migrate error, got doc=%#v", got)
	}
	if apperr.CodeOf(err) != apperr.Lockfile {
		t.Fatalf("code=%s err=%v", apperr.CodeOf(err), err)
	}
}
