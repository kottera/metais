// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/inputs/csv"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/markdown"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/yamlx"
)

// bodyField is the member a Markdown file's text after its frontmatter is
// (ADR-003 D13 item 1).
const bodyField = "body"

// bom is the UTF-8 byte order mark ADR-003 D8 item 1 removes.
var bom = []byte("\xEF\xBB\xBF")

// extensions maps a lower-case extension to the form it names (ADR-003
// D15 item 4, D16 item 2). A Markdown extension names markdown when the
// input accepts it and text otherwise: formOf decides.
var extensions = map[string]manifest.Form{
	".json":     manifest.FormJSON,
	".yaml":     manifest.FormYAML,
	".yml":      manifest.FormYAML,
	".csv":      manifest.FormCSV,
	".txt":      manifest.FormText,
	".md":       manifest.FormMarkdown,
	".markdown": manifest.FormMarkdown,
}

// formOf is the form a file of extension ext is read in for an input
// accepting forms, and false when ext names none of them.
func formOf(ext string, forms []manifest.Form) (manifest.Form, bool) {
	f, ok := extensions[ext]
	if !ok {
		return "", false
	}
	if f == manifest.FormMarkdown && !accepts(forms, f) {
		f = manifest.FormText
	}
	return f, accepts(forms, f)
}

func accepts(forms []manifest.Form, f manifest.Form) bool {
	for _, g := range forms {
		if g == f {
			return true
		}
	}
	return false
}

// accepted lists the extensions an input accepting forms takes, in a
// fixed order, for input.extension_not_accepted.
func accepted(forms []manifest.Form) finding.Value {
	var out []finding.Value
	for _, ext := range []string{".json", ".yaml", ".yml", ".csv", ".txt", ".md", ".markdown"} {
		if _, ok := formOf(ext, forms); ok {
			out = append(out, finding.String(ext))
		}
	}
	return finding.List(out...)
}

// parsed is what one file of an input gave.
type parsed struct {
	// value is the file's value, nil when the file contributes none: a
	// finding refused it whole.
	value *node
	// partial is set for a CSV file some of whose records were left out
	// (ADR-006 D7 item 4): the rows of value have their original indexes.
	partial *partial
	// findings is every finding the file gave.
	findings []finding.Finding
}

// partial is an array validated without the items that did not parse.
type partial struct {
	kept     []int
	unparsed int
}

// reader reads one input's files: its declaration, its subject, and the
// static schema walk that coerces a CSV field (ADR-003 D14).
type reader struct {
	in     manifest.Input
	subj   finding.Subject
	lim    limits.Set
	values *limits.Counter
	walker *walker
}

// read reads src, the bytes of one file in form, at loc (its origin and
// file); raw is the same bytes as a string, which a plain text value
// shares when it is the file's text as it stands. prefix is where the file's value is in its input: "" for an
// input's own file, /<index> for a collection's item, and at is the
// static path of that place, for coercion.
func (r *reader) read(src []byte, raw string, form manifest.Form, loc finding.Location, prefix string, at []schema.Step) parsed {
	if form == manifest.FormYAML && int64(len(src)) > r.lim.Value(limits.YAMLSize) {
		// yaml/v3 builds the whole document's tree before any value can
		// be counted, so a YAML document is bounded by its bytes before it
		// is parsed (ADR-009 Amendment A4 item 2).
		return parsed{findings: []finding.Finding{r.lim.Exceeded(limits.YAMLSize, r.subj, loc, int64(len(src)), true)}}
	}
	if form == manifest.FormJSON {
		doc, fs := jsonx.Parse(src, jsonx.Source{Input: r.subj, At: loc, Values: r.values}, r.lim)
		if len(fs) > 0 {
			return parsed{findings: r.place(fs, prefix)}
		}
		return parsed{value: fromValue(doc.Value, docLocator{at: loc, pos: doc.Positions()})}
	}
	if f, ok := encoding(src, r.subj, loc); !ok {
		return parsed{findings: []finding.Finding{f}}
	}
	text := bytes.TrimPrefix(src, bom)
	switch form {
	case manifest.FormYAML:
		doc, fs := yamlx.Parse(text, yamlx.Source{Subject: r.subj, At: loc, Size: limits.InputSize, Values: r.values}, r.lim)
		if len(fs) > 0 {
			return parsed{findings: r.place(fs, prefix)}
		}
		return parsed{value: fromValue(doc.Value, docLocator{at: loc, pos: doc.Positions()})}
	case manifest.FormCSV:
		return r.csv(normalize(text), loc, prefix, at)
	case manifest.FormText:
		normal := normalize(text)
		if f, ok := noncharacters(normal, r.subj, loc, 1); !ok {
			return parsed{findings: []finding.Finding{f}}
		}
		if f, ok := r.count(1, loc, prefix); !ok {
			return parsed{findings: []finding.Finding{f}}
		}
		if bytes.IndexByte(text, '\r') >= 0 {
			return parsed{value: fromValue(str(bytes.TrimSuffix(normal, []byte("\n"))), plainLocator{loc})}
		}
		// The text needs no line break changed: its value is the raw
		// string's, shared rather than copied (ADR-009 Amendment A4 item 3).
		v, err := value.NewString(strings.TrimSuffix(raw[len(src)-len(text):], "\n"))
		if err != nil {
			panic("inputs: " + err.Error())
		}
		return parsed{value: fromValue(v, plainLocator{loc})}
	case manifest.FormMarkdown:
		return r.markdown(text, loc, prefix)
	}
	panic(fmt.Sprintf("inputs: no reader for the form %q", form))
}

