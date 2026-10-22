// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// dirPkg declares an object input, vendor, and a collection, articles,
// of JSON, Markdown and CSV items.
var dirPkg = pkg{
	inputs: []map[string]any{
		slot("vendor", "inputs/any.json", "required", false),
		slot("articles", "inputs/list.json", "collection", true, "forms", []any{"json", "markdown", "csv"}, "required", false),
	},
	schemas: map[string]string{
		"inputs/any.json":  `{}`,
		"inputs/list.json": `{"type": "array", "items": {}}`,
	},
}

// lowered is the default limits with each NAME=VALUE lowered.
func lowered(t testing.TB, args ...string) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	for _, a := range args {
		if f, ok := lim.Lower(a); !ok {
			t.Fatalf("--limit %s: %s", a, f.Message())
		}
	}
	return lim
}

// TestCollectionItems: a collection is the directory's regular files, one
// item each, ordered by the UTF-8 bytes of their NFC names, its name in
// NFC and its stem the name without its last extension, its form from its
// last extension, and one collection may mix forms (ADR-003 D15 items 3 to
// 6). An entry whose name begins with . is ignored, file or directory.
func TestCollectionItems(t *testing.T) {
	testenv.Home(t)
	dir := dataDir(t, map[string]string{
		"articles/z.json":         `"z"`,
		"articles/B.json":         `"B"`,
		"articles/a.md":           "---\nt: 1\n---\nA\n",
		"articles/é.json":        `"e-acute"`, // NFD on the host
		"articles/Z2.csv":         "c\n1\n",
		"articles/v1.2.json":      `"v"`,
		"articles/.DS_Store":      "\x00",
		"articles/.hidden/x.json": `1`,
		"articles/.gitkeep":       "",
		".hidden.json":            `{}`,
		".git/config":             "",
	})
	res := run(t, layer(t, dirPkg), limits.Defaults(), args{data: dir})
	if len(res.Findings) > 0 {
		t.Fatalf("findings:\n%s", describe(res.Findings))
	}
	r := record(t, res, "articles")
	if v := canonical(r.Value); v != `["B",[{"c":"1"}],{"body":"A","t":1},"v","z","e-acute"]` {
		t.Errorf("value %s", v)
	}
	var got []string
	for _, it := range r.Items {
		got = append(got, it.Name+"|"+it.Stem+"|"+string(it.Form))
	}
	want := []string{"B.json|B|json", "Z2.csv|Z2|csv", "a.md|a|markdown", "v1.2.json|v1.2|json", "z.json|z|json", "é.json|é|json"}
	if !slices.Equal(got, want) {
		t.Errorf("items %q, want %q", got, want)
	}
	if r.Source.Kind != SourceFile || r.Source.File != "articles" || r.HasRaw {
		t.Errorf("source %+v, raw %v: a collection is its directory and has no raw form of its own", r.Source, r.HasRaw)
	}
	for _, it := range r.Items {
		if !it.HasRaw || !it.FromFile {
			t.Errorf("%s: no raw form", it.Name)
		}
	}
}

// TestCollectionRefusals: a subdirectory, a file with no extension or one
// the input's forms do not accept, two items with equal stems or stems
// that differ only in letter case, and a collection supplied as a file,
// are each a finding (ADR-003 D15 items 2, 3 and 5). The items that are
// refused are left out; the rest are validated in part.
func TestCollectionRefusals(t *testing.T) {
	testenv.Home(t)
	l := layer(t, dirPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"articles/sub/x.json": `1`,
		"articles/README":     "x",
		"articles/notes.txt":  "x",
		"articles/a.json":     `1`,
		"articles/a.md":       "a",
		"articles/Intro.md":   "i",
		"articles/intro.json": `2`,
		"articles/ok.json":    `3`,
	})})
	wantFindings(t, res.Findings,
		`collection.subdirectory articles data_dir:articles/sub`,
		`input.extension_not_accepted articles data_dir:articles/README {"accepted":[".json",".csv",".md",".markdown"]}`,
		`input.extension_not_accepted articles data_dir:articles/notes.txt {"accepted":[".json",".csv",".md",".markdown"]}`,
		`collection.stem_duplicate articles data_dir:articles/a.md {"case":false}`,
		`collection.stem_duplicate articles data_dir:articles/intro.json {"case":true}`,
		`input.validation.partial articles data_dir:articles # {"unparsed":5,"suppressed":0}`,
	)
	if v := canonical(record(t, res, "articles").Value); v != `[{"body":"i"},1,3]` {
		t.Errorf("value %s", v)
	}

	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"articles.json": `[]`})})
	wantFindings(t, res.Findings, `collection.as_file articles data_dir:articles.json`)
	// D15 item 2: the collection's directory is <name>/, so a directory
	// whose name has an extension is not it, and is not listed.
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"articles.d/a.json": `1`})})
	wantFindings(t, res.Findings, `collection.directory_name articles data_dir:articles.d`)
}

