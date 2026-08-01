package transform

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// Session encapsulates a per-invocation transform service.
type Session struct {
	Token       string // random per-session bearer auth token
	Endpoint    string // "host:port" for the listener
	endpointEnv string // MEW_TRANSFORM_ENDPOINT env value
	tokenEnv    string // MEW_TRANSFORM_TOKEN env value

	listener net.Listener
	engine   Engine
	cacheDir string
	workers  chan struct{}
	active   atomic.Int32
	closed   atomic.Bool

	// activeCancels tracks per-request cancel functions keyed by request ID.
	activeCancels    map[string]context.CancelFunc
	activeCancelsMu  sync.Mutex

	idleTimeout    time.Duration
	requestTimeout time.Duration
}

// ServiceOptions configures the transform session.
type ServiceOptions struct {
	Engine         Engine
	CacheDir       string // transform cache directory; empty disables cache
	Workers        int    // max concurrent transforms, default 4
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
}

// NewSession creates a transform service session bound to a random local port.
func NewSession(opts ServiceOptions) (*Session, error) {
	if opts.Engine == nil {
		opts.Engine = NewEsbuildEngine()
	}
	if opts.Workers <= 0 {
		opts.Workers = 4
	}
	if opts.IdleTimeout <= 0 {
		opts.IdleTimeout = 30 * time.Second
	}
	if opts.RequestTimeout <= 0 {
		opts.RequestTimeout = 60 * time.Second
	}

	token, err := generateToken(32)
	if err != nil {
		return nil, fmt.Errorf("token generation: %w", err)
	}

	s := &Session{
		Token:          token,
		engine:         opts.Engine,
		cacheDir:       opts.CacheDir,
		workers:        make(chan struct{}, opts.Workers),
		activeCancels:  make(map[string]context.CancelFunc),
		idleTimeout:    opts.IdleTimeout,
		requestTimeout: opts.RequestTimeout,
	}

	// Listen on localhost random port.
	lc := net.ListenConfig{}
	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen: %w", err)
	}
	s.listener = ln
	addr := ln.Addr().(*net.TCPAddr)
	s.Endpoint = fmt.Sprintf("127.0.0.1:%d", addr.Port)
	s.endpointEnv = s.Endpoint
	s.tokenEnv = token

	return s, nil
}

// EndpointEnv returns the environment variable for the listener endpoint.
func (s *Session) EndpointEnv() string { return s.endpointEnv }

// TokenEnv returns the environment variable for the auth token.
func (s *Session) TokenEnv() string { return s.tokenEnv }

// EnvOverlay returns key=value environment pairs for the Node child.
func (s *Session) EnvOverlay() []string {
	return []string{
		"MEW_TRANSFORM_ENDPOINT=" + s.endpointEnv,
		"MEW_TRANSFORM_TOKEN=" + s.tokenEnv,
	}
}

// Start begins accepting connections. Returns after the health check succeeds.
func (s *Session) Start(ctx context.Context) error {
	if s.listener == nil {
		return fmt.Errorf("session not initialized")
	}

	// Health check: connect to ourselves.
	conn, err := net.DialTimeout("tcp", s.Endpoint, 5*time.Second)
	if err != nil {
		_ = s.listener.Close()
		return fmt.Errorf("health check dial: %w", err)
	}
	_ = conn.Close()

	go s.serve(ctx)
	return nil
}

// serve accepts connections on the listener. Exits when ctx is done,
// the listener is closed, or idle timeout expires with no active requests.
func (s *Session) serve(ctx context.Context) {
	defer func() { _ = s.listener.Close() }()

	for {
		if s.idleTimeout > 0 && s.active.Load() == 0 {
			// Set accept deadline so we can check idle expiry periodically.
			_ = s.listener.(*net.TCPListener).SetDeadline(time.Now().Add(s.idleTimeout))
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			// Idle timeout with no active requests: shut down.
			if s.active.Load() == 0 && isTimeoutErr(err) {
				return
			}
			// ponytail: transient accept error → continue.
			continue
		}
		// Clear deadline so active connections aren't affected.
		_ = s.listener.(*net.TCPListener).SetDeadline(time.Time{})
		go s.handleConn(ctx, conn)
	}
}

// isTimeoutErr reports whether err is a network timeout.
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	type timeout interface{ Timeout() bool }
	if t, ok := err.(timeout); ok {
		return t.Timeout()
	}
	return false
}

