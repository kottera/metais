// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/tools/release/buildline"
)

// The fake module every test builds: a github.com/kottera/metais whose
// ./cmd/metais answers version --json as the real one does, with the
// version a release sets and the implements its implements.go states, so
// a build costs seconds rather than the real tool's minutes.
const (
	fakeGoMod   = "module github.com/kottera/metais\n\ngo 1.26.0\n"
	fakeVersion = `// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package version

var release string

func String() string { return release }
`
	fakeMain = `// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"

	"github.com/kottera/metais/internal/version"
)

func main() {
	if len(os.Args) != 3 || os.Args[1] != "version" || os.Args[2] != "--json" {
		os.Exit(2)
	}
	os.Stdout.WriteString("{\"format\":\"metais.version/1\",\"ok\":true,\"result\":{\"tool\":{\"name\":\"metais\",\"version\":\"" +
		reported(version.String()) + "\"},\"specification\":{\"implements\":[" + implements + "]}}}\n")
}
`
	fakeImplements = `package main

const implements = %s

func reported(v string) string { return %s }
`
)

// The notice files' contents in the fake module.
var fakeNotices = map[string]string{
	"LICENSE":                 "the licence\n",
	"NOTICE":                  "the notice\n",
	"THIRD-PARTY-NOTICES.txt": "the third parties\n",
}

