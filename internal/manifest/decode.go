// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"regexp"
	"slices"
	"strconv"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/value"
)

// The names of the rules the decoder judges, which are the rules the
// manifest schema states. A Violation names one of these, or a code of
// Gate's or Check's.
const (
	ruleType        = "type"
	ruleRequired    = "required"
	ruleUnknown     = "unknown_member"
	ruleConst       = "const"
	ruleEnum        = "enum"
	ruleLength      = "length"
	ruleLine        = "line"
	ruleProse       = "prose"
	ruleName        = "name"
	ruleVersion     = "semver"
	ruleEmail       = "email"
	ruleURI         = "http_uri"
	ruleIdentifier  = "identifier"
	ruleReserved    = "reserved_name"
	rulePath        = "path"
	ruleItems       = "items"
	ruleUnique      = "unique_items"
	ruleCSVForms    = "csv_without_csv_form"
	ruleForm        = "one_form"
	ruleForEachPath = "for_each_without_path"
	ruleMimetype    = "mimetype"
)

// reservedInputNames are the input names Liquid gives a meaning (ADR-003
// D1 item 3).
var reservedInputNames = []string{"blank", "empty", "false", "first", "last", "nil", "null", "size", "true"}

// decoder turns a value tree into the model, recording every rule of the
// manifest schema the tree breaks. It judges each rule as the schema does,
// with the schema's own patterns (patterns.go), so the two agree.
type decoder struct {
	violations []Violation
}

func (d *decoder) fail(ptr, rule string) {
	d.violations = append(d.violations, Violation{ptr, rule})
}

// object is one object of the manifest being decoded.
type object struct {
	d   *decoder
	o   value.Object
	ptr string
}

// asObject returns v as an object, or records that it is not one.
func (d *decoder) asObject(v value.Value, ptr string) (object, bool) {
	o, ok := v.(value.Object)
	if !ok {
		d.fail(ptr, ruleType)
		return object{}, false
	}
	return object{d, o, ptr}, true
}

// closed records each member not in allowed and each of required that is
// missing (ADR-002 D4 item 4).
func (o object) closed(required []string, allowed ...string) {
	for _, name := range o.o.Names() {
		if !slices.Contains(allowed, name) && !slices.Contains(required, name) {
			o.d.fail(o.member(name), ruleUnknown)
		}
	}
	for _, name := range required {
		if _, ok := o.o.Get(name); !ok {
			// A missing member is reported at its parent, as the schema's
			// required is.
			o.d.fail(o.ptr, ruleRequired)
		}
	}
}

func (o object) member(name string) string { return o.ptr + finding.Pointer(name) }

func (o object) has(name string) bool {
	_, ok := o.o.Get(name)
	return ok
}

// str returns the string member name, and whether it is present and a
// string. check judges its text, returning the rule it breaks or "".
func (o object) str(name string, check func(string) string) (string, bool) {
	v, ok := o.o.Get(name)
	if !ok {
		return "", false
	}
	s, ok := v.(value.String)
	if !ok {
		o.d.fail(o.member(name), ruleType)
		return "", false
	}
	if check != nil {
		if rule := check(s.Text()); rule != "" {
			o.d.fail(o.member(name), rule)
			return "", false
		}
	}
	return s.Text(), true
}

// boolean returns the boolean member name, or def when it is absent.
func (o object) boolean(name string, def bool) bool {
	v, ok := o.o.Get(name)
	if !ok {
		return def
	}
	b, ok := v.(value.Bool)
	if !ok {
		o.d.fail(o.member(name), ruleType)
		return def
	}
	return bool(b)
}

// array returns the array member name, recording a count outside
// [lo, hi] (hi < 0 for no bound).
func (o object) array(name string, lo, hi int) (value.Array, bool) {
	v, ok := o.o.Get(name)
	if !ok {
		return value.Array{}, false
	}
	a, ok := v.(value.Array)
	if !ok {
		o.d.fail(o.member(name), ruleType)
		return value.Array{}, false
	}
	if a.Len() < lo || hi >= 0 && a.Len() > hi {
		o.d.fail(o.member(name), ruleItems)
	}
	return a, true
}

func index(ptr string, i int) string { return ptr + "/" + strconv.Itoa(i) }

// The text rules. Each returns the rule s breaks, or "".

