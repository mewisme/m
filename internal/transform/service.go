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
	Token       string // random one-time auth token
	Endpoint    string // "host:port" for the listener
	endpointEnv string // MEW_TRANSFORM_ENDPOINT env value
	tokenEnv    string // MEW_TRANSFORM_TOKEN env value

	mu       sync.Mutex
	listener net.Listener
	engine   Engine
	workers  chan struct{}
	active   atomic.Int32
	closed   atomic.Bool

	idleTimeout    time.Duration
	requestTimeout time.Duration
}

// ServiceOptions configures the transform session.
type ServiceOptions struct {
	Engine         Engine
	Workers        int // max concurrent transforms, default 4
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
		workers:        make(chan struct{}, opts.Workers),
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
		s.listener.Close()
		return fmt.Errorf("health check dial: %w", err)
	}
	conn.Close()

	go s.serve(ctx)
	return nil
}

// serve accepts connections on the listener.
func (s *Session) serve(ctx context.Context) {
	defer s.listener.Close()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			if s.closed.Load() {
				return
			}
			// ponytail: transient accept error → log and continue.
			continue
		}
		go s.handleConn(ctx, conn)
	}
}

// handleConn handles a single TCP connection.
func (s *Session) handleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()

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
					ErrCode: "ERR_M_TRANSFORM_INVALID", Error: err.Error(),
				})
				continue
			}
			s.handleTransform(ctx, conn, &req)

		case OpCancel:
			// Cancel is best-effort; the context cancellation chain handles it.
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

	// Acquire worker slot.
	select {
	case s.workers <- struct{}{}:
		defer func() { <-s.workers }()
	case <-reqCtx.Done():
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.Timeout), Error: "service overloaded",
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
				ErrCode: string(apperr.Config), Error: fmt.Sprintf("invalid options: %v", err),
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

	result, err := s.engine.Transform(reqCtx, tReq)
	if err != nil {
		// Check for context cancellation/timeout.
		if reqCtx.Err() != nil {
			_ = EncodeFrame(conn, TransformResponseV2{
				V: ProtocolVersion, ID: req.ID, OK: false,
				ErrCode: string(apperr.Timeout), Error: "transform timeout",
			})
			return
		}
		_ = EncodeFrame(conn, TransformResponseV2{
			V: ProtocolVersion, ID: req.ID, OK: false,
			ErrCode: string(apperr.Integrity), Error: err.Error(),
		})
		return
	}

	cacheStr := "miss"
	if result.CacheStatus == CacheStatusHit {
		cacheStr = "hit"
	} else if result.CacheStatus == CacheStatusBypass {
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
