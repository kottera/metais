// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// The tripwires that join the statements to the vectors and the finding
// registry: TestCoverage, TestCitationsResolve, TestFindingCodesCovered
// and TestExamplesMatchVectors (ADR-011 D14).

// vector is the part of a vector.json these tripwires read, by the
// members ADR-011 D9 item 3 defines. Whether the file is well formed is
// TestVectorsWellFormed's question, against the vector schema; these
// tests read what they need and leave every other member alone.
type vector struct {
	dir string // slash-separated, below spec/conformance/vectors/

	ID         string       `json:"id"`
	Statements []string     `json:"statements"`
	Since      string       `json:"since"`
	Class      string       `json:"class"`
	Tags       []string     `json:"tags"`
	Requires   []string     `json:"requires"`
	Cases      []vectorCase `json:"cases"`
}

type vectorCase struct {
	Name  string       `json:"name"`
	Steps []vectorStep `json:"steps"`
}

type vectorStep struct {
	Op     string `json:"op"`
	Expect struct {
		OK *bool `json:"ok"`
		// Findings names the expected-findings file, relative to the
		// vector's directory (ADR-011 D10 item 2). Absent, the step
		// expects no finding.
		Findings string `json:"findings"`
	} `json:"expect"`
}

// readVectors reads every vector.json under root's suite. A suite with
// no vector directory has no vectors.
func readVectors(root string) ([]vector, error) {
	dir := filepath.Join(root, filepath.FromSlash(vectorsDir))
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}
	var vs []vector
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "vector.json" {
			return err
		}
		rel, err := filepath.Rel(dir, filepath.Dir(p))
		if err != nil {
			return err
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		v := vector{dir: filepath.ToSlash(rel)}
		if err := json.Unmarshal(b, &v); err != nil {
			return fmt.Errorf("%s/%s/vector.json: %w", vectorsDir, v.dir, err)
		}
		vs = append(vs, v)
		return nil
	})
	slices.SortFunc(vs, func(a, b vector) int { return strings.Compare(a.dir, b.dir) })
	return vs, err
}

// expectedCodes reads the code of every finding v's steps expect.
func expectedCodes(root string, v vector) ([]string, error) {
	var codes []string
	for _, c := range v.Cases {
		for _, s := range c.Steps {
			name := s.Expect.Findings
			if name == "" {
				continue
			}
			if !filepath.IsLocal(filepath.FromSlash(name)) {
				return nil, fmt.Errorf("vector %s: the findings file %q is not inside the vector", v.dir, name)
			}
			b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(vectorsDir), filepath.FromSlash(v.dir), filepath.FromSlash(name)))
			if err != nil {
				return nil, fmt.Errorf("vector %s: %w", v.dir, err)
			}
			var fs []struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(b, &fs); err != nil {
				return nil, fmt.Errorf("vector %s: %s: %w", v.dir, name, err)
			}
			for _, f := range fs {
				codes = append(codes, f.Code)
			}
		}
	}
	return codes, nil
}

// registryCode is the part of a registry entry these tripwires read.
type registryCode struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Class    string `json:"class"`
	Subject  string `json:"subject"`
	Params   []struct {
		Name   string `json:"name"`
		Always bool   `json:"always"`
	} `json:"params"`
	Since      string   `json:"since"`
	Retired    string   `json:"retired"`
	Statements []string `json:"statements"`
}

func readRegistry(root string) ([]registryCode, error) {
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(registryFile)))
	if err != nil {
		return nil, err
	}
	var r struct {
		Codes []registryCode `json:"codes"`
	}
	if err := json.Unmarshal(b, &r); err != nil {
		return nil, fmt.Errorf("%s: %w", registryFile, err)
	}
	return r.Codes, nil
}

// unwiredFile is the unwired list, from this package's directory (plan
// §8, "The unwired list").
const unwiredFile = "../finding/testdata/unwired.txt"

// unwired is one line of the unwired list: a registered code no vector
// produces yet, and the wave expected to wire it.
type unwired struct {
	code, wave string
	line       int
}

// The waves a listed code may name: those after this one, up to Wave 7,
// by whose close the list is empty.
var unwiredWaveRe = regexp.MustCompile(`^W[3-7]$`)