// length judges s's length in Unicode scalar values (D4 item 6) against
// [lo, hi], hi < 0 for no bound.
func length(s string, lo, hi int) string {
	n := utf8.RuneCountInString(s)
	if n < lo || hi >= 0 && n > hi {
		return ruleLength
	}
	return ""
}

// line is D4 item 6's line, of 1 to hi characters.
func line(hi int) func(string) string {
	return func(s string) string {
		if r := length(s, 1, hi); r != "" {
			return r
		}
		return match(lineRE, ruleLine)(s)
	}
}

// prose is D4 item 6's prose, of 1 to hi characters, hi < 0 for no bound.
func prose(hi int) func(string) string {
	return func(s string) string {
		if r := length(s, 1, hi); r != "" {
			return r
		}
		return match(proseRE, ruleProse)(s)
	}
}

// match judges s by re, which it must match.
func match(re *regexp.Regexp, rule string) func(string) string {
	return func(s string) string {
		if !re.MatchString(s) {
			return rule
		}
		return ""
	}
}

// oneOf judges s against a closed set of values.
func oneOf(values ...string) func(string) string {
	return func(s string) string {
		if !slices.Contains(values, s) {
			return ruleEnum
		}
		return ""
	}
}

func packageName(s string) string {
	if r := length(s, 1, 128); r != "" {
		return r
	}
	if !nameRE.MatchString(s) || nameDeviceRE.MatchString(s) || nameLongSegmentRE.MatchString(s) {
		return ruleName
	}
	return ""
}

func packageVersion(s string) string {
	if r := length(s, 0, 128); r != "" {
		return r
	}
	return match(semverRE, ruleVersion)(s)
}

func httpURI(s string) string {
	if r := length(s, 0, 2048); r != "" {
		return r
	}
	return match(httpURIRE, ruleURI)(s)
}

func email(s string) string {
	if r := length(s, 3, 254); r != "" {
		return r
	}
	return match(emailRE, ruleEmail)(s)
}

func license(s string) string { return length(s, 1, 512) }

func inputName(s string) string {
	if !identifierRE.MatchString(s) {
		return ruleIdentifier
	}
	if slices.Contains(reservedInputNames, s) {
		return ruleReserved
	}
	return ""
}

func validatePath(s string) string {
	if !relativePathRE.MatchString(s) || compiledRE.MatchString(s) {
		return rulePath
	}
	return ""
}

// The members of the manifest's root under METAIS 1.0 (D4 item 2).
var (
	rootRequired = []string{"metais", "name", "version", "title", "description", "guide", "inputs", "outputs"}
	rootOptional = []string{"authors", "license", "homepage", "input_constraints"}
)

// manifest10 decodes root under METAIS 1.0's rules, which Gate has
// accepted as spec.
func (d *decoder) manifest10(root value.Value, spec Spec) *Manifest {
	o, ok := d.asObject(root, "")
	if !ok {
		return nil
	}
	o.closed(rootRequired, rootOptional...)
	m := &Manifest{Metais: spec}
	o.str("metais", func(s string) string {
		if s != "1.0" {
			return ruleConst
		}
		return ""
	})
	m.Name, _ = o.str("name", packageName)
	m.Version, _ = o.str("version", packageVersion)
	m.Title, _ = o.str("title", line(200))
	m.Description, _ = o.str("description", prose(1000))
	m.Guide, _ = o.str("guide", prose(16384))
	if a, ok := o.array("authors", 1, 64); ok {
		for i, v := range a.All() {
			if au, ok := d.author(v, index(o.member("authors"), i)); ok {
				m.Authors = append(m.Authors, au)
			}
		}
	}
	m.License, _ = o.str("license", license)
	m.Homepage, _ = o.str("homepage", httpURI)
	m.InputConstraints, _ = o.str("input_constraints", match(schemaPathRE, rulePath))
	if a, ok := o.array("inputs", 0, -1); ok {
		for i, v := range a.All() {
			if in, ok := d.input(v, index(o.member("inputs"), i)); ok {
				m.Inputs = append(m.Inputs, in)
			}
		}
	}
	if a, ok := o.array("outputs", 1, -1); ok {
		for i, v := range a.All() {
			if out, ok := d.output(v, index(o.member("outputs"), i), false); ok {
				m.Outputs = append(m.Outputs, out)
			}
		}
	}
	return m
}

