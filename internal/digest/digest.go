// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package digest is ADR-001 D6: the digest list of a package's entries,
// the package digest computed from it, and the digest file
// .compiled/digests.json that carries both in a packed package.
//
// The package digest depends only on the names and the content of the
// entries, never on compression, entry order or the tool that packed them,
// so a source directory and the package packed from it have one digest
// (ADR-001 D6 item 3).
package digest

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// File is the digest file's entry name (ADR-001 D7 item 1).
const File = ".compiled/digests.json"

// Algorithm is the one value of the digest file's algorithm member.
const Algorithm = "sha256"

// Sum is a SHA-256.
type Sum = [sha256.Size]byte

// Entry is one entry of a package: its name and the SHA-256 of its
// uncompressed content.
type Entry struct {
	Name string
	Sum  Sum
}

// The codes Verify reports (ADR-001 Interfaces). Each is class refused,
// about the package.
const (
	CodeNotCanonical    finding.Code = "package.digest.not_canonical"
	CodeEntriesMismatch finding.Code = "package.digest.entries_mismatch"
	CodeMismatch        finding.Code = "package.digest.mismatch"
	CodePackageMismatch finding.Code = "package.digest.package_mismatch"
)

// The reasons of package.digest.not_canonical and entries_mismatch.
const (
	reasonSyntax       = "syntax"
	reasonShape        = "shape"
	reasonNotCanonical = "not_canonical"
	reasonNotListed    = "not_listed"
	reasonNoEntry      = "no_entry"
)

// List is ADR-001 D6 item 2's digest list of entries: one line per entry,
// in ascending order of the names' UTF-8 bytes, each the 64 lowercase
// hexadecimal digits of its SHA-256, two spaces, the name and a line feed.
// It is exactly GNU sha256sum's untagged output for those files (item 4).
// entries need not be sorted, and are not changed.
func List(entries []Entry) []byte {
	sorted := slices.Clone(entries)
	slices.SortFunc(sorted, func(a, b Entry) int { return strings.Compare(a.Name, b.Name) })
	var b bytes.Buffer
	for _, e := range sorted {
		b.WriteString(hex.EncodeToString(e.Sum[:]))
		b.WriteString("  ")
		b.WriteString(e.Name)
		b.WriteByte('\n')
	}
	return b.Bytes()
}

// Package is the package digest of entries: the SHA-256 of their digest
// list (ADR-001 D6 item 2). entries are every entry of the package but the
// digest file itself.
func Package(entries []Entry) Sum {
	return sha256.Sum256(List(entries))
}

// Format presents a digest as ADR-001 D6 item 5 does: sha256: and its 64
// lowercase hexadecimal digits.
func Format(s Sum) string {
	return Algorithm + ":" + hex.EncodeToString(s[:])
}

// Encode is the digest file of entries, every entry of the package but the
// digest file itself, in ADR-002 D9's canonical form (ADR-001 D6 item 1).
func Encode(entries []Entry) []byte {
	members := make([]value.Member, 0, len(entries))
	for _, e := range entries {
		members = append(members, value.Member{Name: e.Name, Value: str(hex.EncodeToString(e.Sum[:]))})
	}
	list, err := value.NewObject(members...)
	if err != nil {
		// Entry names are unique in a package, so this is a defect in
		// the caller.
		panic("digest: " + err.Error())
	}
	pkg := Package(entries)
	root, err := value.NewObject(
		value.Member{Name: "algorithm", Value: str(Algorithm)},
		value.Member{Name: "entries", Value: list},
		value.Member{Name: "package", Value: str(hex.EncodeToString(pkg[:]))},
	)
	if err != nil {
		panic("digest: " + err.Error())
	}
	return jsonx.Canonical(root)
}

func str(s string) value.String {
	v, err := value.NewString(s)
	if err != nil {
		panic("digest: " + err.Error())
	}
	return v
}

