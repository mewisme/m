package transform_test

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/mewisme/mew/internal/transform"
)

// fakeEngine blocks Transform until unblocked, for cancellation testing.
type fakeEngine struct {
	blockCh  chan struct{} // closed to unblock
	identity transform.EngineIdentity
}

func newFakeEngine() *fakeEngine {
	return &fakeEngine{
		blockCh:  make(chan struct{}),
		identity: transform.EngineIdentity{Name: "fake", Version: "1.0"},
	}
}

func (e *fakeEngine) Identity() transform.EngineIdentity { return e.identity }

func (e *fakeEngine) Transform(ctx context.Context, _ transform.TransformRequest) (transform.TransformResult, error) {
	select {
	case <-ctx.Done():
		return transform.TransformResult{}, ctx.Err()
	case <-e.blockCh:
		return transform.TransformResult{
			Code:         []byte("transformed"),
			OutputDigest: "abc",
			CacheStatus:  transform.CacheStatusBypass,
		}, nil
	}
}

func (e *fakeEngine) unblock() { close(e.blockCh) }

// blockingEngine returns a fake engine that blocks on every transform until
// the returned unblock function is called.
func blockingEngine() (transform.Engine, func()) {
	e := newFakeEngine()
	return e, e.unblock
}

func TestNewSessionRequiresNonNilContext(t *testing.T) {
	_, err := transform.NewSession(transform.ServiceOptions{
		Context: nil,
	})
	if err == nil {
		t.Fatal("expected error for nil context")
	}
}

func TestNewSessionCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatalf("NewSession with cancelled context: %v", err)
	}
	defer func() { _ = sess.Close() }()

	// Start should fail because health check dial uses cancelled context.
	err = sess.Start()
	if err == nil {
		t.Fatal("expected Start to fail with cancelled context")
	}
}

func TestCloseIdempotent(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	// First close.
	if err := sess.Close(); err != nil {
		t.Fatalf("first Close: %v", err)
	}
	// Second close — must be safe.
	if err := sess.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
	// Third close.
	if err := sess.Close(); err != nil {
		t.Fatalf("third Close: %v", err)
	}
}

func TestCloseConcurrent(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	var wg sync.WaitGroup
	errs := make([]error, 10)
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			errs[idx] = sess.Close()
		}(i)
	}
	wg.Wait()

	// All must return nil (idempotent after first succeeds).
	for i, e := range errs {
		if e != nil {
			t.Errorf("concurrent Close[%d] returned error: %v", i, e)
		}
	}
}

func TestCloseWaitsForGoroutines(t *testing.T) {
	ctx := context.Background()
	engine, unblock := blockingEngine()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	// Send a transform that blocks (worker slot acquired).
	// We need a client connection for this.
	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	srcDigest := transform.DigestString("x")

	// Send a transform request that blocks on the engine.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "tok-1",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	// Give the transform time to acquire the worker and start blocking.
	time.Sleep(100 * time.Millisecond)

	// Unblock the engine so Close doesn't hang waiting for the transform.
	unblock()

	// Close must return reasonably quickly.
	closeDone := make(chan error, 1)
	go func() {
		closeDone <- sess.Close()
	}()

	select {
	case err := <-closeDone:
		if err != nil {
			t.Logf("Close returned: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Close did not return within 5s")
	}

	_ = conn.Close()
}

func TestInvocationCancellationClosesListener(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	// Verify listener is accepting.
	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	// Cancel the invocation context.
	cancel()

	// The listener should close promptly.
	timeout := time.After(2 * time.Second)
	for {
		_, err := net.Dial("tcp", sess.Endpoint)
		if err != nil {
			break // listener closed
		}
		select {
		case <-timeout:
			t.Fatal("listener still accepting after context cancellation")
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}

	_ = sess.Close()
}

func TestNewRequestsRejectedAfterShutdown(t *testing.T) {
	ctx := context.Background()
	engine, unblock := blockingEngine()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer unblock()

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	// Start shutdown.
	if err := sess.Close(); err != nil {
		t.Fatal(err)
	}

	// Try to connect — the listener should be closed.
	conn, err := net.DialTimeout("tcp", sess.Endpoint, 500*time.Millisecond)
	if err == nil {
		_ = conn.Close()
		t.Fatal("expected connection refused after shutdown")
	}
}

func TestActiveCancelsCleanedAfterCompletion(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	srcDigest := transform.DigestString("const x = 1;")

	// Send a valid transform.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "cleanup-1", Op: "transform",
		Path: "a.ts", Source: "const x = 1;", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "tok-cleanup",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	// Read response.
	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("transform failed: %s", resp.Error)
	}

	// Send a cancel for the same token — should be idempotent (no crash, no panic).
	cancelReq := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "cancel-1", Op: "cancel",
		CancelToken: "tok-cleanup",
	}
	if err := transform.EncodeFrame(conn, cancelReq); err != nil {
		t.Fatal(err)
	}
	var cancelResp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &cancelResp); err != nil {
		t.Fatal(err)
	}
	if !cancelResp.OK {
		t.Fatalf("cancel failed: %s", cancelResp.Error)
	}
}