// gitIn runs git in dir as a test does, with fixed identities and dates so
// no commit depends on the host or the clock.
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(testenv.GitEnv(os.Environ()),
		"GIT_AUTHOR_NAME=Release Test", "GIT_AUTHOR_EMAIL=release@example.invalid",
		"GIT_COMMITTER_NAME=Release Test", "GIT_COMMITTER_EMAIL=release@example.invalid",
		"GIT_AUTHOR_DATE=2026-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// write writes content to rel under dir, making its directories.
func write(t *testing.T, dir, rel, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// implementsGo is the fake's implements.go: a tool implementing revs,
// which reports its version as reported, a Go expression over v.
func implementsGo(revs []string, reported string) string {
	quoted := make([]string, len(revs))
	for i, r := range revs {
		quoted[i] = `\"` + r + `\"`
	}
	return fmt.Sprintf(fakeImplements, `"`+strings.Join(quoted, ",")+`"`, reported)
}

// fakeRepo makes a temporary repository holding the fake module, whose
// tool implements 1.0, with 1.0 released as ADR-011 D5 item 2 releases a
// revision (an annotated tag spec-1.0 and its released list), and HEAD
// tagged v1.0.0. It gives the test a fresh HOME and returns the work tree.
func fakeRepo(t *testing.T) string {
	t.Helper()
	testenv.Home(t)
	dir := testenv.Dir(t)
	write(t, dir, "go.mod", fakeGoMod)
	write(t, dir, "internal/version/version.go", fakeVersion)
	write(t, dir, "cmd/metais/main.go", fakeMain)
	write(t, dir, "cmd/metais/implements.go", implementsGo([]string{"1.0"}, "v"))
	write(t, dir, ".gitignore", "/dist/\n")
	write(t, dir, "spec/conformance/released/1.0.sha256", "")
	for name, content := range fakeNotices {
		write(t, dir, name, content)
	}
	gitIn(t, dir, "init", "--quiet", "--initial-branch=main")
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "--quiet", "-m", "the fake module")
	gitIn(t, dir, "tag", "-a", "-m", "spec 1.0", "spec-1.0")
	gitIn(t, dir, "tag", "v1.0.0")
	return dir
}

// commit commits every change under dir.
func commit(t *testing.T, dir, msg string) {
	t.Helper()
	gitIn(t, dir, "add", ".")
	gitIn(t, dir, "commit", "--quiet", "-m", msg)
}

// runIn runs the builder over the work tree dir for version.
func runIn(dir, version string, tamper func(buildline.Target, int, string) error) (string, error) {
	var out bytes.Buffer
	c := config{dir: dir, version: version, stdout: &out, environ: os.Environ(), tamper: tamper}
	err := c.run()
	return out.String(), err
}

// refused runs the builder and requires the refusal want, with nothing
// written to dist/.
func refused(t *testing.T, dir, version string, want error, tamper func(buildline.Target, int, string) error) error {
	t.Helper()
	_, err := runIn(dir, version, tamper)
	if !errors.Is(err, want) {
		t.Fatalf("the builder returned %v, want %v", err, want)
	}
	if _, serr := os.Lstat(filepath.Join(dir, "dist")); serr == nil {
		t.Errorf("a refused release wrote dist/")
	}
	return err
}

func TestReleaseBuildsTwiceAndWritesDist(t *testing.T) {
	dir := fakeRepo(t)
	built := map[buildline.Target][]string{}
	record := func(tg buildline.Target, copy int, bin string) error {
		copyDir := filepath.Dir(filepath.Dir(filepath.Dir(bin)))
		if fi, err := os.Stat(filepath.Join(copyDir, "gocache")); err != nil || !fi.IsDir() {
			t.Errorf("%s copy %d was built without its own GOCACHE beside its clone: %v", tg, copy, err)
		}
		if _, err := os.Stat(filepath.Join(copyDir, "src", "go.mod")); err != nil {
			t.Errorf("%s copy %d was not built from its own clone: %v", tg, copy, err)
		}
		built[tg] = append(built[tg], copyDir)
		return nil
	}
	head := gitIn(t, dir, "rev-parse", "HEAD")
	report, err := runIn(dir, "1.0.0", record)
	if err != nil {
		t.Fatal(err)
	}

	// W7-R.1: every shipped target, twice, in two fresh directories.
	for _, tg := range buildline.Shipped {
		if got := built[tg]; len(got) != copies || got[0] == got[1] {
			t.Errorf("%s was built in %q, want twice in two directories", tg, got)
		}
	}
	if len(built) != len(buildline.Shipped) {
		t.Errorf("the builder built %d targets, want the %d shipped", len(built), len(buildline.Shipped))
	}

	// W7-R.3: the archives and SHA256SUMS.
	dist := filepath.Join(dir, "dist")
	sums := readFile(t, filepath.Join(dist, "SHA256SUMS"))
	if !strings.HasSuffix(report, sums) {
		t.Errorf("the report %q does not end with SHA256SUMS %q", report, sums)
	}
	var names []string
	for _, tg := range buildline.Shipped {
		names = append(names, "metais_1.0.0_"+tg.OS+"_"+tg.Arch+".tar.gz")
	}
	entries, err := os.ReadDir(dist)
	if err != nil {
		t.Fatal(err)
	}
	var listed []string
	for _, e := range entries {
		listed = append(listed, e.Name())
	}
	if want := append(slices.Clone(names), "SHA256SUMS"); !slices.Equal(listed, slices.Sorted(slices.Values(want))) {
		t.Errorf("dist/ holds %q, want %q", listed, want)
	}
	var lines []string
	for line := range strings.Lines(sums) {
		lines = append(lines, line)
	}
	if len(lines) != len(names) {
		t.Fatalf("SHA256SUMS has %d lines, want %d:\n%s", len(lines), len(names), sums)
	}
	for i, name := range names {
		b := readFile(t, filepath.Join(dist, name))
		if want := sha256Hex(b) + "  " + name + "\n"; lines[i] != want {
			t.Errorf("SHA256SUMS line %d is %q, want %q", i+1, lines[i], want)
		}
		checkArchive(t, name, []byte(b), buildline.Shipped[i], head)
	}

	// Two runs produce matching SHA256SUMS.
	if err := os.Rename(dist, filepath.Join(testenv.Dir(t), "first")); err != nil {
		t.Fatal(err)
	}
	if _, err := runIn(dir, "1.0.0", nil); err != nil {
		t.Fatal(err)
	}
	if again := readFile(t, filepath.Join(dist, "SHA256SUMS")); again != sums {
		t.Errorf("a second run's SHA256SUMS is\n%s\nthe first's\n%s", again, sums)
	}
}

// checkArchive holds one archive to ADR-010 D12 item 5: a gzip header with
// no name and a zero mtime, and a tar of the notice files and the binary in
// byte order of their names, with mode 0755 for the binary and 0644
// otherwise, uid and gid 0, empty owner names and ADR-001's fixed instant.
func checkArchive(t *testing.T, name string, b []byte, tg buildline.Target, head string) {
	t.Helper()
	zr, err := gzip.NewReader(bytes.NewReader(b))
	if err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	if zr.Name != "" || zr.Comment != "" || !zr.ModTime.IsZero() || zr.Extra != nil {
		t.Errorf("%s: the gzip header carries name %q, comment %q, mtime %v, extra %q", name, zr.Name, zr.Comment, zr.ModTime, zr.Extra)
	}
	fixed := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	tr := tar.NewReader(zr)
	var got []string
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		got = append(got, h.Name)
		content, err := io.ReadAll(tr)
		if err != nil {
			t.Fatal(err)
		}
		mode := int64(0o644)
		if h.Name == binaryName {
			mode = 0o755
			bin := filepath.Join(testenv.Dir(t), binaryName)
			if err := os.WriteFile(bin, content, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := checkSettings(goVersionM(t, bin), tg, head); err != nil {
				t.Errorf("%s: the archived binary: %v", name, err)
			}
		} else if string(content) != fakeNotices[h.Name] {
			t.Errorf("%s: %s holds %q, want %q", name, h.Name, content, fakeNotices[h.Name])
		}
		if h.Typeflag != tar.TypeReg || h.Mode != mode || h.Uid != 0 || h.Gid != 0 || h.Uname != "" || h.Gname != "" || !h.ModTime.Equal(fixed) {
			t.Errorf("%s: %s's header is type %c, mode %o, uid %d, gid %d, owner %q:%q, mtime %v", name, h.Name, h.Typeflag, h.Mode, h.Uid, h.Gid, h.Uname, h.Gname, h.ModTime)
		}
	}
	if want := []string{"LICENSE", "NOTICE", "THIRD-PARTY-NOTICES.txt", "metais"}; !slices.Equal(got, want) {
		t.Errorf("%s holds %q, want %q", name, got, want)
	}
}

func readFile(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func goVersionM(t *testing.T, bin string) string {
	t.Helper()
	out, err := exec.Command("go", "version", "-m", bin).Output()
	if err != nil {
		t.Fatalf("go version -m %s: %v", bin, err)
	}
	return string(out)
}

func sha256Hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func TestReleaseRefusesADirtyTree(t *testing.T) {
	t.Run("a modified file", func(t *testing.T) {
		dir := fakeRepo(t)
		write(t, dir, "NOTICE", "changed\n")
		refused(t, dir, "1.0.0", errDirty, nil)
	})
	t.Run("a staged file", func(t *testing.T) {
		dir := fakeRepo(t)
		write(t, dir, "NOTICE", "changed\n")
		gitIn(t, dir, "add", "NOTICE")
		refused(t, dir, "1.0.0", errDirty, nil)
	})
	t.Run("an untracked file", func(t *testing.T) {
		dir := fakeRepo(t)
		write(t, dir, "cmd/metais/extra.go", "package main\n")
		refused(t, dir, "1.0.0", errDirty, nil)
	})
}

func TestReleaseRefusesAHeadThatIsNotTheTag(t *testing.T) {
	t.Run("no tag", func(t *testing.T) {
		dir := fakeRepo(t)
		refused(t, dir, "1.0.1", errNotTag, nil)
	})
	t.Run("the tag is an earlier commit", func(t *testing.T) {
		dir := fakeRepo(t)
		write(t, dir, "NOTICE", "changed\n")
		commit(t, dir, "after the tag")
		refused(t, dir, "1.0.0", errNotTag, nil)
	})
}

func TestReleaseRefusesAVersion(t *testing.T) {
	for _, v := range []string{"", "v1.0.0", "1.0", "1.0.0.0", "01.0.0", "1.00.0", "1.0.0+build", "1.0.0-", "1.0.0-rc..1", "1.0.0-01", "1.0.0-r_c", "1.0.0 "} {
		if releaseVersion(v) {
			t.Errorf("releaseVersion(%q) is true", v)
		}
	}
	for _, v := range []string{"0.0.0", "1.0.0", "10.20.30", "1.0.0-rc.1", "1.0.0-0", "1.0.0-x-y.0a"} {
		if !releaseVersion(v) {
			t.Errorf("releaseVersion(%q) is false", v)
		}
	}
	dir := fakeRepo(t)
	gitIn(t, dir, "tag", "vv1.0.0")
	refused(t, dir, "v1.0.0", errVersion, nil)
}

func TestReleaseRefusesAnExistingDist(t *testing.T) {
	dir := fakeRepo(t)
	write(t, dir, "dist/old.tar.gz", "")
	if _, err := runIn(dir, "1.0.0", nil); !errors.Is(err, errDistExists) {
		t.Fatalf("the builder returned %v, want %v", err, errDistExists)
	}
	if got := readFile(t, filepath.Join(dir, "dist", "old.tar.gz")); got != "" {
		t.Errorf("the builder changed an existing dist/")
	}
}

func TestReleaseRefusesAnUnreleasedRevision(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, dir string)
		want  string
	}{
		{"a revision with no tag", func(t *testing.T, dir string) {
			write(t, dir, "cmd/metais/implements.go", implementsGo([]string{"1.0", "1.1"}, "v"))
			commit(t, dir, "implements 1.1")
		}, "1.1: there is no annotated tag spec-1.1"},
		{"a lightweight tag", func(t *testing.T, dir string) {
			gitIn(t, dir, "tag", "-d", "spec-1.0")
			gitIn(t, dir, "tag", "spec-1.0", "HEAD")
		}, "1.0: there is no annotated tag spec-1.0"},
		{"a tag off HEAD's history", func(t *testing.T, dir string) {
			gitIn(t, dir, "checkout", "--quiet", "-b", "side")
			write(t, dir, "NOTICE", "side\n")
			commit(t, dir, "a side commit")
			gitIn(t, dir, "tag", "-d", "spec-1.0")
			gitIn(t, dir, "tag", "-a", "-m", "spec 1.0", "spec-1.0")
			gitIn(t, dir, "checkout", "--quiet", "main")
		}, "1.0: spec-1.0 is not an ancestor of HEAD"},
		{"no released list", func(t *testing.T, dir string) {
			gitIn(t, dir, "rm", "--quiet", "spec/conformance/released/1.0.sha256")
			commit(t, dir, "no released list")
		}, "1.0: HEAD has no spec/conformance/released/1.0.sha256"},
		{"implements nothing", func(t *testing.T, dir string) {
			write(t, dir, "cmd/metais/implements.go", implementsGo(nil, "v"))
			commit(t, dir, "implements nothing")
		}, "it implements nothing"},
		{"not a revision", func(t *testing.T, dir string) {
			write(t, dir, "cmd/metais/implements.go", implementsGo([]string{"1.0", "1.x"}, "v"))
			commit(t, dir, "implements 1.x")
		}, "1.x: not a revision"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := fakeRepo(t)
			c.setup(t, dir)
			gitIn(t, dir, "tag", "-f", "v1.0.0")
			err := refused(t, dir, "1.0.0", errUnreleased, nil)
			if !strings.Contains(err.Error(), c.want) {
				t.Errorf("the refusal %q does not say %q", err, c.want)
			}
		})
	}
}

