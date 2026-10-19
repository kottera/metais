// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"unicode/utf8"
)

// decodeJSON decodes b keeping numbers as their literal text, so a
// comparison never rounds.
func decodeJSON(b []byte) (any, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.UseNumber()
	var v any
	if err := dec.Decode(&v); err != nil {
		return nil, err
	}
	if dec.More() {
		return nil, fmt.Errorf("more than one JSON value")
	}
	return v, nil
}

// matches reports whether got matches want as ADR-011 D10 item 2 compares
// findings: each member of an expected object equals the actual member of
// the same name, recursively; arrays have the same length and match
// element by element; everything else is equal.
func matches(want, got any) bool {
	switch w := want.(type) {
	case map[string]any:
		g, ok := got.(map[string]any)
		if !ok {
			return false
		}
		for k, wv := range w {
			gv, ok := g[k]
			if !ok || !matches(wv, gv) {
				return false
			}
		}
		return true
	case []any:
		g, ok := got.([]any)
		if !ok || len(g) != len(w) {
			return false
		}
		for i := range w {
			if !matches(w[i], g[i]) {
				return false
			}
		}
		return true
	case json.Number:
		g, ok := got.(json.Number)
		return ok && g == w
	default:
		return want == got
	}
}

// project keeps of got only the members want names, recursively, which is
// what a failure shows beside the expectation.
func project(want, got any) any {
	w, ok := want.(map[string]any)
	g, ok2 := got.(map[string]any)
	if !ok || !ok2 {
		return got
	}
	out := map[string]any{}
	for k, wv := range w {
		if gv, ok := g[k]; ok {
			out[k] = project(wv, gv)
		}
	}
	return out
}

// compareFindings compares the actual findings with the expected ones and
// returns the first difference, with both projections side by side, or "".
func compareFindings(want, got []any) string {
	if len(want) == len(got) {
		for i := range want {
			if !matches(want[i], got[i]) {
				return fmt.Sprintf("finding %d differs\n%s", i+1, sideBySide(want, got))
			}
		}
		return ""
	}
	return fmt.Sprintf("expected %d findings, got %d\n%s", len(want), len(got), sideBySide(want, got))
}

// sideBySide lays the expected findings beside the actual findings'
// projections onto them, one pair per line.
func sideBySide(want, got []any) string {
	var b strings.Builder
	for i := 0; i < max(len(want), len(got)); i++ {
		var w, g any = "-", "-"
		if i < len(want) {
			w = want[i]
		}
		if i < len(got) {
			g = got[i]
			if i < len(want) {
				g = project(want[i], got[i])
			}
		}
		wb, _ := json.Marshal(w)
		gb, _ := json.Marshal(g)
		mark := " "
		if i >= len(want) || i >= len(got) || !matches(want[i], got[i]) {
			mark = "≠"
		}
		fmt.Fprintf(&b, "  %s expected %s\n    actual   %s\n", mark, wb, gb)
	}
	return b.String()
}

// firstDifference shows where got first differs from want (ADR-011 D11
// item 8): a unified diff of at most 40 lines for text, and a 64-byte
// hexadecimal window for anything else.
func firstDifference(want, got []byte) string {
	if isText(want) && isText(got) {
		return lineDiff(string(want), string(got), 40)
	}
	i := 0
	for i < len(want) && i < len(got) && want[i] == got[i] {
		i++
	}
	from := i - i%16
	window := func(b []byte) string {
		if from >= len(b) {
			return "(ends)"
		}
		return hex.EncodeToString(b[from:min(len(b), from+64)])
	}
	return fmt.Sprintf("  first difference at byte %d (%d and %d bytes)\n  expected @%d: %s\n  actual   @%d: %s",
		i, len(want), len(got), from, window(want), from, window(got))
}

// isText reports whether b is UTF-8 with no NUL byte.
func isText(b []byte) bool {
	return utf8.Valid(b) && !bytes.Contains(b, []byte{0})
}

// lineDiff is a unified diff of the region where two texts differ, with
// three lines of context, cut to limit lines.
func lineDiff(want, got string, limit int) string {
	w := strings.SplitAfter(want, "\n")
	g := strings.SplitAfter(got, "\n")
	pre := 0
	for pre < len(w) && pre < len(g) && w[pre] == g[pre] {
		pre++
	}
	suf := 0
	for suf < len(w)-pre && suf < len(g)-pre && w[len(w)-1-suf] == g[len(g)-1-suf] {
		suf++
	}
	from := max(0, pre-3)
	var lines []string
	lines = append(lines, fmt.Sprintf("@@ -%d +%d @@", from+1, from+1))
	for _, l := range w[from:pre] {
		lines = append(lines, " "+l)
	}
	for _, l := range w[pre : len(w)-suf] {
		lines = append(lines, "-"+l)
	}
	for _, l := range g[pre : len(g)-suf] {
		lines = append(lines, "+"+l)
	}
	for _, l := range w[len(w)-suf : min(len(w), len(w)-suf+3)] {
		lines = append(lines, " "+l)
	}
	if len(lines) > limit {
		lines = append(lines[:limit-1], "… (cut)")
	}
	for i, l := range lines {
		if !strings.HasSuffix(l, "\n") {
			lines[i] = l + "\n"
		}
	}
	return strings.Join(lines, "")
}

// treeFile is one regular file of a written tree.
type treeFile struct {
	content []byte
	exec    bool
}

// tree is what a declared tree held after a step: its regular files by
// slash path, and every entry that is not one.
type tree struct {
	files map[string]treeFile
	links []string
	other []string
	empty []string // directories holding nothing
}

