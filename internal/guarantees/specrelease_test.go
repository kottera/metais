// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bufio"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

// The tripwires over what a release fixes: TestReleasedFilesUnchanged,
// TestChangelog and TestSchemasPublished (ADR-011 D5, D7 and D14).

// releasedDir holds each released revision's digest list, below spec/
// (ADR-011 D5 item 2).
const releasedDir = "conformance/released"

var (
	releasedNameRe = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.sha256$`)
	sha256LineRe   = regexp.MustCompile(`^([0-9a-f]{64})  (\S+)$`)
	// erratumFileRe is an erratum's line naming a released file and its new
	// digest (ADR-011 D6 item 2), with the "- Files:" lead on the first.
	erratumFileRe = regexp.MustCompile(`^(?:- Files:)? *(\S+) sha256:([0-9a-f]{64})$`)
)

func sha256Hex(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// releasedProblems is TestReleasedFilesUnchanged's check: every file a
// released list names, by its path below spec/, still has the listed
// digest, or ERRATA.md names the file with its current digest.
func releasedProblems(root string) []string {
	dir := filepath.Join(root, filepath.FromSlash(releasedDir))
	entries, err := os.ReadDir(dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		return []string{err.Error()}
	}
	errata := map[string]bool{} // "path sha256"
	if b, err := os.ReadFile(filepath.Join(root, "ERRATA.md")); err == nil {
		for _, l := range strings.Split(string(b), "\n") {
			if m := erratumFileRe.FindStringSubmatch(strings.TrimRight(l, " ")); m != nil {
				errata[m[1]+" "+m[2]] = true
			}
		}
	} else {
		return []string{err.Error()}
	}
	var problems []string
	for _, e := range entries {
		name := e.Name()
		if !releasedNameRe.MatchString(name) || !e.Type().IsRegular() {
			problems = append(problems, fmt.Sprintf("spec/%s/%s is not a released revision's digest list, <MAJOR>.<MINOR>.sha256", releasedDir, name))
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return append(problems, err.Error())
		}
		prev := ""
		sc := bufio.NewScanner(bytes.NewReader(b))
		for n := 1; sc.Scan(); n++ {
			where := fmt.Sprintf("spec/%s/%s:%d", releasedDir, name, n)
			m := sha256LineRe.FindStringSubmatch(sc.Text())
			if m == nil {
				problems = append(problems, where+": not a line of sha256sum's format")
				continue
			}
			file := m[2]
			if file <= prev {
				problems = append(problems, where+": "+file+" is out of order or listed twice")
			}
			prev = file
			if !filepath.IsLocal(filepath.FromSlash(file)) {
				problems = append(problems, where+": "+file+" is not a path below spec/")
				continue
			}
			got, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file)))
			if err != nil {
				problems = append(problems, fmt.Sprintf("%s: the released file %s: %v", where, file, err))
				continue
			}
			if d := sha256Hex(got); d != m[1] && !errata[file+" "+d] {
				problems = append(problems, fmt.Sprintf("%s: the released file %s has changed, to sha256:%s, and no erratum names it", where, file, d))
			}
		}
	}
	return problems
}

// changelogSection is one revision's section of CHANGELOG.md.
type changelogSection struct {
	version string
	date    string // empty while unreleased
	line    int
	added   []string
	body    []string
}

var (
	sectionRe = regexp.MustCompile(`^## (\S+) — (unreleased|[0-9]{4}-[0-9]{2}-[0-9]{2})$`)
	addedRe   = regexp.MustCompile(`^- (MTS-[A-Z]{3}-[0-9]{3,}) — `)
	// The parts a revision's section may have (ADR-011 D5 item 7).
	changelogParts = []string{"Added", "Errata applied", "Editorial"}
)

func readChangelog(b []byte) ([]changelogSection, []string) {
	var secs []changelogSection
	var problems []string
	part := ""
	for i, l := range strings.Split(string(b), "\n") {
		where := fmt.Sprintf("CHANGELOG.md:%d", i+1)
		switch {
		case strings.HasPrefix(l, "## "):
			m := sectionRe.FindStringSubmatch(l)
			if m == nil || !versionRe.MatchString(m[1]) {
				problems = append(problems, where+": a section heading is not \"## <MAJOR>.<MINOR> — unreleased\" or \"— <date>\"")
				continue
			}
			s := changelogSection{version: m[1], line: i + 1}
			if m[2] != "unreleased" {
				s.date = m[2]
			}
			secs = append(secs, s)
			part = ""
		case strings.HasPrefix(l, "### "):
			part = strings.TrimPrefix(l, "### ")
			if !slices.Contains(changelogParts, part) {
				problems = append(problems, fmt.Sprintf("%s: %q is not a part of a revision's section", where, part))
			}
			if len(secs) == 0 {
				problems = append(problems, where+": a part outside any revision's section")
			}
		case len(secs) > 0:
			s := &secs[len(secs)-1]
			s.body = append(s.body, l)
			if part == "Added" && strings.HasPrefix(l, "- ") {
				m := addedRe.FindStringSubmatch(l)
				if m == nil {
					problems = append(problems, where+": an Added line is not \"- <statement> — <what it adds>\"")
					continue
				}
				s.added = append(s.added, m[1])
			}
		}
	}
	return secs, problems
}

