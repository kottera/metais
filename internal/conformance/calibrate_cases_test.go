// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// calCases are the calibration suite's worst cases: one for each work and
// output limit of ADR-009 D3, then Amendment A1's joint worst case in a
// schema file and in the combined schema. Each is either refused by its
// limit, having done the work the limit allows, or accepted exactly at its
// limit where that does more work than the refusal would, and none is
// refused from metadata. docs/dev/calibration.md describes each.
var calCases = []calCase{
	{
		// The cheapest node over and over, so the most nodes run: 32 empty
		// if tags an iteration charge 512 units, so render-work runs out
		// at about 0.84 of loop-iterations.
		name: "render-work", limit: "render-work",
		lowered: calLimits{"render-work": 4_294_967, "loop-iterations": 10_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			body := strings.Repeat("{% if i %}{% endif %}", 32)
			return renderTemplate(dir, fmt.Sprintf("{%% for i in (1..%d) %%}{%% for j in (1..1000) %%}%s{%% endfor %%}{%% endfor %%}",
				lim["loop-iterations"]/1000, body))
		},
		reached: refusedBy("render-work"),
	},
	{
		// Every iteration the limit allows, nested so no single range is
		// refused before its first iteration until the last.
		name: "loop-iterations", limit: "loop-iterations",
		lowered: calLimits{"loop-iterations": 10_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, fmt.Sprintf("{%% for i in (1..%d) %%}{%% for j in (1..1000) %%}{%% endfor %%}{%% endfor %%}",
				lim["loop-iterations"]/1000+1))
		},
		reached: refusedBy("loop-iterations"),
	},
	{
		// A string doubled by append until it is refused.
		name: "value-size", limit: "value-size",
		lowered: calLimits{"value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, doubled("s", "x", lim["value-size"]+1)+"{{ s.size }}")
		},
		reached: refusedBy("value-size"),
	},
	{
		// A string of "x," doubled until split makes more elements than
		// the limit allows.
		name: "array-size", limit: "array-size",
		lowered: calLimits{"array-size": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			return renderTemplate(dir, doubled("s", "x,", 2*(lim["array-size"]+1))+`{% assign a = s | split: "," %}{{ a.size }}`)
		},
		reached: refusedBy("array-size"),
	},
	{
		// A string at value-size, copied by assign until the bytes assigned
		// pass the limit.
		name: "assign-bytes", limit: "assign-bytes",
		lowered: calLimits{"assign-bytes": 512 << 10, "value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := lim["assign-bytes"]/lim["value-size"] + 1
			return renderTemplate(dir, doubled("s", "x", lim["value-size"])+
				fmt.Sprintf(`{%% for i in (1..%d) %%}{%% assign t = s | append: "" %%}{%% endfor %%}{{ t.size }}`, n))
		},
		reached: refusedBy("assign-bytes"),
	},
	{
		// Exactly the units the limit allows (MTS-SEC-046). The combined
		// schema's root costs 2; its properties/report and the file's root
		// each apply to the array, 1 and 1 for each of its n items; the
		// first item's prefixItems, an allOf of rest, costs 1 + rest; and
		// every other item evaluates items and an allOf of nine, 10: in all
		// 12n + rest - 5. The validation is accepted, so it does all the
		// work the limit allows: one unit more is refused before anything
		// is evaluated.
		name: "schema-work", limit: "schema-work",
		lowered: calLimits{"schema-work": 100_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			rest := (lim["schema-work"] + 5) % 12
			items := (lim["schema-work"] + 5 - rest) / 12
			integers := func(n int64) string {
				return strings.TrimSuffix(strings.Repeat(`{"type":"integer"},`, int(n)), ",")
			}
			first := `{}`
			if rest > 0 {
				first = `{"allOf":[` + integers(rest) + `]}`
			}
			schema := `{"type":"array","prefixItems":[` + first + `],"items":{"allOf":[` + integers(9) + `]}}`
			return validateInput(dir, schema, "["+strings.TrimSuffix(strings.Repeat("0,", int(items)), ",")+"]")
		},
		reached: succeeded,
		beyond:  refusedBy("schema-work"),
	},
	{
		// grep's pattern at about 4,000 instructions, under regex-program,
		// over lines of a thousand bytes it never matches, until the
		// charge of D10 item 7 runs out.
		name: "regex-work", limit: "regex-work",
		lowered: calLimits{"regex-work": 8_388_608},
		build: func(dir string, lim calLimits) ([]string, error) {
			lines := lim["regex-work"]/1_000_000 + 1
			line := strings.Repeat("a", 999) + "\n"
			if err := writePackage(dir, "references/r.txt", strings.Repeat(line, int(lines))); err != nil {
				return nil, err
			}
			return []string{"grep", "--json", "pkg", strings.Repeat("a?", 2_000) + "b"}, nil
		},
		reached: func(r *calResult, lim calLimits) error { return r.refusedBy("regex-work", lim["regex-work"], "") },
	},
	{
		// Exactly the outputs the limit allows, each rendered and written.
		name: "outputs", limit: "outputs",
		lowered: calLimits{"outputs": 100},
		build: func(dir string, lim calLimits) ([]string, error) {
			return render(dir, fmt.Sprintf("  - id: each\n    template: a.txt.liquid\n    for_each: (1..%d)\n    path: \"{{ item }}.txt\"\n", lim["outputs"]),
				map[string]string{"a.txt.liquid": "{{ item }}"})
		},
		reached: func(r *calResult, lim calLimits) error {
			if err := r.succeeded(); err != nil {
				return err
			}
			if r.files != lim["outputs"] {
				return fmt.Errorf("%d files were written, not %d", r.files, lim["outputs"])
			}
			return nil
		},
		beyond: refusedBy("outputs"),
	},
	{
		// Containers nested as deep as the limit allows.
		name: "container-depth", limit: "container-depth",
		lowered: calLimits{"container-depth": 2},
		build: func(dir string, lim calLimits) ([]string, error) {
			return render(dir, nested(int(lim["container-depth"])), map[string]string{"a.txt.liquid": "a"})
		},
		reached: succeeded,
		beyond:  refusedBy("container-depth"),
	},
	{
		// One output of exactly the limit's bytes, rendered and written.
		name: "output-size", limit: "output-size",
		lowered: calLimits{"output-size": 256 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			return render(dir, "  - id: a\n    template: a.txt.liquid\n", map[string]string{"a.txt.liquid": kibLines(lim["output-size"])})
		},
		reached: succeeded,
		beyond:  refusedBy("output-size"),
	},
	{
		// Outputs of output-size each, exactly the limit's bytes together,
		// rendered and written.
		name: "output-total", limit: "output-total",
		lowered: calLimits{"output-total": 1 << 20, "output-size": 256 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := lim["output-total"] / lim["output-size"]
			if n*lim["output-size"] != lim["output-total"] {
				return nil, fmt.Errorf("output-total %d is not a multiple of output-size %d", lim["output-total"], lim["output-size"])
			}
			var outs strings.Builder
			for i := range n {
				fmt.Fprintf(&outs, "  - id: o%d\n    template: a.txt.liquid\n    path: o%d.txt\n", i, i)
			}
			return render(dir, outs.String(), map[string]string{"a.txt.liquid": kibLines(lim["output-size"])})
		},
		reached: succeeded,
		beyond:  refusedBy("output-total"),
	},
	{
		// One more finding than the limit lists: an item of an array that
		// fails its schema for each.
		name: "findings", limit: "findings",
		lowered: calLimits{"findings": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			items := lim["findings"] + 1
			return validateInput(dir, `{"type":"array","items":{"type":"string"}}`, "["+strings.TrimSuffix(strings.Repeat("0,", int(items)), ",")+"]")
		},
		reached: func(r *calResult, lim calLimits) error {
			if r.OK {
				return fmt.Errorf("the command succeeded")
			}
			if r.Summary.Listed != lim["findings"] || r.Summary.Omitted < 1 {
				return fmt.Errorf("%d findings were listed and %d omitted, not %d listed and some omitted", r.Summary.Listed, r.Summary.Omitted, lim["findings"])
			}
			return nil
		},
	},
	{
		// One choice whose alternatives list more findings than the limit
		// (MTS-FND-020): branches of 16 const failures each, every branch
		// rejected, so the choice lists itself and 16 per branch, and the
		// report cuts it short at the limit. The branches share one
		// subschema by $ref, so the tree that fails is the least that
		// lists this many.
		name: "findings-nested", limit: "findings",
		lowered: calLimits{"findings": 1_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			branches := lim["findings"]/16 + 1
			consts := make([]string, 16)
			for i := range consts {
				consts[i] = fmt.Sprintf(`{"const":%d}`, i)
			}
			schema := `{"anyOf":[` + strings.TrimSuffix(strings.Repeat(`{"$ref":"#/$defs/s"},`, int(branches)), ",") +
				`],"$defs":{"s":{"allOf":[` + strings.Join(consts, ",") + `]}}}`
			return validateInput(dir, schema, "16")
		},
		reached: func(r *calResult, lim calLimits) error {
			if r.OK || len(r.Findings) != 1 || r.Findings[0].Code != "input.any_of" {
				return fmt.Errorf("%d findings were listed, not one input.any_of", len(r.Findings))
			}
			listed, omitted := int64(1), int64(0)
			for _, a := range r.Findings[0].Alternatives {
				listed += int64(len(a.Findings))
				omitted += a.Omitted
			}
			if listed != lim["findings"] || omitted < 1 {
				return fmt.Errorf("the choice lists %d findings and omits %d, not %d listed and some omitted", listed, omitted, lim["findings"])
			}
			return nil
		},
	},
	{
		// One more matching line than the limit reports, each as long as a
		// reported line is shown.
		name: "grep-matches", limit: "grep-matches",
		lowered: calLimits{"grep-matches": 100},
		build: func(dir string, lim calLimits) ([]string, error) {
			if err := writePackage(dir, "references/r.txt", strings.Repeat(matchLine, int(lim["grep-matches"]+1))); err != nil {
				return nil, err
			}
			return []string{"grep", "--json", "pkg", "m", "--max-results", strconv.FormatInt(lim["grep-matches"], 10)}, nil
		},
		reached: func(r *calResult, lim calLimits) error {
			if err := r.succeeded(); err != nil {
				return err
			}
			if !r.truncated || r.matches != lim["grep-matches"] {
				return fmt.Errorf("%d matches were reported, truncated %v, not %d truncated", r.matches, r.truncated, lim["grep-matches"])
			}
			return nil
		},
	},
	{
		// The most matches grep reports, each with the most context: every
		// line matches, so each match repeats its neighbours as context.
		name: "grep-context", limit: "grep-context",
		lowered: calLimits{"grep-context": 5, "grep-matches": 100},
		build: func(dir string, lim calLimits) ([]string, error) {
			if err := writePackage(dir, "references/r.txt", strings.Repeat(matchLine, int(lim["grep-matches"]+1))); err != nil {
				return nil, err
			}
			return []string{"grep", "--json", "pkg", "m", "--max-results", strconv.FormatInt(lim["grep-matches"], 10),
				"--context", strconv.FormatInt(lim["grep-context"], 10)}, nil
		},
		reached: func(r *calResult, lim calLimits) error {
			if err := r.succeeded(); err != nil {
				return err
			}
			if int64(r.context) != lim["grep-context"] || r.matches != lim["grep-matches"] {
				return fmt.Errorf("%d matches were reported with at most %d lines of context, not %d with %d", r.matches, r.context, lim["grep-matches"], lim["grep-context"])
			}
			return nil
		},
		beyond: func(r *calResult, lim calLimits) error {
			for _, f := range r.Findings {
				if f.Code == "usage.bad_value" && f.Params["flag"] == "--context" {
					return nil
				}
			}
			return fmt.Errorf("--context %d was not refused", lim["grep-context"]+1)
		},
	},
	{
		// A1 item 5 in a schema file: the files hold exactly
		// schema-subschemas, every location but the two above the leaves
		// exactly schema-location-length. The files pass and compile, and
		// the combined schema, which embeds them, is then refused by its
		// count.
		name: "schema-joint-file", limit: "schema-subschemas",
		lowered: calLimits{"schema-subschemas": 200, "schema-location-length": 128},
		build: func(dir string, lim calLimits) ([]string, error) {
			return jointSchema(dir, lim, lim["schema-subschemas"])
		},
		reached: func(r *calResult, lim calLimits) error {
			return r.refusedBy("schema-subschemas", lim["schema-subschemas"], ".compiled/schema.json")
		},
		beyond: func(r *calResult, lim calLimits) error {
			return r.refusedBy("schema-subschemas", lim["schema-subschemas"], "inputs/report.schema.json")
		},
	},
	{
		// A1 item 5 in the combined schema: the combined schema holds
		// exactly schema-subschemas, and every location in the file it
		// embeds but the two above the leaves is exactly
		// schema-location-length. Everything passes and compiles.
		name: "schema-joint-combined", limit: "schema-subschemas",
		lowered: calLimits{"schema-subschemas": 200, "schema-location-length": 128},
		build: func(dir string, lim calLimits) ([]string, error) {
			return jointSchema(dir, lim, lim["schema-subschemas"]-combinedOverhead)
		},
		reached: succeeded,
		beyond: func(r *calResult, lim calLimits) error {
			return r.refusedBy("schema-subschemas", lim["schema-subschemas"], ".compiled/schema.json")
		},
	},
}

