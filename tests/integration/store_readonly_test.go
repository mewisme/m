package integration_test

import "github.com/mewisme/m/internal/store"

func init() {
	store.SetPublishReadOnly(false)
}
