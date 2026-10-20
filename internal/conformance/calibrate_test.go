// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// The limit-calibration suite of ADR-009 D3 item 3 and D18 item 4: the
// worst case of each work and output limit of D3, and Amendment A1's joint
// worst case, each run at the default as a process of the built binary and
// judged against D18 item 2's ceilings. It runs only when
// -calibration.record names the record to write (docs/dev/calibration.md);
// TestCalibrationLowered runs every case at lowered limits in every go
// test, so a case that stops reaching its limit fails there.
var (
	flagCalRecord = flag.String("calibration.record", "", "run the limit-calibration suite at the defaults and write its record to this file")
	flagCalCases  = flag.String("calibration.cases", "", "run only these calibration cases, comma-separated (default: all)")
)

// The ceilings of ADR-009 D18 item 2.
const (
	calCPU         = 10 * time.Second
	calMetadataCPU = 500 * time.Millisecond
	calRSS         = 1 << 30
)

// calWall is the wall clock a calibration case runs under. It is not a
// ceiling: it ends a case that would otherwise never end, and a case it
// kills fails the suite as a broken CPU ceiling would.
const calWall = 30 * time.Minute

// calCase is one worst case. Its build writes the case for the limits in
// effect, so that at lowered limits it is the same case scaled down, and
// returns the command's arguments. Its reached reads the command's result
// and says how it fell short of reaching the case's limit; its beyond,
// for a case that is accepted at its limit, does the same for the run at
// one less, which must be refused by it.
type calCase struct {
	name  string // the case
	limit string // the limit it reaches
	// lowered are the limits the lowered form lowers, each to a value at
	// which the case is the default case scaled down; the case's own
	// limit is always among them.
	lowered map[string]int64
	// metadata marks a case refused from metadata (ADR-009 D18 item 2),
	// held to calMetadataCPU.
	metadata bool
	// gap, when set, says why the tool does not reach the limit: the case
	// is measured at the default, and the lowered form pins that it is not
	// reached, so the gap's closing turns it red and is noticed.
	gap     string
	build   func(dir string, lim calLimits) ([]string, error)
	reached func(r *calResult, lim calLimits) error
	beyond  func(r *calResult, lim calLimits) error
}

// calLimits are the limits a case runs under, by name.
type calLimits map[string]int64

// calResult is what a run of a case reported: its JSON document's ok,
// findings and summary, and from its result only what a case reads, so a
// result of gigabytes is never held whole.
type calResult struct {
	OK       bool         `json:"ok"`
	Findings []calFinding `json:"findings"`
	Summary  struct {
		Listed  int64 `json:"listed"`
		Omitted int64 `json:"omitted"`
	} `json:"summary"`
	matches   int64 // grep's matches
	truncated bool  // grep's truncated
	context   int   // the most lines of context before and after one match
	files     int64 // files beneath the run's output directory
}

// calFinding is a finding as a case reads it.
type calFinding struct {
	Code     string `json:"code"`
	Location struct {
		File    string `json:"file"`
		Pointer string `json:"pointer"`
	} `json:"location"`
	Params map[string]any `json:"params"`
}

// refusedBy reports how r falls short of a refusal by limit at bound,
// placed in file when file is not "".
func (r *calResult) refusedBy(limit string, bound int64, file string) error {
	if r.OK {
		return fmt.Errorf("the command succeeded; it was not refused by %s", limit)
	}
	var codes []string
	for _, f := range r.Findings {
		if f.Params["limit"] == limit && f.Params["bound"] == float64(bound) && (file == "" || f.Location.File == file) {
			return nil
		}
		codes = append(codes, fmt.Sprintf("%s %v %v at %s", f.Code, f.Params["limit"], f.Params["bound"], f.Location.File))
	}
	at := ""
	if file != "" {
		at = " at " + file
	}
	return fmt.Errorf("no finding names %s with its bound %d%s; the findings: %s", limit, bound, at, strings.Join(codes, "; "))
}

// succeeded reports how r falls short of success.
func (r *calResult) succeeded() error {
	if r.OK {
		return nil
	}
	var codes []string
	for _, f := range r.Findings {
		codes = append(codes, fmt.Sprintf("%s %v at %s%s", f.Code, f.Params, f.Location.File, f.Location.Pointer))
	}
	return fmt.Errorf("the command failed: %s", strings.Join(codes, "; "))
}

// calRun is one measured run of a case.
type calRun struct {
	res  *calResult
	exit int
	cpu  time.Duration
	rss  int64 // bytes; -1 when not measured
	wall time.Duration
	// killed is set when the wall clock ended the case.
	killed bool
}