func TestReleaseRefusesAToolOfAnotherVersion(t *testing.T) {
	dir := fakeRepo(t)
	write(t, dir, "cmd/metais/implements.go", implementsGo([]string{"1.0"}, `"0.0.0-devel"`))
	commit(t, dir, "reports devel")
	gitIn(t, dir, "tag", "-f", "v1.0.0")
	refused(t, dir, "1.0.0", errToolVersion, nil)
}

func TestReleaseRefusesUnequalBuilds(t *testing.T) {
	dir := fakeRepo(t)
	tamper := func(tg buildline.Target, copy int, bin string) error {
		if copy == 0 {
			return nil
		}
		f, err := os.OpenFile(bin, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write([]byte{0})
		return err
	}
	err := refused(t, dir, "1.0.0", errNotReproducible, tamper)
	if !strings.Contains(err.Error(), buildline.Shipped[0].String()) {
		t.Errorf("the refusal %q does not name the target", err)
	}
}

func TestReleaseRefusesABuildWithoutTheReleaseSettings(t *testing.T) {
	dir := fakeRepo(t)
	// The second copy of the first target is rebuilt as a plain go build
	// would build it: no -trimpath, and cgo as the host has it.
	tamper := func(tg buildline.Target, copy int, bin string) error {
		if copy == 0 {
			return nil
		}
		cmd := exec.Command("go", "build", "-o", bin, buildline.Package)
		cmd.Dir = filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(bin))), "src")
		cmd.Env = append(os.Environ(), "GOOS="+tg.OS, "GOARCH="+tg.Arch, "CGO_ENABLED=1", "GOFLAGS=")
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go build: %v\n%s", err, out)
		}
		return nil
	}
	err := refused(t, dir, "1.0.0", errSettings, tamper)
	for _, want := range []string{"copy 2", "lacks -trimpath=true", "has CGO_ENABLED=1, not 0"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("the refusal %q does not say %q", err, want)
		}
	}
}

