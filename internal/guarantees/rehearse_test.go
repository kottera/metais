// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
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

// rehearseWorkflow runs the rehearsal's checks against a published
// release (ADR-013 D7 item 8).
const rehearseWorkflow = "../../.github/workflows/rehearse.yml"

// rehearseScript is the script each of its checks runs.
const rehearseScript = "../../.github/scripts/rehearse.sh"

// TestRehearseWorkflow holds rehearse.yml to the rehearsal's checks
// (ADR-013 D7 item 8, the Wave 3 brief of PLAN-2026-10-metais-public). It
// is dispatched with a version and by nothing else, since there is nothing
// to check before a candidate is published, and its token is read-only.
// Each route of ADR-013 D6 has a job on each clean machine the brief names,
// each job checks the version the route installed and runs metais skill
// with it, and the site job checks every page the brief lists against a
// checkout of the release's tag. No job waits on another or is skipped by a
// condition, so one route's refusal hides no other's. TestRehearseScript
// proves what each check refuses.
func TestRehearseWorkflow(t *testing.T) {
	src, err := os.ReadFile(rehearseWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "rehearse"}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatal(fds[0].Message())
	}
	w := doc.Value.(value.Object)
	on, _ := w.Get("on")
	if got := triggers(on); !slices.Equal(got, []string{"workflow_dispatch"}) {
		t.Errorf("rehearse.yml is triggered by %q, not by workflow_dispatch alone", got)
	}
	dispatch, _ := on.(value.Object).Get("workflow_dispatch")
	inputs, _ := must(dispatch.(value.Object).Get("inputs")).(value.Object)
	if got := inputs.Names(); !slices.Equal(got, []string{"version"}) {
		t.Errorf("rehearse.yml's inputs are %q, want version alone", got)
	}
	if req, _ := must(inputs.Get("version")).(value.Object).Get("required"); req != value.Bool(true) {
		t.Errorf("rehearse.yml's version input is not required")
	}
	readOnly, _ := value.NewObject(value.Member{Name: "contents", Value: mustString("read")})
	if perms, _ := w.Get("permissions"); !value.Equal(perms, readOnly) {
		t.Errorf("rehearse.yml's top-level permissions are not contents: read alone")
	}
	// The version reaches the scripts through the environment, never
	// through an expression in a script's text.
	if env, _ := must(w.Get("env")).(value.Object).Get("VERSION"); text(env) != "${{ inputs.version }}" {
		t.Errorf("rehearse.yml's VERSION is %q, not the version input", text(env))
	}

	const (
		latest   = `.github/scripts/rehearse.sh latest "$VERSION"`
		install  = `.github/scripts/rehearse.sh install "$VERSION"`
		skill    = ".github/scripts/rehearse.sh skill "
		version  = ".github/scripts/rehearse.sh version "
		goGet    = `go install "metais.org/metais/cmd/metais@v$VERSION"`
		goLatest = "go install metais.org/metais/cmd/metais@latest"
		manual   = `.github/scripts/rehearse.sh manual "$VERSION" `
	)
	installSh := []string{latest, install, skill}
	want := []struct {
		job, runsOn string
		runs        []string
	}{
		{"brew-macos", "macos-latest", []string{"brew install kottera/tap/metais", version, skill}},
		{"install-ubuntu", "ubuntu-latest", installSh},
		{"install-macos", "macos-latest", installSh},
		{"install-container", "ubuntu-latest", installSh},
		{"go-install", "ubuntu-latest", []string{goGet, version, skill, goLatest, version + `"$RUNNER_TEMP/latest/metais" "$VERSION"`}},
		{"manual-ubuntu", "ubuntu-latest", []string{manual, skill}},
		{"manual-macos", "macos-latest", []string{manual, skill}},
		{"site", "ubuntu-latest", []string{
			manual,
			".github/scripts/rehearse.sh ids release",
			".github/scripts/rehearse.sh go-import",
			".github/scripts/rehearse.sh reference ",
			".github/scripts/rehearse.sh spec release",
			`.github/scripts/rehearse.sh released release "$RUNNER_TEMP/metais/metais"`,
			".github/scripts/rehearse.sh pages release",
			".github/scripts/rehearse.sh kottera",
		}},
	}
	var names []string
	for _, j := range want {
		names = append(names, j.job)
	}
	jobs, _ := must(w.Get("jobs")).(value.Object)
	if got := jobs.Names(); !slices.Equal(got, slices.Sorted(slices.Values(names))) {
		t.Errorf("rehearse.yml's jobs are %q, want %q", got, names)
	}
	for _, j := range want {
		jo, _ := must(jobs.Get(j.job)).(value.Object)
		if l, _ := runners(must(jo.Get("runs-on"))); !slices.Equal(l, []string{j.runsOn}) {
			t.Errorf("rehearse.yml's job %s runs on %q, want %s", j.job, l, j.runsOn)
		}
		for _, key := range []string{"if", "needs"} {
			if _, ok := jo.Get(key); ok {
				t.Errorf("rehearse.yml's job %s has %s:, so another route's refusal can hide it", j.job, key)
			}
		}
		var run strings.Builder
		checks := 0
		for _, s := range must(jo.Get("steps")).(value.Array).All() {
			so, _ := s.(value.Object)
			if uses, ok := so.Get("uses"); ok && !slices.Contains([]string{"actions/checkout", "actions/setup-go"}, actionName(text(uses))) {
				t.Errorf("rehearse.yml's job %s uses %s", j.job, text(uses))
			}
			r, ok := so.Get("run")
			if !ok {
				continue
			}
			if strings.Contains(text(r), "${{") {
				t.Errorf("rehearse.yml's job %s runs an expression in a script: %s", j.job, text(r))
			}
			if _, ok := so.Get("name"); !ok {
				t.Errorf("rehearse.yml's job %s has a step with no name: %s", j.job, text(r))
			}
			run.WriteString(text(r) + "\n")
			// In the site job, every check after the release's metais
			// is fetched runs whether or not the one before it refused.
			if j.job == "site" && strings.Contains(text(r), "rehearse.sh") {
				checks++
				if cond, _ := so.Get("if"); checks > 1 && text(cond) != "${{ !cancelled() }}" {
					t.Errorf("rehearse.yml's site check %q does not run after another refused", text(r))
				}
			}
		}
		for _, r := range j.runs {
			if !strings.Contains(run.String(), r) {
				t.Errorf("rehearse.yml's job %s does not run %q", j.job, r)
			}
		}
	}

	// The container is an image with nothing installed, given install.sh's
	// prerequisite and nothing else.
	container, _ := must(jobs.Get("install-container")).(value.Object)
	if image := text(must(container.Get("container"))); !strings.HasPrefix(image, "ubuntu:") {
		t.Errorf("rehearse.yml's install-container runs in %q, not a bare ubuntu image", image)
	}
	// go install runs once through the proxy and once direct, through the
	// go-import page, with a module cache the first run did not fill.
	goJob, _ := must(jobs.Get("go-install")).(value.Object)
	var direct, proxy bool
	for _, s := range must(goJob.Get("steps")).(value.Array).All() {
		so, _ := s.(value.Object)
		if !strings.Contains(text(must(so.Get("run"))), goGet) {
			continue
		}
		env, _ := must(so.Get("env")).(value.Object)
		p, hasProxy := env.Get("GOPROXY")
		_, hasCache := env.Get("GOMODCACHE")
		switch {
		case !hasProxy:
			proxy = true
		case text(p) == "direct" && hasCache:
			direct = true
		}
	}
	if !proxy || !direct {
		t.Errorf("rehearse.yml's go-install runs go install through the proxy %t, and direct with its own module cache %t; want both", proxy, direct)
	}
	// For a release, go install ...@latest runs through the proxy into a
	// GOBIN and a module cache of its own, and the version it installed is
	// checked. @latest never resolves to a prerelease while a release
	// exists, so for a prerelease neither step runs; each runs whether or
	// not a step before it refused.
	const forRelease = "${{ !cancelled() && steps.kind.outputs.release == 'true' }}"
	var kind string
	var latestGot, latestChecked bool
	for _, s := range must(goJob.Get("steps")).(value.Array).All() {
		so, _ := s.(value.Object)
		r := text(must(so.Get("run")))
		cond, _ := so.Get("if")
		switch {
		case text(must(so.Get("id"))) == "kind":
			kind = r
			if text(cond) != "${{ !cancelled() }}" {
				t.Errorf("rehearse.yml's go-install classifies the version only if %s, not whether or not a step before it refused", text(cond))
			}
		case strings.Contains(r, goLatest):
			latestGot = true
			if text(cond) != forRelease {
				t.Errorf("rehearse.yml's go-install runs go install ...@latest if %s, want %s", text(cond), forRelease)
			}
			env, _ := must(so.Get("env")).(value.Object)
			if _, ok := env.Get("GOPROXY"); ok {
				t.Errorf("rehearse.yml's go-install runs go install ...@latest with GOPROXY set, not through the Go proxy")
			}
			if text(must(env.Get("GOBIN"))) != "${{ runner.temp }}/latest" || text(must(env.Get("GOMODCACHE"))) != "${{ runner.temp }}/latest-modcache" {
				t.Errorf("rehearse.yml's go-install runs go install ...@latest without a GOBIN and a module cache of its own")
			}
		case strings.Contains(r, `"$RUNNER_TEMP/latest/metais"`):
			latestChecked = true
			if text(cond) != forRelease {
				t.Errorf("rehearse.yml's go-install checks the version @latest installed if %s, want %s", text(cond), forRelease)
			}
		}
	}
	if !latestGot || !latestChecked {
		t.Errorf("rehearse.yml's go-install runs go install ...@latest %t and checks its version %t; want both", latestGot, latestChecked)
	}
	// The step that decides is run as the runner runs it: a version with a
	// prerelease suffix is not a release, and build metadata is not one.
	for v, want := range map[string]string{"1.0.0": "true", "1.10.2": "true", "1.0.0+build-1": "true", "1.0.0-rc.1": "false", "2.0.0-rc.1+b": "false"} {
		out := filepath.Join(testenv.Dir(t), "output")
		cmd := exec.Command("sh", "-c", kind)
		cmd.Env = []string{"VERSION=" + v, "GITHUB_OUTPUT=" + out}
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("rehearse.yml's go-install classifying %s: %v\n%s", v, err, b)
		}
		got, err := os.ReadFile(out)
		if err != nil {
			t.Fatal(err)
		}
		if string(got) != "release="+want+"\n" {
			t.Errorf("rehearse.yml's go-install classifies %s as %q, want release=%s", v, got, want)
		}
	}
	// The site's checks read what the release holds from its tag, checked
	// out with the whole history, so the spec-<M>.<m> tags that say which
	// revisions are released are there for rehearse.sh released to read.
	site, _ := must(jobs.Get("site")).(value.Object)
	tagged := false
	for _, s := range must(site.Get("steps")).(value.Array).All() {
		so, _ := s.(value.Object)
		if with, ok := so.Get("with"); ok {
			wo, _ := with.(value.Object)
			ref, _ := wo.Get("ref")
			path, _ := wo.Get("path")
			depth, _ := wo.Get("fetch-depth")
			tagged = tagged || (text(ref) == "${{ steps.tag.outputs.tag }}" && text(path) == "release" && value.Equal(depth, value.NumberFromInt(0)))
		}
	}
	if !tagged {
		t.Errorf("rehearse.yml's site job does not check the release's tag out into release with fetch-depth: 0")
	}
}

