package store

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	SetPublishReadOnly(false)
	os.Exit(m.Run())
}
