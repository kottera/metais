// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"strings"
	"unicode/utf8"
)

// templateInputs is the inputs every example of the templates topic reads,
// as JSON, and templateInputsText says them as the topic prints them. The
// topic's tests render every example over exactly this value.
const templateInputs = `{"inputs": {"rows": [{"name": "b", "n": 2, "tag": "new"}, {"name": "a", "n": 1, "tag": null}], ` +
	`"note": "line 1\nline 2", "title": null}}`

var templateInputsText = [][2]string{
	{"inputs.rows", `[{"name": "b", "n": 2, "tag": "new"}, {"name": "a", "n": 1, "tag": null}]`},
	{"inputs.note", `"line 1\nline 2"`},
	{"inputs.title", "null: declared by the schema, and not supplied"},
}

// templateExample is one worked example of the templates topic: a
// template, the partial templates/row.md when it renders one, and what it
// renders over templateInputs. html renders it in an output whose escape is
// html.
type templateExample struct {
	intro   string
	partial string
	src     string
	gives   string
	html    bool
}

// leadExamples open the topic, before any explanation (ADR-008 A3).
var leadExamples = []templateExample{
	{
		intro: "A loop over an input, with forloop:",
		src: "{% for row in inputs.rows -%}\n" +
			"{{ forloop.index }} of {{ forloop.length }}: {{ row.name }}\n" +
			"{% endfor -%}",
		gives: "1 of 2: b\n2 of 2: a\n",
	},
	{
		intro: "assign, a filter chain, and a test for nil:",
		src: "{% assign names = inputs.rows | map: \"name\" | sort -%}\n" +
			"{% if inputs.title %}{{ inputs.title }}{% else %}Untitled{% endif %}\n" +
			"{{ names | join: \", \" }}",
		gives: "Untitled\na, b",
	},
	{
		intro:   "A partial, rendered once per element:",
		partial: "- {{ row.name }} ({{ row.n }})\n",
		src:     `{% render "row.md" for inputs.rows as row %}`,
		gives:   "- b (2)\n- a (1)\n",
	},
}

// escapeExamples are the escaping section's examples, each rendered in an
// output whose escape is html.
var escapeExamples = []templateExample{
	{src: `<p>{{ "Fish & <Chips>" }}</p>`, gives: "<p>Fish &amp; &lt;Chips&gt;</p>", html: true},
	{src: `{{ "<b>bold</b>" | safe }} {{ "<b>" | escape }}`, gives: "<b>bold</b> &lt;b&gt;", html: true},
	{src: `{{ inputs.note | escape | newline_to_br | safe }}`, gives: "line 1<br />\nline 2", html: true},
	{src: `<a href="/find?q={{ "a b&c" | url_encode }}">`, gives: `<a href="/find?q=a+b%26c">`, html: true},
	{
		src:   `<script type="application/json">{{ "a</script>" | json_script }}</script>`,
		gives: `<script type="application/json">"a\u003c/script\u003e"</script>`,
		html:  true,
	},
}

// filterDoc is one filter of the dialect as the templates topic gives it:
// what it does in a line, a template applying it to literals or to
// templateInputs, and exactly what that template renders in an output
// that escapes nothing.
type filterDoc struct {
	Name, Line, Example, Gives string
}

// filterGroup is a heading of the topic's filter list and its filters.
type filterGroup struct {
	title string
	docs  []filterDoc
}