// readUnwired reads the unwired list. A line is a code and a wave,
// separated by spaces; a line beginning with # is a comment. The codes are
// in byte order, each once.
func readUnwired(p string) ([]unwired, []string) {
	b, err := os.ReadFile(p)
	if err != nil {
		return nil, []string{err.Error()}
	}
	var list []unwired
	var problems []string
	sc := bufio.NewScanner(bytes.NewReader(b))
	for n := 1; sc.Scan(); n++ {
		l := sc.Text()
		if blank(l) || strings.HasPrefix(l, "#") {
			continue
		}
		f := strings.Fields(l)
		if len(f) != 2 || !unwiredWaveRe.MatchString(f[1]) {
			problems = append(problems, fmt.Sprintf("unwired.txt:%d: %q is not a code and the wave, W3 to W7, expected to wire it", n, l))
			continue
		}
		if k := len(list); k > 0 && list[k-1].code >= f[0] {
			problems = append(problems, fmt.Sprintf("unwired.txt:%d: %s is out of order or listed twice", n, f[0]))
		}
		list = append(list, unwired{code: f[0], wave: f[1], line: n})
	}
	return list, problems
}

// exemptClass reports whether codes of class are proved by internal/cli
// tests, not vectors: the cli adapter fails a step on the exit codes
// they give (plan §8, "Decisions settled before filing").
func exemptClass(class string) bool { return class == "usage" || class == "internal" }

// findingCodeProblems is TestFindingCodesCovered's check: every
// registered, unretired code of class refused, content or destination is
// produced by a vector and names a statement, or is on the unwired list;
// a listed code is registered, of such a class, and produced by no vector;
// and a code of class usage or internal names no statement.
func findingCodeProblems(codes []registryCode, list []unwired, produced map[string][]string) []string {
	var problems []string
	listed := map[string]unwired{}
	for _, u := range list {
		listed[u.code] = u
	}
	registered := map[string]registryCode{}
	for _, c := range codes {
		registered[c.Code] = c
		u, onList := listed[c.Code]
		by := produced[c.Code]
		switch {
		case c.Retired != "":
			if onList {
				problems = append(problems, fmt.Sprintf("unwired.txt:%d: %s is retired, so nothing wires it", u.line, c.Code))
			}
		case exemptClass(c.Class):
			if onList {
				problems = append(problems, fmt.Sprintf("unwired.txt:%d: %s is of class %s, which internal/cli's tests prove, and is not listed", u.line, c.Code, c.Class))
			}
			if len(c.Statements) > 0 {
				problems = append(problems, fmt.Sprintf("%s is of class %s, and names the statements %q; a code the command line alone reports names none", c.Code, c.Class, c.Statements))
			}
		case onList:
			if len(by) > 0 {
				problems = append(problems, fmt.Sprintf("unwired.txt:%d: %s is produced by %s; take it off the list, which only shrinks", u.line, c.Code, strings.Join(by, ", ")))
			}
		default:
			if len(by) == 0 {
				problems = append(problems, fmt.Sprintf("%s is registered, of class %s, and no vector's expected findings hold it; add one, or list it in unwired.txt with the wave that wires it", c.Code, c.Class))
			}
			if len(c.Statements) == 0 {
				problems = append(problems, fmt.Sprintf("%s names no statement", c.Code))
			}
		}
	}
	for _, u := range list {
		if _, ok := registered[u.code]; !ok {
			problems = append(problems, fmt.Sprintf("unwired.txt:%d: %s is not registered in spec/%s", u.line, u.code, registryFile))
		}
	}
	return problems
}

// producedCodes maps each code the vectors' expected findings hold to the
// vectors that hold it.
func producedCodes(root string, vs []vector) (map[string][]string, error) {
	produced := map[string][]string{}
	for _, v := range vs {
		codes, err := expectedCodes(root, v)
		if err != nil {
			return nil, err
		}
		for _, c := range codes {
			if !slices.Contains(produced[c], v.dir) {
				produced[c] = append(produced[c], v.dir)
			}
		}
	}
	return produced, nil
}

// codesCoveredProblems runs findingCodeProblems over root and the unwired
// list at unwiredPath.
func codesCoveredProblems(root, unwiredPath string) []string {
	codes, err := readRegistry(root)
	if err != nil {
		return []string{err.Error()}
	}
	vs, err := readVectors(root)
	if err != nil {
		return []string{err.Error()}
	}
	produced, err := producedCodes(root, vs)
	if err != nil {
		return []string{err.Error()}
	}
	list, problems := readUnwired(unwiredPath)
	return append(problems, findingCodeProblems(codes, list, produced)...)
}

