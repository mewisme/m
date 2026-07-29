package bun

import (
	"context"
	"encoding/json"
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// Adapter implements lockfile adapters for text bun.lock.
type Adapter struct{}

// Read decodes bun.lock at path into a canonical graph.
func (Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	_ = ctx
	data, err := readLockBytes(path)
	if err != nil {
		return nil, err
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

// Write encodes graph to bun.lock (non-preserving; prefer EncodePreserving).
func (Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	res, err := Adapter{}.EncodePreserving(ctx, path, g, nil, nil, lockfile.Detection{Format: FormatV1, Confidence: lockfile.DetectionCertain})
	if err != nil {
		return err
	}
	return os.WriteFile(path, res.Bytes, 0o644)
}

// ReadWithExtensions reads graph and adapter-owned extensions.
func (Adapter) ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, lockfile.Extensions, error) {
	_ = ctx
	data, err := readLockBytes(path)
	if err != nil {
		return nil, nil, err
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
	_ = path
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
	same, err := lockfile.GraphsEqual(priorGraph, g)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if same {
		return lockfile.WriteResult{Unchanged: true, Bytes: prior}, nil
	}

	return lockfile.WriteResult{}, lockfile.NewUnrepresentable("bun.write", "bun.lock",
		"graph-changing bun.lock mutation is not supported in MVP 0025", FieldLossAudit(doc))
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
	if g == nil {
		return nil, apperr.New(apperr.Lockfile, "bun.write", "bun.lock", "nil graph")
	}
	if det.Format == "" {
		det = lockfile.Detection{Format: FormatV1, ProducerMajor: 1, Confidence: lockfile.DetectionCertain}
	}
	doc := &Document{
		LockfileVersion: det.ProducerMajor,
		ConfigVersion:   1,
		Workspaces:      map[string]WorkspaceEntry{"": {Name: "project"}},
		Packages:        map[string]PackageArray{},
		Extensions:      lockfile.Extensions{},
		Detection:       det,
	}
	for _, pkg := range g.Packages {
		short := pkg.ID.Name
		resolution := PackageKey(pkg.ID.Name, pkg.ID.Version)
		arr := PackageArray{}
		arr = append(arr, mustRaw(resolution), mustRaw(""), mustRaw(PackageInfo{}))
		if pkg.Integrity != "" {
			arr = append(arr, mustRaw(pkg.Integrity))
		}
		doc.Packages[short] = arr
	}
	return Encode(doc)
}

func mustRaw(v any) json.RawMessage {
	b, _ := json.Marshal(v)
	return b
}

func readLockBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "bun.read", path, err)
	}
	return data, nil
}

var (
	_ lockfile.ExtensibleAdapter = Adapter{}
	_ lockfile.PreservingEncoder = Adapter{}
)