// runCal runs tool with args in dir, in an environment of its own, with
// its standard output written to a file rather than held, and measures it
// as the conformance runner measures a step: user plus system CPU time and
// peak resident memory, from the process state (ADR-009 D18 item 1).
func runCal(tool string, args []string, dir string, wall time.Duration) (*calRun, error) {
	home, tmp, bin := filepath.Join(dir, "home"), filepath.Join(dir, "tmp"), filepath.Join(dir, "bin")
	for _, d := range []string{home, tmp, bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return nil, err
		}
	}
	stdout, err := os.Create(filepath.Join(dir, "stdout.json"))
	if err != nil {
		return nil, err
	}
	defer stdout.Close()
	stderr := &capped{}
	ctx, cancel := context.WithTimeout(context.Background(), wall)
	defer cancel()
	cmd, usage, err := measured(ctx, tool, args)
	if err != nil {
		return nil, err
	}
	cmd.Dir = filepath.Join(dir, "work")
	cmd.Env = []string{"HOME=" + home, "TMPDIR=" + tmp, "PATH=" + bin, "TZ=UTC", "LC_ALL=C"}
	cmd.Stdout, cmd.Stderr = stdout, stderr
	cmd.WaitDelay = time.Second
	start := time.Now()
	err = cmd.Run()
	run := &calRun{wall: time.Since(start), rss: -1, killed: ctx.Err() != nil}
	if cmd.ProcessState == nil {
		usage(nil)
		return nil, fmt.Errorf("start %s: %w", tool, err)
	}
	u, uerr := usage(cmd.ProcessState)
	if uerr != nil && !run.killed {
		return nil, fmt.Errorf("run %s: %w", tool, uerr)
	}
	run.exit, run.cpu = u.exit, u.cpu
	if u.rssOK {
		run.rss = maxRSSBytes(runtime.GOOS, u.rss)
	}
	if run.killed {
		return run, nil
	}
	if e := stderr.Bytes(); bytes.HasPrefix(e, []byte("panic: ")) || bytes.Contains(e, []byte("\npanic: ")) || bytes.Contains(e, []byte("fatal error: ")) {
		return run, fmt.Errorf("the command exited %d through a panic or a runtime fatal error (ADR-009 D18 item 2):\n%s", run.exit, stderr.Bytes())
	}
	if _, err := stdout.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	if run.res, err = readCalResult(stdout); err != nil {
		return run, fmt.Errorf("the command exited %d, and its JSON: %v; standard error: %s", run.exit, err, stderr.Bytes())
	}
	run.res.files, err = countFiles(filepath.Join(dir, "work", "out"))
	return run, err
}

// readCalResult reads a command's JSON document from r a token at a time,
// keeping ok, the findings and the summary, and from the result only grep's
// match count, truncation and context.
func readCalResult(r io.Reader) (*calResult, error) {
	dec := json.NewDecoder(r)
	res := &calResult{}
	if err := expectDelim(dec, '{'); err != nil {
		return nil, err
	}
	for dec.More() {
		tok, err := dec.Token()
		if err != nil {
			return nil, err
		}
		switch tok {
		case "ok":
			err = dec.Decode(&res.OK)
		case "findings":
			err = dec.Decode(&res.Findings)
		case "summary":
			err = dec.Decode(&res.Summary)
		case "result":
			err = readCalGrep(dec, res)
		default:
			var skip json.RawMessage
			err = dec.Decode(&skip)
		}
		if err != nil {
			return nil, fmt.Errorf("%v: %w", tok, err)
		}
	}
	return res, expectDelim(dec, '}')
}

// readCalGrep reads a command's result, keeping what grep's says of its
// matches and skipping everything else.
func readCalGrep(dec *json.Decoder, res *calResult) error {
	tok, err := dec.Token()
	if err != nil || tok != json.Delim('{') {
		// null, or a result that is not an object: nothing a case reads.
		if err == nil && (tok == json.Delim('[')) {
			return skipRest(dec, 1)
		}
		return err
	}
	for dec.More() {
		key, err := dec.Token()
		if err != nil {
			return err
		}
		switch key {
		case "truncated":
			err = dec.Decode(&res.truncated)
		case "matches":
			if err = expectDelim(dec, '['); err != nil {
				return err
			}
			for dec.More() {
				var m struct{ Before, After []json.RawMessage }
				if err := dec.Decode(&m); err != nil {
					return err
				}
				res.matches++
				res.context = max(res.context, min(len(m.Before), len(m.After)))
			}
			err = expectDelim(dec, ']')
		default:
			var skip json.RawMessage
			err = dec.Decode(&skip)
		}
		if err != nil {
			return err
		}
	}
	return expectDelim(dec, '}')
}

