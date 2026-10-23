// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/tarw"
	"metais.org/metais/internal/tools/release/buildline"
)

// The builder's refusals, each wrapped with what was found.
var (
	errVersion         = errors.New("the version is not a semantic version without a leading v and without build metadata")
	errDirty           = errors.New("the working tree is not clean")
	errNotTag          = errors.New("HEAD is not the release's tag")
	errDistExists      = errors.New("dist/ already exists")
	errSettings        = errors.New("a binary's go version -m does not show the release's build settings")
	errNotReproducible = errors.New("the two builds of a target differ")
	errToolVersion     = errors.New("the built tool does not report the release's version")
	errUnreleased      = errors.New("the tool's implements names an unreleased specification revision")
	errNoToolchain     = errors.New("go.mod has no single toolchain line naming a go release")
)

// The files every archive holds beside the binary, from the work tree's
// root (ADR-010 D12 item 5).
var noticeFiles = []string{"LICENSE", "NOTICE", "THIRD-PARTY-NOTICES.txt"}

// binaryName is the binary's name in each archive.
const binaryName = "metais"

// copies is how many times each target is built.
const copies = 2

// releaseProxy and releaseSumDB are the module proxy and checksum database
// a release fetches go.mod's toolchain and the modules go.sum pins from,
// the go command's own defaults whatever the developer configured.
const (
	releaseProxy = "https://proxy.golang.org,direct"
	releaseSumDB = "sum.golang.org"
)

// config is one run of the builder.
type config struct {
	dir     string // where git is asked for the work tree; "" is the working directory
	version string // the release's semantic version, without a leading v
	stdout  io.Writer
	environ []string // the environment the builder was started with
	proxy   string   // the GOPROXY every go command takes: releaseProxy, or off in a test

	// toolchain is the toolchain go.mod's toolchain line names, which every
	// go command runs as (ADR-010 D1). run reads it from the first clone.
	toolchain string

	// tamper, when set, is called on every binary once it is built and
	// before it is judged. Only tests set it, to make a build differ.
	tamper func(t buildline.Target, copy int, bin string) error
}

