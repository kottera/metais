// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"flag"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

var update = flag.Bool("update", false, "write every committed package's bytes into its vector")

// vectorsDir is the suite's vectors, from this package's directory.
const vectorsDir = "../../spec/conformance/vectors"

// pkgCase is one committed package: the vector it is in, and how it is
// built.
type pkgCase struct {
	vector string // <area>/<slug>
	build  func(t testing.TB) []byte
}

// TestPackages is the program and its reproduction test (ADR-011 D16 item
// 5): every case's package is built and compared with the committed bytes,
// or, with -update, written; and every committed package is some case's.
func TestPackages(t *testing.T) {
	testenv.Home(t)
	cases := packageCases()
	seen := map[string]bool{}
	for _, c := range cases {
		if seen[c.vector] {
			t.Fatalf("%s has two cases", c.vector)
		}
		seen[c.vector] = true
		got := c.build(t)
		path := filepath.Join(vectorsDir, filepath.FromSlash(c.vector), "package.metais")
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
			t.Errorf("%s: %v (run go test ./internal/vectorgen -run TestPackages -args -update)", c.vector, err)
			continue
		}
		if !bytes.Equal(got, want) {
			t.Errorf("%s: the committed package.metais is not what its case builds", c.vector)
		}
	}
	err := filepath.WalkDir(vectorsDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "package.metais" {
			return err
		}
		rel, _ := filepath.Rel(vectorsDir, filepath.Dir(p))
		if !seen[filepath.ToSlash(rel)] {
			t.Errorf("%s: a committed package.metais that no case builds", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// The base package every hostile case starts from: a manifest, one schema
// and one template (ADR-001 Conformance's minimal valid package).
const baseManifest = `metais: "1.0"
name: hostile
version: 1.0.0
title: Hostile
description: A package that breaks one rule.
guide: Supply a report.
inputs:
  - name: report
    title: Report
    schema: inputs/report.schema.json
outputs:
  - id: report
    template: report.md.liquid
`

func baseFiles() map[string][]byte {
	return map[string][]byte{
		"metais.yaml":                []byte(baseManifest),
		"inputs/report.schema.json":  []byte(`{"type": "object", "required": ["title"], "properties": {"title": {"type": "string"}}}` + "\n"),
		"templates/report.md.liquid": []byte("# {{ inputs.report.title }}\n"),
	}
}

// packed is a package's entries as pack writes them: README.md, stored,
// first, then in byte order of their names files, whose compiled files
// loader derives, every entry of extra beside them, and the digest file of
// them all but README.md (ADR-001 A1). extra is how a case adds an entry
// whose name no source directory can hold.
func packed(t testing.TB, files map[string][]byte, extra ...zentry) []zentry {
	t.Helper()
	p, fs := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	if p == nil {
		t.Fatalf("the case's sources do not open: %s", codes(fs))
	}
	var out []zentry
	for n, b := range files {
		out = append(out, zentry{name: n, content: b})
	}
	for _, f := range p.Compiled {
		out = append(out, zentry{name: f.Name, content: f.Content})
	}
	out = append(out, zentry{name: zipfmt.ReadmeName, content: loader.Readme(p.Manifest.Metais)})
	out = append(out, extra...)
	return withDigests(out)
}

// withDigests returns entries with the digest file of every one of them
// but README.md added, sorted by name but for README.md, first. A name that is not UTF-8 cannot be a member name
// of the JSON digest file, so it is left out of it: the case it is in is
// refused for that name, at step 3, before the digest file is read; so is
// a second entry of one name, which the file lists once.
func withDigests(entries []zentry) []zentry {
	var list []digest.Entry
	listed := map[string]bool{}
	for _, e := range entries {
		if e.name != digest.File && e.name != zipfmt.ReadmeName && utf8.ValidString(e.name) && !listed[e.name] {
			listed[e.name] = true
			list = append(list, digest.Entry{Name: e.name, Sum: sha256.Sum256(e.content)})
		}
	}
	out := slices.DeleteFunc(slices.Clone(entries), func(e zentry) bool { return e.name == digest.File })
	out = append(out, zentry{name: digest.File, content: digest.Encode(list)})
	sortEntries(out)
	return out
}

// sortEntries puts es in the order pack writes: README.md first (ADR-001
// A1 item 4), then byte order of the names.
func sortEntries(es []zentry) {
	slices.SortStableFunc(es, func(a, b zentry) int {
		switch {
		case a.name == b.name:
			return 0
		case a.name == zipfmt.ReadmeName:
			return -1
		case b.name == zipfmt.ReadmeName:
			return 1
		}
		return strings.Compare(a.name, b.name)
	})
}

func codes(fs []finding.Finding) string {
	var s []string
	for _, f := range fs {
		s = append(s, string(f.Code()))
	}
	return strings.Join(s, ", ")
}

// edit returns entries with the entry name's fields changed by f.
func edit(entries []zentry, name string, f func(*zentry)) []zentry {
	out := slices.Clone(entries)
	i := slices.IndexFunc(out, func(e zentry) bool { return e.name == name })
	if i < 0 {
		panic("vectorgen: no entry " + name)
	}
	f(&out[i])
	return out
}

func without(entries []zentry, name string) []zentry {
	return slices.DeleteFunc(slices.Clone(entries), func(e zentry) bool { return e.name == name })
}

func hostile(t testing.TB, extra ...zentry) []zentry { return packed(t, baseFiles(), extra...) }

func with(files map[string][]byte, name, content string) map[string][]byte {
	out := maps.Clone(files)
	out[name] = []byte(content)
	return out
}

// sourceFiles reads the source directory of the vector id's package/.
func sourceFiles(t testing.TB, id string) map[string][]byte {
	return treeFiles(t, id, "package")
}

// treeFiles reads the tree dir of the vector id as a source directory.
func treeFiles(t testing.TB, id, dir string) map[string][]byte {
	root := filepath.Join(vectorsDir, filepath.FromSlash(id), filepath.FromSlash(dir))
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		out[filepath.ToSlash(rel)] = b
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// pack is what pack writes from files: README.md stored, every other
// entry method auto, written by zipfmt.Write (ADR-001 D4 item 7, A1 item
// 1).
func pack(t testing.TB, files map[string][]byte) []byte {
	var es []zipfmt.Entry
	for _, e := range packed(t, files) {
		m := zipfmt.Auto
		if e.name == zipfmt.ReadmeName {
			m = zipfmt.Store
		}
		es = append(es, zipfmt.Entry{Name: e.name, Content: e.content, Method: m})
	}
	var b bytes.Buffer
	if err := zipfmt.Write(&b, es); err != nil {
		t.Fatal(err)
	}
	return b.Bytes()
}

func zipOf(es []zentry) []byte { return archive{entries: es}.bytes() }

// packageCases is every committed package.
func packageCases() []pkgCase {
	signature := []byte("PK\x05\x06")
	return append([]pkgCase{
		// A valid package, packed from the source directory of
		// pkg/opens-source: the same package in both forms.
		{"pkg/opens-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "pkg/opens-source")) }},
		{"dat/combined-schema-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "dat/combined-schema-source")) }},
		{"man/canonical-form-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "man/canonical-form-source")) }},
		{"man/large-integer-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "man/large-integer-source")) }},
		{"aut/check-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "aut/check-source")) }},
		{"aut/check-xref-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "aut/check-xref-templates")) }},
		// test on a packed package reads its examples in place, with the
		// result test gives its source directory (W5-11).
		{"aut/test-packed", func(t testing.TB) []byte { return pack(t, sourceFiles(t, "aut/test-compare")) }},
		{"aut/check-refused", func(t testing.TB) []byte { return bytes.Repeat([]byte("not a zip archive\n"), 4) }},

		// unpack's packages (W4-5), each packed from the source directory
		// aut/unpack expects an unpack of it to write, so the expected
		// tree is the round trip's other half.
		{"aut/unpack", func(t testing.TB) []byte { return pack(t, treeFiles(t, "aut/unpack", "expected/src")) }},
		{"aut/unpack-refused", func(t testing.TB) []byte { return pack(t, treeFiles(t, "aut/unpack", "expected/src")) }},
		{"aut/unpack-write-failed", func(t testing.TB) []byte { return pack(t, treeFiles(t, "aut/unpack", "expected/src")) }},
		{"aut/unpack-write-stopped", func(t testing.TB) []byte { return pack(t, treeFiles(t, "aut/unpack", "expected/src")) }},

		// The container (ADR-001 D1, D8 steps 1 to 5).
		{"pkg/not-zip", func(t testing.TB) []byte { return bytes.Repeat([]byte("not a zip archive\n"), 4) }},
		{"pkg/prepended-data", func(t testing.TB) []byte {
			return archive{prefix: []byte("MZ a self-extractor stub"), entries: hostile(t)}.bytes()
		}},
		{"pkg/trailing-data", func(t testing.TB) []byte {
			return archive{entries: hostile(t), suffix: []byte("trailing bytes")}.bytes()
		}},
		{"pkg/archive-comment", func(t testing.TB) []byte {
			return archive{entries: hostile(t), comment: []byte("an archive comment")}.bytes()
		}},
		{"pkg/second-end-record", func(t testing.TB) []byte {
			inner := append(append([]byte("an inner archive: "), signature...), make([]byte, 18)...)
			return zipOf(hostile(t, zentry{name: "assets/inner.zip", content: inner}))
		}},
		{"pkg/zip64", func(t testing.TB) []byte {
			return archive{entries: hostile(t), eocd: func(e *eocd) { e.total, e.onDisk = 0xFFFF, 0xFFFF }}.bytes()
		}},
		{"pkg/encrypted", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.flags |= 1 }
			}))
		}},
		{"pkg/method", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.method = 12 }
			}))
		}},
		{"pkg/flags", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.flags |= 1 << 1 }
			}))
		}},
		{"pkg/data-descriptor", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.flags |= 1 << 3 }
			}))
		}},
		{"pkg/extra-field", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.cdExtra = []byte{0x55, 0x54, 1, 0, 0}
			}))
		}},
		{"pkg/field-value", func(t testing.TB) []byte {
			// A symbolic link's mode in the external attributes.
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.external = 0xA1FF0000 }
			}))
		}},
		{"pkg/header-mismatch", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.local = func(h *header) { h.crc = 0 }
			}))
		}},
		{"pkg/overlap", func(t testing.TB) []byte {
			// Fifield's construction: two central directory headers name
			// one local header.
			es := hostile(t, zentry{name: "assets/a.txt", content: []byte("same")}, zentry{name: "assets/b.txt", content: []byte("same")})
			b := zipOf(es)
			cd := centralHeaders(b)
			a := binary.LittleEndian.Uint32(b[cd["assets/a.txt"]+42:])
			binary.LittleEndian.PutUint32(b[cd["assets/b.txt"]+42:], a)
			return b
		}},
		{"pkg/entry-order", func(t testing.TB) []byte {
			es := hostile(t)
			i := slices.IndexFunc(es, func(e zentry) bool { return e.name == "inputs/report.schema.json" })
			es[i], es[i+1] = es[i+1], es[i]
			return zipOf(es)
		}},
		{"pkg/crc-mismatch", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.crc = 0 }
				e.local = e.central
			}))
		}},
		{"pkg/size-mismatch", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.central = func(h *header) { h.usize++ }
			}))
		}},
		{"pkg/deflate-stream", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.deflate = true
				e.data = append(deflated(e.content), "hidden"...)
			}))
		}},

		// Entry names (ADR-001 D2).
		{"pkg/name-encoding", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/a\xff.txt", content: []byte("a")}))
		}},
		{"pkg/name-not-nfc", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/cafe\u0301.txt", content: []byte("a")}))
		}},
		{"pkg/name-shape", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/../evil.txt", content: []byte("a")}))
		}},
		{"pkg/name-character", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/a\u202etxt.exe", content: []byte("a")}))
		}},
		{"pkg/name-not-portable", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/.git/config", content: []byte("a")}))
		}},
		{"pkg/name-duplicate", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/a.txt", content: []byte("a")}, zentry{name: "assets/a.txt", content: []byte("a")}))
		}},
		{"pkg/name-case-collision", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/STRASSE.txt", content: []byte("a")}, zentry{name: "assets/stra\u00dfe.txt", content: []byte("b")}))
		}},
		{"pkg/name-file-directory", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/a", content: []byte("a")}, zentry{name: "assets/a/b", content: []byte("b")}))
		}},
		{"pkg/name-directory-entry", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/dir/"}))
		}},

		// The layout and .compiled/ (ADR-001 D5, D7).
		{"pkg/unknown-top-level", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "NOTES.md", content: []byte("# Notes\n")}))
		}},

		// README.md (ADR-001 A1 item 6). Each package's digests are
		// correct, since no digest covers README.md: only step 7 refuses
		// it.
		{"pkg/readme-missing", func(t testing.TB) []byte {
			return zipOf(without(hostile(t), zipfmt.ReadmeName))
		}},
		{"pkg/readme-compressed", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), zipfmt.ReadmeName, func(e *zentry) { e.deflate = true }))
		}},
		{"pkg/readme-not-first", func(t testing.TB) []byte {
			// In its place in byte order, after .compiled/, where a
			// packer that sorted every entry would put it.
			es := hostile(t)
			slices.SortStableFunc(es, func(a, b zentry) int { return strings.Compare(a.name, b.name) })
			return zipOf(es)
		}},
		{"pkg/readme-altered", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), zipfmt.ReadmeName, func(e *zentry) {
				e.content = bytes.Replace(e.content, []byte("https://metais.org/"), []byte("https://example.com/"), 1)
			}))
		}},
		{"pkg/missing-manifest", func(t testing.TB) []byte {
			return zipOf(withDigests(without(hostile(t), "metais.yaml")))
		}},
		{"pkg/compiled-missing", func(t testing.TB) []byte {
			return zipOf(withDigests(without(hostile(t), ".compiled/references.json")))
		}},
		{"pkg/compiled-unknown-file", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: ".compiled/extra.json", content: []byte("{}")}))
		}},
		{"pkg/compiled-references-mismatch", func(t testing.TB) []byte {
			return zipOf(withDigests(edit(hostile(t), ".compiled/references.json", func(e *zentry) {
				e.content = []byte(`{"references":[{"lines":1,"path":"references/x.md","size":4,"text":true,"title":null}]}`)
			})))
		}},
		{"dat/compiled-schema-mismatch", func(t testing.TB) []byte {
			return zipOf(withDigests(edit(hostile(t), ".compiled/schema.json", func(e *zentry) {
				e.content = bytes.Replace(e.content, []byte(`"required":["report"]`), []byte(`"required":[]`), 1)
			})))
		}},
		{"man/compiled-manifest-mismatch", func(t testing.TB) []byte {
			return zipOf(withDigests(edit(hostile(t), ".compiled/manifest.json", func(e *zentry) {
				e.content = bytes.Replace(e.content, []byte(`"title":"Hostile"`), []byte(`"title":"Harmless"`), 1)
			})))
		}},
		{"man/compiled-manifest-not-canonical", func(t testing.TB) []byte {
			return zipOf(withDigests(edit(hostile(t), ".compiled/manifest.json", func(e *zentry) {
				e.content = append(slices.Clone(e.content), '\n')
			})))
		}},

		// Digests (ADR-001 D6, D8 step 6).
		{"pkg/digest-mismatch", func(t testing.TB) []byte {
			// An entry changed after packing: its digest is the old one.
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.content = []byte("# Something else\n")
			}))
		}},
		{"pkg/digest-package-mismatch", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), digest.File, func(e *zentry) {
				i := bytes.Index(e.content, []byte(`"package":"`)) + len(`"package":"`)
				e.content = slices.Clone(e.content)
				copy(e.content[i:], strings.Repeat("0", 64))
			}))
		}},
		{"pkg/digest-entries-mismatch", func(t testing.TB) []byte {
			// The digest file lacks the template and lists an entry the
			// package does not have.
			es := hostile(t)
			var list []digest.Entry
			for _, e := range es {
				switch e.name {
				case digest.File, "templates/report.md.liquid":
				default:
					list = append(list, digest.Entry{Name: e.name, Sum: sha256.Sum256(e.content)})
				}
			}
			list = append(list, digest.Entry{Name: "templates/gone.md.liquid", Sum: sha256.Sum256([]byte("gone\n"))})
			return zipOf(edit(es, digest.File, func(e *zentry) { e.content = digest.Encode(list) }))
		}},
		{"pkg/digest-not-canonical", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), digest.File, func(e *zentry) {
				e.content = bytes.ReplaceAll(e.content, []byte(`,"`), []byte(`, "`))
			}))
		}},
	}, corpusCases()...)
}

// centralHeaders is the offset of each central directory header of b, by
// its name.
func centralHeaders(b []byte) map[string]int {
	le := binary.LittleEndian
	out := map[string]int{}
	off := int(le.Uint32(b[len(b)-6:]))
	for le.Uint32(b[off:]) == sigCentral {
		n, x, c := int(le.Uint16(b[off+28:])), int(le.Uint16(b[off+30:])), int(le.Uint16(b[off+32:]))
		out[string(b[off+46:off+46+n])] = off
		off += 46 + n + x + c
	}
	return out
}