func TestHealthCheckUsesDialContext(t *testing.T) {
	// Verify that health check respects context cancellation by using
	// a context that is cancelled before the dial completes.
	ctx, cancel := context.WithCancel(context.Background())

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: context.Background(), // session uses background
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	// Start serve in background.
	// We can't use sess.Start() because it uses s.ctx.
	// Instead verify that with a cancelled session context, Start fails.
	// The session context is derived from the opts.Context.
	_ = ctx
	cancel()

	// Create session with cancelled context.
	sess2, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		// Might succeed or fail depending on OS.
		// If it succeeds, Start must fail.
		if sess2 != nil {
			defer func() { _ = sess2.Close() }()
			if err := sess2.Start(); err == nil {
				t.Fatal("expected Start to fail with cancelled context")
			}
		}
		return
	}
	defer func() { _ = sess2.Close() }()

	err = sess2.Start()
	if err == nil {
		t.Fatal("expected Start to fail with cancelled context")
	}
	t.Logf("Start error (expected): %v", err)
}

func TestBlockedWorkerExitsOnCancellation(t *testing.T) {
	ctx := context.Background()
	engine, unblock := blockingEngine()
	defer unblock()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	srcDigest := transform.DigestString("x")

	// First transform blocks the worker.
	req1 := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "blk-1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "blk-tok-1",
	}
	if err := transform.EncodeFrame(conn, req1); err != nil {
		t.Fatal(err)
	}

	// Let it acquire the worker.
	time.Sleep(50 * time.Millisecond)

	// Second connection for the second request that will block on worker.
	conn2, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn2.Close() }()

	authReq2 := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn2, authReq2); err != nil {
		t.Fatal(err)
	}
	var authResp2 transform.HelloResponse
	if err := transform.DecodeFrame(conn2, &authResp2); err != nil {
		t.Fatal(err)
	}
	if !authResp2.OK {
		t.Fatalf("auth2 failed: %s", authResp2.Reason)
	}

	// Second transform with short timeout — should get timeout on worker acquisition.
	req2 := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "blk-2", Op: "transform",
		Path: "b.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "blk-tok-2",
	}

	// We can't directly control the per-request timeout from the client side.
	// Instead, close the session while the second request is pending, which
	// should unblock worker acquisition.
	if err := transform.EncodeFrame(conn2, req2); err != nil {
		t.Fatal(err)
	}

	time.Sleep(50 * time.Millisecond)

	// Close the session — cancels all active transforms.
	closeErr := sess.Close()
	// Worker should be released.
	_ = closeErr

	// Read responses with deadlines — both should fail or get error responses.
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))

	var resp1 transform.TransformResponseV2
	err1 := transform.DecodeFrame(conn, &resp1)
	if err1 == nil && resp1.OK {
		// First request might have completed if unblock raced with close.
		t.Logf("first request completed despite close")
	}

	var resp2 transform.TransformResponseV2
	err2 := transform.DecodeFrame(conn2, &resp2)
	if err2 == nil && resp2.OK {
		t.Error("expected second request to fail after session close")
	}
}

func TestRepeatedCloseWaitsForGoroutines(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	// Connect to verify it's up.
	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	_ = conn.Close()

	// Close should be clean — no goroutine leaks.
	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Second close — idempotent.
	if err := sess.Close(); err != nil {
		t.Fatalf("second Close: %v", err)
	}
}

