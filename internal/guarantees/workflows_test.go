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
	"strconv"
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

// coAuthorScript is CI's check that no Co-authored-by line reaches main
// (AGENTS.md, Git and pull requests).
const coAuthorScript = "../../.github/scripts/coauthor.sh"

// squashBodyScript is the check of a squash merge's body before the merge
// (AGENTS.md, Git and pull requests).
const squashBodyScript = "../../.github/scripts/squash-body.sh"

// releaseWorkflow publishes a draft release (ADR-013 D7 item 3).
const releaseWorkflow = "../../.github/workflows/release.yml"

// conformanceScript is how CI splits internal/conformance's tests (R-W1-4).
const conformanceScript = "../../.github/scripts/conformance.sh"

// TestWorkflows is ADR-013 D5 item 1's tripwire and ADR-010 Amendment A8
// item 5's: every file under .github/workflows/ runs its jobs on
// GitHub-hosted runners only, never on the pull_request_target trigger,
// and declares a top-level permissions block. A self-hosted runner is
// shared with other work and holds credentials, so no stranger's pull
// request may run on one; pull_request_target runs a pull request's
// workflow with the base repository's token and secrets. Each workflow
// also names, in its env and each job's env and if, only the contexts
// GitHub admits there: GitHub refuses to parse one that names another,
// so it fails at its first dispatch, and a release workflow is dispatched
// first at a release.
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
	const ok = "name: x\non: [workflow_dispatch]\npermissions:\n  contents: read\nenv:\n  G: ${{ github.sha }}-${{ vars.V }}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    if: ${{ inputs.go && needs.b.result == 'runner.temp' }}\n    env:\n      A: ${{ inputs.x }}-${{ secrets.Y }}-${{ matrix.os }}-${{ github.event.inputs.z }}\n    steps:\n      - run: 'true'\n        env:\n          W: ${{ runner.temp }}/w\n      - uses: actions/checkout@fbc6f3992d24b796d5a048ff273f7fcc4a7b6c09 # v5.1.0\n      - uses: github/codeql-action/init@0123456789abcdef0123456789abcdef01234567\n      - uses: ./.github/actions/x\n      - uses: docker://alpine@sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef\n"
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
		{"an action at a tag", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: actions/checkout@v5\n",
			`job a step 2 uses "actions/checkout@v5", which no full commit SHA or digest pins`},
		{"an action at a branch", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: actions/checkout@main\n",
			`job a step 2 uses "actions/checkout@main", which no full commit SHA or digest pins`},
		{"an action at a short SHA", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: actions/checkout@fbc6f39\n",
			`job a step 2 uses "actions/checkout@fbc6f39", which no full commit SHA or digest pins`},
		{"an action with no @", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: actions/checkout\n",
			`job a step 2 uses "actions/checkout", which no full commit SHA or digest pins`},
		{"a SHA with no @", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: fbc6f3992d24b796d5a048ff273f7fcc4a7b6c09\n",
			`job a step 2 uses "fbc6f3992d24b796d5a048ff273f7fcc4a7b6c09", which no full commit SHA or digest pins`},
		{"a docker image with no digest", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: docker://alpine:3.20\n",
			`job a step 2 uses "docker://alpine:3.20", which no full commit SHA or digest pins`},
		{"an action in a path at a tag", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    steps:\n      - run: 'true'\n      - uses: github/codeql-action/init@v3\n",
			`job a step 2 uses "github/codeql-action/init@v3", which no full commit SHA or digest pins`},
		{"a remote workflow at a tag", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    uses: other/repo/.github/workflows/x.yml@v1\n",
			`job a uses "other/repo/.github/workflows/x.yml@v1", which no full commit SHA or digest pins`},
		{"runner in a job's env", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    env:\n      WORK: ${{ runner.temp }}/release\n",
			"job a env WORK names the runner context, which GitHub does not allow in a job's env"},
		{"steps in a job's env", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    env:\n      X: ${{ github.sha }}-${{ steps.s.outputs.o }}\n",
			"job a env X names the steps context"},
		{"runner in an index", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    env:\n      X: ${{ matrix[runner.os] }}\n",
			"job a env X names the runner context"},
		{"runner in the workflow's env", "on: workflow_dispatch\npermissions: {}\nenv:\n  OS: ${{ runner.os }}\njobs: {}\n",
			"env OS names the runner context, which GitHub does not allow in a workflow's env"},
		{"matrix in the workflow's env", "on: workflow_dispatch\npermissions: {}\nenv:\n  OS: ${{ matrix.os }}\njobs: {}\n",
			"env OS names the matrix context"},
		{"steps in a job's if", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    if: steps.s.outputs.o == 'x'\n",
			"job a if names the steps context, which GitHub does not allow in a job's if"},
		{"env in a job's if", "on: workflow_dispatch\npermissions: {}\njobs:\n  a:\n    runs-on: ubuntu-latest\n    if: ${{ !inputs.dry_run && env.X == 'y' }}\n",
			"job a if names the env context"},
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

