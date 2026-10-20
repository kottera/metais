// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/digest"
)

// This file is the hostile corpus of ADR-009 D17 items 1 to 4 and ADR-001's
// Conformance list, beyond the cases packages_test.go began with, and
// D17 item 12's packages for the opening limits (W6-8). Each package
// carries one defect around a correct .compiled/ and correct digests,
// unless the defect is a digest (ADR-011 D16 item 5).

// The external file attributes of each file type ADR-001 D1 item 4
// refuses: a Unix mode in the high 16 bits, and for a directory the
// MS-DOS directory bit in the low byte, as zip tools write them.
const (
	modeSymlink    = 0xA1FF0000 // lrwxrwxrwx
	modeFIFO       = 0x11A40000 // prw-r--r--
	modeCharDevice = 0x21A40000 // crw-r--r--
	modeBlock      = 0x61A40000 // brw-r--r--
	modeSocket     = 0xC1A40000 // srw-r--r--
	modeDirectory  = 0x41ED0010 // drwxr-xr-x, and the MS-DOS directory bit
	modeSetuid     = 0x89A40000 // -rwSr--r--
	modeExecutable = 0x81ED0000 // -rwxr-xr-x
)

// named is the base package with one more entry, name, holding "a".
func named(t testing.TB, name string) []byte {
	return zipOf(hostile(t, zentry{name: name, content: []byte("a")}))
}

// moded is the base package with one more entry, name, holding content,
// whose central directory header gives it the external attributes ext.
func moded(t testing.TB, name, content string, ext uint32) []byte {
	return zipOf(hostile(t, zentry{name: name, content: []byte(content), central: func(h *header) { h.external = ext }}))
}

// tmplHeaders edits the base package's template entry's headers: local and
// central are applied to its local and central directory headers, either
// nil.
func tmplHeaders(t testing.TB, local, central func(*header)) []byte {
	return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
		e.local, e.central = local, central
	}))
}

// onBoth is tmplHeaders applying f to both of the template entry's
// headers.
func onBoth(t testing.TB, f func(*header)) []byte { return tmplHeaders(t, f, f) }

// both applies f to an entry's local and central directory headers.
func both(f func(*header)) (func(*header), func(*header)) { return f, f }

// deflatedTemplate edits the base package's template entry, deflated, with
// f given the entry to change.
func deflatedTemplate(t testing.TB, f func(e *zentry)) []byte {
	return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
		e.deflate = true
		f(e)
	}))
}

// localRecord is the local header and file data the profile writes for
// e, alone: the bytes a quoted overlap carries inside another entry.
func localRecord(e zentry) []byte {
	b := zipOf([]zentry{e})
	return b[:30+len(e.name)+len(e.content)]
}

// excise removes the local header and data of the entry name from b, a
// zip whose entries are stored, and points its central directory header
// at at instead: every later offset moves back by what was removed.
func excise(b []byte, name string, at uint32) []byte {
	le := binary.LittleEndian
	cd := centralHeaders(b)
	h := cd[name]
	off := le.Uint32(b[h+42:])
	n := uint32(30 + int(le.Uint16(b[h+28:])) + int(le.Uint32(b[h+20:])))
	out := append(slices.Clone(b[:off]), b[off+n:]...)
	for other, p := range cd {
		p -= int(n)
		if o := le.Uint32(out[p+42:]); o > off {
			le.PutUint32(out[p+42:], o-n)
		}
		if other == name {
			le.PutUint32(out[p+42:], at)
		}
	}
	eocd := len(out) - 22
	le.PutUint32(out[eocd+16:], le.Uint32(out[eocd+16:])-n)
	return out
}

// dataOffset is the offset of the file data of the entry name in b.
func dataOffset(b []byte, name string) uint32 {
	le := binary.LittleEndian
	h := centralHeaders(b)[name]
	return le.Uint32(b[h+42:]) + 30 + uint32(len(name))
}