// versionLess orders MAJOR.MINOR versions.
func versionLess(a, b string) int {
	pa, pb := strings.Split(a, "."), strings.Split(b, ".")
	for i := range 2 {
		x, _ := strconv.Atoi(pa[i])
		y, _ := strconv.Atoi(pb[i])
		if x != y {
			return x - y
		}
	}
	return 0
}

// frontBlock reads the status block at the top of README.md (ADR-011 D1
// item 5).
func frontBlock(root string) (map[string]string, error) {
	d, err := readDoc(root, "README.md")
	if err != nil {
		return nil, err
	}
	if len(d.fences) == 0 || d.fences[0].open != 0 {
		return nil, errors.New("README.md does not open with its status block")
	}
	block := map[string]string{}
	for _, l := range strings.Split(strings.TrimSuffix(d.fences[0].body(d), "\n"), "\n") {
		k, v, ok := strings.Cut(l, ": ")
		if !ok || !slices.Contains([]string{"Version", "Status", "Latest release"}, k) || block[k] != "" {
			return nil, fmt.Errorf("README.md: the status block's line %q is not Version, Status or Latest release", l)
		}
		block[k] = strings.TrimSpace(v)
	}
	if len(block) != 3 {
		return nil, errors.New("README.md: the status block lacks Version, Status or Latest release")
	}
	return block, nil
}

// changelogProblems is TestChangelog's check: the sections are newest
// first, only the newest may be unreleased, a minor's Added lists exactly
// the statements whose since is that minor, every since names a section,
// and the front page's block agrees with the newest section and the
// newest release.
func changelogProblems(root string, stmts []statement) []string {
	b, err := os.ReadFile(filepath.Join(root, "CHANGELOG.md"))
	if err != nil {
		return []string{err.Error()}
	}
	secs, problems := readChangelog(b)
	if len(secs) == 0 {
		return append(problems, "CHANGELOG.md has no revision's section")
	}
	since := map[string][]string{}
	for _, s := range stmts {
		since[s.Since] = append(since[s.Since], s.ID)
	}
	versions := map[string]bool{}
	for i, s := range secs {
		where := fmt.Sprintf("CHANGELOG.md:%d", s.line)
		versions[s.version] = true
		if i > 0 && versionLess(secs[i-1].version, s.version) <= 0 {
			problems = append(problems, fmt.Sprintf("%s: %s comes after %s; the sections are newest first, each once", where, s.version, secs[i-1].version))
		}
		if i > 0 && s.date == "" {
			problems = append(problems, fmt.Sprintf("%s: %s is unreleased, and only the newest section may be", where, s.version))
		}
		if strings.HasSuffix(s.version, ".0") {
			if !slices.Contains(s.body, "First revision.") || len(s.added) > 0 {
				problems = append(problems, fmt.Sprintf("%s: %s opens its major, and says \"First revision.\" and lists nothing Added", where, s.version))
			}
			continue
		}
		want := slices.Clone(since[s.version])
		got := slices.Clone(s.added)
		slices.SortFunc(want, statementLess)
		slices.SortFunc(got, statementLess)
		for _, id := range got {
			if !slices.Contains(want, id) {
				problems = append(problems, fmt.Sprintf("%s: %s lists %s as Added, and its since is not %s", where, s.version, id, s.version))
			}
		}
		for _, id := range want {
			if !slices.Contains(got, id) {
				problems = append(problems, fmt.Sprintf("%s: %s is since %s and missing from its Added", where, id, s.version))
			}
		}
		if len(slices.Compact(slices.Clone(got))) != len(got) {
			problems = append(problems, fmt.Sprintf("%s: %s's Added lists a statement twice", where, s.version))
		}
	}
	for v, ids := range since {
		if !versions[v] {
			problems = append(problems, fmt.Sprintf("%s are since %s, which CHANGELOG.md has no section for", strings.Join(ids, ", "), v))
		}
	}
	block, err := frontBlock(root)
	if err != nil {
		return append(problems, err.Error())
	}
	top := secs[0]
	if block["Version"] != top.version {
		problems = append(problems, fmt.Sprintf("README.md says Version: %s, and the newest section of CHANGELOG.md is %s", block["Version"], top.version))
	}
	wantStatus := "Draft"
	if top.date != "" {
		wantStatus = fmt.Sprintf("Released %s, tag spec-%s", top.date, top.version)
	}
	if block["Status"] != wantStatus {
		problems = append(problems, fmt.Sprintf("README.md says Status: %s, and CHANGELOG.md gives %s", block["Status"], wantStatus))
	}
	wantLatest := "none"
	for _, s := range secs {
		if s.date != "" {
			wantLatest = fmt.Sprintf("%s, tag spec-%s", s.version, s.version)
			break
		}
	}
	if block["Latest release"] != wantLatest {
		problems = append(problems, fmt.Sprintf("README.md says Latest release: %s, and CHANGELOG.md gives %s", block["Latest release"], wantLatest))
	}
	return problems
}