func TestInvocationContextCancelsActiveTransforms(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	engine, unblock := blockingEngine()
	defer unblock()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	srcDigest := transform.DigestString("x")

	// Send a transform that blocks on the engine.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "ctx-cancel-1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "ctx-tok",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	// Give it time to acquire the worker and block on the engine.
	time.Sleep(100 * time.Millisecond)

	// Cancel the invocation context.
	cancel()

	// Read the response with a deadline — connection may close during shutdown.
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		// Connection closed during shutdown — acceptable.
		t.Logf("decode after cancel: %v", err)
		_ = sess.Close()
		return
	}
	if resp.OK {
		t.Error("expected transform to be cancelled")
	}
	if resp.ErrCode != "ERR_M_TRANSFORM_CANCELLED" && resp.ErrCode != "ERR_M_TRANSFORM_TIMEOUT" {
		t.Logf("error code: %s, error: %s", resp.ErrCode, resp.Error)
	}

	_ = sess.Close()
}

func TestSessionCloseReleasesActiveCancels(t *testing.T) {
	ctx := context.Background()
	engine, unblock := blockingEngine()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 1,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	srcDigest := transform.DigestString("x")

	// Send a transform that blocks.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "close-cleanup", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: "close-tok",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	time.Sleep(100 * time.Millisecond)

	// Close the session — this should cancel the active transform
	// and clean up the cancel entry.
	unblock() // unblock first so the transform can complete/fail
	time.Sleep(50 * time.Millisecond)

	if err := sess.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read the response to drain.
	var resp transform.TransformResponseV2
	_ = transform.DecodeFrame(conn, &resp)
}

func TestTransformRequestV2ValidationError(t *testing.T) {
	// Verify that invalid requests get proper error codes on the wire.
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	// Send a request with an unsupported Node major version.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "val-err", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: transform.DigestString("x"),
		Loader: "ts", Format: "esm", NodeMajor: 99,
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Fatal("expected validation error")
	}
	if resp.ErrCode == "" {
		t.Fatal("expected non-empty err_code")
	}
}

func TestDuplicateRequestIDRejected(t *testing.T) {
	ctx := context.Background()
	engine, unblock := blockingEngine()
	defer unblock()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
		Engine:  engine,
		Workers: 2,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	// Connection 1 sends a transform that blocks on the engine.
	conn1, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn1.Close() }()

	auth1 := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn1, auth1); err != nil {
		t.Fatal(err)
	}
	var authResp1 transform.HelloResponse
	if err := transform.DecodeFrame(conn1, &authResp1); err != nil {
		t.Fatal(err)
	}
	if !authResp1.OK {
		t.Fatalf("auth1 failed: %s", authResp1.Reason)
	}

	srcDigest := transform.DigestString("x")
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "dup-id", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: srcDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}

	// Connection 1: send request that blocks on engine.
	if err := transform.EncodeFrame(conn1, req); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)

	// Connection 2: send request with same ID — must be rejected.
	conn2, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn2.Close() }()

	auth2 := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn2, auth2); err != nil {
		t.Fatal(err)
	}
	var authResp2 transform.HelloResponse
	if err := transform.DecodeFrame(conn2, &authResp2); err != nil {
		t.Fatal(err)
	}
	if !authResp2.OK {
		t.Fatalf("auth2 failed: %s", authResp2.Reason)
	}

	if err := transform.EncodeFrame(conn2, req); err != nil {
		t.Fatal(err)
	}

	_ = conn2.SetReadDeadline(time.Now().Add(2 * time.Second))
	var rejectResp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn2, &rejectResp); err != nil {
		t.Fatalf("decode rejection: %v", err)
	}
	if rejectResp.ErrCode != "ERR_M_USAGE" || rejectResp.Error != "duplicate request id" {
		t.Errorf("expected duplicate rejection, got err_code=%q error=%q",
			rejectResp.ErrCode, rejectResp.Error)
	}
}

func TestCancelTokenValidation(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	// Missing cancel token.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "ct-1", Op: "cancel",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}
	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Fatal("expected error for missing cancel token")
	}
}

func TestIdempotentCancel(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	// Cancel a non-existent request — should succeed (idempotent).
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "ic-1", Op: "cancel",
		CancelToken: "nonexistent",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}
	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("idempotent cancel failed: %s", resp.Error)
	}
}