// combinedOverhead is how many more subschemas the combined schema holds
// than the files it embeds, for a package of one input: the subschemas it
// builds around the file (sec/schema-subschemas's 6 and 9).
const combinedOverhead = 3

// matchLine is a line grep's pattern m matches, 1,024 bytes as shown.
var matchLine = "m" + strings.Repeat("x", 1_022) + "\n"

// refusedBy is a reached that wants a refusal by limit at its bound.
func refusedBy(limit string) func(*calResult, calLimits) error {
	return func(r *calResult, lim calLimits) error { return r.refusedBy(limit, lim[limit], "") }
}

// succeeded is a reached that wants the command to succeed.
func succeeded(r *calResult, _ calLimits) error { return r.succeeded() }

// doubled is Liquid that assigns to name the string seed doubled until it
// is at least n bytes long.
func doubled(name, seed string, n int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "{%% assign %s = %q %%}", name, seed)
	for size := int64(len(seed)); size < n; size *= 2 {
		fmt.Fprintf(&b, "{%% assign %s = %s | append: %s %%}", name, name, name)
	}
	return b.String()
}

// kibLines is a template whose output is n bytes, n a multiple of 1,024,
// as lines of 1,024 bytes written by a loop, assigning nothing.
func kibLines(n int64) string {
	return fmt.Sprintf("{%% for i in (1..%d) %%}%s\n{%% endfor %%}", n/1024, strings.Repeat("x", 1023))
}

