package nub

import (
	"context"
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/pnpm"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

const formatNub = "nub"

// Adapter implements lockfile adapters for nub.lock (pnpm v9-shaped codec).
type Adapter struct {
	inner pnpm.Adapter
}

// Read decodes nub.lock at path into a canonical graph.
func (a Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	return a.inner.Read(ctx, path)
}

// Write encodes graph to nub.lock.
func (a Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	res, err := a.EncodePreserving(ctx, path, g, nil, nil, lockfile.Detection{Format: formatNub, Confidence: lockfile.DetectionCertain})
	if err != nil {
		return err
	}
	return os.WriteFile(path, res.Bytes, 0o644)
}

// ReadWithExtensions reads graph and nub-specific extensions.
func (a Adapter) ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, lockfile.Extensions, error) {
	return a.inner.ReadWithExtensions(ctx, path)
}

// WritePreserving never writes live paths.
func (a Adapter) WritePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) error {
	_, err := a.EncodePreserving(ctx, path, g, prior, ext, det)
	return err
}

// EncodePreserving applies incumbent write policy for nub.lock.
func (a Adapter) EncodePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) (lockfile.WriteResult, error) {
	pnpmDet := det
	if pnpmDet.Format == "" || pnpmDet.Format == formatNub {
		if len(prior) > 0 {
			inferred, err := lockfile.DetectPnpmWithMajor(prior, det.ProducerMajor)
			if err != nil {
				return lockfile.WriteResult{}, err
			}
			pnpmDet = inferred
		} else if pnpmDet.ProducerMajor == 0 {
			pnpmDet = lockfile.Detection{Format: pnpm.FormatV9, ProducerMajor: 9, Confidence: lockfile.DetectionInferred, ExplicitMajor: true}
		}
	}
	return a.inner.EncodePreserving(ctx, path, g, prior, ext, pnpmDet)
}

// LossFromDocument reports nub-specific extension loss.
func (a Adapter) LossFromDocument(ctx context.Context, prior []byte) (lockfile.LossReport, error) {
	report, err := a.inner.LossFromDocument(ctx, prior)
	if err != nil {
		return report, err
	}
	doc, err := pnpm.Decode(prior)
	if err != nil {
		return report, err
	}
	for k := range doc.Extensions {
		if isNubExtensionKey(k) {
			report.Items = append(report.Items, lockfile.LossItem{
				Field:        k,
				Reason:       "nub extension not mapped to canonical graph",
				SourceFormat: formatNub,
			})
		}
	}
	_ = report.Normalize()
	return report, nil
}

func isNubExtensionKey(k string) bool {
	return k == "nub" || k == "nubVersion" || k == "nubLockVersion"
}

var (
	_ lockfile.ExtensibleAdapter = Adapter{}
	_ lockfile.PreservingEncoder = Adapter{}
)

// ErrRead is a typed read failure helper for tests.
func ErrRead(path string, err error) error {
	return apperr.Wrap(apperr.IO, "nub.read", path, err)
}