// Verify is ADR-001 D8 step 6 over src, the digest file's bytes, and
// entries, every entry of the package but the digest file with the SHA-256
// step 5 computed. It reports, all at once: a digest file not in its form,
// which ends the step; every entry the file does not list and every name
// it lists that is no entry; every entry whose content is not the content
// the file names; and a package digest that is not the digest of the
// file's entries. It returns no finding for a digest file that verifies.
func Verify(src []byte, entries []Entry, lim limits.Set) []finding.Finding {
	listed, pkgHex, reason := parse(src, lim)
	if reason != "" {
		return []finding.Finding{refusal(CodeNotCanonical, File, "", finding.Params{"reason": finding.String(reason)})}
	}
	var out []finding.Finding
	computed := map[string]Sum{}
	for _, e := range entries {
		computed[e.Name] = e.Sum
	}
	names := make([]string, 0, len(computed)+len(listed))
	for n := range computed {
		names = append(names, n)
	}
	for n := range listed {
		if _, ok := computed[n]; !ok {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	for _, n := range names {
		sum, isEntry := computed[n]
		hexText, isListed := listed[n]
		switch {
		case !isListed:
			out = append(out, refusal(CodeEntriesMismatch, n, "", finding.Params{"reason": finding.String(reasonNotListed)}))
		case !isEntry:
			out = append(out, refusal(CodeEntriesMismatch, File, finding.Pointer("entries", n), finding.Params{"reason": finding.String(reasonNoEntry)}))
		case hexText != hex.EncodeToString(sum[:]):
			out = append(out, refusal(CodeMismatch, n, "", finding.Params{
				"listed":   finding.String(hexText),
				"computed": finding.String(hex.EncodeToString(sum[:])),
			}))
		}
	}
	// The package digest is the digest of the entries the file lists, so
	// a file that lists one entry wrongly and gives the package digest of
	// what it lists is refused once, for that entry.
	var fromFile []Entry
	for n, h := range listed {
		var s Sum
		hex.Decode(s[:], []byte(h))
		fromFile = append(fromFile, Entry{Name: n, Sum: s})
	}
	if want := Package(fromFile); pkgHex != hex.EncodeToString(want[:]) {
		out = append(out, refusal(CodePackageMismatch, File, finding.Pointer("package"), finding.Params{
			"listed":   finding.String(pkgHex),
			"computed": finding.String(hex.EncodeToString(want[:])),
		}))
	}
	return out
}

// refusal is a finding about the entry file, at ptr within it when ptr is
// not empty.
func refusal(code finding.Code, file, ptr string, p finding.Params) finding.Finding {
	loc := finding.Location{Origin: finding.OriginPackage, File: file}
	if ptr != "" {
		loc = loc.WithPointer(ptr)
	}
	return finding.New(code, finding.Subject{Kind: finding.SubjectPackage}, &loc, p)
}

// parse reads the digest file: its entries' hexadecimal digests by name,
// and its package digest. It returns the reason the file is not in its
// form when it is not: syntax, when it is not one JSON object; shape, when
// it does not hold exactly ADR-001 D6 item 1's members and values; and
// not_canonical, when its bytes are not ADR-002 D9's form of its value.
func parse(src []byte, lim limits.Set) (map[string]string, string, string) {
	doc, fs := jsonx.Parse(src, jsonx.Source{
		Input: finding.Subject{Kind: finding.SubjectInput, Name: "digests"},
		At:    finding.Location{Origin: finding.OriginPackage, File: File},
	}, lim)
	if len(fs) > 0 {
		return nil, "", reasonSyntax
	}
	root, ok := doc.Value.(value.Object)
	if !ok || root.Len() != 3 {
		return nil, "", reasonShape
	}
	alg, okA := text(root, "algorithm")
	pkg, okP := text(root, "package")
	ev, okE := root.Get("entries")
	entries, okO := ev.(value.Object)
	if !okA || !okP || !okE || !okO || alg != Algorithm || !isHex(pkg) {
		return nil, "", reasonShape
	}
	listed := make(map[string]string, entries.Len())
	for name, v := range entries.All() {
		s, ok := v.(value.String)
		if !ok || !isHex(s.Text()) {
			return nil, "", reasonShape
		}
		listed[name] = s.Text()
	}
	if !bytes.Equal(jsonx.Canonical(root), src) {
		return nil, "", reasonNotCanonical
	}
	return listed, pkg, ""
}

func text(o value.Object, name string) (string, bool) {
	v, ok := o.Get(name)
	if !ok {
		return "", false
	}
	s, ok := v.(value.String)
	if !ok {
		return "", false
	}
	return s.Text(), true
}

// isHex reports whether s is 64 lowercase hexadecimal digits.
func isHex(s string) bool {
	if len(s) != 2*sha256.Size {
		return false
	}
	for i := 0; i < len(s); i++ {
		if c := s[i]; !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}
