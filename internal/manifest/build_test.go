// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
)

// TestBuild: the model holds what the manifest writes, with ADR-003 D1's
// and ADR-005's defaults applied where it is silent.
func TestBuild(t *testing.T) {
	isolate(t)
	root, _ := parse(t, encode(t, base()))
	m, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	check := func(what string, got, want any) {
		t.Helper()
		if fmt.Sprint(got) != fmt.Sprint(want) {
			t.Errorf("%s = %v, want %v", what, got, want)
		}
	}
	check("metais", m.Metais, Spec{1, 0})
	check("name", m.Name, "org.example.vpat-acr")
	check("version", m.Version, "2.1.0-rc.1+20261018")
	check("title", m.Title, "Accessibility Conformance Report")
	check("description", m.Description, "An ACR in Markdown and DOCX.\n\tIndented.")
	check("guide", m.Guide, "# Guide\n\nGather the product first.")
	check("authors", m.Authors, []Author{{"Example Org", "acr@example.org", "https://example.org/acr"}, {"A. Person", "", ""}})
	check("license", m.License, "(Apache-2.0 OR MIT) AND GPL-2.0-or-later WITH Classpath-exception-2.0")
	check("homepage", m.Homepage, "https://example.org:8443/packages/acr?tab=readme")
	check("input_constraints", m.InputConstraints, "inputs/constraints.schema.yaml")

	if len(m.Inputs) != 2 {
		t.Fatalf("inputs %v", m.Inputs)
	}
	product, criteria := m.Inputs[0], m.Inputs[1]
	check("product", product, Input{
		Pointer: "/inputs/0", Name: "product", Title: "Product", Description: "The product the report covers.",
		Schema: "inputs/product.schema.json", Meaning: "inputs/product.md",
		Forms: []Form{FormJSON, FormYAML}, Required: true,
	})
	check("criteria", criteria, Input{
		Pointer: "/inputs/1", Name: "criteria", Title: "WCAG criteria", Schema: "inputs/criteria.schema.yaml",
		Forms: []Form{FormCSV, FormJSON}, Delimiter: "\t",
	})

	if len(m.Outputs) != 6 {
		t.Fatalf("outputs %v", m.Outputs)
	}
	report := m.Outputs[0]
	check("report", report, Output{Pointer: "/outputs/0", Form: OutputTemplate, ID: "report", Source: "report.md.liquid", Escape: "none", Newline: "as-written"})
	docx := m.Outputs[1]
	if docx.Form != OutputContainer || docx.Source != "" || docx.Path != "{{ inputs.product.slug }}-acr.docx" || docx.When != "inputs.product" || docx.Container == nil {
		t.Fatalf("acr_docx = %+v", docx)
	}
	c := docx.Container
	check("container", []string{c.Format, c.Profile, c.Mimetype}, []string{"zip", "ooxml", ""})
	if len(c.Entries) != 4 {
		t.Fatalf("entries %v", c.Entries)
	}
	check("mirror entry", c.Entries[0], Output{Pointer: "/outputs/1/container/entries/0", Form: OutputMirror, Source: "docx", Path: ".", Escape: "html", Newline: "crlf", First: true})
	check("asset entry", c.Entries[1], Output{Pointer: "/outputs/1/container/entries/1", Form: OutputAsset, ID: "logo", Source: "vendor-logo.png", Path: "word/media/image1.png", Escape: "none", Newline: "as-written", Stored: true})
	check("template entry", c.Entries[2], Output{Pointer: "/outputs/1/container/entries/2", Form: OutputTemplate, Source: "part.json.liquid", Escape: "none", Newline: "as-written", Kind: "json", Validate: "references/part.schema.json"})
	inner := c.Entries[3]
	if inner.Form != OutputContainer || inner.Path != "inner.odt" || inner.Container == nil ||
		inner.Container.Profile != "odf" || inner.Container.Mimetype != "application/vnd.oasis.opendocument.text" ||
		len(inner.Container.Entries) != 1 || inner.Container.Entries[0].Source != "content.xml" {
		t.Errorf("inner container = %+v", inner)
	}
	check("article", m.Outputs[2], Output{Pointer: "/outputs/2", Form: OutputTemplate, ID: "article", Source: "site/_article.html.liquid",
		Path: "articles/{{ item.slug }}.html", ForEach: `inputs.articles | where: "draft", false`, Escape: "html", Newline: "as-written", Kind: "html"})
	check("site", m.Outputs[3], Output{Pointer: "/outputs/3", Form: OutputMirror, ID: "site", Source: "site", Path: ".", Escape: "html", Newline: "as-written"})
	check("static", m.Outputs[4], Output{Pointer: "/outputs/4", Form: OutputAsset, ID: "static", Source: "static", Escape: "none", Newline: "as-written", Executable: true})
	if books := m.Outputs[5]; books.ForEach != "inputs.books" || books.Container.Profile != "epub" || books.Container.Mimetype != "" {
		t.Errorf("books = %+v", books)
	}
}

