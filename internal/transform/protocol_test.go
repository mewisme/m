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
		SourceDigest: "abc",
		Loader:       "ts",
		Format:       "esm",
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
		V:      transform.ProtocolVersion,
		ID:     "large",
		Op:     "transform",
		Path:   "big.ts",
		Source: "const x = 1;",
		Loader: "ts",
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
	tests := []struct {
		name    string
		req     transform.TransformRequestV2
		wantErr bool
	}{
		{
			name: "valid",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Path: "a.ts", Source: "x", Loader: "ts", Format: "esm",
			},
			wantErr: false,
		},
		{
			name: "wrong version",
			req: transform.TransformRequestV2{
				V: 1, ID: "1", Op: "transform",
				Path: "a.ts", Source: "x", Loader: "ts", Format: "esm",
			},
			wantErr: true,
		},
		{
			name: "missing path",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Source: "x", Loader: "ts", Format: "esm",
			},
			wantErr: true,
		},
		{
			name: "missing source",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "transform",
				Path: "a.ts", Loader: "ts", Format: "esm",
			},
			wantErr: true,
		},
		{
			name: "unknown op",
			req: transform.TransformRequestV2{
				V: transform.ProtocolVersion, ID: "1", Op: "bundle",
				Path: "a.ts", Source: "x", Loader: "ts", Format: "esm",
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
		Path: "a.ts", Source: "x", Loader: "invalid", Format: "esm",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unknown loader")
	}
}

func TestTransformRequestV2ValidateFormat(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", Loader: "ts", Format: "umd",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for unknown format")
	}
}

func TestTransformRequestV2ValidateSourceMapMode(t *testing.T) {
	req := transform.TransformRequestV2{
		V: transform.ProtocolVersion, ID: "1", Op: "transform",
		Path: "a.ts", Source: "x", Loader: "ts", Format: "esm",
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
		Path: "a.ts", Source: "x", Loader: "ts", Format: "esm",
	}
	err := req.Validate()
	if err == nil {
		t.Fatal("expected error for too-long ID")
	}
}

func TestEncodeFrameRejectsOversized(t *testing.T) {
	// Create a payload that exceeds MaxFrameSize.
	huge := transform.TransformRequestV2{
		V:      transform.ProtocolVersion,
		ID:     "1",
		Op:     "transform",
		Path:   "a.ts",
		Source: string(make([]byte, transform.MaxFrameSize)), // > MaxFrameSize when JSON-encoded
		Loader: "ts",
		Format: "esm",
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
