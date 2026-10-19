// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// repoRoot is the repository root, from this package's directory.
const repoRoot = "../.."

// delegatedPath is the delegated-run file of ADR-011 D17 item 1.
var delegatedPath = filepath.Join(repoRoot, "spec", "conformance", "delegated", "liquid-spec.json")

// delegated is the delegated-run file, every member named.
type delegated struct {
	Repository      string      `json:"repository"`
	Commit          string      `json:"commit"`
	Suites          []string    `json:"suites"`
	ErrorMode       string      `json:"error_mode"`
	MissingFeatures []string    `json:"missing_features"`
	Departures      []departure `json:"departures"`
}

// departure is a liquid-spec case the dialect fails, the statement that
// makes it differ, and why (ADR-011 D17 item 2).
type departure struct {
	Spec      string `json:"spec"`
	Statement string `json:"statement"`
	Why       string `json:"why"`
}

// readDelegated reads a delegated-run file, refusing a member the shape
// does not name.
func readDelegated(t *testing.T, b []byte) delegated {
	t.Helper()
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	var d delegated
	if err := dec.Decode(&d); err != nil {
		t.Fatalf("the delegated-run file: %v", err)
	}
	return d
}

// adrPin is the delegated-run file as ADR-011 D17 item 1 writes it: the
// JSON block of that item, read from the ADR's own text.
func adrPin(t *testing.T) delegated {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "docs", "adr", "ADR-011-the-specification-document-and-conformance.md"))
	if err != nil {
		t.Fatal(err)
	}
	_, d17, ok := strings.Cut(string(b), "\n### D17 ")
	if !ok {
		t.Fatal("ADR-011 has no D17")
	}
	d17, _, _ = strings.Cut(d17, "\n### ")
	m := regexp.MustCompile("(?s)```json\n(.*?)```").FindStringSubmatch(d17)
	if m == nil {
		t.Fatal("ADR-011 D17 has no JSON block")
	}
	return readDelegated(t, []byte(m[1]))
}

// TestDelegatedFile is W4-6.3: the delegated-run file carries the pin,
// the suites, the error mode and the feature gates exactly as ADR-011 D17
// item 1 writes them, and every departure is a case name, a statement and
// a reason, each case once, in byte order.
func TestDelegatedFile(t *testing.T) {
	testenv.Home(t)
	b, err := os.ReadFile(delegatedPath)
	if err != nil {
		t.Fatal(err)
	}
	d, want := readDelegated(t, b), adrPin(t)
	if d.Repository != want.Repository || d.Commit != want.Commit || !slices.Equal(d.Suites, want.Suites) ||
		d.ErrorMode != want.ErrorMode || !slices.Equal(d.MissingFeatures, want.MissingFeatures) {
		t.Errorf("the pin is\n%+v\nwant ADR-011 D17 item 1's\n%+v", d, want)
	}
	if !regexp.MustCompile(`^[0-9a-f]{40}$`).MatchString(d.Commit) {
		t.Errorf("commit %q is not a full commit hash", d.Commit)
	}
	if len(d.Departures) == 0 {
		t.Error("no departures")
	}
	for i, x := range d.Departures {
		if x.Spec == "" || x.Statement == "" || x.Why == "" {
			t.Errorf("departure %d: %+v, want a spec, a statement and why", i, x)
		}
		if i > 0 && d.Departures[i-1].Spec >= x.Spec {
			t.Errorf("departure %q follows %q: the departures are each once, in byte order", x.Spec, d.Departures[i-1].Spec)
		}
	}
}

// statementIndex is spec/statements.json, as far as departures read it.
type statementIndex struct {
	Statements []struct {
		ID      string `json:"id"`
		Chapter string `json:"chapter"`
		Status  string `json:"status"`
	} `json:"statements"`
}

// departureProblems returns what is wrong with deps' statements: each
// must be an active statement of chapter 7 or 8 that the chapter's text
// holds (ADR-011 D17 item 2).
func departureProblems(deps []departure, idx statementIndex, chapters map[string]string) []string {
	var problems []string
	for _, x := range deps {
		found := false
		for _, s := range idx.Statements {
			if s.ID != x.Statement {
				continue
			}
			found = true
			switch text, ok := chapters[s.Chapter]; {
			case !ok:
				problems = append(problems, fmt.Sprintf("%s: %s is a statement of %s, not of chapter 7 or 8", x.Spec, x.Statement, s.Chapter))
			case s.Status != "active":
				problems = append(problems, fmt.Sprintf("%s: %s is %s", x.Spec, x.Statement, s.Status))
			case !strings.Contains(text, `<a id="`+strings.ToLower(x.Statement)+`"></a>`):
				problems = append(problems, fmt.Sprintf("%s: %s is not in the text of %s", x.Spec, x.Statement, s.Chapter))
			}
		}
		if !found {
			problems = append(problems, fmt.Sprintf("%s: %s is not a statement", x.Spec, x.Statement))
		}
	}
	return problems
}

