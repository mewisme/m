package store_test

import (
	"context"

	"github.com/mewisme/mew/internal/contentid"
	"github.com/mewisme/mew/internal/store"
)

func importIntegrity(ctx context.Context, ps *store.PackageStore, tgz, sri string) (store.PackageKey, error) {
	id, err := contentid.ParseSRI(sri)
	if err != nil {
		return store.PackageKey{}, err
	}
	result, err := ps.ImportFromTarball(ctx, tgz, id)
	if err != nil {
		return store.PackageKey{}, err
	}
	return result.Key, nil
}
