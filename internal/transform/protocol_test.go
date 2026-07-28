package transform_test

import (
	"bytes"
	"testing"

	"github.com/mewisme/mew/internal/transform"
)

func TestFrameRoundTrip(t *testing.T) {
	req := transform.Request{
		V:      transform.ProtocolVersion,
		ID:     "req-1",
		Op:     "transform",
		Path:   "src/a.ts",
		Source: "const x: number = 1",
	}
	var buf bytes.Buffer
	if err := transform.EncodeFrame(&buf, req); err != nil {
		t.Fatal(err)
	}
	var got transform.Request
	if err := transform.DecodeFrame(&buf, &got); err != nil {
		t.Fatal(err)
	}
	if got != req {
		t.Fatalf("got %+v want %+v", got, req)
	}

	resp := transform.Response{
		V:    transform.ProtocolVersion,
		ID:   "req-1",
		OK:   true,
		Code: "const x = 1",
	}
	buf.Reset()
	if err := transform.EncodeFrame(&buf, resp); err != nil {
		t.Fatal(err)
	}
	var gotResp transform.Response
	if err := transform.DecodeFrame(&buf, &gotResp); err != nil {
		t.Fatal(err)
	}
	if gotResp.V != resp.V || gotResp.ID != resp.ID || gotResp.OK != resp.OK || gotResp.Code != resp.Code {
		t.Fatalf("got %+v want %+v", gotResp, resp)
	}
}