const (
	// schemasDir holds the published schemas, below spec/ (ADR-011 D7).
	schemasDir = "schemas"
	// schemaBase is the $id of the published schema tree.
	schemaBase  = "https://metais.org/schemas/"
	draft202012 = "https://json-schema.org/draft/2020-12/schema"
	// embeddedSchemas is the tool's copy of spec/schemas/, from this
	// package's directory (ADR-010 D2 item 4).
	embeddedSchemas = "../specdata/schemas"
)

// noFetch refuses every URL: a published schema is compiled alone, so an
// external $ref fails, and nothing is fetched (ADR-011 D7 item 7).
type noFetch struct{}

func (noFetch) Load(u string) (any, error) { return nil, errors.New("not fetched: " + u) }

// indentProblem reports the first line of b that is not indented two
// spaces per level of the JSON value it is in.
func indentProblem(b []byte) string {
	depth, line := 0, 1
	inStr, esc, start := false, false, true
	for i := 0; i < len(b); i++ {
		c := b[i]
		if start {
			start = false
			n := 0
			for i+n < len(b) && b[i+n] == ' ' {
				n++
			}
			if j := i + n; j < len(b) && b[j] != '\n' {
				want := depth
				if b[j] == '}' || b[j] == ']' {
					want--
				}
				if n != 2*want {
					return fmt.Sprintf("line %d is indented %d spaces, not %d", line, n, 2*want)
				}
			}
		}
		switch {
		case inStr:
			if esc {
				esc = false
			} else if c == '\\' {
				esc = true
			} else if c == '"' {
				inStr = false
			}
		case c == '"':
			inStr = true
		case c == '{' || c == '[':
			depth++
		case c == '}' || c == ']':
			depth--
		}
		if c == '\n' {
			line++
			start = true
		}
	}
	return ""
}

// externalRefs returns every $ref and $dynamicRef of doc that resolves
// outside the document: to a URI whose part before the fragment is
// neither its $id nor an $id embedded in it.
func externalRefs(doc any) []string {
	ids := map[string]bool{}
	type ref struct {
		base *url.URL
		val  string
	}
	var refs []ref
	var walk func(v any, base *url.URL)
	walk = func(v any, base *url.URL) {
		switch v := v.(type) {
		case map[string]any:
			if id, ok := v["$id"].(string); ok {
				if u, err := url.Parse(id); err == nil {
					if base != nil {
						u = base.ResolveReference(u)
					}
					u.Fragment = ""
					base = u
					ids[u.String()] = true
				}
			}
			for k, m := range v {
				switch k {
				case "$ref", "$dynamicRef":
					if s, ok := m.(string); ok {
						refs = append(refs, ref{base, s})
						continue
					}
				case "const", "enum", "default", "examples":
					continue // data, not schemas
				}
				walk(m, base)
			}
		case []any:
			for _, m := range v {
				walk(m, base)
			}
		}
	}
	walk(doc, nil)
	var out []string
	for _, r := range refs {
		u, err := url.Parse(r.val)
		if err == nil && r.base != nil {
			u = r.base.ResolveReference(u)
		}
		if err != nil || u == nil {
			out = append(out, r.val)
			continue
		}
		u.Fragment = ""
		if !ids[u.String()] {
			out = append(out, r.val)
		}
	}
	slices.Sort(out)
	return out
}

