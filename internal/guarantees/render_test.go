// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/fault"
)

// renderStep is one render step of the conformance suite, as the
// confinement and determinism detectors replay it (ADR-010 D13 item 4).
type renderStep struct {
	id    string // the vector's id and the case's name
	dir   string // the vector's directory
	tags  []string
	setup string // the vector tree copied into the work directory, or ""
	// dirs and unwritable are the setup's directories, the second made
	// unwritable once everything else is in place (ADR-011 D9 item 4);
	// symlinks and fifos are its links and named pipes.
	dirs, unwritable []string
	symlinks         []setupLink
	fifos            []string
	args             renderArgs
	ok               bool
	trees            []expectedTree
	// stream is the vector file the step's stream equals, or "".
	stream string
}

// setupLink is a link a vector's setup makes: at path, to target.
type setupLink struct {
	Path   string `json:"path"`
	Target string `json:"target"`
}

type renderArgs struct {
	Package   string         `json:"package"`
	Overwrite bool           `json:"overwrite"`
	Data      string         `json:"data"`
	Out       string         `json:"out"`
	Now       string         `json:"now"`
	Set       []string       `json:"set"`
	SetJSON   []string       `json:"set_json"`
	Limits    map[string]any `json:"limits"`
	// InputsJSON names a file of the vector holding an --inputs-json
	// argument's JSON text.
	InputsJSON string `json:"inputs_json"`
	FailOn     string `json:"fail_on"`
	// Stream is "archive", or {"only": <path>}: the render writes to
	// standard output.
	Stream json.RawMessage `json:"stream"`
	Only   json.RawMessage `json:"only"`
	DryRun bool            `json:"dry_run"`
	// Sample is a render from sample data's options (ADR-011 D8).
	Sample *struct {
		Optional string          `json:"optional"`
		Seed     json.RawMessage `json:"seed"`
		Items    json.RawMessage `json:"items"`
	} `json:"sample"`
}

// streams reports whether the step writes bytes to standard output.
func (a renderArgs) streams() bool {
	return len(a.Stream) > 0 && !a.DryRun
}

// only is every path the step's only argument selects.
func (a renderArgs) only() []string {
	if len(a.Only) == 0 {
		return nil
	}
	var one string
	if json.Unmarshal(a.Only, &one) == nil {
		return []string{one}
	}
	var many []string
	_ = json.Unmarshal(a.Only, &many)
	return many
}

type expectedTree struct {
	Path   string `json:"path"`
	Equals string `json:"equals"`
	// Substitute is the files in which the tool's name and version stand
	// as ADR-011 D10 item 3's tokens.
	Substitute []string `json:"substitute"`
}