// count counts n values decoded at loc, at the pointer ptr in its input,
// against input-values, and reports the finding that refuses them when
// they pass it.
func (r *reader) count(n int64, loc finding.Location, ptr string) (finding.Finding, bool) {
	if r.values.Add(n) {
		return finding.Finding{}, true
	}
	return r.lim.Exceeded(limits.InputValues, r.subj, loc.WithPointer(ptr), 0, false), false
}

// place puts a reader's findings where the file's value is in its input.
func (r *reader) place(fs []finding.Finding, prefix string) []finding.Finding {
	if prefix == "" {
		return fs
	}
	out := make([]finding.Finding, len(fs))
	for i, f := range fs {
		out[i] = rebuild(f, r.subj, under(prefix))
	}
	return out
}

// encoding refuses a file that is not UTF-8 (ADR-003 D8 item 1) with
// input.encoding_invalid at its first byte that is not, and reports
// whether it is UTF-8. A UTF-16 byte order mark is named as such.
func encoding(src []byte, subj finding.Subject, loc finding.Location) (finding.Finding, bool) {
	if len(src) >= 2 && (src[0] == 0xFE && src[1] == 0xFF || src[0] == 0xFF && src[1] == 0xFE) {
		return encodingFinding(src, 0, reasonUTF16, subj, loc), false
	}
	if utf8.Valid(src) {
		return finding.Finding{}, true
	}
	off := 0
	for off < len(src) {
		c, n := utf8.DecodeRune(src[off:])
		if c == utf8.RuneError && n == 1 {
			break
		}
		off += n
	}
	return encodingFinding(src, off, reasonInvalidUTF8, subj, loc), false
}

func encodingFinding(src []byte, off int, reason string, subj finding.Subject, loc finding.Location) finding.Finding {
	l := loc.At(position(src, off))
	return finding.New(CodeEncodingInvalid, subj, &l, finding.Params{"reason": finding.String(reason), "offset": finding.Int(int64(off))})
}

// position is the line and column of the byte at off in src (ADR-006 D5
// item 3): lines end at LF, CRLF or a lone CR; a column counts code points,
// an invalid byte as one; a leading byte order mark is not counted.
func position(src []byte, off int) finding.Pos {
	start := 0
	if bytes.HasPrefix(src, bom) {
		start = len(bom)
	}
	off = max(off, start)
	line, lineStart := 1, start
	for i := start; i < off; i++ {
		switch src[i] {
		case '\n':
			line, lineStart = line+1, i+1
		case '\r':
			if i+1 < len(src) && src[i+1] == '\n' && i+1 < off {
				i++
			}
			line, lineStart = line+1, i+1
		}
	}
	return finding.Pos{Line: line, Column: utf8.RuneCount(src[lineStart:off]) + 1}
}

// normalize returns b with every CRLF and every lone CR replaced by an LF
// (ADR-003 D8 item 2).
func normalize(b []byte) []byte {
	if bytes.IndexByte(b, '\r') < 0 {
		return b
	}
	out := make([]byte, 0, len(b))
	for i := 0; i < len(b); i++ {
		if b[i] != '\r' {
			out = append(out, b[i])
			continue
		}
		out = append(out, '\n')
		if i+1 < len(b) && b[i+1] == '\n' {
			i++
		}
	}
	return out
}