// filterGroups is every filter of template.Dialect(), each exactly once
// (TestTemplatesTopicFilters).
var filterGroups = []filterGroup{
	{"Strings:", []filterDoc{
		{"append", "Adds text at the end.", `{{ "doc" | append: ".md" }}`, "doc.md"},
		{"prepend", "Adds text at the start.", `{{ "md" | prepend: "." }}`, ".md"},
		{"capitalize", "Upper-cases the first character and lower-cases the rest.", `{{ "hELLO" | capitalize }}`, "Hello"},
		{"downcase", "Lower-cases every letter, by Unicode's simple case mapping.", `{{ "AbC" | downcase }}`, "abc"},
		{"upcase", "Upper-cases every letter, by Unicode's simple case mapping.", `{{ "abc" | upcase }}`, "ABC"},
		{"strip", "Removes the whitespace at both ends.", `{{ " a b " | strip }}`, "a b"},
		{"lstrip", "Removes the whitespace at the start.", `{{ " a " | lstrip | append: "]" }}`, "a ]"},
		{"rstrip", "Removes the whitespace at the end.", `{{ " a " | rstrip | prepend: "[" }}`, "[ a"},
		{"squish", "Strips both ends and makes each run of whitespace one space.", `{{ inputs.note | squish }}`, "line 1 line 2"},
		{"strip_html", "Removes HTML tags.", `{{ "<b>bold</b> text" | strip_html }}`, "bold text"},
		{"strip_newlines", "Removes every line break.", `{{ inputs.note | strip_newlines }}`, "line 1line 2"},
		{"newline_to_br", "Writes <br /> before each line break.", `{{ inputs.note | newline_to_br }}`, "line 1<br />\nline 2"},
		{"remove", "Removes every occurrence of a text.", `{{ "a-b-c" | remove: "-" }}`, "abc"},
		{"remove_first", "Removes the first occurrence.", `{{ "a-b-c" | remove_first: "-" }}`, "ab-c"},
		{"remove_last", "Removes the last occurrence.", `{{ "a-b-c" | remove_last: "-" }}`, "a-bc"},
		{"replace", "Replaces every occurrence of a text with another.", `{{ "a-b-c" | replace: "-", "+" }}`, "a+b+c"},
		{"replace_first", "Replaces the first occurrence.", `{{ "a-b-c" | replace_first: "-", "+" }}`, "a+b-c"},
		{"replace_last", "Replaces the last occurrence.", `{{ "a-b-c" | replace_last: "-", "+" }}`, "a-b+c"},
		{"split", "Splits a string into an array at each separator.", `{{ "a,b,c" | split: "," | join: " " }}`, "a b c"},
		{"truncate", "Cuts a string to a number of characters, the ellipsis (... unless given) included.", `{{ "abcdefgh" | truncate: 6 }}`, "abc..."},
		{"truncatewords", "Keeps a number of words, then the ellipsis (... unless given).", `{{ "one two three" | truncatewords: 2 }}`, "one two..."},
	}},
	{"Numbers (integers give an integer):", []filterDoc{
		{"abs", "The absolute value.", `{{ -3 | abs }}`, "3"},
		{"ceil", "Rounds up to an integer.", `{{ 1.2 | ceil }}`, "2"},
		{"floor", "Rounds down to an integer.", `{{ 1.8 | floor }}`, "1"},
		{"round", "Rounds half away from zero, to a number of decimal places when given.", `{{ 2.5 | round }}`, "3"},
		{"plus", "Adds; a float adds as the decimal it is written as.", `{{ 0.1 | plus: 0.2 }}`, "0.3"},
		{"minus", "Subtracts.", `{{ 5 | minus: 7 }}`, "-2"},
		{"times", "Multiplies.", `{{ 3 | times: 1.5 }}`, "4.5"},
		{"divided_by", "Divides; two integers give the quotient rounded down.", `{{ 7 | divided_by: 2 }}`, "3"},
		{"modulo", "The remainder of the division rounded down.", `{{ -7 | modulo: 2 }}`, "1"},
		{"at_least", "The larger of the value and the argument.", `{{ 3 | at_least: 5 }}`, "5"},
		{"at_most", "The smaller of the value and the argument.", `{{ 3 | at_most: 2 }}`, "2"},
	}},
	{"Arrays:", []filterDoc{
		{"size", "The length of a string, array or object; 0 for a number, a boolean and nil.", `{{ "abc" | size }}`, "3"},
		{"first", "The first element.", `{{ "a,b,c" | split: "," | first }}`, "a"},
		{"last", "The last element.", `{{ "a,b,c" | split: "," | last }}`, "c"},
		{"join", "The elements' texts, with a separator between them (a space unless given).", `{{ "a,b" | split: "," | join: " + " }}`, "a + b"},
		{"reverse", "The elements in reverse order.", `{{ "a,b,c" | split: "," | reverse | join: "," }}`, "c,b,a"},
		{"concat", "The elements of both arrays.", `{{ (1..2) | concat: (3..4) | join: "," }}`, "1,2,3,4"},
		{"compact", "Drops nil elements, or the elements whose property is nil.", `{{ inputs.rows | map: "tag" | compact | join: "," }}`, "new"},
		{"uniq", "Drops repeated elements.", `{{ "a,b,a" | split: "," | uniq | join: "," }}`, "a,b"},
		{"map", "Each element's property.", `{{ inputs.rows | map: "name" | join: "," }}`, "b,a"},
		{"sort", "Sorts, by a property when given.", `{{ inputs.rows | sort: "n" | map: "name" | join: "," }}`, "a,b"},
		{"sort_natural", "Sorts ignoring case.", `{{ "b,C,a" | split: "," | sort_natural | join: "," }}`, "a,b,C"},
		{"slice", "The element or character at an offset, or as many as a length gives.", `{{ "abcdef" | slice: 1, 3 }}`, "bcd"},
		{"sum", "Adds the elements, or their property.", `{{ inputs.rows | sum: "n" }}`, "3"},
		{"where", "Keeps the elements whose property is truthy, or equals a value.", `{{ inputs.rows | where: "n", 1 | map: "name" | join }}`, "a"},
		{"reject", "Drops the elements whose property is truthy, or equals a value.", `{{ inputs.rows | reject: "tag" | map: "name" | join }}`, "a"},
		{"has", "Whether an element's property is truthy, or equals a value.", `{{ inputs.rows | has: "name", "a" }}`, "true"},
		{"find", "The first element whose property is truthy, or equals a value.", `{{ inputs.rows | find: "n", 1 | json }}`, `{"n":1,"name":"a","tag":null}`},
		{"find_index", "The position of that element, from 0.", `{{ inputs.rows | find_index: "name", "a" }}`, "1"},
		{"default", "The value, or its argument when the value is nil, false, or an empty string, array or object; allow_false: true keeps false.", `{{ inputs.title | default: "Untitled" }}`, "Untitled"},
	}},
	{"Encoding (escape, h and escape_once give a safe value):", []filterDoc{
		{"escape", "Writes & < > \" and ' as HTML character references.", `{{ "a < b" | escape }}`, "a &lt; b"},
		{"h", "escape, by another name.", `{{ "a & b" | h }}`, "a &amp; b"},
		{"escape_once", "escape, but leaves an & that already begins a character reference.", `{{ "&lt; & <" | escape_once }}`, "&lt; &amp; &lt;"},
		{"url_encode", "Percent-encodes a URL component; a space is +.", `{{ "a b&c" | url_encode }}`, "a+b%26c"},
		{"url_decode", "Decodes what url_encode writes.", `{{ "a+b%26c" | url_decode }}`, "a b&c"},
		{"base64_encode", "Base64.", `{{ "hi?" | base64_encode }}`, "aGk/"},
		{"base64_decode", "Decodes base64.", `{{ "aGk/" | base64_decode }}`, "hi?"},
		{"base64_url_safe_encode", "Base64 with - and _ for + and /.", `{{ "hi?" | base64_url_safe_encode }}`, "aGk_"},
		{"base64_url_safe_decode", "Decodes URL-safe base64.", `{{ "aGk_" | base64_url_safe_decode }}`, "hi?"},
	}},
	{"METAIS's own (spec chapter 8):", []filterDoc{
		{"json", "RFC 8785 JSON: members in name order, no whitespace; indent: n (1 to 8) pretty-prints.", `{{ inputs.rows | last | json }}`, `{"n":1,"name":"a","tag":null}`},
		{"yaml", "A YAML block document, members in name order, with no final line feed; a string YAML could read otherwise, such as n, is quoted.", `{{ inputs.rows | first | yaml }}`, "\"n\": 2\nname: b\ntag: new"},
		{"csv", "An array of objects as CSV, of the columns named or else every member; header: false, headers: and delimiter: change it.", `{{ inputs.rows | csv: "name", "n" }}`, "name,n\nb,2\na,1\n"},
		{"markdown_table", "An array of objects as a Markdown table, with the columns and headers: of csv.", `{{ inputs.rows | markdown_table: "name", "n" }}`, "| name | n |\n| --- | --- |\n| b | 2 |\n| a | 1 |\n"},
		{"json_script", "json for a <script> element: < > & U+2028 and U+2029 written as \\u escapes; safe.", `{{ "a&b" | json_script }}`, `"a\u0026b"`},
		{"safe", "Marks a value safe from escaping; only as the last filter of {{ }} or echo.", `{{ "<b>" | safe }}`, "<b>"},
		{"slug", "Lower-case ASCII letters and digits joined by -; accents are dropped, and a string with no slug is an error.", `{{ "Hello, World!" | slug }}`, "hello-world"},
		{"date", "Formats an RFC 3339 date or date-time, or Unix seconds, in its own offset, with a strftime format; spec chapter 8 lists the directives.", `{{ "2026-10-18T09:30:00+02:00" | date: "%Y-%m-%d %H:%M %z" }}`, "2026-10-18 09:30 +0200"},
		{"number", "Writes a number or a decimal text for a reader, with decimals:, group: and point:; no group separator and a point . unless given.", `{{ 48920000 | number: decimals: 2, group: "," }}`, "48,920,000.00"},
		{"amount", "Writes an amount in the currency's major unit with its ISO 4217 code, its decimals from spec Appendix C unless decimals: gives them.", `{{ 48920000 | amount: "EUR", group: "," }}`, "EUR 48,920,000.00"},
		{"duration", "Writes seconds, an ISO 8601 duration, or fields: \"d:h:m\" and the like, with %d %H %M %S, as date's directives are written.", `{{ "00:07:25" | duration: "%-H h %M min", fields: "d:h:m" }}`, "7 h 25 min"},
	}},
}

