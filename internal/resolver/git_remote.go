package resolver

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/process"
)

var (
	gitRemoteCache sync.Map
	gitSupervisor  process.ProcessSupervisor = process.NewExecSupervisor()
)

// SetGitSupervisor replaces the git subprocess supervisor (tests).
func SetGitSupervisor(sup process.ProcessSupervisor) {
	if sup == nil {
		gitSupervisor = process.NewExecSupervisor()
		return
	}
	gitSupervisor = sup
}

// ResolveGitCommit resolves ref to a commit hash via git ls-remote.
func ResolveGitCommit(ctx context.Context, rawURL, ref string, offline bool) (string, error) {
	if offline {
		return "", apperr.New(apperr.Network, "resolver.git", rawURL, "offline; cannot resolve git ref")
	}
	normalized, err := NormalizeGitURL(rawURL)
	if err != nil {
		return "", apperr.Wrap(apperr.Resolve, "resolver.git", rawURL, err)
	}
	if err := ValidateGitURL(normalized); err != nil {
		return "", apperr.Wrap(apperr.Resolve, "resolver.git", rawURL, err)
	}
	cacheKey := normalized + "\x00" + ref
	if v, ok := gitRemoteCache.Load(cacheKey); ok {
		if commit, ok := v.(string); ok && commit != "" {
			return commit, nil
		}
	}
	commit, err := lsRemoteCommit(ctx, normalized, ref)
	if err != nil {
		return "", err
	}
	gitRemoteCache.Store(cacheKey, commit)
	return commit, nil
}

func lsRemoteCommit(ctx context.Context, repoURL, ref string) (string, error) {
	args := []string{"ls-remote", repoURL}
	if ref != "" {
		args = append(args, ref)
	}
	out, err := runGit(ctx, "", args...)
	if err != nil {
		return "", apperr.Wrap(apperr.Network, "resolver.git", repoURL, err)
	}
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 1 {
			continue
		}
		commit := strings.ToLower(fields[0])
		if gitCommitPattern.MatchString(commit) {
			return commit, nil
		}
	}
	if ref != "" && gitCommitPattern.MatchString(strings.ToLower(ref)) {
		return strings.ToLower(ref), nil
	}
	return "", apperr.New(apperr.Resolve, "resolver.git", repoURL, "git ref not found")
}

func gitPeekPackageDir(ctx context.Context, repoURL, commit string, offline bool) (string, error) {
	if offline {
		return "", apperr.New(apperr.Network, "resolver.git", repoURL, "offline; cannot read git package")
	}
	dir, err := os.MkdirTemp("", "mew-git-peek-*")
	if err != nil {
		return "", apperr.Wrap(apperr.IO, "resolver.git", repoURL, err)
	}
	if err := cloneGitAt(ctx, repoURL, commit, dir); err != nil {
		_ = os.RemoveAll(dir)
		return "", err
	}
	return dir, nil
}

func cloneGitAt(ctx context.Context, repoURL, commit, dest string) error {
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "resolver.git", dest, err)
	}
	if _, err := runGit(ctx, dest, "init"); err != nil {
		return err
	}
	if _, err := runGit(ctx, dest, "remote", "add", "origin", repoURL); err != nil {
		return err
	}
	fetchRef := commit
	if fetchRef == "" {
		fetchRef = "HEAD"
	}
	if _, err := runGit(ctx, dest, "fetch", "--depth", "1", "origin", fetchRef); err != nil {
		return apperr.Wrap(apperr.Network, "resolver.git", repoURL, err)
	}
	if _, err := runGit(ctx, dest, "checkout", "FETCH_HEAD"); err != nil {
		return apperr.Wrap(apperr.Resolve, "resolver.git", commit, err)
	}
	return nil
}

func runGit(ctx context.Context, dir string, args ...string) ([]byte, error) {
	env := append(process.RestrictedEnv(process.EnvSource{}, ""),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_CONFIG_COUNT=2",
		"GIT_CONFIG_KEY_0=safe.directory",
		"GIT_CONFIG_VALUE_0=*",
		"GIT_CONFIG_KEY_1=protocol.file.allow",
		"GIT_CONFIG_VALUE_1=always",
	)
	fullArgs := append([]string{
		"-c", "core.hooksPath=" + devNullHooksPath(),
		"-c", "init.templateDir=",
		"-c", "protocol.file.allow=always",
	}, args...)
	spec := process.Spec{
		Path: "git",
		Args: fullArgs,
		Dir:  dir,
		Env:  env,
	}
	h, err := gitSupervisor.Start(ctx, spec)
	if err != nil {
		return nil, err
	}
	if err := gitSupervisor.Wait(ctx, h); err != nil {
		return nil, err
	}
	return nil, nil
}

func devNullHooksPath() string {
	if filepath.Separator == '\\' {
		return "NUL"
	}
	return "/dev/null"
}
