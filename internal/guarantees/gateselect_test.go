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

// gateSelect is the scoped tier's selector (ADR-010 Amendment A5 item 3).
// Naming it here is also what has a change to the gate's scripts reach
// these tests.
const gateSelect = "../../.claude/skills/verification-gate/gate-select.sh"

// needGateTools skips a test of the gate's scripts where they cannot run:
// they are bash over git, on the hosts that run the gate.
func needGateTools(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("the gate's scripts are bash, and run on darwin and linux")
	}
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("the gate's scripts need %s: %v", tool, err)
		}
	}
}

// runGateSelect runs the selector in dir with args, and returns its lines.
// Its go list takes goEnv's environment, which outranks the host's, so it
// reads the host's module cache and never the network.
func runGateSelect(t *testing.T, dir string, args ...string) []string {
	t.Helper()
	script, err := filepath.Abs(gateSelect)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", append([]string{script}, args...)...)
	cmd.Dir = dir
	cmd.Env = testenv.GitEnv(append(os.Environ(), goEnv(t)...))
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("gate-select.sh %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

// selectPaths runs the selector over the module, with paths as the change.
func selectPaths(t *testing.T, paths ...string) []string {
	t.Helper()
	list := filepath.Join(testenv.Dir(t), "paths")
	if err := os.WriteFile(list, []byte(strings.Join(paths, "\n")+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return runGateSelect(t, moduleRoot, "--paths", list)
}

// tested reports whether the selector's facts have pkg's tests run.
func tested(facts []string, pkg string) bool {
	p := modulePath + "/" + pkg
	return slices.Contains(facts, "test "+p) || slices.Contains(facts, "blind "+p)
}

// reachProbe is one line of testdata/gateselect/reach.txt: a changed path,
// the packages whose tests it reaches, and lines the output must hold.
type reachProbe struct {
	path  string
	test  []string
	facts []string
}

// reachProbes reads testdata/gateselect/reach.txt, in its order.
func reachProbes(t *testing.T) []reachProbe {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "gateselect", "reach.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var probes []reachProbe
	for line := range strings.SplitSeq(string(b), "\n") {
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, " | ")
		if len(f) < 2 {
			t.Fatalf("reach.txt: %q is not <path> | <packages> [| <line>...]", line)
		}
		p := reachProbe{path: f[0], facts: f[2:]}
		if f[1] != "-" {
			p.test = strings.Fields(f[1])
		}
		probes = append(probes, p)
	}
	if len(probes) == 0 {
		t.Fatal("reach.txt holds no probe")
	}
	return probes
}

// TestGateSelectReach holds the selector to what each probe reaches: the
// changes Wave 6's verification showed reached, a Go file in no package,
// which only the tests that walk the module's tree read, and a control
// that reaches nothing. TestLicenceHeaders and TestNoHostFacts walk every
// Go file below the module root, whatever go list sees, so a Go file
// anywhere reaches internal/guarantees.
func TestGateSelectReach(t *testing.T) {
	needGateTools(t)
	for _, c := range reachProbes(t) {
		t.Run(c.path, func(t *testing.T) {
			t.Parallel()
			facts := selectPaths(t, c.path)
			for _, pkg := range c.test {
				if !tested(facts, pkg) {
					t.Errorf("%s does not reach %s:\n%s", c.path, pkg, strings.Join(facts, "\n"))
				}
			}
			if c.test == nil {
				for _, f := range facts {
					if strings.HasPrefix(f, "test ") || strings.HasPrefix(f, "blind ") {
						t.Errorf("%s reaches a package: %s", c.path, f)
					}
				}
			}
			for _, f := range c.facts {
				if !slices.Contains(facts, f) {
					t.Errorf("%s: no line %q:\n%s", c.path, f, strings.Join(facts, "\n"))
				}
			}
		})
	}
}

// TestGateSelectWorkingTree: measured from a commit, the change includes an
// untracked file and a deleted one, as the scoped tier sees a working tree.
// The untracked file is reach.txt's first probe, a Go file in no package;
// the deleted one is its control, which reaches nothing, so the planted
// file alone reaches the packages its probe names.
func TestGateSelectWorkingTree(t *testing.T) {
	gateSelectWorkingTree(t)
}

// TestGateSelectWorkingTreeHostGit runs TestGateSelectWorkingTree's case
// under a host's git configuration that would change its result if any git
// the test or the selector runs read it: a global excludes file that
// ignores every file, the planted one included, and, from the environment,
// core.precomposeunicode on, which fails every git that reads a directory
// holding a non-ASCII name on a darwin host whose iconv cannot convert from
// UTF-8-MAC.
func TestGateSelectWorkingTreeHostGit(t *testing.T) {
	needGateTools(t)
	dir := testenv.Dir(t)
	ignore, global := filepath.Join(dir, "ignore"), filepath.Join(dir, "gitconfig")
	if err := os.WriteFile(ignore, []byte("*\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(global, []byte("[core]\n\texcludesFile = "+ignore+"\n\tprecomposeUnicode = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GIT_CONFIG_GLOBAL", global)
	t.Setenv("GIT_CONFIG_COUNT", "1")
	t.Setenv("GIT_CONFIG_KEY_0", "core.precomposeunicode")
	t.Setenv("GIT_CONFIG_VALUE_0", "true")
	gateSelectWorkingTree(t)
}

func gateSelectWorkingTree(t *testing.T) {
	t.Helper()
	needGateTools(t)
	probes := reachProbes(t)
	planted, deleted := probes[0], probes[len(probes)-1].path
	clone := filepath.Join(testenv.Dir(t), "clone")
	cmd := exec.Command("git", "clone", "-q", "--no-local", ".", clone)
	cmd.Dir = moduleRoot
	cmd.Env = testenv.GitEnv(os.Environ())
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git clone: %v\n%s", err, out)
	}
	p := filepath.Join(clone, filepath.FromSlash(planted.path))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte("//go:build ignore\n\npackage main\n\nfunc main() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(clone, filepath.FromSlash(deleted))); err != nil {
		t.Fatal(err)
	}
	facts := runGateSelect(t, clone, "--base", "HEAD")
	for _, f := range []string{"changed " + planted.path, "changed " + deleted} {
		if !slices.Contains(facts, f) {
			t.Errorf("no line %q:\n%s", f, strings.Join(facts, "\n"))
		}
	}
	for _, pkg := range planted.test {
		if !tested(facts, pkg) {
			t.Errorf("the change does not reach %s:\n%s", pkg, strings.Join(facts, "\n"))
		}
	}
}
