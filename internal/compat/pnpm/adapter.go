package pnpm

import (
	"context"
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// Adapter implements lockfile adapters for pnpm-lock.yaml.
type Adapter struct{}

// Read decodes pnpm-lock.yaml at path into a canonical graph.
func (Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	_ = ctx
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "pnpm.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, err
	}
	if err := rejectLegacy(doc); err != nil {
		return nil, err
	}
	return ToGraph(doc)
}

// Write encodes graph to pnpm-lock.yaml (non-preserving; prefer EncodePreserving).
func (a Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	res, err := a.EncodePreserving(ctx, path, g, nil, nil, lockfile.Detection{
		Format: FormatV9, ProducerMajor: 9, ExplicitMajor: true, Confidence: lockfile.DetectionInferred,
	})
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
		return nil, nil, apperr.Wrap(apperr.IO, "pnpm.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, nil, err
	}
	if err := rejectLegacy(doc); err != nil {
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
	if err := rejectLegacy(doc); err != nil {
		return lockfile.WriteResult{}, err
	}
	if ext != nil {
		doc.Extensions = ext
	}
	if det.Format == "" {
		var derr error
		det, derr = lockfile.DetectPnpmWithMajor(prior, det.ProducerMajor)
		if derr != nil {
			return lockfile.WriteResult{}, derr
		}
		if det.ProducerMajor != 0 {
			det.ExplicitMajor = true
		}
	}
	doc.Detection = det

	priorGraph, err := ToGraph(doc)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	same, err := GraphsEqual(priorGraph, g)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if same {
		return lockfile.WriteResult{Unchanged: true, Bytes: prior}, nil
	}

	if !det.Certified() {
		return lockfile.WriteResult{}, lockfile.NewAmbiguous("pnpm.write", "pnpm-lock.yaml", "ambiguous pnpm lock generation; set --pnpm-major")
	}

	switch det.Format {
	case FormatV9, FormatV10, FormatV11:
	default:
		return lockfile.WriteResult{}, lockfile.NewUnsupported("pnpm.write", "pnpm-lock.yaml", "unsupported generation (only pnpm 9/10/11)")
	}

	outDoc, err := FromGraph(g, doc, det)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if report := lossAgainstPrior(doc, outDoc); len(report.Items) > 0 {
		return lockfile.WriteResult{}, lockfile.NewUnrepresentable("pnpm.write", "pnpm-lock.yaml", "lossy encode", report)
	}
	encoded, err := Encode(outDoc)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	return lockfile.WriteResult{Bytes: encoded}, nil
}

// LossFromDocument reports fields that would be lost migrating to canonical graph.
func (Adapter) LossFromDocument(ctx context.Context, prior []byte) (lockfile.LossReport, error) {
	_ = ctx
	doc, err := Decode(prior)
	if err != nil {
		return lockfile.LossReport{}, err
	}
	if err := rejectLegacy(doc); err != nil {
		return lockfile.LossReport{}, err
	}
	report := FieldLossAudit(doc)
	_ = report.Normalize()
	return report, nil
}

func encodeFresh(g *graph.Graph, det lockfile.Detection) ([]byte, error) {
	if det.Format == "" {
		det = lockfile.Detection{Format: FormatV9, ProducerMajor: 9, ExplicitMajor: true, Confidence: lockfile.DetectionInferred}
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