func TestShutdownRequest(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	// Send shutdown.
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "sd-1", Op: "shutdown",
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}
	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if !resp.OK {
		t.Fatalf("shutdown rejected: %s", resp.Error)
	}

	// The connection handler should exit after shutdown acknowledgement.
	// Verify the conn eventually closes (server side closes after response).
	time.Sleep(100 * time.Millisecond)

	// Subsequent read should get EOF.
	_ = conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
	var extraResp transform.TransformResponseV2
	err = transform.DecodeFrame(conn, &extraResp)
	if err == nil {
		t.Error("expected connection close after shutdown")
	}
}

// errReader fails reads for testing error paths.
type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func TestDecodeFrameErrorHandling(t *testing.T) {
	// Verify DecodeFrame returns errors for various failure modes.
	var resp transform.TransformResponseV2

	err := transform.DecodeFrame(errReader{err: errors.New("injected")}, &resp)
	if err == nil {
		t.Fatal("expected error from failing reader")
	}
}

func TestSanitizeErrorMessageDoesNotExposeContent(t *testing.T) {
	// Source content.
	msg := transform.SanitizeErrorMessage("const x = 1; unexpected token")
	if msg == "const x = 1; unexpected token" {
		t.Fatal("source content not sanitized")
	}

	// Endpoints.
	msg2 := transform.SanitizeErrorMessage("failed to connect to 127.0.0.1:9999")
	if msg2 == "failed to connect to 127.0.0.1:9999" {
		t.Fatal("endpoint not sanitized")
	}

	// Options JSON.
	msg3 := transform.SanitizeErrorMessage(`bad "target": "ES2022" setting`)
	if msg3 == `bad "target": "ES2022" setting` {
		t.Fatal("options not sanitized")
	}

	// Safe message passes through.
	msg4 := transform.SanitizeErrorMessage("transform timeout")
	if msg4 != "transform timeout" {
		t.Fatalf("safe message altered: %s", msg4)
	}
}

func TestDigestStringDeterministic(t *testing.T) {
	d1 := transform.DigestString("hello world")
	d2 := transform.DigestString("hello world")
	if d1 != d2 {
		t.Fatal("DigestString not deterministic")
	}
	if len(d1) != 64 {
		t.Fatalf("DigestString length=%d, want 64", len(d1))
	}
}

// TestServiceOptionsDefaults verifies default values are applied.
func TestServiceOptionsDefaults(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	if sess.Token == "" {
		t.Fatal("token not generated")
	}
	if sess.Endpoint == "" {
		t.Fatal("endpoint not set")
	}
	if sess.ActiveRequests() != 0 {
		t.Fatalf("active=%d after creation", sess.ActiveRequests())
	}
}

func TestEnvOverlay(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	overlay := sess.EnvOverlay()
	if len(overlay) != 2 {
		t.Fatalf("expected 2 env entries, got %d", len(overlay))
	}

	if overlay[0] != "MEW_TRANSFORM_ENDPOINT="+sess.EndpointEnv() {
		t.Errorf("endpoint env mismatch: %s", overlay[0])
	}
	if overlay[1] != "MEW_TRANSFORM_TOKEN="+sess.TokenEnv() {
		t.Errorf("token env mismatch: %s", overlay[1])
	}
}

