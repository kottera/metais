// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/inputs"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/testenv"
)

// vectors is the conformance suite's vectors, from this package's
// directory.
const vectors = "../../spec/conformance/vectors"

// plainTypes are the only Go types a binding may hold (ADR-009 D13 item
// 2): the data model's, and nothing a template could call.
var plainTypes = []reflect.Type{
	reflect.TypeFor[value.Null](), reflect.TypeFor[value.Bool](), reflect.TypeFor[value.Number](),
	reflect.TypeFor[value.String](), reflect.TypeFor[value.Array](), reflect.TypeFor[value.Object](),
}

// plain walks v and reports the first value whose type is not one of
// plainTypes, by its path.
func plain(v value.Value, at string) string {
	t := reflect.TypeOf(v)
	ok := false
	for _, p := range plainTypes {
		ok = ok || t == p
	}
	if !ok {
		return at + " is a " + t.String()
	}
	switch v := v.(type) {
	case value.Array:
		for i, e := range v.All() {
			if bad := plain(e, at+"["+strconv.Itoa(i)+"]"); bad != "" {
				return bad
			}
		}
	case value.Object:
		for name, m := range v.All() {
			if bad := plain(m, at+"."+name); bad != "" {
				return bad
			}
		}
	}
	return ""
}

// renderVector is what a test reads of a render vector's first step.
type renderVector struct {
	dir  string
	args struct {
		Package string   `json:"package"`
		Data    string   `json:"data"`
		Now     string   `json:"now"`
		Set     []string `json:"set"`
		SetJSON []string `json:"set_json"`
	}
}

// renderVectors returns every vector of the suite whose steps render with
// inputs that validate.
func renderVectors(t *testing.T) []renderVector {
	t.Helper()
	files, err := filepath.Glob(filepath.Join(vectors, "*", "*", "vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []renderVector
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			Cases []struct {
				Steps []struct {
					Op   string          `json:"op"`
					Args json.RawMessage `json:"args"`
				} `json:"steps"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatal(err)
		}
		for _, c := range v.Cases {
			for _, st := range c.Steps {
				if st.Op != "render" {
					continue
				}
				rv := renderVector{dir: filepath.Dir(f)}
				if err := json.Unmarshal(st.Args, &rv.args); err != nil {
					t.Fatal(err)
				}
				out = append(out, rv)
			}
		}
	}
	return out
}

// TestBindingsArePlainData is ADR-009 D13 item 3: the binding set built
// for every render of the suite whose package opens and whose inputs
// validate holds the data model's plain values and nothing else, at every
// depth, so no template can reach a Go method through it; and it is the
// closed set of globals ADR-004 D7 names.
func TestBindingsArePlainData(t *testing.T) {
	testenv.Home(t)
	built := 0
	for _, rv := range renderVectors(t) {
		lim := limits.Defaults()
		pkg, fs := loader.Open(filepath.Join(rv.dir, filepath.FromSlash(rv.args.Package)), loader.Options{Limits: lim}, 0)
		if pkg == nil {
			continue
		}
		layer, err := inputs.New(inputs.Package{Manifest: pkg.Manifest, Compiler: pkg.Schemas, Combined: pkg.CombinedValue, Schema: pkg.Combined})
		if err != nil {
			t.Fatal(err)
		}
		sup := layer.NewSupply(lim)
		if rv.args.Data != "" {
			if _, err := os.Stat(filepath.Join(rv.dir, filepath.FromSlash(rv.args.Data))); err != nil {
				// A data directory an earlier step of the case writes,
				// as sample does, is not in the vector.
				continue
			}
			if err := sup.ReadDataDir(filepath.Join(rv.dir, filepath.FromSlash(rv.args.Data))); err != nil {
				t.Fatal(err)
			}
		}
		for _, s := range rv.args.Set {
			sup.AddSet(s)
		}
		for _, s := range rv.args.SetJSON {
			sup.AddSetJSON(s)
		}
		res, err := sup.Validate()
		if err != nil {
			t.Fatal(err)
		}
		if len(res.Findings) > 0 || len(fs) > 0 {
			continue
		}
		b, err := NewBindings(Package{Tree: pkg.Tree, Manifest: pkg.Manifest, ManifestValue: pkg.ManifestValue,
			Digest: pkg.Digest, Combined: pkg.CombinedValue}, res.Inputs, rv.args.Now, lim)
		if err != nil {
			t.Fatal(err)
		}
		g := b.Each("a/b.txt")
		if bad := plain(g, "globals"); bad != "" {
			t.Errorf("%s: %s", rv.dir, bad)
		}
		if names := g.Names(); strings.Join(names, ",") != "inputs,now,output,package,sources" {
			t.Errorf("%s: the globals are %q", rv.dir, names)
		}
		pkg.Close()
		built++
	}
	if built == 0 {
		t.Fatal("no render vector built a binding set")
	}
}
