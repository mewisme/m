package fetch

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/mewisme/mew/internal/apperr"
	"github.com/mewisme/mew/internal/process"
)

// GitOptions configures a shallow git materialization.
type GitOptions struct {
	URL        string
	Commit     string
	Dest       string
	Offline    bool
	Supervisor process.ProcessSupervisor
}

// FetchGit clones url at commit into dest with hooks disabled and no submodules.
func FetchGit(ctx context.Context, opts GitOptions) error {
	if opts.Offline {
		return apperr.New(apperr.Network, "fetch.git", opts.URL, "offline; cannot fetch git source")
	}
	if opts.URL == "" || opts.Commit == "" {
		return apperr.New(apperr.Usage, "fetch.git", opts.URL, "missing git url or commit")
	}
	if opts.Dest == "" {
		return apperr.New(apperr.Usage, "fetch.git", opts.URL, "missing destination")
	}
	sup := opts.Supervisor
	if sup == nil {
		sup = process.NewExecSupervisor()
	}
	if err := os.RemoveAll(opts.Dest); err != nil {
		return apperr.Wrap(apperr.IO, "fetch.git", opts.Dest, err)
	}
	if err := os.MkdirAll(opts.Dest, 0o755); err != nil {
		return apperr.Wrap(apperr.IO, "fetch.git", opts.Dest, err)
	}
	if bareDir, ok := localGitDirFromURL(opts.URL); ok {
		return checkoutLocalBareRepo(ctx, sup, bareDir, opts.Commit, opts.Dest)
	}
	if err := runGitCmd(ctx, sup, opts.Dest, "init"); err != nil {
		return err
	}
	if err := runGitCmd(ctx, sup, opts.Dest, "remote", "add", "origin", opts.URL); err != nil {
		return err
	}
	if err := runGitCmd(ctx, sup, opts.Dest, "fetch", "--depth", "1", "origin", opts.Commit); err != nil {
		return apperr.Wrap(apperr.Network, "fetch.git", opts.URL, err)
	}
	if err := runGitCmd(ctx, sup, opts.Dest, "checkout", "FETCH_HEAD"); err != nil {
		return apperr.Wrap(apperr.Integrity, "fetch.git", opts.Commit, err)
	}
	return nil
}

func checkoutLocalBareRepo(ctx context.Context, sup process.ProcessSupervisor, gitDir, commit, dest string) error {
	if st, err := os.Stat(gitDir); err != nil || !st.IsDir() {
		return apperr.Wrap(apperr.Resolve, "fetch.git", gitDir, fmt.Errorf("local git repository not found"))
	}
	return runGitCmd(ctx, sup, dest, "--git-dir", gitDir, "--work-tree", dest, "checkout", "-f", commit)
}

func localGitDirFromURL(repoURL string) (string, bool) {
	u, err := url.Parse(repoURL)
	if err != nil || strings.ToLower(u.Scheme) != "file" {
		return "", false
	}
	path := u.Path
	if runtime.GOOS == "windows" {
		if strings.HasPrefix(path, "/") && len(path) > 2 && path[2] == ':' {
			path = strings.TrimPrefix(path, "/")
		}
	}
	path = filepath.Clean(path)
	if path == "" {
		return "", false
	}
	return path, true
}

func runGitCmd(ctx context.Context, sup process.ProcessSupervisor, dir string, args ...string) error {
	env := process.GitSubprocessEnv("")
	fullArgs := append([]string{
		"-c", "core.hooksPath=" + gitHooksDisabledPath(),
		"-c", "init.templateDir=",
		"-c", "protocol.file.allow=always",
	}, args...)
	spec := process.Spec{Path: "git", Args: fullArgs, Dir: dir, Env: env}
	h, err := sup.Start(ctx, spec)
	if err != nil {
		return apperr.Wrap(apperr.IO, "fetch.git", dir, err)
	}
	if err := sup.Wait(ctx, h); err != nil {
		return err
	}
	return nil
}

func gitHooksDisabledPath() string {
	if filepath.Separator == '\\' {
		return "NUL"
	}
	return "/dev/null"
}