// noncharacters refuses text holding a Unicode noncharacter (ADR-003 D8
// item 4, RFC 7493 §2.1) with input.noncharacter at the first one, and
// reports whether it holds none. firstLine is the file's line text begins
// on. text is UTF-8 with LF line breaks.
func noncharacters(text []byte, subj finding.Subject, loc finding.Location, firstLine int) (finding.Finding, bool) {
	line, col := firstLine, 1
	for _, c := range string(text) {
		if noncharacter(c) {
			l := loc.At(finding.Pos{Line: line, Column: col})
			return finding.New(CodeNoncharacter, subj, &l, finding.Params{"codepoint": finding.String(fmt.Sprintf("U+%04X", c))}), false
		}
		if c == '\n' {
			line, col = line+1, 1
		} else {
			col++
		}
	}
	return finding.Finding{}, true
}

// noncharacter reports whether c is one of Unicode's 66 noncharacters.
func noncharacter(c rune) bool {
	return 0xFDD0 <= c && c <= 0xFDEF || c&0xFFFE == 0xFFFE
}

// str is text, which is valid UTF-8, as a string value.
func str(text []byte) value.String {
	s, err := value.NewString(string(text))
	if err != nil {
		panic("inputs: " + err.Error())
	}
	return s
}

// csv reads one CSV file (ADR-003 D11): a header and a row object per
// data record, each field coerced to its column's declared type set
// (D14). A finding in the header, an unterminated quote, a limit or a
// noncharacter refuses the file whole (D11 item 9). A record the reader
// rejects alone, a wrong field count or a quote finding, and a record with
// a number the data model cannot hold, is left out, and the rest of the
// file is validated in part (ADR-006 D7 item 4).
func (r *reader) csv(text []byte, loc finding.Location, prefix string, at []schema.Step) parsed {
	if f, ok := noncharacters(text, r.subj, loc, 1); !ok {
		return parsed{findings: []finding.Finding{f}}
	}
	delim, ok := csv.ParseDelimiter(r.in.Delimiter)
	if !ok {
		panic(fmt.Sprintf("inputs: the manifest gave input %s the delimiter %q", r.in.Name, r.in.Delimiter))
	}
	table, fs := csv.Read(text, delim, csv.Source{Input: r.subj, At: loc, Values: r.values}, r.lim)
	rejected := map[int]bool{}
	whole := table.Header == nil
	for _, f := range fs {
		switch f.Code() {
		case codeCSVFieldCount, codeCSVBareQuote, codeCSVAfterQuote:
			if l, ok := f.At(); ok && l.HasPointer {
				if row, ok := arrayIndex(firstToken(l.Path())); ok {
					rejected[row] = true
					continue
				}
			}
			// A quote finding in the header has no pointer.
			whole = true
		default:
			whole = true
		}
	}
	rows := &node{isArr: true}
	cl := &csvLocator{at: &loc, columns: map[string]int{}}
	for j, h := range table.Header {
		cl.columns[h.Text] = j
	}
	types := map[[2]string]schema.TypeSet{}
	// A row's static subschema depends on its index only while prefixItems
	// covers it, which it does for a prefix of the rows; past that, every
	// row's is the same.
	rowStatic, stable := "", false
	for _, rec := range table.Records {
		row := newObject(nil)
		row.index = rec.Index
		rowAt := append(append([]schema.Step{}, at...), schema.Index(rec.Index))
		if !stable {
			rowStatic = r.walker.pointer(rowAt)
			stable = !strings.HasSuffix(rowStatic, "/prefixItems/"+strconv.Itoa(rec.Index))
		}
		cells := len(cl.cells)
		for _, fd := range rec.Fields {
			cl.cells = append(cl.cells, cellPos{uint32(fd.Line), uint32(fd.Col)})
		}
		ok := true
		for j, fd := range rec.Fields {
			col := table.Header[j].Text
			key := [2]string{rowStatic, col}
			ts, seen := types[key]
			if !seen {
				ts = r.walker.types(append(rowAt[:len(rowAt):len(rowAt)], schema.Member(col)))
				types[key] = ts
			}
			pos := finding.Pos{Line: fd.Line, Column: fd.Col}
			v, present, err := schema.CoerceField(fd.Text, fd.Quoted, ts)
			if err != nil {
				panic("inputs: " + err.Error())
			}
			if !present {
				continue
			}
			if f, refused := refuseNumber(v, r.subj, loc.At(pos).WithPointer(finding.Pointer(strconv.Itoa(rec.Index), col)), r.lim); refused {
				fs = append(fs, f)
				rejected[rec.Index] = true
				ok = false
				continue
			}
			row.obj[col] = &node{scalar: v}
		}
		if !ok {
			cl.cells = cl.cells[:cells]
			continue
		}
		cl.rows = append(cl.rows, rec.Index)
		rows.arr = append(rows.arr, row)
	}
	fs = r.place(fs, prefix)
	if whole {
		return parsed{findings: fs}
	}
	rows.at = cl
	if len(rejected) == 0 {
		return parsed{value: rows, findings: fs}
	}
	if prefix != "" {
		// A collection's item that is itself a table is one item, and
		// one that does not parse is left out whole (ADR-003 D11 item 9).
		return parsed{findings: fs}
	}
	p := &partial{unparsed: len(rejected)}
	for _, row := range rows.arr {
		p.kept = append(p.kept, row.index)
	}
	return parsed{value: rows, partial: p, findings: fs}
}

