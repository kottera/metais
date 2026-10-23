// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"metais.org/metais/internal/tools/release/buildline"
)

// Where the site's sources are, below the checkout.
const (
	packageDir = "site/package" // the site package's source directory
	publicDir  = "site/public"  // the static files, served byte for byte at their paths
	specDir    = "spec"         // the specification, the draft's source
)

// schemaDirs are the schemas the site serves at their $ids: each source
// directory below the checkout, and the path below the site it is served
// at (ADR-013 D1 item 5).
var schemaDirs = []struct{ src, at string }{
	{"spec/schemas", "schemas"},
	{"docs/cli/schemas", "cli/schemas"},
}

var (
	errExists    = errors.New("the site is built into a new directory, and it exists")
	errShallow   = errors.New("the site of a release is built from a whole clone, and the checkout is a shallow one, which may lack the spec-<M>.<m> tags the site serves; fetch its whole history (actions/checkout's fetch-depth: 0)")
	errCollision = errors.New("a static file has the path of a page the package renders")
	errPublic    = errors.New("site/public holds a path the site serves from elsewhere")
	errVersion   = errors.New("the built metais does not report the release's version")
	errNotTag    = errors.New("the site of a release is built from the release's tag")
)

// builder builds the site from root into out.
type builder struct {
	root, out     string
	data          string // where to keep the exported inputs, or ""
	version, sums string // the release, or "" for a build with no release
	environ       []string
	report        io.Writer
}

