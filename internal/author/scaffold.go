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

// Scaffold returns the files of ADR-007 D1 item 4 for a package called
// name, which must satisfy ADR-002 D5's rule, in the order init writes
// them. Every file is UTF-8 with LF line endings and a final newline.
func Scaffold(name string) []File {
	return []File{
		{".gitattributes", []byte(gitattributes)},
		{"metais.yaml", manifestYAML(name)},
		{"inputs/greeting.schema.json", []byte(greetingSchema)},
		{"inputs/greeting.md", []byte(greetingMeaning)},
		{"templates/greeting.md.liquid", []byte(greetingTemplate)},
		{"examples/basic/example.yaml", []byte(exampleYAML)},
		{"examples/basic/inputs/greeting.json", []byte(exampleInputs)},
		{"examples/basic/expected/greeting.md", []byte(exampleExpected)},
	}
}

// manifestYAML is metais.yaml for name. It declares the highest specification
// version the tool implements (ADR-002 D7 item 10). Every string a YAML
// reader could take for another type is quoted: a name such as 1.0 or
// true satisfies the name rule.
func manifestYAML(name string) []byte {
	impl := version.Implements()
	return []byte(`metais: "` + impl[len(impl)-1] + `"
name: "` + name + `"
version: "0.1.0"
title: "` + Title(name) + `"
description: "Replace this line with what the package produces, in a sentence."
guide: |-
  This package produces greeting.md, a Markdown greeting for one person.
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
    template: greeting.md.liquid
`)
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