// readStatements reads the statement index and the text of chapters 7
// and 8.
func readStatements(t *testing.T) (statementIndex, map[string]string) {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(repoRoot, "spec", "statements.json"))
	if err != nil {
		t.Fatal(err)
	}
	var idx statementIndex
	if err := json.Unmarshal(b, &idx); err != nil {
		t.Fatal(err)
	}
	chapters := map[string]string{}
	for _, c := range []string{"07-template-environment.md", "08-filters-and-tags.md"} {
		b, err := os.ReadFile(filepath.Join(repoRoot, "spec", c))
		if err != nil {
			t.Fatal(err)
		}
		chapters[c] = string(b)
	}
	return idx, chapters
}

// TestDepartureStatements is W4-6.4: every departure names a statement
// that exists in chapter 7 or 8; and, as its control, a departure naming
// a statement that does not exist, one of another chapter, and a
// withdrawn one are each found.
func TestDepartureStatements(t *testing.T) {
	testenv.Home(t)
	b, err := os.ReadFile(delegatedPath)
	if err != nil {
		t.Fatal(err)
	}
	idx, chapters := readStatements(t)
	for _, p := range departureProblems(readDelegated(t, b).Departures, idx, chapters) {
		t.Error(p)
	}
	idx.Statements = append(idx.Statements, struct {
		ID      string `json:"id"`
		Chapter string `json:"chapter"`
		Status  string `json:"status"`
	}{"MTS-TPL-999", "07-template-environment.md", "withdrawn"})
	got := departureProblems([]departure{
		{Spec: "a", Statement: "MTS-TPL-002"},
		{Spec: "b", Statement: "MTS-TPL-998"},
		{Spec: "c", Statement: "MTS-RND-001"},
		{Spec: "d", Statement: "MTS-TPL-999"},
	}, idx, chapters)
	want := []string{
		"b: MTS-TPL-998 is not a statement",
		"c: MTS-RND-001 is a statement of 11-rendering.md, not of chapter 7 or 8",
		"d: MTS-TPL-999 is withdrawn",
	}
	if !slices.Equal(got, want) {
		t.Errorf("the control found\n%q\nwant\n%q", got, want)
	}
}

// runResult is liquid-spec's --json result of a run, as far as the
// verdict reads it.
type runResult struct {
	Totals struct {
		Passed int `json:"passed"`
	} `json:"totals"`
	Failures      []runEntry `json:"failures"`
	KnownFailures []runEntry `json:"known_failures"`
	KnownFixed    []runEntry `json:"known_fixed"`
}