func (d *decoder) author(v value.Value, ptr string) (Author, bool) {
	o, ok := d.asObject(v, ptr)
	if !ok {
		return Author{}, false
	}
	o.closed([]string{"name"}, "email", "url")
	var a Author
	a.Name, _ = o.str("name", line(200))
	a.Email, _ = o.str("email", email)
	a.URL, _ = o.str("url", httpURI)
	return a, true
}

// input decodes one input slot (ADR-003 D1).
func (d *decoder) input(v value.Value, ptr string) (Input, bool) {
	o, ok := d.asObject(v, ptr)
	if !ok {
		return Input{}, false
	}
	o.closed([]string{"name", "title", "schema"}, "description", "meaning", "forms", "csv", "collection", "required")
	in := Input{Pointer: ptr}
	in.Name, _ = o.str("name", inputName)
	in.Title, _ = o.str("title", line(200))
	in.Description, _ = o.str("description", prose(-1))
	in.Schema, _ = o.str("schema", match(schemaPathRE, rulePath))
	in.Meaning, _ = o.str("meaning", match(meaningPathRE, rulePath))
	in.Forms = []Form{FormJSON, FormYAML}
	if a, ok := o.array("forms", 1, -1); ok {
		in.Forms = nil
		for i, f := range a.All() {
			fptr := index(o.member("forms"), i)
			s, ok := f.(value.String)
			switch {
			case !ok || !slices.Contains([]string{"json", "yaml", "csv", "text", "markdown"}, s.Text()):
				d.fail(fptr, ruleEnum)
			case slices.Contains(in.Forms, Form(s.Text())):
				d.fail(fptr, ruleUnique)
			default:
				in.Forms = append(in.Forms, Form(s.Text()))
			}
		}
	}
	csvForm := slices.Contains(in.Forms, FormCSV)
	if v, ok := o.o.Get("csv"); ok {
		// ADR-003 D1 item 2: csv only when forms, as written, holds csv.
		if !o.has("forms") || !csvForm {
			d.fail(o.member("csv"), ruleCSVForms)
		}
		if c, ok := d.asObject(v, o.member("csv")); ok {
			c.closed([]string{"delimiter"})
			if dv, ok := c.o.Get("delimiter"); ok {
				// The delimiter's enum admits only strings, so any other
				// kind breaks it, as the schema's enum judges.
				s, isStr := dv.(value.String)
				if !isStr || oneOf(",", ";", "|", "\t")(s.Text()) != "" {
					d.fail(c.member("delimiter"), ruleEnum)
				} else {
					in.Delimiter = s.Text()
				}
			}
		}
	}
	if csvForm && in.Delimiter == "" {
		in.Delimiter = ","
	}
	in.Collection = o.boolean("collection", false)
	in.Required = o.boolean("required", true)
	return in, true
}

// The fields each output form allows, beside its source key (ADR-005 D1
// item 3). An entry also allows first and stored and refuses executable
// (D2 item 6).
var formFields = map[OutputForm][]string{
	OutputTemplate:  {"id", "path", "when", "for_each", "escape", "newline", "validate", "kind", "executable"},
	OutputAsset:     {"id", "path", "when", "executable"},
	OutputMirror:    {"id", "path", "when", "escape", "newline"},
	OutputContainer: {"id", "path", "when", "for_each"},
}

// sourceKeys are the source keys, in the order D1 item 2 lists them.
var sourceKeys = []OutputForm{OutputTemplate, OutputAsset, OutputMirror, OutputContainer}

// kinds are the output kinds of ADR-005 D7 item 1's table.
var kinds = []string{"html", "xml", "markdown", "json", "yaml", "csv", "css", "hash", "slash", "sql", "text"}

