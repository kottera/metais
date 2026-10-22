// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"encoding/hex"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/digest"
	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/inputs"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
)

// The engine codes of METAIS's tags, which their definitions report and
// parseFinding and renderFinding map to the template family's codes and
// to input.raw_unavailable (ADR-004 D16 item 1, D12 item 3).
const (
	codeEmbedUnknownInput  liquid.Code = "embed_unknown_input"
	codeEmbedIndex         liquid.Code = "embed_index"
	codeRawUnavailable     liquid.Code = "raw_unavailable"
	codeProvenanceArgument liquid.Code = "provenance_argument"
	codeProvenanceStyle    liquid.Code = "provenance_style"
)

// tagOf is the tag whose definition reports code, or "".
func tagOf(code liquid.Code) string {
	switch code {
	case codeEmbedUnknownInput, codeEmbedIndex:
		return "embed"
	case codeProvenanceArgument, codeProvenanceStyle:
		return "provenance"
	}
	return ""
}

// The reasons a tag's finding names.
const (
	// template.embed_index: a collection's embed without index:, an index:
	// given for an input that is not a collection, an index that is not an
	// integer, and one beyond the collection's items.
	reasonIndexMissing  = "missing"
	reasonNotCollection = "not_a_collection"
	reasonNotInteger    = "not_an_integer"
	reasonOutOfRange    = "out_of_range"
	// template.provenance_argument: a word outside the closed set or a
	// keyword other than style:, a part named twice, and no part at all.
	reasonUnknownPart = "unknown"
	reasonRepeated    = "repeated"
	reasonNoPart      = "missing"
	// template.provenance_style: a csv file, which has no comments; a text
	// file, whose comment syntax style: names; and a style: that names no
	// style.
	reasonNoComments    = "no_comments"
	reasonStyleRequired = "style_required"
	reasonUnknownStyle  = "unknown_style"
)

// bom is the UTF-8 byte order mark, which embed does not write (ADR-004
// D12 item 2).
var bom = []byte{0xEF, 0xBB, 0xBF}

// embedTag is ADR-004 D12's embed: {% embed "<input>" %}, or
// {% embed "<input>", index: <expr> %} for an item of a collection. r is
// the render it writes in; the parse dialect's has none.
type embedTag struct{ r *Renderer }

func (embedTag) Name() string { return "embed" }

// Check refuses what is wrong whatever the package declares: a name that
// is not a string literal is template.embed_unknown_input (D12 item 1),
// and anything beyond the name and index: is the grammar's. What the
// declarations decide is Templates.Parse's and EmbedFindings'.
func (embedTag) Check(t *ast.Tag) *liquid.Error {
	if _, ok := embedName(t); !ok {
		return &liquid.Error{Code: codeEmbedUnknownInput, Detail: "embed's first argument is not a string literal"}
	}
	for _, a := range t.Args[1:] {
		if a.Name != "index" {
			return &liquid.Error{Code: liquid.CodeSyntax, Detail: "embed takes an input's name, and index: for a collection's item"}
		}
	}
	return nil
}

