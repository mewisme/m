package snapshot

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/fsx"
	"github.com/mewisme/mew/internal/graph"
)

const (
	indexName         = "index.json"
	manifestFileName  = "package.json"
	lockFileName      = "m.lock"
	metaFileName      = "meta.json"
	memberManifestDir = "manifests"
)

// Record is a loaded snapshot with manifest and lock bytes.
type Record struct {
	Meta            *Snapshot
	Manifest        []byte
	Lock            []byte
	MemberManifests map[string][]byte
}

type indexDoc struct {
	SchemaVersion int      `json:"schemaVersion"`
	NextSeq       int      `json:"nextSeq"`
	IDs           []string `json:"ids"`
}

// Store persists snapshots under <project>/.mew/snapshots/.
type Store struct {
	Root        string
	projectRoot string
}

// NewStore returns a snapshot store for projectRoot.
func NewStore(projectRoot string) *Store {
	return &Store{Root: filepath.Join(projectRoot, ".mew", "snapshots"), projectRoot: projectRoot}
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
func (s *Store) StageCreate(stageRoot, id string, manifest, lock []byte, graphDigest string, members map[string][]byte) error {
	if s == nil || stageRoot == "" {
		return apperr.New(apperr.Internal, "snapshot.stage", id, "nil store or stage root")
	}
	if err := s.guardPaths(filepath.Join(".mew", "snapshots", id)); err != nil {
		return err
	}
	if id == "" {
		return apperr.New(apperr.Internal, "snapshot.stage", "", "empty id")
	}
	dir := filepath.Join(stageRoot, id)
	return s.writeSnapshotPayload(dir, id, manifest, lock, graphDigest, members)
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
func (s *Store) Create(id string, manifest, lock []byte, graphDigest string, members map[string][]byte) error {
	if s == nil || s.Root == "" {
		return apperr.New(apperr.Internal, "snapshot.create", "", "nil store")
	}
	if err := s.guardPaths(filepath.Join(".mew", "snapshots", id)); err != nil {
		return err
	}
	if id == "" {
		return apperr.New(apperr.Internal, "snapshot.create", "", "empty id")
	}
	dir := filepath.Join(s.Root, id)
	if err := s.writeSnapshotPayload(dir, id, manifest, lock, graphDigest, members); err != nil {
		return err
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
	if err := s.guardPaths(filepath.Join(".mew", "snapshots", id)); err != nil {
		return nil, err
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
	members, err := s.loadMemberManifests(dir, meta)
	if err != nil {
		return nil, err
	}
	return &Record{Meta: meta, Manifest: manifest, Lock: lock, MemberManifests: members}, nil
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
	if err := fsx.PublishFileDurable(path, data, 0o644); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.write", path, err)
	}
	return nil
}

func (s *Store) guardPaths(rel string) error {
	if s == nil || s.projectRoot == "" {
		return nil
	}
	absRoot, err := filepath.Abs(s.projectRoot)
	if err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.guard", s.projectRoot, err)
	}
	target, err := filepath.Abs(filepath.Join(absRoot, rel))
	if err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.guard", rel, err)
	}
	return fsx.GuardAncestors(absRoot, target)
}

func (s *Store) guardMemberManifest(rel string) error {
	rel = filepath.ToSlash(rel)
	if filepath.IsAbs(rel) || strings.Contains(rel, "..") {
		return apperr.New(apperr.IO, "snapshot.guard", rel, "invalid member manifest path")
	}
	return s.guardPaths(rel)
}

func (s *Store) writeSnapshotPayload(dir, id string, manifest, lock []byte, graphDigest string, members map[string][]byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "snapshot.write", dir, err)
	}
	meta := &Snapshot{
		SchemaVersion: SchemaVersion,
		ID:            id,
		CreatedAt:     time.Now().UTC(),
		GraphDigest:   graphDigest,
	}
	if len(members) > 0 {
		meta.MemberManifests = memberManifestPaths(members)
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
	return s.writeMemberManifests(dir, members)
}

func (s *Store) writeMemberManifests(dir string, members map[string][]byte) error {
	if len(members) == 0 {
		return nil
	}
	for rel, data := range members {
		if err := s.guardMemberManifest(rel); err != nil {
			return err
		}
		dest := filepath.Join(dir, memberManifestDir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return apperr.Wrap(apperr.IO, "snapshot.write", dest, err)
		}
		if err := writeAtomic(dest, data); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadMemberManifests(dir string, meta *Snapshot) (map[string][]byte, error) {
	if meta == nil || len(meta.MemberManifests) == 0 {
		return nil, nil
	}
	out := make(map[string][]byte, len(meta.MemberManifests))
	for _, rel := range meta.MemberManifests {
		rel = filepath.ToSlash(rel)
		if err := s.guardMemberManifest(rel); err != nil {
			return nil, err
		}
		path := filepath.Join(dir, memberManifestDir, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, apperr.Wrap(apperr.IO, "snapshot.load", rel, err)
		}
		out[rel] = data
	}
	return out, nil
}

func memberManifestPaths(members map[string][]byte) []string {
	if len(members) == 0 {
		return nil
	}
	out := make([]string, 0, len(members))
	for rel := range members {
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}
