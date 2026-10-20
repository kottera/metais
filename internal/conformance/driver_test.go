// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// refDriverMain is the reference driver (ADR-011 D11 item 7): it serves
// one request of the driver protocol by running the metais-compatible
// command line at tool with the cli adapter's own command lines, in the
// working directory and the environment it was given.
func refDriverMain(tool string) int {
	fail := func(format string, args ...any) int {
		fmt.Fprintf(os.Stderr, "metais-refdriver: "+format+"\n", args...)
		return 1
	}
	var req driverRequest
	dec := json.NewDecoder(os.Stdin)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&req); err != nil {
		return fail("the request: %v", err)
	}
	if req.Protocol != driverProtocol {
		return fail("protocol %q, not %s", req.Protocol, driverProtocol)
	}
	wd, err := os.Getwd()
	if err != nil {
		return fail("%v", err)
	}
	a := &cliAdapter{path: tool}
	if req.Op == "version" {
		p, err := runProc(tool, []string{"version", "--json"}, wd, os.Environ(), nil, defaultWall)
		if err != nil {
			return fail("version: %v", err)
		}
		id, err := readIdentity(p)
		if err != nil {
			return fail("%v", err)
		}
		return writeResponse(map[string]any{"tool": id})
	}
	st := step{Op: req.Op, Args: req.Args}
	ad := json.NewDecoder(bytes.NewReader(req.Args))
	ad.DisallowUnknownFields()
	if err := ad.Decode(&st.args); err != nil {
		return fail("the arguments: %v", err)
	}
	cmds, err := a.commands(&vector{}, &sandbox{work: wd}, &st)
	if err != nil {
		return fail("%v", err)
	}
	c := cmds[0]
	p, err := runProc(tool, c.args, wd, os.Environ(), c.stdin, defaultWall)
	if err != nil {
		return fail("%v", err)
	}
	if crashed(p.stderr) != "" {
		// The whole traceback, so the runner reads the crash from it.
		return fail("%s crashed\n%s", c.args[0], p.stderr)
	}
	if p.exit == 2 || p.exit == 3 {
		return fail("%s exited %d\n%s", c.args[0], p.exit, tail(p.stderr))
	}
	doc := p.stdout
	if c.streaming {
		if err := os.WriteFile(req.StreamFile, p.stdout, 0o644); err != nil {
			return fail("%v", err)
		}
		doc = p.stderr
	}
	env, err := parseEnvelope(doc)
	if err != nil {
		return fail("the tool's document: %v", err)
	}
	var listing any
	if st.args.DryRun {
		listing = env.files
	}
	return writeResponse(map[string]any{"ok": env.ok, "findings": env.findings, "listing": listing})
}

// writeResponse prints the driver's one response.
func writeResponse(resp map[string]any) int {
	b, err := json.Marshal(resp)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	os.Stdout.Write(append(b, '\n'))
	return 0
}

// testBinary is this test binary's path, which the fakes and the
// reference driver re-execute.
func testBinary(t testing.TB) string {
	t.Helper()
	p, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// refDriver makes a directory below base holding the reference driver
// beside tool, under the name the driver runs it by, and beside the
// measuring stub it starts tool through, and returns the driver's path.
func refDriver(t testing.TB, dir, tool string) string {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if tool != "" {
		if err := os.Symlink(tool, filepath.Join(dir, toolName)); err != nil {
			t.Fatal(err)
		}
	}
	if procStub != "" {
		if err := os.Symlink(procStub, filepath.Join(dir, procStubName)); err != nil {
			t.Fatal(err)
		}
	}
	driver := filepath.Join(dir, refDriverName)
	if err := os.Symlink(testBinary(t), driver); err != nil {
		t.Fatal(err)
	}
	return driver
}

// TestDriverProtocol runs the suite through the exec adapter with the
// reference driver wrapping the metais binary built from source, so the
// published protocol cannot rot unrun (ADR-011 D11 item 7). It also runs
// the self-proof's fixture suite through the reference driver wrapping
// the clean fake, which proves every operation the fixtures use crosses
// the protocol and the driver intact.
func TestDriverProtocol(t *testing.T) {
	if testing.Short() {
		t.Skip("the driver protocol runs the whole suite twice over; skipped under -short (ADR-011 D11 item 7)")
	}
	t.Parallel()
	t.Run("suite", func(t *testing.T) {
		suite := defaultSuite()
		driver := refDriver(t, filepath.Join(testenv.Dir(t), "built"), builtTool)
		a, err := newAdapter("exec", driver, suite)
		if err != nil {
			t.Fatal(err)
		}
		r := newRunner(t, suite, a)
		runSuite(t, r)
		// The one operation the real binary answers this wave crosses the
		// protocol intact.
		got, err := r.identity()
		if err != nil {
			t.Fatalf("version through the reference driver: %v", err)
		}
		want, err := (&cliAdapter{path: builtTool}).identity(testenv.Dir(t))
		if err != nil || got != want {
			t.Errorf("version through the reference driver is %+v; through the command line, %+v (%v)", got, want, err)
		}
	})
	t.Run("fixtures", func(t *testing.T) {
		suite := writeFixture(t, testenv.Dir(t))
		fake := fakeProgram(t, testenv.Dir(t), "clean")
		driver := refDriver(t, filepath.Dir(fake), "")
		a, err := newAdapter("exec", driver, suite)
		if err != nil {
			t.Fatal(err)
		}
		r := newRunner(t, suite, a)
		r.short = false
		results := make([][]problem, len(fixtureIDs))
		var wg sync.WaitGroup
		for i, id := range fixtureIDs {
			base := testenv.Dir(t)
			wg.Go(func() { results[i] = runVector(r, suite, id, base) })
		}
		wg.Wait()
		for i, ps := range results {
			for _, p := range ps {
				t.Errorf("%s: %s", fixtureIDs[i], p)
			}
		}
		id, err := r.identity()
		if err != nil || id != fakeTool {
			t.Errorf("the identity through the reference driver is %+v, %v; want %+v", id, err, fakeTool)
		}
	})
}
