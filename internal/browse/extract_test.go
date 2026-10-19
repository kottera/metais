// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package browse

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/testenv"
)

// seed writes files beneath root, each holding "theirs " and its path; a
// name ending "/" is a directory.
func seed(t *testing.T, root string, files ...string) {
	t.Helper()
	for _, f := range files {
		p := filepath.Join(root, filepath.FromSlash(f))
		if strings.HasSuffix(f, "/") {
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("theirs "+f), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// hostTree is every path beneath root with what it holds, in byte order: a
// directory as its path and "/", a file as its path and its bytes.
func hostTree(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	filepath.WalkDir(root, func(p string, d os.DirEntry, err error) error {
		if err != nil || p == root {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		rel = filepath.ToSlash(rel)
		switch {
		case d.IsDir():
			out = append(out, rel+"/")
		case d.Type().IsRegular():
			b, _ := os.ReadFile(p)
			out = append(out, rel+"="+string(b))
		default:
			out = append(out, rel+" "+d.Type().String())
		}
		return nil
	})
	return out
}

// extractTree is the package every test of an existing destination
// extracts from.
var extractTree = map[string]string{
	"metais.yaml":         "manifest",
	"inputs/a.json":       "{}",
	"inputs/x/b.json":     "{}!",
	"references/notes.md": "notes",
}

var extractAll = []string{"inputs/a.json", "inputs/x/b.json", "metais.yaml", "references/notes.md"}

// TestExtractRefusesANonEmptyDirectory is W5-12.1's refusal: without
// Overwrite, an existing directory that holds any entry, a hidden one
// included, is refused listing up to ten of its entries in byte order,
// with nothing written, in a dry run too.
func TestExtractRefusesANonEmptyDirectory(t *testing.T) {
	tr := tree(t, extractTree)
	for _, c := range []struct {
		name string
		host []string
		want []string
		more bool
	}{
		{"one file", []string{"notes.txt"}, []string{"notes.txt"}, false},
		{"a hidden file", []string{".DS_Store"}, []string{".DS_Store"}, false},
		{"a staging leftover", []string{hostfs.StagingDir + "/"}, []string{hostfs.StagingDir}, false},
		{"a file extract would write", []string{"metais.yaml"}, []string{"metais.yaml"}, false},
		{"eleven", []string{"k", "j", "i", "h", "g", "f", "e", "d", "c", "b", "a"},
			[]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"}, true},
	} {
		for _, dry := range []bool{false, true} {
			to := testenv.Dir(t)
			seed(t, to, c.host...)
			before := hostTree(t, to)
			files, err := Extract(tr, extractAll, to, ExtractOptions{DryRun: dry}, 0)
			var de *DestinationError
			if !errors.As(err, &de) || de.NotDirectory || !slices.Equal(de.Entries, c.want) || de.More != c.more || files != nil {
				t.Errorf("%s, dry run %v: %+v %#v; want the entries %q, more %v", c.name, dry, files, err, c.want, c.more)
			}
			if after := hostTree(t, to); !slices.Equal(after, before) {
				t.Errorf("%s, dry run %v: the refusal wrote: %q, was %q", c.name, dry, after, before)
			}
		}
	}
}

// TestExtractOverwrite is W5-12.1's --overwrite: into an existing
// directory that holds entries, every file is written at its path, a file
// already there is replaced and keeps no old byte, and nothing else in
// the directory is changed or removed, a file beside a replaced one and a
// hidden one included. A dry run lists the same files and writes nothing.
func TestExtractOverwrite(t *testing.T) {
	tr := tree(t, extractTree)
	host := []string{".hidden", "metais.yaml", "inputs/foreign.json", "inputs/x/b.json", "notes.txt"}
	want := []string{
		".hidden=theirs .hidden", "inputs/", "inputs/a.json={}", "inputs/foreign.json=theirs inputs/foreign.json",
		"inputs/x/", "inputs/x/b.json={}!", "metais.yaml=manifest", "notes.txt=theirs notes.txt",
		"references/", "references/notes.md=notes",
	}
	wantFiles := []File{{"inputs/a.json", 2}, {"inputs/x/b.json", 3}, {"metais.yaml", 8}, {"references/notes.md", 5}}

	dry := testenv.Dir(t)
	seed(t, dry, host...)
	before := hostTree(t, dry)
	files, err := Extract(tr, extractAll, dry, ExtractOptions{DryRun: true, Overwrite: true}, 0)
	if err != nil || !slices.Equal(files, wantFiles) {
		t.Errorf("dry run: %+v %v", files, err)
	}
	if after := hostTree(t, dry); !slices.Equal(after, before) {
		t.Errorf("the dry run wrote: %q, was %q", after, before)
	}

	to := testenv.Dir(t)
	seed(t, to, host...)
	files, err = Extract(tr, extractAll, to, ExtractOptions{Overwrite: true}, 0)
	if err != nil || !slices.Equal(files, wantFiles) {
		t.Fatalf("Extract: %+v %v", files, err)
	}
	if got := hostTree(t, to); !slices.Equal(got, want) {
		t.Errorf("after the overwrite:\n%q\nwant\n%q", got, want)
	}

	// Into an empty directory, with or without Overwrite, the same files.
	empty := testenv.Dir(t)
	if files, err := Extract(tr, extractAll, empty, ExtractOptions{Overwrite: true}, 0); err != nil || !slices.Equal(files, wantFiles) {
		t.Errorf("into an empty directory with Overwrite: %+v %v", files, err)
	}
}

// TestExtractRefusesTheChecksTogether is W5-12.1's per-file checks: every
// file the checks refuse is reported in one *CheckError, in path order,
// each with its reason, and nothing is written, in a dry run too.
func TestExtractRefusesTheChecksTogether(t *testing.T) {
	tr := tree(t, extractTree)
	host := []string{"inputs/a.json/", "metais.yaml/", "references", "notes.txt"}
	for _, dry := range []bool{false, true} {
		to := testenv.Dir(t)
		seed(t, to, host...)
		before := hostTree(t, to)
		_, err := Extract(tr, extractAll, to, ExtractOptions{DryRun: dry, Overwrite: true}, 0)
		var ce *CheckError
		if !errors.As(err, &ce) {
			t.Fatalf("dry run %v: %#v", dry, err)
		}
		var got []string
		for _, c := range ce.Refused {
			got = append(got, c.Path+" "+string(hostfs.ReasonOf(c.Err)))
		}
		want := []string{"inputs/a.json is_directory", "metais.yaml is_directory", "references/notes.md not_directory"}
		if !slices.Equal(got, want) {
			t.Errorf("dry run %v: refused %q, want %q", dry, got, want)
		}
		if after := hostTree(t, to); !slices.Equal(after, before) {
			t.Errorf("dry run %v: the refusal wrote: %q, was %q", dry, after, before)
		}
	}
}

// TestExtractRefusesAStagingLeftover is ADR-005 A1 item 1 for extract: a
// staging directory left in the destination refuses the write with
// exists, nothing written and the leftover left, in a dry run too; and a
// file whose first segment is the staging name is refused by the checks
// (A1 item 5).
func TestExtractRefusesAStagingLeftover(t *testing.T) {
	tr := tree(t, extractTree)
	for _, dry := range []bool{false, true} {
		to := testenv.Dir(t)
		seed(t, to, hostfs.StagingDir+"/x")
		before := hostTree(t, to)
		_, err := Extract(tr, extractAll, to, ExtractOptions{DryRun: dry, Overwrite: true}, 0)
		var we *WriteError
		if !errors.As(err, &we) || we.Reason != string(hostfs.Exists) || len(we.Written) != 0 {
			t.Errorf("dry run %v: %#v", dry, err)
		}
		if after := hostTree(t, to); !slices.Equal(after, before) {
			t.Errorf("dry run %v: the refusal wrote: %q, was %q", dry, after, before)
		}
	}
	staging := tree(t, map[string]string{hostfs.StagingDir + "/a.txt": "a", "b.txt": "b"})
	_, err := Extract(staging, []string{hostfs.StagingDir + "/a.txt", "b.txt"}, testenv.Dir(t), ExtractOptions{}, 0)
	var ce *CheckError
	if !errors.As(err, &ce) || len(ce.Refused) != 1 || hostfs.ReasonOf(ce.Refused[0].Err) != hostfs.Exists {
		t.Errorf("a file beneath the staging name: %#v", err)
	}
}

// TestExtractIntoAnExistingDirectoryFailsWhole is a write into an existing
// directory that fails before its commit: the directory is left as it
// was, with no staging directory, and the failure lists no file.
func TestExtractIntoAnExistingDirectoryFailsWhole(t *testing.T) {
	tr := tree(t, extractTree)
	to := testenv.Dir(t)
	seed(t, to, "notes.txt")
	before := hostTree(t, to)
	_, err := Extract(failing{tr, "metais.yaml"}, extractAll, to, ExtractOptions{Overwrite: true}, 0)
	var we *WriteError
	if !errors.As(err, &we) || we.Reason != string(hostfs.IOError) || len(we.Written) != 0 {
		t.Fatalf("a failed read: %#v", err)
	}
	if after := hostTree(t, to); !slices.Equal(after, before) {
		t.Errorf("after the failure: %q, was %q", after, before)
	}
}
