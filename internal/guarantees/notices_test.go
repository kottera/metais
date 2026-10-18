// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// noticesFile is THIRD-PARTY-NOTICES.txt, at the module root (ADR-010 D12
// item 6), which every release archive carries.
const noticesFile = "THIRD-PARTY-NOTICES.txt"

var updateNotices = flag.Bool("notices.update", false, "rewrite "+noticesFile+" from the module cache")

// noticesHeader opens the file.
const noticesHeader = `Third-party notices for metais

The metais binary links the modules below. Each section holds, verbatim, a
licence, notice or patent file from the root of the module, at the version
linked. internal/guarantees/notices_test.go keeps this file exact; after a
dependency changes, rewrite it with

  go test ./internal/guarantees -run '^TestThirdPartyNotices$' -args -notices.update

`

// noticesRule separates the sections.
var noticesRule = strings.Repeat("=", 80)

// linkedModule is a module of the binary's closure.
type linkedModule struct {
	path, version, dir string
}

func (m linkedModule) String() string { return m.path + " " + m.version }

// linkedModules is every module but the main one that a package of
// ./cmd/metais's closure belongs to, over every target: go list -deps -f
// '{{.Module}}' ./cmd/metais, as ADR-010 D12 item 6 names it.
func linkedModules(t testing.TB, root string) []linkedModule {
	t.Helper()
	seen := map[string]linkedModule{}
	for _, tg := range targets {
		for _, p := range cachedList(t, root, tg, "./cmd/metais") {
			if p.Module == nil || p.Module.Main {
				continue
			}
			seen[p.Module.Path+" "+p.Module.Version] = linkedModule{p.Module.Path, p.Module.Version, p.Module.Dir}
		}
	}
	var mods []linkedModule
	for _, m := range seen {
		mods = append(mods, m)
	}
	slices.SortFunc(mods, func(a, b linkedModule) int { return strings.Compare(a.String(), b.String()) })
	return mods
}

// licenceFiles names the files at the root of a module directory whose
// names begin LICENSE, LICENCE, COPYING, NOTICE or PATENTS, in any case.
func licenceFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var names []string
	for _, e := range entries {
		up := strings.ToUpper(e.Name())
		if !e.Type().IsRegular() {
			continue
		}
		for _, prefix := range []string{"LICENSE", "LICENCE", "COPYING", "NOTICE", "PATENTS"} {
			if strings.HasPrefix(up, prefix) {
				names = append(names, e.Name())
				break
			}
		}
	}
	slices.Sort(names)
	return names, nil
}

// renderNotices is the whole of THIRD-PARTY-NOTICES.txt for mods.
func renderNotices(mods []linkedModule) ([]byte, error) {
	var b bytes.Buffer
	b.WriteString(noticesHeader)
	for _, m := range mods {
		names, err := licenceFiles(m.dir)
		if err != nil {
			return nil, err
		}
		if len(names) == 0 {
			return nil, fmt.Errorf("%s has no licence file at its root", m)
		}
		for _, name := range names {
			text, err := os.ReadFile(filepath.Join(m.dir, name))
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&b, "%s\n%s %s\n%s\n\n", noticesRule, m, name, noticesRule)
			b.Write(text)
			if !bytes.HasSuffix(text, []byte("\n")) {
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
	}
	return b.Bytes(), nil
}

// noticeEntries is the module of every section title in a notices file:
// the line between two rules, less its file name.
func noticeEntries(text []byte) []string {
	lines := strings.Split(string(text), "\n")
	var mods []string
	for i := 0; i+2 < len(lines); i++ {
		if lines[i] == noticesRule && lines[i+2] == noticesRule {
			title := lines[i+1]
			if j := strings.LastIndex(title, " "); j > 0 {
				title = title[:j]
			}
			if !slices.Contains(mods, title) {
				mods = append(mods, title)
			}
			i += 2
		}
	}
	return mods
}

// noticeProblems names each linked module the file has no entry for, and
// each entry for a module no longer linked.
func noticeProblems(mods []linkedModule, text []byte) []string {
	entries := noticeEntries(text)
	var bad []string
	for _, m := range mods {
		if !slices.Contains(entries, m.String()) {
			bad = append(bad, fmt.Sprintf("%s is linked into metais and has no entry", m))
		}
	}
	for _, e := range entries {
		if !slices.ContainsFunc(mods, func(m linkedModule) bool { return m.String() == e }) {
			bad = append(bad, fmt.Sprintf("the entry for %s names a module no longer linked", e))
		}
	}
	return bad
}

// TestThirdPartyNotices holds THIRD-PARTY-NOTICES.txt to the binary's
// closure: an entry for every linked module, none for a module no longer
// linked, and each entry's licence text exactly as the module ships it.
func TestThirdPartyNotices(t *testing.T) {
	root := moduleRootAbs(t)
	mods := linkedModules(t, root)
	if len(mods) == 0 {
		t.Fatal("go list found no module linked into ./cmd/metais but the main one")
	}
	want, err := renderNotices(mods)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, noticesFile)
	if *updateNotices {
		if err := os.WriteFile(path, want, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range noticeProblems(mods, got) {
		t.Errorf("%s: %s", noticesFile, p)
	}
	if !bytes.Equal(got, want) {
		t.Errorf("%s does not hold the licence texts of the linked modules %q exactly; rewrite it with -notices.update", noticesFile, mods)
	}
}

func TestNoticeProblems(t *testing.T) {
	dir := moduleRootAbs(t) // any directory with a LICENSE at its root
	a := linkedModule{"example.com/a", "v1.0.0", dir}
	b := linkedModule{"example.com/b", "v2.0.0", dir}
	text, err := renderNotices([]linkedModule{a, b})
	if err != nil {
		t.Fatal(err)
	}
	if got := noticeEntries(text); !slices.Equal(got, []string{a.String(), b.String()}) {
		t.Fatalf("noticeEntries = %q", got)
	}
	if got := noticeProblems([]linkedModule{a, b}, text); len(got) > 0 {
		t.Errorf("an exact file: %q", got)
	}
	cases := []struct {
		name string
		mods []linkedModule
		want string
	}{
		{"a module with no entry", []linkedModule{a, b, {"example.com/c", "v0.1.0", dir}}, "example.com/c v0.1.0 is linked into metais and has no entry"},
		{"an entry no longer linked", []linkedModule{a}, "the entry for example.com/b v2.0.0 names a module no longer linked"},
		{"an upgraded module", []linkedModule{a, {"example.com/b", "v2.1.0", dir}}, "the entry for example.com/b v2.0.0 names a module no longer linked"},
	}
	for _, c := range cases {
		got := noticeProblems(c.mods, text)
		if !slices.Contains(got, c.want) {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
	if _, err := renderNotices([]linkedModule{{"example.com/bare", "v1.0.0", filepath.Join(dir, "internal", "fault")}}); err == nil {
		t.Error("renderNotices accepted a module with no licence file")
	}
}
