package lifecycle

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
	"github.com/mewisme/mew/internal/linker"
)

// Discover builds an ordered lifecycle plan from a staged link layout.
func Discover(g *graph.Graph, plan *linker.Plan) (*Plan, error) {
	if g == nil || plan == nil {
		return &Plan{}, nil
	}
	dirs := packageDirs(plan)
	order, err := topoPackageOrder(g, dirs)
	if err != nil {
		return nil, err
	}
	integrity := integrityByKey(g)
	var scripts []Script
	for _, key := range order {
		dir, ok := dirs[key]
		if !ok || dir == "" {
			continue
		}
		name, body, err := readLifecycleScripts(dir)
		if err != nil {
			return nil, err
		}
		if len(body) == 0 {
			continue
		}
		pkgName := name
		if pkgName == "" {
			pkgName = packageNameFromKey(key)
		}
		for _, scriptName := range InstallScriptNames {
			cmd, ok := body[scriptName]
			if !ok || strings.TrimSpace(cmd) == "" {
				continue
			}
			scripts = append(scripts, Script{
				PackageName: pkgName,
				PackageKey:  key,
				PackageDir:  dir,
				Name:        scriptName,
				Command:     cmd,
				Integrity:   integrity[key],
			})
		}
	}
	return &Plan{Scripts: scripts}, nil
}

func packageDirs(plan *linker.Plan) map[string]string {
	out := map[string]string{}
	for _, p := range plan.Placements {
		if p.Key == "" || p.DestDir == "" {
			continue
		}
		out[p.Key] = p.DestDir
	}
	return out
}

func integrityByKey(g *graph.Graph) map[string]string {
	out := map[string]string{}
	for _, p := range g.Packages {
		out[p.ID.Key()] = p.Integrity
	}
	return out
}

func topoPackageOrder(g *graph.Graph, dirs map[string]string) ([]string, error) {
	keys := make([]string, 0, len(dirs))
	for k := range dirs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	present := map[string]bool{}
	for _, k := range keys {
		present[k] = true
	}
	deps := map[string][]string{}
	indeg := map[string]int{}
	for k := range present {
		indeg[k] = 0
	}
	for _, e := range g.Edges {
		if !present[e.From] || !present[e.To] {
			continue
		}
		if e.From == e.To {
			continue
		}
		deps[e.To] = append(deps[e.To], e.From)
		indeg[e.From]++
	}
	var queue []string
	for _, k := range keys {
		if indeg[k] == 0 {
			queue = append(queue, k)
		}
	}
	sort.Strings(queue)
	var order []string
	for len(queue) > 0 {
		k := queue[0]
		queue = queue[1:]
		order = append(order, k)
		for _, dep := range deps[k] {
			indeg[dep]--
			if indeg[dep] == 0 {
				queue = append(queue, dep)
				sort.Strings(queue)
			}
		}
	}
	if len(order) != len(keys) {
		return nil, apperr.New(apperr.Install, "lifecycle.discover", "", "cycle in lifecycle package order")
	}
	return order, nil
}

func readLifecycleScripts(pkgDir string) (name string, scripts map[string]string, err error) {
	raw, err := os.ReadFile(filepath.Join(pkgDir, "package.json"))
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil, nil
		}
		return "", nil, apperr.Wrap(apperr.IO, "lifecycle.discover", pkgDir, err)
	}
	var doc struct {
		Name    string            `json:"name"`
		Scripts map[string]string `json:"scripts"`
	}
	if err := json.Unmarshal(raw, &doc); err != nil {
		return "", nil, apperr.Wrap(apperr.IO, "lifecycle.discover", pkgDir, err)
	}
	return doc.Name, doc.Scripts, nil
}

func packageNameFromKey(key string) string {
	s := key
	if i := strings.IndexByte(s, '#'); i >= 0 {
		s = s[:i]
	}
	if i := strings.LastIndexByte(s, '@'); i > 0 {
		return s[:i]
	}
	return s
}
