// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"bytes"
	"fmt"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/inputs/csv"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/markdown"
	"metais.org/metais/internal/pkgtree/pkgtreetest"
	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/yamlx"
)

// entry is the schema file a test's one input names.
const entry = "inputs/data.schema.json"

// pkg is a package's schema files, compiled, with the position table of
// each, as the package loader reads them: by internal/jsonx.
type pkg struct {
	c   *schema.Compiler
	pos map[string]finding.Positions
}

// compile compiles files, a package's schema files by path, failing t on
// any finding.
func compile(t testing.TB, files map[string]string) *pkg {
	t.Helper()
	p, fs := compileErr(t, files)
	if len(fs) > 0 {
		t.Fatalf("the schemas do not compile: %s", describe(fs))
	}
	return p
}

func compileErr(t testing.TB, files map[string]string) (*pkg, []finding.Finding) {
	t.Helper()
	raw := map[string][]byte{}
	for k, v := range files {
		raw[k] = []byte(v)
	}
	tree, err := pkgtreetest.New(raw)
	if err != nil {
		t.Fatal(err)
	}
	p := &pkg{pos: map[string]finding.Positions{}}
	decode := func(e string, src []byte) (value.Value, []finding.Finding) {
		doc, fs := jsonx.Parse(src, jsonx.Source{
			Input: finding.Subject{Kind: finding.SubjectInput, Name: "schema"},
			At:    finding.Location{Origin: finding.OriginPackage, File: e},
		}, limits.Defaults())
		if len(fs) > 0 {
			return nil, fs
		}
		p.pos[e] = doc.Positions()
		return doc.Value, nil
	}
	c, fs := schema.NewCompiler(tree, limits.Defaults(), decode)
	p.c = c
	return p, fs
}

// gen is a generator over p.
func (p *pkg) gen(opts Options) *Generator {
	return New(p.c, func(e string) finding.Positions { return p.pos[e] }, limits.Defaults(), opts)
}

// one compiles schemaText as the one schema file and generates the
// required JSON input that names it.
func one(t testing.TB, schemaText string, opts Options) (value.Value, []finding.Finding) {
	t.Helper()
	testenv.Home(t)
	p := compile(t, map[string]string{entry: schemaText})
	v, present, fs, err := p.gen(opts).Generate(Input{Name: "data", Schema: entry, Forms: []Form{FormJSON}, Required: true})
	if err != nil {
		t.Fatal(err)
	}
	if !present {
		t.Fatal("a required input was omitted")
	}
	return v, fs
}

// mustOne is one, failing t on any finding, and checking the value
// validates against the schema.
func mustOne(t testing.TB, schemaText string, opts Options) value.Value {
	t.Helper()
	v, fs := one(t, schemaText, opts)
	if len(fs) > 0 {
		t.Fatalf("%s: %s", schemaText, describe(fs))
	}
	p := compile(t, map[string]string{entry: schemaText})
	s, _ := p.c.Schema(entry)
	if fails, err := s.Validate(v); err != nil || len(fails) > 0 {
		t.Fatalf("%s: the value %s does not validate: %v %v", schemaText, text(v), fails, err)
	}
	return v
}

// text is v's canonical JSON.
func text(v value.Value) string {
	if v == nil {
		return "<nil>"
	}
	return string(jsonx.Canonical(v))
}

// parse is the JSON text src's value.
func parse(t testing.TB, src string) value.Value {
	t.Helper()
	doc, fs := jsonx.Parse([]byte(src), jsonx.Source{Input: finding.Subject{Kind: finding.SubjectInput, Name: "data"}}, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("%s: %s", src, describe(fs))
	}
	return doc.Value
}

// describe lists findings, one a line.
func describe(fs []finding.Finding) string {
	var b strings.Builder
	for _, f := range fs {
		loc, _ := f.Location()
		fmt.Fprintf(&b, "\n  %s %s#%s %s", f.Code(), loc.File, loc.Pointer, f.Message())
	}
	return b.String()
}

