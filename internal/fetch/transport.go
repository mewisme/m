// Package fetch builds shared HTTP clients for registry and (later) tarball downloads.
package fetch

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// Options configures an HTTP client.
type Options struct {
	Timeout  time.Duration
	ProxyURL string // empty → ProxyFromEnvironment
	CAFile   string // optional PEM bundle
}

// NewClient builds an *http.Client with optional proxy and custom CA.
func NewClient(opts Options) (*http.Client, error) {
	if opts.Timeout <= 0 {
		opts.Timeout = 60 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if opts.ProxyURL != "" {
		u, err := url.Parse(opts.ProxyURL)
		if err != nil {
			return nil, apperr.Wrap(apperr.Config, "fetch.proxy", opts.ProxyURL, err)
		}
		switch u.Scheme {
		case "http", "https":
			transport.Proxy = http.ProxyURL(u)
		case "socks5", "socks5h":
			return nil, apperr.New(apperr.Config, "fetch.proxy", opts.ProxyURL,
				"SOCKS proxies are not supported yet")
		default:
			return nil, apperr.New(apperr.Config, "fetch.proxy", opts.ProxyURL,
				fmt.Sprintf("unsupported proxy scheme %q", u.Scheme))
		}
	} else {
		transport.Proxy = http.ProxyFromEnvironment
	}
	if opts.CAFile != "" {
		pem, err := os.ReadFile(opts.CAFile)
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "fetch.ca", opts.CAFile, err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(pem) {
			return nil, apperr.New(apperr.Config, "fetch.ca", opts.CAFile, "no certificates found in PEM")
		}
		if transport.TLSClientConfig == nil {
			transport.TLSClientConfig = &tls.Config{MinVersion: tls.VersionTLS12}
		} else {
			transport.TLSClientConfig = transport.TLSClientConfig.Clone()
		}
		transport.TLSClientConfig.RootCAs = pool
	}
	return &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}, nil
}
