// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

// vector is one vector.json (ADR-011 D9 item 3), with the directory it was
// read from.
type vector struct {
	Schema     string   `json:"$schema"`
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Statements []string `json:"statements"`
	Since      string   `json:"since"`
	Class      string   `json:"class"`
	Tags       []string `json:"tags"`
	Requires   []string `json:"requires"`
	Notes      string   `json:"notes"`
	Cases      []vcase  `json:"cases"`

	dir string // the vector's directory, absolute
}

// vcase is one case of a vector: a setup and an ordered list of steps.
type vcase struct {
	Name  string `json:"name"`
	Setup *setup `json:"setup"`
	Steps []step `json:"steps"`
}

// setup is what a case's work directory holds before its first step (D9
// item 4).
type setup struct {
	Copy     *string   `json:"copy"`
	Dirs     []string  `json:"dirs"`
	Symlinks []symlink `json:"symlinks"`
	Fifos    []string  `json:"fifos"`
	// Unwritable are directories made unwritable to the implementation
	// once everything else is in place: the pre-state of a write that
	// fails (ADR-011 D9 item 6, A2).
	Unwritable []string `json:"unwritable"`
}

type symlink struct {
	Path   string `json:"path"`
	Target string `json:"target"`
}

// step is one operation and what it must give.
type step struct {
	Op         string          `json:"op"`
	Args       json.RawMessage `json:"args"`
	Expect     expect          `json:"expect"`
	Bounds     *bounds         `json:"bounds"`
	Repeatable bool            `json:"repeatable"`

	args stepArgs // Args, decoded
}

// stepArgs are the arguments of ADR-011 D8 item 2's operations. A driver
// receives Args as the vector wrote them; the cli adapter reads these.
type stepArgs struct {
	Package    string                     `json:"package"`
	Source     string                     `json:"source"`
	Data       string                     `json:"data"`
	Set        []string                   `json:"set"`
	SetJSON    []string                   `json:"set_json"`
	InputsJSON string                     `json:"inputs_json"`
	Limits     map[string]json.RawMessage `json:"limits"`
	Out        string                     `json:"out"`
	Stream     json.RawMessage            `json:"stream"`
	Only       stringList                 `json:"only"`
	Overwrite  bool                       `json:"overwrite"`
	DryRun     bool                       `json:"dry_run"`
	FailOn     string                     `json:"fail_on"`
	Now        string                     `json:"now"`
	Sample     *sampleArgs                `json:"sample"`
	Examples   []string                   `json:"examples"`
	To         string                     `json:"to"`
	Example    string                     `json:"example"`
	Optional   json.RawMessage            `json:"optional"`
	Seed       json.RawMessage            `json:"seed"`
	Items      json.RawMessage            `json:"items"`
}

// sampleArgs is render's sample argument.
type sampleArgs struct {
	Optional json.RawMessage `json:"optional"`
	Seed     json.RawMessage `json:"seed"`
	Items    json.RawMessage `json:"items"`
}

// stringList is a string or a list of strings.
type stringList []string

func (l *stringList) UnmarshalJSON(b []byte) error {
	var one string
	if json.Unmarshal(b, &one) == nil {
		*l = stringList{one}
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return err
	}
	*l = many
	return nil
}

// stream returns the step's stream argument: "archive", or the path of the
// one output it streams, or "" when the step has none.
func (a stepArgs) stream() (archive bool, only string, ok bool) {
	if len(a.Stream) == 0 || string(a.Stream) == "null" {
		return false, "", false
	}
	var s string
	if json.Unmarshal(a.Stream, &s) == nil {
		return s == "archive", "", s == "archive"
	}
	var o struct {
		Only string `json:"only"`
	}
	if json.Unmarshal(a.Stream, &o) == nil && o.Only != "" {
		return false, o.Only, true
	}
	return false, "", false
}

// expect is what a step must give (ADR-011 D10).
type expect struct {
	OK       bool           `json:"ok"`
	Findings string         `json:"findings"`
	Trees    []treeExpect   `json:"trees"`
	Stream   *streamExpect  `json:"stream"`
	Listing  *listingExpect `json:"listing"`
}

// listingExpect is a dry run's expected listing: a vector file, and the
// outputs that are containers, of which only the path is compared.
type listingExpect struct {
	Equals     string   `json:"equals"`
	Containers []string `json:"containers"`
}

// UnmarshalJSON reads a listing named by its file alone, or as an object
// that also lists its containers.
func (l *listingExpect) UnmarshalJSON(b []byte) error {
	if json.Unmarshal(b, &l.Equals) == nil {
		return nil
	}
	type plain listingExpect
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	return dec.Decode((*plain)(l))
}

// treeExpect is one declared tree.
type treeExpect struct {
	Path       string          `json:"path"`
	Equals     string          `json:"equals"`
	Files      []string        `json:"files"`
	Containers []containerSpec `json:"containers"`
	Executable []string        `json:"executable"`
	Substitute []string        `json:"substitute"`
}

