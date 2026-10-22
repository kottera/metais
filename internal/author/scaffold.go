// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package author holds the authoring commands' work: init's scaffold here,
// and check, test, pack and unpack as they land (ADR-010 D2, ADR-007).
package author

import (
	"strings"

	"github.com/kottera/metais/internal/version"
)

// File is one file the scaffold writes: its "/"-separated path relative to
// the directory init writes into, and its bytes.
type File struct {
	Path string
	Data []byte
}

// Form is which of init's scaffolds to write (ADR-007 D1 item 4, A1).
type Form struct {
	// Minimal writes the layout, one input and one template to replace,
	// without the greeting and its example (A1 item 1).
	Minimal bool
	// HTML makes the output an HTML page whose output declares escape:
	// html, in place of Markdown (A1 item 2).
	HTML bool
}

// Scaffold returns the files of ADR-007 D1 item 4, in form as A1 amends
// it, for a package called name, which must satisfy ADR-002 D5's rule, in
// the order init writes them. Every file is UTF-8 with LF line endings and
// a final newline.
func Scaffold(name string, form Form) []File {
	if form.Minimal {
		template := File{"templates/document.md.liquid", []byte(documentTemplate)}
		if form.HTML {
			template = File{"templates/page.html.liquid", []byte(pageTemplate)}
		}
		return []File{
			{".gitattributes", []byte(gitattributes)},
			{"metais.yaml", minimalManifestYAML(name, form.HTML)},
			{"inputs/data.schema.json", []byte(dataSchema)},
			{"inputs/data.md", []byte(dataMeaning)},
			template,
		}
	}
	ext, template, inputs, expected := "md", greetingTemplate, exampleInputs, exampleExpected
	if form.HTML {
		ext, template, inputs, expected = "html", greetingHTMLTemplate, exampleHTMLInputs, exampleHTMLExpected
	}
	return []File{
		{".gitattributes", []byte(gitattributes)},
		{"metais.yaml", manifestYAML(name, form.HTML)},
		{"inputs/greeting.schema.json", []byte(greetingSchema)},
		{"inputs/greeting.md", []byte(strings.ReplaceAll(greetingMeaning, "greeting.md,", "greeting."+ext+","))},
		{"templates/greeting." + ext + ".liquid", []byte(template)},
		{"examples/basic/example.yaml", []byte(exampleYAML)},
		{"examples/basic/inputs/greeting.json", []byte(inputs)},
		{"examples/basic/expected/greeting." + ext, []byte(expected)},
	}
}

// specVersion is the highest specification version the tool implements,
// which every scaffold declares (ADR-002 D7 item 10).
func specVersion() string {
	impl := version.Implements()
	return impl[len(impl)-1]
}

// manifestYAML is metais.yaml for name. Every string a YAML reader could
// take for another type is quoted: a name such as 1.0 or true satisfies
// the name rule.
func manifestYAML(name string, html bool) []byte {
	produces, output := "greeting.md, a Markdown greeting", "    template: greeting.md.liquid\n"
	if html {
		produces, output = "greeting.html, an HTML page greeting", "    template: greeting.html.liquid\n    escape: html\n"
	}
	return []byte(`metais: "` + specVersion() + `"
name: "` + name + `"
version: "0.1.0"
title: "` + Title(name) + `"
description: "Replace this line with what the package produces, in a sentence."
guide: |-
  This package produces ` + produces + ` for one person.
  Read [what the greeting means](inputs/greeting.md) before you supply it.
inputs:
  - name: greeting
    title: "Greeting"
    schema: inputs/greeting.schema.json
    meaning: inputs/greeting.md
    forms:
      - json
      - yaml
outputs:
  - id: greeting
` + output)
}

// minimalManifestYAML is the minimal scaffold's metais.yaml for name
// (A1 item 1).
func minimalManifestYAML(name string, html bool) []byte {
	output := "  - id: document\n    template: document.md.liquid\n"
	if html {
		output = "  - id: page\n    template: page.html.liquid\n    escape: html\n"
	}
	return []byte(`metais: "` + specVersion() + `"
name: "` + name + `"
version: "0.1.0"
title: "` + Title(name) + `"
description: "Replace this line with what the package produces, in a sentence."
guide: |-
  Replace this guide with what the package produces and what a supplier
  gives it. Read [what the data means](inputs/data.md) before you supply it.
inputs:
  - name: data
    title: "Data"
    schema: inputs/data.schema.json
    meaning: inputs/data.md
    forms:
      - json
      - yaml
outputs:
` + output)
}