// TestBuildDefaults: the smallest manifest, with every optional member
// omitted, takes the specification's defaults (ADR-003 D1 item 2, ADR-005
// D2) and an empty inputs list.
func TestBuildDefaults(t *testing.T) {
	isolate(t)
	m := tree{
		"metais": "1.0", "name": "status-widget", "version": "1.0.0", "title": "Status widget",
		"description": "A widget.", "guide": "Supply nothing.", "inputs": []any{},
		"outputs": []any{tree{"id": "all", "mirror": "."}},
	}
	text := encode(t, m)
	if ok, why := schemaAccepts(t, text); !ok {
		t.Fatalf("the schema refuses the smallest manifest: %s", why)
	}
	root, _ := parse(t, text)
	got, err := Build(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Authors != nil || got.License != "" || got.Homepage != "" || got.InputConstraints != "" || len(got.Inputs) != 0 {
		t.Errorf("optional members %+v", got)
	}
	if o := got.Outputs[0]; o.Escape != "none" || o.Newline != "as-written" || o.Path != "" || o.Executable {
		t.Errorf("output defaults %+v", o)
	}

	set(t, m, "", "inputs", []any{tree{"name": "rows", "title": "Rows", "schema": "inputs/rows.schema.json", "forms": []any{"csv"}}})
	root, _ = parse(t, encode(t, m))
	if got, err = Build(root); err != nil {
		t.Fatal(err)
	}
	if in := got.Inputs[0]; in.Delimiter != "," || !in.Required || in.Collection || !slices.Equal(in.Forms, []Form{FormCSV}) {
		t.Errorf("csv input defaults %+v", in)
	}
}

// TestBuildReportsEveryViolation: Build names every rule a manifest
// breaks, not the first (ADR-006 D7: every finding is reported).
func TestBuildReportsEveryViolation(t *testing.T) {
	isolate(t)
	m := base()
	set(t, m, "", "title", "")
	set(t, m, "", "name", "Bad")
	set(t, m, "inputs/0", "name", "size")
	set(t, m, "outputs/0", "escape", "xml")
	root, _ := parse(t, encode(t, m))
	_, err := Build(root)
	var inv *InvalidError
	if !errors.As(err, &inv) {
		t.Fatalf("Build: %v", err)
	}
	want := []Violation{{"/name", ruleName}, {"/title", ruleLength}, {"/inputs/0/name", ruleReserved}, {"/outputs/0/escape", ruleEnum}}
	if !slices.Equal(inv.Violations, want) {
		t.Errorf("violations %v, want %v", inv.Violations, want)
	}
	if msg := err.Error(); !strings.Contains(msg, "/inputs/0/name: reserved_name") {
		t.Errorf("error %q does not name the violation", msg)
	}
}
