// Package transform hosts the Go transform service and IPC.
//
// Protocol: docs/architecture/transform-ipc-sketch.md
package transform

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// ProtocolVersion is the current protocol version.
const ProtocolVersion = 2

// MaxFrameSize is the maximum allowed frame payload in bytes (48 MiB).
// Must be large enough to hold a MaxSourceSize source plus JSON encoding overhead.
const MaxFrameSize = 48 << 20

// MaxSourceSize is the maximum source bytes in a transform request (32 MiB).
const MaxSourceSize = 32 << 20

// MaxPathLength limits path, option, and message strings (4 KiB).
const MaxPathLength = 4096

// MaxIDLength limits request IDs (256 bytes).
const MaxIDLength = 256

// Op codes.
const (
	OpHello     = "hello"
	OpHealth    = "health"
	OpTransform = "transform"
	OpCancel    = "cancel"
	OpShutdown  = "shutdown"
)

// ValidLoaderKinds lists the loader strings accepted on the wire.
var ValidLoaderKinds = map[string]bool{
	"ts": true, "mts": true, "cts": true,
}

// ValidFormats lists the format strings accepted on the wire.
var ValidFormats = map[string]bool{
	"esm": true, "cjs": true,
}

// ValidSourceMapModes lists the source-map strings accepted on the wire.
var ValidSourceMapModes = map[string]bool{
	"none": true, "inline": true, "external": true,
}

// HelloRequest carries auth on first connection.
type HelloRequest struct {
	V     int    `json:"v"`
	Token string `json:"token"`
}

// Validate checks protocol version.
func (r *HelloRequest) Validate() error {
	if r.V != ProtocolVersion {
		return apperr.New(apperr.TransformProtocolVersion, "transform.protocol", "",
			fmt.Sprintf("unsupported hello protocol version %d", r.V))
	}
	return nil
}

// HelloResponse confirms or rejects the session.
type HelloResponse struct {
	V       int    `json:"v"`
	OK      bool   `json:"ok"`
	ErrCode string `json:"err_code,omitempty"`
	Reason  string `json:"reason,omitempty"`
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
	SourceMap    string `json:"source_map"`             // "none", "inline", "external"
	CancelToken  string `json:"cancel_token,omitempty"` // ID used by OpCancel to cancel this request
}

// Validate checks required fields, limits, and enum values.
func (r *TransformRequestV2) Validate() error {
	if r.V != ProtocolVersion {
		return apperr.New(apperr.TransformProtocolVersion, "transform.protocol", r.ID,
			fmt.Sprintf("unsupported protocol version %d", r.V))
	}
	if r.ID == "" {
		return apperr.New(apperr.Usage, "transform.protocol", "", "missing request id")
	}
	if len(r.ID) > MaxIDLength {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID, "request id too long")
	}
	if r.Op != OpTransform {
		return apperr.New(apperr.Unsupported, "transform.protocol", r.ID,
			fmt.Sprintf("unknown op %q", r.Op))
	}
	if r.Path == "" {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID, "missing path")
	}
	if len(r.Path) > MaxPathLength {
		return apperr.New(apperr.TransformFrameSize, "transform.protocol", r.ID,
			fmt.Sprintf("path too long: %d", len(r.Path)))
	}
	if len(r.Source) > MaxSourceSize {
		return apperr.New(apperr.TransformFrameSize, "transform.protocol", r.ID,
			fmt.Sprintf("source too large: %d", len(r.Source)))
	}
	if r.Source == "" && r.SourceDigest == "" {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID, "missing source")
	}
	if !ValidLoaderKinds[r.Loader] {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID,
			fmt.Sprintf("unknown loader %q", r.Loader))
	}
	if !ValidFormats[r.Format] {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID,
			fmt.Sprintf("unknown format %q", r.Format))
	}
	if r.SourceMap != "" && !ValidSourceMapModes[r.SourceMap] {
		return apperr.New(apperr.Usage, "transform.protocol", r.ID,
			fmt.Sprintf("unknown source-map mode %q", r.SourceMap))
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
// Rejects frames larger than MaxFrameSize before writing.
func EncodeFrame(w io.Writer, v any) error {
	body, err := json.Marshal(v)
	if err != nil {
		return err
	}
	if len(body) > MaxFrameSize {
		return fmt.Errorf("transform frame too large for encode: %d (max %d)", len(body), MaxFrameSize)
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
// Rejects frames larger than MaxFrameSize before allocating body.
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

// stableErrorCodes lists error codes that are safe to return to clients.
// Source content, secrets, and internal details must not appear in diagnostics.
var stableErrorCodes = map[string]bool{
	string(apperr.TransformSyntax):          true,
	string(apperr.TransformUnsupported):     true,
	string(apperr.TransformConfigParse):     true,
	string(apperr.TransformConfigExtends):   true,
	string(apperr.TransformConfigOption):    true,
	string(apperr.TransformProtocolVersion): true,
	string(apperr.TransformAuth):            true,
	string(apperr.TransformFrameSize):       true,
	string(apperr.TransformTimeout):         true,
	string(apperr.TransformCancelled):       true,
	string(apperr.TransformUnavailable):     true,
	string(apperr.TransformCacheCorrupt):    true,
	string(apperr.TransformEngine):          true,
	string(apperr.Usage):                    true,
	string(apperr.Unsupported):              true,
	string(apperr.Integrity):                true,
	string(apperr.Cancelled):                true,
}

// SanitizeErrorCode returns code if it is a stable, safe-to-expose error code;
// otherwise returns the generic engine error code.
func SanitizeErrorCode(code string) string {
	if stableErrorCodes[code] {
		return code
	}
	return string(apperr.TransformEngine)
}

// SanitizeErrorMessage returns msg if it is safe to expose; strips source content.
func SanitizeErrorMessage(msg string) string {
	if len(msg) > MaxPathLength {
		msg = msg[:MaxPathLength] + "..."
	}
	// Redact source content that may have leaked into error messages.
	if strings.Contains(msg, "const ") || strings.Contains(msg, "import ") {
		return "transform error (details redacted)"
	}
	return msg
}
