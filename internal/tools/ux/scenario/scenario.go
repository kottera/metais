// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package scenario reads the UX scenario set of PLAN-2026-10-metais-ux §5:
// one directory per scenario under internal/tools/ux/scenarios, each
// holding scenario.json, its declaration, and prompt.txt, the prompt a
// nested session is given, beside the sources of any fixture packed from
// it.
//
// The package is handed the set as an fs.FS and refers to no host symbol,
// so the scorer may use it with none of the runner's allowance (ADR-009
// Amendment A8 item 4).
package scenario

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"
)

// Kind says which half of R-UX-5's thresholds a scenario counts towards.
type Kind string

const (
	// Consuming scenarios use a package someone else wrote.
	Consuming Kind = "consuming"
	// Authoring scenarios write or edit one.
	Authoring Kind = "authoring"
)

// Fixture is one package the runner packs, with the metais under test,
// into the session's directory before it starts, or one directory of the
// commit under test it copies there. Exactly one of Example, Source and
// Tree is set.
type Fixture struct {
	// Example names a directory of the repository's examples/.
	Example string `json:"example,omitempty"`
	// Source names a directory beside scenario.json.
	Source string `json:"source,omitempty"`
	// Tree names a top-level directory of the commit under test, which
	// the session's directory starts with as the commit holds it: its
	// tracked files and nothing else, unpacked.
	Tree string `json:"tree,omitempty"`
	// As is the file name the package is written under, or for a tree
	// the directory name it is copied to.
	As string `json:"as"`
}

// Package reports whether the fixture is a package the runner packs,
// rather than a tree it copies.
func (f Fixture) Package() bool { return f.Tree == "" }

// Scenario is one scenario's declaration and prompt.
type Scenario struct {
	// ID is the table's name for it: C0 to C5, A1 to A3.
	ID string `json:"id"`
	// Slug is the table's short name; the directory is ID-Slug.
	Slug string `json:"slug"`
	Kind Kind   `json:"kind"`
	// AsksForOutput says whether the prompt asks for output, such as
	// "render…" or "produce the report from…". R-UX-8 applies to exactly
	// the scenarios that do (the Wave 1 brief's ruling, from the amended
	// R-UX-8's wording).
	AsksForOutput bool `json:"asks_for_output"`
	// Fixtures are the packages the session's directory starts with, in
	// order; none for a scenario that starts from an empty directory.
	Fixtures []Fixture `json:"fixtures"`
	// Check is the place reserved for the scenario's outcome check, which
	// the scorer defines (PLAN-2026-10-metais-ux W1-2). This package keeps
	// it as written and does not interpret it.
	Check json.RawMessage `json:"check,omitempty"`
	// Rewordings record each change of the prompt under R-UX-9, oldest
	// first: a prompt may change only to remove ambiguity about what the
	// user wants, and each change is shown in the next scorecard with the
	// transcript that showed the ambiguity.
	Rewordings []Rewording `json:"rewordings,omitempty"`

	// Dir is the scenario's directory in the set.
	Dir string `json:"-"`
	// Prompt is prompt.txt without its final newline.
	Prompt string `json:"-"`
}

// Rewording is one change of a scenario's prompt under R-UX-9.
type Rewording struct {
	// Date is the day of the change, as YYYY-MM-DD.
	Date string `json:"date"`
	// Was is the prompt before the change, verbatim.
	Was string `json:"was"`
	// Ambiguity says what the user's intent was ambiguous about.
	Ambiguity string `json:"ambiguity"`
	// Transcript names the transcript that showed the ambiguity: the
	// scorecard and run it is kept with.
	Transcript string `json:"transcript"`
}

// Name is the scenario's directory name, ID-Slug.
func (s *Scenario) Name() string { return s.ID + "-" + s.Slug }

var (
	idPattern   = regexp.MustCompile(`^[CA][0-9]+$`)
	slugPattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)
	filePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*\.metais$`)
	datePattern = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}$`)
)

