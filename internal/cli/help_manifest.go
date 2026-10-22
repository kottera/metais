// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import "strings"

// topicManifestHead is the leading example's manifest up to its outputs:
// every field of the top level, an author, and two inputs, one of them
// read from CSV too. Each example manifest is topicManifestHead followed
// by one of the outputs sections below, and TestManifestTopicExample
// builds each.
const topicManifestHead = `metais: "1.0"
name: org.example.release-notes
version: "1.2.0"
title: Release notes
description: Release notes for one product release, in Markdown and HTML.
guide: |-
  Supply the release first, then its changes.

  Read inputs/changes.md for what a good change says.
authors:
  - name: Example Docs Team
    email: docs@example.org
    url: https://example.org/docs
license: Apache-2.0
homepage: https://example.org/release-notes
inputs:
  - name: release
    title: The release
    schema: inputs/release.schema.json
  - name: changes
    title: The changes in the release
    description: One change per row, newest first.
    schema: inputs/changes.schema.json
    meaning: inputs/changes.md
    forms: [json, yaml, csv]
    csv:
      delimiter: ";"
`

// topicManifestOutputs is the leading example's outputs: a Markdown
// template output whose path defaults from its template, and an HTML one
// that escapes.
const topicManifestOutputs = `outputs:
  - id: notes
    template: notes.md.liquid
  - id: page
    template: page.html.liquid
    path: "{{ inputs.release.version }}/index.html"
    escape: html
`

// topicManifestFanOut is the second example's outputs: a template output
// repeated with for_each, and a zip container with an entry of each other
// form.
const topicManifestFanOut = `outputs:
  - id: change_pages
    template: change.md.liquid
    for_each: inputs.changes
    when: item.public
    path: "changes/{{ item.id }}.md"
  - id: bundle
    path: "release-{{ inputs.release.version }}.zip"
    container:
      format: zip
      entries:
        - template: notes.md.liquid
          path: README.md
          first: true
        - asset: logo.png
          path: images/logo.png
          stored: true
        - mirror: site
          path: site
          escape: html
`

// manifestField is one field of metais.yaml as help manifest gives it:
// its path from the manifest's root, a list's members written [], what it
// means, and a YAML line that sets it.
type manifestField struct {
	Path    string
	Text    string
	Example string
}