// readTree reads the tree at dir. A missing tree has no files.
func readTree(dir string) (*tree, error) {
	t := &tree{files: map[string]treeFile{}}
	if _, err := os.Lstat(dir); err != nil {
		if os.IsNotExist(err) {
			return t, nil
		}
		return nil, err
	}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			entries, err := os.ReadDir(p)
			if err != nil {
				return err
			}
			if len(entries) == 0 && rel != "." {
				t.empty = append(t.empty, rel)
			}
		case d.Type().IsRegular():
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			fi, err := d.Info()
			if err != nil {
				return err
			}
			t.files[rel] = treeFile{content: b, exec: fi.Mode().Perm()&0o100 != 0}
		case d.Type()&fs.ModeSymlink != 0:
			t.links = append(t.links, rel)
		default:
			t.other = append(t.other, rel)
		}
		return nil
	})
	return t, err
}

// substitute replaces the tool tokens of ADR-011 D10 item 3 in b.
func substitute(b []byte, id toolID) []byte {
	b = bytes.ReplaceAll(b, []byte("@@TOOL_NAME@@"), []byte(id.Name))
	return bytes.ReplaceAll(b, []byte("@@TOOL_VERSION@@"), []byte(id.Version))
}

// compareTree compares the tree a step wrote with what te expects (ADR-011
// D10 item 3) and returns every difference. The expected tree is read from
// the vector; id fills the substitution tokens.
func compareTree(got *tree, v *vector, te treeExpect, id func() (toolID, error)) []string {
	var diffs []string
	for _, l := range got.links {
		diffs = append(diffs, fmt.Sprintf("%s is a link", l))
	}
	for _, o := range got.other {
		diffs = append(diffs, fmt.Sprintf("%s is not a regular file", o))
	}
	for _, e := range got.empty {
		diffs = append(diffs, fmt.Sprintf("%s is an empty directory", e))
	}
	if te.Equals == "" {
		for _, f := range te.Files {
			if _, ok := got.files[f]; !ok {
				diffs = append(diffs, fmt.Sprintf("%s is missing", f))
			}
		}
		return append(diffs, execDiffs(got, te)...)
	}
	want, err := readTree(filepath.Join(v.dir, filepath.FromSlash(te.Equals)))
	if err != nil {
		return append(diffs, "the expected tree: "+err.Error())
	}
	names := slices.Sorted(maps.Keys(want.files))
	for n := range got.files {
		if _, ok := want.files[n]; !ok {
			names = append(names, n)
		}
	}
	slices.Sort(names)
	for _, n := range names {
		w, inWant := want.files[n]
		g, inGot := got.files[n]
		switch {
		case !inGot:
			diffs = append(diffs, fmt.Sprintf("%s is missing", n))
			continue
		case !inWant:
			diffs = append(diffs, fmt.Sprintf("%s is extra", n))
			continue
		}
		wb := w.content
		if slices.Contains(te.Substitute, n) {
			tid, err := id()
			if err != nil {
				diffs = append(diffs, fmt.Sprintf("%s: the tool's identity: %v", n, err))
				continue
			}
			wb = substitute(wb, tid)
		}
		if i := slices.IndexFunc(te.Containers, func(c containerSpec) bool { return c.Path == n }); i >= 0 {
			methods, all := te.Containers[i].methods()
			if d := compareZips(wb, g.content, methods, all); d != "" {
				diffs = append(diffs, fmt.Sprintf("%s, compared as a container: %s", n, d))
			}
			continue
		}
		if !bytes.Equal(wb, g.content) {
			diffs = append(diffs, fmt.Sprintf("%s differs\n%s", n, firstDifference(wb, g.content)))
		}
	}
	return append(diffs, execDiffs(got, te)...)
}

// execDiffs checks the owner-execute bits: set exactly on the files te
// lists, where the file system has POSIX modes.
func execDiffs(got *tree, te treeExpect) []string {
	if runtime.GOOS == "windows" {
		return nil
	}
	var diffs []string
	for _, n := range slices.Sorted(maps.Keys(got.files)) {
		want := slices.Contains(te.Executable, n)
		if got.files[n].exec != want {
			diffs = append(diffs, fmt.Sprintf("%s: owner-execute bit is %t, expected %t", n, got.files[n].exec, want))
		}
	}
	return diffs
}

// compareListing compares a dry run's listing with the expected one
// exactly, except that only the path of an entry in containers is
// compared (ADR-011 D10 item 6).
func compareListing(want, got any, containers []string) string {
	w, ok1 := want.([]any)
	g, ok2 := got.([]any)
	if !ok1 || !ok2 {
		return "the listing is not an array"
	}
	if len(w) != len(g) {
		return fmt.Sprintf("expected %d listed outputs, got %d", len(w), len(g))
	}
	for i := range w {
		wo, _ := w[i].(map[string]any)
		gobj, _ := g[i].(map[string]any)
		if wo == nil || gobj == nil {
			return fmt.Sprintf("listed output %d is not an object", i+1)
		}
		if p, _ := wo["path"].(string); slices.Contains(containers, p) {
			if wo["path"] != gobj["path"] {
				return fmt.Sprintf("listed output %d: path %v, expected %v", i+1, gobj["path"], p)
			}
			continue
		}
		if !matches(wo, gobj) || len(wo) != len(gobj) {
			wb, _ := json.Marshal(wo)
			gb, _ := json.Marshal(gobj)
			return fmt.Sprintf("listed output %d differs\n  expected %s\n  actual   %s", i+1, wb, gb)
		}
	}
	return ""
}