// filterDocs is every row of filterGroups, in order.
func filterDocs() []filterDoc {
	var out []filterDoc
	for _, g := range filterGroups {
		out = append(out, g.docs...)
	}
	return out
}

// filterWidth is the column the filter list's descriptions begin at.
const filterWidth = 18

// templatesTopic is metais help templates (ADR-008 A6): worked examples
// first, then the template environment (spec chapter 7), every filter of
// the dialect with an example (chapter 8), and escaping (chapter 7.10).
func templatesTopic(h *helpText, _ []*Command) {
	h.para("A template is Liquid, in the dialect spec chapter 7 pins. The examples on this "+
		"page read these inputs:", "")
	h.blank()
	for _, row := range templateInputsText {
		h.item("  "+row[0], 16, row[1])
	}
	for _, e := range leadExamples {
		h.blank()
		templateExampleText(h, e)
	}

	h.heading("The environment:")
	for _, row := range [][2]string{
		{"inputs", "One member per declared input: its validated value, defaults applied. An optional input not supplied is nil."},
		{"package", "The manifest's metais, name, version, title, description, guide, authors, license and homepage, and digest, the package digest."},
		{"output", "In a template body: path, the output's path, and root, the relative prefix from its directory to the root of its tree (\"\", \"../\", ...)."},
		{"now", "The render time the caller supplied, in UTC (2026-10-18T09:30:00Z), or nil. It is a template's only clock."},
		{"sources", "Where each input came from: origin is file, command-line or default; for a collection, each item's file name, stem and form."},
		{"item", "In an output with for_each only: the element being rendered."},
		{"fan_out", "Beside item: index, index0, rindex, rindex0, first, last, length, previous and next."},
	} {
		h.item("  "+row[0], 11, row[1])
	}
	h.blank()
	h.para("Names are strict. A name nothing binds is an error (template.undefined_variable), "+
		"and so is a property the schema does not declare and the data does not hold "+
		"(template.undefined_property), never an empty string. A property the schema "+
		"declares and the data omits is nil, and nil writes nothing. Reading a property of "+
		"nil is an error (template.nil_property), whatever filter follows, default included; "+
		"{% if a and a.b %} reads safely. Only nil and false are false: 0, \"\", [] and {} are "+
		"true, so {% if inputs.x %} asks whether x was supplied. x == empty holds for an "+
		"empty string, array or object; x == blank also for nil, false and whitespace.", "")
	h.blank()
	h.para("for loops over an array, a range (1..3) or an object, whose members it visits "+
		"in name order as [name, value] pairs; it takes limit:, offset: and reversed, and "+
		"else for an empty collection, and break and continue. Inside it, forloop has index and index0 (from 1 "+
		"and 0), rindex and rindex0 (from the end), first, last, length, parentloop and "+
		"name. {%- and -%} remove the whitespace before and after a tag; nothing else "+
		"is trimmed, so a tag alone on its line leaves its line feed.", "")
	h.blank()
	h.para("assign binds a value and capture binds the text its block writes, for the rest "+
		"of the template; neither may take a global's name, forloop or tablerowloop "+
		"(template.reserved_name).", "")
	h.blank()
	h.para("render \"<path>\" renders a partial: a template under templates/, named by its "+
		"full path there as a string literal (\"parts/row.md\"). It takes k: v parameters, "+
		"with x [as n], or for xs [as n], which renders it once per element and binds "+
		"forloop. A partial sees its parameters, its own bindings and the globals, and "+
		"none of its caller's assign, capture or loop variables. include is refused.", "")
	h.blank()
	h.line("METAIS adds three tags, whose text is never escaped:")
	for _, row := range [][2]string{
		{"embed", "{% embed \"<input>\" %} writes an input's bytes exactly as its data-directory file holds them; index: n names an item of a collection."},
		{"asset", "{% asset \"<path>\" %} writes the relative path from the file being rendered to where the render copies an asset under assets/; as: \"contents\" writes its text, as: \"base64\" its bytes."},
		{"provenance", "{% provenance all %} writes one comment line naming the package, its version and digest, the inputs' digests, the tool and the render time when one was given; name parts to write fewer."},
	} {
		h.item("  "+row[0], 14, row[1])
	}

	h.heading("Filters:")
	h.para("Every filter of the dialect, with an example and what it gives. A filter's "+
		"keyword argument follows its positional ones: csv: \"name\", delimiter: \";\". "+
		"In what an example gives, \\n is a line feed.", "")
	for _, g := range filterGroups {
		h.blank()
		h.line(g.title)
		for _, d := range g.docs {
			h.item("  "+d.Name, filterWidth, d.Line)
			h.example(d.Example, d.Gives, filterWidth)
		}
	}

	h.heading("Escaping:")
	h.para("An output's escape is none, the default, or html, set in metais.yaml beside "+
		"its template (escape: html); nothing is escaped unless it is set. In an html output, the text of "+
		"every value {{ }}, echo or cycle writes has & < > \" and ' written as character "+
		"references; the template's own text, and what raw, embed, asset and provenance "+
		"write, are never escaped. A value is safe, and written as it is, when its last "+
		"filter is safe, escape, escape_once or json_script, or it is a capture made in "+
		"an html output. Safety does not pass through a later filter. A partial is "+
		"escaped as the output it renders into is.", "")
	h.blank()
	h.para("Escaping does not make text safe inside a <script> element or a URL: data in a "+
		"<script> element goes through json_script, whose text can neither close the element "+
		"nor open an HTML comment, and a URL component through url_encode; the dialect "+
		"offers nothing else for those contexts. In an html output:", "")
	for _, e := range escapeExamples {
		h.blank()
		templateExampleText(h, e)
	}

	h.blank()
	h.para("'metais help manifest' gives the fields of metais.yaml; spec chapters 7, 8 and 9 "+
		"give the environment, the filters and tags, and the outputs in full.", "")
}

