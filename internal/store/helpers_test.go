package store

import (
	"context"

	"github.com/mewisme/m/internal/contentid"
)

func importIntegrity(ctx context.Context, ps *PackageStore, tgz, sri string) (PackageKey, error) {
	id, err := contentid.ParseSRI(sri)
	if err != nil {
		return PackageKey{}, err
	}
	return ps.ImportFromTarball(ctx, tgz, id)
}
