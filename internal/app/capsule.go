package app

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/capsule"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/fetch"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/lockfile/mlock"
	"github.com/mewisme/mew/internal/project"
	"github.com/mewisme/mew/internal/snapshot"
	"github.com/mewisme/mew/internal/store"
)

// CapsuleCreateOptions configures m capsule create.
type CapsuleCreateOptions struct {
	ProjectDir string
	OutputPath string
}

// CapsuleCreateResult summarizes a created capsule archive.
type CapsuleCreateResult struct {
	ArchivePath string `json:"archive"`
	ID          string `json:"id"`
}

// CapsuleRestoreOptions configures m capsule restore.
type CapsuleRestoreOptions struct {
	ArchivePath string
	ProjectDir  string
}

// CreateCapsule bundles lock, manifests, and cached blobs into a portable archive.
func CreateCapsule(ctx context.Context, ac *Context, opts CapsuleCreateOptions) (CapsuleCreateResult, error) {
	var out CapsuleCreateResult
	if ac == nil || ac.Config == nil {
		return out, apperr.New(apperr.Internal, "app.capsule.create", "", "missing app context")
	}
	root, err := resolveProjectRoot(ac, opts.ProjectDir)
	if err != nil {
		return out, err
	}
	proj, err := project.Open(ctx, root)
	if err != nil {
		return out, err
	}
	lockPath := LockPath(proj)
	lockBytes, err := os.ReadFile(lockPath)
	if err != nil {
		return out, apperr.Wrap(apperr.IO, "app.capsule.create", lockPath, err)
	}
	manifestPath := filepath.Join(root, "package.json")
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		return out, apperr.Wrap(apperr.IO, "app.capsule.create", manifestPath, err)
	}
	g, err := readLockHints(ctx, ac, proj)
	if err != nil {
		return out, err
	}
	if g == nil {
		return out, apperr.New(apperr.Lockfile, "app.capsule.create", lockPath, "no lockfile graph")
	}
	graphDigest, err := snapshot.GraphDigest(g)
	if err != nil {
		return out, err
	}
	blobs, err := collectCapsuleBlobRefs(g)
	if err != nil {
		return out, err
	}
	blobStore := store.NewDir(config.BlobCacheDir(ac.Config))
	for _, ref := range blobs {
		if !blobStore.Exists(store.Key(ref.Algo + "/" + ref.Hex)) {
			return out, apperr.New(apperr.NotFound, "app.capsule.create", ref.BlobPath(),
				"blob missing from cache; run install first")
		}
	}
	members, err := collectCapsuleMemberManifests(root, lockBytes)
	if err != nil {
		return out, err
	}
	platform := runtime.GOOS + "/" + runtime.GOARCH
	man := &capsule.Manifest{
		SchemaVersion:   capsule.SchemaVersion,
		ID:              capsule.ComputeID(lockBytes, platform),
		CreatedAt:       time.Now().UTC(),
		GraphDigest:     graphDigest,
		PolicyDigest:    policyDigestFromLock(lockBytes),
		Platform:        platform,
		NodeVersion:     strings.TrimSpace(os.Getenv("NODE_VERSION")),
		Lock:            lockBytes,
		Manifest:        manifestBytes,
		MemberManifests: members,
		Blobs:           blobs,
	}
	output := opts.OutputPath
	if output == "" {
		output = filepath.Join(ac.CWD, "mew.capsule")
	}
	output, err = filepath.Abs(output)
	if err != nil {
		return out, apperr.Wrap(apperr.IO, "app.capsule.create", output, err)
	}
	openBlob := func(ctx context.Context, ref capsule.BlobRef) (io.ReadCloser, error) {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		path := blobStore.BlobPath(store.Key(ref.Algo + "/" + ref.Hex))
		return os.Open(path)
	}
	if err := capsule.Create(ctx, capsule.CreateOptions{
		OutputPath: output,
		Manifest:   man,
		OpenBlob:   openBlob,
	}); err != nil {
		return out, err
	}
	out.ArchivePath = output
	out.ID = man.ID
	return out, nil
}

