package registry_test

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/mewisme/m/internal/registry"
)

func TestPackumentRejectsHugeBody(t *testing.T) {
	body := strings.Repeat("x", 2048)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)

	client := registry.NewClient(registry.Options{
		BaseURL:           srv.URL,
		HTTPClient:        srv.Client(),
		MaxPackumentBytes: 1024,
		MaxRetries:        0,
	})
	_, err := client.Packument(context.Background(), srv.URL, "huge")
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "exceeds") {
		t.Fatalf("err=%v", err)
	}
}

func TestPackumentRetries429(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := hits.Add(1)
		if n == 1 {
			w.Header().Set("Retry-After", "1")
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		_ = json.NewEncoder(w).Encode(registry.Packument{
			Name:     "pkg",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Version: "1.0.0",
					Dist:    registry.Dist{Tarball: "pkg-1.0.0.tgz", Integrity: "sha512-abc"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 2,
	})
	p, err := client.Packument(context.Background(), srv.URL, "pkg")
	if err != nil {
		t.Fatal(err)
	}
	if p.Name != "pkg" {
		t.Fatalf("%+v", p)
	}
	if hits.Load() < 2 {
		t.Fatalf("hits=%d", hits.Load())
	}
}

func TestPackumentRetries500(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits.Add(1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		_ = json.NewEncoder(w).Encode(registry.Packument{
			Name:     "pkg",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Version: "1.0.0",
					Dist:    registry.Dist{Tarball: "pkg-1.0.0.tgz", Integrity: "sha512-abc"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 2,
	})
	if _, err := client.Packument(context.Background(), srv.URL, "pkg"); err != nil {
		t.Fatal(err)
	}
}

func TestPackumentCancelDuringRetry(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	t.Cleanup(srv.Close)

	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 3,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	_, err := client.Packument(ctx, srv.URL, "pkg")
	if err == nil {
		t.Fatal("expected cancel")
	}
	if err != context.Canceled && err != context.DeadlineExceeded {
		t.Fatalf("err=%v", err)
	}
}

func TestPackumentInFlightDedup(t *testing.T) {
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(50 * time.Millisecond)
		_ = json.NewEncoder(w).Encode(registry.Packument{
			Name:     "pkg",
			DistTags: map[string]string{"latest": "1.0.0"},
			Versions: map[string]registry.VersionMeta{
				"1.0.0": {
					Version: "1.0.0",
					Dist:    registry.Dist{Tarball: "pkg-1.0.0.tgz", Integrity: "sha512-abc"},
				},
			},
		})
	}))
	t.Cleanup(srv.Close)

	client := registry.NewClient(registry.Options{
		BaseURL:    srv.URL,
		HTTPClient: srv.Client(),
		MaxRetries: 0,
	})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := client.Packument(context.Background(), srv.URL, "pkg"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if hits.Load() != 1 {
		t.Fatalf("expected 1 fetch, got %d", hits.Load())
	}
}

func TestParseRetryAfter(t *testing.T) {
	t.Parallel()
	cases := []struct {
		header string
		fb     time.Duration
		want   time.Duration
	}{
		{"", time.Second, time.Second},
		{"5", 0, 5 * time.Second},
		{"-1", time.Second, time.Second},
		{"99999", 0, time.Hour},
	}
	for _, tc := range cases {
		got := registry.ParseRetryAfterForTest(tc.header, tc.fb)
		if got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.header, got, tc.want)
		}
	}
}