// handleConn handles a single TCP connection.
func (s *Session) handleConn(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Step 1: authenticate via hello.
	var hello HelloRequest
	if err := DecodeFrame(conn, &hello); err != nil {
		_ = EncodeFrame(conn, HelloResponse{V: ProtocolVersion, OK: false, Reason: "decode error"})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hello.Token), []byte(s.Token)) != 1 {
		_ = EncodeFrame(conn, HelloResponse{V: ProtocolVersion, OK: false, Reason: "unauthorized"})
		return
	}
	_ = EncodeFrame(conn, HelloResponse{V: ProtocolVersion, OK: true})

	// Step 2: process requests.
	for {
		if s.closed.Load() {
			return
		}

		// Set read deadline for idle detection.
		_ = conn.SetReadDeadline(time.Now().Add(s.idleTimeout))

		var req TransformRequestV2
		if err := DecodeFrame(conn, &req); err != nil {
			if err == io.EOF {
				return // clean disconnect
			}
			return // protocol error → close connection
		}

		// Dispatch.
		switch req.Op {
		case OpHealth:
			_ = EncodeFrame(conn, TransformResponseV2{V: ProtocolVersion, ID: req.ID, OK: true})

		case OpTransform:
			if err := req.Validate(); err != nil {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: string(apperr.Usage), Error: err.Error(),
				})
				continue
			}
			s.handleTransform(ctx, conn, &req)

		case OpCancel:
			// Cancel the matching active request by its cancel token.
			s.activeCancelsMu.Lock()
			if cancel, ok := s.activeCancels[req.CancelToken]; ok {
				cancel()
				delete(s.activeCancels, req.CancelToken)
			}
			s.activeCancelsMu.Unlock()
			_ = EncodeFrame(conn, TransformResponseV2{V: ProtocolVersion, ID: req.ID, OK: true})

		case OpShutdown:
			_ = EncodeFrame(conn, TransformResponseV2{V: ProtocolVersion, ID: req.ID, OK: true})
			return

		default:
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.Unsupported),
				Error:   fmt.Sprintf("unknown op %q", req.Op),
			})
		}
	}
}

// handleTransform processes a single transform request.
func (s *Session) handleTransform(ctx context.Context, conn net.Conn, req *TransformRequestV2) {
	// Apply request deadline.
	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Register cancel for OpCancel tracking.
	if req.CancelToken != "" {
		s.activeCancelsMu.Lock()
		s.activeCancels[req.CancelToken] = cancel
		s.activeCancelsMu.Unlock()
		defer func() {
			s.activeCancelsMu.Lock()
			delete(s.activeCancels, req.CancelToken)
			s.activeCancelsMu.Unlock()
		}()
	}

	// Acquire worker slot.
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	case <-reqCtx.Done():
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.TransformTimeout), Error: "service overloaded",
		})
		return
	}

	s.active.Add(1)
	defer s.active.Add(-1)

	// Parse options.
	var opts NormalizedOptions
	if req.Options != "" {
		if err := json.Unmarshal([]byte(req.Options), &opts); err != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.TransformConfigOption), Error: fmt.Sprintf("invalid options: %v", err),
			})
			return
		}
	}

	sourceMapMode := SourceMapNone
	switch req.SourceMap {
	case "inline":
		sourceMapMode = SourceMapInline
	case "external":
		sourceMapMode = SourceMapExternal
	}

	tReq := TransformRequest{
		RequestID:       req.ID,
		SourcePath:      req.Path,
		SourceBytes:     []byte(req.Source),
		SourceDigest:    req.SourceDigest,
		Loader:          LoaderKind(req.Loader),
		Format:          mapFormatString(req.Format),
		NormalizedOpts:  opts,
		TsconfigDigest:  req.OptsDigest,
		TargetNodeMajor: req.NodeMajor,
		SourceMapMode:   sourceMapMode,
	}

	// Check transform cache.
	var result *TransformResult
	var resultErr error
	if s.cacheDir != "" {
		identity := s.engine.Identity()
		key := CacheKey(tReq, identity)
		if cached, cerr := TryReadCache(s.cacheDir, key); cerr == nil && cached != nil {
			result = cached
		}
	}

	// Cache miss or cache disabled: run engine.
	if result == nil {
		engineResult, engineErr := s.engine.Transform(reqCtx, tReq)
		// Discard late results: if the context expired during transform,
		// the engine result must not be cached or returned to the client.
		if reqCtx.Err() != nil {
			resultErr = reqCtx.Err()
		} else if engineErr == nil {
			result = &engineResult
			// Cache the result on success.
			if s.cacheDir != "" {
				identity := s.engine.Identity()
				key := CacheKey(tReq, identity)
				if werr := WriteCache(s.cacheDir, key, &engineResult); werr != nil {
					_ = werr
				}
			}
		} else {
			resultErr = engineErr
		}
	}

	if resultErr != nil {
		// Check for context cancellation/timeout.
		if reqCtx.Err() != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.TransformTimeout), Error: "transform timeout",
			})
			return
		}
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.Integrity), Error: resultErr.Error(),
		})
		return
	}

	cacheStr := "miss"
	switch result.CacheStatus {
	case CacheStatusHit:
		cacheStr = "hit"
	case CacheStatusBypass:
		cacheStr = "bypass"
	}

	_ = EncodeFrame(conn, TransformResponseV2{
		V:      ProtocolVersion,
		ID:     req.ID,
		OK:     true,
		Code:   string(result.Code),
		Map:    string(result.SourceMap),
		Digest: result.OutputDigest,
		Cache:  cacheStr,
	})
}

// Close initiates graceful shutdown.
func (s *Session) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// ActiveRequests returns the current in-flight request count.
func (s *Session) ActiveRequests() int32 {
	return s.active.Load()
}

// mapFormatString converts a protocol format string to ModuleFormat.
func mapFormatString(f string) ModuleFormat {
	switch f {
	case "cjs":
		return FormatCJS
	default:
		return FormatESM
	}
}

// generateToken returns a random hex token with byteLen random bytes.
func generateToken(byteLen int) (string, error) {
	b := make([]byte, byteLen)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
