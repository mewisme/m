package integration_test

import "github.com/mewisme/mew/internal/store"

func init() {
	store.SetPublishReadOnly(false)
}
