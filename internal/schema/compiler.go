// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"slices"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schemafail"
)

// IsSchemaFile reports whether entry is a schema file: a JSON or YAML file
// under InputsDir (ADR-003 D3 item 1).
func IsSchemaFile(entry string) bool {
	if !strings.HasPrefix(entry, InputsDir) {
		return false
	}
	switch path.Ext(entry) {
	case ".json", ".yaml", ".yml":
		return true
	}
	return false
}

// Decode reads the bytes of a schema file into the data model by the
// rules of its form (ADR-003 D9, D10), reporting what it refuses. The
// package loader supplies it; the compiler never parses a file itself.
type Decode func(entry string, src []byte) (value.Value, []finding.Finding)

// Compiler holds every schema file of a package, compiled.
type Compiler struct {
	lib     *jsonschema.Compiler
	lim     limits.Set
	meter   *regexprofile.Meter
	schemas map[string]*Schema     // by entry
	docs    map[string]value.Value // each file as authored, by entry
	roots   map[string]root        // by the library's root URL
	anchors map[string]root        // by the anchor's URI
	refs    map[root]string        // each $ref's resolved URI, by the schema object holding it
	dynRefs map[root]string        // each $dynamicRef's resolved URI, by the schema object holding it
	subs    map[root]*Schema       // each subschema Subschema compiled
}

// root is where a resource the library holds as a root document is in the
// package: its file, and its pointer within the file.
type root struct {
	entry, ptr string
}

// Schema is one compiled schema file.
type Schema struct {
	c   *Compiler
	lib *jsonschema.Schema
}

// file is one schema file, decoded and walked.
type file struct {
	entry     string
	retrieval string
	canonical string
	hasID     bool
	doc       value.Value
	walk      *prechecker
}

// NewCompiler compiles every schema file of tree (ADR-003 D3 item 1) under
// lim. It is the one call site of jsonschema.NewCompiler in the module
// (ADR-010 D5 item 1), and configures the library closed:
//
//   - the draft is 2020-12;
//   - the loader refuses every URL, so nothing is ever fetched or opened;
//     every schema file is added under its retrieval URI and, where its
//     root declares $id, its canonical URI, and every embedded resource
//     under its own;
//   - formats are asserted, each of ADR-003 D4's with METAIS's validator;
//   - patterns are compiled by the package regular-expression profile.
//
// Before the library sees a file, NewCompiler checks schema-size, then
// decodes it with decode and refuses unknown keywords, formats outside
// ADR-003 D4, any $schema but 2020-12's and any $vocabulary, patterns
// outside the profile, schema-depth, and every reference that leaves the
// package, escapes InputsDir, names a file by its path where it has an $id, or
// resolves to nothing; and two resources with one URI. Every finding of
// every file is reported at once. When there is any, the compiler is nil.
func NewCompiler(tree pkgtree.Tree, lim limits.Set, decode Decode) (*Compiler, []finding.Finding) {
	c := &Compiler{
		lim:     lim,
		meter:   regexprofile.NewMeter(lim.Value(limits.RegexWork)),
		schemas: map[string]*Schema{},
		docs:    map[string]value.Value{},
		roots:   map[string]root{},
		anchors: map[string]root{},
		refs:    map[root]string{},
		dynRefs: map[root]string{},
		subs:    map[root]*Schema{},
	}
	var files []*file
	var findings []finding.Finding
	for _, entry := range tree.List() {
		if !IsSchemaFile(entry) {
			continue
		}
		f, fs := c.read(tree, entry, decode)
		findings = append(findings, fs...)
		if f != nil {
			files = append(files, f)
		}
	}
	findings = append(findings, checkResources(files)...)
	if len(findings) > 0 {
		return nil, findings
	}

	lib := jsonschema.NewCompiler()
	c.lib = lib
	lib.DefaultDraft(jsonschema.Draft2020)
	lib.UseLoader(refuseLoader{})
	lib.AssertFormat()
	for _, name := range Formats() {
		lib.RegisterFormat(&jsonschema.Format{Name: name, Validate: formatCheck(formats[name])})
	}
	lib.UseRegexpEngine(func(p string) (jsonschema.Regexp, error) {
		return regexprofile.Compile(p, lim, c.meter)
	})
	for _, f := range files {
		c.docs[f.entry] = f.doc
		for _, a := range f.walk.anchors {
			c.anchors[a.uri] = root{f.entry, a.ptr}
		}
		c.recordRefs(f.entry, f.walk.refs)
		if err := c.add(lib, f); err != nil {
			// The pre-check refused every URI the library could not take.
			panic(err)
		}
	}
	for _, f := range files {
		s, err := lib.Compile(f.retrieval)
		if err != nil {
			fs, ierr := c.compileFindings(f.entry, err)
			if ierr != nil {
				panic(ierr)
			}
			findings = append(findings, fs...)
			continue
		}
		c.schemas[f.entry] = &Schema{c: c, lib: s}
	}
	if len(findings) > 0 {
		return nil, findings
	}
	return c, nil
}

