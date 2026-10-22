// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/yamlx"
)

// workflowsDir is where GitHub reads the repository's workflows from.
// Naming it here is also what has a change to a workflow reach these
// tests.
const workflowsDir = "../../.github/workflows"

// ciWorkflow is CI's workflow (ADR-013 D5).
const ciWorkflow = "../../.github/workflows/ci.yml"

// dcoScript is CI's DCO check (ADR-013 D3 item 2).
const dcoScript = "../../.github/scripts/dco.sh"

// conformanceScript is how CI splits internal/conformance's tests (R-W1-4).
const conformanceScript = "../../.github/scripts/conformance.sh"

// TestWorkflows is ADR-013 D5 item 1's tripwire and ADR-010 Amendment A8
// item 5's: every file under .github/workflows/ runs its jobs on
// GitHub-hosted runners only, never on the pull_request_target trigger,
// and declares a top-level permissions block. A self-hosted runner is
// shared with other work and holds credentials, so no stranger's pull
// request may run on one; pull_request_target runs a pull request's
// workflow with the base repository's token and secrets.
func TestWorkflows(t *testing.T) {
	n := 0
	err := filepath.WalkDir(workflowsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		n++
		src, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(workflowsDir, p)
		for _, msg := range workflowProblems(src) {
			t.Errorf(".github/workflows/%s %s", filepath.ToSlash(rel), msg)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if n == 0 {
		t.Fatalf("%s holds no workflow; the tripwire read nothing", workflowsDir)
	}
}

// TestWorkflowsControl feeds workflowProblems a workflow breaking each
// rule, and one breaking none.
func TestWorkflowsControl(t *testing.T) {
	const ok = "name: x\non: [workflow_dispatch]\npermissions:\n  contents: read\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n"
	if got := workflowProblems([]byte(ok)); len(got) != 0 {
		t.Errorf("a workflow breaking no rule has problems: %q", got)
	}
	for _, c := range []struct {
		name, src, want string
	}{
		{"self-hosted", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: self-hosted\n",
			`job a runs on runner label "self-hosted", a self-hosted runner`},
		{"self-hosted in a list", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: [Self-Hosted, linux]\n",
			`job a runs on runner label "Self-Hosted", a self-hosted runner`},
		{"self-hosted in labels", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on:\n      labels: [self-hosted]\n",
			`job a runs on runner label "self-hosted", a self-hosted runner`},
		{"a custom label", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: gpu-box\n",
			`job a runs on runner label "gpu-box", which no GitHub-hosted runner carries`},
		{"a runner group", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on:\n      group: builders\n",
			`job a runs on runner group "builders", not a GitHub-hosted runner`},
		{"an expression", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    strategy:\n      matrix:\n        os: [self-hosted]\n    runs-on: ${{ matrix.os }}\n",
			`job a runs on runner label "${{ matrix.os }}", an expression no test can read`},
		{"no runner", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    steps: []\n",
			"job a names no runs-on and no workflow of this repository to call"},
		{"a remote workflow", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    uses: other/repo/.github/workflows/x.yml@main\n",
			`job a calls workflow "other/repo/.github/workflows/x.yml@main", which is not one of this repository's`},
		{"pull_request_target", "on: pull_request_target\npermissions: {}\njobs: {}\n",
			"is triggered by pull_request_target"},
		{"pull_request_target in a list", "on: [push, pull_request_target]\npermissions: {}\njobs: {}\n",
			"is triggered by pull_request_target"},
		{"pull_request_target in a map", "on:\n  pull_request_target:\n    types: [opened]\npermissions: {}\njobs: {}\n",
			"is triggered by pull_request_target"},
		{"no permissions", "on: workflow_dispatch\njobs:\n  a:\n    runs-on: ubuntu-latest\n",
			"declares no top-level permissions"},
		{"job permissions alone", "on: workflow_dispatch\njobs:\n  a:\n    runs-on: ubuntu-latest\n    permissions:\n      contents: read\n",
			"declares no top-level permissions"},
		{"not YAML", "on: [\n", "is not YAML METAIS reads"},
		{"not a mapping", "- a\n", "is not a mapping"},
	} {
		got := workflowProblems([]byte(c.src))
		if !slices.ContainsFunc(got, func(p string) bool { return strings.HasPrefix(p, c.want) }) {
			t.Errorf("%s: problems %q, want one beginning %q", c.name, got, c.want)
		}
	}
}

// hostedLabel is a GitHub-hosted runner's label: an image's name and its
// version or latest, such as ubuntu-latest or macos-15.
var hostedLabel = regexp.MustCompile(`^(ubuntu|macos|windows)-[0-9a-z.-]+$`)

// localWorkflow is a reusable workflow of this repository, which this
// tripwire reads too.
var localWorkflow = regexp.MustCompile(`^\./\.github/workflows/[^/]+\.ya?ml$`)

// workflowProblems is what makes src a workflow TestWorkflows refuses,
// one sentence about the workflow for each problem.
func workflowProblems(src []byte) []string {
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "workflow"}}, limits.Defaults())
	if len(fds) > 0 {
		return []string{"is not YAML METAIS reads: " + fds[0].Message()}
	}
	w, ok := doc.Value.(value.Object)
	if !ok {
		return []string{"is not a mapping"}
	}
	var out []string
	if _, ok := w.Get("permissions"); !ok {
		out = append(out, "declares no top-level permissions block")
	}
	if on, ok := w.Get("on"); ok && slices.Contains(triggers(on), "pull_request_target") {
		out = append(out, "is triggered by pull_request_target")
	}
	jobs, _ := w.Get("jobs")
	jobsObj, _ := jobs.(value.Object)
	for name, job := range jobsObj.All() {
		j, _ := job.(value.Object)
		runsOn, hasRunner := j.Get("runs-on")
		if uses, ok := j.Get("uses"); ok {
			if s := text(uses); !localWorkflow.MatchString(s) {
				out = append(out, "job "+name+" calls workflow "+quote(s)+", which is not one of this repository's")
			}
			continue
		}
		if !hasRunner {
			out = append(out, "job "+name+" names no runs-on and no workflow of this repository to call")
			continue
		}
		labels, group := runners(runsOn)
		if group != "" {
			out = append(out, "job "+name+" runs on runner group "+quote(group)+", not a GitHub-hosted runner")
		}
		for _, l := range labels {
			switch {
			case strings.EqualFold(l, "self-hosted"):
				out = append(out, "job "+name+" runs on runner label "+quote(l)+", a self-hosted runner")
			case strings.Contains(l, "${{"):
				out = append(out, "job "+name+" runs on runner label "+quote(l)+", an expression no test can read")
			case !hostedLabel.MatchString(l):
				out = append(out, "job "+name+" runs on runner label "+quote(l)+", which no GitHub-hosted runner carries")
			}
		}
	}
	return out
}

