// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"unicode"
	"unicode/utf8"

	"metais.org/metais/internal/testenv"
)

// TestModuleZipPaths holds every file of the module metais.org/metais to
// the file-path rules of Go's module zip (golang.org/x/mod's
// module.CheckFilePath and zip.CheckDir). The go command refuses to make
// the zip of a module holding one path that breaks them. So go install
// metais.org/metais/cmd/metais@<version> fails, through the proxy and
// direct alike, for the whole module, whatever the path is for: v1.0.0-rc.2
// shipped a fuzz input named nul and a vector's expected file with an
// apostrophe in its name, and its go install route failed. A directory
// holding a go.mod of its own is another module, outside this one's zip, as
// spec/conformance/vectors is (ADR-010 A11).
func TestModuleZipPaths(t *testing.T) {
	bad, err := moduleZipProblems(moduleRoot)
	if err != nil {
		t.Fatal(err)
	}
	for _, b := range bad {
		t.Errorf("%s; the go command refuses metais.org/metais's module zip, and go install with it", b)
	}
}

// TestModuleZipPathsControl feeds moduleZipPath the paths that broke
// v1.0.0-rc.2's zip, one of each other rule and paths the zip admits, and
// shows a case collision refused and another module's directory skipped.
func TestModuleZipPathsControl(t *testing.T) {
	for _, p := range []string{
		"internal/markdown/testdata/fuzz/FuzzOutline/nul",
		"spec/conformance/vectors/out/path-templates/expected/render-out/Tom & Jerry's.txt",
		"a/NUL.txt", "a/com1", "a/Lpt9.md", "a/aux.tar.gz",
		"a/b.", "a//b", "a/./b", "a/../b",
		`a/"q".txt`, "a/x*y", "a/x?y", "a/x:y", `a/x\y`, "a/x|y", "a/<x>", "a/`x`",
		"a/\x00", "a/☃.txt",
	} {
		if moduleZipPath(p) == "" {
			t.Errorf("moduleZipPath(%q) admits a path Go's module zip refuses", p)
		}
	}
	for _, p := range []string{
		"spec/conformance/vectors/out/path-templates/expected/render-out/ lead 13.txt",
		"Tom & Jerry.txt", "plain{b}.txt", "a/[x]^_~.txt", "a/x!#$%()+,-=@.txt",
		"a/café.txt", "a/中文.md", ".github/workflows/ci.yml", "a/nul-byte", "a/console", "a/com10", "a/x~y", "a/PROGRA~1/x",
	} {
		if msg := moduleZipPath(p); msg != "" {
			t.Errorf("moduleZipPath(%q) refuses a path Go's module zip admits: %s", p, msg)
		}
	}
	if got := zipPathsProblems([]string{"README.md", "docs/a.md", "readme.md"}); len(got) != 1 || !strings.Contains(got[0], "but for case") {
		t.Errorf("zipPathsProblems found %q, want the case collision alone", got)
	}
	dir := testenv.Dir(t)
	for _, f := range []string{"a.txt", "nested/go.mod", "nested/it's.txt"} {
		write(t, filepath.Join(dir, f), "x\n")
	}
	if got, err := moduleZipProblems(dir); err != nil || len(got) != 0 {
		t.Errorf("moduleZipProblems found %q (%v) in another module's directory, which the zip leaves out", got, err)
	}
}

// moduleZipProblems returns a sentence for each file below root that the
// module zip of the module rooted there refuses. It skips .git and each
// directory holding another module, as the zip does, and files that are
// not regular, which the zip leaves out.
func moduleZipProblems(root string) ([]string, error) {
	var paths []string
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if rel == "." {
				return nil
			}
			if d.Name() == ".git" {
				return filepath.SkipDir
			}
			if _, err := os.Stat(filepath.Join(p, "go.mod")); err == nil {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != ".git" && d.Type().IsRegular() {
			paths = append(paths, filepath.ToSlash(rel))
		}
		return nil
	})
	return zipPathsProblems(paths), err
}

// zipPathsProblems returns a sentence for each of a module's
// slash-separated file paths that its zip refuses: one moduleZipPath
// refuses, or one equal to another but for case.
func zipPathsProblems(paths []string) []string {
	var out []string
	folded := map[string]string{}
	for _, p := range paths {
		if msg := moduleZipPath(p); msg != "" {
			out = append(out, p+" "+msg)
		}
		k := strings.ToLower(p)
		if other, ok := folded[k]; ok {
			out = append(out, p+" is the same path as "+other+" but for case")
		} else {
			folded[k] = p
		}
	}
	return out
}

// windowsReserved are the names Windows reserves for devices, in any case
// and with any extension, which a module zip refuses as a path element.
var windowsReserved = []string{
	"CON", "PRN", "AUX", "NUL",
	"COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9",
	"LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9",
}

// moduleZipPath is why a module zip refuses the slash-separated path p, or
// "" when it admits it.
func moduleZipPath(p string) string {
	for _, elem := range strings.Split(p, "/") {
		if elem == "" || elem == "." || elem == ".." {
			return "has an empty, . or .. element"
		}
		if strings.HasSuffix(elem, ".") {
			return "has an element " + quote(elem) + " ending in a dot"
		}
		for _, r := range elem {
			if !moduleZipRune(r) {
				return "has the character " + quote(string(r)) + ", which a module zip refuses"
			}
		}
		short, _, _ := strings.Cut(elem, ".")
		if slices.ContainsFunc(windowsReserved, func(w string) bool { return strings.EqualFold(w, short) }) {
			return "has an element " + quote(elem) + ", whose name Windows reserves"
		}
	}
	return ""
}

// moduleZipRune is whether a module zip admits r in a file path: an ASCII
// letter, digit or one of a few punctuation characters and the space, or a
// letter beyond ASCII.
func moduleZipRune(r rune) bool {
	if r < utf8.RuneSelf {
		return 'a' <= r && r <= 'z' || 'A' <= r && r <= 'Z' || '0' <= r && r <= '9' ||
			strings.ContainsRune("!#$%&()+,-.=@[]^_{}~ ", r)
	}
	return unicode.IsLetter(r)
}