// read checks schema-size, decodes entry and walks it.
func (c *Compiler) read(tree pkgtree.Tree, entry string, decode Decode) (*file, []finding.Finding) {
	info, err := tree.Stat(entry)
	if err != nil {
		// tree.List named it; a Tree answers the same way on every call.
		panic(err)
	}
	if info.Size > c.lim.Value(limits.SchemaSize) {
		loc := finding.Location{Origin: finding.OriginPackage, File: entry}
		return nil, []finding.Finding{c.lim.Breached(limits.SchemaSize, loc, info.Size, true)}
	}
	r, err := tree.Open(entry)
	if err != nil {
		panic(err)
	}
	src, err := io.ReadAll(r)
	r.Close()
	if err != nil {
		panic(err)
	}
	doc, fs := decode(entry, src)
	if len(fs) > 0 {
		return nil, fs
	}
	if doc == nil {
		panic("schema: Decode gave no value and no finding for " + entry)
	}
	f := &file{entry: entry, retrieval: RetrievalURI(entry), doc: doc}
	base, _ := url.Parse(f.retrieval)
	f.walk = &prechecker{entry: entry, lim: c.lim}
	f.walk.walk(doc, "", base, 1)
	f.canonical = f.retrieval
	if rs := f.walk.resources; len(rs) > 0 && rs[0].ptr == "" {
		f.canonical, f.hasID = rs[0].uri, true
		f.walk.resources = rs[1:]
	}
	return f, f.walk.findings
}

// owner is where a resource is declared.
type owner struct{ entry, ptr string }

// builtIn reports whether uri is one the library holds built in, a
// meta-schema of some draft, which no package resource can take.
func builtIn(uri string) bool {
	return strings.HasPrefix(uri, "https://json-schema.org/") || strings.HasPrefix(uri, "http://json-schema.org/")
}

// checkResources is the pre-check across files: two resources with one
// URI (ADR-003 D5 item 6), and every reference's target (D5 items 3 to 5).
func checkResources(files []*file) []finding.Finding {
	var findings []finding.Finding
	byRetrieval := map[string]*file{}
	for _, f := range files {
		byRetrieval[f.retrieval] = f
	}
	known := map[string]owner{}
	claim := func(uri string, o owner, self *file) {
		prior, taken := known[uri]
		if g, ok := byRetrieval[uri]; ok && g != self && !taken {
			prior, taken = owner{g.entry, ""}, true
		}
		p := finding.Params{"id": fstr(uri)}
		switch {
		case taken:
			p["other"] = fstr(prior.entry)
		case builtIn(uri):
		default:
			known[uri] = o
			return
		}
		findings = append(findings, report(CodeIDDuplicate, o.entry, o.ptr+"/$id", p))
	}
	for _, f := range files {
		if f.hasID {
			claim(f.canonical, owner{f.entry, ""}, f)
		} else {
			known[f.retrieval] = owner{f.entry, ""}
		}
		for _, r := range f.walk.resources {
			claim(r.uri, owner{f.entry, r.ptr}, f)
		}
	}
	for _, f := range files {
		for _, r := range f.walk.refs {
			if code, p := checkRef(r, known, byRetrieval); code != "" {
				p["reference"] = fstr(r.text)
				findings = append(findings, report(code, f.entry, r.ptr, p))
			}
		}
	}
	return findings
}

// checkRef returns the finding's code and parameters for a reference that
// does not resolve inside the package, or "" for one that does.
func checkRef(r reference, known map[string]owner, byRetrieval map[string]*file) (finding.Code, finding.Params) {
	u, err := url.Parse(r.text)
	if err != nil {
		return CodeRefUnresolved, finding.Params{}
	}
	if u.Scheme == "" && u.Host == "" && u.Path != "" && r.base.Scheme == "metais" && climbs(r.base.Path, u.Path) {
		return CodeRefEscapes, finding.Params{}
	}
	t := r.base.ResolveReference(u)
	t.Fragment, t.RawFragment = "", ""
	target := t.String()
	if _, ok := known[target]; ok {
		if t.Scheme == "metais" && !strings.HasPrefix(t.Path, "/"+InputsDir) {
			return CodeRefEscapes, finding.Params{}
		}
		return "", nil
	}
	if metaSchemas[target] {
		return "", nil
	}
	if t.Scheme == "metais" {
		if !strings.HasPrefix(t.Path, "/"+InputsDir) {
			return CodeRefEscapes, finding.Params{}
		}
		if f, ok := byRetrieval[target]; ok && f.hasID {
			return CodeRefByPath, finding.Params{"id": fstr(f.canonical)}
		}
	}
	return CodeRefUnresolved, finding.Params{}
}