// triggers is the events an on: names: one, a list of them, or a mapping
// keyed by them.
func triggers(on value.Value) []string {
	switch v := on.(type) {
	case value.String:
		return []string{v.Text()}
	case value.Array:
		var out []string
		for _, e := range v.All() {
			out = append(out, text(e))
		}
		return out
	case value.Object:
		return v.Names()
	}
	return nil
}

// runners is the labels and the group a runs-on names: a label, a list of
// them, or a mapping of a group and labels. A value of no such shape is
// returned as a label of its own text, which no rule accepts.
func runners(runsOn value.Value) (labels []string, group string) {
	switch v := runsOn.(type) {
	case value.String:
		return []string{v.Text()}, ""
	case value.Array:
		for _, e := range v.All() {
			labels = append(labels, text(e))
		}
		return labels, ""
	case value.Object:
		if l, ok := v.Get("labels"); ok {
			labels, _ = runners(l)
		}
		if g, ok := v.Get("group"); ok {
			group = text(g)
		}
		return labels, group
	}
	return []string{text(runsOn)}, ""
}

// text is a string value's text, and a marker for any other value.
func text(v value.Value) string {
	if s, ok := v.(value.String); ok {
		return s.Text()
	}
	return "(not a string)"
}

func quote(s string) string { return `"` + s + `"` }