// rehearsal is a release, its site and kottera.com, served on 127.0.0.1
// for rehearse.sh to check: tree is the release's checkout, web the site's
// published tree, dl the release's downloads, metais a stand-in of the
// release's metais, and env the environment that points the script at
// them, with home as HOME.
type rehearsal struct {
	tree, web, dl, kottera, home string
	metais, archive              string
	env                          []string
}

// rehearsed is the release the fixture publishes.
const rehearsed = "1.0.0-rc.1"

// fakeMetais is a metais that reports $FAKE_REPORTS, or $FAKE_BUILT, or the
// version it was built with, implementing the revisions $FAKE_IMPLEMENTS
// lists as JSON strings, or 1.0; lists one command and one topic; and
// writes a skill stamped with its version, unless $FAKE_SKILL names another
// or is none.
const fakeMetais = `#!/bin/sh
v=${FAKE_REPORTS:-${FAKE_BUILT:-%s}}
case $1 in
--version) printf 'metais %%s\nspecification 1.0\nMETAIS is made by Kottera.\n' "$v" ;;
version) printf '{"ok":true,"result":{"tool":{"name":"metais","version":"%%s"},"specification":{"implements":[%%s]}}}\n' "$v" "${FAKE_IMPLEMENTS:-"\"1.0\""}" ;;
help) echo '{"ok":true,"result":{"commands":[{"name":"render"}],"topics":[{"name":"templates"}]}}' ;;
skill)
	[ "${FAKE_SKILL:-}" = none ] && exit 0
	mkdir -p .claude/skills/metais
	printf '# metais\n<!-- metais skill: version %%s, sha256 00 -->\n' "${FAKE_SKILL:-$v}" >.claude/skills/metais/SKILL.md
	;;
esac
`