// TestCollectionEmptyAndMissing: "An empty directory is an empty array,
// present; a missing directory is an absent input" (ADR-003 D15 item 7).
func TestCollectionEmptyAndMissing(t *testing.T) {
	testenv.Home(t)
	l := layer(t, dirPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"articles/": "", "articles/.keep": ""})})
	if r := record(t, res, "articles"); canonical(r.Value) != "[]" || len(res.Findings) > 0 {
		t.Errorf("empty: %s\n%s", canonical(r.Value), describe(res.Findings))
	}
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"vendor.json": `{}`})})
	if r := record(t, res, "articles"); r.Value != nil || r.Source.Kind != SourceNone || r.HasDigest {
		t.Errorf("missing: %+v", r)
	}
}

// TestDataDirEntries: an entry's input name is the text before its first
// "."; one that names no declared input is a finding, as is one whose
// extension maps to no accepted form, such as .schema.json; two entries
// for one input are a finding and neither is read; a directory for an
// input that is not a collection is a finding (ADR-003 D16 items 2 to 5).
// Names beginning with "." are ignored.
func TestDataDirEntries(t *testing.T) {
	testenv.Home(t)
	l := layer(t, dirPkg)
	res := run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"vendr.json":         `{}`,
		"Vendor.json":        `{}`,
		"vendor.schema.json": `{}`,
		"README":             "",
		".env":               "SECRET=1",
	})})
	wantFindings(t, res.Findings,
		`input.file_unknown vendr data_dir:vendr.json`,
		`input.file_unknown Vendor data_dir:Vendor.json`,
		`input.file_unknown README data_dir:README`,
		`input.extension_not_accepted vendor data_dir:vendor.schema.json {"accepted":[".json",".yaml",".yml"]}`,
	)
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{
		"vendor.json":     `{}`,
		"vendor.yaml":     `{}`,
		"articles.json":   `[]`,
		"articles/a.json": `1`,
	})})
	wantFindings(t, res.Findings,
		`input.file_duplicate vendor data_dir:vendor.yaml`,
		`input.file_duplicate articles data_dir:articles.json`,
	)
	for _, name := range []string{"vendor", "articles"} {
		if r := record(t, res, name); !r.Unparsed || r.Value != nil {
			t.Errorf("%s: two entries for one input gave %s", name, canonical(r.Value))
		}
	}
	res = run(t, l, limits.Defaults(), args{data: dataDir(t, map[string]string{"vendor/a.json": `{}`})})
	wantFindings(t, res.Findings, `input.not_regular_file vendor data_dir:vendor {"kind":"directory"}`)
}