// output decodes one output declaration, a container's entry when entry is
// set.
func (d *decoder) output(v value.Value, ptr string, entry bool) (Output, bool) {
	o, ok := d.asObject(v, ptr)
	if !ok {
		return Output{}, false
	}
	var forms []OutputForm
	for _, k := range sourceKeys {
		if o.has(string(k)) {
			forms = append(forms, k)
		}
	}
	if len(forms) != 1 {
		// ADR-005 D1 item 2: none of the source keys, or more than one,
		// matches none of the schema's oneOf branches.
		d.fail(ptr, ruleForm)
		return Output{}, false
	}
	out := Output{Pointer: ptr, Form: forms[0], Escape: "none", Newline: "as-written"}
	allowed := []string{string(out.Form)}
	for _, f := range formFields[out.Form] {
		if entry && f == "executable" {
			continue
		}
		allowed = append(allowed, f)
	}
	if entry {
		allowed = append(allowed, "first", "stored")
	}
	var required []string
	if !entry {
		required = append(required, "id")
	}
	if out.Form == OutputContainer || o.has("for_each") && slices.Contains(allowed, "for_each") {
		// D4 item 4: a container and a for_each declaration name their
		// path.
		if !o.has("path") {
			rule := ruleRequired
			if out.Form != OutputContainer {
				rule = ruleForEachPath
			}
			d.fail(ptr, rule)
		}
	}
	o.closed(required, allowed...)
	nonEmpty := func(s string) string { return length(s, 1, -1) }
	out.ID, _ = o.str("id", match(identifierRE, ruleIdentifier))
	out.Path, _ = o.str("path", nonEmpty)
	out.When, _ = o.str("when", nonEmpty)
	if slices.Contains(allowed, "for_each") {
		out.ForEach, _ = o.str("for_each", nonEmpty)
	}
	if out.Form != OutputContainer {
		out.Source, _ = o.str(string(out.Form), match(relativePathRE, rulePath))
	}
	if slices.Contains(allowed, "escape") {
		if s, ok := o.enum("escape", "none", "html"); ok {
			out.Escape = s
		}
		if s, ok := o.enum("newline", "as-written", "crlf"); ok {
			out.Newline = s
		}
	}
	if slices.Contains(allowed, "validate") {
		out.Validate, _ = o.str("validate", validatePath)
		out.Kind, _ = o.enum("kind", kinds...)
	}
	if slices.Contains(allowed, "executable") {
		out.Executable = o.boolean("executable", false)
	}
	if entry {
		out.First = o.boolean("first", false)
		out.Stored = o.boolean("stored", false)
	}
	if out.Form == OutputContainer {
		if cv, ok := o.o.Get("container"); ok {
			out.Container = d.container(cv, o.member("container"))
		}
	}
	return out, true
}

// enum returns the member name when it is one of values. An enum of
// strings refuses a value of any other kind as not among them.
func (o object) enum(name string, values ...string) (string, bool) {
	v, ok := o.o.Get(name)
	if !ok {
		return "", false
	}
	s, ok := v.(value.String)
	if !ok || !slices.Contains(values, s.Text()) {
		o.d.fail(o.member(name), ruleEnum)
		return "", false
	}
	return s.Text(), true
}

// container decodes a container's shape (ADR-005 D8 item 1, D9).
func (d *decoder) container(v value.Value, ptr string) *Container {
	o, ok := d.asObject(v, ptr)
	if !ok {
		return nil
	}
	o.closed([]string{"entries"}, "format", "profile", "mimetype")
	c := &Container{Format: "zip"}
	if v, ok := o.o.Get("format"); ok {
		if s, isStr := v.(value.String); !isStr || s.Text() != "zip" {
			d.fail(o.member("format"), ruleConst)
		}
	}
	c.Profile, _ = o.enum("profile", "ooxml", "odf", "epub")
	// D9 item 2: odf requires the media type, and nothing else takes one.
	// The schema's if judges profile by its value as written, so a profile
	// that is not one of the three is not odf.
	odf := false
	if pv, ok := o.o.Get("profile"); ok {
		s, isStr := pv.(value.String)
		odf = isStr && s.Text() == "odf"
	}
	switch {
	case odf && !o.has("mimetype"):
		d.fail(ptr, ruleRequired)
	case !odf && o.has("mimetype"):
		d.fail(o.member("mimetype"), ruleMimetype)
	case odf:
		c.Mimetype, _ = o.str("mimetype", match(mimetypeRE, ruleMimetype))
	}
	if a, ok := o.array("entries", 1, -1); ok {
		for i, ev := range a.All() {
			if e, ok := d.output(ev, index(o.member("entries"), i), true); ok {
				c.Entries = append(c.Entries, e)
			}
		}
	}
	return c
}