// run builds the site. Everything is assembled in a work directory beside
// out, and moved to out in one rename at the end, so out either does not
// exist or holds the whole site.
func (b *builder) run() error {
	if _, err := os.Lstat(b.out); err == nil {
		return fmt.Errorf("%w: %s", errExists, b.out)
	}
	for _, d := range schemaDirs {
		top, _, _ := strings.Cut(d.at, "/")
		if _, err := os.Lstat(filepath.Join(b.root, filepath.FromSlash(publicDir), top)); err == nil {
			return fmt.Errorf("%w: %s/, where %s/ is served", errPublic, top, d.src)
		}
	}
	rel, err := b.releaseInput()
	if err != nil {
		return err
	}
	if rel.Published {
		if err := b.releasable(rel.Tag); err != nil {
			return err
		}
	}

	parent := filepath.Dir(b.out)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	work, err := os.MkdirTemp(parent, ".metais-site-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(work)

	metais := filepath.Join(work, "metais")
	fmt.Fprintln(b.report, "building metais")
	if err := b.buildMetais(metais); err != nil {
		return err
	}
	fmt.Fprintln(b.report, "exporting the inputs")
	data := filepath.Join(work, "data")
	if err := os.Mkdir(data, 0o755); err != nil {
		return err
	}
	inputs, err := b.inputs(metais, work, rel)
	if err != nil {
		return err
	}
	for _, name := range slices.Sorted(maps.Keys(inputs)) {
		if err := os.WriteFile(filepath.Join(data, name+".json"), inputs[name], 0o644); err != nil {
			return err
		}
	}

	if b.data != "" {
		if err := os.Mkdir(b.data, 0o755); err != nil {
			return err
		}
		if err := copyTree(data, b.data, ""); err != nil {
			return err
		}
	}

	fmt.Fprintln(b.report, "rendering", packageDir)
	site := filepath.Join(work, "site")
	if _, err := b.run1(metais, work, "render", filepath.Join(b.root, filepath.FromSlash(packageDir)), "--data", data, "--out", site); err != nil {
		return err
	}
	if err := copyTree(filepath.Join(b.root, filepath.FromSlash(publicDir)), site, ""); err != nil {
		return err
	}
	for _, d := range schemaDirs {
		if err := copyTree(filepath.Join(b.root, filepath.FromSlash(d.src)), site, d.at); err != nil {
			return err
		}
	}
	if _, err := os.Lstat(b.out); err == nil {
		return fmt.Errorf("%w: %s", errExists, b.out)
	}
	if err := os.Rename(site, b.out); err != nil {
		return err
	}
	fmt.Fprintln(b.report, "wrote", b.out)
	return nil
}

// releasable refuses a checkout the site of the release tagged tag cannot
// be built from: a shallow clone, which may lack the spec-<M>.<m> tags the
// site serves, and one whose HEAD is not the commit tag names.
func (b *builder) releasable(tag string) error {
	shallow, err := b.git("rev-parse", "--is-shallow-repository")
	if err != nil {
		return err
	}
	if string(shallow) != "false\n" {
		return errShallow
	}
	head, err := b.git("rev-parse", "--verify", "HEAD^{commit}")
	if err != nil {
		return err
	}
	tagged, err := b.git("rev-parse", "--verify", "--quiet", "refs/tags/"+tag+"^{commit}")
	if err != nil {
		return fmt.Errorf("%w: there is no tag %s", errNotTag, tag)
	}
	if !bytes.Equal(head, tagged) {
		return fmt.Errorf("%w: HEAD is %s and %s is %s", errNotTag, bytes.TrimSpace(head), tag, bytes.TrimSpace(tagged))
	}
	return nil
}

// buildMetais builds ./cmd/metais from the checkout to bin, for the host:
// as a release is built (ADR-010 D12 item 1) when the site is built for
// one, and otherwise with the static build's flags and the version Go
// stamps from the checkout.
func (b *builder) buildMetais(bin string) error {
	args := []string{"build", "-trimpath", "-buildvcs=true", "-o", bin, buildline.Package}
	if b.version != "" {
		args = buildline.Args(b.version, bin)
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = b.root
	cmd.Env = goEnv(b.environ)
	_, err := output(cmd)
	return err
}

// inputs runs metais at bin, in dir, and returns each input's JSON by its
// name.
func (b *builder) inputs(bin, dir string, rel release) (map[string][]byte, error) {
	help, err := b.result(bin, dir, "metais.help/1", "help", "--json")
	if err != nil {
		return nil, err
	}
	tool, err := b.result(bin, dir, "metais.version/1", "version", "--json")
	if err != nil {
		return nil, err
	}
	if rel.Published {
		var v struct {
			Tool struct {
				Version string `json:"version"`
			} `json:"tool"`
		}
		if err := json.Unmarshal(tool, &v); err != nil {
			return nil, err
		}
		if v.Tool.Version != rel.Version {
			return nil, fmt.Errorf("%w: it reports %q, and the release is %q", errVersion, v.Tool.Version, rel.Version)
		}
	}
	skillText, err := b.run1(bin, dir, "skill", "--print")
	if err != nil {
		return nil, err
	}
	sk, err := skillInput(skillText)
	if err != nil {
		return nil, err
	}
	sp, err := b.specInput()
	if err != nil {
		return nil, err
	}
	out := map[string][]byte{"help": help, "tool": tool}
	for name, v := range map[string]any{"skill": sk, "spec": sp, "release": rel} {
		if out[name], err = json.Marshal(v); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// result runs metais at bin with args, and returns the result of the JSON
// document it prints, which must be of format and report ok.
func (b *builder) result(bin, dir, format string, args ...string) ([]byte, error) {
	stdout, err := b.run1(bin, dir, args...)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Format string          `json:"format"`
		OK     bool            `json:"ok"`
		Result json.RawMessage `json:"result"`
	}
	if err := json.Unmarshal(stdout, &doc); err != nil {
		return nil, fmt.Errorf("metais %s: %w", strings.Join(args, " "), err)
	}
	if doc.Format != format || !doc.OK || len(doc.Result) == 0 {
		return nil, fmt.Errorf("metais %s: printed a %q document with ok %v; want a %s result", strings.Join(args, " "), doc.Format, doc.OK, format)
	}
	return doc.Result, nil
}

// run1 runs metais at bin with args in dir, with no environment at all:
// metais reads none but NO_COLOR, and nothing of the host reaches a
// template (ADR-009 D14). It returns the standard output.
func (b *builder) run1(bin, dir string, args ...string) ([]byte, error) {
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Env = []string{}
	return output(cmd)
}

// git runs git in the checkout under gitEnv, and returns its standard
// output.
func (b *builder) git(args ...string) ([]byte, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = b.root
	cmd.Env = gitEnv(b.environ)
	return output(cmd)
}

// output runs cmd, and names it and its standard error when it fails.
func output(cmd *exec.Cmd) ([]byte, error) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s: %w\n%s", strings.Join(cmd.Args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.Bytes(), nil
}

// gitEnv is environ with nothing of the host's able to change what git
// reports: every GIT_* variable dropped, and no system or global
// configuration read.
func gitEnv(environ []string) []string {
	env := slices.DeleteFunc(slices.Clone(environ), func(kv string) bool {
		return strings.HasPrefix(kv, "GIT_")
	})
	return append(env,
		"GIT_CONFIG_NOSYSTEM=1",
		"GIT_ATTR_NOSYSTEM=1",
		"GIT_CONFIG_GLOBAL="+os.DevNull,
		"GIT_CONFIG_COUNT=1",
		"GIT_CONFIG_KEY_0=core.precomposeunicode",
		"GIT_CONFIG_VALUE_0=false",
	)
}

// goEnv is gitEnv's environment, for the git the go command runs to stamp
// the build, with buildline's settings, no workspace, and no GOOS or
// GOARCH, so metais is built for the host that runs it.
func goEnv(environ []string) []string {
	env := slices.DeleteFunc(gitEnv(environ), func(kv string) bool {
		return strings.HasPrefix(kv, "GOOS=") || strings.HasPrefix(kv, "GOARCH=")
	})
	return slices.Concat(env, buildline.Settings, []string{"GOWORK=off"})
}

// copyTree copies every regular file below src into dst, below at (slash
// separated, "" for dst itself), each to the same path below it. A file
// whose path is already taken, by a page the package rendered or another
// copy, is refused, and so is anything but a directory or a regular file.
func copyTree(src, dst, at string) error {
	return filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, p)
		if err != nil {
			return err
		}
		to := filepath.Join(dst, filepath.FromSlash(at), rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(to, 0o755)
		case !d.Type().IsRegular():
			return fmt.Errorf("%s is not a regular file", p)
		}
		if _, err := os.Lstat(to); err == nil {
			return fmt.Errorf("%w: %s", errCollision, p)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(to, data, 0o644)
	})
}
