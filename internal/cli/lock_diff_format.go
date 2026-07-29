package cli

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/lockfile"
)

func runLockDiff(ctx context.Context, w io.Writer, ac *app.Context, opts app.LockDiffOptions, asJSON bool) error {
	diff, err := app.LockDiff(ctx, ac, opts)
	if err != nil {
		return err
	}
	if asJSON {
		data, err := lockfile.EncodeDiffJSON(diff)
		if err != nil {
			return err
		}
		_, err = w.Write(data)
		return err
	}
	return formatLockDiffHuman(w, diff)
}

// FormatLockDiffHuman writes a human-readable lock diff summary.
func FormatLockDiffHuman(w io.Writer, diff *lockfile.GraphDiff) error {
	return formatLockDiffHuman(w, diff)
}

func formatLockDiffHuman(w io.Writer, diff *lockfile.GraphDiff) error {
	if diff == nil {
		return nil
	}
	empty := len(diff.PackagesAdded) == 0 && len(diff.PackagesRemoved) == 0 &&
		len(diff.EdgesAdded) == 0 && len(diff.EdgesRemoved) == 0 && len(diff.Specifiers) == 0
	if empty {
		_, err := fmt.Fprintln(w, "no changes")
		return err
	}
	for _, pkg := range diff.PackagesAdded {
		if _, err := fmt.Fprintf(w, "+%s\n", pkg); err != nil {
			return err
		}
	}
	for _, pkg := range diff.PackagesRemoved {
		if _, err := fmt.Fprintf(w, "-%s\n", pkg); err != nil {
			return err
		}
	}
	for _, sp := range diff.Specifiers {
		line := formatSpecifierDiff(sp)
		if _, err := fmt.Fprintln(w, line); err != nil {
			return err
		}
	}
	return nil
}

func formatSpecifierDiff(sp lockfile.ImporterSpecifierDiff) string {
	importer := sp.Importer
	if importer == "" {
		importer = "."
	}
	kind := sp.Kind
	if kind != "" {
		kind = " " + kind
	}
	name := sp.Name
	switch {
	case sp.Before == "" && sp.After != "":
		return fmt.Sprintf("~ %s %s%s: +%s", importer, name, kind, sp.After)
	case sp.Before != "" && sp.After == "":
		return fmt.Sprintf("~ %s %s%s: -%s", importer, name, kind, sp.Before)
	default:
		return fmt.Sprintf("~ %s %s%s: %s -> %s", importer, name, kind, sp.Before, sp.After)
	}
}

func shortDigest(digest string) string {
	const prefix = "sha256:"
	if strings.HasPrefix(digest, prefix) && len(digest) > len(prefix)+12 {
		return digest[:len(prefix)+12] + "…"
	}
	if len(digest) > 16 {
		return digest[:16] + "…"
	}
	return digest
}
