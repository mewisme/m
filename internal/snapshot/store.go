package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/graph"
)

const (
	indexName        = "index.json"
	manifestFileName = "package.json"
	lockFileName     = "m.lock"
	metaFileName     = "meta.json"
)

// Record is a loaded snapshot with manifest and lock bytes.
type Record struct {
	Meta     *Snapshot
	Manifest []byte
	Lock     []byte
}

type indexDoc struct {
	SchemaVersion int      `json:"schemaVersion"`
	NextSeq       int      `json:"nextSeq"`
	IDs           []string `json:"ids"`
}

// Store persists snapshots under <project>/.mew/snapshots/.
type Store struct {
	Root string
}

// NewStore returns a snapshot store for projectRoot.
func NewStore(projectRoot string) *Store {
	return &Store{Root: filepath.Join(projectRoot, ".mew", "snapshots")}
}

// GraphDigest returns a stable sha256 digest of canonical graph JSON.
func GraphDigest(g *graph.Graph) (string, error) {
	data, err := graph.EncodeJSON(g)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

// StageCreate writes snapshot payload under stageRoot without touching the live index.
func (s *Store) StageCreate(stageRoot, id string, manifest, lock []byte, graphDigest string) error {
	if s == nil || stageRoot == "" {
		return apperr.New(apperr.Internal, "snapshot.stage", id, "nil store or stage root")
	}
	if id == "" {
		return apperr.New(apperr.Internal, "snapshot.stage", "", "empty id")
	}
	dir := filepath.Join(stageRoot, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.stage", dir, err)
	}
	meta := &Snapshot{
		SchemaVersion: SchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC(),
		GraphDigest:   graphDigest,
	}
	metaBytes, err := EncodeJSON(meta)
	if err != nil {
		return err
	}
	for name, data := range map[string][]byte{
		metaFileName:     metaBytes,
		manifestFileName: manifest,
		lockFileName:     lock,
	} {
		if err := writeAtomic(filepath.Join(dir, name), data); err != nil {
			return err
		}
	}
	return nil
}

// StageIndex writes the snapshot index document under stageRoot.
func (s *Store) StageIndex(stageRoot string, ids []string, nextSeq int) error {
	if s == nil || stageRoot == "" {
		return apperr.New(apperr.Internal, "snapshot.stage-index", "", "nil store or stage root")
	}
	idx := &indexDoc{SchemaVersion: SchemaVersion, NextSeq: nextSeq, IDs: append([]string(nil), ids...)}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(idx); err != nil {
		return apperr.Wrap(apperr.Internal, "snapshot.stage-index", indexName, err)
	}
	return writeAtomic(filepath.Join(stageRoot, indexName), buf.Bytes())
}

// PlannedIndex returns live index ids plus the next snapshot id and next sequence.
func (s *Store) PlannedIndex() (ids []string, nextID string, nextSeq int, err error) {
	idx, err := s.readIndex()
	if err != nil {
		return nil, "", 0, err
	}
	nextID = fmt.Sprintf("%06d", idx.NextSeq)
	nextSeq = idx.NextSeq + 1
	ids = append(append([]string(nil), idx.IDs...), nextID)
	return ids, nextID, nextSeq, nil
}

// Create writes manifest, lock, and metadata for id.
func (s *Store) Create(id string, manifest, lock []byte, graphDigest string) error {
	if s == nil || s.Root == "" {
		return apperr.New(apperr.Internal, "snapshot.create", "", "nil store")
	}
	if id == "" {
		return apperr.New(apperr.Internal, "snapshot.create", "", "empty id")
	}
	dir := filepath.Join(s.Root, id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.create", dir, err)
	}
	meta := &Snapshot{
		SchemaVersion: SchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC(),
		GraphDigest:   graphDigest,
	}
	metaBytes, err := EncodeJSON(meta)
	if err != nil {
		return err
	}
	for name, data := range map[string][]byte{
		metaFileName:     metaBytes,
		manifestFileName: manifest,
		lockFileName:     lock,
	} {
		if err := writeAtomic(filepath.Join(dir, name), data); err != nil {
			return err
		}
	}
	return s.appendID(id)
}

// List returns snapshots newest-first.
func (s *Store) List() ([]Snapshot, error) {
	if s == nil || s.Root == "" {
		return nil, apperr.New(apperr.Internal, "snapshot.list", "", "nil store")
	}
	idx, err := s.readIndex()
	if err != nil {
		return nil, err
	}
	out := make([]Snapshot, 0, len(idx.IDs))
	for i := len(idx.IDs) - 1; i >= 0; i-- {
		rec, err := s.Load(idx.IDs[i])
		if err != nil {
			return nil, err
		}
		out = append(out, *rec.Meta)
	}
	return out, nil
}

// Load reads snapshot id.
func (s *Store) Load(id string) (*Record, error) {
	if s == nil || s.Root == "" {
		return nil, apperr.New(apperr.Internal, "snapshot.load", id, "nil store")
	}
	dir := filepath.Join(s.Root, id)
	metaBytes, err := os.ReadFile(filepath.Join(dir, metaFileName))
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "snapshot.load", id, err)
	}
	meta, err := DecodeJSON(metaBytes)
	if err != nil {
		return nil, err
	}
	manifest, err := os.ReadFile(filepath.Join(dir, manifestFileName))
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "snapshot.load", id, err)
	}
	lock, err := os.ReadFile(filepath.Join(dir, lockFileName))
	if err != nil {
		return nil, apperr.Wrap(apperr.IO, "snapshot.load", id, err)
	}
	return &Record{Meta: meta, Manifest: manifest, Lock: lock}, nil
}

