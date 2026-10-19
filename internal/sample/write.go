// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/yamlx"
)

// File is one file, or one directory, of a data directory (ADR-003 D16).
type File struct {
	// Path is the path in the data directory, slash-separated. A
	// directory's ends in "/" and has no Data.
	Path string
	Data []byte
}

// Extension is the file extension ADR-003 assigns each form (D15 item 4).
func Extension(f Form) string {
	switch f {
	case FormJSON:
		return ".json"
	case FormYAML:
		return ".yaml"
	case FormCSV:
		return ".csv"
	case FormText:
		return ".txt"
	case FormMarkdown:
		return ".md"
	}
	return ""
}

// UnwritableError is a value that the input's form cannot write so that
// its reader reads it back as the same value.
type UnwritableError struct {
	// Input is the input's name, and Pointer the value's place in it.
	Input, Pointer string
	// Reason says why.
	Reason string
}

func (e *UnwritableError) Error() string {
	return fmt.Sprintf("sample: the input %s cannot be written at %q: %s", e.Input, e.Pointer, e.Reason)
}

// Files returns the files that hold v, the value of in, as a data
// directory holds them (ADR-007 D9 item 21), in the input's first form
// and with the extension ADR-003 assigns it:
//
//   - JSON: two-space indentation, LF, a final newline;
//   - YAML: the `yaml` filter's block style, LF, a final newline;
//   - CSV: a header record of the record schema's properties in
//     declaration order, then the names no property declares; the input's
//     delimiter; RFC 4180 quoting where a field holds the delimiter, a
//     quote, CR or LF, and the empty string quoted as ""; CRLF; numbers
//     as plain decimals, booleans true and false, an absent value an empty
//     field;
//   - text: the string and a final newline;
//   - Markdown: YAML frontmatter of every member but body, then the body
//     and a final newline;
//   - a collection: a directory named after the input holding one file
//     per item, named 001, 002, … zero-padded to at least three digits
//     and to the width of the count, so that file order is item order.
//
// Members are written in canonical order by the JSON and YAML writers, as
// every serializer of the data model writes them (ADR-004 D4 item 5). A
// text or Markdown file ends in one LF whatever its string ends in, since
// its reader removes one (ADR-003 D12). What Files writes, the form's
// reader reads back as v; a value it cannot write so is refused with an
// *UnwritableError: a CR where line breaks are normalized, an array,
// object or null in a CSV field, a value of the wrong kind for the form,
// and an integral number of magnitude above 2^53−1 in a CSV field, which
// has no text an integer column reads back as it. The JSON and YAML
// writers spell that number with an exponent, the shortest that
// round-trips, which reads back; ADR-003 D8 item 5 refuses the integer
// literal the json filter writes for it.
func (s *Generator) Files(in Input, v value.Value) ([]File, error) {
	s.input = in
	form, ext := in.form(), Extension(in.form())
	if !in.Collection {
		data, err := s.encode(in, form, v, "", node{in.Schema, ""})
		if err != nil {
			return nil, err
		}
		return []File{{Path: in.Name + ext, Data: data}}, nil
	}
	items, ok := v.(value.Array)
	if !ok {
		return nil, &UnwritableError{in.Name, "", "a collection's value is not an array"}
	}
	width := max(3, len(strconv.Itoa(items.Len())))
	files := []File{{Path: in.Name + "/"}}
	for i, item := range items.All() {
		at := "/" + itoa(i)
		data, err := s.encode(in, form, item, at, s.itemSchema(node{in.Schema, ""}, i))
		if err != nil {
			return nil, err
		}
		files = append(files, File{Path: fmt.Sprintf("%s/%0*d%s", in.Name, width, i+1, ext), Data: data})
	}
	return files, nil
}

// itemSchema is the schema the static walk (ADR-003 D14 item 1) gives the
// element at i of the array whose schema is n: through $ref targets, its
// prefixItems entry for i, or its items.
func (s *Generator) itemSchema(n node, i int) node {
	for x, ok := n, true; ok; x, ok = s.ref(x) {
		v, _ := s.c.Lookup(x.entry, x.ptr)
		o, isObj := v.(value.Object)
		if !isObj {
			break
		}
		if p, ok := o.Get("prefixItems"); ok {
			if a, ok := p.(value.Array); ok && i < a.Len() {
				return x.child("prefixItems", itoa(i))
			}
		}
		if _, ok := o.Get("items"); ok {
			return x.child("items")
		}
	}
	return node{}
}