func TestCheckSettings(t *testing.T) {
	const head = "0123456789abcdef0123456789abcdef01234567"
	good := map[buildline.Target]string{}
	for _, tg := range buildline.Shipped {
		level := "\tbuild\tGOAMD64=v1\n"
		if tg.Arch == "arm64" {
			level = "\tbuild\tGOARM64=v8.0\n"
		}
		good[tg] = "metais: go1.26.9\n\tpath\tgithub.com/kottera/metais/cmd/metais\n" +
			"\tmod\tgithub.com/kottera/metais\t(devel)\t\n" +
			"\tbuild\t-buildmode=exe\n\tbuild\t-compiler=gc\n\tbuild\t-trimpath=true\n" +
			"\tbuild\tCGO_ENABLED=0\n\tbuild\tGOARCH=" + tg.Arch + "\n\tbuild\tGOOS=" + tg.OS + "\n" + level +
			"\tbuild\tvcs=git\n\tbuild\tvcs.revision=" + head + "\n\tbuild\tvcs.time=2026-01-01T00:00:00Z\n\tbuild\tvcs.modified=false\n"
		if err := checkSettings(good[tg], tg, head); err != nil {
			t.Errorf("%s: a release's settings were refused: %v", tg, err)
		}
	}
	linux := buildline.Target{OS: "linux", Arch: "amd64"}
	arm := buildline.Target{OS: "darwin", Arch: "arm64"}
	cases := []struct {
		name string
		tg   buildline.Target
		info string
		want string
	}{
		{"no CGO_ENABLED", linux, strings.Replace(good[linux], "\tbuild\tCGO_ENABLED=0\n", "", 1), "lacks CGO_ENABLED=0"},
		{"cgo", linux, strings.Replace(good[linux], "CGO_ENABLED=0", "CGO_ENABLED=1", 1), "has CGO_ENABLED=1, not 0"},
		{"no -trimpath", linux, strings.Replace(good[linux], "\tbuild\t-trimpath=true\n", "", 1), "lacks -trimpath=true"},
		{"another GOOS", linux, strings.Replace(good[linux], "GOOS=linux", "GOOS=darwin", 1), "has GOOS=darwin, not linux"},
		{"no GOOS", linux, strings.Replace(good[linux], "\tbuild\tGOOS=linux\n", "", 1), "lacks GOOS=linux"},
		{"another GOARCH", linux, strings.Replace(good[linux], "GOARCH=amd64", "GOARCH=arm64", 1), "has GOARCH=arm64, not amd64"},
		{"no GOARCH", linux, strings.Replace(good[linux], "\tbuild\tGOARCH=amd64\n", "", 1), "lacks GOARCH=amd64"},
		{"another GOAMD64", linux, strings.Replace(good[linux], "GOAMD64=v1", "GOAMD64=v3", 1), "has GOAMD64=v3, not v1"},
		{"another GOARM64", arm, strings.Replace(good[arm], "GOARM64=v8.0", "GOARM64=v9.0", 1), "has GOARM64=v9.0, not v8.0"},
		{"a dirty tree", linux, strings.Replace(good[linux], "vcs.modified=false", "vcs.modified=true", 1), "has vcs.modified=true, not false"},
		{"another revision", linux, strings.Replace(good[linux], head, strings.Repeat("f", 40), 1), "has vcs.revision="},
		{"no VCS stamp", linux, strings.Replace(good[linux], "\tbuild\tvcs=git\n", "", 1), "lacks vcs=git"},
		{"build tags", linux, good[linux] + "\tbuild\t-tags=metais_canary_env\n", "has -tags=metais_canary_env, which a release does not set"},
		{"the race detector", linux, good[linux] + "\tbuild\t-race=true\n", "has -race=true, which a release does not set"},
		{"an experiment", linux, good[linux] + "\tbuild\tGOEXPERIMENT=foo\n", "has GOEXPERIMENT=foo, which a release does not set"},
		{"a quoted value", linux, good[linux] + "\tbuild\t-ldflags=\"-X a=b\"\n", "has -ldflags=-X a=b, which a release does not set"},
	}
	for _, c := range cases {
		err := checkSettings(c.info, c.tg, head)
		if !errors.Is(err, errSettings) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: checkSettings returned %v, want a refusal saying %q", c.name, err, c.want)
		}
	}
}