// readBack reads files, which Files wrote for in, through Wave 2's reader
// of in's form, as the inputs layer reads a data directory (ADR-003 D8 to
// D16): line breaks normalized for csv, text and markdown; a CSV field
// coerced by the column's declared type set, found by the static walk of
// in's schema (D14); one final LF removed from a text file and a body.
func readBack(t testing.TB, p *pkg, in Input, files []File) value.Value {
	t.Helper()
	doc, _ := p.c.Lookup(in.Schema, "")
	walker, err := schema.NewWalker(doc, schema.RetrievalURI(in.Schema))
	if err != nil {
		t.Fatal(err)
	}
	read := func(f File, path []schema.Step) value.Value {
		subj := finding.Subject{Kind: finding.SubjectInput, Name: in.Name}
		at := finding.Location{Origin: finding.OriginDataDir, File: f.Path}
		var fs []finding.Finding
		var v value.Value
		switch in.form() {
		case FormJSON:
			var d jsonx.Doc
			d, fs = jsonx.Parse(f.Data, jsonx.Source{Input: subj, At: at}, limits.Defaults())
			v = d.Value
		case FormYAML:
			var d yamlx.Doc
			d, fs = yamlx.Parse(f.Data, yamlx.Source{Subject: subj, At: at, Size: limits.InputSize}, limits.Defaults())
			v = d.Value
		case FormText:
			v = mustString(strings.TrimSuffix(string(normalize(f.Data)), "\n"))
		case FormMarkdown:
			front, line, body, mfs := markdown.SplitFrontmatter(f.Data, markdown.Source{Input: subj, At: at}, limits.Defaults())
			fs = mfs
			members := []value.Member{{Name: "body", Value: mustString(strings.TrimSuffix(string(body), "\n"))}}
			if front != nil && len(fs) == 0 {
				d, yfs := yamlx.Parse(front, yamlx.Source{Subject: subj, At: at, Size: limits.FrontmatterSize, FirstLine: line}, limits.Defaults())
				fs = yfs
				if o, ok := d.Value.(value.Object); ok {
					for n, m := range o.All() {
						members = append(members, value.Member{Name: n, Value: m})
					}
				}
			}
			o, err := value.NewObject(members...)
			if err != nil {
				t.Fatal(err)
			}
			v = o
		case FormCSV:
			delim := in.Delimiter
			if delim == 0 {
				delim = csv.Comma
			}
			tab, cfs := csv.Read(normalize(f.Data), delim, csv.Source{Input: subj, At: at}, limits.Defaults())
			fs = cfs
			var rows []value.Value
			for i, rec := range tab.Records {
				var ms []value.Member
				for j, fl := range rec.Fields {
					name := tab.Header[j].Text
					st, ok, err := walker.Walk(append(slices.Clone(path), schema.Index(i), schema.Member(name)))
					if err != nil {
						t.Fatal(err)
					}
					var types schema.TypeSet
					if ok {
						types = st.Types
					}
					cv, present, err := schema.CoerceField(fl.Text, fl.Quoted, types)
					if err != nil {
						t.Fatal(err)
					}
					if present {
						ms = append(ms, value.Member{Name: name, Value: cv})
					}
				}
				o, err := value.NewObject(ms...)
				if err != nil {
					t.Fatal(err)
				}
				rows = append(rows, o)
			}
			a, _ := value.NewArray(rows...)
			v = a
		}
		if len(fs) > 0 {
			t.Fatalf("%s does not read back:%s\n%s", f.Path, describe(fs), f.Data)
		}
		return v
	}
	if !in.Collection {
		if len(files) != 1 || files[0].Path != in.Name+Extension(in.form()) {
			t.Fatalf("files %v for a single input", paths(files))
		}
		return read(files[0], nil)
	}
	if len(files) == 0 || files[0].Path != in.Name+"/" {
		t.Fatalf("files %v for a collection", paths(files))
	}
	var items []value.Value
	for i, f := range files[1:] {
		items = append(items, read(f, []schema.Step{schema.Index(i)}))
	}
	a, _ := value.NewArray(items...)
	return a
}

// normalize is ADR-003 D8 item 2: CRLF and CR read as LF.
func normalize(b []byte) []byte {
	b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
	return bytes.ReplaceAll(b, []byte("\r"), []byte("\n"))
}

func paths(files []File) []string {
	var out []string
	for _, f := range files {
		out = append(out, f.Path)
	}
	return out
}

// reasons is the reason of each finding, which is every one
// schema.sample_unsatisfiable, in order.
func reasons(t testing.TB, fs []finding.Finding) []string {
	t.Helper()
	var out []string
	for _, f := range fs {
		if f.Code() != CodeUnsatisfiable {
			t.Fatalf("finding %s", f.Code())
		}
		r, _ := f.Param("reason")
		str, _ := r.Str()
		out = append(out, str)
	}
	return out
}