// TestCIWorkflow holds ci.yml to ADR-013 D5: while the repository is
// private it runs only when dispatched (item 5), its token is read-only
// (ADR-013 D5 item 4's default, declared in the workflow itself), and its
// jobs are item 2's, each on the platform it names, running what it names,
// with internal/conformance's tests split out of the run of the others and
// run on their own, as R-W1-4 splits them: a job that skipped them and did
// not run them would run them nowhere.
// The flip adds pull_request and push to main, and changes this test.
func TestCIWorkflow(t *testing.T) {
	src, err := os.ReadFile(ciWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "ci"}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatal(fds[0].Message())
	}
	w := doc.Value.(value.Object)
	on, _ := w.Get("on")
	if got := triggers(on); !slices.Equal(got, []string{"workflow_dispatch"}) {
		t.Errorf("ci.yml is triggered by %q; while the repository is private, only by workflow_dispatch", got)
	}
	perms, _ := w.Get("permissions")
	readOnly, _ := value.NewObject(value.Member{Name: "contents", Value: mustString("read")})
	if !value.Equal(perms, readOnly) {
		t.Errorf("ci.yml's top-level permissions are not contents: read alone")
	}
	jobs, _ := w.Get("jobs")
	jobsObj, _ := jobs.(value.Object)
	want := []struct {
		job, runsOn string
		runs        []string
	}{
		{"wave", "ubuntu-latest", []string{"skip=$(.github/scripts/conformance.sh skip)", `GOFLAGS="-skip=$skip" METAIS_GATE_P=$(nproc) .claude/skills/verification-gate/gate.sh wave`, ".github/scripts/conformance.sh run"}},
		{"macos", "macos-latest", []string{"go build ./...", "CGO_ENABLED=0 go build -trimpath ./cmd/metais", "skip=$(.github/scripts/conformance.sh skip)", `go test -race -count=1 ./... -skip "$skip"`, ".github/scripts/conformance.sh run"}},
		{"dco", "ubuntu-latest", []string{".github/scripts/dco.sh"}},
		{"shell", "ubuntu-latest", []string{"xargs -0 shellcheck"}},
		{"site", "ubuntu-latest", []string{"site/build.sh", "go test -race -count=1 ./site/..."}},
	}
	var names []string
	for _, j := range want {
		names = append(names, j.job)
	}
	if got := jobsObj.Names(); !slices.Equal(got, slices.Sorted(slices.Values(names))) {
		t.Errorf("ci.yml's jobs are %q, want %q", got, names)
	}
	for _, j := range want {
		job, _ := jobsObj.Get(j.job)
		jo, _ := job.(value.Object)
		if l, _ := runners(must(jo.Get("runs-on"))); !slices.Equal(l, []string{j.runsOn}) {
			t.Errorf("ci.yml's job %s runs on %q, want %s", j.job, l, j.runsOn)
		}
		var run strings.Builder
		steps, _ := jo.Get("steps")
		stepsArr, _ := steps.(value.Array)
		for _, s := range stepsArr.All() {
			so, _ := s.(value.Object)
			if r, ok := so.Get("run"); ok {
				run.WriteString(text(r) + "\n")
			}
		}
		for _, r := range j.runs {
			if !strings.Contains(run.String(), r) {
				t.Errorf("ci.yml's job %s does not run %q", j.job, r)
			}
		}
	}
}

func mustString(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic(err)
	}
	return v
}

// must is v, or a null when there is none, which no rule accepts.
func must(v value.Value, ok bool) value.Value {
	if !ok {
		return value.Null{}
	}
	return v
}