// RestoreCapsule imports blobs and runs a frozen install from capsule metadata.
func RestoreCapsule(ctx context.Context, ac *Context, opts CapsuleRestoreOptions) (InstallResult, error) {
	var res InstallResult
	if ac == nil || ac.Config == nil {
		return res, apperr.New(apperr.Internal, "app.capsule.restore", "", "missing app context")
	}
	if strings.TrimSpace(opts.ArchivePath) == "" {
		return res, apperr.New(apperr.Usage, "app.capsule.restore", "", "missing archive path")
	}
	archivePath, err := filepath.Abs(opts.ArchivePath)
	if err != nil {
		return res, apperr.Wrap(apperr.IO, "app.capsule.restore", opts.ArchivePath, err)
	}
	root, err := resolveProjectRoot(ac, opts.ProjectDir)
	if err != nil {
		return res, err
	}
	blobStore := store.NewDir(config.BlobCacheDir(ac.Config))
	man, err := capsule.Restore(ctx, capsule.RestoreOptions{
		ArchivePath: archivePath,
		WriteBlob: func(ctx context.Context, ref capsule.BlobRef, r io.Reader) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			data, err := io.ReadAll(io.LimitReader(r, 512<<20))
			if err != nil {
				return apperr.Wrap(apperr.IO, "app.capsule.restore", ref.BlobPath(), err)
			}
			return blobStore.Put(ctx, store.Key(ref.Algo+"/"+ref.Hex), data)
		},
	})
	if err != nil {
		return res, err
	}
	rec := snapshot.Record{
		Meta: &snapshot.Snapshot{
			SchemaVersion:   snapshot.SchemaVersion,
			ID:              man.ID,
			CreatedAt:       man.CreatedAt,
			GraphDigest:     man.GraphDigest,
			PolicyDigest:    man.PolicyDigest,
			MemberManifests: memberManifestPathsFromMap(man.MemberManifests),
		},
		Manifest:        man.Manifest,
		Lock:            man.Lock,
		MemberManifests: man.MemberManifests,
	}
	g, manifestBytes, err := snapshot.ValidateRestorePair(rec)
	if err != nil {
		return res, err
	}
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		return res, err
	}
	res, err = runInstallInSession(ctx, sess, InstallOptions{
		Frozen:                true,
		WriteManifest:         true,
		PreResolvedGraph:      g,
		StagedManifest:        manifestBytes,
		StagedLock:            man.Lock,
		StagedMemberManifests: man.MemberManifests,
		SkipSnapshot:          true,
	}, nil, nil)
	if err != nil {
		abortRes, abortErr := abortMutation(ctx, sess, sess.Runner(), err)
		res = mergeInstallResults(res, abortRes)
		return res, abortErr
	}
	finish, finishErr := sess.Finish(ctx, false)
	if finish.Committed {
		res.Committed = true
	}
	if finish.HasCriticalCleanupFailure() {
		populateCleanupResult(&res, finish)
		return res, finishErr
	}
	populateWarningCleanup(&res, finish)
	return res, finishErr
}

// FormatCapsuleCreateLine returns human-readable capsule create output.
func FormatCapsuleCreateLine(r CapsuleCreateResult) string {
	return capsule.FormatCreateLine(r.ArchivePath, r.ID)
}

func collectCapsuleBlobRefs(g *graph.Graph) ([]capsule.BlobRef, error) {
	if g == nil {
		return nil, apperr.New(apperr.Internal, "app.capsule", "graph", "nil graph")
	}
	seen := map[string]struct{}{}
	var refs []capsule.BlobRef
	for _, pkg := range g.Packages {
		if strings.TrimSpace(pkg.Integrity) == "" {
			continue
		}
		parsed, err := fetch.ParseIntegrity(pkg.Integrity)
		if err != nil {
			return nil, err
		}
		ref := capsule.BlobRef{Algo: parsed.Algo, Hex: parsed.Hex}
		key := ref.Algo + "/" + ref.Hex
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		refs = append(refs, ref)
	}
	sort.Slice(refs, func(i, j int) bool {
		if refs[i].Algo != refs[j].Algo {
			return refs[i].Algo < refs[j].Algo
		}
		return refs[i].Hex < refs[j].Hex
	})
	return refs, nil
}

func collectCapsuleMemberManifests(projRoot string, lockBytes []byte) (map[string][]byte, error) {
	doc, err := mlock.Decode(lockBytes)
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{}
	for _, im := range doc.Importers {
		if im.ID == graph.RootImporter {
			continue
		}
		rel := filepath.ToSlash(filepath.Join(string(im.ID), "package.json"))
		path := filepath.Join(projRoot, filepath.FromSlash(rel))
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.IO, "app.capsule.create", rel, err)
		}
		out[rel] = data
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}

func policyDigestFromLock(lockBytes []byte) string {
	doc, err := mlock.Decode(lockBytes)
	if err != nil {
		return ""
	}
	parts := []string{
		doc.Settings.ResolverPolicyFingerprint,
		doc.Settings.TargetPlatformFingerprint,
	}
	parts = append(parts, doc.Settings.OverridesFingerprint)
	return strings.Join(parts, "|")
}

func memberManifestPathsFromMap(members map[string][]byte) []string {
	if len(members) == 0 {
		return nil
	}
	out := make([]string, 0, len(members))
	for rel := range members {
		out = append(out, rel)
	}
	sort.Strings(out)
	return out
}
