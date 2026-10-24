// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/yamlx"
)

// pagesWorkflow builds and deploys metais.org from a published release's
// tag (ADR-013 D7 item 3.4).
const pagesWorkflow = "../../.github/workflows/pages.yml"

// pagesGuard is the name of pages.yml's build step that checks the tag's
// release is published, before anything checks the tag out.
const pagesGuard = "the tag's release is published"

// fakePagesGH stands in for gh in TestPagesGuard. $FAKE/<tag> holds what
// GitHub answers for the release of tag, its draft field, and a tag with
// no file has no release. Every call is logged to $FAKE/calls.
const fakePagesGH = `#!/bin/sh
printf '%s\n' "$*" >>"$FAKE/calls"
case "$1 $2 $3 $4" in
"api repos/$GH_REPO/releases/tags/$TAG --jq .draft")
	[ -f "$FAKE/$TAG" ] || { echo "gh: Not Found (HTTP 404)" >&2; exit 1; }
	cat "$FAKE/$TAG" ;;
*)
	echo "fake gh: $*" >&2; exit 1 ;;
esac
`

// TestPagesGuard runs pages.yml's guard step, as the workflow holds it,
// against a stand-in gh (ruling R-W2-1 on ADR-013 D7 item 3.4). The site
// is never built from a branch, from a tag that is not a release's, or
// before its release is published, so the guard refuses a tag that is
// not v<version> whatever GitHub answers for it, a draft, and a tag GitHub
// answers no release for, since a failed answer is not a published
// release. A dispatch can redeploy a published tag, a prerelease's too.
func TestPagesGuard(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pages.yml runs on linux")
	}
	// A run step with no shell runs under bash -e on GitHub's linux
	// runners.
	shell := []string{"bash", "-e"}
	if _, err := exec.LookPath("bash"); err != nil {
		shell = []string{"sh", "-e"}
	}
	run := pagesGuardRun(t)
	dir := testenv.Dir(t)
	script := filepath.Join(dir, "guard.sh")
	write(t, script, run)
	bin := filepath.Join(dir, "bin")
	write(t, filepath.Join(bin, "gh"), fakePagesGH)
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, c := range []struct {
		name, tag string
		// draft is what GitHub answers for the tag's release's draft
		// field, or "" when it has no release.
		draft string
		code  int
		says  string
	}{
		{"a published release", "v1.0.0", "false", 0, ""},
		{"a published prerelease", "v1.0.0-rc.1", "false", 0, ""},
		{"a draft", "v1.0.0", "true", 1, "the release v1.0.0 is not published"},
		{"no release", "v1.0.0", "", 1, "there is no published release v1.0.0"},
		{"a branch", "main", "false", 1, "main is not a release's tag, v<version>"},
		{"the specification's tag", "spec-1.0", "false", 1, "spec-1.0 is not a release's tag, v<version>"},
		{"a tag of no version", "vnext", "false", 1, "vnext is not a release's tag, v<version>"},
		{"no tag", "", "false", 1, " is not a release's tag, v<version>"},
	} {
		t.Run(c.name, func(t *testing.T) {
			fake := testenv.Dir(t)
			write(t, filepath.Join(fake, "calls"), "")
			if c.draft != "" && c.tag != "" {
				write(t, filepath.Join(fake, c.tag), c.draft+"\n")
			}
			cmd := exec.Command(shell[0], append(shell[1:], script)...)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(),
				"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
				"FAKE="+fake, "GH_REPO=kottera/metais", "TAG="+c.tag)
			out, err := cmd.CombinedOutput()
			code := 0
			var exit *exec.ExitError
			if errors.As(err, &exit) {
				code = exit.ExitCode()
			} else if err != nil {
				t.Fatal(err)
			}
			if code != c.code || !strings.Contains(string(out), c.says) {
				t.Errorf("the guard of %q exited %d saying\n%s\nwant exit %d saying %q", c.tag, code, out, c.code, c.says)
			}
			calls, _ := os.ReadFile(filepath.Join(fake, "calls"))
			if c.code == 0 && string(calls) != "api repos/kottera/metais/releases/tags/"+c.tag+" --jq .draft\n" {
				t.Errorf("the guard of %q asked gh\n%s", c.tag, calls)
			}
		})
	}
}

// pagesGuardRun is the run of pages.yml's build step named pagesGuard.
func pagesGuardRun(t *testing.T) string {
	t.Helper()
	src, err := os.ReadFile(pagesWorkflow)
	if err != nil {
		t.Fatal(err)
	}
	doc, fds := yamlx.Parse(src, yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectInput, Name: "pages"}}, limits.Defaults())
	if len(fds) > 0 {
		t.Fatal(fds[0].Message())
	}
	jobs, _ := must(doc.Value.(value.Object).Get("jobs")).(value.Object)
	build, _ := must(jobs.Get("build")).(value.Object)
	for _, s := range must(build.Get("steps")).(value.Array).All() {
		so, _ := s.(value.Object)
		if name, ok := so.Get("name"); ok && text(name) == pagesGuard {
			if _, ok := so.Get("shell"); ok {
				t.Fatalf("pages.yml's step %q names a shell, which TestPagesGuard does not run it under", pagesGuard)
			}
			return text(must(so.Get("run")))
		}
	}
	t.Fatalf("pages.yml's build job has no step %q", pagesGuard)
	return ""
}