// recordRefs records the resolved URI of each $ref and $dynamicRef of
// refs, the references the pre-check found in entry.
func (c *Compiler) recordRefs(entry string, refs []reference) {
	for _, r := range refs {
		u, err := url.Parse(r.text)
		if err != nil {
			continue
		}
		if at, ok := strings.CutSuffix(r.ptr, "/$ref"); ok {
			c.refs[root{entry, at}] = r.base.ResolveReference(u).String()
		} else if at, ok := strings.CutSuffix(r.ptr, "/$dynamicRef"); ok {
			c.dynRefs[root{entry, at}] = r.base.ResolveReference(u).String()
		}
	}
}

// add registers f with the library under its retrieval URI, its canonical
// URI and each embedded resource's URI.
func (c *Compiler) add(lib *jsonschema.Compiler, f *file) error {
	doc := toLibrary(f.doc)
	if err := lib.AddResource(f.retrieval, doc); err != nil {
		return err
	}
	c.roots[f.retrieval] = root{f.entry, ""}
	if f.canonical != f.retrieval {
		if err := lib.AddResource(f.canonical, doc); err != nil {
			return err
		}
		c.roots[f.canonical] = root{f.entry, ""}
	}
	for _, r := range f.walk.resources {
		// An embedded resource is a root of its own as well, so that a
		// reference from another file finds it; its $id is written
		// absolute, as its place in its file resolved it.
		sub := toLibrary(r.v).(map[string]any)
		own := make(map[string]any, len(sub))
		for k, v := range sub {
			own[k] = v
		}
		own["$id"] = r.uri
		if err := lib.AddResource(r.uri, own); err != nil {
			return err
		}
		c.roots[r.uri] = root{f.entry, r.ptr}
	}
	return nil
}

// refuseLoader is the library's loader: every schema it may reach was
// added, so any URL it is asked for is outside the package, and refused
// (ADR-010 D5 item 2). The library's FileLoader is never installed.
type refuseLoader struct{}

// ErrRefused is the loader's refusal.
var ErrRefused = errors.New("schema: nothing is loaded from outside the package")

func (refuseLoader) Load(string) (any, error) { return nil, ErrRefused }

// Schema returns the compiled schema file entry, and false when entry is
// not a schema file of the package.
func (c *Compiler) Schema(entry string) (*Schema, bool) {
	s, ok := c.schemas[entry]
	return s, ok
}

// Validate validates v against s. It returns the failures beneath the
// schema's root, each a schemafail tree in a fixed order that owes
// nothing to the library's (ADR-010 D5 item 5), or none when v is valid.
// It returns a *regexprofile.LimitError naming regex-work, and no
// failures, once the command's matching budget is spent: a pattern's
// answer can no longer be had, so no answer is given.
func (s *Schema) Validate(v value.Value) ([]schemafail.Failure, error) {
	err := s.lib.Validate(toLibrary(v))
	if s.c.meter.Exhausted() {
		return nil, &regexprofile.LimitError{Limit: limits.RegexWork}
	}
	if err == nil {
		return nil, nil
	}
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		return nil, fmt.Errorf("schema: the library's validation gave %T", err)
	}
	return s.c.adapt(ve)
}

// locate splits a URL the library gives, a root's URL and a fragment, into
// the package entry and the pointer within it.
func (c *Compiler) locate(u string) (entry, ptr string, ok bool) {
	base, frag, _ := strings.Cut(u, "#")
	r, ok := c.roots[base]
	if !ok {
		return "", "", false
	}
	if frag != "" && !strings.HasPrefix(frag, "/") {
		// An anchor names no pointer.
		return r.entry, r.ptr, true
	}
	entry, ptr = c.embedded(r.entry, r.ptr+unescapePointer(frag))
	return entry, ptr, true
}