// fakeInstall is an install.sh that installs, into ~/.local/bin, a metais
// of METAIS_VERSION that runs the fakeMetais at %s. With no METAIS_VERSION
// it refuses as install.sh does with no stable release, or installs
// $FAKE_LATEST when that is set.
const fakeInstall = `#!/bin/sh
set -eu
v=${METAIS_VERSION:-${FAKE_LATEST:-}}
if [ -z "$v" ]; then
	echo "install.sh: metais has no stable release yet, only prereleases. Set METAIS_VERSION to one" >&2
	exit 1
fi
mkdir -p "$HOME/.local/bin"
printf '#!/bin/sh\nFAKE_BUILT=%%s exec %%s "$@"\n' "${v#v}" '%s' >"$HOME/.local/bin/metais"
chmod 755 "$HOME/.local/bin/metais"
echo "installed metais ${v#v}"
`

func newRehearsal(t *testing.T) *rehearsal {
	t.Helper()
	var os_, arch string
	switch runtime.GOOS {
	case "darwin", "linux":
		os_ = runtime.GOOS
	default:
		t.Skipf("the rehearsal's checks run on macOS and Linux, and this is %s", runtime.GOOS)
	}
	switch runtime.GOARCH {
	case "amd64", "arm64":
		arch = runtime.GOARCH
	default:
		t.Skipf("the release has no archive for %s", runtime.GOARCH)
	}
	r := &rehearsal{tree: testenv.Dir(t), web: testenv.Dir(t), dl: testenv.Dir(t), kottera: testenv.Dir(t), home: testenv.Dir(t)}
	metais := fmt.Sprintf(fakeMetais, rehearsed)
	r.metais = filepath.Join(testenv.Dir(t), "metais")
	write(t, r.metais, metais)
	if err := os.Chmod(r.metais, 0o755); err != nil {
		t.Fatal(err)
	}
	installSh := fmt.Sprintf(fakeInstall, r.metais)

	files := map[string]string{
		"tree/spec/README.md":                        "# METAIS\n",
		"tree/spec/01-packages.md":                   "# Packages\n",
		"tree/spec/schemas/1.0/manifest.schema.json": `{"$id": "https://metais.org/schemas/1.0/manifest.schema.json"}`,
		"tree/docs/cli/schemas/metais.render.1.json": `{"$id": "https://metais.org/cli/schemas/metais.render.1.json"}`,
		"tree/site/public/install.sh":                installSh,
		"web/schemas/1.0/manifest.schema.json":       `{"$id": "https://metais.org/schemas/1.0/manifest.schema.json"}`,
		"web/cli/schemas/metais.render.1.json":       `{"$id": "https://metais.org/cli/schemas/metais.render.1.json"}`,
		"web/index.html":                             "<title>METAIS</title>",
		"web/install/index.html":                     "<title>Install</title>",
		"web/install.sh":                             installSh,
		"web/reference/index.html":                   "<title>Reference</title>",
		"web/reference/skill/index.html":             "<title>Skill</title>",
		"web/reference/commands/render/index.html":   "<title>render</title>",
		"web/reference/topics/templates/index.html":  "<title>templates</title>",
		"web/spec/draft/index.html":                  `<meta name="robots" content="noindex"><title>METAIS</title>`,
		"web/spec/draft/01-packages/index.html":      `<meta name="robots" content="noindex"><title>Packages</title>`,
		"web/spec/1.0/index.html":                    "<title>METAIS</title>",
		"web/spec/1.0/01-packages/index.html":        "<title>Packages</title>",
		"web/spec/index.html":                        `<meta http-equiv="refresh" content="0; url=/spec/1.0/"><title>The METAIS specification</title>`,
		"web/metais/index.html":                      goImportPage,
		"web/metais/cmd/metais/index.html":           goImportPage,
		"kottera/index.html":                         "<title>Kottera</title>",
	}
	for name, data := range files {
		dir, rest, _ := strings.Cut(name, "/")
		root := map[string]string{"tree": r.tree, "web": r.web, "kottera": r.kottera}[dir]
		write(t, filepath.Join(root, rest), data)
	}

	// The release's archive of this platform, and SHA256SUMS.
	r.archive = "metais_" + rehearsed + "_" + os_ + "_" + arch + ".tar.gz"
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for _, f := range []struct{ name, data string }{{"metais", metais}, {"LICENSE", "Apache-2.0\n"}} {
		if err := tw.WriteHeader(&tar.Header{Name: f.name, Mode: 0o755, Size: int64(len(f.data))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(f.data)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	rel := filepath.Join(r.dl, "v"+rehearsed)
	write(t, filepath.Join(rel, r.archive), buf.String())
	other := "metais_" + rehearsed + "_windows_amd64.tar.gz"
	write(t, filepath.Join(rel, "SHA256SUMS"), fmt.Sprintf("%x  %s\n%x  %s\n", sha256.Sum256(buf.Bytes()), r.archive, sha256.Sum256(nil), other))

	// Served as GitHub Pages serves a site: a directory is its index.html,
	// and a directory without one is not found, not listed.
	serve := func(dir string) string {
		files := http.FileServer(http.Dir(dir))
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if strings.HasSuffix(req.URL.Path, "/") {
				if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(req.URL.Path), "index.html")); err != nil {
					http.NotFound(w, req)
					return
				}
			}
			files.ServeHTTP(w, req)
		}))
		t.Cleanup(s.Close)
		return s.URL
	}
	for _, kv := range hostEnviron {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && !strings.HasPrefix(k, "METAIS_") && !strings.HasPrefix(k, "FAKE_") {
			r.env = append(r.env, kv)
		}
	}
	r.env = append(testenv.GitEnv(r.env), "HOME="+r.home,
		"METAIS_REHEARSE_SITE="+serve(r.web),
		"METAIS_REHEARSE_DOWNLOAD="+serve(r.dl),
		"METAIS_REHEARSE_KOTTERA="+serve(r.kottera))
	return r
}

