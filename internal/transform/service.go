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
	"strings"
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

	// activeCancels tracks per-request cancel functions keyed by cancel token.
	activeCancels   map[string]context.CancelFunc
	activeCancelsMu sync.Mutex

	// activeIDs tracks in-flight request IDs for duplicate detection.
	activeIDs   map[string]bool
	activeIDsMu sync.Mutex

	idleTimeout    time.Duration
	requestTimeout time.Duration

	// Session-scoped context and cancel, derived from the invocation context.
	// Cancel is called by Close to initiate coordinated shutdown.
	ctx    context.Context
	cancel context.CancelFunc

	// Tracked connections for coordinated shutdown.
	connsMu sync.Mutex
	conns   map[net.Conn]struct{}

	// WaitGroup tracks server, connection, and request goroutines.
	wg sync.WaitGroup
}

// ServiceOptions configures the transform session.
type ServiceOptions struct {
	Engine         Engine
	CacheDir       string // transform cache directory; empty disables cache
	Workers        int    // max concurrent transforms, default 4
	IdleTimeout    time.Duration
	RequestTimeout time.Duration
	// Context is the invocation context. Required for production sessions.
	// Session-scoped context is derived from this; cancellation propagates
	// to listener, connections, and active transforms.
	Context context.Context
}

// NewSession creates a transform service session bound to a random local port.
// Requires a non-nil Context in opts for production use.
func NewSession(opts ServiceOptions) (*Session, error) {
	if opts.Context == nil {
		return nil, fmt.Errorf("transform session requires a non-nil context")
	}
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

	ctx, cancel := context.WithCancel(opts.Context)

	s := &Session{
		Token:          token,
		engine:         opts.Engine,
		cacheDir:       opts.CacheDir,
		workers:        make(chan struct{}, opts.Workers),
		activeCancels:  make(map[string]context.CancelFunc),
		activeIDs:      make(map[string]bool),
		idleTimeout:    opts.IdleTimeout,
		requestTimeout: opts.RequestTimeout,
		ctx:            ctx,
		cancel:         cancel,
		conns:          make(map[net.Conn]struct{}),
	}

	// Listen on localhost random port using the session context.
	lc := net.ListenConfig{}
	ln, err := lc.Listen(ctx, "tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
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

// Start begins accepting connections. Returns after the authenticated health
// check succeeds: it connects to the listener and performs the real protocol
// hello handshake to verify the service is reachable and authenticated.
// The health check aborts promptly when the session context is cancelled.
func (s *Session) Start() error {
	if s.listener == nil {
		return fmt.Errorf("session not initialized")
	}

	s.wg.Add(1)
	go s.serve()

	// Health check: connect and perform real auth handshake using
	// DialContext so it aborts on session context cancellation.
	var d net.Dialer
	conn, err := d.DialContext(s.ctx, "tcp", s.Endpoint)
	if err != nil {
		_ = s.listener.Close()
		return fmt.Errorf("health check dial: %w", err)
	}
	defer func() { _ = conn.Close() }()

	// Abort pending I/O when session context is cancelled.
	go func() {
		<-s.ctx.Done()
		_ = conn.Close()
	}()

	// Set a deadline so encode/decode don't block indefinitely.
	_ = conn.SetDeadline(time.Now().Add(5 * time.Second))

	// Perform real hello handshake.
	if err := EncodeFrame(conn, HelloRequest{V: ProtocolVersion, Token: s.Token}); err != nil {
		_ = s.listener.Close()
		return fmt.Errorf("health check hello encode: %w", err)
	}
	var helloResp HelloResponse
	if err := DecodeFrame(conn, &helloResp); err != nil {
		_ = s.listener.Close()
		return fmt.Errorf("health check hello decode: %w", err)
	}
	if !helloResp.OK {
		_ = s.listener.Close()
		return fmt.Errorf("health check auth failed: %s", helloResp.Reason)
	}

	return nil
}

// serve accepts connections on the listener. Exits when the session context is
// cancelled, the listener is closed, shutdown has begun, or an idle timeout
// expires with no active requests.
func (s *Session) serve() {
	defer s.wg.Done()
	defer func() { _ = s.listener.Close() }()

	// Close listener promptly when session context is cancelled.
	// This unblocks Accept so serve can drain and return.
	go func() {
		<-s.ctx.Done()
		_ = s.listener.Close()
	}()

	for {
		if s.closed.Load() {
			return
		}
		if s.ctx.Err() != nil {
			return
		}

		if s.idleTimeout > 0 && s.active.Load() == 0 {
			// Set accept deadline so we can check idle expiry periodically.
			if tl, ok := s.listener.(*net.TCPListener); ok {
				_ = tl.SetDeadline(time.Now().Add(s.idleTimeout))
			}
		}

		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			if s.ctx.Err() != nil {
				return
			}
			// Idle timeout with no active requests: shut down.
			if s.active.Load() == 0 && isTimeoutErr(err) {
				return
			}
			// Retry genuine transient accept errors while session remains active.
			if isTemporaryAcceptErr(err) {
				continue
			}
			return
		}
		// Clear deadline so active connections aren't affected.
		if tl, ok := s.listener.(*net.TCPListener); ok {
			_ = tl.SetDeadline(time.Time{})
		}

		// Track connection for coordinated shutdown.
		s.connsMu.Lock()
		s.conns[conn] = struct{}{}
		s.connsMu.Unlock()
		s.wg.Add(1)
		go func() {
			defer s.wg.Done()
			defer func() {
				s.connsMu.Lock()
				delete(s.conns, conn)
				s.connsMu.Unlock()
			}()
			s.handleConn(s.ctx, conn)
		}()
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

// isTemporaryAcceptErr reports whether err is a transient accept error
// that should be retried.
func isTemporaryAcceptErr(err error) bool {
	if err == nil {
		return false
	}
	type temporary interface{ Temporary() bool }
	if t, ok := err.(temporary); ok {
		return t.Temporary()
	}
	return false
}

// handleConn handles a single TCP connection.
func (s *Session) handleConn(ctx context.Context, conn net.Conn) {
	defer func() { _ = conn.Close() }()

	// Step 1: authenticate via hello.
	var hello HelloRequest
	if err := DecodeFrame(conn, &hello); err != nil {
		_ = EncodeFrame(conn, HelloResponse{
			V: ProtocolVersion, OK: false,
			ErrCode: string(apperr.TransformProtocolVersion), Reason: "decode error",
		})
		return
	}
	if err := hello.Validate(); err != nil {
		_ = EncodeFrame(conn, HelloResponse{
			V: ProtocolVersion, OK: false,
			ErrCode: SanitizeErrorCode(string(apperr.TransformProtocolVersion)),
			Reason:  SanitizeErrorMessage(err.Error()),
		})
		return
	}
	if subtle.ConstantTimeCompare([]byte(hello.Token), []byte(s.Token)) != 1 {
		_ = EncodeFrame(conn, HelloResponse{
			V: ProtocolVersion, OK: false,
			ErrCode: string(apperr.TransformAuth), Reason: "unauthorized",
		})
		return
	}
	_ = EncodeFrame(conn, HelloResponse{V: ProtocolVersion, OK: true})

	// Step 2: process requests.
	for {
		if s.closed.Load() {
			return
		}
		if ctx.Err() != nil {
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

		// Dispatch with per-operation validation.
		switch req.Op {
		case OpHealth:
			if err := ValidateRequestHeader(req.V, req.ID, req.Op, OpHealth); err != nil {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
					Error:   SanitizeErrorMessage(err.Error()),
				})
				continue
			}
			_ = EncodeFrame(conn, TransformResponseV2{V: ProtocolVersion, ID: req.ID, OK: true})

		case OpTransform:
			if err := req.Validate(); err != nil {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
					Error:   SanitizeErrorMessage(err.Error()),
				})
				continue
			}
			s.handleTransform(ctx, conn, &req)

		case OpCancel:
			if err := ValidateRequestHeader(req.V, req.ID, req.Op, OpCancel); err != nil {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
					Error:   SanitizeErrorMessage(err.Error()),
				})
				continue
			}
			if req.CancelToken == "" {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: string(apperr.Usage),
					Error:   "missing cancel token",
				})
				continue
			}
			if len(req.CancelToken) > MaxCancelTokenLength {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: string(apperr.Usage),
					Error:   "cancel token too long",
				})
				continue
			}
			// Cancel the matching active request by its cancel token.
			// Unknown or already-completed tokens: OK (idempotent cancel).
			s.activeCancelsMu.Lock()
			if cancel, ok := s.activeCancels[req.CancelToken]; ok {
				cancel()
				delete(s.activeCancels, req.CancelToken)
			}
			s.activeCancelsMu.Unlock()
			_ = EncodeFrame(conn, TransformResponseV2{V: ProtocolVersion, ID: req.ID, OK: true})

		case OpShutdown:
			if err := ValidateRequestHeader(req.V, req.ID, req.Op, OpShutdown); err != nil {
				_ = EncodeFrame(conn, TransformResponseV2{
					V: ProtocolVersion, ID: req.ID, OK: false,
					ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
					Error:   SanitizeErrorMessage(err.Error()),
				})
				continue
			}
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
	// Reject new requests after shutdown has begun.
	if s.closed.Load() {
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.TransformUnavailable),
			Error:   "service shutting down",
		})
		return
	}

	// Reject duplicate active request IDs before any expensive work.
	s.activeIDsMu.Lock()
	if s.activeIDs[req.ID] {
		s.activeIDsMu.Unlock()
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.Usage),
			Error:   "duplicate request id",
		})
		return
	}
	s.activeIDs[req.ID] = true
	s.activeIDsMu.Unlock()
	defer func() {
		s.activeIDsMu.Lock()
		delete(s.activeIDs, req.ID)
		s.activeIDsMu.Unlock()
	}()

	// Apply request deadline derived from session context.
	reqCtx, cancel := context.WithTimeout(ctx, s.requestTimeout)
	defer cancel()

	// Register cancel for OpCancel tracking.
	if req.CancelToken != "" {
		s.activeCancelsMu.Lock()
		// Reject duplicate cancel tokens (already-active transform).
		if _, exists := s.activeCancels[req.CancelToken]; exists {
			s.activeCancelsMu.Unlock()
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.Usage),
				Error:   "duplicate cancel token",
			})
			return
		}
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

	// Verify source digest before any processing.
	if err := VerifySourceDigest(req.Source, req.SourceDigest); err != nil {
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
			Error:   SanitizeErrorMessage(err.Error()),
		})
		return
	}

	// Verify options digest and parse options.
	var opts NormalizedOptions
	if req.Options != "" {
		if err := VerifyOptionsDigest(req.Options, req.OptsDigest); err != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: SanitizeErrorCode(string(apperr.CodeOf(err))),
				Error:   SanitizeErrorMessage(err.Error()),
			})
			return
		}
		if err := json.Unmarshal([]byte(req.Options), &opts); err != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.TransformConfigOption),
				Error:   SanitizeErrorMessage(fmt.Sprintf("invalid options: %v", err)),
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
		OptsDigest:      req.OptsDigest,
		TargetNodeMajor: req.NodeMajor,
		SourceMapMode:   sourceMapMode,
	}

	// Check transform cache.
	var result *TransformResult
	var resultErr error
	if s.cacheDir != "" {
		identity := s.engine.Identity()
		key := CacheKey(tReq, identity)
		cached, cerr := TryReadCache(s.cacheDir, key)
		if cerr != nil {
			if !isCacheCorruption(cerr) {
				resultErr = cerr
			}
		} else if cached != nil {
			result = cached
		}
	}

	// Cache miss, corruption, or cache disabled: run engine.
	if result == nil && resultErr == nil {
		engineResult, engineErr := s.engine.Transform(reqCtx, tReq)
		if reqCtx.Err() != nil {
			resultErr = reqCtx.Err()
		} else if engineErr == nil {
			result = &engineResult
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
		if reqCtx.Err() != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.TransformTimeout), Error: "transform timeout",
			})
			return
		}
		// Check if this is a context cancellation (from session shutdown).
		if ctx.Err() != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.TransformCancelled), Error: "transform cancelled",
			})
			return
		}
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: SanitizeErrorCode(string(apperr.CodeOf(resultErr))),
			Error:   SanitizeErrorMessage(resultErr.Error()),
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

