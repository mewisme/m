package conformance

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
)

// WaiverRecord is one structured waiver entry.
type WaiverRecord struct {
	ID                  string   `json:"id"`
	SuiteID             string   `json:"suiteId"`
	Platforms           []string `json:"platforms"`
	Reason              string   `json:"reason"`
	OpenedDate          string   `json:"openedDate"`
	ReviewDate          string   `json:"reviewDate"`
	ExpiryDate          string   `json:"expiryDate"`
	AllowPassWithWaiver bool     `json:"allowPassWithWaiver"`
	FollowUpMVP         string   `json:"followUpMVP,omitempty"`
	Owner               string   `json:"owner"`
}

// WaiverManifest is the machine-readable waiver file beside the runner manifest.
type WaiverManifest struct {
	SchemaVersion int            `json:"schemaVersion"`
	Matrix        string         `json:"matrix"`
	Waivers       []WaiverRecord `json:"waivers"`
}

// LoadWaiverManifest reads and validates waivers.v1.json.
func LoadWaiverManifest(path string) (WaiverManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return WaiverManifest{SchemaVersion: WaiverSchemaVersion, Matrix: RunnerMatrix, Waivers: nil}, nil
		}
		return WaiverManifest{}, apperr.Wrap(apperr.IO, "conformance.runner.waiver", path, err)
	}
	var w WaiverManifest
	if err := json.Unmarshal(data, &w); err != nil {
		return WaiverManifest{}, apperr.Wrap(apperr.Manifest, "conformance.runner.waiver", path, err)
	}
	if w.SchemaVersion != WaiverSchemaVersion {
		return WaiverManifest{}, apperr.New(apperr.Unsupported, "conformance.runner.waiver", "", fmt.Sprintf("unsupported schema %d", w.SchemaVersion))
	}
	if w.Matrix != RunnerMatrix {
		return WaiverManifest{}, apperr.New(apperr.Manifest, "conformance.runner.waiver", "", fmt.Sprintf("unsupported matrix %q", w.Matrix))
	}
	if err := validateWaiverManifest(&w); err != nil {
		return WaiverManifest{}, err
	}
	return w, nil
}

func validateWaiverManifest(w *WaiverManifest) error {
	seen := map[string]struct{}{}
	for i := range w.Waivers {
		rec := &w.Waivers[i]
		if strings.TrimSpace(rec.ID) == "" {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", "", "waiver missing id")
		}
		if _, ok := seen[rec.ID]; ok {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", rec.ID, "duplicate waiver id")
		}
		seen[rec.ID] = struct{}{}
		if strings.TrimSpace(rec.SuiteID) == "" {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", rec.ID, "missing suiteId")
		}
		if !rec.AllowPassWithWaiver {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", rec.ID, "allowPassWithWaiver must be true")
		}
		if err := validateWaiverDates(rec); err != nil {
			return apperr.Wrap(apperr.Manifest, "conformance.runner.waiver", rec.ID, err)
		}
	}
	sortWaiverManifestForDigest(w)
	return nil
}

func validateWaiverDates(rec *WaiverRecord) error {
	opened, err := time.Parse("2006-01-02", rec.OpenedDate)
	if err != nil {
		return fmt.Errorf("invalid openedDate")
	}
	review, err := time.Parse("2006-01-02", rec.ReviewDate)
	if err != nil {
		return fmt.Errorf("invalid reviewDate")
	}
	expiry, err := time.Parse("2006-01-02", rec.ExpiryDate)
	if err != nil {
		return fmt.Errorf("invalid expiryDate")
	}
	if opened.After(review) || review.After(expiry) {
		return fmt.Errorf("openedDate <= reviewDate <= expiryDate required")
	}
	return nil
}

func sortWaiverManifestForDigest(w *WaiverManifest) {
	sort.Slice(w.Waivers, func(i, j int) bool {
		return w.Waivers[i].ID < w.Waivers[j].ID
	})
	for i := range w.Waivers {
		sort.Strings(w.Waivers[i].Platforms)
	}
}

func validateWaiverReferences(manifest RunnerManifest, waivers WaiverManifest) error {
	suiteByID := map[string]RunnerSuite{}
	for _, s := range manifest.Suites {
		suiteByID[s.ID] = s
	}
	waiverByID := map[string]WaiverRecord{}
	for _, w := range waivers.Waivers {
		waiverByID[w.ID] = w
	}
	for _, s := range manifest.Suites {
		for _, wid := range s.WaiverIDs {
			w, ok := waiverByID[wid]
			if !ok {
				return apperr.New(apperr.Manifest, "conformance.runner.waiver", wid, "suite references unknown waiver")
			}
			if w.SuiteID != s.ID {
				return apperr.New(apperr.Manifest, "conformance.runner.waiver", wid, "waiver suiteId mismatch")
			}
			if s.WaiverPolicy != "allowed" {
				return apperr.New(apperr.Manifest, "conformance.runner.suite", s.ID, "waiver on forbidden-policy suite")
			}
		}
	}
	for _, w := range waivers.Waivers {
		s, ok := suiteByID[w.SuiteID]
		if !ok {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", w.ID, "orphan waiver suite")
		}
		found := false
		for _, wid := range s.WaiverIDs {
			if wid == w.ID {
				found = true
				break
			}
		}
		if !found {
			return apperr.New(apperr.Manifest, "conformance.runner.waiver", w.ID, "suite does not reference waiver")
		}
		for _, p := range w.Platforms {
			if !runnerSuiteSupportedOnPlatform(s, platformToGOOS(p)) {
				return apperr.New(apperr.Manifest, "conformance.runner.waiver", w.ID, fmt.Sprintf("platform %q not in suite platforms", p))
			}
		}
	}
	return nil
}

func activeWaiversForSuite(waivers WaiverManifest, suite RunnerSuite, goos string) []string {
	now := time.Now().UTC()
	platform := goosToPlatform(goos)
	var out []string
	for _, w := range waivers.Waivers {
		if w.SuiteID != suite.ID {
			continue
		}
		if len(w.Platforms) > 0 {
			ok := false
			for _, p := range w.Platforms {
				if p == platform {
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		expiry, err := time.Parse("2006-01-02", w.ExpiryDate)
		if err != nil || now.After(expiry.Add(24*time.Hour)) {
			continue
		}
		out = append(out, w.ID)
	}
	sort.Strings(out)
	return out
}
