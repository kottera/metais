// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// The flags of ADR-011 D11 item 3, given after -args.
var (
	flagAdapter  = flag.String("conformance.adapter", "cli", "cli drives a metais-compatible command line; exec drives a driver")
	flagTool     = flag.String("conformance.tool", "", "the command-line tool for cli (default: the binary built from ./cmd/metais)")
	flagDriver   = flag.String("conformance.driver", "", "the driver executable for exec")
	flagSuite    = flag.String("conformance.suite", "", "the suite directory (default: spec/conformance in the module)")
	flagVersion  = flag.String("conformance.version", "", "the highest revision claimed (default: the front page's Version)")
	flagClasses  = flag.String("conformance.classes", "processor,authoring", "which classes to run")
	flagTags     = flag.String("conformance.tags", "", "run only vectors with one of these tags (default: all)")
	flagSkipTags = flag.String("conformance.skip-tags", "", "skip vectors with one of these tags")
	flagRecord   = flag.String("conformance.record", "", "write the cross-platform record to this file")
	flagCompare  = flag.String("conformance.compare", "", "compare these records, comma-separated, instead of running vectors")
	flagKeep     = flag.String("conformance.keep", "", "keep each failing case's sandbox under this directory")
)

// builtTool is the metais binary TestMain built from ./cmd/metais, or "".
var builtTool string

// moduleRoot is the module's root, from this package's directory.
const moduleRoot = "../.."

// The names the test binary answers to when it is re-executed: as the
// reference driver, as a fake implementation in a directory named for its
// fault (ADR-011 D11 item 7, D18 item 1), and as the stub that starts a
// measured process on linux (measured).
const (
	refDriverName = "metais-refdriver"
	procStubName  = "metais-procstub"
	fakeDirPrefix = "metais-fake-"
	toolName      = "metais"
)

// TestMain re-executes as the reference driver, a fake or the stub when
// the test binary is run under one of their names. Otherwise it builds ./cmd/metais
// once, with -trimpath and with -race when this binary is race-built
// (ADR-011 D11 item 2), and runs the tests.
func TestMain(m *testing.M) {
	base := filepath.Base(os.Args[0])
	if base == procStubName {
		os.Exit(procStubMain(os.Args[1:]))
	}
	if base == refDriverName {
		os.Exit(refDriverMain(filepath.Join(filepath.Dir(os.Args[0]), toolName)))
	}
	if fault, ok := strings.CutPrefix(filepath.Base(filepath.Dir(os.Args[0])), fakeDirPrefix); ok && base == toolName {
		os.Exit(fakeMain(fault, os.Args[1:]))
	}
	flag.Parse()
	var dir string
	if *flagCompare == "" && *flagTool == "" {
		var err error
		dir, err = os.MkdirTemp("", "metais-conformance-")
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		builtTool = filepath.Join(dir, toolName)
		if err := buildTool(builtTool); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.RemoveAll(dir)
			os.Exit(1)
		}
	}
	code := m.Run()
	if dir != "" {
		os.RemoveAll(dir)
	}
	os.Exit(code)
}

// buildTool builds ./cmd/metais to out.
func buildTool(out string) error {
	args := []string{"build", "-trimpath", "-o", out}
	if raceEnabled {
		args = append(args, "-race")
	}
	cmd := exec.Command("go", append(args, "./cmd/metais")...)
	cmd.Dir = moduleRoot
	if b, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("building ./cmd/metais: %v\n%s", err, b)
	}
	return nil
}

// defaultSuite is spec/conformance in the module, absolute.
func defaultSuite() string {
	p, err := filepath.Abs(filepath.Join(moduleRoot, "spec", "conformance"))
	if err != nil {
		return filepath.Join(moduleRoot, "spec", "conformance")
	}
	return p
}

// list splits a comma-separated flag value.
func list(s string) []string {
	var out []string
	for _, x := range strings.Split(s, ",") {
		if x = strings.TrimSpace(x); x != "" {
			out = append(out, x)
		}
	}
	return out
}

