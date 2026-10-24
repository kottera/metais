// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// releaseScript is release.yml's steps (ADR-013 D7 item 3).
const releaseScript = "../../../.github/scripts/release.sh"

// asCLI is set in the environment of this test binary when release.sh
// runs it as RELEASE_TOOL, so the script drives the real verify and
// formula without a build of its own.
const asCLI = "METAIS_RELEASE_TEST_AS_CLI"

func TestMain(m *testing.M) {
	if os.Getenv(asCLI) == "1" {
		os.Exit(cli(os.Args[1:], os.Stdout, os.Stderr, os.Environ()))
	}
	os.Exit(m.Run())
}

// fakeGH stands in for gh in TestReleaseScript. $FAKE/release/<tag>/
// holds a release's fields, one file each, and its assets; $FAKE holds the
// repository's private, its release environment's reviewers and the
// dispatching account's name. Every call is logged to $FAKE/calls.
const fakeGH = `#!/bin/sh
printf '%s\n' "$*" >>"$FAKE/calls"
case "$1 $2" in
"release view")
	[ -d "$FAKE/release/$3" ] || { echo "release not found" >&2; exit 1; }
	cat "$FAKE/release/$3/$5" ;;
"release download")
	cp "$FAKE/release/$3/assets/"* "$5" ;;
"release edit")
	echo false >"$FAKE/release/$3/isDraft" ;;
"api repos/$GITHUB_REPOSITORY")
	cat "$FAKE/private" ;;
"api repos/$GITHUB_REPOSITORY/environments/release")
	[ -f "$FAKE/reviewers" ] || { echo "HTTP 404" >&2; exit 1; }
	cat "$FAKE/reviewers" ;;
"api users/$ACTOR")
	cat "$FAKE/name" ;;
*)
	echo "fake gh: $*" >&2; exit 1 ;;
esac
`

// releaseRepos is a repository and its tap as TestReleaseScript builds
// them: origin, a bare repository, holds main and a side branch; clone is
// a whole clone of it, where release.sh runs; tap is the tap's bare
// repository.
type releaseRepos struct {
	origin, clone, tap, fake, work string
	// specCommit releases the specification 1.0; target, on top of it, is
	// the release's commit; side is a commit off main's history that
	// holds the released list too.
	specCommit, target, side string
	env                      []string
}

// newReleaseRepos builds the repositories and a draft of tag whose tool
// reports version and implements 1.0, with body as its text.
func newReleaseRepos(t *testing.T, tag, body string) *releaseRepos {
	t.Helper()
	r := &releaseRepos{
		origin: testenv.Dir(t), clone: filepath.Join(testenv.Dir(t), "clone"),
		tap: testenv.Dir(t), fake: testenv.Dir(t),
		work: filepath.Join(testenv.Dir(t), "work"),
	}
	src := testenv.Dir(t)
	gitIn(t, r.origin, "init", "--quiet", "--bare", "--initial-branch=main", ".")
	gitIn(t, src, "init", "--quiet", "--initial-branch=main", ".")
	write(t, src, "README.md", "metais\n")
	write(t, src, "spec/conformance/released/1.0.sha256", "the released list\n")
	gitIn(t, src, "add", ".")
	gitIn(t, src, "commit", "--quiet", "-m", "release the specification 1.0")
	r.specCommit = gitIn(t, src, "rev-parse", "HEAD")
	write(t, src, "README.md", "metais, released\n")
	gitIn(t, src, "commit", "--quiet", "-am", "the release")
	r.target = gitIn(t, src, "rev-parse", "HEAD")
	gitIn(t, src, "checkout", "--quiet", "-b", "side", r.specCommit+"~0")
	write(t, src, "README.md", "a side branch\n")
	gitIn(t, src, "commit", "--quiet", "-am", "off main")
	r.side = gitIn(t, src, "rev-parse", "HEAD")
	gitIn(t, src, "push", "--quiet", r.origin, "main", "side")
	gitIn(t, filepath.Dir(r.clone), "clone", "--quiet", r.origin, r.clone)

	gitIn(t, r.tap, "init", "--quiet", "--bare", "--initial-branch=main", ".")
	tapSrc := testenv.Dir(t)
	gitIn(t, tapSrc, "init", "--quiet", "--initial-branch=main", ".")
	write(t, tapSrc, "README.md", "the tap\n")
	gitIn(t, tapSrc, "add", ".")
	gitIn(t, tapSrc, "commit", "--quiet", "-m", "the tap")
	gitIn(t, tapSrc, "push", "--quiet", r.tap, "main")

	bin := filepath.Join(r.fake, "bin")
	write(t, bin, "gh", fakeGH)
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, r.fake, "private", "false\n")
	write(t, r.fake, "reviewers", "1\n")
	write(t, r.fake, "name", "Ed Example\n")
	write(t, r.fake, "calls", "")
	if tag != "" {
		version := strings.TrimPrefix(tag, "v")
		rel := filepath.Join(r.fake, "release", tag)
		write(t, rel, "isDraft", "true\n")
		write(t, rel, "targetCommitish", r.target+"\n")
		write(t, rel, "body", body)
		if err := os.MkdirAll(filepath.Join(rel, "assets"), 0o755); err != nil {
			t.Fatal(err)
		}
		writeAssets(t, filepath.Join(rel, "assets"), version, version, `"1.0"`)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	r.env = append(testenv.GitEnv(os.Environ()),
		"PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"),
		"FAKE="+r.fake, asCLI+"=1", "RELEASE_TOOL="+exe,
		"TAG="+tag, "GITHUB_REPOSITORY=kottera/metais",
		"ACTOR=edexample", "ACTOR_ID=12345", "TAP_REPO="+r.tap,
		"TAP_TOKEN_SET=true", "TAP_TOKEN=unused")
	return r
}

