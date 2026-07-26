// Package transform hosts the Go transform service and IPC sketch.
//
// Protocol sketch: docs/architecture/transform-ipc-sketch.md
package transform

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// ProtocolVersion is the sketch framing version (MVP 0051 may bump this).
const ProtocolVersion = 1

// Request is a length-prefixed JSON transform request body.
type Request struct {
	V           int    `json:"v"`
	ID          string `json:"id"`
	Op          string `json:"op"`
	Path        string `json:"path,omitempty"`
	Source      string `json:"source,omitempty"`
	CancelToken string `json:"cancel_token,omitempty"`
}

// Response is a length-prefixed JSON transform response body.
type Response struct {
	V     int     `json:"v"`
	ID    string  `json:"id"`
	OK    bool    `json:"ok"`
	Code  string  `json:"code,omitempty"`
	Map   *string `json:"map"`
	Error *string `json:"error"`
}

// EncodeFrame writes a u32le length-prefixed JSON payload.
func EncodeFrame(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	var hdr [4]byte
	binary.LittleEndian.PutUint32(hdr[:], uint32(len(body)))
	if _, err := w.Write(hdr[:]); err != nil {
		return err
	}
	_, err = w.Write(body)
	return err
}

// DecodeFrame reads a u32le length-prefixed JSON payload into dest.
func DecodeFrame(r io.Reader, dest any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n > 16<<20 {
		return fmt.Errorf("transform frame too large: %d", n)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}