// skipRest skips the tokens that close depth open arrays or objects.
func skipRest(dec *json.Decoder, depth int) error {
	for depth > 0 {
		tok, err := dec.Token()
		if err != nil {
			return err
		}
		switch tok {
		case json.Delim('{'), json.Delim('['):
			depth++
		case json.Delim('}'), json.Delim(']'):
			depth--
		}
	}
	return nil
}

// expectDelim reads the delimiter d from dec.
func expectDelim(dec *json.Decoder, d json.Delim) error {
	tok, err := dec.Token()
	if err != nil {
		return err
	}
	if tok != d {
		return fmt.Errorf("%v where %v was expected", tok, d)
	}
	return nil
}

// countFiles counts the regular files beneath dir, 0 when there is no dir.
func countFiles(dir string) (int64, error) {
	var n int64
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			n++
		}
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	return n, err
}

// toolDefaults reads every limit's default from tool's metais.limits/1
// document, so the suite runs at the tool's own defaults.
func toolDefaults(tool, dir string) (calLimits, error) {
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		return nil, err
	}
	p, err := runProc(tool, []string{"limits", "--json"}, filepath.Join(dir, "work"), nil, nil, defaultWall)
	if err != nil {
		return nil, err
	}
	var doc struct {
		Result struct {
			Limits []struct {
				Name    string `json:"name"`
				Default int64  `json:"default"`
			} `json:"limits"`
		} `json:"result"`
	}
	if err := json.Unmarshal(p.stdout, &doc); err != nil {
		return nil, fmt.Errorf("metais limits --json: %w", err)
	}
	lim := calLimits{}
	for _, l := range doc.Result.Limits {
		lim[l.Name] = l.Default
	}
	return lim, nil
}

// limitArgs are the --limit flags that lower defaults to lim, in name
// order.
func limitArgs(defaults, lim calLimits) []string {
	var out []string
	for _, name := range slices.Sorted(maps.Keys(lim)) {
		if lim[name] != defaults[name] {
			out = append(out, "--limit", name+"="+strconv.FormatInt(lim[name], 10))
		}
	}
	return out
}

// with is lim with over applied.
func (lim calLimits) with(over calLimits) calLimits {
	out := calLimits{}
	for k, v := range lim {
		out[k] = v
	}
	for k, v := range over {
		out[k] = v
	}
	return out
}

// runCase builds c beneath dir for lim and runs it.
func runCase(tool string, c calCase, defaults, lim calLimits, dir string, wall time.Duration) (*calRun, error) {
	if err := os.MkdirAll(filepath.Join(dir, "work"), 0o755); err != nil {
		return nil, err
	}
	args, err := c.build(filepath.Join(dir, "work"), lim)
	if err != nil {
		return nil, fmt.Errorf("building the case: %w", err)
	}
	return runCal(tool, append(args, limitArgs(defaults, lim)...), dir, wall)
}

// calTool is the program the suite runs: -conformance.tool, or the binary
// TestMain built.
func calTool(t *testing.T) string {
	t.Helper()
	tool := *flagTool
	if tool == "" {
		tool = builtTool
	}
	if tool == "" {
		t.Fatal("no tool: TestMain built none and -conformance.tool names none")
	}
	return tool
}