// memberSchema is the schema the static walk gives the member name of the
// object whose schema is n, or the zero node when it gives none.
func (s *Generator) memberSchema(n node, name string) node {
	for x, ok := n, true; ok; x, ok = s.ref(x) {
		v, _ := s.c.Lookup(x.entry, x.ptr)
		o, isObj := v.(value.Object)
		if !isObj {
			break
		}
		if ps, ok := o.Get("properties"); ok {
			if po, ok := ps.(value.Object); ok {
				if _, ok := po.Get(name); ok {
					return x.child("properties", name)
				}
			}
		}
	}
	return node{}
}

// ref is the $ref target of n.
func (s *Generator) ref(n node) (node, bool) {
	if n.entry == "" {
		return node{}, false
	}
	e, p, ok := s.c.Ref(n.entry, n.ptr)
	return node{e, p}, ok
}

// declaredTypes is the declared type set at n (ADR-003 D14 item 1): the
// type of the first schema that declares one in the chain from n through
// $ref targets.
func (s *Generator) declaredTypes(n node) schema.TypeSet {
	for x, ok := n, n.entry != ""; ok; x, ok = s.ref(x) {
		v, _ := s.c.Lookup(x.entry, x.ptr)
		o, isObj := v.(value.Object)
		if !isObj {
			break
		}
		if t, ok := o.Get("type"); ok {
			var set schema.TypeSet
			for _, name := range typeNames(t) {
				set |= typeBits[name]
			}
			return set
		}
	}
	return 0
}

var typeBits = map[string]schema.TypeSet{
	"null": schema.TypeNull, "boolean": schema.TypeBoolean, "object": schema.TypeObject, "array": schema.TypeArray,
	"number": schema.TypeNumber, "string": schema.TypeString, "integer": schema.TypeInteger,
}

// encode writes v, the value of one file of in at ptr whose schema the
// static walk gives as n, in form.
func (s *Generator) encode(in Input, form Form, v value.Value, ptr string, n node) ([]byte, error) {
	bad := func(at, reason string) error { return &UnwritableError{in.Name, ptr + at, reason} }
	switch form {
	case FormJSON:
		// A supplied value can hold an integral number beyond 2^53−1 (item
		// 2). ADR-003 D8 item 5 refuses an integer literal by its spelling,
		// so 11e17 is read while 1100000000000000000 is refused: the
		// writer spells it with an exponent.
		b, err := jsonx.EncodeInput(v, 2)
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	case FormYAML:
		return append(yamlx.Write(v), '\n'), nil
	case FormText:
		str, ok := v.(value.String)
		if !ok {
			return nil, bad("", "a text file's value is not a string")
		}
		if strings.ContainsRune(str.Text(), '\r') {
			return nil, bad("", "a text file holds no CR")
		}
		return []byte(str.Text() + "\n"), nil
	case FormMarkdown:
		return s.markdown(v, bad)
	case FormCSV:
		return s.table(in, v, n, bad)
	}
	return nil, bad("", "no form "+string(form))
}

// markdown writes a Markdown document: the frontmatter of every member
// but body, when there is one, and the body (ADR-003 D13). A body whose
// first line is --- after no frontmatter is written after an empty one,
// so that it is not read as one.
func (s *Generator) markdown(v value.Value, bad func(string, string) error) ([]byte, error) {
	o, ok := v.(value.Object)
	if !ok {
		return nil, bad("", "a Markdown document's value is not an object")
	}
	bv, ok := o.Get("body")
	if !ok {
		return nil, bad("", "a Markdown document has a body")
	}
	body, ok := bv.(value.String)
	if !ok {
		return nil, bad("/body", "a Markdown document's body is a string")
	}
	if strings.ContainsRune(body.Text(), '\r') {
		return nil, bad("/body", "a Markdown body holds no CR")
	}
	var front []value.Member
	for name, m := range o.All() {
		if name != "body" {
			front = append(front, value.Member{Name: name, Value: m})
		}
	}
	var b bytes.Buffer
	switch {
	case len(front) > 0:
		fo, err := value.NewObject(front...)
		if err != nil {
			panic(err)
		}
		b.WriteString("---\n")
		b.Write(yamlx.Write(fo))
		b.WriteString("\n---\n")
	case body.Text() == "---" || strings.HasPrefix(body.Text(), "---\n"):
		b.WriteString("---\n---\n")
	}
	b.WriteString(body.Text())
	b.WriteByte('\n')
	return b.Bytes(), nil
}