// goImportPage is a go-import page as the site serves it.
const goImportPage = `<meta name="go-import" content="metais.org/metais git https://github.com/kottera/metais">` + "\n" +
	`<meta name="go-source" content="metais.org/metais https://github.com/kottera/metais">` + "\n"

// TestRehearseScript runs each of rehearse.sh's checks against a release,
// its site and kottera.com served on 127.0.0.1, and requires each to pass
// against them as published and to refuse each defect it is there to find,
// saying what it found. metais and install.sh are stand-ins that report
// the versions the case gives them; the workflow's run installs the real
// ones.
func TestRehearseScript(t *testing.T) {
	for _, tool := range []string{"curl", "jq"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("rehearse.sh runs %s, as GitHub's runners have it: %v", tool, err)
		}
	}
	script, err := filepath.Abs(rehearseScript)
	if err != nil {
		t.Fatal(err)
	}
	checker := []string{"sha256sum", "--version"}
	if runtime.GOOS == "darwin" {
		checker = []string{"shasum", "-a", "256", "--version"}
	}
	checkerRuns := exec.Command(checker[0], checker[1:]...).Run() == nil

	rm := func(path string) func(*rehearsal) {
		return func(r *rehearsal) {
			if err := os.Remove(filepath.Join(r.web, path)); err != nil {
				t.Fatal(err)
			}
		}
	}
	put := func(root func(*rehearsal) string, path, data string) func(*rehearsal) {
		return func(r *rehearsal) { write(t, filepath.Join(root(r), path), data) }
	}
	web := func(r *rehearsal) string { return r.web }
	tree := func(r *rehearsal) string { return r.tree }
	// git runs git in the release's checkout.
	git := func(r *rehearsal, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = r.tree
		cmd.Env = append(testenv.GitEnv(os.Environ()),
			"GIT_AUTHOR_NAME=A Contributor", "GIT_AUTHOR_EMAIL=a@example.invalid",
			"GIT_COMMITTER_NAME=A Contributor", "GIT_COMMITTER_EMAIL=a@example.invalid")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	// tags makes the release's checkout a repository whose commit is tagged
	// each of tags, after each setup of before.
	tags := func(tags []string, before ...func(*rehearsal)) func(*rehearsal) {
		return func(r *rehearsal) {
			for _, f := range before {
				f(r)
			}
			git(r, "init", "--quiet", ".")
			git(r, "add", ".")
			git(r, "commit", "--quiet", "-m", "the release")
			for _, tag := range tags {
				git(r, "tag", tag)
			}
		}
	}
	// commit commits the release's checkout as it stands, after its tags:
	// a commit no spec-<M>.<m> tag names.
	commit := func(r *rehearsal) {
		git(r, "add", ".")
		git(r, "commit", "--quiet", "-m", "after the release")
	}
	// then runs each setup in turn.
	then := func(fs ...func(*rehearsal)) func(*rehearsal) {
		return func(r *rehearsal) {
			for _, f := range fs {
				f(r)
			}
		}
	}
	released := tags([]string{"spec-1.0"})
	const stable = "FAKE_REPORTS=1.0.0"

	for _, c := range []struct {
		name string
		// args are rehearse.sh's, where {metais} names the case's metais
		// and {dir} a directory that does not exist
		args []string
		// first are the arguments of a check run before args, in the same
		// HOME, that must pass: a step of the same job the workflow runs
		// before this one
		first  []string
		setup  func(*rehearsal)
		env    []string
		code   int
		says   string
		manual bool // needs the platform's checker of SHA256SUMS
	}{
		{name: "no check", args: nil, code: 2, says: "usage: rehearse.sh install"},
		{name: "an unknown check", args: []string{"render"}, code: 2, says: "usage: rehearse.sh"},
		{name: "not a version", args: []string{"version", "{metais}", "latest"}, code: 1, says: `FAIL: "latest" is not a metais version`},

		{name: "version", args: []string{"version", "{metais}", rehearsed}, says: "ok: {metais} reports " + rehearsed},
		{name: "version with its v", args: []string{"version", "{metais}", "v" + rehearsed}, says: "ok: {metais} reports " + rehearsed},
		{name: "version, another reported", args: []string{"version", "{metais}", rehearsed}, env: []string{"FAKE_REPORTS=1.0.0-rc.2"},
			code: 1, says: `FAIL: {metais} --version reports "metais 1.0.0-rc.2", want "metais ` + rehearsed + `"`},

		{name: "install", args: []string{"install", rehearsed}, says: "ok: {installed} reports " + rehearsed},
		{name: "install, another version installed", args: []string{"install", rehearsed}, env: []string{"FAKE_REPORTS=0.9.0"},
			code: 1, says: `FAIL: {installed} --version reports "metais 0.9.0"`},
		{name: "install, install.sh missing", args: []string{"install", rehearsed}, setup: rm("install.sh"),
			code: 1, says: "FAIL: install.sh did not install {installed}"},
		// A stable release's job runs latest first, which installs the
		// same version's own bytes, so a correct install changes nothing
		// in the file it writes.
		{name: "install, the same release installed by latest", first: []string{"latest", "1.0.0"}, args: []string{"install", "1.0.0"},
			env: []string{"FAKE_LATEST=1.0.0"}, says: "ok: {installed} reports 1.0.0"},
		{name: "install, install.sh writing nothing after latest", first: []string{"latest", "1.0.0"}, args: []string{"install", "1.0.0"},
			env: []string{"FAKE_LATEST=1.0.0"}, setup: func(r *rehearsal) {
				p := filepath.Join(r.web, "install.sh")
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				write(t, p, "if [ -n \"${METAIS_VERSION:-}\" ]; then echo \"installed metais $METAIS_VERSION\"; exit 0; fi\n"+string(b))
			},
			code: 1, says: "FAIL: install.sh did not install {installed}"},
		{name: "install, install.sh writing nothing", args: []string{"install", rehearsed}, setup: put(web, "install.sh", "echo installed metais\n"),
			code: 1, says: "FAIL: install.sh did not install {installed}"},
		{name: "install, install.sh failing", args: []string{"install", rehearsed}, setup: put(web, "install.sh", "exit 3\n"),
			code: 1, says: "FAIL: install.sh exited 3"},

		{name: "latest, refused", args: []string{"latest", rehearsed}, says: "ok: install.sh refuses, naming METAIS_VERSION, and installs nothing"},
		{name: "latest, installed with no stable release", args: []string{"latest", rehearsed}, env: []string{"FAKE_LATEST=" + rehearsed},
			code: 1, says: "FAIL: install.sh succeeded with no stable release"},
		{name: "latest, refused without naming METAIS_VERSION", args: []string{"latest", rehearsed},
			setup: put(web, "install.sh", "echo 'install.sh: metais has no stable release yet' >&2; exit 1\n"),
			code:  1, says: `FAIL: install.sh's refusal does not say "METAIS_VERSION"`},
		{name: "latest, refused for another reason", args: []string{"latest", rehearsed},
			setup: put(web, "install.sh", "echo 'install.sh: GitHub answered HTTP 403. Set METAIS_VERSION' >&2; exit 1\n"),
			code:  1, says: `FAIL: install.sh's refusal does not say "no stable release"`},
		{name: "latest, a release installed", args: []string{"latest", "1.0.0"}, env: []string{"FAKE_LATEST=1.0.0"}, says: "ok: {installed} reports 1.0.0"},
		{name: "latest, a release refused", args: []string{"latest", "1.0.0"}, code: 1, says: "FAIL: install.sh exited 1"},
		{name: "latest, a release installing another version", args: []string{"latest", "1.0.0"}, env: []string{"FAKE_LATEST=1.0.0", "FAKE_REPORTS=0.9.0"},
			code: 1, says: `FAIL: {installed} --version reports "metais 0.9.0", want "metais 1.0.0"`},

		{name: "manual", args: []string{"manual", rehearsed, "{dir}"}, manual: true, says: "ok: {unpacked} reports " + rehearsed},
		{name: "manual, the archive altered", args: []string{"manual", rehearsed, "{dir}"}, manual: true,
			setup: func(r *rehearsal) {
				p := filepath.Join(r.dl, "v"+rehearsed, r.archive)
				b, err := os.ReadFile(p)
				if err != nil {
					t.Fatal(err)
				}
				write(t, p, string(b)+"\x00")
			},
			code: 1, says: "FAIL: " + checker[0] + " refused the archive"},
		{name: "manual, the archive's line missing", args: []string{"manual", rehearsed, "{dir}"}, manual: true,
			setup: func(r *rehearsal) {
				write(t, filepath.Join(r.dl, "v"+rehearsed, "SHA256SUMS"), fmt.Sprintf("%x  metais_%s_windows_amd64.tar.gz\n", sha256.Sum256(nil), rehearsed))
			},
			code: 1, says: "FAIL: " + checker[0]},
		{name: "manual, no such release", args: []string{"manual", "1.0.0-rc.9", "{dir}"}, code: 1, says: "FAIL: could not download metais_1.0.0-rc.9_"},

		{name: "skill", args: []string{"skill", "{metais}", rehearsed, "{dir}"}, says: "ok: metais skill wrote .claude/skills/metais/SKILL.md, of version " + rehearsed},
		{name: "skill, another version's", args: []string{"skill", "{metais}", rehearsed, "{dir}"}, env: []string{"FAKE_SKILL=0.9.0"},
			code: 1, says: "FAIL: .claude/skills/metais/SKILL.md does not end naming version " + rehearsed},
		{name: "skill, none written", args: []string{"skill", "{metais}", rehearsed, "{dir}"}, env: []string{"FAKE_SKILL=none"},
			code: 1, says: "FAIL: metais skill did not write .claude/skills/metais/SKILL.md"},

		{name: "ids", args: []string{"ids", "{tree}"}, says: "ok: 2 of 2 schemas' $ids resolve as JSON with their own $id"},
		{name: "ids, one not served", args: []string{"ids", "{tree}"}, setup: rm("cli/schemas/metais.render.1.json"),
			code: 1, says: "FAIL: https://metais.org/cli/schemas/metais.render.1.json answered 404"},
		{name: "ids, one added since the site was built", args: []string{"ids", "{tree}"},
			setup: put(tree, "spec/schemas/1.1/manifest.schema.json", `{"$id": "https://metais.org/schemas/1.1/manifest.schema.json"}`),
			code:  1, says: "FAIL: https://metais.org/schemas/1.1/manifest.schema.json answered 404"},
		{name: "ids, another schema served", args: []string{"ids", "{tree}"},
			setup: put(web, "schemas/1.0/manifest.schema.json", `{"$id": "https://metais.org/schemas/1.0/finding.schema.json"}`),
			code:  1, says: `FAIL: https://metais.org/schemas/1.0/manifest.schema.json serves JSON whose $id is "https://metais.org/schemas/1.0/finding.schema.json"`},
		{name: "ids, not JSON", args: []string{"ids", "{tree}"}, setup: put(web, "schemas/1.0/manifest.schema.json", "<html>"),
			code: 1, says: `serves JSON whose $id is "(not JSON)"`},
		{name: "ids, served as HTML", args: []string{"ids", "{tree}"},
			setup: func(r *rehearsal) {
				p := filepath.Join(r.web, "schemas/1.0/manifest.schema.json")
				if err := os.Remove(p); err != nil {
					t.Fatal(err)
				}
				write(t, filepath.Join(p, "index.html"), `{"$id": "https://metais.org/schemas/1.0/manifest.schema.json"}`)
			},
			code: 1, says: "answered 200 text/html"},
		{name: "ids, a schema with none", args: []string{"ids", "{tree}"}, setup: put(tree, "docs/cli/schemas/metais.x.1.json", `{"type": "object"}`),
			code: 1, says: `FAIL: docs/cli/schemas/metais.x.1.json has no $id under https://metais.org/: ""`},

		{name: "go-import", args: []string{"go-import"}, says: "ok: 2 of 2 go-import pages"},
		{name: "go-import, another repository", args: []string{"go-import"},
			setup: put(web, "metais/cmd/metais/index.html", `<meta name="go-import" content="metais.org/metais git https://github.com/other/metais">`),
			code:  1, says: "FAIL: metais.org/metais/cmd/metais answered 200 without the go-import tag"},
		{name: "go-import, a page missing", args: []string{"go-import"}, setup: rm("metais/index.html"),
			code: 1, says: "FAIL: metais.org/metais answered"},

		{name: "reference", args: []string{"reference", "{metais}"}, says: "ok: 4 of 4 reference pages serve HTML"},
		{name: "reference, a topic's page missing", args: []string{"reference", "{metais}"}, setup: rm("reference/topics/templates/index.html"),
			code: 1, says: "/reference/topics/templates/ answered 404"},

		{name: "spec", args: []string{"spec", "{tree}"}, says: "ok: 2 of 2 chapters of the draft specification serve HTML, unindexed"},
		{name: "spec, a chapter missing", args: []string{"spec", "{tree}"}, setup: put(tree, "spec/02-schemas.md", "# Schemas\n"),
			code: 1, says: "/spec/draft/02-schemas/ answered 404"},
		{name: "spec, indexed", args: []string{"spec", "{tree}"}, setup: put(web, "spec/draft/01-packages/index.html", "<title>Packages</title>"),
			code: 1, says: "/spec/draft/01-packages/ does not ask not to be indexed"},

		{name: "released", args: []string{"released", "{tree}", "{metais}"}, setup: released, env: []string{stable},
			says: "ok: 3 of 3 pages of the released specification, 1.0, serve HTML, indexed, and /spec/ leads to 1.0"},
		{name: "released, a prerelease of a released revision", args: []string{"released", "{tree}", "{metais}"}, setup: released,
			says: "ok: 3 of 3 pages of the released specification, 1.0"},
		{name: "released, a prerelease of the draft", args: []string{"released", "{tree}", "{metais}"}, setup: tags(nil),
			says: "ok: " + rehearsed + " is a prerelease, and no revision it implements is released, so the site serves the draft alone"},
		{name: "released, a release of no released revision", args: []string{"released", "{tree}", "{metais}"}, setup: tags(nil), env: []string{stable},
			code: 1, says: "FAIL: 1.0.0 is a release, and no revision it implements is released"},
		{name: "released, a release of a revision not yet released", args: []string{"released", "{tree}", "{metais}"}, setup: released,
			env: []string{stable, `FAKE_IMPLEMENTS="1.0","1.1"`}, says: "1.1 is not released: "},
		{name: "released, a chapter missing", args: []string{"released", "{tree}", "{metais}"},
			setup: tags([]string{"spec-1.0"}, put(tree, "spec/02-schemas.md", "# Schemas\n")), env: []string{stable},
			code: 1, says: "/spec/1.0/02-schemas/ answered 404"},
		// A chapter committed after the tag is in HEAD and the working
		// tree, and not at spec-1.0, so reading either is refused.
		{name: "released, its chapters read from its tag", args: []string{"released", "{tree}", "{metais}"},
			setup: then(released, put(tree, "spec/02-schemas.md", "# Schemas\n"), commit), env: []string{stable},
			says: "ok: 3 of 3 pages of the released specification, 1.0"},
		{name: "released, its contents missing", args: []string{"released", "{tree}", "{metais}"}, setup: then(released, rm("spec/1.0/index.html")),
			env: []string{stable}, code: 1, says: "/spec/1.0/ answered 404"},
		{name: "released, unindexed", args: []string{"released", "{tree}", "{metais}"},
			setup: then(released, put(web, "spec/1.0/01-packages/index.html", `<meta name="robots" content="noindex"><title>Packages</title>`)),
			env:   []string{stable}, code: 1, says: "/spec/1.0/01-packages/ asks not to be indexed"},
		{name: "released, /spec/ leading to the draft", args: []string{"released", "{tree}", "{metais}"},
			setup: then(released, put(web, "spec/index.html", `<meta http-equiv="refresh" content="0; url=/spec/draft/">`)),
			env:   []string{stable}, code: 1, says: "/spec/ does not lead to /spec/1.0/"},
		{name: "released, /spec/ missing", args: []string{"released", "{tree}", "{metais}"}, setup: then(released, rm("spec/index.html")),
			env: []string{stable}, code: 1, says: "/spec/ answered 404"},
		{name: "released, /spec/ leading to an earlier revision", args: []string{"released", "{tree}", "{metais}"},
			setup: tags([]string{"spec-1.0", "spec-1.9", "spec-1.10"}), env: []string{stable},
			code: 1, says: "/spec/ does not lead to /spec/1.10/"},
		{name: "released, outside a repository", args: []string{"released", "{tree}", "{metais}"}, env: []string{stable},
			code: 1, says: "FAIL: could not list {tree}'s tags"},

		{name: "pages", args: []string{"pages", "{tree}"}, says: "ok: 3 of 3 pages serve, install.sh as the release holds it"},
		{name: "pages, install.sh not the release's", args: []string{"pages", "{tree}"}, setup: put(web, "install.sh", "#!/bin/sh\n"),
			code: 1, says: "/install.sh answered 200, not the release's site/public/install.sh"},
		{name: "pages, the install page missing", args: []string{"pages", "{tree}"}, setup: rm("install/index.html"),
			code: 1, says: "/install/ answered 404"},

		{name: "kottera", args: []string{"kottera"}, says: "title: Kottera"},
		{name: "kottera, not served", args: []string{"kottera"},
			setup: func(r *rehearsal) {
				if err := os.Remove(filepath.Join(r.kottera, "index.html")); err != nil {
					t.Fatal(err)
				}
			},
			code: 1, says: "/ answered 404"},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.manual && !checkerRuns {
				t.Skipf("%s does not run on this host, so the manual route cannot be checked here; CI's runners have it", strings.Join(checker, " "))
			}
			r := newRehearsal(t)
			if c.setup != nil {
				c.setup(r)
			}
			dir := filepath.Join(r.home, "new")
			subst := strings.NewReplacer("{metais}", r.metais, "{dir}", dir, "{tree}", r.tree,
				"{installed}", filepath.Join(r.home, ".local/bin/metais"), "{unpacked}", filepath.Join(dir, "metais"))
			run := func(argv []string) ([]string, []byte, error) {
				args := []string{script}
				for _, a := range argv {
					args = append(args, subst.Replace(a))
				}
				cmd := exec.Command("sh", args...)
				cmd.Dir = r.home
				cmd.Env = append(slices.Clone(r.env), c.env...)
				out, err := cmd.CombinedOutput()
				return args, out, err
			}
			if c.first != nil {
				if args, out, err := run(c.first); err != nil {
					t.Fatalf("rehearse.sh %s, run first, failed: %v\n%s", strings.Join(args[1:], " "), err, out)
				}
			}
			args, out, err := run(c.args)
			code := 0
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				code = exit.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			says := subst.Replace(c.says)
			if code != c.code || !strings.Contains(string(out), says) {
				t.Errorf("rehearse.sh %s exited %d saying\n%s\nwant exit %d saying %q", strings.Join(args[1:], " "), code, out, c.code, says)
			}
			if c.code == 0 && strings.Contains(string(out), "FAIL") {
				t.Errorf("rehearse.sh %s passed, saying FAIL:\n%s", strings.Join(args[1:], " "), out)
			}
		})
	}
}