// zip64End is a zip64 end of central directory record and its locator
// (APPNOTE 4.3.14, 4.3.15) for an archive of n entries whose central
// directory is size bytes at off, the record itself at off+size.
func zip64End(n int, size, off uint32) []byte {
	le := binary.LittleEndian
	var b []byte
	b = le.AppendUint32(b, 0x06064b50)
	b = le.AppendUint64(b, 44)
	b = le.AppendUint16(b, versionMadeBy)
	b = le.AppendUint16(b, 45)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint64(b, uint64(n))
	b = le.AppendUint64(b, uint64(n))
	b = le.AppendUint64(b, uint64(size))
	b = le.AppendUint64(b, uint64(off))
	b = le.AppendUint32(b, 0x07064b50)
	b = le.AppendUint32(b, 0)
	b = le.AppendUint64(b, uint64(off+size))
	return le.AppendUint32(b, 1)
}

// zip64Extra is a zip64 extended information extra field (APPNOTE 4.5.3)
// holding an entry's uncompressed and compressed sizes.
func zip64Extra(usize, csize int) []byte {
	le := binary.LittleEndian
	b := le.AppendUint16(nil, 0x0001)
	b = le.AppendUint16(b, 16)
	b = le.AppendUint64(b, uint64(usize))
	return le.AppendUint64(b, uint64(csize))
}

// digestsOf is the digest file of entries' names and contents, except
// skip's, with extra listed beside them.
func digestsOf(entries []zentry, skip string, extra ...digest.Entry) []byte {
	var list []digest.Entry
	for _, e := range entries {
		if e.name != digest.File && e.name != skip {
			list = append(list, digest.Entry{Name: e.name, Sum: sha256.Sum256(e.content)})
		}
	}
	return digest.Encode(append(list, extra...))
}

// textLike is n bytes of text that DEFLATE compresses to about 100:1: a
// sentence repeated 24 times, then eight pseudo-random hexadecimal digits.
func textLike(n int) []byte {
	var b bytes.Buffer
	x := uint32(1)
	for b.Len() < n {
		b.WriteString(strings.Repeat("The quick brown fox jumps over the lazy dog. ", 24))
		x = x*1664525 + 1013904223
		fmt.Fprintf(&b, "%08x\n", x)
	}
	return b.Bytes()[:n]
}

// limitAsset is the base package with assets/k.bin of n bytes of text,
// deflated when deflate is set: the package of a limit's pair, whose
// case lowers the limit to its bound and past it.
func limitAsset(t testing.TB, content []byte, deflate bool) []byte {
	return zipOf(hostile(t, zentry{name: "assets/k.bin", content: content, deflate: deflate}))
}

// pathOf is an entry name under assets/ of exactly length bytes, in
// segments of at most 255 bytes.
func pathOf(length int) string {
	name := "assets"
	for len(name) < length {
		seg := min(255, length-len(name)-1)
		name += "/" + strings.Repeat("p", seg)
	}
	return name
}

// depthOf is an entry name under assets/ of exactly depth segments.
func depthOf(depth int) string {
	return "assets/" + strings.Repeat("d/", depth-2) + "x"
}

// nestedBomb is a zip of Fifield's simplest overlap construction, as
// opaque bytes: n central directory headers naming one local header
// whose content is size zero bytes, deflated, so that a reader that
// opened it would inflate n times size bytes.
func nestedBomb(n, size int) []byte {
	es := []zentry{{name: "bomb", content: make([]byte, size), deflate: true}}
	b := zipOf(es)
	le := binary.LittleEndian
	cd := int(le.Uint32(b[len(b)-6:]))
	record := slices.Clone(b[cd : len(b)-22])
	out := slices.Clone(b[:cd])
	for range n {
		out = append(out, record...)
	}
	eocd := slices.Clone(b[len(b)-22:])
	le.PutUint16(eocd[8:], uint16(n))
	le.PutUint16(eocd[10:], uint16(n))
	le.PutUint32(eocd[12:], uint32(n*len(record)))
	return append(out, eocd...)
}