// coverageProblems is TestCoverage's check: every active, observable
// requirement is cited by a vector. It returns, too, each statement's
// citing vectors.
func coverageProblems(stmts []statement, vs []vector) ([]string, map[string][]string) {
	citing := map[string][]string{}
	for _, v := range vs {
		for _, id := range v.Statements {
			citing[id] = append(citing[id], v.dir)
		}
	}
	var problems []string
	for _, s := range stmts {
		if s.active() && s.Observable && s.Kind == "requirement" && len(citing[s.ID]) == 0 {
			problems = append(problems, fmt.Sprintf("%s:%d: %s is an observable requirement, and no vector cites it", s.File, s.Line, s.ID))
		}
	}
	return problems, citing
}

var (
	mdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\(([^)\s]*)\)`)
	stmtIDRe = regexp.MustCompile(`^MTS-[A-Z]{3}-[0-9]{3,}$`)
)

// citationProblems is TestCitationsResolve's check: every statement a
// vector or a registry entry names exists, and a vector names none that
// is withdrawn; every link to a statement anchor, or with a statement
// identifier as its text, resolves to that statement.
func citationProblems(docs []*mdDoc, stmts []statement, vs []vector, codes []registryCode) []string {
	byID := map[string]statement{}
	anchors := map[string]bool{} // "file#anchor"
	for _, s := range stmts {
		byID[s.ID] = s
		anchors[s.File+"#"+s.anchor()] = true
	}
	var problems []string
	for _, v := range vs {
		for _, id := range v.Statements {
			s, ok := byID[id]
			switch {
			case !ok:
				problems = append(problems, fmt.Sprintf("vector %s cites %s, which no chapter states", v.dir, id))
			case !s.active():
				problems = append(problems, fmt.Sprintf("vector %s cites %s, which %s withdrew", v.dir, id, s.Withdrawn))
			}
		}
	}
	for _, c := range codes {
		for _, id := range c.Statements {
			if _, ok := byID[id]; !ok {
				problems = append(problems, fmt.Sprintf("the registry's %s names %s, which no chapter states", c.Code, id))
			}
		}
	}
	for _, d := range docs {
		for i, l := range d.lines {
			if d.fenced[i] {
				continue
			}
			for _, m := range mdLinkRe.FindAllStringSubmatch(l, -1) {
				text, target := m[1], m[2]
				file, frag, _ := strings.Cut(target, "#")
				isID := stmtIDRe.MatchString(text)
				if !isID && !strings.HasPrefix(strings.ToLower(frag), "mts-") {
					continue
				}
				where := fmt.Sprintf("%s:%d", d.path, i+1)
				if strings.Contains(file, "://") {
					problems = append(problems, fmt.Sprintf("%s: the link %s to a statement leaves the specification", where, m[0]))
					continue
				}
				if file == "" {
					file = d.path
				} else {
					file = path.Clean(path.Join(path.Dir(d.path), file))
				}
				switch {
				case isID && frag != strings.ToLower(text):
					problems = append(problems, fmt.Sprintf("%s: the link %s names %s and points at #%s", where, m[0], text, frag))
				case !anchors[file+"#"+frag]:
					problems = append(problems, fmt.Sprintf("%s: the link %s does not resolve to a statement", where, m[0]))
				}
			}
		}
	}
	return problems
}

var (
	exampleRe   = regexp.MustCompile(`^<!-- example: ([^ ]+) -->$`)
	pathAttrRe  = regexp.MustCompile(`(?:^|\s)path=(\S+)`)
	vectorIDRe  = regexp.MustCompile(`^[a-z]{3}/[a-z0-9]+(?:-[a-z0-9]+)*$`)
	exampleLead = "<!-- example:"
)

// exampleProblems is TestExamplesMatchVectors' check (ADR-011 D4): an
// example comment names a vector, the fenced blocks that follow it each
// name a file of that vector with path=, and each block is that file,
// byte for byte. A block with path= outside an example is checked by
// nothing, and is refused.
func exampleProblems(root string, docs []*mdDoc) []string {
	var problems []string
	for _, d := range docs {
		claimed := map[int]bool{} // fence opening lines inside an example
		for i, l := range d.lines {
			if d.fenced[i] || !strings.HasPrefix(strings.TrimSpace(l), exampleLead) {
				continue
			}
			where := fmt.Sprintf("%s:%d", d.path, i+1)
			m := exampleRe.FindStringSubmatch(strings.TrimSpace(l))
			if m == nil || !vectorIDRe.MatchString(m[1]) {
				problems = append(problems, fmt.Sprintf("%s: %q is not an example comment naming a vector", where, l))
				continue
			}
			vdir := filepath.Join(root, filepath.FromSlash(vectorsDir), filepath.FromSlash(m[1]))
			if _, err := os.Stat(filepath.Join(vdir, "vector.json")); err != nil {
				problems = append(problems, fmt.Sprintf("%s: the example names the vector %s, which does not exist", where, m[1]))
				continue
			}
			shown := 0
			for j := i + 1; j < len(d.lines); j++ {
				if blank(d.lines[j]) {
					continue
				}
				k := slices.IndexFunc(d.fences, func(f fenceBlock) bool { return f.open == j })
				if k < 0 {
					break
				}
				f := d.fences[k]
				pm := pathAttrRe.FindStringSubmatch(f.info)
				if pm == nil {
					break
				}
				claimed[j] = true
				shown++
				j = f.close
				name := pm[1]
				if !filepath.IsLocal(filepath.FromSlash(name)) {
					problems = append(problems, fmt.Sprintf("%s:%d: the example's path %q is not inside the vector", d.path, f.open+1, name))
					continue
				}
				want, err := os.ReadFile(filepath.Join(vdir, filepath.FromSlash(name)))
				if err != nil {
					problems = append(problems, fmt.Sprintf("%s:%d: the example shows %s, which the vector %s does not hold", d.path, f.open+1, name, m[1]))
					continue
				}
				if got := f.body(d); got != string(want) {
					problems = append(problems, fmt.Sprintf("%s:%d: the example's %s differs from the vector %s's file", d.path, f.open+1, name, m[1]))
				}
			}
			if shown == 0 {
				problems = append(problems, fmt.Sprintf("%s: the example names %s and shows no file of it", where, m[1]))
			}
		}
		for _, f := range d.fences {
			if !claimed[f.open] && pathAttrRe.MatchString(f.info) {
				problems = append(problems, fmt.Sprintf("%s:%d: a block naming a vector file with path= is not inside an example", d.path, f.open+1))
			}
		}
	}
	return problems
}

func mustVectors(t *testing.T, root string) []vector {
	t.Helper()
	vs, err := readVectors(root)
	if err != nil {
		t.Fatal(err)
	}
	return vs
}

func mustRegistry(t *testing.T, root string) []registryCode {
	t.Helper()
	codes, err := readRegistry(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(codes) == 0 {
		t.Fatalf("the registry under %s has no code", root)
	}
	return codes
}

// TestCoverage is ADR-011 D14's tripwire over spec/. With -v it prints
// every statement and the vectors citing it.
func TestCoverage(t *testing.T) {
	stmts, _ := specStatements(mustDocs(t, specRoot))
	problems, citing := coverageProblems(stmts, mustVectors(t, specRoot))
	report(t, problems)
	if testing.Verbose() {
		var b strings.Builder
		fmt.Fprintf(&b, "%d statements\n", len(stmts))
		for _, s := range stmts {
			fmt.Fprintf(&b, "%-14s %-14s %-11s %s\n", s.ID, s.Kind, map[bool]string{true: "observable", false: "-"}[s.Observable], orDash(strings.Join(citing[s.ID], " ")))
		}
		t.Log("\n" + b.String())
	}
}

// TestCitationsResolve is ADR-011 D14's tripwire over spec/.
func TestCitationsResolve(t *testing.T) {
	docs := mustDocs(t, specRoot)
	stmts, _ := specStatements(docs)
	report(t, citationProblems(docs, stmts, mustVectors(t, specRoot), mustRegistry(t, specRoot)))
}

// TestFindingCodesCovered is ADR-011 D14's tripwire and ADR-006 D13 item
// 4's second test, over spec/ and the unwired list.
func TestFindingCodesCovered(t *testing.T) {
	report(t, codesCoveredProblems(specRoot, unwiredFile))
}

// TestExamplesMatchVectors is ADR-011 D14's tripwire over spec/.
func TestExamplesMatchVectors(t *testing.T) {
	report(t, exampleProblems(specRoot, mustDocs(t, specRoot)))
}