// pinnedAction is a uses: that names exactly what runs: an action or
// workflow of another repository at a full commit SHA, which its owner
// cannot move as they can a tag or a branch, or a container image at its
// digest.
var pinnedAction = regexp.MustCompile(`^([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+(/[^@\s]+)?@[0-9a-f]{40}|docker://[^@\s]+@sha256:[0-9a-f]{64})$`)

// The contexts GitHub admits in a workflow's env, a job's env and a job's
// if (docs.github.com, "Contexts", "Context availability"). It refuses
// to parse a workflow that names any other there, so a dispatch of it is
// refused before a run exists. runner, env, steps and job are a step's
// alone.
var (
	workflowEnvContexts = []string{"github", "inputs", "secrets", "vars"}
	jobEnvContexts      = []string{"github", "inputs", "matrix", "needs", "secrets", "strategy", "vars"}
	jobIfContexts       = []string{"github", "inputs", "needs", "vars"}
)

// expression is a ${{ }} in a workflow's value.
var expression = regexp.MustCompile(`\$\{\{(.*?)\}\}`)

// quotedLiteral is a string in an expression, whose text names no context.
var quotedLiteral = regexp.MustCompile(`'(?:[^']|'')*'`)

// identifier is a name in an expression. It reads a context when it is
// not itself a property (no . before it) and a property or an index
// follows it.
var identifier = regexp.MustCompile(`[A-Za-z_][A-Za-z0-9_-]*`)