// Close initiates coordinated shutdown. It is idempotent and concurrency-safe.
//
// Shutdown order:
//  1. Cancel session context — propagates to all derived contexts.
//  2. Close listener — stops the accept loop.
//  3. Cancel active transforms — unblocks workers.
//  4. Close tracked connections — unblocks reads/writes.
//  5. Wait for all tracked goroutines to finish.
//
// Returns the listener close error, or nil. Connection close errors are
// not aggregated (they are expected side-effects of listener shutdown).
func (s *Session) Close() error {
	if !s.closed.CompareAndSwap(false, true) {
		return nil
	}

	// 1. Cancel session context.
	s.cancel()

	// 2. Close listener (may race with serve's deferred close or
	// ctx-cancellation goroutine — "use of closed network connection"
	// is benign in that case).
	var closeErr error
	if s.listener != nil {
		closeErr = s.listener.Close()
		if closeErr != nil && isClosedNetworkErr(closeErr) {
			closeErr = nil
		}
	}

	// 3. Cancel all active transforms.
	s.activeCancelsMu.Lock()
	for _, cancel := range s.activeCancels {
		cancel()
	}
	s.activeCancelsMu.Unlock()

	// 4. Close active connections to unblock reads/writes.
	s.connsMu.Lock()
	for c := range s.conns {
		_ = c.Close()
	}
	s.connsMu.Unlock()

	// 5. Wait for server, connection, and request goroutines.
	s.wg.Wait()

	// 6. Clean up remaining active cancel/ID entries.
	s.activeCancelsMu.Lock()
	for k := range s.activeCancels {
		delete(s.activeCancels, k)
	}
	s.activeCancelsMu.Unlock()
	s.activeIDsMu.Lock()
	for k := range s.activeIDs {
		delete(s.activeIDs, k)
	}
	s.activeIDsMu.Unlock()

	return closeErr
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

// isClosedNetworkErr reports whether err is "use of closed network connection",
// which is expected when Close and serve's deferred close race.
func isClosedNetworkErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "use of closed network connection")
}

// isCacheCorruption reports whether err is a recoverable cache corruption
// (entry was cleaned up, caller should regenerate). Permission and I/O
// errors are NOT corruption — they indicate a disk problem.
func isCacheCorruption(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "cache code digest mismatch") ||
		strings.Contains(msg, "cache map digest mismatch") ||
		strings.Contains(msg, "cache code missing") ||
		strings.Contains(msg, "cache map missing") ||
		strings.Contains(msg, "corrupt cache meta") ||
		strings.Contains(msg, "cache output digest mismatch") ||
		strings.Contains(msg, "invalid cache key shape")
}