// table writes a CSV file: v, an array of records, whose schema the
// static walk gives as n.
func (s *Generator) table(in Input, v value.Value, n node, bad func(string, string) error) ([]byte, error) {
	rows, ok := v.(value.Array)
	if !ok {
		return nil, bad("", "a CSV file's value is not an array")
	}
	record := s.itemSchema(n, 0)
	var header []string
	seen := map[string]bool{}
	if record.entry != "" {
		for _, name := range s.declared(s.plain([]node{record})) {
			if !seen[name] {
				seen[name] = true
				header = append(header, name)
			}
		}
	}
	for i, r := range rows.All() {
		o, ok := r.(value.Object)
		if !ok {
			return nil, bad("/"+itoa(i), "a CSV record's value is not an object")
		}
		for name := range o.All() {
			if !seen[name] {
				seen[name] = true
				header = append(header, name)
			}
		}
	}
	if len(header) == 0 {
		return nil, bad("", "a CSV file has a header record of at least one name")
	}
	records := [][]csv.Cell{make([]csv.Cell, len(header))}
	for j, name := range header {
		if name == "" {
			return nil, bad("", "a CSV header names every column")
		}
		records[0][j] = csv.Cell{Text: name}
	}
	for i, r := range rows.All() {
		o := r.(value.Object)
		rec := make([]csv.Cell, len(header))
		for j, name := range header {
			m, ok := o.Get(name)
			if !ok {
				if len(header) == 1 {
					// The record would be an empty line, which is no record
					// (ADR-003 D11 item 1).
					return nil, bad("/"+itoa(i), "a one-column CSV record holds its value")
				}
				continue
			}
			at := "/" + itoa(i) + "/" + escape(name)
			cell, err := field(m, s.declaredTypes(s.memberSchema(record, name)))
			if err != "" {
				return nil, bad(at, err)
			}
			rec[j] = cell
		}
		records = append(records, rec)
	}
	delim := rune(in.Delimiter)
	if delim == 0 {
		delim = ','
	}
	b, err := csv.WriteCells(records, delim)
	if err != nil {
		return nil, err
	}
	return bytes.ReplaceAll(b, []byte("\n"), []byte("\r\n")), nil
}

// field is the CSV field that holds m, a member of a record whose
// column's declared type set is t, or why there is none: one that the
// reader's coercion (ADR-003 D14 item 2) would read as another value.
func field(m value.Value, t schema.TypeSet) (csv.Cell, string) {
	var c csv.Cell
	switch m := m.(type) {
	case value.String:
		if strings.ContainsRune(m.Text(), '\r') {
			return c, "a CSV field holds no CR"
		}
		c = csv.Cell{Text: m.Text(), Quoted: m.Text() == ""}
	case value.Number:
		f := m.Float64()
		if m.Integral() && math.Abs(f) > value.MaxSafeInteger {
			return c, "an integer beyond 2^53−1 has no CSV field the reader accepts"
		}
		c = csv.Cell{Text: strconv.FormatFloat(f, 'f', -1, 64)}
	case value.Bool:
		c = csv.Cell{Text: strconv.FormatBool(bool(m))}
	default:
		return c, "a CSV field holds no " + m.Kind().String()
	}
	if c.Text == "" {
		return c, ""
	}
	back, err := schema.Coerce(c.Text, t)
	if err != nil || !value.Equal(back, m) {
		return c, "the column's declared type reads the field as another value"
	}
	return c, ""
}

// escape is name as a JSON Pointer token.
func escape(name string) string {
	return strings.ReplaceAll(strings.ReplaceAll(name, "~", "~0"), "/", "~1")
}

// plain is C over starts with no branch taken: the schemas a record's
// header is declared by.
func (s *Generator) plain(starts []node) *conj {
	c := &conj{base: starts}
	seen := map[node]bool{}
	for _, n := range starts {
		c.starts = append(c.starts, n)
		s.collect(c, n, seen)
	}
	return c
}