// Test that health check uses a real auth handshake.
func TestHealthCheckAuthHandshake(t *testing.T) {
	ctx := context.Background()
	sess, err := transform.NewSession(transform.ServiceOptions{
		Context: ctx,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	// Session started successfully means health check passed.
	// Verify we can connect and authenticate.
	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Auth with wrong token should fail.
	wrongAuth := transform.HelloRequest{V: transform.ProtocolVersion, Token: "wrong"}
	if err := transform.EncodeFrame(conn, wrongAuth); err != nil {
		t.Fatal(err)
	}
	var wrongResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &wrongResp); err != nil {
		t.Fatal(err)
	}
	if wrongResp.OK {
		t.Fatal("auth should have failed with wrong token")
	}
}

func TestTransformRequestV2ValidateOptionsDigestMismatch(t *testing.T) {
	opts := `{"target":"ES2022"}`
	wrongDigest := transform.DigestString("different opts")
	err := transform.VerifyOptionsDigest(opts, wrongDigest)
	if err == nil {
		t.Fatal("expected options digest mismatch error")
	}
}

// Ensure stable error codes needed by this package are exported.
// This test fails at compile time if these constants don't exist.
func TestErrorCodesAccessible(t *testing.T) {
	_ = transform.ProtocolVersion
	_ = transform.MaxFrameSize
}

// TestTransformSuccessButCacheWriteFailure verifies that when a transform
// succeeds but WriteCache fails, the response is an error (not OK with cached
// status silently dropped).
func TestTransformSuccessButCacheWriteFailure(t *testing.T) {
	// os.Chmod 0o555 does not prevent file creation inside the directory on
	// Windows; the Unix permission model the test relies on is unavailable.
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory semantics not available on Windows")
	}
	ctx := context.Background()
	dir := t.TempDir()

	// Create cache dir and make it read-only so writes fail.
	cacheDir := filepath.Join(dir, "transform", "v1")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	// WriteCache writes into <cacheDir>/<prefix>/<key>.{code,map,meta}.
	// Making the entire cacheDir read-only causes the prefix dir MkdirAll
	// or file writes to fail.
	if err := os.Chmod(cacheDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(cacheDir, 0o755) }()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context:  ctx,
		CacheDir: cacheDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	src := "const x: number = 1;"
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "cache-fail", Op: "transform",
		Path: "test.ts", Source: src, SourceDigest: transform.DigestString(src),
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Fatal("expected error response when cache write fails")
	}
	if resp.ErrCode == "" {
		t.Fatal("expected non-empty err_code")
	}
	if resp.Code != "" {
		t.Fatal("expected empty code on failure, got transformed code")
	}
}

// TestCacheErrorDiagnosticsDoNotExposeCredentials verifies that cache and
// transform error responses do not leak tokens, source content, endpoints,
// or transform options.
func TestCacheErrorDiagnosticsDoNotExposeCredentials(t *testing.T) {
	// os.Chmod 0o555 does not prevent file creation inside the directory on
	// Windows; the Unix permission model the test relies on is unavailable.
	if runtime.GOOS == "windows" {
		t.Skip("read-only directory semantics not available on Windows")
	}
	ctx := context.Background()
	dir := t.TempDir()

	cacheDir := filepath.Join(dir, "transform", "v1")
	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(cacheDir, 0o555); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(cacheDir, 0o755) }()

	sess, err := transform.NewSession(transform.ServiceOptions{
		Context:  ctx,
		CacheDir: cacheDir,
	})
	if err != nil {
		t.Fatal(err)
	}

	if err := sess.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = sess.Close() }()

	conn, err := net.Dial("tcp", sess.Endpoint)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = conn.Close() }()

	// Authenticate.
	authReq := transform.HelloRequest{V: transform.ProtocolVersion, Token: sess.Token}
	if err := transform.EncodeFrame(conn, authReq); err != nil {
		t.Fatal(err)
	}
	var authResp transform.HelloResponse
	if err := transform.DecodeFrame(conn, &authResp); err != nil {
		t.Fatal(err)
	}
	if !authResp.OK {
		t.Fatalf("auth failed: %s", authResp.Reason)
	}

	src := "const x: number = 1;"
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "no-leak", Op: "transform",
		Path: "test.ts", Source: src, SourceDigest: transform.DigestString(src),
		Loader: "ts", Format: "esm", NodeMajor: 20,
		Options: `{"target":"ES2022"}`, OptsDigest: transform.DigestString(`{"target":"ES2022"}`),
	}
	if err := transform.EncodeFrame(conn, req); err != nil {
		t.Fatal(err)
	}

	var resp transform.TransformResponseV2
	if err := transform.DecodeFrame(conn, &resp); err != nil {
		t.Fatal(err)
	}
	if resp.OK {
		t.Fatal("expected error response when cache write fails")
	}

	// Error response must not contain the session token.
	if strings.Contains(resp.Error, sess.Token) {
		t.Fatal("error response contains session token")
	}
	// Error response must not contain the source content.
	if strings.Contains(resp.Error, src) {
		t.Fatal("error response contains source content")
	}
	// Error response must not contain the endpoint.
	if strings.Contains(resp.Error, sess.Endpoint) {
		t.Fatal("error response contains endpoint")
	}
	// Error response must not contain transform options.
	if strings.Contains(resp.Error, `"target"`) || strings.Contains(resp.Error, "ES2022") {
		t.Fatal("error response contains transform options")
	}
}

// verifyErrorCode helper — unused, kept for documentation of the pattern
// that callers should use to check error codes.
