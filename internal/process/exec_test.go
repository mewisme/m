package process_test

import (
	"context"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/mewisme/m/internal/process"
)

func TestExecSupervisorEcho(t *testing.T) {
	sup := process.NewExecSupervisor()
	dir := t.TempDir()
	spec := process.Spec{
		Path: "echo hello",
		Dir:  dir,
		Env:  process.RestrictedEnv(process.EnvSource{Vars: os.Environ(), Explicit: true}, dir),
	}
	h, err := sup.Start(context.Background(), spec)
	if err != nil {
		t.Fatal(err)
	}
	if err := sup.Wait(context.Background(), h); err != nil {
		t.Fatal(err)
	}
}

func TestRestrictedEnvExplicitEmpty(t *testing.T) {
	t.Setenv("NPM_TOKEN", "leaked")
	env := process.RestrictedEnv(process.EnvSource{Vars: []string{}, Explicit: true}, "/nm/.bin")
	if len(env) != 1 {
		t.Fatalf("want exactly PATH entry, got %v", env)
	}
	if !strings.HasPrefix(env[0], "PATH=") && !strings.HasPrefix(env[0], "Path=") {
		t.Fatalf("want PATH prefix, got %q", env[0])
	}
	for _, kv := range env {
		if strings.Contains(kv, "leaked") {
			t.Fatal("host env leaked into explicit-empty")
		}
	}
}

func TestRestrictedEnvUnsetFallsBack(t *testing.T) {
	t.Setenv("MEW_TEST_ENV_MARKER", "present")
	env := process.RestrictedEnv(process.EnvSource{Explicit: false}, "/bin")
	found := false
	for _, kv := range env {
		if kv == "MEW_TEST_ENV_MARKER=present" {
			found = true
		}
	}
	if !found {
		t.Fatal("unset EnvSource should inherit host env")
	}
}

func TestRestrictedEnvSecretStripping(t *testing.T) {
	cases := []struct {
		key   string
		strip bool
		keep  string
	}{
		{"NPM_TOKEN", true, ""},
		{"GH_TOKEN", true, ""},
		{"GITLAB_TOKEN", true, ""},
		{"SSH_AUTH_SOCK", true, ""},
		{"DOCKER_HOST", true, ""},
		{"AWS_SECRET_KEY", true, ""},
		{"AZURE_CLIENT_SECRET", true, ""},
		{"GOOGLE_APPLICATION_CREDENTIALS", true, ""},
		{"MY_PRIVATE_KEY", true, ""},
		{"HOME", false, "/tmp"},
		{"NODE_OPTIONS", false, "--max-old-space-size=4096"},
		{"FOO", false, "bar=baz"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.key, func(t *testing.T) {
			val := "secret"
			if tc.keep != "" {
				val = tc.keep
			}
			env := process.RestrictedEnv(process.EnvSource{
				Vars:     []string{tc.key + "=" + val, "PATH=/bin"},
				Explicit: true,
			}, "/nm/.bin")
			want := tc.key + "=" + val
			got := false
			for _, kv := range env {
				if kv == want {
					got = true
				}
			}
			if tc.strip && got {
				t.Fatalf("%s should be stripped", tc.key)
			}
			if !tc.strip && !got {
				t.Fatalf("%s should be preserved, env=%v", tc.key, env)
			}
		})
	}
}

func TestRestrictedEnvPATHPrefix(t *testing.T) {
	env := process.RestrictedEnv(process.EnvSource{
		Vars:     []string{"PATH=/usr/bin"},
		Explicit: true,
	}, "/nm/.bin")
	pathKey := "PATH"
	if runtime.GOOS == "windows" {
		pathKey = "Path"
	}
	var pathCount int
	for _, kv := range env {
		if strings.HasPrefix(kv, pathKey+"=") {
			pathCount++
			if !strings.Contains(kv, "/nm/.bin") && !strings.Contains(kv, `\nm\.bin`) {
				t.Fatalf("PATH missing binDir prefix: %q", kv)
			}
		}
	}
	if pathCount != 1 {
		t.Fatalf("want exactly one PATH entry, got %d", pathCount)
	}
}

func TestResolveCommandComSpecFromEnv(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows only")
	}
	path, args, err := process.ResolveCommandForTest(process.Spec{
		Path: "echo ok",
		Env:  []string{"ComSpec=C:\\custom\\cmd.exe"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if path != `C:\custom\cmd.exe` {
		t.Fatalf("got shell %q", path)
	}
	if len(args) != 2 || args[0] != "/c" {
		t.Fatalf("args=%v", args)
	}
}