// manifestFields is every field of the 1.0 manifest schema, grouped by
// the object that holds it. A container entry takes the fields of
// outputs[], so only the fields of an entry alone have a row of their own
// beneath outputs[].container.entries[]. TestManifestTopicFields holds
// the rows to the schema.
var manifestFields = []manifestField{
	{"metais", "Required. The METAIS specification version the package targets, as a quoted " +
		"string: unquoted, 1.0 is the number 1. A tool processes the package under exactly " +
		"that version's rules.", `metais: "1.0"`},
	{"name", "Required. The package's name: segments of lower-case ASCII letters and digits, " +
		"single hyphens between them, joined by dots. A segment is at most 63 characters and " +
		"the name at most 128, and the first segment is not a Windows device name (con, prn, " +
		"aux, nul, com0 to com9, lpt0 to lpt9). A package published beyond its author should " +
		"begin with a domain its publisher controls, reversed.", "name: org.example.release-notes"},
	{"version", "Required. The package's own version, a Semantic Versioning 2.0.0 version " +
		"of at most 128 characters, without a leading v and quoted. Pre-release and build " +
		"metadata are allowed.", `version: "1.2.0-rc.1"`},
	{"title", "Required. What the package is, as a person names it: one line of 1 to 200 " +
		"characters.", "title: Release notes"},
	{"description", "Required. What the package produces, in a sentence or a short " +
		"paragraph of 1 to 1000 characters, displayed verbatim.",
		"description: Release notes for one product release."},
	{"guide", "Required. The guide for suppliers, in CommonMark, of 1 to 16384 characters: " +
		"what the package is for, how its inputs relate, the order to gather them in, and " +
		"where to read further. It may not end with a line break, so write a block as " +
		"guide: |- rather than guide: |.", "guide: Supply the release first, then its changes."},
	{"authors", "Optional. The package's authors, 1 to 64 people or organizations, each an " +
		"object with the fields under authors[].", "authors: [{name: Example Docs Team}]"},
	{"license", "Optional. The terms of the package's own content, an SPDX license " +
		"expression of 1 to 512 characters with AND, OR and WITH in upper case. Identifiers " +
		"are not checked against the SPDX License List.", "license: Apache-2.0 OR MIT"},
	{"homepage", "Optional. The package's home page, an absolute https or http URI of at " +
		"most 2048 characters, with a host, no user information and no fragment. No command " +
		"ever fetches it.", "homepage: https://example.org/release-notes"},
	{"inputs", "Required. The inputs the package takes, in the order a supplier meets them, " +
		"each an object with the fields under inputs[]. It may be empty.", "inputs: []"},
	{"input_constraints", "Optional. The package path, under inputs/ and ending .json, " +
		".yaml or .yml, of a schema evaluated against the object holding every input, for " +
		"rules that relate one input to another.", "input_constraints: inputs/constraints.schema.json"},
	{"outputs", "Required. The output declarations, at least one, in the order they are " +
		"planned, each with the fields under outputs[].",
		"outputs: [{id: notes, template: notes.md.liquid}]"},

	{"authors[].name", "Required. The person or organization: one line of 1 to 200 " +
		"characters.", "name: Ada Lovelace"},
	{"authors[].email", "Optional. An email address of 3 to 254 characters, with exactly " +
		"one @ and no space. It is not validated further, and no command sends mail.",
		"email: ada@example.org"},
	{"authors[].url", "Optional. The author's web page, an absolute https or http URI " +
		"with a host and no user information, as homepage.", "url: https://example.org/ada"},

	{"inputs[].name", "Required. The input's name, unique among the inputs: a lower-case " +
		"letter, then up to 63 lower-case letters, digits and underscores, and none of " +
		"Liquid's reserved words blank, empty, false, first, last, nil, null, size and true. " +
		"A template reads the input as inputs.<name>, a data directory's file for it is " +
		"named after it, and it is the first token of a --set pointer.", "name: changes"},
	{"inputs[].title", "Required. What the input is, in brief: one line of 1 to 200 " +
		"characters.", "title: The changes in the release"},
	{"inputs[].description", "Optional. What the input means, as plain text, line breaks " +
		"allowed.", "description: One change per row, newest first."},
	{"inputs[].schema", "Required. The package path, under inputs/, of the input's JSON " +
		"Schema file, ending .json, .yaml or .yml.", "schema: inputs/changes.schema.json"},
	{"inputs[].meaning", "Optional. The package path, under inputs/, of a CommonMark " +
		"document ending .md that says what the data is for and what good data looks like. " +
		"No tool interprets it.", "meaning: inputs/changes.md"},
	{"inputs[].forms", "Optional. The file forms the input may be supplied in, each at " +
		"most once, of json, yaml, csv, text and markdown. Default: json and yaml.",
		"forms: [json, yaml, csv]"},
	{"inputs[].csv", "Optional, and allowed only when forms holds csv. How a CSV file of " +
		"the input is read: an object whose one member is delimiter.", `csv: {delimiter: ";"}`},
	{"inputs[].csv.delimiter", "Required in csv. The character that separates fields: a " +
		"comma, a semicolon, a vertical bar or a tab.", `delimiter: "\t"`},
	{"inputs[].collection", "Optional. true for a collection: an array of items, supplied " +
		"as a directory of item files. Default false.", "collection: true"},
	{"inputs[].required", "Optional. false for an input a supplier may leave out. Default " +
		"true.", "required: false"},

	{"outputs[].id", "Every form. Required at the top level and optional on a container " +
		"entry. The declaration's identifier: a lower-case letter, then up to 63 lower-case " +
		"letters, digits and underscores, unique among the top-level declarations, and an " +
		"entry's among its container's entries.", "id: change_pages"},
	{"outputs[].template", "The template form's source key: the template's path relative " +
		"to templates/, rendered to one file per evaluation.", "template: notes.md.liquid"},
	{"outputs[].asset", "The asset form's source key: a file or a directory relative to " +
		"assets/, . being the whole of assets/. Every file is copied byte for byte, never " +
		"rendered.", "asset: images"},
	{"outputs[].mirror", "The mirror form's source key: a directory relative to templates/, " +
		". being the whole of templates/, whose every template is rendered to its relative " +
		"path with a final .liquid removed. A template whose name, or a directory's between " +
		"it and the mirror's, begins with _ is a partial and is not mirrored, and nor is a " +
		"template a template field names.", "mirror: site"},
	{"outputs[].container", "The container form's source key: the zip's shape, an object " +
		"with the fields under outputs[].container.", "container: {entries: [{template: notes.md.liquid}]}"},
	{"outputs[].path", "Every form. Required on a container and with for_each, and " +
		"optional otherwise. The destination relative to the root of the tree, a Liquid " +
		"template that is never escaped; for an asset directory or a mirror it names a " +
		"directory, . being the root. Without it, a template output is written to its " +
		"template's path with a final .liquid removed, an asset to its path under assets/, " +
		"and a mirror beneath its directory. Quote a path that begins with {.",
		`path: "changes/{{ item.id }}.md"`},
	{"outputs[].when", "Every form. Optional. A Liquid condition, what may follow if in " +
		"{% if %}: the declaration writes files only where it holds, and without when it " +
		"always holds. With for_each it is judged once per element.", "when: inputs.release.public"},
	{"outputs[].for_each", "Template and container forms. Optional, and needs path. A " +
		"Liquid value expression, what may appear inside {{ }}: its array gives one " +
		"evaluation per element, in order, with the element bound to item and its position " +
		"to fan_out, and nil gives none. A declaration beneath a container with for_each has " +
		"none of its own.", "for_each: inputs.changes"},
	{"outputs[].escape", "Template and mirror forms. Optional. How the values a template " +
		"writes are escaped: none, the default, or html, which writes & < > \" and ' as " +
		"entities.", "escape: html"},
	{"outputs[].newline", "Template and mirror forms. Optional. The rendered file's line " +
		"endings: as-written, the default, or crlf, which makes every LF not preceded by CR " +
		"a CR LF before the file is validated.", "newline: crlf"},
	{"outputs[].validate", "Template form. Optional. The package path, outside .compiled/, " +
		"of a JSON Schema the file's final bytes are validated against, parsed by its kind, " +
		"which must be json or yaml. A file that fails refuses the render, and nothing is " +
		"written.", "validate: schemas/feed.schema.json"},
	{"outputs[].kind", "Template form. Optional. The output kind of every file it renders, " +
		"overriding the one its path's extension gives: html, xml, markdown, json, yaml, csv, " +
		"css, hash, slash, sql or text. The kind chooses the comment syntax provenance " +
		"writes; it is not escaping.", "kind: yaml"},
	{"outputs[].executable", "Template and asset forms, never a container entry. " +
		"Optional. true writes the file with mode 0755 instead of 0644, less the umask. " +
		"Default false.", "executable: true"},

	{"outputs[].container.format", "Optional. The container's format: zip is the only one.",
		"format: zip"},
	{"outputs[].container.profile", "Optional. The packaging rules of a format the " +
		"container follows: ooxml, which needs an entry [Content_Types].xml; odf, which " +
		"writes the mimetype entry first and needs META-INF/manifest.xml; or epub, which " +
		"writes the mimetype entry first and needs META-INF/container.xml. Without one, the " +
		"container is written exactly as declared.", "profile: epub"},
	{"outputs[].container.mimetype", "Required with the odf profile, and allowed only with " +
		"it. The media type, type/subtype, written as the mimetype entry.",
		"mimetype: application/vnd.oasis.opendocument.text"},
	{"outputs[].container.entries", "Required. The container's entries, at least one, each " +
		"a declaration with the fields under outputs[].container.entries[].",
		"entries: [{template: notes.md.liquid, path: README.md}]"},

	{"outputs[].container.entries[].first", "Optional. true places the entry's files " +
		"before every other entry the profile does not place, in declaration order. Default " +
		"false.", "first: true"},
	{"outputs[].container.entries[].stored", "Optional. true stores the entry's files " +
		"rather than compressing them. Default false.", "stored: true"},
}

