package app

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/mewisme/m/internal/apperr"
	"github.com/mewisme/m/internal/config"
	"github.com/mewisme/m/internal/graph"
	"github.com/mewisme/m/internal/transaction"
)

func TestUsesStagedSnapshotInputs(t *testing.T) {
	g := &graph.Graph{}
	if usesStagedSnapshotInputs(InstallOptions{}) {
		t.Fatal("empty opts should not use staged snapshot")
	}
	if usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g}) {
		t.Fatal("graph alone is insufficient")
	}
	if !usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g, StagedManifest: []byte(`{}`)}) {
		t.Fatal("graph + staged manifest should skip live frozen validation")
	}
	if !usesStagedSnapshotInputs(InstallOptions{PreResolvedGraph: g, StagedLock: []byte(`{}`)}) {
		t.Fatal("graph + staged lock should skip live frozen validation")
	}
}

func TestBeginMutationSessionCancelledContext(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"x"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		CWD:    root,
		Config: &config.Effective{},
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := BeginMutationSession(ctx, ac, root)
	if err == nil {
		t.Fatal("expected cancellation error")
	}
	if !isLockWaitCancellation(err) {
		t.Fatalf("expected cancellation, got %v", err)
	}
	if apperr.CodeOf(err) != apperr.Cancelled {
		t.Fatalf("code=%s", apperr.CodeOf(err))
	}
}

func TestBeginMutationSessionReopenProject(t *testing.T) {
	root := t.TempDir()
	manifest := `{"name":"mut-session","version":"1.0.0"}`
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}
	ac := &Context{
		CWD:    root,
		Config: &config.Effective{},
	}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	proj, err := sess.ReopenProject(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if proj.Doc.Name != "mut-session" {
		t.Fatalf("name=%q", proj.Doc.Name)
	}
	if sess.Project() != proj {
		t.Fatal("Project() should return reopened project")
	}
}

func TestBeginMutationSessionRecoversBeforeReopen(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "package.json"), []byte(`{"name":"recover"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	txnRoot := transaction.TxnRoot(root)
	dir := filepath.Join(txnRoot, "stale")
	if err := os.MkdirAll(filepath.Join(dir, "stage"), 0o755); err != nil {
		t.Fatal(err)
	}
	doc := &transaction.Document{
		SchemaVersion: transaction.SchemaVersion,
		ID:            "stale",
		ProjectRoot:   root,
		State:         transaction.StateStaging,
	}
	data, err := transaction.Encode(doc)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "journal.000001.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}

	ac := &Context{CWD: root, Config: &config.Effective{}}
	ctx := context.Background()
	sess, err := BeginMutationSession(ctx, ac, root)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _, _ = sess.Abort(ctx) }()

	txns, err := transaction.ScanIncompleteTxns(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(txns) != 1 || txns[0].ID == "stale" {
		t.Fatalf("expected one new incomplete txn, got %+v", txns)
	}
	if _, err := sess.ReopenProject(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestIsLockWaitCancellation(t *testing.T) {
	if !isLockWaitCancellation(context.Canceled) {
		t.Fatal("context.Canceled")
	}
	if !isLockWaitCancellation(apperr.Wrap(apperr.Cancelled, "x", "", context.DeadlineExceeded)) {
		t.Fatal("wrapped deadline")
	}
	if isLockWaitCancellation(errors.New("other")) {
		t.Fatal("unexpected match")
	}
}