// nested is the outputs of a manifest with one container nested depth
// deep, whose innermost entry is a.txt.liquid.
func nested(depth int) string {
	var b strings.Builder
	b.WriteString("  - id: outer\n    path: outer.zip\n    container:\n      entries:\n")
	indent := "        "
	for level := 2; level <= depth; level++ {
		fmt.Fprintf(&b, "%s- path: level%d.zip\n%s  stored: true\n%s  container:\n%s    entries:\n", indent, level, indent, indent, indent)
		indent += "      "
	}
	fmt.Fprintf(&b, "%s- template: a.txt.liquid\n%s  stored: true\n", indent, indent)
	return b.String()
}

// manifestHead is a manifest up to its inputs.
const manifestHead = "metais: \"1.0\"\nname: example\nversion: 1.0.0\ntitle: Example\ndescription: An example package.\nguide: Supply the inputs and render.\n"

// writePackage writes the package pkg beneath dir: a manifest of one
// output, and the file at path with content.
func writePackage(dir, path, content string) error {
	files := map[string]string{
		"metais.yaml":            manifestHead + "inputs: []\noutputs:\n  - id: a\n    template: a.txt.liquid\n",
		"templates/a.txt.liquid": "a",
		path:                     content,
	}
	return writeFiles(filepath.Join(dir, "pkg"), files)
}

