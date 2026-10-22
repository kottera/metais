// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/testenv"
)

func TestStepLimits(t *testing.T) {
	for _, c := range []struct {
		name string
		b    *bounds
		race bool
		want limits
	}{
		{"defaults", nil, false, limits{10 * time.Second, 1024 << 20, time.Minute}},
		{"lowered", &bounds{CPUms: 250, RSSMiB: 64}, false, limits{250 * time.Millisecond, 64 << 20, time.Minute}},
		{"never raised", &bounds{CPUms: 20000, RSSMiB: 4096}, false, limits{10 * time.Second, 1024 << 20, time.Minute}},
		{"race-built: CPU times 20, memory times 10, wall as is", &bounds{CPUms: 250, RSSMiB: 64}, true,
			limits{5 * time.Second, 640 << 20, time.Minute}},
		{"race-built defaults", nil, true, limits{200 * time.Second, 10240 << 20, time.Minute}},
	} {
		if got := stepLimits(c.b, c.race, time.Minute); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}

// TestMaxRSSBytes pins ru_maxrss's unit: bytes on darwin, KiB on linux
// (ADR-011 D13 item 2).
func TestMaxRSSBytes(t *testing.T) {
	if got := maxRSSBytes("darwin", 5<<20); got != 5<<20 {
		t.Errorf("darwin: %d", got)
	}
	if got := maxRSSBytes("linux", 5<<10); got != 5<<20 {
		t.Errorf("linux: %d", got)
	}
}

func TestBoundProblems(t *testing.T) {
	lim := limits{cpu: time.Second, rss: 1 << 20, wall: time.Minute}
	if ps := boundProblems(&proc{cpu: time.Second, rss: 1 << 20}, lim); len(ps) > 0 {
		t.Errorf("a process at its bounds: %v", ps)
	}
	if ps := boundProblems(&proc{rss: -1}, lim); len(ps) > 0 {
		t.Errorf("a process whose memory was not measured: %v", ps)
	}
	for _, c := range []struct {
		p     proc
		check string
	}{
		{proc{cpu: time.Second + 1}, checkCPU},
		{proc{rss: 1<<20 + 1}, checkRSS},
		{proc{killed: true}, checkWall},
	} {
		ps := boundProblems(&c.p, lim)
		if len(ps) != 1 || ps[0].check != c.check {
			t.Errorf("%+v: %v, want one %s problem", c.p, ps, c.check)
		}
	}
}

func TestRaceBuilt(t *testing.T) {
	if got := raceBuilt(testBinary(t)); got != raceEnabled {
		t.Errorf("raceBuilt(this test binary) = %t, but it is race-built: %t", got, raceEnabled)
	}
	if got := raceBuilt(builtTool); got != raceEnabled {
		t.Errorf("raceBuilt(the built tool) = %t, want %t: TestMain builds it with -race exactly when this binary is", got, raceEnabled)
	}
}

// TestExitFor pins ADR-008 D13's mapping as the cli adapter checks it.
func TestExitFor(t *testing.T) {
	reg := map[string]codeInfo{
		"internal.error":  {Severity: "error", Class: "internal"},
		"usage.flag":      {Severity: "error", Class: "usage"},
		"package.refused": {Severity: "error", Class: "refused"},
		"input.invalid":   {Severity: "error", Class: "content"},
		"template.broken": {Severity: "error", Class: "content"},
		"output.exists":   {Severity: "error", Class: "destination"},
		"input.unused":    {Severity: "warning", Class: "content"},
	}
	f := func(code, kind string) any {
		return map[string]any{"code": code, "subject": map[string]any{"kind": kind}}
	}
	for _, c := range []struct {
		name      string
		ok        bool
		findings  []any
		authoring bool
		failOn    bool
		want      int
		err       string
	}{
		{"ok", true, []any{f("input.unused", "input")}, false, false, 0, ""},
		{"content", false, []any{f("input.invalid", "input")}, false, false, 1, ""},
		{"destination", false, []any{f("output.exists", "output")}, false, false, 5, ""},
		{"content before destination", false, []any{f("output.exists", "output"), f("input.invalid", "input")}, false, false, 1, ""},
		{"refused before content", false, []any{f("input.invalid", "input"), f("package.refused", "package")}, false, false, 4, ""},
		{"usage before refused", false, []any{f("package.refused", "package"), f("usage.flag", "invocation")}, false, false, 3, ""},
		{"internal first", false, []any{f("usage.flag", "invocation"), f("internal.error", "invocation")}, false, false, 2, ""},
		{"a package's content error under a consumer command", false, []any{f("template.broken", "template")}, false, false, 4, ""},
		{"a package's content error under an authoring command", false, []any{f("template.broken", "template")}, true, false, 1, ""},
		{"a warning explains nothing", false, []any{f("input.unused", "input")}, false, false, 0, "no error finding"},
		{"a warning under --fail-on warning", false, []any{f("input.unused", "input")}, false, true, 1, ""},
		{"an error before a warning under --fail-on warning", false, []any{f("input.unused", "input"), f("output.exists", "output")}, false, true, 5, ""},
		{"an unregistered code", false, []any{f("input.unknown", "input")}, false, false, 0, "not in the registry"},
	} {
		got, err := exitFor(reg, c.ok, c.findings, c.authoring, c.failOn)
		switch {
		case c.err != "" && (err == nil || !strings.Contains(err.Error(), c.err)):
			t.Errorf("%s: %d, %v; want an error saying %q", c.name, got, err, c.err)
		case c.err == "" && (err != nil || got != c.want):
			t.Errorf("%s: %d, %v; want %d", c.name, got, err, c.want)
		}
	}
}

// TestMatches pins ADR-011 D10 item 2's comparison: an expected member
// equals the actual one, recursively; an actual member the expectation
// does not name is free.
func TestMatches(t *testing.T) {
	dec := func(s string) any {
		v, err := decodeJSON([]byte(s))
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	got := dec(`{"code":"a.b","message":"m","schema":{"entry":"e","keyword":"type","keyword_location":"/x"},"params":{"n":1.0},"via":[{"k":1},{"k":2}]}`)
	for want, ok := range map[string]bool{
		`{"code":"a.b"}`: true,
		`{"code":"a.b","schema":{"entry":"e","keyword":"type"}}`: true,
		`{"code":"a.c"}`:                  false,
		`{"params":{"n":1}}`:              false, // numbers compare by their text
		`{"params":{"n":1.0}}`:            true,
		`{"via":[{"k":1}]}`:               false,
		`{"via":[{"k":1},{"k":2}]}`:       true,
		`{"schema":{"pointer":"/p"}}`:     false,
		`{"location":{"entry":"absent"}}`: false,
	} {
		if matches(dec(want), got) != ok {
			t.Errorf("matches(%s) = %t, want %t", want, !ok, ok)
		}
	}
	if d := compareFindings([]any{dec(`{"code":"a.b"}`)}, []any{got, got}); !strings.Contains(d, "expected 1 findings, got 2") {
		t.Errorf("an extra finding: %q", d)
	}
}

// TestScanInflatesToDepthThree shows the scan finds a token only an
// inflation reaches, three zips deep, and no deeper.
func TestScanInflatesToDepthThree(t *testing.T) {
	token := []byte(padding + sentinel + "ENV" + padding)
	nest := func(levels int) []byte {
		b := writeZip([]zipEntry{{name: "deep.txt", method: methodDeflate, content: token}}, false)
		for range levels - 1 {
			b = writeZip([]zipEntry{{name: "inner.zip", method: methodDeflate, content: b}}, false)
		}
		return b
	}
	for levels := 1; levels <= zipDepth+1; levels++ {
		b := nest(levels)
		if bytes.Contains(bytes.ToUpper(b), []byte(sentinel)) {
			t.Fatalf("%d levels: the token is in the raw bytes, so this proves nothing", levels)
		}
		hit := scanBytes("f", b, zipDepth)
		if found := hit != ""; found != (levels <= zipDepth) {
			t.Errorf("%d levels deep: hit %q", levels, hit)
		}
		if levels <= zipDepth && !strings.Contains(hit, "METAIS_SENTINEL") {
			t.Errorf("%d levels deep: the hit %q does not name its source", levels, hit)
		}
	}
}

func TestSentinelSources(t *testing.T) {
	for suffix, src := range map[string]string{
		"HOME": "HOME", "TMP": "TMPDIR", "BIN": "PATH", "PATH-3": "the sandbox path", "CWD": "a working-directory decoy",
		"ENV": "METAIS_SENTINEL", "USER": "USER", "HOST": "HOSTNAME", "NOCOLOR": "NO_COLOR", "TERM": "TERM", "XDG": "an XDG directory",
	} {
		for _, b := range []string{sentinel + suffix, "x " + strings.ToLower(sentinel+suffix) + " y"} {
			if got := sentinelAt([]byte(b)); got != src {
				t.Errorf("sentinelAt(%q) = %q, want %q", b, got, src)
			}
		}
	}
	if got := sentinelAt([]byte("MTSSENTINE")); got != "" {
		t.Errorf("a partial token: %q", got)
	}
	// Bytes that are not UTF-8 keep their length when the scan folds case,
	// so the source is read where the token is.
	if got := sentinelAt(append(bytes.Repeat([]byte{0xff}, 100), sentinel+"HOME"...)); got != "HOME" {
		t.Errorf("a token after invalid UTF-8: %q", got)
	}
	// A planted path carries the sandbox's token first; the more specific
	// source is named.
	if got := sentinelAt([]byte("/t/" + sentinel + "PATH-7/home/" + sentinel + "HOME/x")); got != "HOME" {
		t.Errorf("a leaked HOME: %q", got)
	}
	if got := sentinelAt([]byte("/t/" + sentinel + "PATH-7/work/x")); got != "the sandbox path" {
		t.Errorf("a leaked work path: %q", got)
	}
}

func TestChangesOutside(t *testing.T) {
	f := func(d string) node { return node{kind: "file", digest: d} }
	before := map[string]node{"work": {kind: "dir"}, "work/in.txt": f("1"), "home/x": f("1"), "work/out/old": f("1")}
	after := map[string]node{"work": {kind: "dir"}, "work/in.txt": f("2"), "tmp/new": f("1"),
		"work/out/old": f("2"), "work/out/new": f("1"), "work/outside": f("1"), "work/.stream": f("1")}
	got := changesOutside(before, after, []string{"out"}, []string{".stream"})
	want := []string{"home/x was removed", "tmp/new was created", "work/in.txt was changed", "work/outside was created"}
	if !slices.Equal(got, want) {
		t.Errorf("changes %q, want %q", got, want)
	}
}

// TestEmptySuiteRunsNone is W2-2.4: with no vector, the runner reports
// that it ran none, and passes.
func TestEmptySuiteRunsNone(t *testing.T) {
	root := testenv.Dir(t)
	suite := writeFixture(t, root)
	if err := os.RemoveAll(filepath.Join(suite, "vectors")); err != nil {
		t.Fatal(err)
	}
	r := fixtureRunner(t, suite, "cli", "clean")
	var stats *suiteStats
	passed := t.Run("suite", func(t *testing.T) { stats = runSuite(t, r) })
	if !passed || stats.summary() != "ran no vector (0 not selected)" {
		t.Errorf("an empty suite: passed %t, reported %q", passed, stats.summary())
	}
}

// TestSelection pins which vectors a run selects (ADR-011 D11 item 3).
func TestSelection(t *testing.T) {
	fresh := func() (*runner, *vector) {
		return &runner{version: "1.0", classes: []string{"processor"}},
			&vector{Class: "processor", Since: "1.0", Tags: []string{"security"}}
	}
	if r, v := fresh(); r.selected(v) != "" {
		t.Errorf("a selected vector: %q", r.selected(v))
	}
	for name, edit := range map[string]func(*runner, *vector){
		"class":     func(_ *runner, v *vector) { v.Class = "authoring" },
		"since":     func(_ *runner, v *vector) { v.Since = "1.1" },
		"tags":      func(r *runner, _ *vector) { r.tags = []string{"limits"} },
		"skip-tags": func(r *runner, _ *vector) { r.skipTags = []string{"security"} },
	} {
		r, v := fresh()
		edit(r, v)
		if why := r.selected(v); why == "" {
			t.Errorf("%s: selected", name)
		}
	}
}

// TestZipFixedFields shows the container check reads ADR-001 D1's fields
// archive/zip passes over: a clean container passes, and bytes between
// the entries and the central directory, internal attributes and a
// comment are each caught.
func TestZipFixedFields(t *testing.T) {
	clean := writeZip([]zipEntry{
		{name: "a.txt", method: methodDeflate, content: []byte(padding)},
		{name: "mimetype", method: methodStore, content: []byte("x")},
	}, false)
	if bad := zipFixedFields(clean); len(bad) > 0 {
		t.Fatalf("a clean container: %q", bad)
	}
	le := binary.LittleEndian
	cd := int(le.Uint32(clean[len(clean)-6:]))
	for name, c := range map[string]struct {
		edit func([]byte) []byte
		want string
	}{
		"bytes before the directory": {func(b []byte) []byte {
			out := append(slices.Clone(b[:cd]), 0)
			out = append(out, b[cd:]...)
			le.PutUint32(out[len(out)-6:], uint32(cd+1))
			return out
		}, "the central directory starts at"},
		"internal attributes": {func(b []byte) []byte { b[cd+36] = 1; return b }, "internal attributes"},
		"an archive comment": {func(b []byte) []byte {
			b = append(b, 'c')
			le.PutUint16(b[len(b)-3:], 1)
			return b
		}, "comment"},
	} {
		bad := zipFixedFields(c.edit(slices.Clone(clean)))
		if !slices.ContainsFunc(bad, func(s string) bool { return strings.Contains(s, c.want) }) {
			t.Errorf("%s: reported %q", name, bad)
		}
	}
}

// TestCrashed holds crashed to what the Go runtime writes when a process
// exits through a panic or a fatal error (ADR-009 D18 item 2), and to
// nothing else: a recovered panic's internal error, and text that merely
// begins like a traceback, are not crashes.
func TestCrashed(t *testing.T) {
	for _, c := range []struct {
		name, stderr, want string
	}{
		{"a panic", "panic: boom\n\ngoroutine 1 [running]:\nmain.main()\n\t/x.go:3 +0x1\nexit status 2\n", "panic: boom"},
		{"a panic with the runtime's goroutine detail",
			"panic: boom [recovered]\n\tpanic: again\n\ngoroutine 7 gp=0x1400 m=0 mp=0x1028 [running]:\nmain.f()\n", "panic: boom [recovered]"},
		{"a fatal error", "runtime: goroutine stack exceeds 65536-byte limit\nfatal error: stack overflow\n\nruntime stack:\nruntime.throw()\n\ngoroutine 1 gp=0x14 m=0 mp=0x10 [running]:\n", "fatal error: stack overflow"},
		{"an internal error the tool reported", "error internal.panic: the command stopped on an internal error\n", ""},
		{"a message that begins like a panic", "panic: is a word a package may print\n", ""},
		{"a goroutine line alone", "goroutine 1 [running]:\n", ""},
		{"nothing", "", ""},
	} {
		if got := crashed([]byte(c.stderr)); got != c.want {
			t.Errorf("%s: crashed is %q, want %q", c.name, got, c.want)
		}
	}
}

// TestRefDriverForwardsCrash runs the reference driver over a fake that
// panics: the driver's own exit says only that it failed, and the runner
// must still read the crash from the traceback it forwards.
func TestRefDriverForwardsCrash(t *testing.T) {
	t.Parallel()
	suite := writeFixture(t, testenv.Dir(t))
	fake := fakeProgram(t, testenv.Dir(t), "panic")
	a, err := newAdapter("exec", refDriver(t, filepath.Dir(fake), ""), suite)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, suite, a)
	r.short = true
	ps := runVector(r, suite, fixRenderTree, testenv.Dir(t))
	if !slices.ContainsFunc(ps, func(p problem) bool { return p.check == checkCrash }) {
		t.Errorf("the reference driver over a fake that panics gave no crash; the runner reported %q", ps)
	}
}

// TestFactAt holds check 2's matching to ADR-009 D16 item 1: each kind of
// host fact is found in a stream, whatever its case, and a word or a path
// that merely begins or ends with one is not.
func TestFactAt(t *testing.T) {
	facts := addPath(addPath([]hostFact{
		{what: "the host name", value: "buildhost"},
		{what: "the user name", value: "edwardf"},
	}, "the real home directory", "/home/edwardf"), "the test's temporary root", "/tmp/TestX123/001")
	for _, c := range []struct{ in, want string }{
		{"rendered on buildhost.", "the host name"},
		{"rendered on BuildHost", "the host name"},
		{"rendered on buildhosts", ""},
		{"by edwardf", "the user name"},
		{`{"path":"/home/edwardf/x.json"}`, "the user name"},
		{"file:///home/edwardf", "the user name"},
		{"/tmp/TestX123/001/MTSSENTINELPATH-1/work", "the test's temporary root"},
		{"/tmp/TestX123/0011", ""},
		{"/tmp/TestX123/001.bak", ""},
		{"an edwardfs value", ""},
		{"", ""},
	} {
		if got := factAt([]byte(c.in), facts); got != c.want {
			t.Errorf("factAt(%q) = %q, want %q", c.in, got, c.want)
		}
	}
	paths := addPath(nil, "the checkout", "/src/metais")
	if got := factAt([]byte("in /src/metais/internal"), paths); got != "the checkout" {
		t.Errorf("a path below the checkout: %q", got)
	}
	if got := factAt([]byte("in /src/metaisx"), paths); got != "" {
		t.Errorf("a path the checkout's begins: %q", got)
	}
	if addPath(nil, "the root", "/") != nil {
		t.Error("the root is a fact")
	}
}

// TestHostFactsSkipVocabulary pins newHostFacts's one exclusion: a host
// or user name that is a word of the tool's own text, or no longer than
// minNameLength, is not looked for, since a scan cannot tell it from that
// text; every other is.
func TestHostFactsSkipVocabulary(t *testing.T) {
	h := newHostFacts()
	if len(h.paths) == 0 {
		t.Fatal("no path fact: the checkout is always one")
	}
	for _, f := range h.names {
		hv := newHostFacts([]byte("The tool's text names " + f.value + " itself."))
		if slices.ContainsFunc(hv.names, func(g hostFact) bool { return g.value == f.value }) {
			t.Errorf("%s %q is looked for although the tool's text holds it", f.what, f.value)
		}
		if len(f.value) <= minNameLength {
			t.Errorf("%s %q is looked for, and is no longer than %d", f.what, f.value, minNameLength)
		}
	}
}

// TestCappedHoldsThroughCopy pins the cap on a process's stream as
// os/exec fills it, by io.Copy from a pipe: the copy reaches the cap's
// own Write, never the embedded buffer's ReadFrom, which takes everything.
func TestCappedHoldsThroughCopy(t *testing.T) {
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		w.Write(bytes.Repeat([]byte("x"), 100))
		w.Close()
	}()
	c := &capped{limit: 10}
	if _, err := io.Copy(c, r); err != nil {
		t.Fatal(err)
	}
	r.Close()
	if !c.over || c.Len() != 10 {
		t.Errorf("a stream of 100 bytes under a cap of 10: over %t, kept %d bytes", c.over, c.Len())
	}
}