// runEntry is one case of a run's result.
type runEntry struct {
	Name   string `json:"name"`
	Actual any    `json:"actual"`
	Error  *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// verdict judges a run of the pinned selection against its departures
// (ADR-011 D17 item 2): every departure fails, and one that passes fails
// the run; every other case passes, but for those the server held out
// (HeldOut); and a departure the run did not reach fails the run, since
// the selection and the file have stopped agreeing. It returns the run's
// counts and every problem.
func verdict(r runResult, deps []departure) (counts string, problems []string) {
	listed := map[string]bool{}
	for _, x := range deps {
		listed[x.Spec] = true
	}
	seen := map[string]bool{}
	for _, e := range r.KnownFixed {
		seen[e.Name] = true
		problems = append(problems, fmt.Sprintf("%s: a departure passes", e.Name))
	}
	for _, e := range r.KnownFailures {
		seen[e.Name] = true
		if !listed[e.Name] {
			problems = append(problems, fmt.Sprintf("%s: liquid-spec counts a known failure the file does not list", e.Name))
		}
	}
	held := 0
	for _, e := range r.Failures {
		text := fmt.Sprint(e.Actual)
		if e.Error != nil {
			text += " " + e.Error.Message
		}
		if strings.Contains(text, HeldOut) {
			held++
			continue
		}
		problems = append(problems, fmt.Sprintf("%s: fails, and is no departure: %s", e.Name, strings.TrimSpace(text)))
	}
	for _, x := range deps {
		if !seen[x.Spec] {
			problems = append(problems, fmt.Sprintf("%s: a departure the run did not reach", x.Spec))
		}
	}
	return fmt.Sprintf("%d pass, %d departures fail as listed, %d held out for applying date, %d fail otherwise, %d departures pass",
		r.Totals.Passed, len(r.KnownFailures), held, len(r.Failures)-held, len(r.KnownFixed)), problems
}

// TestDelegatedRun is the verdict on the delegated run, the out-of-band
// act of ADR-011 D17 item 3: given liquid-spec's --json result with
// -results, it fails on every problem verdict finds and reports the
// counts. Without -results there is no run to judge; README.md beside the
// delegated-run file is how a person makes one.
func TestDelegatedRun(t *testing.T) {
	testenv.Home(t)
	if *resultsFlag == "" {
		t.Skip("no -results: no delegated run to judge (spec/conformance/delegated/README.md)")
	}
	b, err := os.ReadFile(delegatedPath)
	if err != nil {
		t.Fatal(err)
	}
	deps := readDelegated(t, b).Departures
	rb, err := os.ReadFile(*resultsFlag)
	if err != nil {
		t.Fatal(err)
	}
	var r runResult
	if err := json.Unmarshal(rb, &r); err != nil {
		t.Fatalf("%s is not liquid-spec's --json result: %v", *resultsFlag, err)
	}
	counts, problems := verdict(r, deps)
	t.Log(counts)
	for _, p := range problems {
		t.Error(p)
	}
}

// TestVerdictControl holds verdict to each problem it must find, and to
// none in a run that agrees with the file.
func TestVerdictControl(t *testing.T) {
	testenv.Home(t)
	deps := []departure{{Spec: "a"}, {Spec: "b"}, {Spec: "c"}}
	entry := func(name, actual, message string) runEntry {
		e := runEntry{Name: name, Actual: actual}
		if message != "" {
			e.Error = &struct {
				Message string `json:"message"`
			}{message}
		}
		return e
	}
	var ok runResult
	ok.Totals.Passed = 7
	ok.KnownFailures = []runEntry{entry("a", "x", ""), entry("b", "", "y"), entry("c", "", "")}
	ok.Failures = []runEntry{entry("d", "", "Liquid syntax error: "+HeldOut), entry("e", "LiquidSyntaxError: "+HeldOut, "")}
	counts, problems := verdict(ok, deps)
	if len(problems) != 0 || counts != "7 pass, 3 departures fail as listed, 2 held out for applying date, 0 fail otherwise, 0 departures pass" {
		t.Errorf("an agreeing run: %s, %q", counts, problems)
	}
	bad := ok
	bad.KnownFailures = []runEntry{entry("a", "", ""), entry("z", "", "")}
	bad.KnownFixed = []runEntry{entry("b", "", "")}
	bad.Failures = append(slices.Clone(ok.Failures), entry("f", "rendered x", ""))
	_, problems = verdict(bad, deps)
	want := []string{
		"b: a departure passes",
		"z: liquid-spec counts a known failure the file does not list",
		"f: fails, and is no departure: rendered x",
		"c: a departure the run did not reach",
	}
	if !reflect.DeepEqual(problems, want) {
		t.Errorf("a disagreeing run found\n%q\nwant\n%q", problems, want)
	}
}

// TestServeProgram runs the test binary as liquid-spec runs it, with
// -serve, over its standard input and output.
func TestServeProgram(t *testing.T) {
	testenv.Home(t)
	cmd := exec.Command(os.Args[0], "-serve")
	cmd.Stdin = strings.NewReader(request(1, "initialize", map[string]any{"version": "1.0"}) + "\n" +
		request(2, "compile", map[string]any{"template": "{{ a | plus: 1 }}"}) + "\n" +
		request(3, "render", map[string]any{"template_id": "tmpl_1", "environment": map[string]any{"a": 41}}) + "\n" +
		`{"jsonrpc": "2.0", "method": "quit"}` + "\n")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("%v: %s", err, stderr.Bytes())
	}
	lines := strings.Split(strings.TrimSuffix(string(out), "\n"), "\n")
	if len(lines) != 3 || !strings.Contains(lines[2], `"output":"42"`) {
		t.Errorf("standard output %q, want three responses, the last rendering 42", out)
	}
}
