// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// This file is the program of the hostile corpus of ADR-009 D17 item 6,
// schemas and patterns, and of item 12's pairs for the schema limits
// (W6-10). Those vectors' packages are source directories; the files of
// them too long or too regular to write by hand are built here, and
// TestSchemaCorpus is their reproduction test (ADR-011 D16 item 5).

// genFile is one generated file of a vector: its path below the vectors
// directory, and its bytes.
type genFile struct {
	path  string
	build func() []byte
}

// schemaCorpusFiles is every generated file of the schema corpus.
func schemaCorpusFiles() []genFile {
	return []genFile{
		{"sec/ref-depth/package/inputs/report.schema.json", func() []byte { return refChain(32) }},
		{"sec/ref-depth/beyond/inputs/report.schema.json", func() []byte { return refChain(33) }},
		{"sec/schema-work/package/inputs/report.schema.json", func() []byte { return anyOfChain(4) }},
		{"sec/schema-work/exponential/inputs/report.schema.json", func() []byte { return anyOfChain(30) }},
		{"sec/pattern-length/package/inputs/report.schema.json", func() []byte { return classPattern(4096) }},
		{"sec/pattern-length/beyond/inputs/report.schema.json", func() []byte { return classPattern(4097) }},
		{"sec/pattern-backtracking/data/report.json", func() []byte { return runOfA(64 << 10) }},
		{"dat/unique-items-million/data/report.json", millionItems},
	}
}

// TestSchemaCorpus builds every generated file of the schema corpus and
// compares it with the committed file, or, with -update, writes it:
//
//	go test ./internal/vectorgen -run TestSchemaCorpus -args -update
func TestSchemaCorpus(t *testing.T) {
	for _, f := range schemaCorpusFiles() {
		got := f.build()
		path := filepath.Join(vectorsDir, filepath.FromSlash(f.path))
		if *update {
			if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, got, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		want, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("%s: %v (run go test ./internal/vectorgen -run TestSchemaCorpus -args -update)", f.path, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: the committed file is not what its program builds", f.path)
		}
	}
}

// schemaJSON is v as a schema file: JSON, members in byte order, and a
// final newline.
func schemaJSON(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return append(b, '\n')
}

// refChain is a schema whose root refers to /$defs/s0, each definition but
// the last referring to the next: a chain of n references at one instance
// location, ending in an empty schema (ADR-009 D3's ref-depth).
func refChain(n int) []byte {
	defs := map[string]any{}
	for i := range n - 1 {
		defs["s"+strconv.Itoa(i)] = map[string]any{"$ref": "#/$defs/s" + strconv.Itoa(i+1)}
	}
	defs["s"+strconv.Itoa(n-1)] = map[string]any{}
	return schemaJSON(map[string]any{"$ref": "#/$defs/s0", "$defs": defs})
}

// anyOfChain is n levels of anyOf, each of two references to the next
// level, over an integer: a validation of 2^n paths whose every reference
// consumes nothing of the instance (ADR-009 D17 item 6). Its longest chain
// is n+1 references, within ref-depth for n up to 31.
func anyOfChain(n int) []byte {
	defs := map[string]any{}
	for i := range n {
		next := map[string]any{"$ref": "#/$defs/s" + strconv.Itoa(i+1)}
		defs["s"+strconv.Itoa(i)] = map[string]any{"anyOf": []any{next, next}}
	}
	defs["s"+strconv.Itoa(n)] = map[string]any{"type": "integer"}
	return schemaJSON(map[string]any{"$ref": "#/$defs/s0", "$defs": defs})
}

// classPattern is a schema whose member p has a pattern of n bytes: one
// character class of the letter a, so that its program is a few
// instructions whatever its length, and only pattern-length can refuse it.
func classPattern(n int) []byte {
	p := "[" + strings.Repeat("a", n-2) + "]"
	return schemaJSON(map[string]any{"type": "object", "properties": map[string]any{"p": map[string]any{"pattern": p}}})
}

// runOfA is a JSON string input of n bytes: a run of the letter a ending
// in an exclamation mark, the subject on which ^(a+)+$ backtracks
// exponentially in an engine that backtracks.
func runOfA(n int) []byte {
	return []byte(`"` + strings.Repeat("a", n-3) + `!"`)
}

// millionItems is a JSON array input of a million integers, all distinct
// but the last, which equals the first: uniqueItems must see every item to
// find the one duplicate.
func millionItems() []byte {
	var b bytes.Buffer
	b.WriteByte('[')
	for i := range 999_999 {
		b.WriteString(strconv.Itoa(i))
		b.WriteByte(',')
	}
	b.WriteString("0]\n")
	return b.Bytes()
}
