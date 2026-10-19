// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/testenv"
)

// outputs returns the SHA-256 of every file below dir, by its path
// relative to dir.
func outputs(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		rel, _ := filepath.Rel(dir, p)
		out[filepath.ToSlash(rel)] = hex.EncodeToString(sum[:])
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// baseline is the digest of every output the step's vector expects.
func (st renderStep) baseline(t *testing.T) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, tr := range st.trees {
		for p, d := range outputs(t, filepath.Join(st.dir, filepath.FromSlash(tr.Equals))) {
			out[tr.Path+"/"+p] = d
		}
	}
	return out
}

// rendered is the digest of every output a render of st wrote in work.
func (st renderStep) rendered(t *testing.T, work string) map[string]string {
	t.Helper()
	out := map[string]string{}
	for _, tr := range st.trees {
		for p, d := range outputs(t, filepath.Join(work, filepath.FromSlash(tr.Path))) {
			out[tr.Path+"/"+p] = d
		}
	}
	return out
}

// differs describes how got differs from want, or returns "".
func differs(want, got map[string]string) string {
	for p, d := range want {
		if got[p] != d {
			return fmt.Sprintf("%s is %.12s, want %.12s", p, got[p], d)
		}
	}
	for p := range got {
		if _, ok := want[p]; !ok {
			return p + " is written, and not expected"
		}
	}
	return ""
}

// renderIn renders st in a fresh directory, in-process, under faults, and
// returns the digests of what it wrote.
func renderIn(t *testing.T, st renderStep, faults fault.Set) map[string]string {
	t.Helper()
	work := st.prepare(t, testenv.Dir(t))
	if work == "" {
		t.Fatalf("%s writes, and needs a directory this host cannot make unwritable", st.id)
	}
	if !st.run(t, work, faults) {
		t.Fatalf("%s was refused", st.id)
	}
	return st.rendered(t, work)
}

// TestDeterminismDetector is ADR-010 D13 item 4's determinism guarantee
// (R10 items 2 and 3, ADR-005 D19): every render of the suite that writes
// renders three times in one process, with GOMAXPROCS 1 and with
// runtime.NumCPU(), and once in a fresh process of the built binary under
// a host's worst settings — TZ=Pacific/Chatham, LC_ALL and LANG of
// tr_TR.UTF-8, a umask of 077 and GODEBUG=zipinsecurepath=0,tarinsecurepath=0
// — and every output's SHA-256 is its vector's. Its control, in the same
// run, is the MapOrder fault over the 20-member object of tpl/object-order:
// with object iteration made to follow a Go map, twenty renders must not
// all match, so a detector gone blind fails here.
func TestDeterminismDetector(t *testing.T) {
	testenv.Home(t)
	bin := buildMetais(t)
	var control *renderStep
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(0))
	for _, st := range renderSteps(t) {
		if !st.ok {
			continue
		}
		if st.id == "tpl/object-order/render" {
			control = &st
		}
		want := st.baseline(t)
		for i, procs := range []int{1, runtime.NumCPU(), runtime.NumCPU()} {
			runtime.GOMAXPROCS(procs)
			if d := differs(want, renderIn(t, st, 0)); d != "" {
				t.Errorf("%s, render %d in process with GOMAXPROCS %d: %s", st.id, i+1, procs, d)
			}
		}
		work := st.prepare(t, testenv.Dir(t))
		if err := runHostile(bin, st.argv(work), work); err != nil {
			t.Errorf("%s in a fresh process: %v", st.id, err)
			continue
		}
		if d := differs(want, st.rendered(t, work)); d != "" {
			t.Errorf("%s in a fresh process under TZ, LC_ALL, umask and GODEBUG: %s", st.id, d)
		}
	}
	if control == nil {
		t.Fatal("the suite holds no tpl/object-order")
	}
	want := control.baseline(t)
	for range 20 {
		if differs(want, renderIn(t, *control, fault.MapOrder)) != "" {
			return
		}
	}
	t.Error("with MapOrder, twenty renders of a 20-member object all matched: the detector cannot tell canonical order from a Go map's")
}

// buildMetais builds ./cmd/metais into a temporary directory and returns
// its path.
func buildMetais(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(testenv.Dir(t), "metais")
	cmd := exec.Command("go", "build", "-trimpath", "-o", bin, "./cmd/metais")
	cmd.Dir = moduleRootAbs(t)
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/metais: %v\n%s", err, b)
	}
	return bin
}

// hostileEnviron is the environment of the fresh process: a host whose
// zone, locale and debug settings would change bytes if they reached them.
var hostileEnviron = []string{
	"TZ=Pacific/Chatham", "LC_ALL=tr_TR.UTF-8", "LANG=tr_TR.UTF-8",
	"GODEBUG=zipinsecurepath=0,tarinsecurepath=0",
}

// runHostile runs bin with args in dir, under hostileEnviron and a umask
// of 077, and fails unless it exits 0.
func runHostile(bin string, args []string, dir string) error {
	cmd := umaskCommand(bin, args)
	cmd.Dir = dir
	cmd.Env = append(slices.Clone(hostileEnviron), "HOME="+dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return fmt.Errorf("exited %d: %s", exit.ExitCode(), stdout.Bytes())
	}
	return err
}
