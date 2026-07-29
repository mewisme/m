package registry_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mewisme/mew/internal/registry"
)

func TestPublishPUTSuccess(t *testing.T) {
	t.Setenv("NO_PROXY", "*")
	var auth, otp string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			http.Error(w, "method", http.StatusMethodNotAllowed)
			return
		}
		auth = r.Header.Get("Authorization")
		otp = r.Header.Get("npm-otp")
		body, _ := io.ReadAll(r.Body)
		if len(body) < 50 {
			http.Error(w, "short", http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	token := "npm_secret_publish_token"
	_, err := registry.Publish(context.Background(), registry.PublishOptions{
		RegistryBase: srv.URL,
		Name:         "pkg-a",
		Version:      "1.0.0",
		Tag:          "beta",
		Access:       "public",
		OTP:          "654321",
		AuthToken:    token,
		Tarball:      []byte("fake-tarball-bytes"),
		PackageJSON:  []byte(`{"name":"pkg-a","version":"1.0.0"}`),
		HTTPClient:   srv.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if auth != "Bearer "+token {
		t.Fatalf("auth %q", auth)
	}
	if otp != "654321" {
		t.Fatalf("otp %q", otp)
	}
}

func TestPublishRedactsTokenInError(t *testing.T) {
	t.Setenv("NO_PROXY", "*")
	token := "npm_super_secret_token_value"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "auth failed for Bearer "+token, http.StatusUnauthorized)
	}))
	defer srv.Close()

	_, err := registry.Publish(context.Background(), registry.PublishOptions{
		RegistryBase: srv.URL,
		Name:         "pkg-a",
		Version:      "1.0.0",
		AuthToken:    token,
		Tarball:      []byte("fake-tarball-bytes"),
		PackageJSON:  []byte(`{"name":"pkg-a","version":"1.0.0"}`),
		HTTPClient:   srv.Client(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	msg := err.Error()
	if strings.Contains(msg, token) {
		t.Fatalf("token leaked in error: %q", msg)
	}
}