// writeFiles writes files beneath dir.
func writeFiles(dir string, files map[string]string) error {
	for p, content := range files {
		p = filepath.Join(dir, filepath.FromSlash(p))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// render writes a package of outputs, the manifest's outputs list, and of
// templates, and returns the command that renders it into out.
func render(dir, outputs string, templates map[string]string) ([]string, error) {
	files := map[string]string{"metais.yaml": manifestHead + "inputs: []\noutputs:\n" + outputs}
	for name, t := range templates {
		files["templates/"+name] = t
	}
	if err := writeFiles(filepath.Join(dir, "pkg"), files); err != nil {
		return nil, err
	}
	return []string{"render", "--json", "pkg", "--out", "out"}, nil
}

// renderTemplate is render of one output, a.txt from template.
func renderTemplate(dir, template string) ([]string, error) {
	return render(dir, "  - id: a\n    template: a.txt.liquid\n", map[string]string{"a.txt.liquid": template})
}

// validateInput writes a package of one input, report, of schema, and a
// data directory supplying it as input, and returns the command that
// validates it.
func validateInput(dir, schema, input string) ([]string, error) {
	files := map[string]string{
		"pkg/metais.yaml": manifestHead + "inputs:\n  - name: report\n    title: Report\n    schema: inputs/report.schema.json\n" +
			"outputs:\n  - id: a\n    template: a.txt.liquid\n",
		"pkg/templates/a.txt.liquid":    "a",
		"pkg/inputs/report.schema.json": schema,
		"data/report.json":              input,
	}
	if err := writeFiles(dir, files); err != nil {
		return nil, err
	}
	return []string{"validate", "--json", "pkg", "--data", "data"}, nil
}

// jointSchema writes a package of one input whose schema file holds n
// subschemas: its root, one property beneath it, and n-2 empty leaves
// beneath that, each leaf's location exactly schema-location-length long.
// The property's name pads the locations, so the file stays small.
func jointSchema(dir string, lim calLimits, n int64) ([]string, error) {
	const resource = "metais:///inputs/report.schema.json#"
	leaves := n - 2
	width := len(strconv.FormatInt(leaves, 10))
	pad := int(lim["schema-location-length"]) - len(resource) - len("/properties/")*2 - (width + 1)
	if pad < 1 || leaves < 1 {
		return nil, fmt.Errorf("no file of %d subschemas has locations of %d bytes", n, lim["schema-location-length"])
	}
	var b strings.Builder
	fmt.Fprintf(&b, `{"type":"object","properties":{"%s":{"type":"object","properties":{`, strings.Repeat("p", pad))
	for i := range leaves {
		if i > 0 {
			b.WriteByte(',')
		}
		fmt.Fprintf(&b, `"n%0*d":{}`, width, i)
	}
	b.WriteString("}}}}")
	files := map[string]string{
		"pkg/metais.yaml": manifestHead + "inputs:\n  - name: report\n    title: Report\n    schema: inputs/report.schema.json\n" +
			"outputs:\n  - id: a\n    template: a.txt.liquid\n",
		"pkg/templates/a.txt.liquid":    "a",
		"pkg/inputs/report.schema.json": b.String(),
	}
	if err := writeFiles(dir, files); err != nil {
		return nil, err
	}
	return []string{"inspect", "--json", "pkg"}, nil
}