// refuseNumber refuses v, a value coerced from text at l, when it is a
// number the data model or ADR-009's limits do not admit: a literal longer
// than number-length, which a reader refuses in every form (ADR-009 D3,
// D10), or one written as an integer beyond 2^53−1 (ADR-003 D8 item 5).
func refuseNumber(v value.Value, subj finding.Subject, l finding.Location, lim limits.Set) (finding.Finding, bool) {
	n, ok := v.(value.Number)
	if !ok {
		return finding.Finding{}, false
	}
	if size := int64(len(n.Literal())); size > lim.Value(limits.NumberLength) {
		return lim.Exceeded(limits.NumberLength, subj, l, size, true), true
	}
	if unsafeInteger(v) {
		return finding.New(CodeNumberOutOfRange, subj, &l, finding.Params{"reason": finding.String(reasonUnsafeInt)}), true
	}
	return finding.Finding{}, false
}

// unsafeInteger reports whether v is a number written without a fraction
// or exponent whose magnitude exceeds 2^53−1, which ADR-003 D8 item 5
// refuses rather than let it change silently.
func unsafeInteger(v value.Value) bool {
	n, ok := v.(value.Number)
	if !ok || !n.WrittenAsInteger() {
		return false
	}
	_, safe := n.SafeInt()
	return !safe
}

// markdown reads one Markdown file (ADR-003 D13): an object of the
// frontmatter's fields and body, the text after it.
func (r *reader) markdown(text []byte, loc finding.Location, prefix string) parsed {
	front, frontLine, body, fs := markdown.SplitFrontmatter(text, markdown.Source{Input: r.subj, At: loc}, r.lim)
	if len(fs) > 0 {
		return parsed{findings: r.place(fs, prefix)}
	}
	ml := markdownLocator{at: loc, bodyLine: 1}
	obj := newObject(ml)
	if front == nil {
		// The file's object, which a frontmatter's mapping is when it has
		// one (ADR-009 Amendment A4 item 1).
		if f, ok := r.count(1, loc, prefix); !ok {
			return parsed{findings: []finding.Finding{f}}
		}
	} else {
		ml.bodyLine = frontLine + bytes.Count(front, []byte("\n")) + 1
		doc, fs := yamlx.Parse(front, yamlx.Source{Subject: r.subj, At: loc, Size: limits.FrontmatterSize, FirstLine: frontLine, Values: r.values}, r.lim)
		if len(fs) > 0 {
			return parsed{findings: r.place(fs, prefix)}
		}
		ml.front = doc.Positions()
		obj.at = ml
		switch v := doc.Value.(type) {
		case value.Null:
		case value.Object:
			for name, m := range v.All() {
				if name == bodyField {
					l := loc.WithPointer(prefix + finding.Pointer(bodyField))
					if p, ok := doc.Positions().Key(finding.Pointer(bodyField)); ok {
						l = l.At(p)
					}
					return parsed{findings: []finding.Finding{finding.New(CodeFrontmatterBodyField, r.subj, &l, finding.Params{})}}
				}
				obj.obj[name] = build(m)
			}
		default:
			l := loc.At(finding.Pos{Line: frontLine, Column: 1})
			if p, ok := doc.Positions().Value(""); ok {
				l = loc.At(p)
			}
			return parsed{findings: []finding.Finding{finding.New(CodeFrontmatterNotMapping, r.subj, &l,
				finding.Params{"actual": finding.String(v.Kind().String())})}}
		}
	}
	if f, ok := noncharacters(body, r.subj, loc, ml.bodyLine); !ok {
		return parsed{findings: []finding.Finding{f}}
	}
	if f, ok := r.count(1, loc.At(finding.Pos{Line: ml.bodyLine, Column: 1}), prefix+finding.Pointer(bodyField)); !ok {
		return parsed{findings: []finding.Finding{f}}
	}
	obj.obj[bodyField] = &node{scalar: str(bytes.TrimSuffix(body, []byte("\n")))}
	return parsed{value: obj}
}

// firstToken is the first reference token of p, decoded, and "" for the
// root. It is read from the path, which costs nothing for the pointer's
// length.
func firstToken(p *finding.Path) string {
	if p.Depth() == 0 {
		return ""
	}
	return p.Tokens()[0]
}