// commit commits to the clone's main a change to the file rel, which
// becomes content or is removed when content is empty, and makes the
// commit the draft's target.
func (r *releaseRepos) commit(t *testing.T, rel, content string) string {
	t.Helper()
	if content == "" {
		gitIn(t, r.clone, "rm", "--quiet", rel)
	} else {
		write(t, r.clone, rel, content)
		gitIn(t, r.clone, "add", rel)
	}
	gitIn(t, r.clone, "commit", "--quiet", "-m", "change "+rel)
	c := gitIn(t, r.clone, "rev-parse", "HEAD")
	for _, tag := range []string{"v1.0.0", "v1.0.0-rc.1"} {
		if _, err := os.Stat(filepath.Join(r.fake, "release", tag)); err == nil {
			write(t, filepath.Join(r.fake, "release", tag), "targetCommitish", c+"\n")
		}
	}
	return c
}

// run runs release.sh's step in the clone, under env after r's own, and
// returns its exit status and output.
func (r *releaseRepos) run(t *testing.T, step string, env ...string) (int, string) {
	t.Helper()
	abs, err := filepath.Abs(releaseScript)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("sh", abs, step, r.work)
	cmd.Dir = r.clone
	cmd.Env = append(append([]string(nil), r.env...), env...)
	out, err := cmd.CombinedOutput()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return exit.ExitCode(), string(out)
	}
	if err != nil {
		t.Fatal(err)
	}
	return 0, string(out)
}

// remoteTags is what git ls-remote --tags lists of repo.
func remoteTags(t *testing.T, repo string) string {
	t.Helper()
	return gitIn(t, repo, "ls-remote", "--tags", repo)
}

// release runs every step of a run that is not dry, failing t on a step
// that refuses.
func (r *releaseRepos) release(t *testing.T) {
	t.Helper()
	for _, step := range []string{"prepare", "push-tags", "publish", "push-tap"} {
		if code, out := r.run(t, step, "DRY_RUN=false"); code != 0 {
			t.Fatalf("%s exited %d:\n%s", step, code, out)
		}
	}
}