// schemaFileProblems checks one published schema, at rel below
// spec/schemas/, against ADR-011 D7 item 4.
func schemaFileProblems(rel string, b []byte) []string {
	where := "spec/" + schemasDir + "/" + rel
	var problems []string
	if !strings.HasSuffix(rel, ".schema.json") {
		problems = append(problems, where+": a file under spec/schemas/ is a schema named <subject>.schema.json")
	}
	if len(b) == 0 || b[len(b)-1] != '\n' || bytes.HasSuffix(b, []byte("\n\n")) || bytes.ContainsAny(b, "\r\t") {
		problems = append(problems, where+": does not end in one LF, or holds a CR or a tab")
	}
	if p := indentProblem(b); p != "" {
		problems = append(problems, where+": "+p)
	}
	var head struct {
		Schema string `json:"$schema"`
		ID     string `json:"$id"`
	}
	if err := json.Unmarshal(b, &head); err != nil {
		return append(problems, fmt.Sprintf("%s: %v", where, err))
	}
	if head.Schema != draft202012 {
		problems = append(problems, fmt.Sprintf("%s: $schema is %q, not %q", where, head.Schema, draft202012))
	}
	if want := schemaBase + rel; head.ID != want {
		problems = append(problems, fmt.Sprintf("%s: $id is %q, not %q", where, head.ID, want))
	}
	doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return append(problems, fmt.Sprintf("%s: %v", where, err))
	}
	if refs := externalRefs(doc); len(refs) > 0 {
		problems = append(problems, fmt.Sprintf("%s: the external $ref %q", where, refs))
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(noFetch{})
	id := schemaBase + rel
	if err := c.AddResource(id, doc); err != nil {
		return append(problems, fmt.Sprintf("%s: %v", where, err))
	}
	if _, err := c.Compile(id); err != nil {
		problems = append(problems, fmt.Sprintf("%s: does not compile against the 2020-12 meta-schema: %v", where, err))
	}
	return problems
}

// treeFiles returns every regular file below dir by its slash path, and
// none when dir does not exist.
func treeFiles(dir string) (map[string][]byte, error) {
	files := map[string][]byte{}
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if errors.Is(err, fs.ErrNotExist) && p == dir {
			return filepath.SkipAll
		}
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			files[filepath.ToSlash(rel)] = nil
			return nil
		}
		b, err := os.ReadFile(p)
		files[filepath.ToSlash(rel)] = b
		return err
	})
	return files, err
}

// schemaProblems is TestSchemasPublished's check: every file under
// spec/schemas/ is a schema published as ADR-011 D7 item 4 says, and
// every embedded copy under embedded is its published file, byte for
// byte (D7 item 6).
func schemaProblems(root, embedded string) []string {
	published, err := treeFiles(filepath.Join(root, schemasDir))
	if err != nil {
		return []string{err.Error()}
	}
	if len(published) == 0 {
		return []string{"spec/" + schemasDir + " holds no schema"}
	}
	var problems []string
	for _, rel := range slices.Sorted(maps.Keys(published)) {
		if published[rel] == nil {
			problems = append(problems, "spec/"+schemasDir+"/"+rel+": not a regular file")
			continue
		}
		problems = append(problems, schemaFileProblems(rel, published[rel])...)
	}
	copies, err := treeFiles(embedded)
	if err != nil {
		return append(problems, err.Error())
	}
	for _, rel := range slices.Sorted(maps.Keys(copies)) {
		want, ok := published[rel]
		switch {
		case !ok:
			problems = append(problems, fmt.Sprintf("the embedded schema %s has no published file spec/%s/%s", rel, schemasDir, rel))
		case !bytes.Equal(copies[rel], want):
			problems = append(problems, fmt.Sprintf("the embedded schema %s differs from spec/%s/%s; run go generate ./internal/specdata", rel, schemasDir, rel))
		}
	}
	return problems
}

// TestReleasedFilesUnchanged is ADR-011 D14's tripwire over spec/.
func TestReleasedFilesUnchanged(t *testing.T) {
	report(t, releasedProblems(specRoot))
}

// TestChangelog is ADR-011 D14's tripwire over spec/.
func TestChangelog(t *testing.T) {
	stmts, _ := specStatements(mustDocs(t, specRoot))
	report(t, changelogProblems(specRoot, stmts))
}

// TestSchemasPublished is ADR-011 D14's tripwire over spec/schemas/ and
// the tool's embedded copies.
func TestSchemasPublished(t *testing.T) {
	report(t, schemaProblems(specRoot, embeddedSchemas))
}
