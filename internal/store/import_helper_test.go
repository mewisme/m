package store_test

import (
	"context"

	"github.com/mewisme/m/internal/contentid"
	"github.com/mewisme/m/internal/store"
)

func importIntegrity(ctx context.Context, ps *store.PackageStore, tgz, sri string) (store.PackageKey, error) {
	id, err := contentid.ParseSRI(sri)
	if err != nil {
		return store.PackageKey{}, err
	}
	return ps.ImportFromTarball(ctx, tgz, id)
}