// templateExampleText writes e: its introduction, its template indented
// four columns, and what it gives, each line kept whole.
func templateExampleText(h *helpText, e templateExample) {
	if e.intro != "" {
		h.para(e.intro, "  ")
		h.blank()
	}
	if e.partial != "" {
		h.line("  templates/row.md holds")
		for _, l := range strings.Split(strings.TrimSuffix(e.partial, "\n"), "\n") {
			h.line("    " + l)
		}
		h.line("  and the template")
	}
	for _, l := range strings.Split(e.src, "\n") {
		h.line("    " + l)
	}
	h.line("  gives")
	for _, l := range strings.Split(strings.TrimSuffix(e.gives, "\n"), "\n") {
		h.line("    " + l)
	}
}

// example writes a filter's example and what it gives beside the last
// line written when they fit there, and otherwise on lines of their own
// indented by width; neither is ever broken, so an example's spaces are
// kept.
func (h *helpText) example(ex, gives string, width int) {
	text := ex + " gives " + strings.ReplaceAll(gives, "\n", `\n`)
	if width+utf8.RuneCountInString(text) <= helpWidth {
		h.join(text, width)
		return
	}
	indent := strings.Repeat(" ", width)
	h.line(indent + ex)
	h.line(indent + "gives " + strings.ReplaceAll(gives, "\n", `\n`))
}
