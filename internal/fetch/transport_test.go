package fetch_test

import (
	"net/http"
	"testing"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/fetch"
)

func TestNewClientDefault(t *testing.T) {
	c, err := fetch.NewClient(fetch.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if c.Timeout != time.Second {
		t.Fatalf("%v", c.Timeout)
	}
	_ = http.StatusOK
}

func TestSocksRejected(t *testing.T) {
	_, err := fetch.NewClient(fetch.Options{ProxyURL: "socks5://127.0.0.1:1080"})
	if err == nil {
		t.Fatal("expected socks reject")
	}
	if apperr.CodeOf(err) != apperr.Config {
		t.Fatalf("%v", err)
	}
}