// TestCalibrationLowered runs every calibration case at its lowered
// limits, and proves it reaches its limit there: refused by it, or
// accepted at it and refused at one less. A case with a gap is pinned
// short of its limit instead. The lowered limits keep each case the shape
// it has at the default, so this is the default case scaled down.
func TestCalibrationLowered(t *testing.T) {
	tool := calTool(t)
	defaults, err := toolDefaults(tool, testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range calCases {
		t.Run(c.name, func(t *testing.T) {
			if _, ok := c.lowered[c.limit]; !ok {
				t.Fatalf("the case does not lower its own limit, %s", c.limit)
			}
			for name, v := range c.lowered {
				if d, ok := defaults[name]; !ok || v >= d {
					t.Fatalf("%s=%d is not a lowered limit of the tool (default %d)", name, v, d)
				}
			}
			lim := defaults.with(c.lowered)
			run, err := runCase(tool, c, defaults, lim, testenv.Dir(t), defaultWall)
			if err != nil {
				t.Fatal(err)
			}
			if run.killed {
				t.Fatalf("the case ran past %v", defaultWall)
			}
			reach := c.reached(run.res, lim)
			if c.gap != "" {
				if reach == nil {
					t.Fatalf("the case reaches %s, which its gap says the tool does not: remove the gap (%s)", c.limit, c.gap)
				}
				t.Logf("not reached, as its gap says (%s): %v", c.gap, reach)
				return
			}
			if reach != nil {
				t.Fatalf("the case does not reach %s=%d: %v", c.limit, lim[c.limit], reach)
			}
			if c.beyond == nil {
				return
			}
			less := lim.with(calLimits{c.limit: lim[c.limit] - 1})
			run, err = runCase(tool, c.atLimit(lim), defaults, less, testenv.Dir(t), defaultWall)
			if err != nil {
				t.Fatal(err)
			}
			if err := c.beyond(run.res, less); err != nil {
				t.Fatalf("one past %s=%d: %v", c.limit, less[c.limit], err)
			}
		})
	}
}

// TestCalibrationJointLocations proves that Amendment A1's joint cases
// hold their leaves' locations at schema-location-length as well as their
// count at schema-subschemas: one byte less refuses the schema file.
func TestCalibrationJointLocations(t *testing.T) {
	tool := calTool(t)
	defaults, err := toolDefaults(tool, testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	var n int
	for _, c := range calCases {
		if !strings.HasPrefix(c.name, "schema-joint-") {
			continue
		}
		n++
		t.Run(c.name, func(t *testing.T) {
			lim := defaults.with(c.lowered)
			less := lim.with(calLimits{"schema-location-length": lim["schema-location-length"] - 1})
			run, err := runCase(tool, c.atLimit(lim), defaults, less, testenv.Dir(t), defaultWall)
			if err != nil {
				t.Fatal(err)
			}
			if err := run.res.refusedBy("schema-location-length", less["schema-location-length"], "inputs/report.schema.json"); err != nil {
				t.Fatalf("one byte under the locations: %v", err)
			}
		})
	}
	if n != 2 {
		t.Fatalf("%d joint cases, not A1 item 5's two", n)
	}
}

// atLimit is c built for lim whatever limits it is run under, so the run
// at one less than its limit is the same case.
func (c calCase) atLimit(lim calLimits) calCase {
	build := c.build
	c.build = func(dir string, _ calLimits) ([]string, error) { return build(dir, lim) }
	return c
}

// calRecord is the record of one calibration run on one target, the
// metais.calibration/1 format docs/dev/calibration.md describes.
type calRecord struct {
	Format   string        `json:"format"`
	GOOS     string        `json:"goos"`
	GOARCH   string        `json:"goarch"`
	Go       string        `json:"go"`
	Tool     string        `json:"tool"`
	CPUs     int           `json:"cpus"`
	Ceilings calCeilings   `json:"ceilings"`
	Cases    []calRecorded `json:"cases"`
	Pass     bool          `json:"pass"`
}

// calCeilings are D18 item 2's ceilings, as the record states them.
type calCeilings struct {
	CPUNanos         int64 `json:"cpu_ns"`
	MetadataCPUNanos int64 `json:"metadata_cpu_ns"`
	RSSBytes         int64 `json:"rss_bytes"`
}

// calRecorded is one case's measurement.
type calRecorded struct {
	Case        string   `json:"case"`
	Limit       string   `json:"limit"`
	Bound       int64    `json:"bound"`
	Metadata    bool     `json:"metadata"`
	Reached     bool     `json:"reached"`
	Gap         string   `json:"gap,omitempty"`
	Exit        int      `json:"exit"`
	CPUNanos    int64    `json:"cpu_ns"`
	RSSBytes    *int64   `json:"rss_bytes"`
	WallNanos   int64    `json:"wall_ns"`
	Broken      []string `json:"broken"`
	Problem     string   `json:"problem,omitempty"`
	CPUCeiling  int64    `json:"cpu_ceiling_ns"`
	RSSCeiling  int64    `json:"rss_ceiling_bytes"`
	CPUFraction float64  `json:"cpu_fraction"`
}

// TestCalibration is the limit-calibration suite at the defaults
// (ADR-009 D18 item 4). It runs only with -calibration.record, by the
// command docs/dev/calibration.md gives, one case at a time; it writes the
// record whatever the outcome, then fails naming each case and the ceiling
// it broke.
func TestCalibration(t *testing.T) {
	if *flagCalRecord == "" {
		t.Skip("the calibration suite runs at the defaults only with -calibration.record (docs/dev/calibration.md); TestCalibrationLowered runs it scaled down")
	}
	tool := calTool(t)
	if raceEnabled || raceBuilt(tool) {
		t.Fatal("the calibration suite measures a binary built as it is released: run it without -race")
	}
	defaults, err := toolDefaults(tool, testenv.Dir(t))
	if err != nil {
		t.Fatal(err)
	}
	only := list(*flagCalCases)
	for _, name := range only {
		if !slices.ContainsFunc(calCases, func(c calCase) bool { return c.name == name }) {
			t.Fatalf("-calibration.cases: %q is not a calibration case", name)
		}
	}
	version, err := runProc(tool, []string{"version"}, testenv.Dir(t), nil, nil, defaultWall)
	if err != nil {
		t.Fatal(err)
	}
	rec := &calRecord{
		Format: "metais.calibration/1", GOOS: runtime.GOOS, GOARCH: runtime.GOARCH,
		Go: runtime.Version(), Tool: strings.TrimSpace(string(version.stdout)), CPUs: runtime.NumCPU(),
		Ceilings: calCeilings{CPUNanos: int64(calCPU), MetadataCPUNanos: int64(calMetadataCPU), RSSBytes: calRSS},
		Pass:     true,
	}
	var failures []string
	for _, c := range calCases {
		if len(only) > 0 && !slices.Contains(only, c.name) {
			continue
		}
		r := measureCase(t, tool, c, defaults)
		rec.Cases = append(rec.Cases, r)
		rss := "not measured"
		if r.RSSBytes != nil {
			rss = fmt.Sprintf("%d MiB", *r.RSSBytes>>20)
		}
		t.Logf("%s: %v of CPU, %s peak resident, %v wall, exit %d, reached %v", c.name,
			time.Duration(r.CPUNanos), rss, time.Duration(r.WallNanos).Round(time.Millisecond), r.Exit, r.Reached)
		for _, b := range r.Broken {
			failures = append(failures, fmt.Sprintf("%s broke the %s", c.name, b))
		}
		if r.Problem != "" {
			failures = append(failures, fmt.Sprintf("%s: %s", c.name, r.Problem))
		}
	}
	rec.Pass = len(failures) == 0
	b, err := json.MarshalIndent(rec, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(*flagCalRecord, append(b, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Logf("the record is %s", *flagCalRecord)
	for _, f := range failures {
		t.Error(f)
	}
}

// measureCase runs c at the defaults and judges it.
func measureCase(t *testing.T, tool string, c calCase, defaults calLimits) calRecorded {
	r := calRecorded{Case: c.name, Limit: c.limit, Bound: defaults[c.limit], Metadata: c.metadata, Gap: c.gap,
		CPUCeiling: int64(calCPU), RSSCeiling: calRSS, Broken: []string{}}
	if c.metadata {
		r.CPUCeiling = int64(calMetadataCPU)
	}
	dir, err := os.MkdirTemp(testenv.Dir(t), "case-")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir) // a case's outputs reach a gibibyte
	run, err := runCase(tool, c, defaults, defaults, dir, calWall)
	if run == nil {
		r.Problem = err.Error()
		return r
	}
	r.Exit, r.CPUNanos, r.WallNanos = run.exit, int64(run.cpu), int64(run.wall)
	r.CPUFraction = float64(r.CPUNanos) / float64(r.CPUCeiling)
	if run.rss >= 0 {
		r.RSSBytes = &run.rss
	}
	switch {
	case run.killed:
		r.Broken = append(r.Broken, fmt.Sprintf("CPU ceiling of %v: the case ran past the %v wall clock and was killed", time.Duration(r.CPUCeiling), calWall))
	case err != nil:
		r.Problem = err.Error()
	default:
		reach := c.reached(run.res, defaults)
		r.Reached = reach == nil
		switch {
		case reach != nil && c.gap == "":
			r.Problem = fmt.Sprintf("the case does not reach %s at its default, so it is not the limit's worst case: %v", c.limit, reach)
		case reach == nil && c.gap != "":
			r.Problem = fmt.Sprintf("the case reaches %s, which its gap says the tool does not: remove the gap (%s)", c.limit, c.gap)
		}
	}
	if !run.killed && r.CPUNanos > r.CPUCeiling {
		r.Broken = append(r.Broken, fmt.Sprintf("CPU ceiling of %v with %v", time.Duration(r.CPUCeiling), time.Duration(r.CPUNanos)))
	}
	if run.rss > calRSS {
		r.Broken = append(r.Broken, fmt.Sprintf("peak resident memory ceiling of %d MiB with %d MiB", calRSS>>20, run.rss>>20))
	}
	return r
}