// run builds the release and writes dist/, or returns the refusal that
// stopped it. It writes nothing outside a temporary directory it removes,
// until every check has passed.
func (c *config) run() error {
	if !releaseVersion(c.version) {
		return fmt.Errorf("%w: %q", errVersion, c.version)
	}
	root, err := c.git(c.dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return err
	}
	status, err := c.git(root, "status", "--porcelain=v1", "--untracked-files=all", "--ignore-submodules=none")
	if err != nil {
		return err
	}
	if status != "" {
		return fmt.Errorf("%w:\n%s", errDirty, status)
	}
	head, err := c.git(root, "rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	tag := "v" + c.version
	tagged, err := c.git(root, "rev-parse", "--verify", "--quiet", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		return fmt.Errorf("%w: there is no tag %s", errNotTag, tag)
	}
	if tagged != head {
		return fmt.Errorf("%w: HEAD is %s and %s is %s", errNotTag, head, tag, tagged)
	}
	dist := filepath.Join(root, "dist")
	if _, err := os.Lstat(dist); err == nil {
		return fmt.Errorf("%w: %s; move it away before building another release", errDistExists, dist)
	}

	work, err := os.MkdirTemp("", "metais-release-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	var srcs [copies]string
	for i := range copies {
		srcs[i] = filepath.Join(work, "copy"+strconv.Itoa(i), "src")
		if _, err := c.git("", "clone", "--quiet", "--no-checkout", "--no-hardlinks", "--", root, srcs[i]); err != nil {
			return err
		}
		if _, err := c.git(srcs[i], "checkout", "--quiet", "--detach", head); err != nil {
			return err
		}
	}

	gomod, err := os.ReadFile(filepath.Join(srcs[0], "go.mod"))
	if err != nil {
		return err
	}
	if c.toolchain, err = goModToolchain(gomod); err != nil {
		return err
	}

	if err := c.checkImplements(root, srcs[0]); err != nil {
		return err
	}

	binaries := map[buildline.Target][]byte{}
	for _, t := range buildline.Shipped {
		var sums [copies][sha256.Size]byte
		for i := range copies {
			fmt.Fprintf(c.stdout, "building %s, copy %d of %d\n", t, i+1, copies)
			bin := filepath.Join(filepath.Dir(srcs[i]), "bin", t.OS+"_"+t.Arch, binaryName)
			if err := c.build(srcs[i], t, bin); err != nil {
				return err
			}
			if c.tamper != nil {
				if err := c.tamper(t, i, bin); err != nil {
					return err
				}
			}
			info, err := c.goCmd(srcs[i], nil, "version", "-m", bin)
			if err != nil {
				return err
			}
			if err := checkSettings(info, t, head, c.toolchain); err != nil {
				return fmt.Errorf("%s, copy %d: %w", t, i+1, err)
			}
			b, err := os.ReadFile(bin)
			if err != nil {
				return err
			}
			sums[i] = sha256.Sum256(b)
			binaries[t] = b
		}
		if sums[0] != sums[1] {
			return fmt.Errorf("%w: %s built to SHA-256 %x and %x", errNotReproducible, t, sums[0], sums[1])
		}
	}

	notices := map[string][]byte{}
	for _, name := range noticeFiles {
		b, err := os.ReadFile(filepath.Join(srcs[0], name))
		if err != nil {
			return err
		}
		notices[name] = b
	}
	archives := map[string][]byte{}
	for _, t := range buildline.Shipped {
		b, err := archive(binaries[t], notices)
		if err != nil {
			return err
		}
		archives[buildline.ArchiveName(c.version, t)] = b
	}
	return writeDist(dist, archives, c.stdout)
}

// build builds the release of target t from the clone src to bin, with the
// clone's own GOCACHE beside it.
func (c *config) build(src string, t buildline.Target, bin string) error {
	cache := filepath.Join(filepath.Dir(src), "gocache")
	_, err := c.goCmd(src, []string{"GOOS=" + t.OS, "GOARCH=" + t.Arch, "GOCACHE=" + cache}, buildline.Args(c.version, bin)...)
	return err
}

// checkImplements builds the tool for the host from src, asks it what it
// is, and refuses a tool that does not report the release's version, that
// implements nothing or something that is not a revision, or, unless the
// version is a prerelease, whose implements names a specification revision
// root's history has not released (ADR-011 D5 items 2 and 6, A8 item 5). A
// prerelease's unreleased revisions are reported on stdout.
func (c *config) checkImplements(root, src string) error {
	host, err := c.goCmd(src, nil, "env", "GOHOSTOS", "GOHOSTARCH")
	if err != nil {
		return err
	}
	hostOS, hostArch, _ := strings.Cut(host, "\n")
	bin := filepath.Join(filepath.Dir(src), "host", binaryName)
	if err := c.build(src, buildline.Target{OS: hostOS, Arch: hostArch}, bin); err != nil {
		return err
	}
	cmd := exec.Command(bin, "version", "--json")
	cmd.Dir = filepath.Dir(bin)
	cmd.Env = []string{}
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("%s version --json: %w", binaryName, err)
	}
	var v struct {
		Result struct {
			Tool struct {
				Version string `json:"version"`
			} `json:"tool"`
			Specification struct {
				Implements []string `json:"implements"`
			} `json:"specification"`
		} `json:"result"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return fmt.Errorf("%s version --json: %w", binaryName, err)
	}
	if got := v.Result.Tool.Version; got != c.version {
		return fmt.Errorf("%w: it reports %q", errToolVersion, got)
	}
	if len(v.Result.Specification.Implements) == 0 {
		return fmt.Errorf("%w: it implements nothing", errUnreleased)
	}
	pre := prerelease(c.version)
	for _, rev := range v.Result.Specification.Implements {
		if !revision(rev) {
			return fmt.Errorf("%w: %s: not a revision <major>.<minor>", errUnreleased, rev)
		}
		err := c.released(root, rev)
		switch {
		case err != nil && pre:
			fmt.Fprintf(c.stdout, "the prerelease %s implements %s, which is not released: %v\n", c.version, rev, err)
		case err != nil:
			return fmt.Errorf("%w: %s: %v", errUnreleased, rev, err)
		}
	}
	return nil
}

// prerelease reports whether the release version v has a prerelease
// suffix, as 1.0.0-rc.1 has. A prerelease may implement a revision that is
// not yet released, so a release candidate can be rehearsed end to end
// before the specification's release commit (ADR-011 A8 item 5).
func prerelease(v string) bool {
	return strings.Contains(v, "-")
}

// revision reports whether rev is a revision <major>.<minor>.
func revision(rev string) bool {
	major, minor, ok := strings.Cut(rev, ".")
	return ok && numeral(major) && numeral(minor)
}

// released reports why rev, a revision <major>.<minor>, is not a released
// revision of the specification in root's history at HEAD, or nil when it is: an annotated
// tag spec-<rev> on an ancestor of HEAD, and the released list
// spec/conformance/released/<rev>.sha256 in HEAD's tree (ADR-011 D5 item 2).
func (c *config) released(root, rev string) error {
	tag := "refs/tags/spec-" + rev
	if kind, err := c.git(root, "cat-file", "-t", tag); err != nil || kind != "tag" {
		return fmt.Errorf("there is no annotated tag spec-%s", rev)
	}
	if _, err := c.git(root, "merge-base", "--is-ancestor", tag+"^{commit}", "HEAD"); err != nil {
		return fmt.Errorf("spec-%s is not an ancestor of HEAD", rev)
	}
	list := "spec/conformance/released/" + rev + ".sha256"
	if _, err := c.git(root, "cat-file", "-e", "HEAD:"+list); err != nil {
		return fmt.Errorf("HEAD has no %s", list)
	}
	return nil
}

// checkSettings judges go version -m's report on a binary built for t: it
// must have been built by toolchain, go.mod's (ADR-010 D1), and the build
// settings must be exactly the release's (ADR-010 D12 items 1, 2 and 4),
// from a clean tree at head, and nothing else. Under -trimpath the go
// command records no -ldflags, so the version the release sets is judged
// by asking the host's build (checkImplements).
func checkSettings(info string, t buildline.Target, head, toolchain string) error {
	want := map[string]string{
		"-buildmode":   "exe",
		"-compiler":    "gc",
		"-trimpath":    "true",
		"CGO_ENABLED":  "0",
		"GOOS":         t.OS,
		"GOARCH":       t.Arch,
		"vcs":          "git",
		"vcs.revision": head,
		"vcs.modified": "false",
	}
	switch t.Arch {
	case "amd64":
		want["GOAMD64"] = "v1"
	case "arm64":
		want["GOARM64"] = "v8.0"
	}
	var bad []string
	first, _, _ := strings.Cut(info, "\n")
	built := ""
	if i := strings.LastIndex(first, ": "); i >= 0 {
		built = first[i+len(": "):]
	}
	if built != toolchain {
		bad = append(bad, fmt.Sprintf("was built by %s, not go.mod's %s", built, toolchain))
	}
	got := map[string]string{}
	for line := range strings.Lines(info) {
		rest, ok := strings.CutPrefix(strings.TrimRight(line, "\n"), "\tbuild\t")
		if !ok {
			continue
		}
		k, v, _ := strings.Cut(rest, "=")
		if strings.HasPrefix(v, `"`) {
			if u, err := strconv.Unquote(v); err == nil {
				v = u
			}
		}
		got[k] = v
	}
	for k, w := range want {
		if g, ok := got[k]; !ok {
			bad = append(bad, fmt.Sprintf("lacks %s=%s", k, w))
		} else if g != w {
			bad = append(bad, fmt.Sprintf("has %s=%s, not %s", k, g, w))
		}
	}
	for k, g := range got {
		if _, ok := want[k]; !ok && k != "vcs.time" {
			bad = append(bad, fmt.Sprintf("has %s=%s, which a release does not set", k, g))
		}
	}
	if bad != nil {
		slices.Sort(bad)
		return fmt.Errorf("%w: it %s", errSettings, strings.Join(bad, "; "))
	}
	return nil
}

