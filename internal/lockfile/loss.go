package lockfile

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/mewisme/m/internal/apperr"
)

// Normalize sorts loss items for deterministic encoding.
func (r *LossReport) Normalize() error {
	if r == nil {
		return apperr.New(apperr.Lockfile, "loss.normalize", "loss-report", "nil loss report")
	}
	if r.SchemaVersion == 0 {
		r.SchemaVersion = LossReportSchemaVersion
	}
	if r.SchemaVersion != LossReportSchemaVersion {
		return apperr.New(apperr.Lockfile, "loss.normalize", "loss-report",
			fmt.Sprintf("unsupported schemaVersion %d", r.SchemaVersion))
	}
	if r.Items == nil {
		r.Items = []LossItem{}
	}
	sort.SliceStable(r.Items, func(i, j int) bool {
		a, b := r.Items[i], r.Items[j]
		if a.Field != b.Field {
			return a.Field < b.Field
		}
		if a.SourceFormat != b.SourceFormat {
			return a.SourceFormat < b.SourceFormat
		}
		return a.Reason < b.Reason
	})
	return nil
}

// EncodeLossJSON normalizes and encodes a loss report.
func EncodeLossJSON(r *LossReport) ([]byte, error) {
	if err := r.Normalize(); err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "loss.encode", "loss-report", err)
	}
	return buf.Bytes(), nil
}

// DecodeLossJSON unmarshals and normalizes a loss report.
func DecodeLossJSON(data []byte) (*LossReport, error) {
	var r LossReport
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, apperr.Wrap(apperr.Lockfile, "loss.decode", "loss-report", err)
	}
	if err := r.Normalize(); err != nil {
		return nil, err
	}
	return &r, nil
}