// Render writes the raw form of the input, or of the item index: gives,
// unchanged but for a leading byte order mark, and writes nothing for an
// input not supplied (D12 items 2 and 3). It never escapes (D9 item 4).
func (e embedTag) Render(c liquid.TagContext, t *ast.Tag) *liquid.Error {
	if e.r == nil {
		return &liquid.Error{Code: liquid.CodeInternal, Detail: "embed rendered in the parse dialect"}
	}
	name, _ := embedName(t)
	in, rec, ok := e.r.b.input(name)
	if !ok {
		return &liquid.Error{Code: liquid.CodeInternal, Detail: "embed names an input its template's parse did not find: " + name}
	}
	if rec.Value == nil {
		return nil
	}
	raw, has := rec.Raw, rec.HasRaw
	args := map[string]string{"input": name}
	if in.Collection {
		v, err := c.Eval(embedIndex(t).Value)
		if err != nil {
			return err
		}
		i, isInt := templateInt(v)
		switch {
		case !isInt:
			args["reason"], args["type"] = reasonNotInteger, kindName(v)
			return &liquid.Error{Code: codeEmbedIndex, Args: args, Detail: "embed's index is not an integer"}
		case i < 0 || i >= int64(len(rec.Items)):
			args["reason"], args["index"], args["items"] = reasonOutOfRange, strconv.FormatInt(i, 10), strconv.Itoa(len(rec.Items))
			return &liquid.Error{Code: codeEmbedIndex, Args: args, Detail: "embed's index is beyond the collection's items"}
		}
		args["index"] = strconv.FormatInt(i, 10)
		raw, has = rec.Items[i].Raw, rec.Items[i].HasRaw
	}
	if !has {
		return &liquid.Error{Code: codeRawUnavailable, Args: args, Detail: "the input has no raw form"}
	}
	raw = strings.TrimPrefix(raw, string(bom))
	if !utf8.ValidString(raw) {
		return &liquid.Error{Code: liquid.CodeInvalidUTF8, Args: map[string]string{"tag": "embed"}, Detail: "the raw form is not UTF-8"}
	}
	return c.Write([]byte(raw))
}

// embedName is an embed tag's input name: its first argument, a string
// literal written before any keyword.
func embedName(t *ast.Tag) (string, bool) {
	if len(t.Args) == 0 || t.Args[0].Name != "" {
		return "", false
	}
	lit, ok := t.Args[0].Value.(*ast.Literal)
	if !ok || lit == nil || lit.Kind != ast.LitString {
		return "", false
	}
	return lit.Text, true
}

// embedIndex is an embed tag's index: argument, or nil.
func embedIndex(t *ast.Tag) *ast.Arg {
	for _, a := range t.Args {
		if a.Name == "index" {
			return a
		}
	}
	return nil
}

// embeds returns every embed tag of t whose name is a literal, in source
// order.
func embeds(t *liquid.Template) []*ast.Tag {
	var out []*ast.Tag
	ast.Walk(t.Root, func(n ast.Node) bool {
		if tag, ok := n.(*ast.Tag); ok && tag.Name == "embed" {
			if _, ok := embedName(tag); ok {
				out = append(out, tag)
			}
		}
		return true
	})
	return out
}

// EmbedFindings returns the findings of ADR-004 D12 item 1 that the
// package's declared inputs decide when a template is parsed, for the
// template entry, which parsed as t: template.embed_index for an embed of
// a collection without index: and for one of an input that is not a
// collection with it. An embed naming no declared input is
// template.embed_unknown_input to a render, which Templates.Parse
// reports, and check's cross-reference's template.xref.unknown_input
// (ADR-007 D3 item 12).
func EmbedFindings(entry string, t *liquid.Template, decls []manifest.Input) []finding.Finding {
	var out []finding.Finding
	for _, l := range embedChecks(entry, t, decls, false) {
		out = append(out, l.f)
	}
	return out
}

// embedChecks is EmbedFindings, with each finding's offset, and with
// template.embed_unknown_input for an embed naming no input of decls when
// unknown is set.
func embedChecks(entry string, t *liquid.Template, decls []manifest.Input, unknown bool) []located {
	var out []located
	for _, tag := range embeds(t) {
		name, _ := embedName(tag)
		i := slices.IndexFunc(decls, func(d manifest.Input) bool { return d.Name == name })
		var e *liquid.Error
		switch hasIndex := embedIndex(tag) != nil; {
		case i < 0 && unknown:
			e = &liquid.Error{Code: codeEmbedUnknownInput, Args: map[string]string{"name": name}}
		case i < 0:
		case decls[i].Collection && !hasIndex:
			e = &liquid.Error{Code: codeEmbedIndex, Args: map[string]string{"input": name, "reason": reasonIndexMissing}}
		case !decls[i].Collection && hasIndex:
			e = &liquid.Error{Code: codeEmbedIndex, Args: map[string]string{"input": name, "reason": reasonNotCollection}}
		}
		if e != nil {
			e.Source, e.Line, e.Col, e.Offset = entry, tag.Span.Line, tag.Span.Col, tag.Span.Start
			out = append(out, located{tag.Span.Start, parseFinding(e, limits.Set{}, "")})
		}
	}
	return out
}

