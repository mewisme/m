package transform_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/mew/internal/transform"
)

func TestFrameRoundTrip(t *testing.T) {
	req := transform.TransformRequestV2{
		V:            transform.ProtocolVersion,
		ID:           "req-1",
		Op:           "transform",
		Path:         "src/a.ts",
		Source:       "const x: number = 1",
		SourceDigest: transform.DigestString("const x: number = 1"),
		Loader:       "ts",
		Format:       "esm",
		NodeMajor:    20,
		SourceMap:    "none",
	}
	var buf bytes.Buffer
	if err := transform.EncodeFrame(&buf, req); err != nil {
		t.Fatal(err)
	}
	var got transform.TransformRequestV2
	if err := transform.DecodeFrame(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.ID != req.ID || got.Path != req.Path || got.Source != req.Source {
		t.Fatalf("got %+v want %+v", got, req)
	}

	resp := transform.TransformResponseV2{
		V:    transform.ProtocolVersion,
		ID:   "req-1",
		OK:   true,
		Code: "const x = 1;\n",
	}
	buf.Reset()
	if err := transform.EncodeFrame(&buf, resp); err != nil {
		t.Fatal(err)
	}
	var gotResp transform.TransformResponseV2
	if err := transform.DecodeFrame(&buf, &gotResp); err != nil {
		t.Fatal(err)
	}
	if gotResp.V != resp.V || gotResp.ID != resp.ID || !gotResp.OK || gotResp.Code != resp.Code {
		t.Fatalf("got %+v want %+v", gotResp, resp)
	}
}

func TestEncodeFrameLargePayload(t *testing.T) {
	// 16 MiB limit is enforced on decode, not encode.
	// Just verify encode doesn't panic on reasonable sizes.
	req := transform.TransformRequestV2{
		V:            transform.ProtocolVersion,
		ID:           "large",
		Op:           "transform",
		Path:         "big.ts",
		Source:       "const x = 1;",
		SourceDigest: transform.DigestString("const x = 1;"),
		Loader:       "ts",
		Format:       "esm",
		NodeMajor:    20,
	}
	var buf bytes.Buffer
	if err := transform.EncodeFrame(&buf, req); err != nil {
		t.Fatal(err)
	}
	if buf.Len() == 0 {
		t.Fatal("empty frame")
	}
}

func TestValidateTransformRequestV2(t *testing.T) {
	validDigest := transform.DigestString("x")
	tests := []struct {
		name    string
		req     transform.TransformRequestV2
		wantErr bool
	}{
		{
			name: "valid",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Path: "a.ts", Source: "x", SourceDigest: validDigest,
				Loader: "ts", Format: "esm", NodeMajor: 20,
			},
			wantErr: false,
		},
		{
			name: "wrong version",
			req: transform.TransformRequestV2{
				V: 1, ID: "1", Op: "transform",
				Path: "a.ts", Source: "x", SourceDigest: validDigest,
				Loader: "ts", Format: "esm", NodeMajor: 20,
			},
			wantErr: true,
		},
		{
			name: "missing path",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Source: "x", SourceDigest: validDigest,
				Loader: "ts", Format: "esm", NodeMajor: 20,
			},
			wantErr: true,
		},
		{
			name: "malformed source digest",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Path: "a.ts", Source: "x", SourceDigest: "not-hex",
				Loader: "ts", Format: "esm", NodeMajor: 20,
			},
			wantErr: true,
		},
		{
			name: "unknown op",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "bundle",
				Path: "a.ts", Source: "x", SourceDigest: validDigest,
				Loader: "ts", Format: "esm", NodeMajor: 20,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestHelloRoundTrip(t *testing.T) {
	hello := transform.HelloRequest{V: transform.ProtocolVersion, Token: "secret"}
	var buf bytes.Buffer
	if err := transform.EncodeFrame(&buf, hello); err != nil {
		t.Fatal(err)
	}
	var got transform.HelloRequest
	if err := transform.DecodeFrame(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got.Token != "secret" {
		t.Fatalf("token=%q", got.Token)
	}
}

func TestHelloRequestValidateWrongVersion(t *testing.T) {
	hello := transform.HelloRequest{V: 1, Token: "secret"}
	err := hello.Validate()
	if err == nil {
		t.Fatal("expected error for wrong hello protocol version")
	}
}

func TestHelloRequestValidateOK(t *testing.T) {
	hello := transform.HelloRequest{V: transform.ProtocolVersion, Token: "secret"}
	if err := hello.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestTransformRequestV2ValidateLoader(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: transform.DigestString("x"),
		Loader: "invalid", Format: "esm", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unknown loader")
	}
}

func TestTransformRequestV2ValidateFormat(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: transform.DigestString("x"),
		Loader: "ts", Format: "umd", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestTransformRequestV2ValidateSourceMapMode(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: transform.DigestString("x"),
		Loader: "ts", Format: "esm", NodeMajor: 20,
		SourceMap: "foobar",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unknown source-map mode")
	}

	// Empty source-map mode is OK (defaults to none).
	req.SourceMap = ""
	if err := req.Validate(); err != nil {
		t.Fatalf("empty source-map should be valid: %v", err)
	}
}

func TestTransformRequestV2ValidateIDTooLong(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: string(make([]byte, 300)), Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: transform.DigestString("x"),
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for too-long ID")
	}
}

func TestEncodeFrameRejectsOversized(t *testing.T) {
	// Create a payload that exceeds MaxFrameSize.
	huge := transform.TransformRequestV2{
		V:            transform.ProtocolVersion,
		ID:           "1",
		Op:           "transform",
		Path:         "a.ts",
		Source:       string(make([]byte, transform.MaxFrameSize)), // > MaxFrameSize when JSON-encoded
		SourceDigest: transform.DigestString(string(make([]byte, transform.MaxFrameSize))),
		Loader:       "ts",
		Format:       "esm",
		NodeMajor:    20,
	}
	var buf bytes.Buffer
	err := transform.EncodeFrame(&buf, huge)
	if err == nil {
		t.Fatal("expected error for oversized frame")
	}
}

func TestSanitizeErrorCode(t *testing.T) {
	stable := transform.SanitizeErrorCode("ERR_M_TRANSFORM_SYNTAX")
	if stable != "ERR_M_TRANSFORM_SYNTAX" {
		t.Fatalf("stable code sanitized: %s", stable)
	}

	unknown := transform.SanitizeErrorCode("ERR_M_TOP_SECRET")
	if unknown != "ERR_M_TRANSFORM_ENGINE" {
		t.Fatalf("unknown code not sanitized: %s", unknown)
	}

	empty := transform.SanitizeErrorCode("")
	if empty != "ERR_M_TRANSFORM_ENGINE" {
		t.Fatalf("empty code not sanitized: %s", empty)
	}
}

func TestSanitizeErrorMessage(t *testing.T) {
	msg := transform.SanitizeErrorMessage("const x = 1; unexpected token")
	if msg == "const x = 1; unexpected token" {
		t.Fatal("source content not sanitized from error message")
	}

	ok := transform.SanitizeErrorMessage("transform timeout")
	if ok != "transform timeout" {
		t.Fatalf("safe message altered: %s", ok)
	}
}

func TestSanitizeErrorMessageRedactsEndpoint(t *testing.T) {
	msg := transform.SanitizeErrorMessage("dial 127.0.0.1:12345: connection refused")
	if msg == "dial 127.0.0.1:12345: connection refused" {
		t.Fatal("endpoint not sanitized")
	}
}

func TestSanitizeErrorMessageRedactsOptions(t *testing.T) {
	msg := transform.SanitizeErrorMessage(`bad option "target": "ES2022"`)
	if msg == `bad option "target": "ES2022"` {
		t.Fatal("options content not sanitized")
	}
}

func TestSanitizeErrorMessageRedactsToken(t *testing.T) {
	// A 64-char hex string looks like a token/digest.
	msg := transform.SanitizeErrorMessage("token abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789 leaked")
	if msg == "token abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789 leaked" {
		t.Fatal("hex token not sanitized")
	}
}

func TestValidateRequestHeader(t *testing.T) {
	tests := []struct {
		name    string
		v       int
		id      string
		op      string
		expect  string
		wantErr bool
	}{
		{name: "valid health", v: transform.ProtocolVersion, id: "1", op: "health", expect: "health", wantErr: false},
		{name: "valid shutdown", v: transform.ProtocolVersion, id: "2", op: "shutdown", expect: "shutdown", wantErr: false},
		{name: "wrong version", v: 1, id: "1", op: "health", expect: "health", wantErr: true},
		{name: "missing id", v: transform.ProtocolVersion, id: "", op: "health", expect: "health", wantErr: true},
		{name: "wrong op", v: transform.ProtocolVersion, id: "1", op: "health", expect: "shutdown", wantErr: true},
		{name: "id too long", v: transform.ProtocolVersion, id: string(make([]byte, 300)), op: "health", expect: "health", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := transform.ValidateRequestHeader(tt.v, tt.id, tt.op, tt.expect)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateRequestHeader() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestCancelRequestValidate(t *testing.T) {
	tests := []struct {
		name    string
		req     transform.CancelRequest
		wantErr bool
	}{
		{
			name: "valid",
			req: transform.CancelRequest{
				V: transform.ProtocolVersion, ID: "1", Op: "cancel", CancelToken: "req-1",
			},
			wantErr: false,
		},
		{
			name: "missing cancel token",
			req: transform.CancelRequest{
				V: transform.ProtocolVersion, ID: "1", Op: "cancel",
			},
			wantErr: true,
		},
		{
			name: "wrong version",
			req: transform.CancelRequest{
				V: 1, ID: "1", Op: "cancel", CancelToken: "req-1",
			},
			wantErr: true,
		},
		{
			name: "wrong op",
			req: transform.CancelRequest{
				V: transform.ProtocolVersion, ID: "1", Op: "transform", CancelToken: "req-1",
			},
			wantErr: true,
		},
		{
			name: "cancel token too long",
			req: transform.CancelRequest{
				V: transform.ProtocolVersion, ID: "1", Op: "cancel",
				CancelToken: string(make([]byte, 300)),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.req.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() error = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestVerifySourceDigest(t *testing.T) {
	source := "const x = 1;"
	validDigest := transform.DigestString(source)

	// Valid match.
	if err := transform.VerifySourceDigest(source, validDigest); err != nil {
		t.Fatalf("valid digest rejected: %v", err)
	}

	// Mismatch.
	if err := transform.VerifySourceDigest(source, transform.DigestString("different")); err == nil {
		t.Fatal("expected mismatch error")
	}

	// Malformed.
	if err := transform.VerifySourceDigest(source, "not-hex"); err == nil {
		t.Fatal("expected malformed error")
	}
	if err := transform.VerifySourceDigest(source, "deadbeef"); err == nil {
		t.Fatal("expected malformed error for short hex")
	}
}

func TestVerifyOptionsDigest(t *testing.T) {
	opts := `{"target":"ES2022"}`
	validDigest := transform.DigestString(opts)

	// Valid match.
	if err := transform.VerifyOptionsDigest(opts, validDigest); err != nil {
		t.Fatalf("valid digest rejected: %v", err)
	}

	// Mismatch.
	if err := transform.VerifyOptionsDigest(opts, transform.DigestString("{}")); err == nil {
		t.Fatal("expected mismatch error")
	}

	// Malformed.
	if err := transform.VerifyOptionsDigest(opts, "not-hex"); err == nil {
		t.Fatal("expected malformed error")
	}
}

func TestTransformRequestV2ValidateNodeMajor(t *testing.T) {
	validDigest := transform.DigestString("x")
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: validDigest,
		Loader: "ts", Format: "esm", NodeMajor: 99,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported node major")
	}
}

func TestTransformRequestV2ValidateSourceDigestMissing(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: "",
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing source digest")
	}
}

func TestTransformRequestV2ValidateOptsDigestMissing(t *testing.T) {
	validDigest := transform.DigestString("x")
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: validDigest,
		Options: `{"target":"ES2022"}`, OptsDigest: "",
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for missing opts digest when options present")
	}
}

func TestTransformRequestV2ValidateOptionsLength(t *testing.T) {
	validDigest := transform.DigestString("x")
	longOpts := `"target":"ES2022",` + string(make([]byte, transform.MaxOptionsLength+1))
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: validDigest,
		Options: longOpts, OptsDigest: transform.DigestString(longOpts),
		Loader: "ts", Format: "esm", NodeMajor: 20,
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for too-long options")
	}
}

func TestTransformRequestV2ValidateCancelTokenLength(t *testing.T) {
	validDigest := transform.DigestString("x")
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", SourceDigest: validDigest,
		Loader: "ts", Format: "esm", NodeMajor: 20,
		CancelToken: string(make([]byte, 300)),
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for too-long cancel token")
	}
}

func TestDigestString(t *testing.T) {
	d := transform.DigestString("hello")
	if len(d) != 64 {
		t.Fatalf("digest length=%d, want 64", len(d))
	}
	// Deterministic.
	if d != transform.DigestString("hello") {
		t.Fatal("digest not deterministic")
	}
}
