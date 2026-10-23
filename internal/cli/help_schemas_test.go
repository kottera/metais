// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"os"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/schema"
	"metais.org/metais/internal/testenv"
)

// schemaPackage writes a package whose one input, name, has the schema
// src, and whose data directory, when data is not empty, supplies it.
func schemaPackage(t *testing.T, name, src, data string) string {
	t.Helper()
	files := map[string]string{
		"metais.yaml": `metais: "1.0"
name: org.example.schemas
version: 1.0.0
title: Schemas
description: A package of one input.
guide: Supply the input.
inputs:
  - name: ` + name + `
    title: The input
    schema: inputs/` + name + `.schema.json
outputs:
  - id: report
    template: report.md.liquid
`,
		"inputs/" + name + ".schema.json": src,
		"templates/report.md.liquid":      "report\n",
	}
	if data != "" {
		files["data/"+name+".json"] = data
	}
	return writeTree(t, files)
}

// TestSchemasTopicKeywords holds help schemas' keyword lists to the
// compiler's own (ADR-003 D3 items 3 to 5, A2): every keyword a schema may
// use is listed once, and no other.
func TestSchemasTopicKeywords(t *testing.T) {
	var listed []string
	for _, g := range slices.Concat(assertedKeywords, annotationKeywords) {
		listed = append(listed, g.Names...)
	}
	sorted := slices.Sorted(slices.Values(listed))
	if !slices.Equal(sorted, slices.Compact(slices.Clone(sorted))) {
		t.Errorf("help schemas lists a keyword twice: %v", sorted)
	}
	if want := schema.Keywords(); !slices.Equal(slices.Compact(sorted), want) {
		t.Errorf("help schemas lists the keywords\n%v\nthe compiler allows\n%v", sorted, want)
	}
	text := runTool(t, "help", "schemas").stdout
	for _, k := range listed {
		if !strings.Contains(text, " "+k+" ") && !strings.Contains(text, " "+k+"\n") {
			t.Errorf("help schemas does not print the keyword %s", k)
		}
	}
}

// TestSchemasTopicFormats holds help schemas' formats to ADR-003 D4: one
// row for each format the compiler asserts and no other, and each row's
// example a string of its format, which validates. The pin: a string
// that is not of its format fails, so the formats are asserted.
func TestSchemasTopicFormats(t *testing.T) {
	testenv.Home(t)
	var names []string
	props := map[string]any{}
	values := map[string]string{}
	for _, f := range formatDocs {
		names = append(names, f.Name)
		props[f.Name] = map[string]string{"type": "string", "format": f.Name}
		values[f.Name] = f.Example
	}
	if want := schema.Formats(); !slices.Equal(slices.Sorted(slices.Values(names)), want) {
		t.Errorf("help schemas gives the formats\n%v\nthe compiler asserts\n%v", names, want)
	}
	src, _ := json.Marshal(map[string]any{"type": "object", "properties": props})
	for _, c := range []struct {
		name string
		bad  string // the value of date, when not the example's
		want int
	}{{"examples", "", 0}, {"pin", "2026-13-01", 1}} {
		v := map[string]string{}
		for k, s := range values {
			v[k] = s
		}
		if c.bad != "" {
			v["date"] = c.bad
		}
		data, _ := json.Marshal(v)
		dir := schemaPackage(t, "formats", string(src), string(data))
		r := runTool(t, "validate", dir, "--data", dir+"/data", "--json")
		if r.status != c.want {
			t.Errorf("%s: validate exited %d, want %d: %v", c.name, r.status, c.want, r.codes())
		}
	}
}

// TestSchemasTopicExample is W2-6.7: the example help schemas gives is
// chapter 13's (§13.1, step 5, "Rules across the rows of a table"), the
// same JSON value laid out to the help's width; it compiles under the
// dialect, and metais check finds nothing in it; and schema --fields
// lists its two rules directly beneath the array's line, beginning as the
// topic prints them. The pin: the same schema with uniqueKeys misspelt
// is refused, so the compile is a check of the dialect.
func TestSchemasTopicExample(t *testing.T) {
	testenv.Home(t)
	chapter, err := os.ReadFile("../../spec/13-authoring.md")
	if err != nil {
		t.Fatal(err)
	}
	_, section, ok := strings.Cut(string(chapter), "\n#### Rules across the rows of a table\n")
	section, _, _ = strings.Cut(section, "\n#### ")
	_, block, ok2 := strings.Cut(section, "\n```json\n")
	block, _, ok3 := strings.Cut(block, "\n```\n")
	if !ok || !ok2 || !ok3 {
		t.Fatal("chapter 13 has no JSON example under Rules across the rows of a table")
	}
	var fromSpec, fromTopic any
	if err := json.Unmarshal([]byte(block), &fromSpec); err != nil {
		t.Fatalf("chapter 13's example: %v", err)
	}
	if err := json.Unmarshal([]byte(topicSchemasExample), &fromTopic); err != nil {
		t.Fatalf("the topic's example: %v", err)
	}
	if !reflect.DeepEqual(fromSpec, fromTopic) {
		t.Errorf("the topic's example is not chapter 13's:\n%s\n%s", topicSchemasExample, block)
	}

	text := runTool(t, "help", "schemas").stdout
	indented := "  " + strings.ReplaceAll(strings.TrimSuffix(topicSchemasExample, "\n"), "\n", "\n  ") + "\n"
	if !strings.Contains(text, indented) {
		t.Errorf("help schemas does not print the example:\n%s", text)
	}

	dir := schemaPackage(t, "criteria", block, "")
	r := runTool(t, "check", dir, "--json")
	for _, c := range r.codes() {
		if strings.HasPrefix(c, "schema.") {
			t.Errorf("check reports %s of chapter 13's example", c)
		}
	}
	r = runTool(t, "schema", dir, "criteria", "--fields")
	if r.status != 0 {
		t.Fatalf("schema --fields exited %d: %s", r.status, r.stderr)
	}
	lines := strings.Split(strings.TrimSuffix(r.stdout, "\n"), "\n")
	i := slices.IndexFunc(lines, func(l string) bool { return strings.HasPrefix(l, "/criteria\tarray\t") })
	if i < 0 || i+2 >= len(lines) {
		t.Fatalf("schema --fields lists no array line and two after it:\n%s", r.stdout)
	}
	var quoted []string
	for _, v := range []string{"1.1.1", "1.2.1", "1.2.2", "1.2.3", "1.2.4", "1.2.5"} {
		quoted = append(quoted, strconv.Quote(v))
	}
	want := []string{
		"/criteria\trule\trows must include, at `/criterion`, each of: " + strings.Join(quoted, ", "),
		"/criteria\trule\tno two rows share `/criterion`",
	}
	if got := lines[i+1 : i+3]; !slices.Equal(got, want) {
		t.Errorf("schema --fields lists beneath the array\n%q\nwant\n%q", got, want)
	}
	for j, l := range topicSchemasRules {
		if !strings.HasPrefix(strings.TrimPrefix(want[j], "/criteria\trule\t"), strings.TrimSuffix(l, "…")) {
			t.Errorf("help schemas shows the rule %q, which schema --fields lists as %q", l, want[j])
		}
	}

	misspelt := schemaPackage(t, "criteria", strings.Replace(block, `"uniqueKeys"`, `"uniqueKey"`, 1), "")
	if r := runTool(t, "schema", misspelt, "criteria", "--fields", "--json"); r.status == 0 || !slices.Contains(r.codes(), "schema.keyword_unknown") {
		t.Errorf("a misspelt uniqueKeys is not refused: exit %d, %v", r.status, r.codes())
	}
}
