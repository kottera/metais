// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/pkgtree/pkgtreetest"
)

// schemaFile is the source a test's schema text is read as.
var schemaFile = jsonx.Source{
	Input: finding.Subject{Kind: finding.SubjectSchema, Name: "inputs/vendor.schema.json"},
	At:    finding.Location{Origin: finding.OriginPackage, File: "inputs/vendor.schema.json"},
}

// parse returns the value of the JSON text src, failing t on any finding.
func parse(t *testing.T, src string) value.Value {
	t.Helper()
	doc, fs := jsonx.Parse([]byte(src), schemaFile, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("parse %s: %v", src, fs)
	}
	return doc.Value
}

// canonical returns v's canonical JSON text.
func canonical(v value.Value) string { return string(jsonx.Canonical(v)) }

// ptr returns a pointer to s.
func ptr(s string) *string { return &s }

// parseJSON reads src into the data model, a stand-in for the package
// loader's reader in these tests only: it keeps each number's literal and
// refuses a repeated member name.
func parseJSON(src []byte) (value.Value, error) {
	dec := json.NewDecoder(bytes.NewReader(src))
	dec.UseNumber()
	v, err := readJSON(dec)
	if err != nil {
		return nil, err
	}
	if _, err := dec.Token(); err != io.EOF {
		return nil, fmt.Errorf("content after the value")
	}
	return v, nil
}

func readJSON(dec *json.Decoder) (value.Value, error) {
	tok, err := dec.Token()
	if err != nil {
		return nil, err
	}
	switch tok := tok.(type) {
	case nil:
		return value.Null{}, nil
	case bool:
		return value.Bool(tok), nil
	case json.Number:
		return value.ParseNumber(string(tok))
	case string:
		return value.NewString(tok)
	case json.Delim:
		switch tok {
		case '[':
			var elems []value.Value
			for dec.More() {
				e, err := readJSON(dec)
				if err != nil {
					return nil, err
				}
				elems = append(elems, e)
			}
			dec.Token()
			return value.NewArray(elems...)
		case '{':
			var members []value.Member
			for dec.More() {
				k, err := dec.Token()
				if err != nil {
					return nil, err
				}
				e, err := readJSON(dec)
				if err != nil {
					return nil, err
				}
				members = append(members, value.Member{Name: k.(string), Value: e})
			}
			dec.Token()
			return value.NewObject(members...)
		}
	}
	return nil, fmt.Errorf("unexpected token %v", tok)
}

// decodeJSON is a Decode over parseJSON. A file it cannot read is a
// defect in the test.
func decodeJSON(entry string, src []byte) (value.Value, []finding.Finding) {
	v, err := parseJSON(src)
	if err != nil {
		panic(fmt.Sprintf("%s: %v", entry, err))
	}
	return v, nil
}

// mustValue parses s.
func mustValue(t testing.TB, s string) value.Value {
	t.Helper()
	v, err := parseJSON([]byte(s))
	if err != nil {
		t.Fatalf("parse %s: %v", s, err)
	}
	return v
}

// compileFiles compiles the package holding files.
func compileFiles(t testing.TB, lim limits.Set, files map[string]string) (*Compiler, []finding.Finding) {
	t.Helper()
	m := map[string][]byte{}
	for k, v := range files {
		m[k] = []byte(v)
	}
	tree, err := pkgtreetest.New(m)
	if err != nil {
		t.Fatal(err)
	}
	return NewCompiler(tree, lim, decodeJSON)
}

// mustCompile compiles the package holding files, which must compile, and
// returns the schema of entry.
func mustCompile(t testing.TB, files map[string]string, entry string) *Schema {
	t.Helper()
	c, fs := compileFiles(t, limits.Defaults(), files)
	if len(fs) > 0 {
		t.Fatalf("compile: %s", describe(fs))
	}
	s, ok := c.Schema(entry)
	if !ok {
		t.Fatalf("no schema %s", entry)
	}
	return s
}

// describe writes findings as code@file#pointer lines, sorted.
func describe(fs []finding.Finding) string {
	var lines []string
	for _, f := range fs {
		l, _ := f.Location()
		lines = append(lines, fmt.Sprintf("%s@%s#%s", f.Code(), l.File, l.Pointer))
	}
	slices.Sort(lines)
	return strings.Join(lines, "\n")
}

// codes returns the distinct codes of fs, sorted.
func codes(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, string(f.Code()))
	}
	slices.Sort(out)
	return slices.Compact(out)
}
