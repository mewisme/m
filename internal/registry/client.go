package registry

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/mewisme/m/internal/apperr"
)

// ponytail: packument cap is a fixed constant; upgrade = Options.MaxPackumentBytes only.

const defaultMaxPackumentBytes = 64 << 20 // 64 MiB

// Options configures a registry Client.
type Options struct {
	BaseURL           string
	CacheDir          string
	Offline           bool
	PreferOffline     bool
	AuthToken         string
	HTTPClient        *http.Client
	MaxWorkers        int
	MaxRetries        int
	MaxPackumentBytes int64
}

// Client fetches and caches npm packuments.
type Client struct {
	opts     Options
	cache    *DiskCache
	http     *http.Client
	sem      chan struct{}
	flightMu sync.Mutex
	flight   map[string]*flightCall
}

type flightCall struct {
	done chan struct{}
	val  *Packument
	err  error
}

// NewClient constructs a Client. BaseURL is the default registry when not overridden per call.
func NewClient(opts Options) *Client {
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = 8
	}
	if opts.MaxRetries <= 0 {
		opts.MaxRetries = 3
	}
	hc := opts.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}
	var cache *DiskCache
	if opts.CacheDir != "" {
		cache = &DiskCache{Root: opts.CacheDir}
	}
	return &Client{
		opts:   opts,
		cache:  cache,
		http:   hc,
		sem:    make(chan struct{}, opts.MaxWorkers),
		flight: map[string]*flightCall{},
	}
}

// Metadata implements Registry.
func (c *Client) Metadata(ctx context.Context, name, version string) (*PackageMetadata, error) {
	p, err := c.Packument(ctx, c.opts.BaseURL, name)
	if err != nil {
		return nil, err
	}
	meta, err := p.SelectVersion(version)
	if err != nil {
		return nil, err
	}
	tarball := absoluteTarballURL(c.opts.BaseURL, name, meta.Dist.Tarball)
	return &PackageMetadata{
		Name:       name,
		Version:    meta.Version,
		Integrity:  meta.Dist.Integrity,
		TarballURL: tarball,
	}, nil
}

func absoluteTarballURL(base, name, tarball string) string {
	if tarball == "" {
		return ""
	}
	if strings.Contains(tarball, "://") {
		return tarball
	}
	base = strings.TrimRight(base, "/")
	tarball = strings.TrimPrefix(tarball, "/")
	return base + "/" + EncodeNamePath(name) + "/-/" + tarball
}

// Packument fetches (or loads) a packument from registryBase for name.
func (c *Client) Packument(ctx context.Context, registryBase, name string) (*Packument, error) {
	if registryBase == "" {
		registryBase = c.opts.BaseURL
	}
	origin := OriginKey(registryBase)
	key := origin + "|" + name

	c.flightMu.Lock()
	if call, ok := c.flight[key]; ok {
		c.flightMu.Unlock()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-call.done:
			return call.val, call.err
		}
	}
	call := &flightCall{done: make(chan struct{})}
	c.flight[key] = call
	c.flightMu.Unlock()

	call.val, call.err = c.packumentLocked(ctx, registryBase, origin, name)
	close(call.done)

	c.flightMu.Lock()
	delete(c.flight, key)
	c.flightMu.Unlock()
	return call.val, call.err
}

// Packuments fetches many packuments with a bounded worker pool.
func (c *Client) Packuments(ctx context.Context, registryBase string, names []string) (map[string]*Packument, error) {
	out := make(map[string]*Packument, len(names))
	if len(names) == 0 {
		return out, nil
	}
	workers := c.opts.MaxWorkers
	if workers > len(names) {
		workers = len(names)
	}
	work := make(chan string)
	var mu sync.Mutex
	var firstErr error
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for name := range work {
				if ctx.Err() != nil {
					return
				}
				p, err := c.Packument(ctx, registryBase, name)
				mu.Lock()
				if err != nil {
					if firstErr == nil {
						firstErr = err
					}
				} else {
					out[name] = p
				}
				mu.Unlock()
			}
		}()
	}
	for _, name := range names {
		select {
		case <-ctx.Done():
			close(work)
			wg.Wait()
			return out, ctx.Err()
		case work <- name:
		}
	}
	close(work)
	wg.Wait()
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	if firstErr != nil {
		return out, firstErr
	}
	return out, nil
}

func (c *Client) maxPackumentBytes() int64 {
	if c.opts.MaxPackumentBytes > 0 {
		return c.opts.MaxPackumentBytes
	}
	return defaultMaxPackumentBytes
}

func (c *Client) packumentLocked(ctx context.Context, registryBase, origin, name string) (*Packument, error) {
	if c.opts.Offline {
		if body, _, ok := c.cacheLookup(origin, name); ok {
			return ParsePackument(body)
		}
		return nil, apperr.New(apperr.Network, "registry.offline", name,
			"packument not in cache (offline mode)")
	}
	if c.opts.PreferOffline {
		if body, _, ok := c.cacheLookup(origin, name); ok {
			return ParsePackument(body)
		}
	}

	var etag string
	if _, tag, ok := c.cacheLookup(origin, name); ok {
		etag = tag
	}

	select {
	case c.sem <- struct{}{}:
		defer func() { <-c.sem }()
	case <-ctx.Done():
		return nil, ctx.Err()
	}

	body, newETag, fromCache, err := c.fetchPackument(ctx, registryBase, name, etag)
	if err != nil {
		return nil, err
	}
	if fromCache {
		cached, _, ok := c.cacheLookup(origin, name)
		if !ok {
			return nil, apperr.New(apperr.Network, "registry.cache", name, "304 without cache body")
		}
		body = cached
	} else if c.cache != nil {
		_ = c.cache.Store(origin, name, newETag, body)
	}
	return ParsePackument(body)
}