// unavailable is each context src's expressions read that allowed does
// not hold. whole says src is an expression even without ${{ }}, as a
// job's if is.
func unavailable(src string, whole bool, allowed []string) []string {
	var exprs []string
	for _, m := range expression.FindAllStringSubmatch(src, -1) {
		exprs = append(exprs, m[1])
	}
	if exprs == nil && whole {
		exprs = []string{src}
	}
	var out []string
	for _, e := range exprs {
		e = quotedLiteral.ReplaceAllString(e, "''")
		for _, loc := range identifier.FindAllStringIndex(e, -1) {
			if loc[0] > 0 && e[loc[0]-1] == '.' {
				continue
			}
			rest := strings.TrimLeft(e[loc[1]:], " \t")
			if rest == "" || (rest[0] != '.' && rest[0] != '[') {
				continue
			}
			if c := e[loc[0]:loc[1]]; !slices.Contains(allowed, c) && !slices.Contains(out, c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// pinned is whether a uses: names exactly what runs: a path in this
// repository, reviewed with it, or a pinnedAction.
func pinned(uses string) bool {
	return strings.HasPrefix(uses, "./") || pinnedAction.MatchString(uses)
}

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
	env, _ := w.Get("env")
	envObj, _ := env.(value.Object)
	for key, v := range envObj.All() {
		for _, c := range unavailable(text(v), false, workflowEnvContexts) {
			out = append(out, "env "+key+" names the "+c+" context, which GitHub does not allow in a workflow's env")
		}
	}
	jobs, _ := w.Get("jobs")
	jobsObj, _ := jobs.(value.Object)
	for name, job := range jobsObj.All() {
		j, _ := job.(value.Object)
		if env, ok := j.Get("env"); ok {
			envObj, _ := env.(value.Object)
			for key, v := range envObj.All() {
				for _, c := range unavailable(text(v), false, jobEnvContexts) {
					out = append(out, "job "+name+" env "+key+" names the "+c+" context, which GitHub does not allow in a job's env")
				}
			}
		}
		if cond, ok := j.Get("if"); ok {
			for _, c := range unavailable(text(cond), true, jobIfContexts) {
				out = append(out, "job "+name+" if names the "+c+" context, which GitHub does not allow in a job's if")
			}
		}
		runsOn, hasRunner := j.Get("runs-on")
		if uses, ok := j.Get("uses"); ok {
			if s := text(uses); !pinned(s) {
				out = append(out, "job "+name+" uses "+quote(s)+", which no full commit SHA or digest pins")
			}
			if s := text(uses); !localWorkflow.MatchString(s) {
				out = append(out, "job "+name+" calls workflow "+quote(s)+", which is not one of this repository's")
			}
			continue
		}
		steps, _ := j.Get("steps")
		stepsArr, _ := steps.(value.Array)
		for i, step := range stepsArr.All() {
			so, _ := step.(value.Object)
			if uses, ok := so.Get("uses"); ok && !pinned(text(uses)) {
				out = append(out, "job "+name+" step "+strconv.Itoa(i+1)+" uses "+quote(text(uses))+", which no full commit SHA or digest pins")
			}
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

// texts is the text of each element of an array, or nil for any other
// value.
func texts(v value.Value) []string {
	a, ok := v.(value.Array)
	if !ok {
		return nil
	}
	var out []string
	for _, e := range a.All() {
		out = append(out, text(e))
	}
	return out
}

// actionName is a uses: without the ref after its @, which
// workflowProblems holds to a full commit SHA.
func actionName(uses string) string {
	name, _, _ := strings.Cut(uses, "@")
	return name
}

func quote(s string) string { return `"` + s + `"` }

// TestCIWorkflow holds ci.yml to ADR-013 D5: it runs on every pull
// request and on every push to main (items 2 and 5), and on dispatch. Its
// pull_request trigger names no branch, so a pull request into any branch
// runs it, and includes the edited type, so an edit to a pull request's
// title or body runs the co-author check again. Its token is read-only
// (ADR-013 D5 item 4's default, declared in the workflow itself), and its
// jobs are item 2's, each on the platform it names, running what it names,
// with internal/conformance's tests split out of the run of the others and
// run on their own, as R-W1-4 splits them: a job that skipped them and did
// not run them would run them nowhere. The install jobs are D6 item 8's
// real installs, one on each platform install.sh serves. Its concurrency
// cancels a pull request's superseded run and never a push run on main.
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
	if got := slices.Sorted(slices.Values(triggers(on))); !slices.Equal(got, []string{"pull_request", "push", "workflow_dispatch"}) {
		t.Errorf("ci.yml is triggered by %q, want pull_request, push and workflow_dispatch", got)
	}
	onObj, _ := on.(value.Object)
	pr, _ := must(onObj.Get("pull_request")).(value.Object)
	if got := pr.Names(); !slices.Equal(got, []string{"types"}) {
		t.Errorf("ci.yml's pull_request trigger has %q; want types alone, so every pull request runs it", got)
	}
	if got := strings.Join(texts(must(pr.Get("types"))), " "); got != "opened synchronize reopened edited" {
		t.Errorf("ci.yml's pull_request trigger has the types %q, want opened, synchronize, reopened and edited", got)
	}
	push, _ := must(onObj.Get("push")).(value.Object)
	if got := push.Names(); !slices.Equal(got, []string{"branches"}) {
		t.Errorf("ci.yml's push trigger has %q; want branches alone", got)
	}
	if got := texts(must(push.Get("branches"))); !slices.Equal(got, []string{"main"}) {
		t.Errorf("ci.yml's push trigger runs on the branches %q, want main alone", got)
	}
	perms, _ := w.Get("permissions")
	readOnly, _ := value.NewObject(value.Member{Name: "contents", Value: mustString("read")})
	if !value.Equal(perms, readOnly) {
		t.Errorf("ci.yml's top-level permissions are not contents: read alone")
	}
	// A push run on main is the record of what merged, so no newer run
	// may cancel it, while a pull request's superseded run is waste. Both
	// halves hold that: a pull request's runs share the group of its
	// number, and every other run's group is its own run's id, since a
	// newer run in a group replaces the one pending in it even when
	// nothing in progress is cancelled; and only a pull request's run
	// cancels the run in progress in its group.
	conc, _ := must(w.Get("concurrency")).(value.Object)
	if got := conc.Names(); !slices.Equal(got, []string{"cancel-in-progress", "group"}) {
		t.Errorf("ci.yml's concurrency has %q, want group and cancel-in-progress alone", got)
	}
	if got := text(must(conc.Get("group"))); got != "ci-${{ github.event.pull_request.number || github.run_id }}" {
		t.Errorf("ci.yml's concurrency group is %q, want a pull request's number, or the run's own id for a push or a dispatch", got)
	}
	if got := text(must(conc.Get("cancel-in-progress"))); got != "${{ github.event_name == 'pull_request' }}" {
		t.Errorf("ci.yml's concurrency cancels in progress when %q, want only a pull request's run, so a push run on main is never cancelled", got)
	}
	jobs, _ := w.Get("jobs")
	jobsObj, _ := jobs.(value.Object)
	want := []struct {
		job, runsOn string
		runs        []string
	}{
		{"wave", "ubuntu-latest", []string{"skip=$(.github/scripts/conformance.sh skip)", `GOFLAGS="-skip=$skip" METAIS_GATE_P=$(nproc) .claude/skills/verification-gate/gate.sh wave`, ".github/scripts/conformance.sh run"}},
		{"macos", "macos-latest", []string{"go build ./...", "CGO_ENABLED=0 go build -trimpath ./cmd/metais", "skip=$(.github/scripts/conformance.sh skip)", `go test -race -count=1 ./... -skip "$skip"`, ".github/scripts/conformance.sh run"}},
		{"dco", "ubuntu-latest", []string{".github/scripts/dco.sh", `.github/scripts/coauthor.sh "$BASE" HEAD "$RUNNER_TEMP/squash.txt"`, `.github/scripts/coauthor.sh "$BASE" HEAD` + "\n"}},
		{"shell", "ubuntu-latest", []string{"xargs -0 shellcheck"}},
		{"site", "ubuntu-latest", []string{"site/build.sh", "go test -race -count=1 ./site/... ./internal/tools/site/..."}},
		{"install-linux", "ubuntu-latest", []string{`.github/scripts/install-test.sh "$RUNNER_TEMP/install"`}},
		{"install-macos", "macos-latest", []string{`.github/scripts/install-test.sh "$RUNNER_TEMP/install"`}},
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

// jobsCondition matches a job's if: that the jobs input selects it by:
// one or more comparisons of inputs.jobs with a value, joined by ||.
var jobsCondition = regexp.MustCompile(`^\$\{\{ inputs\.jobs == '[^']*'( \|\| inputs\.jobs == '[^']*')* \}\}$`)

// jobsValue matches one comparison of jobsCondition, and its value.
var jobsValue = regexp.MustCompile(`inputs\.jobs == '([^']*)'`)

// TestCIJobsInput holds the selection ci.yml's jobs input makes: a run
// with no input, as a pull request's or a push's is, and a run of
// all run every job, and any other option runs the job of its own name
// and the jobs whose names begin with it and a hyphen, so install runs
// install-linux and install-macos, and nothing else. Every option is a
// job's name or a group of them, and every job can be run alone.
// TestCIWorkflow holds what each job runs.
func TestCIJobsInput(t *testing.T) {
	src, err := os.ReadFile(ciWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "ci"}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatal(fds[0].Message())
	}
	w := doc.Value.(value.Object)
	on, _ := must(w.Get("on")).(value.Object)
	dispatch, _ := must(on.Get("workflow_dispatch")).(value.Object)
	inputs, _ := must(dispatch.Get("inputs")).(value.Object)
	input, _ := must(inputs.Get("jobs")).(value.Object)
	optionsArr, _ := must(input.Get("options")).(value.Array)
	var options []string
	for _, o := range optionsArr.All() {
		options = append(options, text(o))
	}
	if def, _ := input.Get("default"); text(def) != "all" {
		t.Errorf("ci.yml's jobs input does not default to all")
	}

	jobs, _ := must(w.Get("jobs")).(value.Object)
	selects := map[string][]string{}
	for _, name := range jobs.Names() {
		job, _ := must(jobs.Get(name)).(value.Object)
		cond := text(must(job.Get("if")))
		if !jobsCondition.MatchString(cond) {
			t.Errorf("ci.yml's job %s runs if %q, which is not a selection by the jobs input", name, cond)
			continue
		}
		for _, m := range jobsValue.FindAllStringSubmatch(cond, -1) {
			selects[name] = append(selects[name], m[1])
		}
	}
	if len(selects) < 2 {
		t.Fatalf("ci.yml has %d jobs selected by the jobs input; the check is blind", len(selects))
	}

	for _, o := range append([]string{""}, options...) {
		var got, want []string
		for _, name := range jobs.Names() {
			if slices.Contains(selects[name], o) {
				got = append(got, name)
			}
			if o == "" || o == "all" || o == name || strings.HasPrefix(name, o+"-") {
				want = append(want, name)
			}
		}
		if len(want) == 0 {
			t.Errorf("ci.yml's jobs input has the option %q, which names no job", o)
		}
		if !slices.Equal(got, want) {
			t.Errorf("ci.yml's jobs input %q runs %q, want %q", o, got, want)
		}
	}
	for _, name := range jobs.Names() {
		if !slices.Contains(options, name) {
			t.Errorf("ci.yml's jobs input has no option that runs the job %s alone", name)
		}
		for _, v := range selects[name] {
			if v != "" && !slices.Contains(options, v) {
				t.Errorf("ci.yml's job %s runs for the jobs input %q, which is not one of its options", name, v)
			}
		}
	}
}

// TestReleaseWorkflow holds release.yml to ADR-013 D7 item 3. Ed
// dispatches it, naming the draft, and its release job runs in the release
// environment, whose approval is his release act. Its dry_run input turns
// off every step that pushes or publishes, and the pages job, and nothing
// else: prepare, which checks everything and pushes nothing, runs either
// way. Only the job that pushes may write the repository's contents, and
// only the step that pushes to the tap is given TAP_TOKEN. It builds no
// release and attests nothing (ADR-013 D7 items 2 and 4), and once the
// release is published it calls pages.yml with the released tag (ruling
// R-W2-1). release.sh's own test, internal/tools/release's
// TestReleaseScript, proves what each step does.
func TestReleaseWorkflow(t *testing.T) {
	src, err := os.ReadFile(releaseWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "release"}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatal(fds[0].Message())
	}
	w := doc.Value.(value.Object)
	on, _ := w.Get("on")
	if got := triggers(on); !slices.Equal(got, []string{"workflow_dispatch"}) {
		t.Errorf("release.yml is triggered by %q, not by workflow_dispatch alone", got)
	}
	dispatch, _ := on.(value.Object).Get("workflow_dispatch")
	inputs, _ := must(dispatch.(value.Object).Get("inputs")).(value.Object)
	if got := inputs.Names(); !slices.Equal(got, []string{"dry_run", "tag"}) {
		t.Errorf("release.yml's inputs are %q, want dry_run and tag", got)
	}
	dry, _ := must(inputs.Get("dry_run")).(value.Object)
	if typ, _ := dry.Get("type"); text(typ) != "boolean" {
		t.Errorf("release.yml's dry_run is not a boolean")
	}
	if def, _ := dry.Get("default"); def != value.Bool(false) {
		t.Errorf("release.yml's dry_run does not default to false")
	}
	if perms, _ := w.Get("permissions"); !value.Equal(perms, value.Object{}) {
		t.Errorf("release.yml's top-level permissions are not empty")
	}

	jobsObj, _ := must(w.Get("jobs")).(value.Object)
	if got := jobsObj.Names(); !slices.Equal(got, []string{"pages", "release"}) {
		t.Fatalf("release.yml's jobs are %q, want pages and release", got)
	}
	job, _ := must(jobsObj.Get("release")).(value.Object)
	if env, _ := job.Get("environment"); text(env) != "release" {
		t.Errorf("release.yml's release job does not run in the release environment")
	}
	if _, ok := job.Get("if"); ok {
		t.Errorf("release.yml's release job is conditional")
	}
	write, _ := value.NewObject(value.Member{Name: "actions", Value: mustString("read")}, value.Member{Name: "contents", Value: mustString("write")})
	if perms, _ := job.Get("permissions"); !value.Equal(perms, write) {
		t.Errorf("release.yml's release job's permissions are not contents: write and actions: read alone")
	}
	const dryOff = "${{ !inputs.dry_run"
	pushSteps := map[string]bool{"push-tags": false, "publish": false, "push-tap": false}
	prepared := false
	for _, s := range must(job.Get("steps")).(value.Array).All() {
		so, _ := s.(value.Object)
		run := text(must(so.Get("run")))
		cond, hasIf := so.Get("if")
		// The release job runs only the actions that check the
		// repository out and set up go: no builder, and no attestation.
		if uses, ok := so.Get("uses"); ok && !slices.Contains([]string{"actions/checkout", "actions/setup-go"}, actionName(text(uses))) {
			t.Errorf("release.yml's release job uses %s", text(uses))
		}
		if env, ok := so.Get("env"); ok {
			if tok, ok := env.(value.Object).Get("TAP_TOKEN"); ok && !strings.Contains(run, "push-tap") {
				t.Errorf("release.yml's step %q is given TAP_TOKEN: %s", run, text(tok))
			}
		}
		if strings.Contains(run, ".github/scripts/release.sh prepare ") {
			prepared = true
			if hasIf {
				t.Errorf("release.yml's prepare is conditional: %s", text(cond))
			}
			continue
		}
		pushes := strings.Contains(run, "git push") || strings.Contains(run, "gh release")
		for step := range pushSteps {
			if strings.Contains(run, ".github/scripts/release.sh "+step+" ") {
				pushSteps[step] = true
				pushes = true
			}
		}
		if pushes && (!hasIf || !strings.HasPrefix(text(cond), dryOff)) {
			t.Errorf("release.yml's step %q pushes or publishes in a dry run", run)
		}
	}
	if !prepared {
		t.Errorf("release.yml's release job does not run release.sh prepare")
	}
	for step, seen := range pushSteps {
		if !seen {
			t.Errorf("release.yml's release job does not run release.sh %s", step)
		}
	}

	pages, _ := must(jobsObj.Get("pages")).(value.Object)
	if uses, _ := pages.Get("uses"); text(uses) != "./.github/workflows/pages.yml" {
		t.Errorf("release.yml's pages job does not call pages.yml")
	}
	if needs, _ := pages.Get("needs"); text(needs) != "release" {
		t.Errorf("release.yml's pages job does not wait for the release job")
	}
	if cond, _ := pages.Get("if"); !strings.HasPrefix(text(cond), dryOff) {
		t.Errorf("release.yml's pages job runs in a dry run")
	}
	with, _ := must(pages.Get("with")).(value.Object)
	if tag, _ := with.Get("tag"); text(tag) != "${{ inputs.tag }}" {
		t.Errorf("release.yml's pages job is not given the released tag")
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

// TestCoAuthorCheck runs CI's co-author check over a repository of
// commits with and without a Co-authored-by line, and over a pull
// request's title and body, the squash merge's message (AGENTS.md, Git
// and pull requests; the ruling at PLAN-2026-10-metais-v1's T-A4). A line
// that opens with the key fails the change in any case and wherever it
// sits in the message, since GitHub's squash merge carries a commit's
// body into the squash message; the key named inside a line of prose does
// not. The change is the DCO check's: every commit reachable from the
// head and not from the base, a merge commit included, or the head alone
// when it is on the base.
func TestCoAuthorCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the co-author check runs on CI's linux runner")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("the co-author check reads git: %v", err)
	}
	script, err := filepath.Abs(coAuthorScript)
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
	git("checkout", "--quiet", "-b", "clean")
	commit("one" + signed)
	cleanHead := commit("two\n\nThis names no Co-authored-by: line of its own." + signed)
	git("checkout", "--quiet", "-b", "trailer")
	trailerHead := commit("three\n\nSigned-off-by: A Contributor <a@example.invalid>\nCo-authored-by: B <b@example.invalid>")
	git("checkout", "--quiet", "-b", "capitals", cleanHead)
	capitalsHead := commit("four\n\nCo-Authored-By: B <b@example.invalid>" + signed)
	git("checkout", "--quiet", "-b", "inbody", cleanHead)
	inBodyHead := commit("five\n\nco-authored-by: B <b@example.invalid>\n\nis carried into a squash body." + signed)
	git("checkout", "--quiet", "-b", "merge", cleanHead)
	git("merge", "--quiet", "--no-ff", "--no-edit", "-m", "a clean merge"+signed, "trailer")
	git("checkout", "--quiet", "main")
	onMain := commit("on main\n\nCo-authored-by: B <b@example.invalid>" + signed)

	dir := testenv.Dir(t)
	message := func(name, data string) string {
		p := filepath.Join(dir, name)
		write(t, p, data)
		return p
	}
	cleanPR := message("clean", "ci(x): do a thing (#1)\n\nA body that mentions Co-authored-by: in prose.\n\nSigned-off-by: A Contributor <a@example.invalid>\n")
	trailerPR := message("trailer", "ci(x): do a thing (#1)\n\nA body.\n\nSigned-off-by: A Contributor <a@example.invalid>\r\nCo-authored-by: B <b@example.invalid>\r\n")

	for _, c := range []struct {
		name string
		args []string
		code int
		says string
	}{
		{"no commit carries the line", []string{base, cleanHead}, 0, "no Co-authored-by line in 2 commit(s)"},
		{"a trailer", []string{base, trailerHead}, 1, "three carries a Co-authored-by line"},
		{"a trailer in capitals", []string{base, capitalsHead}, 1, "four carries a Co-authored-by line"},
		{"a line in the body", []string{base, inBodyHead}, 1, "five carries a Co-authored-by line"},
		{"a clean merge of a commit with a trailer", []string{base, "merge"}, 1, "three carries a Co-authored-by line"},
		{"the head on the base", []string{"main", onMain}, 1, "on main carries a Co-authored-by line"},
		{"a clean title and body", []string{base, cleanHead, cleanPR}, 0, "no Co-authored-by line in 2 commit(s) and the pull request's title and body"},
		{"a trailer in the body", []string{base, cleanHead, trailerPR}, 1, "the pull request's title and body carry a Co-authored-by line"},
		{"a body that cannot be read", []string{base, cleanHead, filepath.Join(dir, "nosuchfile")}, 2, "cannot read"},
		{"a base that does not exist", []string{"nosuchref", cleanHead}, 2, "cannot list the commits"},
		{"no head", []string{base}, 2, "usage"},
	} {
		cmd := exec.Command("sh", append([]string{script}, c.args...)...)
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
			t.Errorf("%s: coauthor.sh exited %d saying\n%s\nwant exit %d saying %q", c.name, code, out, c.code, c.says)
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

// TestSquashBodyCheck runs the check a merger runs on a squash merge's
// body before the merge (AGENTS.md, Git and pull requests), which CI's
// DCO check cannot see, since it reads the pull request's commits. A body
// passes when its last line that is not blank is a Signed-off-by: line
// naming a person and an email address, git parses that line as a
// trailer of the merged commit's message, and no line opens with
// Co-authored-by. An empty body, and a sign-off followed by a paragraph,
// each landed on main unsigned before the check existed.
func TestSquashBodyCheck(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the squash body check runs where a merger runs gh, on a POSIX shell")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("the squash body check reads git: %v", err)
	}
	script, err := filepath.Abs(squashBodyScript)
	if err != nil {
		t.Fatal(err)
	}
	dir := testenv.Dir(t)
	env := testenv.GitEnv(os.Environ())
	const signed = "Signed-off-by: A Contributor <a@example.invalid>"
	for _, c := range []struct {
		name, body string
		code       int
		says       string
	}{
		{"a body that ends with its sign-off", "What changed.\n\nWhy.\n\n" + signed + "\n", 0, "ends with its Signed-off-by trailer"},
		{"a body of the sign-off alone", signed, 0, "ends with its Signed-off-by trailer"},
		{"a sign-off beside another trailer", "What changed.\n\nReviewed-by: B <b@example.invalid>\n" + signed + "\n\n\n", 0, "ends with its Signed-off-by trailer"},
		{"an empty body", "", 1, "is empty"},
		{"a body of blank lines", "\n  \n\n", 1, "is empty"},
		{"a paragraph after the sign-off", "What changed.\n\n" + signed + "\n\nGenerated by a tool.\n", 1, "does not end with a Signed-off-by"},
		{"no sign-off", "What changed.\n", 1, "does not end with a Signed-off-by"},
		{"a sign-off with no address", "What changed.\n\nSigned-off-by: A Contributor\n", 1, "does not end with a Signed-off-by"},
		{"a sign-off git does not parse as a trailer", "What changed.\nMore prose.\nAnd more.\nAnd more still.\n" + signed + "\n", 1, "git parses no Signed-off-by"},
		{"a co-author", "What changed.\n\nco-authored-by: B <b@example.invalid>\n" + signed + "\n", 1, "carries a Co-authored-by line"},
	} {
		f := filepath.Join(dir, "body.txt")
		write(t, f, c.body)
		code, out := runScript(t, env, dir, script, f)
		if code != c.code || !strings.Contains(out, c.says) {
			t.Errorf("%s: squash-body.sh exited %d saying\n%s\nwant exit %d saying %q", c.name, code, out, c.code, c.says)
		}
	}
	if code, out := runScript(t, env, dir, script, filepath.Join(dir, "absent.txt")); code != 2 || !strings.Contains(out, "cannot read") {
		t.Errorf("a missing body: squash-body.sh exited %d saying\n%s\nwant exit 2 saying %q", code, out, "cannot read")
	}
}

// runScript runs a shell script with args in dir, and returns its exit
// code and what it printed.
func runScript(t *testing.T, env []string, dir, script string, args ...string) (int, string) {
	t.Helper()
	cmd := exec.Command("sh", append([]string{script}, args...)...)
	cmd.Dir = dir
	cmd.Env = env
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), string(out)
	} else if err != nil {
		t.Fatal(err)
	}
	return 0, string(out)
}

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