// manifestGroups is the heading and the opening text of each object's
// fields, in the order they are printed, by the prefix of their paths.
var manifestGroups = []struct{ prefix, heading, intro string }{
	{"", "The manifest's top level:", ""},
	{"authors[].", "Each author, authors[]:", ""},
	{"inputs[].", "Each input, inputs[] (spec chapter 5):", ""},
	{"outputs[].", "Each output declaration, outputs[] (spec chapters 9 and 10):",
		"A declaration takes exactly one form, named by the one source key it carries: " +
			"template, asset, mirror or container. Each field below names the forms that " +
			"take it, and a form refuses any other field."},
	{"outputs[].container.", "A zip container's shape, outputs[].container (spec chapter 10):", ""},
	{"outputs[].container.entries[].", "Each container entry, outputs[].container.entries[]:",
		"An entry is a declaration of one of the four forms and takes the fields outputs[] " +
			"gives its form, except that id is optional, executable is refused, since a zip " +
			"records no file mode, and path names the entry inside the container. Entries are " +
			"written into the container only; a part that must also be a file of its own is " +
			"declared again at the top level. Two fields are an entry's alone:"},
}

// manifestGroup is the index in manifestGroups of the object that holds
// the field at path: the longest prefix it has.
func manifestGroup(path string) int {
	best := 0
	for i, g := range manifestGroups {
		if strings.HasPrefix(path, g.prefix) && len(g.prefix) > len(manifestGroups[best].prefix) {
			best = i
		}
	}
	return best
}