func (c *Client) cacheLookup(origin, name string) ([]byte, string, bool) {
	if c.cache == nil {
		return nil, "", false
	}
	return c.cache.Lookup(origin, name)
}

func (c *Client) fetchPackument(ctx context.Context, registryBase, name, etag string) (body []byte, newETag string, notModified bool, err error) {
	url := strings.TrimRight(registryBase, "/") + "/" + EncodeNamePath(name)
	var lastErr error
	for attempt := 0; attempt <= c.opts.MaxRetries; attempt++ {
		if attempt > 0 {
			delay := time.Duration(1<<uint(attempt-1)) * 100 * time.Millisecond
			if delay > 2*time.Second {
				delay = 2 * time.Second
			}
			if err := waitRetry(ctx, "", delay); err != nil {
				return nil, "", false, err
			}
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, "", false, apperr.Wrap(apperr.Network, "registry.fetch", name, err)
		}
		req.Header.Set("Accept", "application/json")
		if etag != "" {
			req.Header.Set("If-None-Match", etag)
		}
		if c.opts.AuthToken != "" {
			req.Header.Set("Authorization", "Bearer "+c.opts.AuthToken)
		}
		res, err := c.http.Do(req)
		if err != nil {
			lastErr = apperr.Wrap(apperr.Network, "registry.fetch", name, err)
			continue
		}
		func() {
			defer func() { _ = res.Body.Close() }()
			switch res.StatusCode {
			case http.StatusOK:
				limit := c.maxPackumentBytes() + 1
				b, rerr := io.ReadAll(io.LimitReader(res.Body, limit))
				if rerr != nil {
					lastErr = apperr.Wrap(apperr.Network, "registry.fetch", name, rerr)
					return
				}
				if int64(len(b)) > c.maxPackumentBytes() {
					lastErr = apperr.New(apperr.Network, "registry.fetch", name,
						fmt.Sprintf("packument exceeds %d bytes", c.maxPackumentBytes()))
					err = lastErr
					return
				}
				body = b
				newETag = res.Header.Get("ETag")
				err = nil
				lastErr = nil
			case http.StatusNotModified:
				notModified = true
				newETag = etag
				if v := res.Header.Get("ETag"); v != "" {
					newETag = v
				}
				lastErr = nil
				err = nil
			case http.StatusNotFound:
				lastErr = apperr.New(apperr.NotFound, "registry.fetch", name, "package not found")
				err = lastErr
			case http.StatusUnauthorized, http.StatusForbidden:
				lastErr = apperr.New(apperr.Network, "registry.fetch", name,
					fmt.Sprintf("registry auth failed (%d)", res.StatusCode))
				err = lastErr
			case http.StatusTooManyRequests, http.StatusRequestTimeout:
				lastErr = apperr.New(apperr.Network, "registry.fetch", name,
					fmt.Sprintf("transient status %d", res.StatusCode))
				if werr := waitRetry(ctx, res.Header.Get("Retry-After"), 0); werr != nil {
					err = werr
					return
				}
				err = lastErr
			default:
				if res.StatusCode >= 500 {
					lastErr = apperr.New(apperr.Network, "registry.fetch", name,
						fmt.Sprintf("registry status %d", res.StatusCode))
					err = lastErr
					return
				}
				lastErr = apperr.New(apperr.Network, "registry.fetch", name,
					fmt.Sprintf("registry status %d", res.StatusCode))
				err = lastErr
			}
		}()
		if notModified || (body != nil && lastErr == nil) {
			return body, newETag, notModified, nil
		}
		if err != nil && !retryable(err) {
			return nil, "", false, err
		}
	}
	if lastErr != nil {
		return nil, "", false, lastErr
	}
	return nil, "", false, apperr.New(apperr.Network, "registry.fetch", name, "exhausted retries")
}

func retryable(err error) bool {
	if err == nil {
		return false
	}
	c := apperr.CodeOf(err)
	if c == apperr.NotFound {
		return false
	}
	msg := err.Error()
	if strings.Contains(msg, "auth failed") {
		return false
	}
	if strings.Contains(msg, "status 4") && !strings.Contains(msg, "408") && !strings.Contains(msg, "429") {
		return false
	}
	return true
}

func waitRetry(ctx context.Context, retryAfter string, fallback time.Duration) error {
	delay := parseRetryAfter(retryAfter, fallback)
	if delay <= 0 {
		return nil
	}
	t := time.NewTimer(delay)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func parseRetryAfter(header string, fallback time.Duration) time.Duration {
	header = strings.TrimSpace(header)
	if header == "" {
		return fallback
	}
	if sec, err := strconv.Atoi(header); err == nil {
		if sec <= 0 {
			return fallback
		}
		if sec > 3600 {
			sec = 3600
		}
		return time.Duration(sec) * time.Second
	}
	if when, err := http.ParseTime(header); err == nil {
		d := time.Until(when)
		if d < 0 {
			return 0
		}
		if d > time.Hour {
			return time.Hour
		}
		return d
	}
	return fallback
}

// ParseRetryAfterForTest exposes parseRetryAfter for unit tests.
func ParseRetryAfterForTest(header string, fallback time.Duration) time.Duration {
	return parseRetryAfter(header, fallback)
}

// Cache returns the disk cache (may be nil).
func (c *Client) Cache() *DiskCache { return c.cache }

// BaseURL returns the configured default registry.
func (c *Client) BaseURL() string { return c.opts.BaseURL }