// corpusCases is W6-8's committed packages.
func corpusCases() []pkgCase {
	return []pkgCase{
		// Entry names (ADR-009 D17 item 1; ADR-001 Conformance, Names).
		{"pkg/name-parent", func(t testing.TB) []byte { return named(t, "../x") }},
		{"pkg/name-parent-nested", func(t testing.TB) []byte { return named(t, "a/../../x") }},
		{"pkg/name-absolute", func(t testing.TB) []byte { return named(t, "/etc/x") }},
		{"pkg/name-drive-backslash", func(t testing.TB) []byte { return named(t, `C:\x`) }},
		{"pkg/name-drive-relative", func(t testing.TB) []byte { return named(t, "C:x") }},
		{"pkg/name-unc", func(t testing.TB) []byte { return named(t, `\\server\share\x`) }},
		{"pkg/name-backslash-parent", func(t testing.TB) []byte { return named(t, `a\..\..\x`) }},
		{"pkg/name-nul", func(t testing.TB) []byte { return named(t, "assets/a\x00b") }},
		{"pkg/name-empty", func(t testing.TB) []byte { return named(t, "") }},
		{"pkg/name-dot", func(t testing.TB) []byte { return named(t, "assets/./x") }},
		{"pkg/name-empty-segment", func(t testing.TB) []byte { return named(t, "assets//b") }},
		{"pkg/name-segment-length", func(t testing.TB) []byte { return named(t, "assets/"+strings.Repeat("s", 256)) }},
		{"pkg/name-reserved", func(t testing.TB) []byte { return named(t, "assets/CON") }},
		{"pkg/name-reserved-extension", func(t testing.TB) []byte { return named(t, "assets/nul.txt") }},
		{"pkg/name-reserved-port", func(t testing.TB) []byte { return named(t, "assets/com1") }},
		{"pkg/name-reserved-superscript", func(t testing.TB) []byte { return named(t, "assets/COM¹.txt") }},
		{"pkg/name-reserved-spaced", func(t testing.TB) []byte { return named(t, "assets/Con .md") }},
		{"pkg/name-stream", func(t testing.TB) []byte { return named(t, "assets/a:b") }},
		{"pkg/name-asterisk", func(t testing.TB) []byte { return named(t, "assets/a*b") }},
		{"pkg/name-trailing-dot", func(t testing.TB) []byte { return named(t, "assets/name.") }},
		{"pkg/name-trailing-space", func(t testing.TB) []byte { return named(t, "assets/name ") }},
		{"pkg/name-control", func(t testing.TB) []byte { return named(t, "assets/a\x01b") }},
		{"pkg/name-git-upper", func(t testing.TB) []byte { return named(t, "assets/.GIT/config") }},
		{"pkg/name-git-joiner", func(t testing.TB) []byte { return named(t, "assets/.g\u200cit/config") }},
		{"pkg/name-git-short", func(t testing.TB) []byte { return named(t, "assets/git~1/config") }},
		{"pkg/name-zero-width-space", func(t testing.TB) []byte { return named(t, "assets/a\u200bb") }},
		{"pkg/name-private-use", func(t testing.TB) []byte { return named(t, "assets/a\ue000b") }},
		{"pkg/name-unassigned", func(t testing.TB) []byte { return named(t, "assets/a\u0378b") }},
		{"pkg/name-kelvin", func(t testing.TB) []byte { return named(t, "assets/\u212a.txt") }},
		{"pkg/name-case-collision-ascii", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/A.txt", content: []byte("a")}, zentry{name: "assets/a.txt", content: []byte("b")}))
		}},
		{"pkg/name-case-collision-prefix", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/Docs/a", content: []byte("a")}, zentry{name: "assets/docs/b", content: []byte("b")}))
		}},

		// Entry types and modes (D17 item 2; ADR-001 D1 item 4).
		{"pkg/symlink-inside", func(t testing.TB) []byte { return moded(t, "assets/link", "x.txt", modeSymlink) }},
		{"pkg/symlink-outside", func(t testing.TB) []byte { return moded(t, "assets/link", "/etc/passwd", modeSymlink) }},
		{"pkg/mode-fifo", func(t testing.TB) []byte { return moded(t, "assets/fifo", "", modeFIFO) }},
		{"pkg/mode-character-device", func(t testing.TB) []byte { return moded(t, "assets/tty", "", modeCharDevice) }},
		{"pkg/mode-block-device", func(t testing.TB) []byte { return moded(t, "assets/disk", "", modeBlock) }},
		{"pkg/mode-socket", func(t testing.TB) []byte { return moded(t, "assets/socket", "", modeSocket) }},
		{"pkg/mode-setuid", func(t testing.TB) []byte { return moded(t, "assets/run", "a", modeSetuid) }},
		{"pkg/mode-executable", func(t testing.TB) []byte { return moded(t, "assets/run", "a", modeExecutable) }},
		{"pkg/directory-entry-mode", func(t testing.TB) []byte { return moded(t, "assets/dir/", "", modeDirectory) }},
		{"pkg/directory-entry-data", func(t testing.TB) []byte { return moded(t, "assets/dir/", "data", modeDirectory) }},

		// Header fields (ADR-001 D1 items 3, 4 and 6; Conformance, each D1
		// field).
		{"pkg/version-needed", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.needed = 20 })) }},
		{"pkg/flag-bit-6", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.flags |= 1 << 6 })) }},
		{"pkg/flag-bit-13", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.flags |= 1 << 13 })) }},
		{"pkg/flag-reserved", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.flags |= 1 << 12 })) }},
		{"pkg/flag-utf8-clear", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.flags &^= flagUTF8 })) }},
		{"pkg/method-deflate64", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.method = 9 })) }},
		{"pkg/method-lzma", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.method = 14 })) }},
		{"pkg/method-zstd", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.method = 93 })) }},
		{"pkg/method-aes", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.method = 99 })) }},
		{"pkg/mod-time", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.time = 0x6000 })) }},
		{"pkg/mod-date", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.date = 0x5a21 })) }},
		{"pkg/version-made-by", func(t testing.TB) []byte { return tmplHeaders(t, nil, func(h *header) { h.madeBy = 0x0014 }) }},
		{"pkg/internal-attributes", func(t testing.TB) []byte { return tmplHeaders(t, nil, func(h *header) { h.internal = 1 }) }},
		{"pkg/external-attributes", func(t testing.TB) []byte {
			// The MS-DOS read-only bit beside the fixed regular-file mode.
			return tmplHeaders(t, nil, func(h *header) { h.external = externalAttrs | 1 })
		}},
		{"pkg/disk-number-start", func(t testing.TB) []byte { return tmplHeaders(t, nil, func(h *header) { h.disk = 1 }) }},
		{"pkg/file-comment", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) { e.cdComment = []byte("a file comment") }))
		}},
		{"pkg/extra-field-unicode-path", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				// Info-ZIP's Unicode Path field: a second name for the entry.
				name := "templates/other.md.liquid"
				x := binary.LittleEndian.AppendUint16(nil, 0x7075)
				x = binary.LittleEndian.AppendUint16(x, uint16(5+len(name)))
				x = append(x, 1, 0, 0, 0, 0)
				e.cdExtra = append(x, name...)
			}))
		}},
		{"pkg/zip64-extra-field", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) {
				e.cdExtra = zip64Extra(len(e.content), len(e.content))
				e.localExtra = e.cdExtra
			}))
		}},
		{"pkg/zip64-end-record", func(t testing.TB) []byte {
			es := hostile(t)
			b := zipOf(es)
			le := binary.LittleEndian
			size, off := le.Uint32(b[len(b)-10:]), le.Uint32(b[len(b)-6:])
			tail := zip64End(len(es), size, off)
			return archive{entries: es, cdTail: tail, eocd: func(e *eocd) { e.cdSize -= uint32(len(tail)) }}.bytes()
		}},
		{"pkg/end-disk-number", func(t testing.TB) []byte {
			return archive{entries: hostile(t), eocd: func(e *eocd) { e.disk, e.cdDisk = 1, 1 }}.bytes()
		}},

		// Structure (D17 item 3; ADR-001 Conformance, Structure and Data).
		{"pkg/overlap-quoted", func(t testing.TB) []byte {
			// Fifield's quoted overlap: the next entry's local header and
			// data are the content of a stored entry before it, and its
			// central directory header points inside that content.
			b := zentry{name: "assets/b.txt", content: []byte("quoted")}
			a := zentry{name: "assets/a.bin", content: localRecord(b)}
			z := zipOf(hostile(t, a, b))
			return excise(z, b.name, dataOffset(z, a.name))
		}},
		{"pkg/gap", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "inputs/report.schema.json", func(e *zentry) { e.gapAfter = []byte("hidden") }))
		}},
		{"pkg/central-order", func(t testing.TB) []byte {
			// The entries out of their names' order in the file, and the
			// central directory in their names' order, each header
			// pointing at its entry.
			es := hostile(t)
			i := slices.IndexFunc(es, func(e zentry) bool { return e.name == "inputs/report.schema.json" })
			es[i], es[i+1] = es[i+1], es[i]
			b := zipOf(es)
			le := binary.LittleEndian
			cd := int(le.Uint32(b[len(b)-6:]))
			var records [][]byte
			for p := cd; le.Uint32(b[p:]) == sigCentral; {
				n := 46 + int(le.Uint16(b[p+28:])) + int(le.Uint16(b[p+30:])) + int(le.Uint16(b[p+32:]))
				records = append(records, b[p:p+n])
				p += n
			}
			records[i], records[i+1] = records[i+1], records[i]
			return append(append(slices.Clone(b[:cd]), bytes.Join(records, nil)...), b[len(b)-22:]...)
		}},
		{"pkg/count-more", func(t testing.TB) []byte {
			return archive{entries: hostile(t), eocd: func(e *eocd) { e.onDisk++; e.total++ }}.bytes()
		}},
		{"pkg/count-fewer", func(t testing.TB) []byte {
			return archive{entries: hostile(t), eocd: func(e *eocd) { e.onDisk--; e.total-- }}.bytes()
		}},
		{"pkg/count-disk", func(t testing.TB) []byte {
			return archive{entries: hostile(t), eocd: func(e *eocd) { e.onDisk-- }}.bytes()
		}},
		{"pkg/local-name", func(t testing.TB) []byte {
			return zipOf(edit(hostile(t), "templates/report.md.liquid", func(e *zentry) { e.localName = "templates/report.md.liquix" }))
		}},
		{"pkg/local-method", func(t testing.TB) []byte { return tmplHeaders(t, func(h *header) { h.method = 8 }, nil) }},
		{"pkg/local-compressed-size", func(t testing.TB) []byte { return tmplHeaders(t, func(h *header) { h.csize-- }, nil) }},
		{"pkg/local-uncompressed-size", func(t testing.TB) []byte { return tmplHeaders(t, func(h *header) { h.usize-- }, nil) }},
		{"pkg/local-flags", func(t testing.TB) []byte { return tmplHeaders(t, func(h *header) { h.flags |= 1 << 12 }, nil) }},
		{"pkg/crc-wrong", func(t testing.TB) []byte { return onBoth(t, (func(h *header) { h.crc ^= 1 })) }},
		{"pkg/size-more-data", func(t testing.TB) []byte {
			// A deflated entry whose stream decodes past its declared size.
			return deflatedTemplate(t, func(e *zentry) {
				e.local, e.central = both(func(h *header) { h.usize-- })
			})
		}},
		{"pkg/size-less-data", func(t testing.TB) []byte {
			return deflatedTemplate(t, func(e *zentry) {
				e.local, e.central = both(func(h *header) { h.usize++ })
			})
		}},
		{"pkg/deflate-truncated", func(t testing.TB) []byte {
			return deflatedTemplate(t, func(e *zentry) {
				d := deflated(e.content)
				e.data = d[:len(d)-2]
			})
		}},
		{"pkg/truncated", func(t testing.TB) []byte {
			b := zipOf(hostile(t))
			return b[:len(b)/2]
		}},

		// The layout and .compiled/ (ADR-001 Conformance, Layout).
		{"pkg/compiled-missing-manifest", func(t testing.TB) []byte {
			return zipOf(withDigests(without(hostile(t), ".compiled/manifest.json")))
		}},
		{"pkg/compiled-missing-schema", func(t testing.TB) []byte {
			return zipOf(withDigests(without(hostile(t), ".compiled/schema.json")))
		}},
		{"pkg/compiled-missing-digests", func(t testing.TB) []byte {
			return zipOf(without(hostile(t), digest.File))
		}},
		{"pkg/compiled-stale", func(t testing.TB) []byte {
			// The combined schema and the references index both disagree
			// with what the sources derive, as a .compiled/ left from an
			// earlier packing would: two independent findings of one step.
			// A stale .compiled/manifest.json is not among them, since its
			// comparison gates the version check both derivations follow
			// (ADR-002 D11 item 8).
			es := hostile(t)
			es = edit(es, ".compiled/schema.json", func(e *zentry) {
				e.content = bytes.Replace(e.content, []byte(`"required":["report"]`), []byte(`"required":[]`), 1)
			})
			es = edit(es, ".compiled/references.json", func(e *zentry) {
				e.content = []byte(`{"references":[{"lines":1,"path":"references/x.md","size":4,"text":true,"title":null}]}`)
			})
			return zipOf(withDigests(es))
		}},

		// Digests (D17 item 4; ADR-001 Conformance, Digests).
		{"pkg/digest-not-listed", func(t testing.TB) []byte {
			// The digest file, with its package digest, made without the
			// template's line.
			es := hostile(t)
			return zipOf(edit(es, digest.File, func(e *zentry) { e.content = digestsOf(es, "templates/report.md.liquid") }))
		}},
		{"pkg/digest-no-entry", func(t testing.TB) []byte {
			es := hostile(t)
			gone := digest.Entry{Name: "templates/gone.md.liquid", Sum: sha256.Sum256([]byte("gone\n"))}
			return zipOf(edit(es, digest.File, func(e *zentry) { e.content = digestsOf(es, "", gone) }))
		}},
		{"pkg/digest-entry-added", func(t testing.TB) []byte {
			// An entry added after packing, beside the digest file it
			// predates.
			es := append(hostile(t), zentry{name: "assets/added.txt", content: []byte("added after packing\n")})
			sortEntries(es)
			return zipOf(es)
		}},
		{"sec/nested-archive", func(t testing.TB) []byte {
			return zipOf(hostile(t, zentry{name: "assets/bomb.zip", content: nestedBomb(32, 32<<20), deflate: true}))
		}},

		// The opening limits at their bounds (D17 item 12, D4 item 4):
		// each package is refused past a lowered limit its case names, and
		// accepted at it.
		{"sec/package-size-packed", func(t testing.TB) []byte { return zipOf(hostile(t)) }},
		{"sec/package-uncompressed", func(t testing.TB) []byte { return limitAsset(t, textLike(4096), true) }},
		{"sec/entries-packed", func(t testing.TB) []byte { return zipOf(hostile(t)) }},
		{"sec/entry-size", func(t testing.TB) []byte { return limitAsset(t, textLike(4096), false) }},
		{"sec/entry-ratio", func(t testing.TB) []byte { return limitAsset(t, textLike(1<<20+4096), true) }},
		{"sec/entry-ratio-floor", func(t testing.TB) []byte { return limitAsset(t, make([]byte, 1<<20), true) }},
		{"sec/entry-ratio-default", func(t testing.TB) []byte { return limitAsset(t, make([]byte, 1<<20+1), true) }},
		{"sec/path-length", func(t testing.TB) []byte { return named(t, pathOf(512)) }},
		{"sec/path-length-past", func(t testing.TB) []byte { return named(t, pathOf(513)) }},
		{"sec/path-depth", func(t testing.TB) []byte { return named(t, depthOf(32)) }},
		{"sec/path-depth-past", func(t testing.TB) []byte { return named(t, depthOf(33)) }},
	}
}