// Unavailable returns input.raw_unavailable for every embed tag of the
// template name whose input and item are known before anything renders —
// a literal name, and for a collection a literal integer index within its
// items — and whose raw form does not exist (ADR-004 D12 item 3, ADR-003
// D21 item 2). The template parsed without a finding. An embed whose index
// is computed is checked when it is evaluated.
func (r *Renderer) Unavailable(name string) []finding.Finding {
	t, fs := r.ts.Parse(name)
	if t == nil {
		panic("template: the raw forms of " + name + ", whose parse failed: " + string(fs[0].Code()))
	}
	var out []finding.Finding
	for _, tag := range embeds(t) {
		input, _ := embedName(tag)
		in, rec, ok := r.b.input(input)
		if !ok || rec.Value == nil {
			continue
		}
		var p finding.Params
		if in.Collection {
			idx := embedIndex(tag)
			lit, isLit := idx.Value.(*ast.Literal)
			if !isLit || lit == nil {
				continue
			}
			i, isInt := templateInt(lit.Value)
			if !isInt || i < 0 || i >= int64(len(rec.Items)) || rec.Items[i].HasRaw {
				continue
			}
			p = finding.Params{"index": finding.Int(i)}
		} else if rec.HasRaw {
			continue
		}
		if p == nil {
			p = finding.Params{}
		}
		p["column"] = finding.Int(int64(tag.Span.Col))
		loc := finding.Location{Origin: finding.OriginPackage, File: Dir + name, Line: tag.Span.Line, Column: tag.Span.Col}
		out = append(out, finding.New(CodeRawUnavailable, finding.Subject{Kind: finding.SubjectInput, Name: input}, &loc, p))
	}
	return out
}

// templateInt is v as a template integer (ADR-004 D4 item 2), and whether
// it is one.
func templateInt(v value.Value) (int64, bool) {
	n, ok := v.(value.Number)
	if !ok || !n.WrittenAsInteger() {
		return 0, false
	}
	return n.SafeInt()
}

// kindName names v's kind as a finding does: integer and float apart.
func kindName(v value.Value) string {
	if n, ok := v.(value.Number); ok {
		if _, isInt := templateInt(n); isInt {
			return "integer"
		}
		return "float"
	}
	return v.Kind().String()
}

// provenanceTag is ADR-004 D14's provenance: {% provenance <part> …
// [style: "<style>"] %}. r is the render it writes in; the parse
// dialect's has none.
type provenanceTag struct{ r *Renderer }

// provenanceParts are D14 item 1's closed set of parts, in the order their
// fields are written (item 2).
var provenanceParts = []string{"package", "version", "package_digest", "inputs", "tool", "time"}

// partAll names every part.
const partAll = "all"

// provenanceStyles are D14 item 4's styles: what a line is written
// between, json's being written as an object instead.
var provenanceStyles = map[string][2]string{
	"html":      {"<!-- ", " -->"},
	"hash":      {"# ", ""},
	"slash":     {"// ", ""},
	"block":     {"/* ", " */"},
	"dash":      {"-- ", ""},
	"percent":   {"% ", ""},
	"semicolon": {"; ", ""},
	"json":      {},
}

// kindStyles are D14 item 5's styles by output kind (ADR-005 D7). csv and
// text have none.
var kindStyles = map[string]string{
	"html": "html", "xml": "html", "markdown": "html",
	"yaml": "hash", "hash": "hash",
	"slash": "slash",
	"css":   "block",
	"sql":   "dash",
	"json":  "json",
}

func (provenanceTag) Name() string { return "provenance" }

