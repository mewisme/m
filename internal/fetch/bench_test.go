package fetch_test

import (
	"context"
	"testing"

	"github.com/mewisme/m/internal/fetch"
	"github.com/mewisme/m/internal/store"
	"github.com/mewisme/m/internal/testkit"
)

func BenchmarkDownloadVerify(b *testing.B) {
	reg := testkit.LoadRegistry(b, "registry/v1")
	srv := reg.Start(b)
	integrity := "sha256-758b80171fc185274170cb6db31a08042813d860a47b612d0671122a306b8b63"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		st := store.NewDir(b.TempDir())
		dl := &fetch.Downloader{Client: srv.Client(), Store: st, StagingDir: b.TempDir()}
		_, err := dl.Download(context.Background(), fetch.DownloadRequest{
			URL: srv.URL + "/lodash/-/lodash-4.17.21.tgz", Integrity: integrity,
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
