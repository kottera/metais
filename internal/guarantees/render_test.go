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
	// unwritable once everything else is in place (ADR-011 D9 item 4).
	dirs, unwritable []string
	args             renderArgs
	ok               bool
	trees            []expectedTree
}

type renderArgs struct {
	Package string         `json:"package"`
	Data    string         `json:"data"`
	Out     string         `json:"out"`
	Now     string         `json:"now"`
	Set     []string       `json:"set"`
	SetJSON []string       `json:"set_json"`
	Limits  map[string]any `json:"limits"`
	// InputsJSON names a file of the vector holding an --inputs-json
	// argument's JSON text.
	InputsJSON string `json:"inputs_json"`
	FailOn     string `json:"fail_on"`
}

type expectedTree struct {
	Path   string `json:"path"`
	Equals string `json:"equals"`
	// Substitute is the files in which the tool's name and version stand
	// as ADR-011 D10 item 3's tokens.
	Substitute []string `json:"substitute"`
}

// renderSteps returns every render step of the suite whose case is its
// first step, the steps that render into a directory with no stream,
// dry run or overwrite, which are the renders this wave has.
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
					Copy       *string  `json:"copy"`
					Dirs       []string `json:"dirs"`
					Unwritable []string `json:"unwritable"`
					Symlinks   []any    `json:"symlinks"`
					Fifos      []string `json:"fifos"`
				} `json:"setup"`
				Steps []struct {
					Op     string          `json:"op"`
					Args   json.RawMessage `json:"args"`
					Expect struct {
						OK    bool           `json:"ok"`
						Trees []expectedTree `json:"trees"`
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
			if c.Setup != nil {
				if len(c.Setup.Symlinks) > 0 || len(c.Setup.Fifos) > 0 {
					t.Fatalf("%s: a setup with links or pipes, which this detector does not replay", st.id)
				}
				if c.Setup.Copy != nil {
					st.setup = *c.Setup.Copy
				}
				st.dirs, st.unwritable = c.Setup.Dirs, c.Setup.Unwritable
			}
			var all map[string]any
			if err := json.Unmarshal(c.Steps[0].Args, &all); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(c.Steps[0].Args, &st.args); err != nil {
				t.Fatal(err)
			}
			for _, k := range []string{"stream", "only", "overwrite", "dry_run", "sample"} {
				if _, ok := all[k]; ok {
					t.Fatalf("%s: a render step with %s, which this detector does not replay", st.id, k)
				}
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
// names unwritable, as root cannot, and the step cannot be replayed, as
// the runner's unwritable-dirs probe skips it.
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
	a := []string{"render", "--json", abs(st.args.Package), "--out", abs(st.args.Out)}
	if st.args.Data != "" {
		a = append(a, "--data", abs(st.args.Data))
	}
	if st.args.Now != "" {
		a = append(a, "--now", st.args.Now)
	}
	if st.args.FailOn != "" {
		a = append(a, "--fail-on", st.args.FailOn)
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
// succeeded.
func (st renderStep) run(t *testing.T, work string, faults fault.Set) bool {
	t.Helper()
	var stdout, stderr bytes.Buffer
	cli.Run(context.Background(), cli.Invocation{
		Args: st.argv(work), Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	var doc struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
		t.Fatalf("%s: %v: %q", st.id, err, stdout.String())
	}
	return doc.OK
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
