// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package hostfs_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/hostenv"
	"metais.org/metais/internal/testenv"
)

// The variables that make this test binary the tool: the arguments, as a
// JSON array, and the fault set, so that the presentation test's control
// runs a build that breaks the guarantee. The tool itself takes no fault
// from its environment; only this test binary does.
const (
	toolArgs   = "METAIS_TEST_TOOL_ARGS"
	toolFaults = "METAIS_TEST_TOOL_FAULTS"
)

// TestMain runs the tests, or, given toolArgs, is the tool: main's own
// Isolate and dispatch, with the faults toolFaults names.
func TestMain(m *testing.M) {
	a, ok := os.LookupEnv(toolArgs)
	if !ok {
		os.Exit(m.Run())
	}
	var args []string
	if err := json.Unmarshal([]byte(a), &args); err != nil {
		panic(err)
	}
	faults, _ := strconv.ParseUint(os.Getenv(toolFaults), 10, 64)
	d := hostenv.Isolate()
	os.Exit(cli.Run(context.Background(), cli.Invocation{
		Args: args, Stdin: os.Stdin, Stdout: os.Stdout, Stderr: os.Stderr,
		Options: cli.Options{Display: d, Faults: fault.Set(faults)},
	}))
}

// The sentinels a run under a terminal is given as NO_COLOR and TERM
// (ADR-008 D16 item 6). TERM's proves it is not read.
const (
	sentinelNoColor = "metais-sentinel-no-color-5d1c"
	sentinelTerm    = "metais-sentinel-term-a07e"
)

// requirePTY is the probe of a pseudo-terminal device: it opens one, or
// skips the test naming what is missing.
func requirePTY(t *testing.T) (master, slave *os.File) {
	t.Helper()
	m, s, err := openPTY()
	if err != nil {
		t.Skipf("requires a pseudo-terminal: %v", err)
	}
	t.Cleanup(func() { m.Close(); s.Close() })
	return m, s
}

// tool is how a run starts the tool: its executable, and the environment
// beside HOME, NO_COLOR and TERM.
type tool struct {
	path string
	env  []string
}

// presented is what one run gave: its exit status, both streams, and the
// files it wrote, by path.
type presented struct {
	status         int
	stdout, stderr []byte
	files          map[string][]byte
}

// present runs args with tl in dir, under a pseudo-terminal on standard
// input, output and error with NO_COLOR and TERM set to the sentinels
// when terminal is true, and otherwise piped with neither set. The
// terminal is raw, so that it hands on the bytes the tool wrote.
func present(t *testing.T, tl tool, dir string, args []string, terminal bool) presented {
	t.Helper()
	b, _ := json.Marshal(args)
	// The built tool reads its arguments; this test binary as the tool
	// reads toolArgs, and ignores them.
	cmd := exec.Command(tl.path, args...)
	cmd.Dir = dir
	cmd.Env = append(slicesClone(tl.env), "HOME="+dir, toolArgs+"="+string(b))
	var p presented
	var reads []chan []byte
	if terminal {
		cmd.Env = append(cmd.Env, "NO_COLOR="+sentinelNoColor, "TERM="+sentinelTerm)
		out, outSlave := requirePTY(t)
		errm, errSlave := requirePTY(t)
		for _, s := range []*os.File{outSlave, errSlave} {
			if err := rawOutput(s); err != nil {
				t.Fatalf("a raw terminal: %v", err)
			}
		}
		cmd.Stdin, cmd.Stdout, cmd.Stderr = outSlave, outSlave, errSlave
		for _, m := range []*os.File{out, errm} {
			c := make(chan []byte, 1)
			go func() {
				var buf bytes.Buffer
				// A master reads until every slave is closed: EOF on
				// darwin, EIO on linux.
				_, _ = io.Copy(&buf, m)
				c <- buf.Bytes()
			}()
			reads = append(reads, c)
		}
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		outSlave.Close()
		errSlave.Close()
	}
	var stdout, stderr bytes.Buffer
	if !terminal {
		cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader(nil), &stdout, &stderr
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
	}
	err := cmd.Wait()
	var exit *exec.ExitError
	switch {
	case errors.As(err, &exit):
		p.status = exit.ExitCode()
	case err != nil:
		t.Fatal(err)
	}
	p.stdout, p.stderr = stdout.Bytes(), stderr.Bytes()
	if terminal {
		p.stdout, p.stderr = <-reads[0], <-reads[1]
	}
	p.files = map[string][]byte{}
	out := filepath.Join(dir, "out")
	err = filepath.WalkDir(out, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(out, path)
		p.files[filepath.ToSlash(rel)], err = os.ReadFile(path)
		return err
	})
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}
	if err := os.RemoveAll(out); err != nil {
		t.Fatal(err)
	}
	return p
}

