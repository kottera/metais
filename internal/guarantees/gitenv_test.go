// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// gitEnvScript is what the gate's scripts source before they run git.
// Naming it here is also what has a change to it reach this test.
const gitEnvScript = "../../.claude/skills/verification-gate/git-env.sh"

// TestGitEnvScriptUnderPOSIXShells sources git-env.sh under each POSIX
// shell the host has, as merge-setup.sh is sourced when a fresh clone runs
// it directly on a /bin/sh that is dash (W7-NB1.B). The caller's
// environment sets GIT_* variables that would send git to another
// repository and give it configuration of the caller's, GIT_CONFIG_PARAMETERS
// among them, which outranks git-env.sh's own GIT_CONFIG_COUNT. After
// sourcing it, the only GIT_* variables left must be git-env.sh's own, and
// git must find the repository it runs in and none of the caller's
// configuration.
func TestGitEnvScriptUnderPOSIXShells(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the gate's scripts run on darwin and linux")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("git-env.sh is for git: %v", err)
	}
	script, err := filepath.Abs(gitEnvScript)
	if err != nil {
		t.Fatal(err)
	}
	type shell struct {
		name string
		argv []string
	}
	var shells []shell
	if p, err := exec.LookPath("dash"); err == nil {
		shells = append(shells, shell{"dash", []string{p}})
	}
	if p, err := exec.LookPath("bash"); err == nil {
		shells = append(shells, shell{"bash --posix", []string{p, "--posix"}})
	}
	if p, err := exec.LookPath("sh"); err == nil {
		shells = append(shells, shell{"sh", []string{p}})
	}
	if len(shells) == 0 {
		t.Skip("the host has no POSIX shell")
	}

	dir := testenv.Dir(t)
	repo := filepath.Join(dir, "repo")
	other := filepath.Join(dir, "other")
	for _, d := range []string{repo, other} {
		cmd := exec.Command("git", "init", "--quiet", d)
		cmd.Env = testenv.GitEnv(os.Environ())
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git init %s: %v\n%s", d, err, out)
		}
	}

	caller := append(testenv.GitEnv(os.Environ()),
		"GIT_DIR="+filepath.Join(other, ".git"),
		"GIT_WORK_TREE="+other,
		"GIT_INDEX_FILE="+filepath.Join(other, ".git", "index"),
		"GIT_CONFIG_PARAMETERS='user.name=the caller'",
		"GIT_CONFIG_KEY_3=user.email",
		"GIT_CONFIG_VALUE_3=caller@example.invalid",
		"GIT_CEILING_DIRECTORIES="+dir,
		"GIT_TRACE_SETUP=0",
		"GIT_MULTILINE=a value\nGIT_NOT_A_VARIABLE=over two lines",
	)
	want := []string{
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_GLOBAL=" + os.DevNull,
		"GIT_CONFIG_KEY_0=core.precomposeunicode",
		"GIT_CONFIG_KEY_1=core.excludesFile",
		"GIT_CONFIG_KEY_2=core.attributesFile",
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_CONFIG_VALUE_0=false",
		"GIT_CONFIG_VALUE_1=" + os.DevNull,
		"GIT_CONFIG_VALUE_2=" + os.DevNull,
	}
	// The probe sources git-env.sh as the scripts do, under set -u, then
	// prints the GIT_* variables it left, the repository git finds, and
	// the caller's configuration git still sees.
	const probe = `set -u
. "$1" || exit 2
env | grep '^GIT_' | LC_ALL=C sort
echo ---
git rev-parse --absolute-git-dir
git config user.name || echo no user.name
git config user.email || echo no user.email
`
	for _, sh := range shells {
		t.Run(sh.name, func(t *testing.T) {
			cmd := exec.Command(sh.argv[0], slices.Concat(sh.argv[1:], []string{"-c", probe, "probe", script})...)
			cmd.Dir = repo
			cmd.Env = caller
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("sourcing git-env.sh: %v\n%s", err, out)
			}
			vars, rest, ok := strings.Cut(string(out), "---\n")
			if !ok {
				t.Fatalf("the probe printed no separator:\n%s", out)
			}
			if got := strings.Split(strings.TrimSpace(vars), "\n"); !slices.Equal(got, want) {
				t.Errorf("git-env.sh left the GIT_* variables\n%s\nwant exactly\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
			}
			gitDir, err := filepath.EvalSymlinks(filepath.Join(repo, ".git"))
			if err != nil {
				t.Fatal(err)
			}
			if wantRest := gitDir + "\nno user.name\nno user.email\n"; rest != wantRest {
				t.Errorf("git under git-env.sh printed\n%s\nwant\n%s", rest, wantRest)
			}
		})
	}
}