// renderSteps returns every render step of the suite whose case is its
// first step: into a directory or to standard output, as a dry run or
// not, of one file or of all, into a new directory or an existing one with
// or without --overwrite, from data or from samples: every render the tool
// has.
func renderSteps(t *testing.T) []renderStep {
	t.Helper()
	root := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors")
	files, err := filepath.Glob(filepath.Join(root, "*", "*", "vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []renderStep
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			ID    string   `json:"id"`
			Tags  []string `json:"tags"`
			Cases []struct {
				Name  string `json:"name"`
				Setup *struct {
					Copy       *string     `json:"copy"`
					Dirs       []string    `json:"dirs"`
					Unwritable []string    `json:"unwritable"`
					Symlinks   []setupLink `json:"symlinks"`
					Fifos      []string    `json:"fifos"`
				} `json:"setup"`
				Steps []struct {
					Op     string          `json:"op"`
					Args   json.RawMessage `json:"args"`
					Expect struct {
						OK     bool           `json:"ok"`
						Trees  []expectedTree `json:"trees"`
						Stream *struct {
							Equals string `json:"equals"`
						} `json:"stream"`
					} `json:"expect"`
				} `json:"steps"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, c := range v.Cases {
			if len(c.Steps) != 1 || c.Steps[0].Op != "render" {
				continue
			}
			st := renderStep{id: v.ID + "/" + c.Name, dir: filepath.Dir(f), tags: v.Tags, ok: c.Steps[0].Expect.OK, trees: c.Steps[0].Expect.Trees}
			if e := c.Steps[0].Expect.Stream; e != nil {
				st.stream = e.Equals
			}
			if c.Setup != nil {
				if c.Setup.Copy != nil {
					st.setup = *c.Setup.Copy
				}
				st.dirs, st.unwritable = c.Setup.Dirs, c.Setup.Unwritable
				st.symlinks, st.fifos = c.Setup.Symlinks, c.Setup.Fifos
			}
			if err := json.Unmarshal(c.Steps[0].Args, &st.args); err != nil {
				t.Fatal(err)
			}
			out = append(out, st)
		}
	}
	if len(out) == 0 {
		t.Fatal("the suite holds no render step")
	}
	return out
}

// prepare makes the work directory of st below base: <base>/parent, with
// the vector's setup tree and the package and data trees the step names,
// and returns it; or "" when this host cannot make a directory the setup
// names unwritable, as root cannot, or a link or a named pipe it names,
// and the step cannot be replayed, as the runner's probes skip it.
func (st renderStep) prepare(t *testing.T, base string) string {
	t.Helper()
	work := filepath.Join(base, "parent")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if st.setup != "" {
		copyTree(t, filepath.Join(st.dir, filepath.FromSlash(st.setup)), work)
	}
	for _, d := range append(slices.Clone(st.dirs), st.unwritable...) {
		if err := os.MkdirAll(filepath.Join(work, filepath.FromSlash(d)), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, l := range st.symlinks {
		p := filepath.Join(work, filepath.FromSlash(l.Path))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(filepath.FromSlash(l.Target), p); err != nil {
			return ""
		}
	}
	for _, f := range st.fifos {
		p := filepath.Join(work, filepath.FromSlash(f))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := mkfifo(p); err != nil {
			return ""
		}
	}
	for _, p := range []string{st.args.Package, st.args.Data} {
		if p != "" {
			copyTree(t, filepath.Join(st.dir, filepath.FromSlash(p)), filepath.Join(work, filepath.FromSlash(p)))
		}
	}
	for _, d := range st.unwritable {
		p := filepath.Join(work, filepath.FromSlash(d))
		if err := os.Chmod(p, 0o555); err != nil {
			t.Fatal(err)
		}
		t.Cleanup(func() { os.Chmod(p, 0o755) })
		if f, err := os.Create(filepath.Join(p, "probe")); err == nil {
			f.Close()
			os.Remove(f.Name())
			return ""
		}
	}
	return work
}

// argv is the command line of st in work, every path absolute, and an
// inputs_json file's text given as --inputs-json's value.
func (st renderStep) argv(work string) []string {
	abs := func(p string) string { return filepath.Join(work, filepath.FromSlash(p)) }
	a := []string{"render", "--json", abs(st.args.Package)}
	if len(st.args.Stream) > 0 {
		a = append(a, "--out", "-")
		var o struct {
			Only string `json:"only"`
		}
		if json.Unmarshal(st.args.Stream, &o) == nil && o.Only != "" {
			a = append(a, "--only", o.Only)
		}
	} else {
		a = append(a, "--out", abs(st.args.Out))
	}
	for _, o := range st.args.only() {
		a = append(a, "--only", o)
	}
	if st.args.DryRun {
		a = append(a, "--dry-run")
	}
	if st.args.Data != "" {
		a = append(a, "--data", abs(st.args.Data))
	}
	if st.args.Overwrite {
		a = append(a, "--overwrite")
	}
	if st.args.Now != "" {
		a = append(a, "--now", st.args.Now)
	}
	if st.args.FailOn != "" {
		a = append(a, "--fail-on", st.args.FailOn)
	}
	if sm := st.args.Sample; sm != nil {
		a = append(a, "--sample")
		if sm.Optional != "" {
			a = append(a, "--optional", sm.Optional)
		}
		if len(sm.Seed) > 0 {
			a = append(a, "--seed", string(sm.Seed))
		}
		if len(sm.Items) > 0 {
			a = append(a, "--items", string(sm.Items))
		}
	}
	for _, s := range st.args.Set {
		a = append(a, "--set", s)
	}
	for _, s := range st.args.SetJSON {
		a = append(a, "--set-json", s)
	}
	if st.args.InputsJSON != "" {
		b, err := os.ReadFile(filepath.Join(st.dir, filepath.FromSlash(st.args.InputsJSON)))
		if err != nil {
			panic("guarantees: reading " + st.id + "'s inputs_json: " + err.Error())
		}
		a = append(a, "--inputs-json", string(b))
	}
	names := make([]string, 0, len(st.args.Limits))
	for k := range st.args.Limits {
		names = append(names, k)
	}
	slices.Sort(names)
	for _, k := range names {
		a = append(a, "--limit", fmt.Sprintf("%s=%v", k, st.args.Limits[k]))
	}
	return a
}

// run renders st in-process in work under faults and reports whether it
// succeeded, and the bytes it wrote to standard output when it streams,
// whose document is then on standard error (ADR-008 D11 item 2).
func (st renderStep) run(t *testing.T, work string, faults fault.Set) (bool, []byte) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cli.Run(context.Background(), cli.Invocation{
		Args: st.argv(work), Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	doc, stream := &stdout, []byte(nil)
	if st.args.streams() {
		doc, stream = &stderr, stdout.Bytes()
	}
	var d struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(doc.Bytes(), &d); err != nil {
		t.Fatalf("%s: %v: %q", st.id, err, doc.String())
	}
	return d.OK, stream
}

// snapEntry is one path of a snapshot: its type and, for a file, its digest.
type snapEntry struct {
	mode   fs.FileMode
	digest string
}

// renderSnapshot records every path below root.
func renderSnapshot(t *testing.T, root string) map[string]snapEntry {
	t.Helper()
	out := map[string]snapEntry{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		e := snapEntry{mode: d.Type()}
		if d.Type().IsRegular() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			e.digest = hex.EncodeToString(sum[:])
		}
		out[filepath.ToSlash(rel)] = e
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// changed returns every path that differs between before and after, in
// order.
func changed(before, after map[string]snapEntry) []string {
	var out []string
	for p, e := range after {
		if b, ok := before[p]; !ok || b != e {
			out = append(out, p)
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// under reports whether p is dir or beneath it.
func under(p, dir string) bool {
	return p == dir || strings.HasPrefix(p, dir+"/")
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, p)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(target, b, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
}