// TestDCOCheck runs CI's DCO check over a repository of signed and
// unsigned commits (ADR-013 D3 item 2). Every commit of the change must
// carry a Signed-off-by: trailer naming a person and an email address, a
// merge commit included, whoever the commit's author is; when the change has no commit of its own, the
// commit checked is its head.
func TestDCOCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the DCO check runs on CI's linux runner")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("the DCO check reads git: %v", err)
	}
	script, err := filepath.Abs(dcoScript)
	if err != nil {
		t.Fatal(err)
	}
	repo := testenv.Dir(t)
	env := append(testenv.GitEnv(os.Environ()),
		"GIT_AUTHOR_NAME=A Contributor", "GIT_AUTHOR_EMAIL=a@example.invalid",
		"GIT_COMMITTER_NAME=A Contributor", "GIT_COMMITTER_EMAIL=a@example.invalid")
	git := func(args ...string) string {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = repo
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	commit := func(msg string) string {
		t.Helper()
		git("commit", "--quiet", "--allow-empty", "-m", msg)
		return git("rev-parse", "HEAD")
	}
	const signed = "\n\nSigned-off-by: A Contributor <a@example.invalid>"
	git("init", "--quiet", "--initial-branch=main", ".")
	base := commit("base" + signed)
	git("checkout", "--quiet", "-b", "signed")
	commit("one" + signed)
	signedHead := commit("two\n\nA body.\n\nReviewed-by: B <b@example.invalid>\nSigned-off-by: A Contributor <a@example.invalid>")
	git("checkout", "--quiet", "-b", "unsigned")
	unsignedHead := commit("three, unsigned")
	git("checkout", "--quiet", "-b", "nameless", signedHead)
	namelessHead := commit("four" + "\n\nSigned-off-by: A Contributor")
	git("checkout", "--quiet", "-b", "otherperson", signedHead)
	otherHead := commit("six\n\nSigned-off-by: B Maintainer <b@example.invalid>")
	git("checkout", "--quiet", "-b", "inbody", signedHead)
	inBodyHead := commit("five\n\nSigned-off-by: A Contributor <a@example.invalid>\n\nis not a trailer here.")
	git("checkout", "--quiet", "-b", "merge", signedHead)
	git("merge", "--quiet", "--no-ff", "--no-edit", "-m", "an unsigned merge", unsignedHead)
	unsignedMerge := git("rev-parse", "HEAD")
	git("checkout", "--quiet", "-b", "signedmerge", signedHead)
	git("merge", "--quiet", "--no-ff", "--no-edit", "-m", "a signed merge"+signed, "nameless")
	git("checkout", "--quiet", "main")
	unsignedMain := commit("on main, unsigned")

	for _, c := range []struct {
		name, base, head string
		code             int
		says             string
	}{
		{"every commit signed off", base, signedHead, 0, "all 2 commit(s) are signed off"},
		{"one commit unsigned", base, unsignedHead, 1, "three, unsigned carries no Signed-off-by"},
		{"a sign-off with no address", base, namelessHead, 1, "four carries no Signed-off-by"},
		{"a sign-off by someone other than the author", signedHead, otherHead, 0, "all 1 commit(s) are signed off"},
		{"a sign-off that is not a trailer", base, inBodyHead, 1, "five carries no Signed-off-by"},
		{"an unsigned merge", base, unsignedMerge, 1, "an unsigned merge carries no Signed-off-by"},
		{"a signed merge of an unsigned commit", base, "signedmerge", 1, "four carries no Signed-off-by"},
		{"the head on the base, signed", signedHead, signedHead, 0, "all 1 commit(s) are signed off"},
		{"the head on the base, unsigned", "main", unsignedMain, 1, "on main, unsigned carries no Signed-off-by"},
		{"a base that does not exist", "nosuchref", signedHead, 2, "cannot list the commits"},
	} {
		cmd := exec.Command("sh", script, c.base, c.head)
		cmd.Dir = repo
		cmd.Env = env
		out, err := cmd.CombinedOutput()
		code := 0
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			code = exit.ExitCode()
		} else if err != nil {
			t.Fatalf("%s: %v", c.name, err)
		}
		if code != c.code || !strings.Contains(string(out), c.says) {
			t.Errorf("%s: dco.sh exited %d saying\n%s\nwant exit %d saying %q", c.name, code, out, c.code, c.says)
		}
	}
}

// conformanceStub stands in for the go command in TestConformanceSplit.
// -list prints the file list; -run records its selection in runs and
// prints the file out/<selection, its punctuation as _>, exiting 1 when a
// .fail file sits beside it.
const conformanceStub = `#!/bin/sh
while [ $# -gt 0 ]; do
	case $1 in
	-list) cat "$STUB/list"; exit 0 ;;
	-run)
		shift
		printf '%s\n' "$1" >>"$STUB/runs"
		f="$STUB/out/$(printf '%s' "$1" | tr -c 'A-Za-z0-9' _)"
		[ -f "$f" ] && cat "$f"
		[ -f "$f.fail" ] && exit 1
		exit 0 ;;
	esac
	shift
done
exit 2
`

