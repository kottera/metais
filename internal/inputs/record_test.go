// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"crypto/sha256"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// recordPkg declares vendor, an object with a defaulted member, and
// articles, a collection of Markdown items.
var recordPkg = pkg{
	inputs: []map[string]any{
		slot("vendor", "inputs/vendor.json", "required", false),
		slot("articles", "inputs/articles.json", "collection", true, "forms", []any{"markdown", "json"}, "required", false),
	},
	schemas: map[string]string{
		"inputs/vendor.json":   `{"type": "object", "properties": {"name": {"type": "string"}, "tier": {"type": "string", "default": "basic"}}}`,
		"inputs/articles.json": `{"type": "array", "items": {"type": "object"}}`,
	},
}

// TestRecordRaw: "An input has raw bytes when its whole value came from one
// data-directory file and no --inputs-json member or path set touched it.
// A collection has no raw form of its own; each item has one under the
// same rule, applied per item (--set /articles/0/title=… removes item 0's
// raw form and no other's)" (ADR-003 D21 item 2).
func TestRecordRaw(t *testing.T) {
	testenv.Home(t)
	l := layer(t, recordPkg)
	vendor := "\uFEFF{\"name\": \"Acme\"}\r\n"
	dir := dataDir(t, map[string]string{"vendor.json": vendor, "articles/a.md": "A\r\n", "articles/b.md": "B\n"})

	res := run(t, l, limits.Defaults(), args{data: dir})
	v, a := record(t, res, "vendor"), record(t, res, "articles")
	if !v.HasRaw || string(v.Raw) != vendor {
		t.Errorf("vendor raw %q", v.Raw)
	}
	if a.HasRaw || len(a.Items) != 2 || string(a.Items[0].Raw) != "A\r\n" || string(a.Items[1].Raw) != "B\n" {
		t.Errorf("articles raw %v, items %+v", a.HasRaw, a.Items)
	}

	res = run(t, l, limits.Defaults(), args{data: dir, flags: []string{"--set=/vendor/name=X", "--set=/articles/0/title=T"}})
	v, a = record(t, res, "vendor"), record(t, res, "articles")
	if v.HasRaw || a.Items[0].HasRaw || !a.Items[1].HasRaw {
		t.Errorf("after the sets: vendor raw %v, items %v %v", v.HasRaw, a.Items[0].HasRaw, a.Items[1].HasRaw)
	}
	if !a.Items[0].FromFile || a.Items[0].Name != "a.md" {
		t.Errorf("a touched item is still its file: %+v", a.Items[0])
	}

	res = run(t, l, limits.Defaults(), args{data: dir, flags: []string{`--inputs-json={"articles": [{"x": 1}]}`, "--set-json=/articles/-={}"}})
	a = record(t, res, "articles")
	if len(a.Items) != 2 || a.Items[0].FromFile || a.Items[0].HasRaw || a.Items[1].FromFile || a.Source.Kind != SourceCommandLine {
		t.Errorf("articles from the command line: %+v", a)
	}
}

// TestRecordSource: "for an input: file with its data-directory path, or
// command-line, or default; for each collection item: its name and stem
// (D15 item 5) and its form, or none for an item supplied on the command
// line" (ADR-003 D21 item 3); From lists every source in the order applied
// (ADR-008 D7 item 5).
func TestRecordSource(t *testing.T) {
	testenv.Home(t)
	l := layer(t, recordPkg)
	dir := dataDir(t, map[string]string{"vendor.yaml": "name: Acme\n", "articles/a.md": "A"})
	res := run(t, l, limits.Defaults(), args{data: dir, flags: []string{"--set=/vendor/name=B", "--set-json=/articles/-={}"}})
	v, a := record(t, res, "vendor"), record(t, res, "articles")
	if v.Source != (Source{Kind: SourceFile, File: "vendor.yaml"}) {
		t.Errorf("vendor source %+v", v.Source)
	}
	if want := []From{{Kind: FromFile, File: "vendor.yaml"}, {Kind: FromSet, Argument: 1}, {Kind: FromDefault}}; !slices.Equal(v.From, want) {
		t.Errorf("vendor from %+v, want %+v", v.From, want)
	}
	if a.Source != (Source{Kind: SourceFile, File: "articles"}) || len(a.Items) != 2 || !a.Items[0].FromFile || a.Items[1].FromFile {
		t.Errorf("articles %+v", a)
	}
	if want := []From{{Kind: FromFile, File: "articles"}, {Kind: FromSetJSON, Argument: 2}}; !slices.Equal(a.From, want) {
		t.Errorf("articles from %+v", a.From)
	}
	res = run(t, l, limits.Defaults(), args{flags: []string{"--set=/vendor/name=B"}})
	if v := record(t, res, "vendor"); v.Source.Kind != SourceCommandLine || v.HasRaw {
		t.Errorf("a set creates vendor from the command line: %+v", v)
	}
	if a := record(t, res, "articles"); a.Source.Kind != SourceNone || a.Value != nil || a.HasDigest || a.Items != nil {
		t.Errorf("an absent input: %+v", a)
	}
}

// TestRecordDigest: "SHA-256 over the canonical JSON serialization (RFC
// 8785, ADR-002 D9) of value, for the input and for each item; none when
// absent. Computed after defaults … an omitted field and the same field
// supplied with its default value render alike and digest alike"
// (ADR-003 D21 item 4); a value supplied by file and by --inputs-json
// digests alike.
func TestRecordDigest(t *testing.T) {
	testenv.Home(t)
	l := layer(t, recordPkg)
	byFile := record(t, run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"vendor.yaml": "name: Acme\n"})}), "vendor")
	byJSON := record(t, run(t, l, limits.Defaults(), args{flags: []string{`--inputs-json={"vendor": {"tier": "basic", "name": "Acme"}}`}}), "vendor")
	if byFile.Digest != byJSON.Digest || !byFile.HasDigest {
		t.Errorf("digests %x and %x", byFile.Digest, byJSON.Digest)
	}
	if want := sha256.Sum256([]byte(`{"name":"Acme","tier":"basic"}`)); byFile.Digest != want {
		t.Errorf("digest %x, want %x", byFile.Digest, want)
	}
	a := record(t, run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"articles/x.json": `{"b": 1, "a": 2}`})}), "articles")
	if want := sha256.Sum256([]byte(`{"a":2,"b":1}`)); len(a.Items) != 1 || a.Items[0].Digest != want {
		t.Errorf("item digest %+v", a.Items)
	}
	if want := sha256.Sum256([]byte(`[{"a":2,"b":1}]`)); a.Digest != want {
		t.Errorf("collection digest %x", a.Digest)
	}
}
