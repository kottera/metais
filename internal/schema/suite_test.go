// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// The JSON-Schema-Test-Suite (ADR-010 D5 item 7, D14 item 4), vendored
// under testdata/json-schema-test-suite with its licence and SOURCE.
const suiteDir = "testdata/json-schema-test-suite"

// suiteFiles are the files TestJSONSchemaTestSuite runs: every required
// test of draft 2020-12, and the optional format tests of every format
// ADR-003 D4 asserts, which hold METAIS's own validators to the suite's
// cases.
func suiteFiles(t *testing.T) []string {
	t.Helper()
	var files []string
	entries, err := os.ReadDir(path.Join(suiteDir, "draft2020-12"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			files = append(files, e.Name())
		}
	}
	for _, name := range Formats() {
		files = append(files, "optional/format/"+name+".json")
	}
	slices.Sort(files)
	return files
}

type suiteCase struct {
	Description string          `json:"description"`
	Schema      json.RawMessage `json:"schema"`
	Tests       []struct {
		Description string          `json:"description"`
		Data        json.RawMessage `json:"data"`
		Valid       bool            `json:"valid"`
	} `json:"tests"`
}

// exclusion is one line of exclusions.txt: a case, or one test of it,
// whose outcome METAIS's own rules change, what METAIS gives instead, and
// the ADR-003 item that says so.
type exclusion struct {
	file, kase, test string // test is "*" for every test of the case
	outcome          string // a finding code the compiler reports, or "valid" or "invalid"
	item             string
	used             bool
}

var itemRef = regexp.MustCompile(`^ADR-003 D[0-9]+ item [0-9]+$`)

func readExclusions(t *testing.T) []*exclusion {
	t.Helper()
	b, err := os.ReadFile(path.Join(suiteDir, "exclusions.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var out []*exclusion
	sc := bufio.NewScanner(bytes.NewReader(b))
	for n := 1; sc.Scan(); n++ {
		line := sc.Text()
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		f := strings.Split(line, "\t")
		if len(f) != 5 {
			t.Fatalf("exclusions.txt:%d: %d fields, want file, case, test, outcome and ADR-003 item", n, len(f))
		}
		if !itemRef.MatchString(f[4]) {
			t.Fatalf("exclusions.txt:%d: %q names no ADR-003 item", n, f[4])
		}
		out = append(out, &exclusion{file: f[0], kase: f[1], test: f[2], outcome: f[3], item: f[4]})
	}
	return out
}

func find(ex []*exclusion, file, kase, test string) *exclusion {
	for _, e := range ex {
		if e.file == file && e.kase == kase && (e.test == "*" || e.test == test) {
			e.used = true
			return e
		}
	}
	return nil
}

// TestJSONSchemaTestSuite runs the suite over the configured compiler:
// each case's schema is the one schema file of a package, compiled by
// NewCompiler, and each test's data is validated against it. A test
// passes when METAIS agrees with the suite. Where an exclusion names the
// case, METAIS must give exactly the exclusion's outcome instead, so an
// exclusion records what METAIS does and is not a skip. Every exclusion
// must be used.
func TestJSONSchemaTestSuite(t *testing.T) {
	testenv.Home(t)
	ex := readExclusions(t)
	ran, excluded := 0, 0
	for _, file := range suiteFiles(t) {
		b, err := os.ReadFile(path.Join(suiteDir, "draft2020-12", file))
		if err != nil {
			t.Fatal(err)
		}
		var cases []suiteCase
		if err := json.Unmarshal(b, &cases); err != nil {
			t.Fatalf("%s: %v", file, err)
		}
		for _, c := range cases {
			comp, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/schema.json": string(c.Schema)})
			for _, tc := range c.Tests {
				name := file + "/" + c.Description + "/" + tc.Description
				got := "invalid"
				if len(fs) > 0 {
					got = strings.Join(codes(fs), ",")
				} else {
					s, _ := comp.Schema("inputs/schema.json")
					fails, err := s.Validate(mustValue(t, string(tc.Data)))
					if err != nil {
						t.Errorf("%s: %v", name, err)
						continue
					}
					if len(fails) == 0 {
						got = "valid"
					}
					for i := range fails {
						if err := fails[i].Check(); err != nil {
							t.Errorf("%s: the adapter built a malformed tree: %v", name, err)
						}
					}
				}
				want := "invalid"
				if tc.Valid {
					want = "valid"
				}
				if e := find(ex, file, c.Description, tc.Description); e != nil {
					want = e.outcome
					excluded++
				}
				ran++
				if got != want {
					t.Errorf("%s: got %s, want %s", name, got, want)
				}
			}
		}
	}
	for _, e := range ex {
		if !e.used {
			t.Errorf("exclusion %s / %s / %s matches no test", e.file, e.kase, e.test)
		}
	}
	t.Logf("ran %d tests, %d of them under an exclusion", ran, excluded)
}
