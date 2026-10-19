// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/testenv"
)

// existingWrite is one writer of an existing output directory, as the
// existing-destination detector runs it: files are the paths it writes
// beneath the output directory, "/"-separated, dirs the empty directories
// it creates there, and run makes the write into out under faults. A command that writes into an existing directory
// under ADR-005 Amendment A1 — render, extract --overwrite, sample --to,
// test --update — is run through the detector by giving it one of these,
// whose run invokes the command with out as its destination.
type existingWrite struct {
	name  string
	files []string
	dirs  []string
	run   func(t *testing.T, out string, faults fault.Set) error
}

// existingWrites are the writers the detector runs. internal/hostfs's
// write shape is the first; each command that writes through it adds its
// own.
var existingWrites = []existingWrite{
	{
		name:  "hostfs.Existing",
		files: hostfsFiles,
		run:   runHostfsExisting,
	},
	{
		name:  "metais render --overwrite",
		files: hostfsFiles,
		run:   runRenderOverwrite,
	},
	{
		name:  "metais extract --overwrite",
		files: extractFiles,
		run:   runExtractOverwrite,
	},
}

// extractFiles are the entries runExtractOverwrite extracts. The detector
// seeds every other one, so extract replaces a file in a directory it
// creates in the staging directory and one at the top, writes a file
// beneath directories that do not exist, and one into a directory that
// does.
var extractFiles = []string{"examples/basic/expected/report.md", "references/standards/practice.md", "inputs/project.md", "inputs/items.schema.yaml", "metais.yaml"}

