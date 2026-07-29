package yarn

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/compat/yarn/berry"
	"github.com/mewisme/mew/internal/compat/yarn/classic"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

// Adapter implements lockfile adapters for yarn.lock (classic and Berry).
type Adapter struct{}

// Read decodes yarn.lock at path into a canonical graph.
func (a Adapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	g, _, err := a.ReadWithExtensions(ctx, path)
	return g, err
}

// Write encodes graph to yarn.lock (non-preserving; prefer EncodePreserving).
func (a Adapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	_, err := a.EncodePreserving(ctx, path, g, nil, nil, lockfile.Detection{})
	return err
}

// ReadWithExtensions reads graph and adapter-owned extensions.
func (a Adapter) ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, lockfile.Extensions, error) {
	_ = ctx
	data, err := readLockBytes(path)
	if err != nil {
		return nil, nil, err
	}
	root := filepath.Dir(path)
	variant := DetectVariant(data, root)
	switch variant {
	case VariantBerryPnP:
		doc, err := berry.Decode(data)
		if err != nil {
			return nil, nil, err
		}
		doc.Linker = "pnp"
		doc.Detection.Format = berry.FormatBerryPnP
		return berry.ToPnPGraph(doc)
	case VariantBerryNM:
		doc, err := berry.Decode(data)
		if err != nil {
			return nil, nil, err
		}
		doc.Linker = "node-modules"
		g, err := berry.ToGraph(doc)
		if err != nil {
			return nil, nil, err
		}
		return g, doc.Extensions, nil
	default:
		doc, err := classic.Decode(data)
		if err != nil {
			return nil, nil, err
		}
		g, err := classic.ToGraph(doc)
		if err != nil {
			return nil, nil, err
		}
		return g, doc.Extensions, nil
	}
}

// WritePreserving never writes live paths; use EncodePreserving for txn staging.
func (a Adapter) WritePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) error {
	_, err := a.EncodePreserving(ctx, path, g, prior, ext, det)
	return err
}

// EncodePreserving applies incumbent write policy and returns staged bytes.
func (a Adapter) EncodePreserving(ctx context.Context, path string, g *graph.Graph, prior []byte, ext lockfile.Extensions, det lockfile.Detection) (lockfile.WriteResult, error) {
	_ = ctx
	_ = det
	if len(prior) == 0 {
		return lockfile.WriteResult{}, lockfile.NewUnsupported("yarn.write", "yarn.lock", "fresh yarn.lock generation is not supported")
	}
	root := filepath.Dir(path)
	variant := DetectVariant(prior, root)
	switch variant {
	case VariantBerryPnP:
		doc, err := berry.Decode(prior)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		priorGraph, _, err := berry.ToPnPGraph(doc)
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
		return lockfile.WriteResult{}, lockfile.NewUnrepresentable("yarn.write", "yarn.lock",
			"graph-changing berry PnP lock mutation is not supported", berry.FieldLossAudit(doc))
	case VariantBerryNM:
		doc, err := berry.Decode(prior)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		priorGraph, err := berry.ToGraph(doc)
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
		return lockfile.WriteResult{}, lockfile.NewUnrepresentable("yarn.write", "yarn.lock",
			"graph-changing berry node-modules lock mutation is not supported in MVP 0025", berry.FieldLossAudit(doc))
	default:
		doc, err := classic.Decode(prior)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		priorGraph, err := classic.ToGraph(doc)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		if err := classic.WriteGate(priorGraph, g, prior); err != nil {
			return lockfile.WriteResult{}, err
		}
		return lockfile.WriteResult{Unchanged: true, Bytes: prior}, nil
	}
}

// LossFromDocument reports fields that would be lost migrating to canonical graph.
func (a Adapter) LossFromDocument(ctx context.Context, prior []byte) (lockfile.LossReport, error) {
	_ = ctx
	variant := DetectVariant(prior, "")
	switch variant {
	case VariantBerryPnP, VariantBerryNM:
		doc, err := berry.Decode(prior)
		if err != nil {
			return lockfile.LossReport{}, err
		}
		report := berry.FieldLossAudit(doc)
		_ = report.Normalize()
		return report, nil
	default:
		doc, err := classic.Decode(prior)
		if err != nil {
			return lockfile.LossReport{}, err
		}
		report := classic.FieldLossAudit(doc)
		_ = report.Normalize()
		return report, nil
	}
}

// IsPnPProject reports whether the project uses Yarn Berry PnP.
func IsPnPProject(root string, prior []byte) bool {
	return DetectVariant(prior, root) == VariantBerryPnP
}

// ExtensionsLinker returns the linker value from extensions, if any.
func ExtensionsLinker(ext lockfile.Extensions) string {
	if ext == nil {
		return ""
	}
	raw, ok := ext[berry.ExtLinkerKey]
	if !ok {
		return ""
	}
	var linker string
	_ = json.Unmarshal(raw, &linker)
	return linker
}

func readLockBytes(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "yarn.read", path, err)
	}
	return data, nil
}

var (
	_ lockfile.ExtensibleAdapter = Adapter{}
	_ lockfile.PreservingEncoder = Adapter{}
)
