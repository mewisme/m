package runner

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
)

// Lookup resolves a script selector to one or more script names.
// Exact names return a single match. Regex selectors use /pattern/ syntax and
// return matching script names sorted lexicographically.
func Lookup(scripts map[string]string, selector string) ([]string, error) {
	if selector == "" {
		return nil, apperr.New(apperr.Usage, "runner.lookup", selector, "missing script name")
	}
	isRegex, pattern, err := parseRegexSelector(selector)
	if err != nil {
		return nil, err
	}
	if !isRegex {
		if _, ok := scripts[selector]; !ok {
			return nil, missingScriptError(selector, scripts)
		}
		return []string{selector}, nil
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, apperr.New(apperr.Usage, "runner.lookup", selector,
			fmt.Sprintf("invalid script pattern %q: %v", pattern, err))
	}
	var matches []string
	for name := range scripts {
		if re.MatchString(name) {
			matches = append(matches, name)
		}
	}
	if len(matches) == 0 {
		return nil, missingScriptError(selector, scripts)
	}
	sort.Strings(matches)
	return matches, nil
}

func parseRegexSelector(selector string) (isRegex bool, pattern string, err error) {
	if len(selector) < 2 || selector[0] != '/' {
		return false, selector, nil
	}
	last := strings.LastIndex(selector, "/")
	if last <= 0 {
		return false, "", apperr.New(apperr.Usage, "runner.lookup", selector,
			fmt.Sprintf("invalid script pattern %q: missing closing /", selector))
	}
	pattern = selector[1:last]
	if pattern == "" {
		return false, "", apperr.New(apperr.Usage, "runner.lookup", selector,
			fmt.Sprintf("invalid script pattern %q: empty pattern", selector))
	}
	return true, pattern, nil
}

func missingScriptError(selector string, scripts map[string]string) error {
	names := sortedScriptNames(scripts)
	msg := fmt.Sprintf("Missing script: %q", selector)
	if len(names) > 0 {
		msg += "\n\nAvailable scripts:\n"
		for _, name := range names {
			msg += "  " + name + "\n"
		}
	}
	return apperr.New(apperr.NotFound, "runner.lookup", selector, strings.TrimRight(msg, "\n"))
}

func sortedScriptNames(scripts map[string]string) []string {
	if len(scripts) == 0 {
		return nil
	}
	names := make([]string, 0, len(scripts))
	for name := range scripts {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
