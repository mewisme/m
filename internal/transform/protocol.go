// Package transform hosts the Go transform service and IPC.
//
// Protocol: docs/architecture/transform-ipc-sketch.md
package transform

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
)

// ProtocolVersion is the current protocol version.
const ProtocolVersion = 2

// MaxFrameSize is the maximum allowed frame payload in bytes (16 MiB).
const MaxFrameSize = 16 << 20

// MaxSourceSize is the maximum source bytes in a transform request (32 MiB).
const MaxSourceSize = 32 << 20

// MaxPathLength limits path, option, and message strings (4 KiB).
const MaxPathLength = 4096

// Op codes.
const (
	OpHello     = "hello"
	OpHealth    = "health"
	OpTransform = "transform"
	OpCancel    = "cancel"
	OpShutdown  = "shutdown"
)

// HelloRequest carries auth on first connection.
type HelloRequest struct {
	V     int    `json:"v"`
	Token string `json:"token"`
}

// HelloResponse confirms or rejects the session.
type HelloResponse struct {
	V      int    `json:"v"`
	OK     bool   `json:"ok"`
	Reason string `json:"reason,omitempty"`
}

// TransformRequestV2 is the production transform request.
type TransformRequestV2 struct {
	V            int    `json:"v"`
	ID           string `json:"id"`
	Op           string `json:"op"`
	Path         string `json:"path"`
	Source       string `json:"source"`
	SourceDigest string `json:"source_digest"`
	Loader       string `json:"loader"`
	Format       string `json:"format"`
	Options      string `json:"options"` // JSON-encoded NormalizedOptions
	OptsDigest   string `json:"opts_digest"`
	NodeMajor    int    `json:"node_major"`
	SourceMap    string `json:"source_map"` // "none", "inline", "external"
}

// Validate checks required fields and limits.
func (r *TransformRequestV2) Validate() error {
	if r.V != ProtocolVersion {
		return fmt.Errorf("unsupported protocol version %d", r.V)
	}
	if r.ID == "" {
		return fmt.Errorf("missing request id")
	}
	if r.Op != OpTransform {
		return fmt.Errorf("unknown op %q", r.Op)
	}
	if r.Path == "" {
		return fmt.Errorf("missing path")
	}
	if len(r.Path) > MaxPathLength {
		return fmt.Errorf("path too long: %d", len(r.Path))
	}
	if len(r.Source) > MaxSourceSize {
		return fmt.Errorf("source too large: %d", len(r.Source))
	}
	if r.Source == "" && r.SourceDigest == "" {
		return fmt.Errorf("missing source")
	}
	if r.Loader == "" {
		return fmt.Errorf("missing loader")
	}
	return nil
}

// TransformResponseV2 is the production transform response.
type TransformResponseV2 struct {
	V       int    `json:"v"`
	ID      string `json:"id"`
	OK      bool   `json:"ok"`
	Code    string `json:"code,omitempty"`
	Map     string `json:"map,omitempty"` // source map as string
	Digest  string `json:"digest,omitempty"`
	ErrCode string `json:"err_code,omitempty"`
	Error   string `json:"error,omitempty"`
	Cache   string `json:"cache,omitempty"` // "hit", "miss", "bypass"
}

// CancelRequest cancels an in-flight transform.
type CancelRequest struct {
	V  int    `json:"v"`
	ID string `json:"id"`
	Op string `json:"op"`
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
// Rejects frames larger than MaxFrameSize.
func DecodeFrame(r io.Reader, dest any) error {
	var hdr [4]byte
	if _, err := io.ReadFull(r, hdr[:]); err != nil {
		return err
	}
	n := binary.LittleEndian.Uint32(hdr[:])
	if n > MaxFrameSize {
		return fmt.Errorf("transform frame too large: %d (max %d)", n, MaxFrameSize)
	}
	body := make([]byte, n)
	if _, err := io.ReadFull(r, body); err != nil {
		return err
	}
	return json.Unmarshal(body, dest)
}
