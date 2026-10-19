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
	args  renderArgs
	ok    bool
	trees []expectedTree
}

type renderArgs struct {
	Package string         `json:"package"`
	Data    string         `json:"data"`
	Out     string         `json:"out"`
	Now     string         `json:"now"`
	Set     []string       `json:"set"`
	SetJSON []string       `json:"set_json"`
	Limits  map[string]any `json:"limits"`
}

type expectedTree struct {
	Path   string `json:"path"`
	Equals string `json:"equals"`
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
					Copy *string `json:"copy"`
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
			if c.Setup != nil && c.Setup.Copy != nil {
				st.setup = *c.Setup.Copy
			}
			var all map[string]any
			if err := json.Unmarshal(c.Steps[0].Args, &all); err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(c.Steps[0].Args, &st.args); err != nil {
				t.Fatal(err)
			}
			for _, k := range []string{"stream", "only", "overwrite", "dry_run", "sample", "inputs_json"} {
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
// and returns it.
func (st renderStep) prepare(t *testing.T, base string) string {
	t.Helper()
	work := filepath.Join(base, "parent")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if st.setup != "" {
		copyTree(t, filepath.Join(st.dir, filepath.FromSlash(st.setup)), work)
	}
	for _, p := range []string{st.args.Package, st.args.Data} {
		if p != "" {
			copyTree(t, filepath.Join(st.dir, filepath.FromSlash(p)), filepath.Join(work, filepath.FromSlash(p)))
		}
	}
	return work
}

// argv is the command line of st in work, every path absolute.
func (st renderStep) argv(work string) []string {
	abs := func(p string) string { return filepath.Join(work, filepath.FromSlash(p)) }
	a := []string{"render", "--json", abs(st.args.Package), "--out", abs(st.args.Out)}
	if st.args.Data != "" {
		a = append(a, "--data", abs(st.args.Data))
	}
	if st.args.Now != "" {
		a = append(a, "--now", st.args.Now)
	}
	for _, s := range st.args.Set {
		a = append(a, "--set", s)
	}
	for _, s := range st.args.SetJSON {
		a = append(a, "--set-json", s)
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