func TestGoEnvSetsTheReleaseSettings(t *testing.T) {
	host := []string{
		"PATH=/bin", "CGO_ENABLED=1", "GOAMD64=v3", "GOARM64=v9.0", "GOFLAGS=-race -tags=x",
		"GOCACHE=/host/cache", "GOOS=windows", "GIT_DIR=/elsewhere", "GOCACHEPROG=x",
	}
	env := goEnv(host, []string{"GOOS=linux", "GOARCH=arm64", "GOCACHE=/fresh"})
	value := func(name string) []string {
		var vs []string
		for _, kv := range env {
			if v, ok := strings.CutPrefix(kv, name+"="); ok {
				vs = append(vs, v)
			}
		}
		return vs
	}
	for name, want := range map[string]string{
		"CGO_ENABLED": "0", "GOAMD64": "v1", "GOARM64": "v8.0", "GOFLAGS": "-mod=readonly",
		"GOCACHE": "/fresh", "GOOS": "linux", "GOARCH": "arm64", "PATH": "/bin",
		"GIT_CONFIG_NOSYSTEM": "1", "GIT_CONFIG_GLOBAL": os.DevNull,
	} {
		if got := value(name); !slices.Equal(got, []string{want}) {
			t.Errorf("%s is %q, want exactly %q", name, got, want)
		}
	}
	if got := value("GIT_DIR"); got != nil {
		t.Errorf("GIT_DIR is %q, want it dropped", got)
	}
}

