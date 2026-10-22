// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/pkgtree/pkgtreetest"
	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/yamlx"
)

// pkg is a package as a test writes it: its input slots, as the manifest
// declares them, the path of its input constraints, and its schema files.
type pkg struct {
	inputs      []map[string]any
	constraints string
	schemas     map[string]string
	// lim is the limits the package is opened under, which bound its
	// validation's regex-work; the defaults when nil.
	lim *limits.Set
}

// slot is one input slot: name, title and schema, and the other fields
// given as key and value pairs.
func slot(name, schemaPath string, kv ...any) map[string]any {
	m := map[string]any{"name": name, "title": strings.ToUpper(name[:1]) + name[1:], "schema": schemaPath}
	for i := 0; i+1 < len(kv); i += 2 {
		m[kv[i].(string)] = kv[i+1]
	}
	return m
}

// decodeSchema reads a schema file by ADR-003 D9 or D10, as the loader's
// Decode does.
func decodeSchema(entry string, src []byte) (value.Value, []finding.Finding) {
	subj := finding.Subject{Kind: finding.SubjectInput, Name: "schema"}
	at := finding.Location{Origin: finding.OriginPackage, File: entry}
	if strings.HasSuffix(entry, ".json") {
		doc, fs := jsonx.Parse(src, jsonx.Source{Input: subj, At: at}, limits.Defaults())
		return doc.Value, fs
	}
	doc, fs := yamlx.Parse(src, yamlx.Source{Subject: subj, At: at, Size: limits.SchemaSize}, limits.Defaults())
	return doc.Value, fs
}

