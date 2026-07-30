package cli

import (
	"sort"
	"strings"
)

// DispatchKind classifies a dispatch resolution or suggestion target.
type DispatchKind string

const (
	DispatchBuiltin DispatchKind = "builtin"
	DispatchAlias   DispatchKind = "alias"
	DispatchScript  DispatchKind = "script"
)

// Suggestion is a ranked dispatch hint for unknown selectors.
type Suggestion struct {
	Name       string
	Kind       DispatchKind
	Invocation string
	Distance   int
}

// maxDistanceFor returns the Levenshtein threshold for a query of the given length.
// Shorter queries tolerate fewer edits to avoid noisy matches.
func maxDistanceFor(n int) int {
	switch {
	case n <= 3:
		return 1
	case n <= 6:
		return 2
	default:
		return 3
	}
}

func levenshtein(a, b string) int {
	a = strings.ToLower(a)
	b = strings.ToLower(b)
	if a == b {
		return 0
	}
	la, lb := len(a), len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	prev := make([]int, lb+1)
	cur := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		cur[0] = i
		for j := 1; j <= lb; j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			del := prev[j] + 1
			ins := cur[j-1] + 1
			sub := prev[j-1] + cost
			cur[j] = min(del, min(ins, sub))
		}
		prev, cur = cur, prev
	}
	return prev[lb]
}

func suggestFromNames(query string, names []string, kind DispatchKind, format func(string) string) []Suggestion {
	if query == "" || len(names) == 0 {
		return nil
	}
	maxDist := maxDistanceFor(len(query))
	var out []Suggestion
	for _, name := range names {
		d := levenshtein(query, name)
		if d > maxDist {
			continue
		}
		out = append(out, Suggestion{
			Name:       name,
			Kind:       kind,
			Invocation: format(name),
			Distance:   d,
		})
	}
	return out
}

func kindRank(k DispatchKind) int {
	switch k {
	case DispatchBuiltin:
		return 0
	case DispatchAlias:
		return 1
	case DispatchScript:
		return 2
	default:
		return 3
	}
}

// mergeSuggestions combines candidate lists, deduplicates by Invocation, ranks, and
// returns at most maxDispatchSuggestions entries.
func mergeSuggestions(candidates ...[]Suggestion) []Suggestion {
	const maxDispatchSuggestions = 3
	seen := map[string]struct{}{}
	var all []Suggestion
	for _, list := range candidates {
		for _, s := range list {
			if s.Invocation == "" {
				continue
			}
			if _, ok := seen[s.Invocation]; ok {
				continue
			}
			seen[s.Invocation] = struct{}{}
			all = append(all, s)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		ri, rj := kindRank(all[i].Kind), kindRank(all[j].Kind)
		if ri != rj {
			return ri < rj
		}
		if all[i].Distance != all[j].Distance {
			return all[i].Distance < all[j].Distance
		}
		return all[i].Name < all[j].Name
	})
	if len(all) > maxDispatchSuggestions {
		all = all[:maxDispatchSuggestions]
	}
	return all
}

func formatBuiltinInvocation(name string) string {
	return "m " + name
}

func formatScriptInvocation(name string, reserved bool, directEnabled bool) string {
	if reserved || !directEnabled {
		return "m run " + name
	}
	return "m " + name
}
