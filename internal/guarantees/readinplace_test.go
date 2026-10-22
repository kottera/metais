// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/cli"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/testenv"
)

// snapshot is a directory tree as the detector compares it: each path
// beneath the root, with its type, size and SHA-256.
func snapshot(t *testing.T, root string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, p)
		info, err := d.Info()
		if err != nil {
			return err
		}
		desc := info.Mode().Type().String()
		if info.Mode().IsRegular() {
			b, err := os.ReadFile(p)
			if err != nil {
				return err
			}
			sum := sha256.Sum256(b)
			desc += " " + strconv.Itoa(len(b)) + " " + hex.EncodeToString(sum[:])
		}
		out[filepath.ToSlash(rel)] = desc
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// changes lists what differs between two snapshots of root.
func changes(name string, before, after map[string]string) []string {
	var out []string
	for p, d := range after {
		if b, ok := before[p]; !ok {
			out = append(out, fmt.Sprintf("%s: %s appeared", name, p))
		} else if b != d {
			out = append(out, fmt.Sprintf("%s: %s changed", name, p))
		}
	}
	for p := range before {
		if _, ok := after[p]; !ok {
			out = append(out, fmt.Sprintf("%s: %s disappeared", name, p))
		}
	}
	slices.Sort(out)
	return out
}

// readingLines are the command lines the detector runs over pkg: ls
// --recursive, read of every entry, outline of every Markdown entry, grep
// over every entry and over .compiled/, schema of the combined schema and
// of every input, and extract, whose destination is out, outside every
// watched directory.
func readingLines(t *testing.T, pkg, out string, faults fault.Set) [][]string {
	t.Helper()
	var listing struct {
		Result struct {
			Entries []struct {
				Path string `json:"path"`
				Type string `json:"type"`
			} `json:"entries"`
		} `json:"result"`
	}
	runReading(t, &listing, faults, "ls", "--json", "-r", pkg)
	var inspect struct {
		Result struct {
			Inputs []struct {
				Name string `json:"name"`
			} `json:"inputs"`
		} `json:"result"`
	}
	runReading(t, &inspect, faults, "inspect", "--json", pkg)
	lines := [][]string{
		{"ls", pkg}, {"ls", "--json", "-r", pkg},
		{"grep", pkg, "e", "-C", "2"}, {"grep", "--json", pkg, "."}, {"grep", "--json", pkg, "{", ".compiled/**"},
		{"schema", pkg}, {"schema", "--json", pkg},
		{"extract", "--json", pkg, "**", "--out", filepath.Join(out, "all")},
		{"extract", pkg, "**", "--out", filepath.Join(out, "dry"), "--dry-run"},
	}
	for _, e := range listing.Result.Entries {
		if e.Type != "file" {
			continue
		}
		var read struct {
			Result struct {
				Binary bool `json:"binary"`
			} `json:"result"`
		}
		runReading(t, &read, faults, "read", "--json", pkg, e.Path)
		lines = append(lines, []string{"read", pkg, e.Path})
		if !read.Result.Binary {
			lines = append(lines, []string{"read", pkg, e.Path, "--lines", "1-", "--number"})
		}
		if strings.HasSuffix(e.Path, ".md") {
			lines = append(lines, []string{"outline", pkg, e.Path}, []string{"outline", "--json", pkg, e.Path})
		}
	}
	for _, in := range inspect.Result.Inputs {
		lines = append(lines, []string{"schema", pkg, in.Name})
	}
	if len(listing.Result.Entries) == 0 || len(inspect.Result.Inputs) == 0 {
		t.Fatalf("%s lists no entry or no input", pkg)
	}
	return lines
}

// runReading runs the tool in-process under faults and returns its exit
// status, decoding its standard output into doc when doc is not nil.
func runReading(t *testing.T, doc any, faults fault.Set, argv ...string) int {
	t.Helper()
	var stdout, stderr bytes.Buffer
	status := cli.Run(context.Background(), cli.Invocation{
		Args: argv, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	})
	if doc != nil {
		if err := json.Unmarshal(stdout.Bytes(), doc); err != nil {
			t.Fatalf("metais %q: %v: %q", argv, err, stdout.String())
		}
	}
	return status
}

// detectReadInPlace is the read-in-place detector (ADR-010 D13 item 4,
// PRD R2 item 1): every reading command runs over a packed package and
// its source directory with TMPDIR, the working directory and HOME at
// directories snapshotted before and after, and it returns what appeared
// or changed in them. faults are handed to every run.
func detectReadInPlace(t *testing.T, vectors string, faults fault.Set) []string {
	t.Helper()
	base := testenv.Dir(t)
	home := testenv.Home(t)
	tmp, work, out := filepath.Join(base, "tmp"), filepath.Join(base, "work"), filepath.Join(base, "out")
	for _, d := range []string{tmp, work, out} {
		if err := os.Mkdir(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	for _, v := range []string{"TMPDIR", "TMP", "TEMP"} {
		t.Setenv(v, tmp)
	}
	t.Chdir(work)
	watched := map[string]string{"TMPDIR": tmp, "the working directory": work, "HOME": home}
	before := map[string]map[string]string{}
	for name, d := range watched {
		before[name] = snapshot(t, d)
	}
	ran := 0
	for i, pkg := range []string{
		filepath.Join(vectors, "pkg", "opens-packed", "package.metais"),
		filepath.Join(vectors, "pkg", "opens-source", "package"),
	} {
		for _, argv := range readingLines(t, pkg, filepath.Join(out, strconv.Itoa(i)), faults) {
			if status := runReading(t, nil, faults, argv...); status != 0 {
				t.Errorf("metais %q exited %d", argv, status)
			}
			ran++
		}
	}
	if ran < 20 {
		t.Fatalf("the detector ran %d command lines; it has gone blind", ran)
	}
	if _, err := os.Stat(filepath.Join(out, "0", "all", "metais.yaml")); err != nil {
		t.Errorf("extract wrote nothing beneath its destination: %v", err)
	}
	var found []string
	for name, d := range watched {
		found = append(found, changes(name, before[name], snapshot(t, d))...)
	}
	slices.Sort(found)
	return found
}

// TestReadInPlaceDetector is ADR-010 D13's read-in-place guarantee (PRD R2
// item 1, W4-2.5): the reading commands list, read, outline, search and
// print schemas of a package, packed or source, without writing anything
// to TMPDIR, the working directory or HOME; extract writes only beneath
// the destination it is given. Its control, in the same run, is
// UnpackToTemp: with browse made to read each entry through a file it
// extracts, the detector must see the file, so a detector gone blind
// fails here.
func TestReadInPlaceDetector(t *testing.T) {
	vectors := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors")
	if found := detectReadInPlace(t, vectors, 0); len(found) > 0 {
		t.Errorf("the reading commands wrote to the host:\n%s", strings.Join(found, "\n"))
	}
	t.Run("control", func(t *testing.T) {
		found := detectReadInPlace(t, vectors, fault.UnpackToTemp)
		if !slices.ContainsFunc(found, func(f string) bool { return strings.HasPrefix(f, "the working directory: .metais-unpack-") }) {
			t.Errorf("UnpackToTemp went undetected: %q", found)
		}
	})
}