// newRunner makes a runner over suite for adapter, with the defaults of
// ADR-011 D11 item 3 for everything the caller does not set.
func newRunner(t testing.TB, suite string, a adapter) *runner {
	t.Helper()
	version, err := frontPageVersion(suite)
	if err != nil {
		t.Fatalf("the claimed revision: %v", err)
	}
	return &runner{
		suite:   suite,
		adapter: a,
		version: version,
		classes: []string{"processor", "authoring"},
		short:   testing.Short(),
		wall:    defaultWall,
		base:    t.TempDir(),
	}
}

// newAdapter makes the adapter of kind for program, reading the registry
// and the driver-response schema of suite's spec tree.
func newAdapter(kind, program, suite string) (adapter, error) {
	if err := executable(program); err != nil {
		return nil, err
	}
	switch kind {
	case "cli":
		reg, err := readRegistry(suite)
		if err != nil {
			return nil, err
		}
		return &cliAdapter{path: program, reg: reg}, nil
	case "exec":
		set, err := suiteSchemas(suite)
		if err != nil {
			return nil, err
		}
		return &execAdapter{path: program, validate: set.validator(driverResponseSchema)}, nil
	}
	return nil, fmt.Errorf("-conformance.adapter=%s: not cli or exec", kind)
}

// TestConformance runs every vector of the suite against the
// implementation the flags name (ADR-011 D11), or, with
// -conformance.compare, compares cross-platform records (D15 item 4).
func TestConformance(t *testing.T) {
	suite := *flagSuite
	if suite == "" {
		suite = defaultSuite()
	}
	if *flagCompare != "" {
		if err := compareFiles(suite, list(*flagCompare)); err != nil {
			t.Fatal(err)
		}
		return
	}
	program := *flagTool
	if *flagAdapter == "exec" {
		program = *flagDriver
		if program == "" {
			t.Fatal("-conformance.adapter=exec needs -conformance.driver")
		}
	} else if program == "" {
		program = builtTool
	}
	a, err := newAdapter(*flagAdapter, program, suite)
	if err != nil {
		t.Fatal(err)
	}
	r := newRunner(t, suite, a)
	if *flagVersion != "" {
		if _, err := parseRevision(*flagVersion); err != nil {
			t.Fatalf("-conformance.version: %v", err)
		}
		r.version = *flagVersion
	}
	r.classes = list(*flagClasses)
	r.tags, r.skipTags = list(*flagTags), list(*flagSkipTags)
	if err := checkSelection(r.classes, append(slices.Clone(r.tags), r.skipTags...)); err != nil {
		t.Fatal(err)
	}
	r.keep = *flagKeep
	if *flagRecord != "" {
		r.rec = &recorder{path: *flagRecord}
	}
	runSuite(t, r)
}

// checkSelection refuses a class or a tag the suite does not define, so a
// mistyped flag cannot pass by selecting nothing.
func checkSelection(classes, tags []string) error {
	for _, c := range classes {
		if c != "processor" && c != "authoring" {
			return fmt.Errorf("-conformance.classes: %q is not processor or authoring", c)
		}
	}
	known := []string{"security", "determinism", "portable-paths", "container", "byte-level", "limits", "regression"}
	for _, tag := range tags {
		if !slices.Contains(known, tag) {
			return fmt.Errorf("-conformance.tags: %q is not a tag of ADR-011 D9 item 5", tag)
		}
	}
	return nil
}

// compareFiles compares the records at paths, validating each against the
// record schema of suite's spec tree.
func compareFiles(suite string, paths []string) error {
	if len(paths) < 2 {
		return errors.New("-conformance.compare needs at least two records")
	}
	set, err := suiteSchemas(suite)
	if err != nil {
		return err
	}
	var recs []*record
	for _, p := range paths {
		rec, err := readRecord(p, set.validator(recordSchema))
		if err != nil {
			return err
		}
		recs = append(recs, rec)
	}
	return errors.Join(compareRecords(paths, recs)...)
}
