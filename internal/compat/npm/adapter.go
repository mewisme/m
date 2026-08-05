package npm

import (
	"context"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// Adapter implements lockfile adapters for package-lock.json and npm-shrinkwrap.json.
type Adapter struct{}

// Read decodes an npm lock at path into a canonical graph.
func (Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	_ = ctx
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "npm.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	if err := ValidateSupported(doc); err != nil {
		return nil, err
	}
	return ToGraph(doc)
}

// Write encodes graph to npm lock (non-preserving; prefer EncodePreserving).
func (Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	res, err := Adapter{}.EncodePreserving(ctx, path, g, nil, nil, lockfile.Detection{Format: FormatV3, Confidence: lockfile.DetectionCertain})
	if err != nil {
		return err
	}
	return os.WriteFile(path, res.Bytes, 0o644)
}

// ReadWithExtensions reads graph and adapter-owned extensions.
func (Adapter) ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, lockfile.Extensions, error) {
	_ = ctx
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.IO, "npm.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, nil, err
	}
	if err := ValidateSupported(doc); err != nil {
		return nil, nil, err
	}
	g, err := ToGraph(doc)
	if err != nil {
		return nil, nil, err
	}
	return g, doc.Extensions, nil
}

// WritePreserving never writes live paths; use EncodePreserving for txn staging.
func (a Adapter) WritePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) error {
	_, err := a.EncodePreserving(ctx, path, g, prior, ext, det)
	return err
}

// EncodePreserving applies incumbent write policy and returns staged bytes.
func (Adapter) EncodePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) (lockfile.WriteResult, error) {
	_ = ctx
	if len(prior) == 0 {
		out, err := encodeFresh(g, det)
		return lockfile.WriteResult{Bytes: out}, err
	}

	doc, err := Decode(prior)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if err := ValidateSupported(doc); err != nil {
		return lockfile.WriteResult{}, err
	}
	if ext != nil {
		doc.Extensions = ext
	}
	if det.Format == "" {
		det = DetectFromDocument(doc)
	}
	doc.Detection = det

	priorGraph, err := ToGraph(doc)
	if err != nil {
		return lockfile.WriteResult{}, err
	}

	// Normalize TarballURL before comparison — resolved URLs differ across
	// registries but do not represent a semantic graph change.
	normalized := cloneGraphForCompare(g)
	for i := range priorGraph.Packages {
		priorGraph.Packages[i].TarballURL = ""
	}
	same, err := lockfile.GraphsEqual(priorGraph, normalized)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if same {
		return lockfile.WriteResult{Unchanged: true, Bytes: prior}, nil
	}

	subject := filepath.Base(path)
	if subject == "" || subject == "." {
		subject = "package-lock.json"
	}
	return lockfile.WriteResult{}, ErrMutationUnsupported("npm.write", subject)
}

func cloneGraphForCompare(g *graph.Graph) *graph.Graph {
	data, err := graph.EncodeJSON(g)
	if err != nil {
		return g
	}
	cp, err := graph.DecodeJSON(data)
	if err != nil {
		return g
	}
	for i := range cp.Packages {
		cp.Packages[i].TarballURL = ""
	}
	return cp
}

// LossFromDocument reports fields that would be lost migrating to canonical graph.
func (Adapter) LossFromDocument(ctx context.Context, prior []byte) (lockfile.LossReport, error) {
	_ = ctx
	doc, err := Decode(prior)
	if err != nil {
		return lockfile.LossReport{}, err
	}
	if err := ValidateSupported(doc); err != nil {
		return lockfile.LossReport{}, err
	}
	report := FieldLossAudit(doc)
	_ = report.Normalize()
	return report, nil
}

func encodeFresh(g *graph.Graph, det lockfile.Detection) ([]byte, error) {
	if det.Format == "" {
		det = lockfile.Detection{Format: FormatV3, ProducerMajor: 3, Confidence: lockfile.DetectionCertain}
	}
	doc, err := FromGraph(g, nil, det)
	if err != nil {
		return nil, err
	}
	return Encode(doc)
}

var (
	_ lockfile.ExtensibleAdapter = Adapter{}
	_ lockfile.PreservingEncoder = Adapter{}
)