func slicesClone(s []string) []string { return append([]string(nil), s...) }

// presentationLines are the lines the proof runs: a render that writes
// outputs, two reports full of findings and their messages, a document
// under --color always, and a refusal.
func presentationLines(pkg string) [][]string {
	return [][]string{
		{"render", pkg, "--out", "out", "--json"},
		{"test", pkg, "--json"},
		{"inspect", pkg, "--json"},
		{"--color", "always", "version", "--json"},
		{"version", "--bogus", "--json"},
	}
}

// presentationProblems runs every line with tl under a terminal and piped
// (ADR-008 D16 item 6) and returns every way the two runs differ, in
// status, standard output, standard error or a file written, and every
// sentinel in any of them.
func presentationProblems(t *testing.T, tl tool, pkg string) []string {
	t.Helper()
	var problems []string
	for _, args := range presentationLines(pkg) {
		dir := testenv.Dir(t)
		piped := present(t, tl, dir, args, false)
		tty := present(t, tl, dir, args, true)
		if piped.status != tty.status {
			problems = append(problems, strconv.Quote(args[0])+": the exit status differs")
		}
		if !bytes.Equal(piped.stdout, tty.stdout) {
			problems = append(problems, strconv.Quote(args[0])+": the document differs:\n"+string(piped.stdout)+"\n"+string(tty.stdout))
		}
		if !bytes.Equal(piped.stderr, tty.stderr) {
			problems = append(problems, strconv.Quote(args[0])+": standard error differs")
		}
		if len(piped.files) != len(tty.files) {
			problems = append(problems, strconv.Quote(args[0])+": the files written differ")
		}
		for name, b := range piped.files {
			if !bytes.Equal(b, tty.files[name]) {
				problems = append(problems, strconv.Quote(args[0])+": "+name+" differs")
			}
		}
		for _, p := range []presented{piped, tty} {
			all := [][]byte{p.stdout, p.stderr}
			for _, b := range p.files {
				all = append(all, b)
			}
			for _, b := range all {
				if bytes.Contains(b, []byte(sentinelNoColor)) || bytes.Contains(b, []byte(sentinelTerm)) {
					problems = append(problems, strconv.Quote(args[0])+": a sentinel is in what the tool wrote")
				}
			}
		}
		if args[0] == "render" && len(piped.files) == 0 {
			t.Fatalf("the render wrote nothing: %s", piped.stdout)
		}
	}
	return problems
}

// TestPresentationUnderTerminal is ADR-008 D16 item 6: a run of the built
// tool under a pseudo-terminal, with NO_COLOR and TERM set to sentinels,
// and the same run piped with neither set, write the same JSON documents
// and the same outputs, and no sentinel appears in any output, finding or
// message. Its control is this test binary as the tool, which passes
// without a fault and is caught with fault.ColourJSON, a build that
// colours a JSON document written to a terminal.
//
// It is internal/hostfs's because it opens the pseudo-terminal through
// golang.org/x/sys, which hostfs alone may import (ADR-010 Amendment A1
// item 4); it is an external test, so that it may import internal/cli.
func TestPresentationUnderTerminal(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	requirePTY(t)
	pkg, err := filepath.Abs("../../spec/conformance/vectors/aut/test-containers/package")
	if err != nil {
		t.Fatal(err)
	}
	bin := filepath.Join(testenv.Dir(t), "metais")
	if out, err := goCommand(t, "build", "-trimpath", "-o", bin, "../../cmd/metais").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, p := range presentationProblems(t, tool{path: bin}, pkg) {
		t.Errorf("the built tool: %s", p)
	}
	self, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range presentationProblems(t, tool{path: self}, pkg) {
		t.Errorf("the control without a fault: %s", p)
	}
	faulted := tool{path: self, env: []string{toolFaults + "=" + strconv.FormatUint(uint64(fault.ColourJSON), 10)}}
	if len(presentationProblems(t, faulted, pkg)) == 0 {
		t.Error("a build that colours JSON on a terminal is not caught")
	}
}
