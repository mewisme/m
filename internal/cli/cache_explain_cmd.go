package cli

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mewisme/mew/internal/app"
	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/config"
	"github.com/mewisme/mew/internal/presentation"
	"github.com/mewisme/mew/internal/transform"
)

func newCacheExplainCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "explain",
		Short: "Show transform cache status and statistics",
		Long:  "Displays the transform cache directory, entry count, total disk usage, and schema version.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			ac := app.FromContext(cmd.Context())
			if ac == nil {
				return apperr.New(apperr.Internal, "cache.explain", "", "missing app context")
			}
			info, err := gatherCacheInfo(ac.Config)
			if err != nil {
				return err
			}
			if asJSON {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetEscapeHTML(false)
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}
			g := ownerFlags(cmd.Root())
			r := g.mustStaticRenderer(cmd)
			if err := writeStaticOut(cmd, r.Summary(cacheExplainSummary(info))); err != nil {
				return err
			}
			if table := r.Table(cacheExplainTable(info)); table != "" {
				if err := writeStaticOut(cmd, table); err != nil {
					return err
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "print cache info as JSON")
	return cmd
}

type cacheExplainInfo struct {
	CacheDir   string `json:"cacheDir"`
	SchemaVer  int    `json:"schemaVersion"`
	EntryCount int    `json:"entryCount"`
	CodeBytes  int64  `json:"codeBytes"`
	MapBytes   int64  `json:"mapBytes"`
	MetaBytes  int64  `json:"metaBytes"`
	TotalBytes int64  `json:"totalBytes"`
}

func gatherCacheInfo(eff *config.Effective) (*cacheExplainInfo, error) {
	cacheDir := transform.TransformCacheDir(eff)
	info := &cacheExplainInfo{
		CacheDir:  cacheDir,
		SchemaVer: transform.CacheSchemaVersion,
	}

	entries, err := os.ReadDir(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			return info, nil
		}
		return nil, err
	}

	// Cache layout: <cacheDir>/<prefix2>/<sha64>/{.code, .map, .meta}
	for _, prefixEntry := range entries {
		if !prefixEntry.IsDir() || len(prefixEntry.Name()) != 2 {
			continue
		}
		prefixDir := filepath.Join(cacheDir, prefixEntry.Name())
		keyEntries, err := os.ReadDir(prefixDir)
		if err != nil {
			continue
		}
		for _, keyEntry := range keyEntries {
			if keyEntry.IsDir() {
				continue
			}
			if filepath.Ext(keyEntry.Name()) != ".meta" {
				continue
			}
			baseName := keyEntry.Name()[:len(keyEntry.Name())-5] // strip ".meta"
			metaPath := filepath.Join(prefixDir, keyEntry.Name())
			codePath := filepath.Join(prefixDir, baseName+".code")
			mapPath := filepath.Join(prefixDir, baseName+".map")

			if st, err := os.Stat(metaPath); err == nil {
				info.MetaBytes += st.Size()
			}
			if st, err := os.Stat(codePath); err == nil {
				info.CodeBytes += st.Size()
			}
			if st, err := os.Stat(mapPath); err == nil {
				info.MapBytes += st.Size()
			}
			info.EntryCount++
		}
	}

	info.TotalBytes = info.CodeBytes + info.MapBytes + info.MetaBytes
	return info, nil
}

func cacheExplainSummary(info *cacheExplainInfo) presentation.Summary {
	if info == nil {
		return presentation.Summary{Status: presentation.StatusError, Title: "Failed to gather cache info"}
	}
	st := presentation.StatusSuccess
	title := "Transform cache active"
	if info.EntryCount == 0 {
		title = "Transform cache empty"
	}
	return presentation.Summary{
		Status: st,
		Title:  title,
		Metrics: []presentation.KeyValue{
			{Key: "directory", Value: info.CacheDir, Style: presentation.ValuePath},
			{Key: "entries", Value: fmt.Sprintf("%d", info.EntryCount), Style: presentation.ValueNumber},
			{Key: "total size", Value: formatBytes(info.TotalBytes), Style: presentation.ValueNumber},
		},
	}
}

func cacheExplainTable(info *cacheExplainInfo) presentation.TableModel {
	if info == nil {
		return presentation.TableModel{}
	}
	cols := []presentation.TableColumn{
		{Key: "metric", Header: "METRIC", MinWidth: 8, Prefer: 16, Primary: true},
		{Key: "value", Header: "VALUE", MinWidth: 8, Prefer: 40},
	}
	rows := []map[string]string{
		{"metric": "Cache directory", "value": info.CacheDir},
		{"metric": "Schema version", "value": fmt.Sprintf("v%d", info.SchemaVer)},
		{"metric": "Entries", "value": fmt.Sprintf("%d", info.EntryCount)},
		{"metric": "Code bytes", "value": formatBytes(info.CodeBytes)},
		{"metric": "Map bytes", "value": formatBytes(info.MapBytes)},
		{"metric": "Meta bytes", "value": formatBytes(info.MetaBytes)},
		{"metric": "Total", "value": formatBytes(info.TotalBytes)},
	}
	return presentation.TableModel{Columns: cols, Rows: rows}
}

func formatBytes(n int64) string {
	switch {
	case n >= 1<<30:
		return fmt.Sprintf("%.1f GiB", float64(n)/(1<<30))
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.1f KiB", float64(n)/(1<<10))
	default:
		return fmt.Sprintf("%d B", n)
	}
}