// Check refuses a part outside the closed set, a keyword but style:, a
// part named twice — all naming every part, so all with another is
// twice — and no part (template.provenance_argument), and a literal style:
// that names no style (template.provenance_style), D14 items 1 and 5.
func (provenanceTag) Check(t *ast.Tag) *liquid.Error {
	if _, _, e := provenanceArgs(t); e != nil {
		return e
	}
	if s := provenanceStyle(t); s != nil {
		if lit, ok := s.Value.(*ast.Literal); ok && lit != nil {
			return checkStyle(lit.Value)
		}
	}
	return nil
}

// provenanceArgs returns the parts the tag names, all expanded, and its
// style: argument, or nil.
func provenanceArgs(t *ast.Tag) (map[string]bool, *ast.Arg, *liquid.Error) {
	parts := map[string]bool{}
	var style *ast.Arg
	named := 0
	for _, a := range t.Args {
		if a.Name != "" {
			if a.Name != "style" {
				return nil, nil, provenanceArgErr(reasonUnknownPart, a.Name)
			}
			style = a
			continue
		}
		p, ok := a.Value.(*ast.Path)
		if !ok || p == nil || len(p.Segments) > 0 || (p.Root != partAll && !slices.Contains(provenanceParts, p.Root)) {
			word := ""
			if ok && p != nil && len(p.Segments) == 0 {
				word = p.Root
			}
			return nil, nil, provenanceArgErr(reasonUnknownPart, word)
		}
		named++
		if parts[p.Root] || parts[partAll] || (p.Root == partAll && named > 1) {
			return nil, nil, provenanceArgErr(reasonRepeated, p.Root)
		}
		parts[p.Root] = true
	}
	if named == 0 {
		return nil, nil, provenanceArgErr(reasonNoPart, "")
	}
	if parts[partAll] {
		for _, p := range provenanceParts {
			parts[p] = true
		}
	}
	return parts, style, nil
}

// provenanceStyle is the tag's style: argument, or nil.
func provenanceStyle(t *ast.Tag) *ast.Arg {
	for _, a := range t.Args {
		if a.Name == "style" {
			return a
		}
	}
	return nil
}

func provenanceArgErr(reason, word string) *liquid.Error {
	args := map[string]string{"reason": reason}
	if word != "" {
		args["argument"] = word
	}
	return &liquid.Error{Code: codeProvenanceArgument, Args: args, Detail: "provenance's argument is refused: " + reason}
}

// checkStyle refuses a style: value that names no style of D14 item 4.
func checkStyle(v value.Value) *liquid.Error {
	s, ok := v.(value.String)
	if ok {
		if _, known := provenanceStyles[s.Text()]; known {
			return nil
		}
	}
	args := map[string]string{"reason": reasonUnknownStyle}
	if ok {
		args["style"] = s.Text()
	} else {
		args["type"] = kindName(v)
	}
	return &liquid.Error{Code: codeProvenanceStyle, Args: args, Detail: "style: names no style"}
}

// Render writes the line of the parts named, in the style style: names or
// the output's kind gives (D14 items 2 to 5), with no line feed (item 6),
// never escaped (D9 item 4).
func (p provenanceTag) Render(c liquid.TagContext, t *ast.Tag) *liquid.Error {
	r := p.r
	if r == nil {
		return &liquid.Error{Code: liquid.CodeInternal, Detail: "provenance rendered in the parse dialect"}
	}
	parts, styleArg, e := provenanceArgs(t)
	if e != nil {
		return &liquid.Error{Code: liquid.CodeInternal, Detail: "provenance's arguments passed its parse and are refused at render"}
	}
	var style string
	if styleArg != nil {
		v, err := c.Eval(styleArg.Value)
		if err != nil {
			return err
		}
		if err := checkStyle(v); err != nil {
			return err
		}
		style = v.(value.String).Text()
	} else {
		kind := r.out.Kind
		var ok bool
		if style, ok = kindStyles[kind]; !ok {
			reason := reasonStyleRequired
			if kind == "csv" {
				reason = reasonNoComments
			}
			return &liquid.Error{Code: codeProvenanceStyle, Args: map[string]string{"reason": reason, "kind": kind},
				Detail: "the output's kind has no provenance style"}
		}
	}
	fields := r.b.provenance(parts)
	if canaryProvenance != nil {
		fields = append(fields, canaryProvenance())
	}
	return c.Write(provenanceLine(fields, parts["inputs"], style))
}

