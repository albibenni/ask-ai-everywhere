package release_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type releaseFixture struct {
	repo, remote, makeLog string
	env                   []string
}

func newReleaseFixture(t *testing.T) *releaseFixture {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the Bash release workflow is run from Linux or macOS")
	}
	root := t.TempDir()
	f := &releaseFixture{
		repo: filepath.Join(root, "work"), remote: filepath.Join(root, "origin.git"),
		makeLog: filepath.Join(root, "make.log"),
	}
	bin := filepath.Join(root, "bin")
	for _, path := range []string{f.repo, bin} {
		if err := os.MkdirAll(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	f.env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_AUTHOR_NAME=Release Test", "GIT_AUTHOR_EMAIL=release@example.invalid",
		"GIT_COMMITTER_NAME=Release Test", "GIT_COMMITTER_EMAIL=release@example.invalid",
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "RELEASE_TEST_MAKE_LOG="+f.makeLog)
	if err := os.WriteFile(filepath.Join(bin, "make"), []byte("#!/usr/bin/env bash\nprintf '%s\\n' \"$*\" >> \"$RELEASE_TEST_MAKE_LOG\"\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(t, "init", "--bare", "--initial-branch=main", f.remote)
	f.git(t, "init", "--initial-branch=main", f.repo)
	script, err := os.ReadFile("../../scripts/release.sh")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(f.repo, "release.sh"), script, 0o755); err != nil {
		t.Fatal(err)
	}
	f.git(t, "add", ".")
	f.git(t, "commit", "-m", "initial")
	f.git(t, "remote", "add", "origin", f.remote)
	f.git(t, "push", "-u", "origin", "main")
	f.git(t, "tag", "v0.2.6")
	return f
}

func (f *releaseFixture) git(t *testing.T, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir, command.Env = f.repo, f.env
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return strings.TrimSpace(string(output))
}

func (f *releaseFixture) run(input string) (string, error) {
	command := exec.Command("bash", "release.sh")
	command.Dir, command.Env, command.Stdin = f.repo, f.env, strings.NewReader(input)
	output, err := command.CombinedOutput()
	return string(output), err
}

func (f *releaseFixture) assertNoBuild(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(f.makeLog); !os.IsNotExist(err) {
		t.Fatalf("release should stop before building; make log error = %v", err)
	}
}

func TestReleaseSuggestsNextVersionFromRemoteTags(t *testing.T) {
	f := newReleaseFixture(t)
	// Reproduce a stale local tag after a failed release, and a higher release
	// known only to the remote. Fetching tags must not overwrite the local tag.
	f.git(t, "tag", "v0.2.7")
	f.git(t, "--git-dir="+f.remote, "tag", "v0.2.8", "main")
	output, err := f.run("\ny\n")
	if err != nil {
		t.Fatalf("release failed: %v\n%s", err, output)
	}
	if !strings.Contains(output, "Released v0.2.9") {
		t.Fatalf("expected next unused version v0.2.9:\n%s", output)
	}
	if got := f.git(t, "--git-dir="+f.remote, "rev-parse", "v0.2.9^{}"); got != f.git(t, "rev-parse", "HEAD") {
		t.Fatal("release tag does not point to the released commit")
	}
}

func TestReleaseStopsBeforeBuildWhenRemoteMainHasNewCommits(t *testing.T) {
	for _, diverged := range []bool{false, true} {
		name := "behind"
		if diverged {
			name = "diverged"
		}
		t.Run(name, func(t *testing.T) {
			f := newReleaseFixture(t)
			f.git(t, "checkout", "-b", "remote-change")
			f.git(t, "commit", "--allow-empty", "-m", "remote change")
			f.git(t, "push", "origin", "HEAD:main")
			f.git(t, "checkout", "main")
			if diverged {
				f.git(t, "commit", "--allow-empty", "-m", "local change")
			}
			output, err := f.run("v0.2.9\ny\n")
			if err == nil || !strings.Contains(output, "origin/main") {
				t.Fatalf("expected branch preflight failure: error = %v\n%s", err, output)
			}
			f.assertNoBuild(t)
		})
	}
}

func TestReleaseRejectsExistingRemoteTagBeforeBuild(t *testing.T) {
	f := newReleaseFixture(t)
	f.git(t, "--git-dir="+f.remote, "tag", "v0.2.8", "main")
	output, err := f.run("v0.2.8\ny\n")
	if err == nil || !strings.Contains(output, "already exists on origin") {
		t.Fatalf("expected existing remote tag rejection: error = %v\n%s", err, output)
	}
	f.assertNoBuild(t)
}
