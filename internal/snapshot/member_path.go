package snapshot

import (
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/graph"
)

const memberManifestSuffix = "/package.json"

var reservedMemberPrefixes = []string{".mew/", ".git/"}

// ParseMemberManifestPath validates rel and returns the workspace importer ID.
func ParseMemberManifestPath(rel string) (graph.ImporterID, error) {
	rel = filepath.ToSlash(strings.TrimSpace(rel))
	if rel == "" {
		return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "empty member manifest path")
	}
	if filepath.IsAbs(rel) || strings.HasPrefix(rel, "//") {
		return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path must be relative")
	}
	if strings.Contains(rel, "..") {
		return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path must not contain ..")
	}
	if !strings.HasSuffix(rel, memberManifestSuffix) {
		return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path must end with /package.json")
	}
	dir := strings.TrimSuffix(rel, memberManifestSuffix)
	if runtime.GOOS == "windows" {
		dir = strings.ToLower(dir)
	}
	if dir == "" || dir == "package.json" {
		return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path must include workspace member directory")
	}
	for _, part := range strings.Split(dir, "/") {
		if part == "" || part == "." {
			return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path has invalid component")
		}
	}
	for _, prefix := range reservedMemberPrefixes {
		if dir == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(dir, prefix) {
			return "", apperr.New(apperr.IO, "snapshot.member_path", rel, "member manifest path uses reserved prefix")
		}
	}
	return graph.ImporterID(dir), nil
}