// archive is one target's .tar.gz (ADR-010 D12 item 5): the notice files
// and the binary, in byte order of their names, each header written by
// internal/tarw from its name, executable bit and size alone (mode 0755
// for the binary and 0644 otherwise, uid and gid 0, empty owner names,
// ADR-001's fixed instant), under a gzip header with no name and a zero
// mtime.
func archive(binary []byte, notices map[string][]byte) ([]byte, error) {
	files := map[string][]byte{binaryName: binary}
	for name, b := range notices {
		files[name] = b
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)

	var buf bytes.Buffer
	zw := gzip.NewWriter(&buf)
	tw := tarw.NewWriter(zw)
	for _, name := range names {
		b := files[name]
		if err := tw.Add(name, name == binaryName, int64(len(b)), bytes.NewReader(b)); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// writeDist creates dist, which must not exist, and writes each archive
// and SHA256SUMS into it, one "<sha256>  <name>" line per archive in byte
// order of the name, as sha256sum writes them. It reports SHA256SUMS on
// report. On a failure it removes dist.
func writeDist(dist string, archives map[string][]byte, report io.Writer) (err error) {
	if err := os.Mkdir(dist, 0o755); err != nil {
		return fmt.Errorf("%w: %v", errDistExists, err)
	}
	defer func() {
		if err != nil {
			os.RemoveAll(dist)
		}
	}()
	names := make([]string, 0, len(archives))
	for name := range archives {
		names = append(names, name)
	}
	slices.Sort(names)
	var sums strings.Builder
	for _, name := range names {
		sum := sha256.Sum256(archives[name])
		fmt.Fprintf(&sums, "%s  %s\n", hex.EncodeToString(sum[:]), name)
		if err := os.WriteFile(filepath.Join(dist, name), archives[name], 0o644); err != nil {
			return err
		}
	}
	if err := os.WriteFile(filepath.Join(dist, "SHA256SUMS"), []byte(sums.String()), 0o644); err != nil {
		return err
	}
	_, err = io.WriteString(report, sums.String())
	return err
}

// git runs git in dir ("" is the working directory) under gitEnv, and
// returns its standard output without its final newline.
func (c *config) git(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv(c.environ)
	return output(cmd)
}

// goCmd runs the go command in dir with the release's environment and
// set's variables, and returns its standard output without its final
// newline.
func (c *config) goCmd(dir string, set []string, args ...string) (string, error) {
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	cmd.Env = goEnv(c.environ, c.toolchain, c.proxy, set)
	return output(cmd)
}

// output runs cmd, and names it and its standard error when it fails.
func output(cmd *exec.Cmd) (string, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSuffix(stdout.String(), "\n"), nil
}

// gitEnv is environ with nothing of the host's able to change what git
// reports: every GIT_* variable dropped, and no system or global
// configuration, attributes or excludes file read. Variables given this
// way outrank every configuration file.
func gitEnv(environ []string) []string {
	env := slices.DeleteFunc(slices.Clone(environ), func(kv string) bool {
		return strings.HasPrefix(kv, "GIT_")
	})
	return append(env,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_COUNT=3",
		"GIT_CONFIG_KEY_0=core.precomposeunicode",
		"GIT_CONFIG_VALUE_0=false",
		"GIT_CONFIG_KEY_1=core.excludesFile",
		"GIT_CONFIG_VALUE_1="+os.DevNull,
		"GIT_CONFIG_KEY_2=core.attributesFile",
		"GIT_CONFIG_VALUE_2="+os.DevNull,
	)
}

// goEnv is gitEnv's environment, for the git the go command runs to stamp
// the build, with buildline's settings, and set's variables, each
// replacing any of the same name. Nothing of the developer's go
// configuration reaches the build: GOFLAGS is fixed so no flag of the
// host's joins it, GOTOOLCHAIN is toolchain, go.mod's, GOWORK and GOENV
// are off so no workspace and no go env file is read, and the module proxy
// is proxy and the checksum database releaseSumDB, with no module exempt
// from either.
func goEnv(environ []string, toolchain, proxy string, set []string) []string {
	over := slices.Concat(buildline.Settings, []string{
		"GOFLAGS=-mod=readonly",
		"GOTOOLCHAIN=" + toolchain,
		"GOWORK=off",
		"GOENV=off",
		"GOPROXY=" + proxy,
		"GOSUMDB=" + releaseSumDB,
		"GOPRIVATE=", "GONOPROXY=", "GONOSUMDB=", "GOINSECURE=",
	}, set)
	env := slices.DeleteFunc(gitEnv(environ), func(kv string) bool {
		name, _, _ := strings.Cut(kv, "=")
		return slices.ContainsFunc(over, func(o string) bool { return strings.HasPrefix(o, name+"=") })
	})
	return append(env, over...)
}

// releaseVersion reports whether v is a semantic version as Semantic
// Versioning 2.0.0 defines it, MAJOR.MINOR.PATCH with an optional
// pre-release, without a leading v and without build metadata, which a
// Go module version cannot carry.
func releaseVersion(v string) bool {
	core, pre, hasPre := strings.Cut(v, "-")
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !numeral(p) {
			return false
		}
	}
	if !hasPre {
		return true
	}
	for _, id := range strings.Split(pre, ".") {
		if id == "" || strings.Trim(id, "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ-") != "" {
			return false
		}
		if strings.Trim(id, "0123456789") == "" && !numeral(id) {
			return false
		}
	}
	return true
}

// numeral reports whether s is a decimal numeral without a leading zero.
func numeral(s string) bool {
	if s == "" || strings.Trim(s, "0123456789") != "" {
		return false
	}
	return s == "0" || s[0] != '0'
}

// goModToolchain is the toolchain go.mod's one toolchain line names, a go
// release such as go1.26.9 (ADR-010 D1).
func goModToolchain(gomod []byte) (string, error) {
	var found []string
	for line := range strings.Lines(string(gomod)) {
		line, _, _ = strings.Cut(line, "//")
		if f := strings.Fields(line); len(f) > 0 && f[0] == "toolchain" {
			found = append(found, strings.Join(f[1:], " "))
		}
	}
	if len(found) != 1 || !strings.HasPrefix(found[0], "go1.") || strings.ContainsAny(found[0], " ") {
		return "", fmt.Errorf("%w: it has %q", errNoToolchain, found)
	}
	return found[0], nil
}
