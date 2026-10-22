// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/testenv"
)

// packSources returns the source directory of every pack step of the
// suite that packs, as vector-relative paths by "<id>/<case>": every
// source pack has a vector for.
func packSources(t *testing.T) map[string]string {
	t.Helper()
	root := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors")
	files, err := filepath.Glob(filepath.Join(root, "*", "*", "vector.json"))
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		var v struct {
			ID    string `json:"id"`
			Cases []struct {
				Name  string `json:"name"`
				Steps []struct {
					Op   string `json:"op"`
					Args struct {
						Source string `json:"source"`
					} `json:"args"`
					Expect struct {
						OK bool `json:"ok"`
					} `json:"expect"`
				} `json:"steps"`
			} `json:"cases"`
		}
		if err := json.Unmarshal(b, &v); err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		for _, c := range v.Cases {
			if len(c.Steps) == 0 || c.Steps[0].Op != "pack" || !c.Steps[0].Expect.OK {
				continue
			}
			src := filepath.Join(filepath.Dir(f), filepath.FromSlash(c.Steps[0].Args.Source))
			if _, err := os.Stat(src); err == nil {
				out[v.ID+"/"+c.Name] = src
			}
		}
	}
	if len(out) == 0 {
		t.Fatal("the suite holds no pack step that packs")
	}
	return out
}

// packIn packs src in-process under faults into a new file and returns
// its bytes.
func packIn(t *testing.T, src string, faults fault.Set) []byte {
	t.Helper()
	out := filepath.Join(testenv.Dir(t), "p.metais")
	var stdout, stderr bytes.Buffer
	if status := cli.Run(context.Background(), cli.Invocation{
		Args: []string{"pack", "--json", src, "--out", out}, Stdin: strings.NewReader(""), Stdout: &stdout, Stderr: &stderr,
		Options: cli.Options{Faults: faults},
	}); status != 0 {
		t.Fatalf("pack %s exited %d: %s", src, status, stdout.String())
	}
	b, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// copySource copies the vector's source directory src into a new
// directory and returns it.
func copySource(t *testing.T, src string) string {
	t.Helper()
	to := filepath.Join(testenv.Dir(t), "src")
	copyTree(t, src, to)
	return to
}

// disturb gives every file and directory beneath src another modification
// time and other permissions, none of which a package may record (ADR-007
// D7 item 4, ADR-001 D4 item 4).
func disturb(t *testing.T, src string) {
	t.Helper()
	when := time.Date(1999, 12, 31, 23, 59, 59, 0, time.UTC)
	var dirs []string
	err := filepath.WalkDir(src, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			dirs = append(dirs, p)
			return nil
		}
		if err := os.Chmod(p, 0o600); err != nil {
			return err
		}
		return os.Chtimes(p, when, when)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range dirs {
		if err := os.Chtimes(d, when, when); err != nil {
			t.Fatal(err)
		}
	}
}

// TestPackDeterminism is ADR-007 D7 item 4 and ADR-001 D4 item 6 within
// one build, pinned red first (W6-1.6): every source a pack step of the
// suite packs is packed three times in one process, with GOMAXPROCS 1 and
// with runtime.NumCPU(); once from a copy whose files have another
// modification time and mode; and once in a fresh process of the built
// binary under a host's worst settings (TestDeterminismDetector's) — and
// every package is byte for byte the first. Its control, in the same run,
// is fault.PackMapOrder: with the entries put in a Go map's order, twenty
// packs must not all match, so a detector gone blind fails here.
func TestPackDeterminism(t *testing.T) {
	testenv.Home(t)
	bin := buildMetais(t)
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(0))
	var control string
	for id, src := range packSources(t) {
		if id == "aut/pack/into-a-directory" {
			control = src
		}
		runtime.GOMAXPROCS(1)
		want := packIn(t, copySource(t, src), 0)
		for i, procs := range []int{1, runtime.NumCPU(), runtime.NumCPU()} {
			runtime.GOMAXPROCS(procs)
			if got := packIn(t, copySource(t, src), 0); !bytes.Equal(got, want) {
				t.Errorf("%s, pack %d in process with GOMAXPROCS %d: %d bytes differ from the first pack's %d", id, i+1, procs, len(got), len(want))
			}
		}
		disturbed := copySource(t, src)
		disturb(t, disturbed)
		if got := packIn(t, disturbed, 0); !bytes.Equal(got, want) {
			t.Errorf("%s, packed with other modification times and modes: the bytes differ", id)
		}
		work := testenv.Dir(t)
		if _, err := runHostile(bin, []string{"pack", "--json", copySource(t, src), "--out", filepath.Join(work, "p.metais")}, work); err != nil {
			t.Errorf("%s in a fresh process: %v", id, err)
			continue
		}
		if got, err := os.ReadFile(filepath.Join(work, "p.metais")); err != nil || !bytes.Equal(got, want) {
			t.Errorf("%s in a fresh process under TZ, LC_ALL, umask and GODEBUG: the bytes differ (%v)", id, err)
		}
	}
	if control == "" {
		t.Fatal("the suite holds no aut/pack")
	}
	want := packIn(t, copySource(t, control), 0)
	for range 20 {
		if !bytes.Equal(packIn(t, copySource(t, control), fault.PackMapOrder), want) {
			return
		}
	}
	t.Error("with PackMapOrder, twenty packs of aut/pack all matched: the detector cannot tell byte order from a Go map's")
}
