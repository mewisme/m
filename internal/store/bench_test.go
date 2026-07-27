package store_test

import (
	"context"
	"path/filepath"
	"sync"
	"testing"

	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

func BenchmarkStoreImport(b *testing.B) {
	root := filepath.Join(b.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(b, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, err := ps.ImportFromTarball(ctx, tgz, integrity); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreVerifyWarm(b *testing.B) {
	root := filepath.Join(b.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(b, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ps.VerifyPackage(ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkStoreImportContention(b *testing.B) {
	root := filepath.Join(b.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(b, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 4; j++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if _, err := ps.ImportFromTarball(ctx, tgz, integrity); err != nil {
					b.Error(err)
				}
			}()
		}
		wg.Wait()
	}
}

func BenchmarkStoreFullTreeVerify(b *testing.B) {
	root := filepath.Join(b.TempDir(), "store")
	ps := store.NewPackageStore(root)
	tgz := filepath.Join(testkit.FixtureDir(b, "registry/v1"), "tarballs", "lodash-4.17.21.tgz")
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	ctx := context.Background()
	key, err := ps.ImportFromTarball(ctx, tgz, integrity)
	if err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := ps.VerifyPackage(ctx, key); err != nil {
			b.Fatal(err)
		}
	}
}