// TestReleaseScript runs release.yml's steps (ADR-013 D7 item 3) against
// a repository, its tap and a stand-in gh. A dry run checks everything,
// makes the tags and the tap's commit in its own clones, and pushes and
// publishes nothing; each push step refuses it. A run that is not dry
// pushes the annotated tags, publishes the draft marked as its version
// says, and pushes the generated formula to the tap, signed off and with no
// Co-authored-by line. Every refusal is prepare's, before anything is
// pushed.
func TestReleaseScript(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("release.yml runs on linux")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skipf("release.sh drives git: %v", err)
	}

	t.Run("a dry run pushes nothing", func(t *testing.T) {
		r := newReleaseRepos(t, "v1.0.0-rc.1", "The first candidate.\n")
		tapMain := gitIn(t, r.tap, "rev-parse", "main")
		code, out := r.run(t, "prepare", "DRY_RUN=true", "TAP_TOKEN_SET=", "TAP_TOKEN=")
		if code != 0 {
			t.Fatalf("prepare exited %d:\n%s", code, out)
		}
		for _, want := range []string{"implements 1.0, which is not released", "a prerelease", "push-tap pushes Formula/metais.rb of 1.0.0-rc.1", "dry run: nothing is pushed or published"} {
			if !strings.Contains(out, want) {
				t.Errorf("prepare's report does not say %q:\n%s", want, out)
			}
		}
		if got := gitIn(t, r.clone, "cat-file", "-t", "v1.0.0-rc.1"); got != "tag" {
			t.Errorf("the clone's v1.0.0-rc.1 is a %s, not an annotated tag", got)
		}
		if got := gitIn(t, r.clone, "rev-parse", "v1.0.0-rc.1^{commit}"); got != r.target {
			t.Errorf("the clone's v1.0.0-rc.1 is at %s, not the draft's target %s", got, r.target)
		}
		if got := gitIn(t, r.clone, "for-each-ref", "--format=%(taggername) %(taggeremail)", "refs/tags/v1.0.0-rc.1"); got != "Ed Example <12345+edexample@users.noreply.github.com>" {
			t.Errorf("v1.0.0-rc.1 is tagged by %q", got)
		}
		tapClone := filepath.Join(r.work, "tap")
		sums := readFile(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1", "assets", sumsName))
		want, err := formula("1.0.0-rc.1", []byte(sums))
		if err != nil {
			t.Fatal(err)
		}
		if got := readFile(t, filepath.Join(tapClone, "Formula", "metais.rb")); got != string(want) {
			t.Errorf("the tap's commit holds a formula that is not the generator's:\n%s", got)
		}
		msg := gitIn(t, tapClone, "log", "-1", "--format=%an <%ae>%n%B")
		if !strings.HasPrefix(msg, "Ed Example <12345+edexample@users.noreply.github.com>\nmetais 1.0.0-rc.1\n") ||
			!strings.HasSuffix(msg, "\nSigned-off-by: Ed Example <12345+edexample@users.noreply.github.com>") ||
			strings.Contains(strings.ToLower(msg), "co-authored-by") {
			t.Errorf("the tap's commit is\n%s", msg)
		}

		for _, step := range []string{"push-tags", "publish", "push-tap"} {
			if code, out := r.run(t, step, "DRY_RUN=true"); code != 1 || !strings.Contains(out, "a dry run pushes nothing") {
				t.Errorf("%s in a dry run exited %d:\n%s", step, code, out)
			}
			if code, out := r.run(t, step, "DRY_RUN=false"); code != 1 || !strings.Contains(out, "a dry run pushes nothing") {
				t.Errorf("%s after a dry run's prepare exited %d:\n%s", step, code, out)
			}
		}
		if got := remoteTags(t, r.origin); got != "" {
			t.Errorf("a dry run pushed tags to origin:\n%s", got)
		}
		if got := gitIn(t, r.tap, "rev-parse", "main"); got != tapMain {
			t.Errorf("a dry run moved the tap's main")
		}
		if calls := readFile(t, filepath.Join(r.fake, "calls")); strings.Contains(calls, "release edit") {
			t.Errorf("a dry run edited the release:\n%s", calls)
		}
	})

	t.Run("a prerelease", func(t *testing.T) {
		r := newReleaseRepos(t, "v1.0.0-rc.1", "The first candidate.\n")
		r.release(t)
		tags := remoteTags(t, r.origin)
		if !strings.Contains(tags, r.target+"\trefs/tags/v1.0.0-rc.1^{}") || strings.Contains(tags, "spec-") {
			t.Errorf("origin's tags are\n%s\nwant the annotated v1.0.0-rc.1 at %s alone", tags, r.target)
		}
		if calls := readFile(t, filepath.Join(r.fake, "calls")); !strings.Contains(calls, "release edit v1.0.0-rc.1 --draft=false --prerelease=true --verify-tag\n") {
			t.Errorf("the draft was not published as a prerelease:\n%s", calls)
		}
		if got := gitIn(t, r.tap, "show", "main:Formula/metais.rb"); !strings.Contains(got, "/v1.0.0-rc.1/metais_1.0.0-rc.1_linux_arm64.tar.gz") {
			t.Errorf("the tap's main holds\n%s", got)
		}
	})

	t.Run("a release with its specification", func(t *testing.T) {
		r := newReleaseRepos(t, "v1.0.0", "")
		write(t, filepath.Join(r.fake, "release", "v1.0.0"), "body", "metais 1.0.0.\r\n\r\nSpecification release: spec-1.0 "+r.specCommit+"\r\n")
		r.release(t)
		tags := remoteTags(t, r.origin)
		for _, want := range []string{r.target + "\trefs/tags/v1.0.0^{}", r.specCommit + "\trefs/tags/spec-1.0^{}"} {
			if !strings.Contains(tags, want) {
				t.Errorf("origin's tags are\n%s\nwant %q", tags, want)
			}
		}
		if got := gitIn(t, r.origin, "cat-file", "-p", "spec-1.0"); !strings.HasSuffix(got, "\n\nMETAIS specification 1.0") {
			t.Errorf("spec-1.0 is\n%s", got)
		}
		if calls := readFile(t, filepath.Join(r.fake, "calls")); !strings.Contains(calls, "release edit v1.0.0 --draft=false --prerelease=false --latest=true --verify-tag\n") {
			t.Errorf("the draft was not published as a release:\n%s", calls)
		}
		if got := gitIn(t, r.tap, "show", "main:Formula/metais.rb"); !strings.Contains(got, "/v1.0.0/metais_1.0.0_darwin_arm64.tar.gz") {
			t.Errorf("the tap's main holds\n%s", got)
		}
	})

	// The tap, and GitHub's latest release, follow the newest release that
	// is not a prerelease (ADR-013 D6 item 1, D7 item 5). A prerelease
	// updates the tap only while origin has no such release's tag, so
	// earlier candidates do not stop one, and whatever its suffix it is
	// a prerelease. A release updates the tap after an earlier release,
	// but one older than the newest, compared as numbers, moves neither
	// the tap nor GitHub's latest back.
	for _, c := range []struct {
		name, tag string
		earlier   []string
		tap       bool
		edit      string
	}{
		{"a candidate after a candidate updates the tap", "v1.0.0-rc.2", []string{"v1.0.0-rc.1"}, true, "--prerelease=true --verify-tag"},
		{"a prerelease after a release leaves the tap", "v1.1.0-rc.1", []string{"v1.0.0"}, false, "--prerelease=true --verify-tag"},
		{"an alpha after a release is a prerelease", "v1.1.0-alpha.1", []string{"v1.0.0"}, false, "--prerelease=true --verify-tag"},
		{"a release after a release updates the tap", "v1.0.1", []string{"v1.0.0-rc.1", "v1.0.0"}, true, "--prerelease=false --latest=true --verify-tag"},
		{"a release after a candidate of the next updates the tap", "v1.0.1", []string{"v1.0.0", "v1.1.0-rc.1"}, true, "--prerelease=false --latest=true --verify-tag"},
		{"a release is compared as numbers", "v1.10.0", []string{"v1.9.0"}, true, "--prerelease=false --latest=true --verify-tag"},
		{"an older line leaves the tap and the latest", "v1.0.2", []string{"v1.0.0", "v1.1.0", "v1.0.1"}, false, "--prerelease=false --latest=false --verify-tag"},
		{"an older line is compared as numbers", "v1.9.1", []string{"v1.10.0"}, false, "--prerelease=false --latest=false --verify-tag"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newReleaseRepos(t, c.tag, "")
			gitIn(t, r.clone, "tag", "-a", "-m", "spec", "spec-1.0", r.specCommit)
			push := []string{"push", "--quiet", "origin", "spec-1.0"}
			for _, tag := range c.earlier {
				gitIn(t, r.clone, "tag", tag, r.specCommit)
				push = append(push, tag)
			}
			gitIn(t, r.clone, push...)
			tapMain := gitIn(t, r.tap, "rev-parse", "main")
			r.release(t)
			if !strings.Contains(remoteTags(t, r.origin), r.target+"\trefs/tags/"+c.tag+"^{}") {
				t.Errorf("%s was not pushed", c.tag)
			}
			if calls := readFile(t, filepath.Join(r.fake, "calls")); !strings.Contains(calls, "release edit "+c.tag+" --draft=false "+c.edit+"\n") {
				t.Errorf("the draft was not published with %s:\n%s", c.edit, calls)
			}
			version := strings.TrimPrefix(c.tag, "v")
			if c.tap {
				if got := gitIn(t, r.tap, "show", "main:Formula/metais.rb"); !strings.Contains(got, "/"+c.tag+"/metais_"+version+"_darwin_arm64.tar.gz") {
					t.Errorf("after %q, %s did not update the tap, whose main holds\n%s", c.earlier, c.tag, got)
				}
			} else if got := gitIn(t, r.tap, "rev-parse", "main"); got != tapMain {
				t.Errorf("after %q, %s updated the tap", c.earlier, c.tag)
			}
		})
	}

	// GitHub refuses the workflow's token a tag where .github/workflows is
	// not main's, so prepare refuses that target; a target behind main
	// whose workflows are main's is released.
	t.Run("a target behind main with main's workflows", func(t *testing.T) {
		r := newReleaseRepos(t, "v1.0.0-rc.1", "")
		r.commit(t, ".github/workflows/ci.yml", "name: ci\n")
		write(t, r.clone, "README.md", "metais, after the draft\n")
		gitIn(t, r.clone, "commit", "--quiet", "-am", "a change after the draft")
		target := strings.TrimSpace(readFile(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1", "targetCommitish")))
		r.release(t)
		if !strings.Contains(remoteTags(t, r.origin), target+"\trefs/tags/v1.0.0-rc.1^{}") {
			t.Errorf("v1.0.0-rc.1 was not pushed at the draft's target %s", target)
		}
	})

	t.Run("a tag pushed before is not pushed again", func(t *testing.T) {
		r := newReleaseRepos(t, "v1.0.0-rc.1", "")
		gitIn(t, r.clone, "tag", "-a", "-m", "earlier", "v1.0.0-rc.1", r.target)
		gitIn(t, r.clone, "push", "--quiet", "origin", "v1.0.0-rc.1")
		before := remoteTags(t, r.origin)
		r.release(t)
		if got := remoteTags(t, r.origin); got != before {
			t.Errorf("origin's tags moved from\n%s\nto\n%s", before, got)
		}
	})

	for _, c := range []struct {
		name, tag string
		dry       bool
		change    func(t *testing.T, r *releaseRepos)
		want      string
	}{
		{"a tag that is not v<version>", "1.0.0-rc.1", false, nil, "is not v<version>"},
		{"no such release", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			os.RemoveAll(filepath.Join(r.fake, "release"))
		}, "there is no release v1.0.0-rc.1"},
		{"a published release", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1"), "isDraft", "false\n")
		}, "is published already"},
		{"a branch as the target", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1"), "targetCommitish", "main\n")
		}, "is not a commit's full SHA"},
		{"a target the repository lacks", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1"), "targetCommitish", strings.Repeat("ab", 20)+"\n")
		}, "holds no commit"},
		{"an asset changed", "v1.0.0-rc.1", true, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1", "assets"), "metais_1.0.0-rc.1_linux_amd64.tar.gz", "not the archive")
		}, "the draft's assets are refused"},
		{"a prerelease with a specification release", "v1.0.0-rc.1", true, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0-rc.1"), "body", "Specification release: spec-1.0 "+r.specCommit+"\n")
		}, "a prerelease never carries a specification release"},
		{"two specification releases", "v1.0.0", true, func(t *testing.T, r *releaseRepos) {
			line := "Specification release: spec-1.0 " + r.specCommit + "\n"
			write(t, filepath.Join(r.fake, "release", "v1.0.0"), "body", line+line)
		}, "more than one specification release"},
		{"a specification release off the target's history", "v1.0.0", true, func(t *testing.T, r *releaseRepos) {
			write(t, filepath.Join(r.fake, "release", "v1.0.0"), "body", "Specification release: spec-1.0 "+r.side+"\n")
		}, "is not an ancestor of the draft's target"},
		{"a specification release that is not one", "v1.0.0", true, func(t *testing.T, r *releaseRepos) {
			c := r.commit(t, "spec/conformance/released/1.0.sha256", "")
			write(t, filepath.Join(r.fake, "release", "v1.0.0"), "body", "Specification release: spec-1.0 "+c+"\n")
		}, "holds no spec/conformance/released/1.0.sha256"},
		{"a specification release the tool does not implement", "v1.0.0", true, func(t *testing.T, r *releaseRepos) {
			c := r.commit(t, "spec/conformance/released/1.1.sha256", "the released list\n")
			write(t, filepath.Join(r.fake, "release", "v1.0.0"), "body", "Specification release: spec-1.1 "+c+"\n")
		}, "the release's tool implements only 1.0"},
		{"a release of an unreleased revision", "v1.0.0", true, nil, "implements 1.0, which is not released"},
		{"a tag at another commit", "v1.0.0-rc.1", true, func(t *testing.T, r *releaseRepos) {
			gitIn(t, r.clone, "tag", "v1.0.0-rc.1", r.specCommit)
			gitIn(t, r.clone, "push", "--quiet", "origin", "v1.0.0-rc.1")
			gitIn(t, r.clone, "tag", "-d", "v1.0.0-rc.1")
		}, "a tag never moves"},
		{"a target whose workflows are not main's", "v1.0.0-rc.1", true, func(t *testing.T, r *releaseRepos) {
			r.commit(t, ".github/workflows/ci.yml", "name: ci\n")
			write(t, r.clone, ".github/workflows/ci.yml", "name: ci, pinned\n")
			gitIn(t, r.clone, "commit", "--quiet", "-am", "change ci.yml after the draft")
		}, "holds workflows other than main's"},
		{"a private repository", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			write(t, r.fake, "private", "true\n")
		}, "is private"},
		{"no required reviewer", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			write(t, r.fake, "reviewers", "0\n")
		}, "requires no reviewer"},
		{"no release environment", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			os.Remove(filepath.Join(r.fake, "reviewers"))
		}, "requires no reviewer"},
		{"no TAP_TOKEN", "v1.0.0-rc.1", false, func(t *testing.T, r *releaseRepos) {
			r.env = append(r.env, "TAP_TOKEN_SET=false")
		}, "holds no TAP_TOKEN"},
	} {
		t.Run(c.name, func(t *testing.T) {
			r := newReleaseRepos(t, c.tag, "")
			if c.change != nil {
				c.change(t, r)
			}
			tapMain := gitIn(t, r.tap, "rev-parse", "main")
			code, out := r.run(t, "prepare", "DRY_RUN="+map[bool]string{true: "true", false: "false"}[c.dry])
			if code != 1 || !strings.Contains(out, c.want) {
				t.Errorf("prepare exited %d saying\n%s\nwant exit 1 saying %q", code, out, c.want)
			}
			if code, out := r.run(t, "push-tags", "DRY_RUN=false"); code != 1 || !strings.Contains(out, "prepare runs first") {
				t.Errorf("push-tags after a refused prepare exited %d:\n%s", code, out)
			}
			if got := gitIn(t, r.tap, "rev-parse", "main"); got != tapMain {
				t.Errorf("a refused release moved the tap's main")
			}
			if calls := readFile(t, filepath.Join(r.fake, "calls")); strings.Contains(calls, "release edit") {
				t.Errorf("a refused release edited the release:\n%s", calls)
			}
		})
	}
}