// open is ADR-003 D22 step 1 for p, built from Wave 2's libraries as the
// loader composes them: the manifest, every schema file compiled, and the
// combined schema bundled and compiled.
func open(t testing.TB, p pkg) Package {
	t.Helper()
	m := map[string]any{
		"metais": "1.0", "name": "org.example.test", "version": "1.0.0", "title": "Test",
		"description": "A test package.", "guide": "Supply the inputs.",
		"inputs":  p.inputs,
		"outputs": []any{map[string]any{"id": "report", "template": "report.md.liquid"}},
	}
	if p.constraints != "" {
		m["input_constraints"] = p.constraints
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(m); err != nil {
		t.Fatal(err)
	}
	doc, fs := jsonx.Parse(b.Bytes(), jsonx.Source{
		Input: finding.Subject{Kind: finding.SubjectInput, Name: "manifest"},
		At:    finding.Location{Origin: finding.OriginPackage, File: "metais.json"},
	}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("manifest: %s", describe(fs))
	}
	man, err := manifest.Build(doc.Value)
	if err != nil {
		t.Fatalf("manifest: %v", err)
	}
	files := map[string][]byte{}
	for k, v := range p.schemas {
		files[k] = []byte(v)
	}
	tree, err := pkgtreetest.New(files)
	if err != nil {
		t.Fatal(err)
	}
	lim := limits.Defaults()
	if p.lim != nil {
		lim = *p.lim
	}
	c, fs := schema.NewCompiler(tree, lim, decodeSchema)
	if len(fs) > 0 {
		t.Fatalf("compile: %s", describe(fs))
	}
	var decl schema.Declarations
	for _, in := range man.Inputs {
		si := schema.Input{Name: in.Name, Title: in.Title, Schema: in.Schema, Required: in.Required}
		if in.Description != "" {
			d := in.Description
			si.Description = &d
		}
		decl.Inputs = append(decl.Inputs, si)
	}
	decl.Constraints = man.InputConstraints
	var sf []schema.File
	for _, entry := range slices.Sorted(func(yield func(string) bool) {
		for k := range p.schemas {
			if !yield(k) {
				return
			}
		}
	}) {
		v, fs := decodeSchema(entry, files[entry])
		if len(fs) > 0 {
			t.Fatalf("%s: %s", entry, describe(fs))
		}
		sf = append(sf, schema.File{Path: entry, Root: v})
	}
	combined, err := schema.Bundle(decl, sf)
	if err != nil {
		t.Fatal(err)
	}
	s, fs := c.Combined(combined)
	if len(fs) > 0 {
		t.Fatalf("combined: %s", describe(fs))
	}
	return Package{Manifest: man, Compiler: c, Combined: combined, Schema: s}
}

// layer is the inputs layer of p.
func layer(t testing.TB, p pkg) *Layer {
	t.Helper()
	l, err := New(open(t, p))
	if err != nil {
		t.Fatal(err)
	}
	return l
}

// dataDir writes files, by slash-separated path, into a fresh directory
// and returns it. A path ending in / is an empty directory.
func dataDir(t testing.TB, files map[string]string) string {
	t.Helper()
	dir := testenv.Dir(t)
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if strings.HasSuffix(name, "/") {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// args is one command's supply flags: --data and every other flag, in
// command-line order, written "--set=<arg>", "--set-json=<arg>" and
// "--inputs-json=<text>".
type args struct {
	data  string
	flags []string
}

// run supplies a to l under lim and validates.
func run(t testing.TB, l *Layer, lim limits.Set, a args) *Result {
	t.Helper()
	s := l.NewSupply(lim)
	if a.data != "" {
		if err := s.ReadDataDir(a.data); err != nil {
			t.Fatalf("ReadDataDir: %v", err)
		}
	}
	for _, f := range a.flags {
		switch flag, arg, _ := strings.Cut(f, "="); flag {
		case "--set":
			s.AddSet(arg)
		case "--set-json":
			s.AddSetJSON(arg)
		case "--inputs-json":
			s.AddInputsJSON([]byte(arg))
		default:
			t.Fatalf("no flag %q", flag)
		}
	}
	res, err := s.Validate()
	if err != nil {
		t.Fatalf("Validate: %v", err)
	}
	return res
}

// describe writes each finding as one line: its code, subject, origin,
// file or argument, line and column, pointer and params, sorted, so a test
// compares what a finding is made of whatever order it came in.
func describe(fs []finding.Finding) string {
	var lines []string
	for _, f := range fs {
		lines = append(lines, line(f))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

func line(f finding.Finding) string {
	var b strings.Builder
	b.WriteString(string(f.Code()))
	if n := f.Subject().Name; n != "" {
		b.WriteString(" " + n)
	}
	if l, ok := f.Location(); ok {
		fmt.Fprintf(&b, " %s", l.Origin)
		if l.File != "" {
			b.WriteString(":" + l.File)
		}
		if l.Argument > 0 {
			fmt.Fprintf(&b, ":%d", l.Argument)
		}
		if l.Line > 0 {
			fmt.Fprintf(&b, ":%d:%d", l.Line, l.Column)
		}
		if l.HasPointer {
			b.WriteString(" #" + l.Pointer)
		}
	}
	js := finding.AppendJSON(nil, f)
	var obj map[string]json.RawMessage
	var arr []map[string]json.RawMessage
	if json.Unmarshal(js, &arr) == nil && len(arr) == 1 {
		obj = arr[0]
	}
	if p, ok := obj["params"]; ok && string(p) != "{}" {
		b.WriteString(" " + string(p))
	}
	for _, v := range f.Via() {
		fmt.Fprintf(&b, " via:%s", v.Keyword)
	}
	return b.String()
}

// want fails t unless fs, described, are exactly the lines given.
func wantFindings(t testing.TB, fs []finding.Finding, lines ...string) {
	t.Helper()
	slices.Sort(lines)
	if got := describe(fs); got != strings.Join(lines, "\n") {
		t.Errorf("findings:\n%s\nwant:\n%s", got, strings.Join(lines, "\n"))
	}
}

// codes is the codes of fs, sorted.
func codes(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, string(f.Code()))
	}
	slices.Sort(out)
	return out
}

// canonical is v's canonical JSON text, or "absent".
func canonical(v value.Value) string {
	if v == nil {
		return "absent"
	}
	return string(jsonx.Canonical(v))
}

// record is the record of the input name in res.
func record(t testing.TB, res *Result, name string) Record {
	t.Helper()
	for _, r := range res.Inputs {
		if r.Name == name {
			return r
		}
	}
	t.Fatalf("no record of %s", name)
	return Record{}
}