// canaryProvenance, when set, gives a field every provenance line gains.
// Only a canary build sets it (ADR-009 D16 item 2): the pid canary records
// the process id, which the SC7 harness must catch. The tool's own build
// never does, and a line holds only D14 item 2's fields.
var canaryProvenance func() field

// field is one field of a provenance line: its name, and its value, or
// none for an input not supplied.
type field struct {
	name  string
	value string
	none  bool
}

// inputPrefix begins the name of an input's field.
const inputPrefix = "input."

// provenance returns the fields of parts, in D14 item 2's order: the
// package's name, version and digest, each declared input's digest by
// name in canonical order, the tool, and the render time when --now gave
// one.
func (b *Bindings) provenance(parts map[string]bool) []field {
	var out []field
	if parts["package"] {
		out = append(out, field{name: "package", value: b.pkg.Manifest.Name})
	}
	if parts["version"] {
		out = append(out, field{name: "version", value: b.pkg.Manifest.Version})
	}
	if parts["package_digest"] {
		out = append(out, field{name: "package_digest", value: digest.Format(b.pkg.Digest)})
	}
	if parts["inputs"] {
		recs := slices.Clone(b.records)
		slices.SortFunc(recs, func(x, y inputs.Record) int { return value.CompareNames(x.Name, y.Name) })
		for _, rec := range recs {
			f := field{name: inputPrefix + rec.Name, none: !rec.HasDigest}
			if rec.HasDigest {
				f.value = digest.Algorithm + ":" + hex.EncodeToString(rec.Digest[:])
			}
			out = append(out, f)
		}
	}
	if parts["tool"] {
		out = append(out, field{name: "tool", value: b.pkg.Tool})
	}
	if parts["time"] && b.now != "" {
		out = append(out, field{name: "time", value: b.now})
	}
	return out
}

// provenanceLine is the line of fields in style (D14 items 3 and 4):
// metais-provenance and each field, separated by one space, between the
// style's delimiters; or, for json, the fields as an RFC 8785 object, the
// inputs as an object, present when the inputs part is named.
func provenanceLine(fields []field, inputs bool, style string) []byte {
	if style == "json" {
		var top, in []value.Member
		for _, f := range fields {
			var v value.Value = value.Null{}
			if !f.none {
				v = str(f.value)
			}
			if name, ok := strings.CutPrefix(f.name, inputPrefix); ok {
				in = append(in, value.Member{Name: name, Value: v})
			} else {
				top = append(top, value.Member{Name: f.name, Value: v})
			}
		}
		if inputs {
			top = append(top, value.Member{Name: "inputs", Value: obj(in...)})
		}
		return jsonx.Canonical(obj(top...))
	}
	delims := provenanceStyles[style]
	var b strings.Builder
	b.WriteString(delims[0])
	b.WriteString("metais-provenance")
	for _, f := range fields {
		b.WriteString(" " + f.name + "=")
		if f.none {
			b.WriteString("absent")
		} else {
			b.WriteString(encodeValue(f.value, style == "html"))
		}
	}
	b.WriteString(delims[1])
	return []byte(b.String())
}

// encodeValue writes a field's value (D14 item 3): a byte outside A–Z,
// a–z, 0–9 and . _ + : / @ - as %XX in upper-case hex, and, when html is
// set, a - that follows a - as %2D, so that no value ends an XML comment.
func encodeValue(s string, html bool) string {
	const hexDigits = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case c == '-' && html && i > 0 && s[i-1] == '-':
			b.WriteString("%2D")
		case 'A' <= c && c <= 'Z', 'a' <= c && c <= 'z', '0' <= c && c <= '9', strings.IndexByte("._+:/@-", c) >= 0:
			b.WriteByte(c)
		default:
			b.WriteByte('%')
			b.WriteByte(hexDigits[c>>4])
			b.WriteByte(hexDigits[c&0xF])
		}
	}
	return b.String()
}