// TestConformanceSplit runs CI's split of internal/conformance (R-W1-4)
// over a stand-in go command and a suite of two areas. skip names every
// test the package lists but TestHostIsolation, and refuses a name another
// package declares; run runs each in its own invocation, TestConformance
// once per area, and fails on an invocation that fails or that did not run
// what it selects, after running the rest.
func TestConformanceSplit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the split runs on CI's linux and macOS runners")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("the split reads git: %v", err)
	}
	script, err := filepath.Abs(conformanceScript)
	if err != nil {
		t.Fatal(err)
	}
	outName := func(sel string) string {
		return regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(sel, "_")
	}
	passing := map[string]string{
		"^TestConformance$/^a$": "    --- PASS: TestConformance/a (0.00s)\n",
		"^TestConformance$/^b$": "    --- PASS: TestConformance/b (0.00s)\n",
		"^TestOther$":           "--- PASS: TestOther (0.00s)\n",
		"^ExampleX$":            "--- PASS: ExampleX (0.00s)\n",
	}
	allRuns := []string{"^TestConformance$/^a$", "^TestConformance$/^b$", "^TestOther$", "^ExampleX$"}
	for _, c := range []struct {
		name, mode string
		edit       func(repo, stub string)
		code       int
		says       string
		runs       []string
	}{
		{"skip names every test but TestHostIsolation", "skip", nil, 0, "^(TestConformance|TestOther|ExampleX)$", nil},
		{"skip refuses a name another package declares", "skip", func(repo, _ string) {
			write(t, filepath.Join(repo, "internal", "other", "other_test.go"), "package other\n\nfunc TestOther(t *testing.T) {}\n")
		}, 1, "another package declares TestOther", nil},
		{"run runs each test, and TestConformance per area", "run", nil, 0, "=== TestConformance/b", allRuns},
		{"run fails on an area that did not run", "run", func(_, stub string) {
			write(t, filepath.Join(stub, "out", outName("^TestConformance$/^b$")), "ok\n")
		}, 1, "conformance: TestConformance/b did not run", allRuns},
		{"run fails on a failing test, and runs the rest", "run", func(_, stub string) {
			f := filepath.Join(stub, "out", outName("^TestOther$"))
			write(t, f, "--- FAIL: TestOther (0.00s)\n")
			write(t, f+".fail", "")
		}, 1, "--- FAIL: TestOther", allRuns},
	} {
		t.Run(c.name, func(t *testing.T) {
			repo, stub := testenv.Dir(t), testenv.Dir(t)
			write(t, filepath.Join(stub, "go"), conformanceStub)
			if err := os.Chmod(filepath.Join(stub, "go"), 0o755); err != nil {
				t.Fatal(err)
			}
			write(t, filepath.Join(stub, "list"), "TestConformance\nTestHostIsolation\nTestOther\nExampleX\nok  \tmetais.org/metais/internal/conformance\t0.01s\n")
			for sel, out := range passing {
				write(t, filepath.Join(stub, "out", outName(sel)), out)
			}
			for _, v := range []string{"a/one", "b/two"} {
				write(t, filepath.Join(repo, "spec", "conformance", "vectors", filepath.FromSlash(v), "vector.json"), "{}\n")
			}
			write(t, filepath.Join(repo, "internal", "conformance", "x_test.go"), "package conformance\n\nfunc TestOther(t *testing.T) {}\n")
			if c.edit != nil {
				c.edit(repo, stub)
			}
			env := append(testenv.GitEnv(os.Environ()), "STUB="+stub, "PATH="+stub+string(os.PathListSeparator)+os.Getenv("PATH"))
			for _, args := range [][]string{{"init", "--quiet", "."}, {"add", "."}} {
				cmd := exec.Command("git", args...)
				cmd.Dir = repo
				cmd.Env = env
				if out, err := cmd.CombinedOutput(); err != nil {
					t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
				}
			}
			cmd := exec.Command("sh", script, c.mode)
			cmd.Dir = repo
			cmd.Env = env
			out, err := cmd.CombinedOutput()
			code := 0
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				code = exit.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if code != c.code || !strings.Contains(string(out), c.says) {
				t.Errorf("conformance.sh %s exited %d saying\n%s\nwant exit %d saying %q", c.mode, code, out, c.code, c.says)
			}
			runs, _ := os.ReadFile(filepath.Join(stub, "runs"))
			if got := strings.Fields(string(runs)); !slices.Equal(got, c.runs) {
				t.Errorf("conformance.sh %s ran %q, want %q", c.mode, got, c.runs)
			}
		})
	}
}

// write writes a file, and the directories above it.
func write(t *testing.T, name, data string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}