// Load reads every scenario of the set fsys holds, one directory per
// scenario at its root, in the order of their names. It reports every
// problem it finds, not the first.
func Load(fsys fs.FS) ([]*Scenario, error) {
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	var (
		set  []*Scenario
		errs []error
	)
	for _, e := range entries {
		if !e.IsDir() {
			errs = append(errs, fmt.Errorf("%s: not a scenario directory", e.Name()))
			continue
		}
		s, err := read(fsys, e.Name())
		if err != nil {
			errs = append(errs, err)
			continue
		}
		set = append(set, s)
	}
	seen := map[string]bool{}
	for _, s := range set {
		if seen[s.ID] {
			errs = append(errs, fmt.Errorf("%s: a second scenario %s", s.Dir, s.ID))
		}
		seen[s.ID] = true
	}
	if len(set) == 0 && len(errs) == 0 {
		errs = append(errs, errors.New("the set holds no scenario"))
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return set, nil
}

// count is how many of vs are not empty.
func count(vs ...string) int {
	n := 0
	for _, v := range vs {
		if v != "" {
			n++
		}
	}
	return n
}

// read reads and checks the scenario in directory dir.
func read(fsys fs.FS, dir string) (*Scenario, error) {
	b, err := fs.ReadFile(fsys, path.Join(dir, "scenario.json"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	var s Scenario
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return nil, fmt.Errorf("%s/scenario.json: %w", dir, err)
	}
	if dec.More() {
		return nil, fmt.Errorf("%s/scenario.json: more than one value", dir)
	}
	s.Dir = dir
	p, err := fs.ReadFile(fsys, path.Join(dir, "prompt.txt"))
	if err != nil {
		return nil, fmt.Errorf("%s: %w", dir, err)
	}
	s.Prompt = strings.TrimSuffix(string(p), "\n")

	var errs []error
	bad := func(format string, args ...any) {
		errs = append(errs, fmt.Errorf("%s: "+format, append([]any{dir}, args...)...))
	}
	if !idPattern.MatchString(s.ID) {
		bad("id %q is not C<n> or A<n>", s.ID)
	}
	if !slugPattern.MatchString(s.Slug) {
		bad("slug %q is not lower-case words joined by hyphens", s.Slug)
	}
	if dir != s.Name() {
		bad("the directory is not named %s, its id and slug", s.Name())
	}
	switch {
	case s.Kind == Consuming && !strings.HasPrefix(s.ID, "C"), s.Kind == Authoring && !strings.HasPrefix(s.ID, "A"):
		bad("kind %s does not match id %s: consuming scenarios are C<n>, authoring A<n>", s.Kind, s.ID)
	case s.Kind != Consuming && s.Kind != Authoring:
		bad("kind %q is neither %s nor %s", s.Kind, Consuming, Authoring)
	}
	if strings.TrimSpace(s.Prompt) == "" {
		bad("prompt.txt is empty")
	}
	if s.Fixtures == nil {
		bad("fixtures is missing: write [] for a scenario that starts from an empty directory")
	}
	var names []string
	for i, f := range s.Fixtures {
		switch {
		case count(f.Example, f.Source, f.Tree) != 1:
			bad("fixture %d names not exactly one of example, source and tree", i)
		case f.Example != "" && !slugPattern.MatchString(f.Example):
			bad("fixture %d: example %q is not a directory name of examples/", i, f.Example)
		case f.Source != "":
			if !slugPattern.MatchString(f.Source) {
				bad("fixture %d: source %q is not a directory name beside scenario.json", i, f.Source)
			} else if st, err := fs.Stat(fsys, path.Join(dir, f.Source, "metais.yaml")); err != nil || !st.Mode().IsRegular() {
				bad("fixture %d: source %s holds no metais.yaml", i, f.Source)
			}
		}
		switch {
		case f.Tree != "" && !slugPattern.MatchString(f.Tree):
			bad("fixture %d: tree %q is not a top-level directory name of the repository", i, f.Tree)
		case f.Tree != "" && !slugPattern.MatchString(f.As):
			bad("fixture %d: as %q is not a directory name", i, f.As)
		case f.Tree == "" && !filePattern.MatchString(f.As):
			bad("fixture %d: as %q is not a file name ending .metais", i, f.As)
		}
		if slices.Contains(names, f.As) {
			bad("fixture %d: a second fixture named %s", i, f.As)
		}
		names = append(names, f.As)
	}
	for i, w := range s.Rewordings {
		if !datePattern.MatchString(w.Date) || strings.TrimSpace(w.Was) == "" || strings.TrimSpace(w.Ambiguity) == "" || strings.TrimSpace(w.Transcript) == "" {
			bad("rewording %d needs a date (YYYY-MM-DD), the prompt it was, the ambiguity, and the transcript that showed it (R-UX-9)", i)
		}
		if w.Was == s.Prompt {
			bad("rewording %d: the prompt it was is the prompt it is", i)
		}
	}
	if len(s.Check) > 0 && !json.Valid(s.Check) {
		bad("check is not JSON")
	}
	if err := errors.Join(errs...); err != nil {
		return nil, err
	}
	return &s, nil
}
