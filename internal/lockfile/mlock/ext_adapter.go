package mlock

import (
	"context"
	"os"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile"
)

func init() {
	lockfile.RegisterDefaultMewAdapter(Adapter{}, ExtAdapter{})
}

// ExtAdapter implements lockfile.ExtensibleAdapter and PreservingEncoder for m.lock.
type ExtAdapter struct{}

func (ExtAdapter) Read(ctx context.Context, path string) (*graph.Graph, error) {
	return Adapter{}.Read(ctx, path)
}

func (ExtAdapter) Write(ctx context.Context, path string, g *graph.Graph) error {
	return Adapter{}.Write(ctx, path, g)
}

func (ExtAdapter) ReadWithExtensions(ctx context.Context, path string) (*graph.Graph, lockfile.Extensions, error) {
	_ = ctx
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, apperr.Wrap(apperr.IO, "mlock.read", path, err)
	}
	doc, err := Decode(data)
	if err != nil {
		return nil, nil, err
	}
	g, err := ToGraph(doc)
	if err != nil {
		return nil, nil, err
	}
	return g, doc.Extensions, nil
}

func (a ExtAdapter) WritePreserving(
	ctx context.Context,
	path string,
	g *graph.Graph,
	prior []byte,
	ext lockfile.Extensions,
	det lockfile.Detection,
) error {
	res, err := a.EncodePreserving(ctx, path, g, prior, ext, det)
	if err != nil {
		return err
	}
	if res.Unchanged {
		return nil
	}
	return Adapter{}.Write(ctx, path, g)
}

func (ExtAdapter) EncodePreserving(
	_ context.Context,
	_ string,
	g *graph.Graph,
	prior []byte,
	ext lockfile.Extensions,
	det lockfile.Detection,
) (lockfile.WriteResult, error) {
	if len(prior) > 0 {
		doc, err := Decode(prior)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		priorG, err := ToGraph(doc)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		equal, err := lockfile.GraphsEqual(priorG, g)
		if err != nil {
			return lockfile.WriteResult{}, err
		}
		if equal {
			return lockfile.WriteResult{Unchanged: true, Bytes: prior}, nil
		}
	}
	if !det.Certified() {
		return lockfile.WriteResult{}, lockfile.NewAmbiguous("lock.write", "m.lock", "ambiguous lock generation; use --pnpm-major")
	}
	specs := SpecifiersFromGraph(g)
	doc, err := FromGraph(g, specs, DefaultSettings())
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	if len(ext) > 0 {
		doc.Extensions = ext
	}
	data, err := Encode(doc)
	if err != nil {
		return lockfile.WriteResult{}, err
	}
	return lockfile.WriteResult{Bytes: data}, nil
}

func (ExtAdapter) LossFromDocument(_ context.Context, prior []byte) (lockfile.LossReport, error) {
	if len(prior) == 0 {
		return lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}, nil
	}
	_, err := Decode(prior)
	if err != nil {
		return lockfile.LossReport{}, err
	}
	return lockfile.LossReport{SchemaVersion: lockfile.LossReportSchemaVersion, Items: []lockfile.LossItem{}}, nil
}