// Prune retains the newest retain snapshots and removes older dirs.
func (s *Store) Prune(retain int) error {
	if s == nil || s.Root == "" {
		return apperr.New(apperr.Internal, "snapshot.prune", "", "nil store")
	}
	if retain < 0 {
		retain = 0
	}
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	if len(idx.IDs) <= retain {
		return nil
	}
	remove := idx.IDs[:len(idx.IDs)-retain]
	keep := idx.IDs[len(idx.IDs)-retain:]
	for _, id := range remove {
		_ = os.RemoveAll(filepath.Join(s.Root, id))
	}
	idx.IDs = keep
	return s.writeIndex(idx)
}

// NextID allocates a monotonic zero-padded snapshot id.
func (s *Store) NextID() (string, error) {
	idx, err := s.readIndex()
	if err != nil {
		return "", err
	}
	id := fmt.Sprintf("%06d", idx.NextSeq)
	idx.NextSeq++
	if err := s.writeIndex(idx); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Store) appendID(id string) error {
	idx, err := s.readIndex()
	if err != nil {
		return err
	}
	idx.IDs = append(idx.IDs, id)
	return s.writeIndex(idx)
}

func (s *Store) readIndex() (*indexDoc, error) {
	path := filepath.Join(s.Root, indexName)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if err := os.MkdirAll(s.Root, 0o755); err != nil {
				return nil, apperr.Wrap(apperr.IO, "snapshot.index", s.Root, err)
			}
			return &indexDoc{SchemaVersion: SchemaVersion, NextSeq: 1}, nil
		}
		return nil, apperr.Wrap(apperr.IO, "snapshot.index", path, err)
	}
	var idx indexDoc
	if err := json.Unmarshal(data, &idx); err != nil {
		return nil, apperr.Wrap(apperr.Internal, "snapshot.index", path, err)
	}
	if idx.SchemaVersion == 0 {
		idx.SchemaVersion = SchemaVersion
	}
	if idx.NextSeq == 0 {
		idx.NextSeq = 1
	}
	return &idx, nil
}

func (s *Store) writeIndex(idx *indexDoc) error {
	if err := os.MkdirAll(s.Root, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.index", s.Root, err)
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(idx); err != nil {
		return apperr.Wrap(apperr.Internal, "snapshot.index", indexName, err)
	}
	return writeAtomic(filepath.Join(s.Root, indexName), buf.Bytes())
}

func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.write", path, err)
	}
	tmpName := tmp.Name()
	defer func() { _ = os.Remove(tmpName) }()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "snapshot.write", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return apperr.Wrap(apperr.IO, "snapshot.write", path, err)
	}
	if err := tmp.Close(); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.write", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		_ = os.Remove(path)
		if err2 := os.Rename(tmpName, path); err2 != nil {
			return apperr.Wrap(apperr.IO, "snapshot.write", path, err2)
		}
	}
	return nil
}