// manifestTopic is metais help manifest (ADR-008 A6): the examples
// first, then every field, each with an example.
func manifestTopic(h *helpText, _ []*Command) {
	example := func(text string) {
		for _, l := range strings.Split(strings.TrimSuffix(text, "\n"), "\n") {
			if l == "" {
				h.blank()
				continue
			}
			h.line("  " + l)
		}
	}
	h.line("Examples:")
	h.blank()
	h.para("A complete metais.yaml: a package with two inputs, one of which may be "+
		"supplied as CSV, rendered to Markdown and to an HTML page that escapes its values.", "")
	h.blank()
	example(topicManifestHead + topicManifestOutputs)
	h.blank()
	h.para("The same package's outputs could instead write one page per change with "+
		"for_each, and a zip holding a template, an asset and a mirror:", "")
	h.blank()
	example(topicManifestFanOut)
	group := -1
	for _, f := range manifestFields {
		if g := manifestGroup(f.Path); g != group {
			group = g
			h.heading(manifestGroups[g].heading)
			if manifestGroups[g].intro != "" {
				h.para(manifestGroups[g].intro, "")
			}
			h.blank()
		}
		h.line("  " + f.Path)
		h.para(f.Text, "      ")
		h.line("      Example: " + f.Example)
	}
	h.blank()
	h.para("metais check [<dir>] checks a package's manifest against these rules, with "+
		"every other step, and reports every finding at once. metais init writes a "+
		"package to start from. metais help templates covers the template language a "+
		"template, path, when and for_each are written in. The specification's chapter 3 "+
		"(the manifest), chapter 5 (inputs), chapter 9 (outputs) and chapter 10 (containers "+
		"and transforms) state each rule, and schemas/1.0/manifest.schema.json holds the "+
		"manifest's schema.", "")
}