func TestCLI(t *testing.T) {
	for _, args := range [][]string{nil, {"-version"}, {"-version", "1.0.0", "extra"}, {"-unknown"}} {
		var stdout, stderr bytes.Buffer
		if code := cli(args, &stdout, &stderr, nil); code != 2 {
			t.Errorf("cli(%q) exited %d, want 2", args, code)
		}
		if stdout.Len() > 0 {
			t.Errorf("cli(%q) wrote %q to stdout", args, stdout.String())
		}
	}
}

// TestHarnessBuildsByTheReleaseLine holds the SC7 harness's release build
// to buildline's: the runner may link nothing of the implementation
// (ADR-011 D11 item 2), so it states the line itself, and this test reads
// it there.
func TestHarnessBuildsByTheReleaseLine(t *testing.T) {
	src := readFile(t, filepath.Join("..", "..", "conformance", "hostisolation_test.go"))
	args := buildline.Args("0.0.0-sc7", "OUT")
	flags := args[1:slices.Index(args, "-o")]
	quote := func(ss []string) string {
		q := make([]string, len(ss))
		for i, s := range ss {
			q[i] = strconv.Quote(s)
		}
		return strings.Join(q, ", ")
	}
	for _, want := range []string{
		"releaseFlags    = []string{" + quote(flags) + "}",
		"releaseSettings = []string{" + quote(buildline.Settings) + "}",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("internal/conformance/hostisolation_test.go does not state %s", want)
		}
	}
}
