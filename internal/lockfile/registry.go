package lockfile

import (
	"context"
	"path/filepath"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/project"
)

var (
	adapterMu     sync.RWMutex
	adapters      = map[project.Identity]LockfileAdapter{}
	extAdapters   = map[project.Identity]ExtensibleAdapter{}
	defaultMew    LockfileAdapter
	defaultMewExt ExtensibleAdapter
)

// RegisterAdapter registers a lock adapter for identity.
func RegisterAdapter(id project.Identity, a LockfileAdapter) {
	adapterMu.Lock()
	defer adapterMu.Unlock()
	adapters[id] = a
}

// RegisterExtAdapter registers an extensible adapter for identity.
func RegisterExtAdapter(id project.Identity, a ExtensibleAdapter) {
	adapterMu.Lock()
	defer adapterMu.Unlock()
	extAdapters[id] = a
	adapters[id] = a
}

// RegisterDefaultMewAdapter sets the fallback m.lock adapter.
func RegisterDefaultMewAdapter(a LockfileAdapter, ext ExtensibleAdapter) {
	adapterMu.Lock()
	defer adapterMu.Unlock()
	defaultMew = a
	defaultMewExt = ext
}

// AdapterFor returns the lock adapter for project identity.
func AdapterFor(id project.Identity) LockfileAdapter {
	adapterMu.RLock()
	defer adapterMu.RUnlock()
	if a, ok := adapters[id]; ok {
		return a
	}
	if id == project.IdentityMew {
		return defaultMew
	}
	return nil
}

// ExtAdapterFor returns an extensible adapter when supported.
func ExtAdapterFor(id project.Identity) (ExtensibleAdapter, bool) {
	adapterMu.RLock()
	defer adapterMu.RUnlock()
	if a, ok := extAdapters[id]; ok {
		return a, true
	}
	if id == project.IdentityMew && defaultMewExt != nil {
		return defaultMewExt, true
	}
	return nil, false
}

// ReadGraph reads the incumbent lock for identity into a canonical graph.
func ReadGraph(ctx context.Context, root string, id project.Identity) (*graph.Graph, error) {
	path, ok := project.IncumbentLockPath(root, id)
	if !ok {
		return nil, apperr.New(apperr.NotFound, "lock.read", project.LockFilename(id), "lockfile not found")
	}
	a := AdapterFor(id)
	if a == nil {
		return nil, NewUnsupported("lock.read", project.LockFilename(id), "adapter not implemented")
	}
	return a.Read(ctx, path)
}

// WriteGraph writes graph to the incumbent lockfile path for identity.
func WriteGraph(ctx context.Context, root string, id project.Identity, g *graph.Graph) error {
	name := project.LockFilename(id)
	if name == "" {
		return NewUnsupported("lock.write", string(id), "no incumbent lockfile for identity")
	}
	path := filepath.Join(root, name)
	a := AdapterFor(id)
	if a == nil {
		return NewUnsupported("lock.write", name, "adapter not implemented")
	}
	return a.Write(ctx, path, g)
}

// EncodePreserving runs byte-preserving encode when the adapter supports it.
func EncodePreserving(ctx context.Context, ext ExtensibleAdapter, path string, g *graph.Graph, prior []byte, extensions Extensions, det Detection) (WriteResult, error) {
	enc, ok := ext.(PreservingEncoder)
	if !ok {
		return WriteResult{}, NewUnsupported("lock.encode", path, "adapter does not support byte-preserving encode")
	}
	return enc.EncodePreserving(ctx, path, g, prior, extensions, det)
}