// runExtractOverwrite runs metais extract --overwrite in-process, with each
// of extractFiles as a glob, from a committed packed package into out.
func runExtractOverwrite(t *testing.T, out string, faults fault.Set) error {
	t.Helper()
	pkg, err := filepath.Abs(filepath.Join(moduleRoot, "spec", vectorsDir, "pkg", "opens-packed", "package.metais"))
	if err != nil {
		t.Fatal(err)
	}
	args := append([]string{"extract", "--json", pkg}, extractFiles...)
	args = append(args, "--to", out, "--overwrite")
	var stdout, stderr bytes.Buffer
	cli.Run(context.Background(), cli.Invocation{
		Args: args, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	var doc struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil || !doc.OK {
		return fmt.Errorf("metais extract --overwrite: %q %q", stdout.String(), stderr.String())
	}
	return nil
}

// hostfsFiles are the files runHostfsExisting writes. The detector seeds
// every other one, so it writes two replaces, a file beneath directories
// that do not exist, and a file into a directory that does.
var hostfsFiles = []string{"a/b/replaced.txt", "fresh/deep/new.txt", "top.txt", "a/new.txt"}

// runHostfsExisting writes hostfsFiles into out through hostfs's write shape for
// an existing directory, each file holding its own path.
func runHostfsExisting(t *testing.T, out string, faults fault.Set) error {
	t.Helper()
	files := hostfsFiles
	x, err := hostfs.OpenExisting(out, hostfs.WithFaults(faults))
	if err != nil {
		return err
	}
	defer x.Close()
	checks := x.Check(files)
	for _, c := range checks {
		if c.Err != nil {
			return fmt.Errorf("%s: %w", c.Path, c.Err)
		}
	}
	o, err := x.Stage(checks)
	if err != nil {
		return err
	}
	for _, f := range files {
		if err := o.Create(f, false, strings.NewReader("new "+f)); err != nil {
			o.Abort()
			return err
		}
	}
	return o.Commit()
}

// runRenderOverwrite renders, with metais render --overwrite, a package
// whose outputs are hostfsFiles into out, each output holding its own
// path (ADR-005 D15 item 3).
func runRenderOverwrite(t *testing.T, out string, faults fault.Set) error {
	t.Helper()
	pkg := filepath.Join(testenv.Dir(t), "pkg")
	manifest := "metais: \"1.0\"\nname: example\nversion: 1.0.0\ntitle: Example\n" +
		"description: An example package.\nguide: Supply the inputs and render.\ninputs: []\noutputs:\n"
	for i, f := range hostfsFiles {
		manifest += fmt.Sprintf("  - id: f%d\n    template: f.txt.liquid\n    path: %s\n", i, f)
	}
	if err := os.MkdirAll(filepath.Join(pkg, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{"metais.yaml": manifest, "templates/f.txt.liquid": "new {{ output.path }}"} {
		if err := os.WriteFile(filepath.Join(pkg, filepath.FromSlash(name)), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), cli.Invocation{
		Args:  []string{"render", pkg, "--out", out, "--overwrite", "--json"},
		Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	if status != 0 {
		return fmt.Errorf("metais render --overwrite: status %d: %s", status, stdout.String())
	}
	return nil
}

// oldTime is the access time the detector gives every file it seeds, and
// one second less than their modification time, so that a host that
// updates the access time only of a file not read since it was modified
// (relatime) updates it on the next read.
var oldTime = time.Date(2001, 1, 1, 0, 0, 0, 0, time.UTC)

// seeded is a file the detector put in the output directory before the
// write: at path, holding body, and reached afterwards through probe,
// which for a file the write replaces is a hard link outside the output
// directory, so the replaced file itself can still be examined.
type seeded struct {
	path, body, probe string
	replaced          bool
}

// recordsReads reports whether the file system beneath dir updates a
// file's access time when it is read, which the detector's read half
// relies on. It is probed with a file of its own, never with one the
// detector watches.
func recordsReads(t *testing.T, dir string) bool {
	t.Helper()
	p := filepath.Join(dir, "probe")
	if err := os.WriteFile(p, []byte("probe"), 0o644); err != nil {
		t.Fatal(err)
	}
	defer os.Remove(p)
	if err := os.Chtimes(p, oldTime, oldTime.Add(time.Second)); err != nil {
		t.Fatal(err)
	}
	before, ok := atime(t, p)
	if !ok {
		return false
	}
	if _, err := os.ReadFile(p); err != nil {
		t.Fatal(err)
	}
	after, _ := atime(t, p)
	return after != before
}

func atime(t *testing.T, p string) (int64, bool) {
	t.Helper()
	info, err := os.Lstat(p)
	if err != nil {
		return 0, false
	}
	return accessTime(info)
}

// detectExisting is the existing-destination detector (ADR-009 D8 item 4,
// ADR-005 D15 items 5 to 7 and Amendment A1 item 6, W5-0.8). It seeds an
// output directory with files the write replaces, each hard-linked from
// outside it, and with foreign files the write does not plan, beside them
// and at the top, a hidden one included; sets every seeded file's access
// time behind its modification time; runs the write under faults; and
// returns what it saw go wrong:
//
//   - the write failed;
//   - a foreign file is gone or changed: the write removed or wrote a name
//     it did not create;
//   - a replaced file, through its hard link, holds other bytes: the write
//     wrote into it rather than renaming over it;
//   - a seeded file's access time moved: the write read an existing file
//     (when the file system records reads at all, which reads reports);
//   - a planned file is not written, a planned directory not created, or
//     the staging directory is left.
func detectExisting(t *testing.T, w existingWrite, faults fault.Set) (found []string, reads bool) {
	t.Helper()
	base := testenv.Dir(t)
	out, links := filepath.Join(base, "out"), filepath.Join(base, "links")
	for _, d := range []string{out, links} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	reads = recordsReads(t, base)
	host := func(p string) string { return filepath.Join(out, filepath.FromSlash(p)) }
	var seeds []seeded
	dirs := map[string]bool{".": true}
	for i, f := range w.files {
		if i%2 == 1 {
			continue
		}
		s := seeded{path: f, body: "old " + f, probe: filepath.Join(links, strconv.Itoa(i)), replaced: true}
		seeds = append(seeds, s)
		dirs[path.Dir(f)] = true
	}
	for d := range dirs {
		seeds = append(seeds, seeded{path: path.Join(d, "foreign.txt"), body: "foreign " + d})
	}
	seeds = append(seeds, seeded{path: ".hidden", body: "hidden"})
	for i := range seeds {
		s := &seeds[i]
		p := host(s.path)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(s.body), 0o644); err != nil {
			t.Fatal(err)
		}
		if s.replaced {
			if err := os.Link(p, s.probe); err != nil {
				t.Skipf("cannot make a hard link here: %v", err)
			}
		} else {
			s.probe = p
		}
		if err := os.Chtimes(p, oldTime, oldTime.Add(time.Second)); err != nil {
			t.Fatal(err)
		}
	}
	accessed := map[string]int64{}
	for _, s := range seeds {
		accessed[s.path], _ = atime(t, s.probe)
	}

	if err := w.run(t, out, faults); err != nil {
		found = append(found, fmt.Sprintf("the write failed: %v", err))
	}

	for _, s := range seeds {
		if reads {
			if a, ok := atime(t, s.probe); ok && a != accessed[s.path] {
				found = append(found, "read "+s.path+", which existed")
			}
		}
		b, err := os.ReadFile(s.probe)
		switch {
		case err != nil && !s.replaced:
			found = append(found, "removed "+s.path+", which it did not create")
		case err != nil:
			found = append(found, "lost the hard link to "+s.path)
		case !bytes.Equal(b, []byte(s.body)) && s.replaced:
			found = append(found, "wrote into "+s.path+" rather than replacing it")
		case !bytes.Equal(b, []byte(s.body)):
			found = append(found, "changed "+s.path+", which it did not plan")
		}
	}
	for _, f := range w.files {
		info, err := os.Lstat(host(f))
		if err != nil || !info.Mode().IsRegular() {
			found = append(found, "did not write "+f)
			continue
		}
		if b, _ := os.ReadFile(host(f)); bytes.Equal(b, []byte("old "+f)) {
			found = append(found, "left the old bytes of "+f)
		}
	}
	for _, d := range w.dirs {
		if info, err := os.Lstat(host(d)); err != nil || !info.IsDir() {
			found = append(found, "did not create "+d+"/")
		}
	}
	if _, err := os.Lstat(filepath.Join(out, hostfs.StagingDir)); err == nil {
		found = append(found, "left "+hostfs.StagingDir)
	}
	slices.Sort(found)
	return found, reads
}

// TestExistingDestinationDetector is W5-0.8: every writer of an existing
// output directory reads no existing file there and removes no name it did
// not create. Its controls, in the same run, are the two faults at
// internal/hostfs's seam: with ReadExisting the write reads every file it
// replaces, and with RemoveForeign it removes the files beside those it
// commits; the detector must see each, so a detector gone blind fails here.
// The read half needs a file system that records reads in access times;
// where the probe finds none, that half is skipped and says so.
func TestExistingDestinationDetector(t *testing.T) {
	testenv.Home(t)
	for _, w := range existingWrites {
		t.Run(w.name, func(t *testing.T) {
			found, reads := detectExisting(t, w, 0)
			if len(found) > 0 {
				t.Errorf("the write into an existing directory:\n%s", strings.Join(found, "\n"))
			}
			t.Run("ReadExisting", func(t *testing.T) {
				if !reads {
					t.Skip("this file system records no read in a file's access time")
				}
				found, _ := detectExisting(t, w, fault.ReadExisting)
				if !slices.ContainsFunc(found, func(f string) bool { return strings.HasPrefix(f, "read ") }) {
					t.Errorf("ReadExisting went undetected: %q", found)
				}
			})
			t.Run("RemoveForeign", func(t *testing.T) {
				found, _ := detectExisting(t, w, fault.RemoveForeign)
				if !slices.ContainsFunc(found, func(f string) bool { return strings.HasPrefix(f, "removed ") }) {
					t.Errorf("RemoveForeign went undetected: %q", found)
				}
			})
		})
	}
}