// compileFindings maps an error of the library's Compile, which the
// pre-check left it to find, to findings about entry. An error of a type
// it does not know is returned, as a defect in the tool.
func (c *Compiler) compileFindings(entry string, err error) ([]finding.Finding, error) {
	where := func(u string) (string, string) {
		if e, p, ok := c.locate(u); ok {
			return e, p
		}
		return entry, ""
	}
	var (
		sve   *jsonschema.SchemaValidationError
		load  *jsonschema.LoadURLError
		anc   *jsonschema.AnchorNotFoundError
		ptr   *jsonschema.JSONPointerNotFoundError
		iptr  *jsonschema.InvalidJsonPointerError
		purl  *jsonschema.ParseURLError
		dupID *jsonschema.DuplicateIDError
		dupA  *jsonschema.DuplicateAnchorError
		re    *jsonschema.InvalidRegexError
		pid   *jsonschema.ParseIDError
		panc  *jsonschema.ParseAnchorError
		vocab *jsonschema.UnsupportedVocabularyError
		draft *jsonschema.UnsupportedDraftError
		cycle *jsonschema.MetaSchemaCycleError
		mism  *jsonschema.MetaSchemaMismatchError
		murl  *jsonschema.InvalidMetaSchemaURLError
	)
	switch {
	case errors.As(err, &sve):
		var ve *jsonschema.ValidationError
		if !errors.As(sve.Err, &ve) {
			return nil, fmt.Errorf("schema: a schema validation error holds %T", sve.Err)
		}
		e, base := where(sve.URL)
		fails, aerr := c.adaptMeta(ve)
		if aerr != nil {
			return nil, aerr
		}
		var out []finding.Finding
		for _, f := range leaves(fails) {
			out = append(out, report(CodeInvalid, e, base+f.Instance(), finding.Params{"keyword": fstr(f.Keyword)}))
		}
		return out, nil
	case errors.As(err, &load):
		return []finding.Finding{report(CodeRefUnresolved, entry, "", finding.Params{"reference": fstr(load.URL)})}, nil
	case errors.As(err, &anc):
		e, p := where(anc.URL)
		return []finding.Finding{report(CodeRefUnresolved, e, p, finding.Params{"reference": fstr(anc.Reference)})}, nil
	case errors.As(err, &ptr):
		return []finding.Finding{report(CodeRefUnresolved, entry, "", finding.Params{"reference": fstr(ptr.URL)})}, nil
	case errors.As(err, &iptr):
		return []finding.Finding{report(CodeRefUnresolved, entry, "", finding.Params{"reference": fstr(iptr.URL)})}, nil
	case errors.As(err, &purl):
		return []finding.Finding{report(CodeRefUnresolved, entry, "", finding.Params{"reference": fstr(purl.URL)})}, nil
	case errors.As(err, &dupID):
		e, base := where(dupID.URL)
		first, _ := where(dupID.URL + "#" + dupID.Ptr1)
		return []finding.Finding{report(CodeIDDuplicate, e, base+dupID.Ptr2, finding.Params{"id": fstr(dupID.ID), "other": fstr(first)})}, nil
	case errors.As(err, &dupA):
		e, base := where(dupA.URL)
		return []finding.Finding{report(CodeIDDuplicate, e, base+dupA.Ptr2, finding.Params{"id": fstr("#" + dupA.Anchor), "other": fstr(e)})}, nil
	case errors.As(err, &pid):
		e, p := where(pid.URL)
		return []finding.Finding{report(CodeInvalid, e, p+"/$id", finding.Params{"keyword": fstr("$id")})}, nil
	case errors.As(err, &panc):
		e, p := where(panc.URL)
		return []finding.Finding{report(CodeInvalid, e, p+"/$anchor", finding.Params{"keyword": fstr("$anchor")})}, nil
	case errors.As(err, &vocab), errors.As(err, &draft), errors.As(err, &cycle), errors.As(err, &mism), errors.As(err, &murl):
		// The pre-check refuses every $schema but 2020-12's, and every
		// $vocabulary, so none of these is reached.
		return []finding.Finding{report(CodeDialect, entry, "", finding.Params{"reason": fstr(dialectSchema)})}, nil
	case errors.As(err, &re):
		e, p := where(re.URL)
		if f, ok := regexFinding(re.Err, e, p, c.lim); ok {
			return []finding.Finding{f}, nil
		}
	}
	return nil, fmt.Errorf("schema: compiling %s gave %T", entry, err)
}

// leaves returns the failures of fs with no children, in order.
func leaves(fs []schemafail.Failure) []schemafail.Failure {
	var out []schemafail.Failure
	for i := range fs {
		fs[i].Walk(func(_ []*schemafail.Failure, f *schemafail.Failure) bool {
			if len(f.Children) == 0 {
				out = append(out, *f)
			}
			return true
		})
	}
	return slices.CompactFunc(out, func(a, b schemafail.Failure) bool {
		return a.Keyword == b.Keyword && a.Instance() == b.Instance()
	})
}