// Title is the scaffold's title for name: the name with each - and _ as
// a space, and each word capitalised (ADR-007 D1 item 4).
func Title(name string) string {
	words := strings.Fields(strings.NewReplacer("-", " ", "_", " ").Replace(name))
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + w[1:]
	}
	return strings.Join(words, " ")
}

// .gitattributes stops git from converting line endings in a checkout,
// which would change templates and expected files on Windows and break
// test's byte comparison.
const gitattributes = "* -text\n"

const greetingSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Greeting",
  "description": "Who the greeting is for, and an optional note printed beneath it.",
  "type": "object",
  "additionalProperties": false,
  "required": ["name"],
  "properties": {
    "name": {
      "type": "string",
      "minLength": 1,
      "maxLength": 80,
      "title": "Name",
      "description": "The name of the person greeted, as the heading prints it.",
      "examples": ["Ada Lovelace"]
    },
    "note": {
      "type": "string",
      "maxLength": 400,
      "title": "Note",
      "description": "A short message printed beneath the heading. Leaving it out prints nothing there."
    }
  }
}
`

const greetingMeaning = `# Greeting

The greeting is the person a generated document welcomes. It gives the
heading of greeting.md, and an optional note beneath it.

- **name** is the person's name as it should be printed, such as
  "Ada Lovelace".
- **note** is one or two sentences shown beneath the heading: a welcome, a
  reminder, or what the reader should do next. A good note speaks to the
  person by what they are about to do, not by who they are. Leave it out
  when there is nothing to add, and the document holds the heading alone.
`

// greetingTemplate is exactly ADR-007 D1 item 4's template. An absent
// field is nil, the only falsy value besides false, so if tests presence.
const greetingTemplate = `# Hello, {{ inputs.greeting.name }}
{%- if inputs.greeting.note %}

{{ inputs.greeting.note }}
{%- endif %}
`

const exampleYAML = `title: Basic
description: The scaffold's example, as metais init wrote it.
`

const exampleInputs = `{
  "name": "Ada Lovelace",
  "note": "Welcome."
}
`

const exampleExpected = "# Hello, Ada Lovelace\n\nWelcome.\n"

// htmlComment opens each HTML template: what escape: html does, and how
// data reaches a script (A1 item 2).
const htmlComment = `{%- comment %}
metais.yaml declares this output's escape: html, so every value written
with {{ }} is HTML-escaped. Data a script reads goes inside a <script>
element through json_script, which no value can break out of; never json.
{% endcomment -%}
`

// greetingHTMLTemplate is the greeting as an HTML page (A1 item 2). The
// note is escaped where it is written, and the whole greeting reaches the
// page's script through json_script.
const greetingHTMLTemplate = htmlComment + `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Hello, {{ inputs.greeting.name }}</title>
</head>
<body>
<h1>Hello, {{ inputs.greeting.name }}</h1>
{%- if inputs.greeting.note %}
<p>{{ inputs.greeting.note }}</p>
{%- endif %}
<script type="application/json" id="greeting">{{ inputs.greeting | json_script }}</script>
</body>
</html>
`

// exampleHTMLInputs gives the note an & so the expected page shows both
// escapes: &amp; in the text, \u0026 in the script.
const exampleHTMLInputs = `{
  "name": "Ada Lovelace",
  "note": "Welcome to R&D."
}
`

const exampleHTMLExpected = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>Hello, Ada Lovelace</title>
</head>
<body>
<h1>Hello, Ada Lovelace</h1>
<p>Welcome to R&amp;D.</p>
<script type="application/json" id="greeting">{"name":"Ada Lovelace","note":"Welcome to R\u0026D."}</script>
</body>
</html>
`

const dataSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "title": "Data",
  "description": "Replace this schema with the fields the package needs.",
  "type": "object",
  "additionalProperties": false,
  "required": ["title"],
  "properties": {
    "title": {
      "type": "string",
      "minLength": 1,
      "maxLength": 200,
      "title": "Title",
      "description": "The title the document prints as its heading."
    }
  }
}
`

const dataMeaning = `# Data

Replace this document with what the data means: what each field holds, and
where a supplier finds it.

- **title** is the document's title, printed as its heading.
`

const documentTemplate = "# {{ inputs.data.title }}\n"

const pageTemplate = htmlComment + `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<title>{{ inputs.data.title }}</title>
</head>
<body>
<h1>{{ inputs.data.title }}</h1>
<script type="application/json" id="data">{{ inputs.data | json_script }}</script>
</body>
</html>
`