// TestDataDirLinksNotFollowed: "Symbolic links and special files in the
// data directory are findings and are never followed or read" (ADR-003
// D16 item 6, ADR-009 D7 item 2), even a link whose target stays inside;
// the data directory itself may be a link the caller named (ADR-009 D7
// item 4).
func TestDataDirLinksNotFollowed(t *testing.T) {
	testenv.Home(t)
	outside := filepath.Join(testenv.Dir(t), "secret.json")
	if err := os.WriteFile(outside, []byte(`{"secret": true}`), 0o644); err != nil {
		t.Fatal(err)
	}
	dir := dataDir(t, map[string]string{"inside.json": `{}`, "articles/a.json": `1`})
	for link, target := range map[string]string{
		"vendor.json":     outside,
		"articles/b.json": filepath.Join(dir, "inside.json"),
	} {
		if err := os.Symlink(target, filepath.Join(dir, filepath.FromSlash(link))); err != nil {
			t.Skipf("no symbolic links here: %v", err)
		}
	}
	via := filepath.Join(testenv.Dir(t), "data")
	if err := os.Symlink(dir, via); err != nil {
		t.Fatal(err)
	}
	res := run(t, layer(t, dirPkg), limits.Defaults(), args{data: via})
	wantFindings(t, res.Findings,
		`input.file_unknown inside data_dir:inside.json`,
		`input.link vendor data_dir:vendor.json`,
		`input.link articles data_dir:articles/b.json`,
		`input.validation.partial articles data_dir:articles # {"unparsed":1,"suppressed":0}`,
	)
	if r := record(t, res, "vendor"); r.Value != nil {
		t.Errorf("a link was followed: %s", canonical(r.Value))
	}
}

// TestReadDataDirRefusal: the data directory itself is the caller's
// argument; when it cannot be read, ReadDataDir says why, naming no path,
// and the supply is unchanged.
func TestReadDataDirRefusal(t *testing.T) {
	testenv.Home(t)
	l := layer(t, dirPkg)
	dir := testenv.Dir(t)
	file := filepath.Join(dir, "file")
	if err := os.WriteFile(file, nil, 0o644); err != nil {
		t.Fatal(err)
	}
	for path, reason := range map[string]string{filepath.Join(dir, "absent"): "not_found", file: "not_regular_file"} {
		s := l.NewSupply(limits.Defaults())
		err := s.ReadDataDir(path)
		var de *DataDirError
		if !errors.As(err, &de) || de.Reason != reason || !errors.Is(err, ErrDataDir) || strings.Contains(err.Error(), dir) {
			t.Errorf("%s: %v", path, err)
		}
		res, verr := s.Validate()
		if verr != nil || len(res.Findings) > 0 {
			t.Errorf("after a refused directory: %v %s", verr, describe(res.Findings))
		}
	}
	// A data directory holding more entries than collection-items.
	s := l.NewSupply(lowered(t, "collection-items=2"))
	big := dataDir(t, map[string]string{"a": "", "b": "", "c": ""})
	var de *DataDirError
	if err := s.ReadDataDir(big); !errors.As(err, &de) || de.Reason != "too_large" {
		t.Errorf("too many entries: %v", err)
	}
}

// TestDataLimits: input-size bounds one file and inputs-total every byte a
// command reads; collection-items bounds a collection's directory
// (ADR-009 D3). Each is limit.exceeded about the input (ADR-009 D19).
func TestDataLimits(t *testing.T) {
	testenv.Home(t)
	l := layer(t, dirPkg)
	files := map[string]string{"vendor.json": `{"a": "0123456789"}`, "articles/a.json": `1`, "articles/b.json": `2`}
	res := run(t, l, lowered(t, "input-size=10"), args{data: dataDir(t, files)})
	wantFindings(t, res.Findings, `limit.exceeded vendor data_dir:vendor.json {"limit":"input-size","bound":10,"lowered":true,"observed":19}`)
	res = run(t, l, lowered(t, "inputs-total=20"), args{data: dataDir(t, files)})
	wantFindings(t, res.Findings,
		`limit.exceeded articles data_dir:articles/b.json {"limit":"inputs-total","bound":20,"lowered":true}`,
		`input.validation.partial articles data_dir:articles # {"unparsed":1,"suppressed":0}`,
	)
	delete(files, "vendor.json")
	res = run(t, l, lowered(t, "collection-items=1"), args{data: dataDir(t, files)})
	wantFindings(t, res.Findings, `limit.exceeded articles data_dir:articles {"limit":"collection-items","bound":1,"lowered":true}`)
}