// containerSpec names a file compared as a zip, and the entries whose
// compression method is compared.
type containerSpec struct {
	Path    string          `json:"path"`
	Methods json.RawMessage `json:"methods"`
}

// methods returns the entries whose method is compared, or all.
func (c containerSpec) methods() (names []string, all bool) {
	var s string
	if json.Unmarshal(c.Methods, &s) == nil {
		return nil, s == "all"
	}
	_ = json.Unmarshal(c.Methods, &names)
	return names, false
}

// streamExpect is the stream a step must write.
type streamExpect struct {
	Equals    string `json:"equals"`
	Container *struct {
		Methods json.RawMessage `json:"methods"`
	} `json:"container"`
}

// bounds lowers a step's resource bounds (ADR-011 D13 item 1).
type bounds struct {
	CPUms  int64 `json:"cpu_ms"`
	RSSMiB int64 `json:"rss_mib"`
}

// readVector reads dir/vector.json strictly.
func readVector(dir string) (*vector, error) {
	b, err := os.ReadFile(filepath.Join(dir, "vector.json"))
	if err != nil {
		return nil, err
	}
	var v vector
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&v); err != nil {
		return nil, fmt.Errorf("vector.json: %w", err)
	}
	for ci := range v.Cases {
		for si := range v.Cases[ci].Steps {
			st := &v.Cases[ci].Steps[si]
			if len(st.Args) == 0 {
				st.Args = json.RawMessage("{}")
			}
			ad := json.NewDecoder(bytes.NewReader(st.Args))
			ad.DisallowUnknownFields()
			if err := ad.Decode(&st.args); err != nil {
				return nil, fmt.Errorf("vector.json: case %q step %d: args: %w", v.Cases[ci].Name, si+1, err)
			}
		}
	}
	v.dir = dir
	return &v, nil
}

// vectorDirs returns every directory under suite/vectors that holds a
// vector.json, sorted, as absolute paths. A missing vectors directory holds
// no vector.
func vectorDirs(suite string) ([]string, error) {
	root := filepath.Join(suite, "vectors")
	var dirs []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == root && errors.Is(err, fs.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		if !d.IsDir() && d.Name() == "vector.json" {
			dirs = append(dirs, filepath.Dir(path))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	slices.Sort(dirs)
	return dirs, nil
}

// vectorPath is dir's path below suite/vectors, with slashes: the vector's
// id when the vector is well formed.
func vectorPath(suite, dir string) string {
	rel, err := filepath.Rel(filepath.Join(suite, "vectors"), dir)
	if err != nil {
		return dir
	}
	return filepath.ToSlash(rel)
}

// frontPageVersion reads the revision the front page beside suite names in
// its status block: the line "Version: M.m" of spec/README.md.
func frontPageVersion(suite string) (string, error) {
	f, err := os.Open(filepath.Join(suite, "..", "README.md"))
	if err != nil {
		return "", err
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if v, ok := strings.CutPrefix(sc.Text(), "Version: "); ok {
			if _, err := parseRevision(v); err != nil {
				return "", fmt.Errorf("front page: %w", err)
			}
			return v, nil
		}
	}
	return "", errors.New("front page: no Version line")
}

// parseRevision reads "M.m" as a comparable pair.
func parseRevision(s string) ([2]int, error) {
	maj, min, ok := strings.Cut(s, ".")
	a, err1 := strconv.Atoi(maj)
	b, err2 := strconv.Atoi(min)
	if !ok || err1 != nil || err2 != nil || a < 0 || b < 0 || strconv.Itoa(a) != maj || strconv.Itoa(b) != min {
		return [2]int{}, fmt.Errorf("revision %q is not MAJOR.MINOR", s)
	}
	return [2]int{a, b}, nil
}

// laterRevision reports whether a is a later revision than b. Malformed
// revisions are never later; the schema refuses them.
func laterRevision(a, b string) bool {
	ra, err1 := parseRevision(a)
	rb, err2 := parseRevision(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return ra[0] > rb[0] || ra[0] == rb[0] && ra[1] > rb[1]
}

// codeInfo is what the runner reads of one registered finding code.
type codeInfo struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Class    string `json:"class"`
}

// readRegistry reads the finding-code registry beside suite, at
// ../findings/registry.json, which the cli adapter's exit check reads each
// code's class from.
func readRegistry(suite string) (map[string]codeInfo, error) {
	b, err := os.ReadFile(filepath.Join(suite, "..", "findings", "registry.json"))
	if err != nil {
		return nil, err
	}
	var reg struct {
		Codes []codeInfo `json:"codes"`
	}
	if err := json.Unmarshal(b, &reg); err != nil {
		return nil, fmt.Errorf("registry: %w", err)
	}
	m := make(map[string]codeInfo, len(reg.Codes))
	for _, c := range reg.Codes {
		m[c.Code] = c
	}
	return m, nil
}
