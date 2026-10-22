// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/testenv"
	"metais.org/metais/internal/version"
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

// baseline is the digest of every output the step's vector expects from
// the tool whose version is version: in a file the tree lists in
// substitute, ADR-011 D10 item 3's tokens stand for metais and version.
func (st renderStep) baseline(t *testing.T, version string) map[string]string {
	t.Helper()
	out := map[string]string{}
	if st.stream != "" {
		b, err := os.ReadFile(filepath.Join(st.dir, filepath.FromSlash(st.stream)))
		if err != nil {
			t.Fatal(err)
		}
		out[streamKey] = digestHex(b)
	}
	for _, tr := range st.trees {
		dir := filepath.Join(st.dir, filepath.FromSlash(tr.Equals))
		for p, d := range outputs(t, dir) {
			if slices.Contains(tr.Substitute, p) {
				b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
				if err != nil {
					t.Fatal(err)
				}
				b = bytes.ReplaceAll(b, []byte("@@TOOL_NAME@@"), []byte("metais"))
				sum := sha256.Sum256(bytes.ReplaceAll(b, []byte("@@TOOL_VERSION@@"), []byte(version)))
				d = hex.EncodeToString(sum[:])
			}
			out[tr.Path+"/"+p] = d
		}
	}
	return out
}

// toolVersion is the version the binary bin reports (ADR-008 D9).
func toolVersion(t *testing.T, bin string) string {
	t.Helper()
	b, err := exec.Command(bin, "version", "--json").Output()
	if err != nil {
		t.Fatalf("%s version: %v", bin, err)
	}
	var doc struct {
		Result struct {
			Tool struct {
				Version string `json:"version"`
			} `json:"tool"`
		} `json:"result"`
	}
	if err := json.Unmarshal(b, &doc); err != nil || doc.Result.Tool.Version == "" {
		t.Fatalf("%s version --json names no version: %v", bin, err)
	}
	return doc.Result.Tool.Version
}

// streamKey is the key of a step's stream among its outputs' digests.
const streamKey = "<standard output>"

// rendered is the digest of every output a render of st wrote in work,
// and of stream when the step streams.
func (st renderStep) rendered(t *testing.T, work string, stream []byte) map[string]string {
	t.Helper()
	out := map[string]string{}
	if st.stream != "" {
		out[streamKey] = digestHex(stream)
	}
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
	ok, stream := st.run(t, work, faults)
	if !ok {
		t.Fatalf("%s was refused", st.id)
	}
	return st.rendered(t, work, stream)
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
	binVersion := toolVersion(t, bin)
	var control *renderStep
	defer runtime.GOMAXPROCS(runtime.GOMAXPROCS(0))
	for _, st := range renderSteps(t) {
		if !st.ok {
			continue
		}
		if st.id == "tpl/object-order/render" {
			control = &st
		}
		want := st.baseline(t, version.String())
		for i, procs := range []int{1, runtime.NumCPU(), runtime.NumCPU()} {
			runtime.GOMAXPROCS(procs)
			if d := differs(want, renderIn(t, st, 0)); d != "" {
				t.Errorf("%s, render %d in process with GOMAXPROCS %d: %s", st.id, i+1, procs, d)
			}
		}
		work := st.prepare(t, testenv.Dir(t))
		stream, err := runHostile(bin, st.argv(work), work)
		if err != nil {
			t.Errorf("%s in a fresh process: %v", st.id, err)
			continue
		}
		if d := differs(st.baseline(t, binVersion), st.rendered(t, work, stream)); d != "" {
			t.Errorf("%s in a fresh process under TZ, LC_ALL, umask and GODEBUG: %s", st.id, d)
		}
	}
	if control == nil {
		t.Fatal("the suite holds no tpl/object-order")
	}
	want := control.baseline(t, version.String())
	for range 20 {
		if differs(want, renderIn(t, *control, fault.MapOrder)) != "" {
			return
		}
	}
	t.Error("with MapOrder, twenty renders of a 20-member object all matched: the detector cannot tell canonical order from a Go map's")
}

// buildMetais builds ./cmd/metais into a temporary directory and returns
// its path. The go command takes nothing from the host but its toolchain
// and caches (goCommand), so no module is downloaded, and none is written
// under a temporary directory where RemoveAll could not delete it.
func buildMetais(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(testenv.Dir(t), "metais")
	cmd := goCommand(t, "build", "-trimpath", "-o", bin, "./cmd/metais")
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
// of 077, fails unless it exits 0, and returns its standard output.
func runHostile(bin string, args []string, dir string) ([]byte, error) {
	cmd := umaskCommand(bin, args)
	cmd.Dir = dir
	cmd.Env = append(slices.Clone(hostileEnviron), "HOME="+dir)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	err := cmd.Run()
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		return nil, fmt.Errorf("exited %d: %s%s", exit.ExitCode(), stdout.Bytes(), stderr.Bytes())
	}
	return stdout.Bytes(), err
}

// digestHex is the lowercase hex SHA-256 of b.
func digestHex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}
