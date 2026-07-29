package policy

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

const orgPolicySchemaVersion = 1

// SeverityLevel classifies org-policy violation impact.
type SeverityLevel string

const (
	SeverityWarn  SeverityLevel = "warn"
	SeverityError SeverityLevel = "error"
)

// Waiver exempts a package from org-policy rules until expiry.
type Waiver struct {
	Package string `json:"package"`
	Reason  string `json:"reason"`
	Expires string `json:"expires,omitempty"` // RFC3339
}

// OrgPolicy is project-level supply-chain policy (mew.policy.json).
type OrgPolicy struct {
	SchemaVersion     int           `json:"schemaVersion,omitempty"`
	DenyPackages      []string      `json:"denyPackages,omitempty"`
	DenyLicenses      []string      `json:"denyLicenses,omitempty"`
	SeverityThreshold SeverityLevel `json:"severityThreshold,omitempty"`
	Waivers           []Waiver      `json:"waivers,omitempty"`
}

// PolicyViolation is one org-policy rule breach.
type PolicyViolation struct {
	Package  string        `json:"package"`
	Rule     string        `json:"rule"`
	Severity SeverityLevel `json:"severity"`
	Message  string        `json:"message"`
}

// PolicyResult is the outcome of org-policy evaluation.
type PolicyResult struct {
	Passed     bool              `json:"passed"`
	Violations []PolicyViolation `json:"violations,omitempty"`
}

// LoadOrgPolicy reads mew.policy.json then .mew/policy.json under projectRoot.
// Missing files return (nil, nil).
func LoadOrgPolicy(projectRoot string) (*OrgPolicy, error) {
	candidates := []string{
		filepath.Join(projectRoot, "mew.policy.json"),
		filepath.Join(projectRoot, ".mew", "policy.json"),
	}
	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, apperr.Wrap(apperr.Config, "policy.load", path, err)
		}
		var p OrgPolicy
		if err := json.Unmarshal(data, &p); err != nil {
			return nil, apperr.Wrap(apperr.Config, "policy.load", path, err)
		}
		if err := p.Normalize(); err != nil {
			return nil, err
		}
		return &p, nil
	}
	return nil, nil
}

// Normalize validates org-policy fields and fills defaults.
func (p *OrgPolicy) Normalize() error {
	if p == nil {
		return apperr.New(apperr.Config, "policy.normalize", "orgPolicy", "nil policy")
	}
	if p.SchemaVersion == 0 {
		p.SchemaVersion = orgPolicySchemaVersion
	}
	if p.SchemaVersion != orgPolicySchemaVersion {
		return apperr.New(apperr.Config, "policy.normalize", "orgPolicy",
			fmt.Sprintf("unsupported schemaVersion %d", p.SchemaVersion))
	}
	if p.SeverityThreshold == "" {
		p.SeverityThreshold = SeverityError
	}
	switch p.SeverityThreshold {
	case SeverityWarn, SeverityError:
	default:
		return apperr.New(apperr.Config, "policy.normalize", "severityThreshold",
			fmt.Sprintf("unknown severityThreshold %q", p.SeverityThreshold))
	}
	return nil
}

// Evaluate checks graph packages and installedLicenses (package key → license) against org rules.
func Evaluate(g *graph.Graph, installedLicenses map[string]string, org *OrgPolicy) PolicyResult {
	if org == nil || g == nil {
		return PolicyResult{Passed: true}
	}
	now := time.Now()
	var violations []PolicyViolation
	severity := org.SeverityThreshold

	denyPkg := make(map[string]struct{}, len(org.DenyPackages))
	for _, name := range org.DenyPackages {
		denyPkg[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
	}
	denyLic := make([]string, len(org.DenyLicenses))
	for i, lic := range org.DenyLicenses {
		denyLic[i] = strings.ToUpper(strings.TrimSpace(lic))
	}

	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		name := pkg.ID.Name
		if waived(org.Waivers, name, key, now) {
			continue
		}
		if _, denied := denyPkg[strings.ToLower(name)]; denied {
			violations = append(violations, PolicyViolation{
				Package:  key,
				Rule:     "denied_package",
				Severity: severity,
				Message:  fmt.Sprintf("package %q denied by org policy", name),
			})
			continue
		}
		license := strings.TrimSpace(installedLicenses[key])
		if license == "" {
			continue
		}
		for _, denied := range denyLic {
			if denied == "" {
				continue
			}
			if licenseMatchesDenied(license, denied) {
				violations = append(violations, PolicyViolation{
					Package:  key,
					Rule:     "denied_license",
					Severity: severity,
					Message:  fmt.Sprintf("license %q denied by org policy", license),
				})
				break
			}
		}
	}

	sort.Slice(violations, func(i, j int) bool {
		if violations[i].Package != violations[j].Package {
			return violations[i].Package < violations[j].Package
		}
		return violations[i].Rule < violations[j].Rule
	})

	passed := true
	for _, v := range violations {
		if v.Severity == SeverityError {
			passed = false
			break
		}
	}
	return PolicyResult{Passed: passed, Violations: violations}
}

func licenseMatchesDenied(license, denied string) bool {
	license = strings.ToUpper(strings.TrimSpace(license))
	denied = strings.ToUpper(strings.TrimSpace(denied))
	if license == denied {
		return true
	}
	return strings.Contains(license, denied)
}

func waived(waivers []Waiver, name, key string, now time.Time) bool {
	nameLower := strings.ToLower(name)
	keyLower := strings.ToLower(key)
	for _, w := range waivers {
		target := strings.ToLower(strings.TrimSpace(w.Package))
		if target == "" {
			continue
		}
		if target != nameLower && target != keyLower {
			continue
		}
		if w.Expires != "" {
			exp, err := time.Parse(time.RFC3339, w.Expires)
			if err == nil && now.After(exp) {
				continue
			}
		}
		return true
	}
	return false
}

// LicensesFromExtractDirs reads license fields from staged extract directories.
func LicensesFromExtractDirs(extractDirs map[string]string) map[string]string {
	out := make(map[string]string, len(extractDirs))
	for key, dir := range extractDirs {
		if lic := readPackageLicense(dir); lic != "" {
			out[key] = lic
		}
	}
	return out
}

// LicensesFromNodeModules reads license fields for graph packages under nmRoot.
func LicensesFromNodeModules(nmRoot string, g *graph.Graph) map[string]string {
	if g == nil || nmRoot == "" {
		return map[string]string{}
	}
	out := make(map[string]string, len(g.Packages))
	for _, pkg := range g.Packages {
		key := pkg.ID.Key()
		manifestPath := packageManifestPath(nmRoot, pkg.ID.Name)
		if lic := readPackageLicense(filepath.Dir(manifestPath)); lic != "" {
			out[key] = lic
		}
	}
	return out
}

func packageManifestPath(nmRoot, name string) string {
	parts := append([]string{nmRoot}, strings.Split(name, "/")...)
	return filepath.Join(append(parts, "package.json")...)
}

func readPackageLicense(pkgDir string) string {
	data, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		return ""
	}
	var doc struct {
		License any `json:"license"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return ""
	}
	switch v := doc.License.(type) {
	case string:
		return strings.TrimSpace(v)
	case map[string]any:
		if t, ok := v["type"].(string); ok {
			return strings.TrimSpace(t)
		}
	}
	return ""
}
