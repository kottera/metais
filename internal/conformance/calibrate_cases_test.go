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
// refused from metadata. Amendment A6's cases follow them
// (calibrate_bounds_test.go). docs/dev/calibration.md describes each.
var calCases = append([]calCase{
	{
		// The cheapest node over and over, so the most nodes run: 32 empty
		// if tags an iteration charge 512 units, so render-work runs out
		// at about 0.05 of loop-iterations. The costliest units are
		// renderWorkUnitCases (calibrate_units_test.go).
		name: "render-work", limit: "render-work",
		lowered: calLimits{"render-work": 268_435, "loop-iterations": 10_000},
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
		// A string doubled until it is refused.
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
		// A string at value-size, copied by capture until the bytes held
		// pass the limit.
		name: "assign-bytes", limit: "assign-bytes",
		lowered: calLimits{"assign-bytes": 512 << 10, "value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := lim["assign-bytes"]/lim["value-size"] + 1
			return renderTemplate(dir, doubled("s", "x", lim["value-size"])+
				fmt.Sprintf(`{%% for i in (1..%d) %%}{%% capture t %%}{{ s }}{%% endcapture %%}{%% endfor %%}{{ t.size }}`, n))
		},
		reached: refusedBy("assign-bytes"),
	},
	{
		// Captures nested one in another, each writing a string at
		// value-size before opening the next, so their buffers fill at
		// once until the bytes held pass the limit. A capture's bytes are
		// charged as they are written, so this is refused before the
		// buffers pass assign-bytes, not when the outermost ends.
		name: "assign-bytes-capture", limit: "assign-bytes",
		lowered: calLimits{"assign-bytes": 512 << 10, "value-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			n := int(lim["assign-bytes"]/lim["value-size"] + 1)
			return renderTemplate(dir, doubled("s", "x", lim["value-size"])+
				strings.Repeat("{% capture c %}{{ s }}", n)+strings.Repeat("{% endcapture %}", n)+"{{ c.size }}")
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
		// A failing tree of exactly the units the limit allows: two-branch
		// anyOfs nested by $ref over a type every level fails, so each
		// choice's alternatives nest the next and the findings listed
		// reach the findings limit. Every unit evaluated fails, so the
		// library, the adapter and the mapping each hold a failure for it.
		// schemaWorkAnyOf says how the units are made up. The validation
		// is not refused, so it does all the work the limit allows: one
		// unit more is refused before anything is evaluated.
		name: "schema-work-anyof", limit: "schema-work",
		lowered: calLimits{"schema-work": 100_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			schema, err := schemaWorkAnyOf(lim["schema-work"])
			if err != nil {
				return nil, err
			}
			return validateInput(dir, schema, "5")
		},
		reached: func(r *calResult, lim calLimits) error {
			if r.OK {
				return fmt.Errorf("the command succeeded")
			}
			choice := false
			for _, f := range r.Findings {
				if f.Params["limit"] != nil {
					return fmt.Errorf("the validation was refused by %v", f.Params["limit"])
				}
				choice = choice || f.Code == "input.any_of"
			}
			if !choice {
				return fmt.Errorf("no input.any_of was reported")
			}
			return nil
		},
		beyond: refusedBy("schema-work"),
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
	{
		// One YAML document of exactly yaml-size bytes, in its costliest
		// shape: a flow sequence of zeros, a value for every two bytes, as
		// many as input-values allows (Amendment A6 item 4). It is read,
		// and a byte less is refused before it is parsed (ADR-009
		// Amendment A4 item 2).
		name: "yaml-size", limit: "yaml-size",
		lowered: calLimits{"yaml-size": 64 << 10, "input-values": 30_000},
		build: func(dir string, lim calLimits) ([]string, error) {
			doc, _ := yamlZerosAtMost(lim["yaml-size"], lim["input-values"])
			return loadInputs(dir, []loadInput{{name: "y", form: "yaml", files: map[string]string{"y.yaml": doc}}})
		},
		reached: notRefusedLoading,
		beyond:  refusedBy("yaml-size"),
	},
	jointCase("json"),
	jointCase("csv"),
	jointCase("yaml"),
	renderJointCase("json"),
	renderJointCase("yaml"),
}, append(a6Cases, renderWorkUnitCases...)...)

// schemaWorkAnyOf is an input schema whose validation of 5 costs exactly
// units of schema-work, and fails at every unit: an allOf of $refs to
// levels of two-branch anyOfs, level j reaching level j-1 by $ref twice
// and level 0 a string type. As the input of validateInput, the schema
// costs 4 units, and each $ref to level j 4×2^j - 2 more; a last allOf of
// k string types costs k + 1. The deepest level comes first, so its
// choice nests the most.
func schemaWorkAnyOf(units int64) (string, error) {
	left := units - 4
	if left < 2 {
		return "", fmt.Errorf("no schema costs %d units this way", units)
	}
	cost := func(j int) int64 { return 4<<j - 2 }
	var levels []int
	for j := 40; j >= 0; j-- {
		for cost(j) <= left {
			levels = append(levels, j)
			left -= cost(j)
		}
	}
	k := int64(0)
	if left == 1 {
		// No level costs one unit: give back the shallowest level and
		// make its units and this one up with string types.
		last := levels[len(levels)-1]
		levels = levels[:len(levels)-1]
		k = cost(last)
	}
	parts := make([]string, 0, len(levels)+1)
	for _, j := range levels {
		parts = append(parts, fmt.Sprintf(`{"$ref":"#/$defs/l%d"}`, j))
	}
	if k > 0 {
		parts = append(parts, `{"allOf":[`+strings.TrimSuffix(strings.Repeat(`{"type":"string"},`, int(k)), ",")+`]}`)
	}
	defs := []string{`"l0":{"type":"string"}`}
	for j := 1; j <= levels[0]; j++ {
		defs = append(defs, fmt.Sprintf(`"l%d":{"anyOf":[{"$ref":"#/$defs/l%d"},{"$ref":"#/$defs/l%d"}]}`, j, j-1, j-1))
	}
	return `{"allOf":[` + strings.Join(parts, ",") + `],"$defs":{` + strings.Join(defs, ",") + `}}`, nil
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
// is at least n bytes long. It doubles by capture, whose output tags cost
// render-work's units whatever their length, so a string of value-size
// costs a few hundred units to build, and every case leaves render-work
// to the unit it measures.
func doubled(name, seed string, n int64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "{%% assign %s = %q %%}", name, seed)
	for size := int64(len(seed)); size < n; size *= 2 {
		fmt.Fprintf(&b, "{%% capture %s %%}{{ %s }}{{ %s }}{%% endcapture %%}", name, name, name)
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

// loadInput is one input of a loading case: its name, its form, whether
// it is a collection, and its files beneath the data directory.
type loadInput struct {
	name, form string
	collection bool
	files      map[string]string
}

// loadInputs writes a package of inputs, each optional and of any value,
// and a data directory holding their files, and returns the command that
// validates them.
func loadInputs(dir string, inputs []loadInput) ([]string, error) {
	var m strings.Builder
	m.WriteString(manifestHead + "inputs:\n")
	files := map[string]string{
		"pkg/templates/a.txt.liquid": "a",
		"pkg/inputs/any.schema.json": "{}",
	}
	for _, in := range inputs {
		fmt.Fprintf(&m, "  - name: %s\n    title: %s\n    schema: inputs/any.schema.json\n    required: false\n    forms: [%s]\n", in.name, strings.ToUpper(in.name), in.form)
		if in.collection {
			m.WriteString("    collection: true\n")
		}
		for name, content := range in.files {
			files["data/"+name] = content
		}
	}
	m.WriteString("outputs:\n  - id: a\n    template: a.txt.liquid\n")
	files["pkg/metais.yaml"] = m.String()
	if err := writeFiles(dir, files); err != nil {
		return nil, err
	}
	return []string{"validate", "--json", "pkg", "--data", "data"}, nil
}

// yamlZeros is a YAML document of exactly size bytes, size at least 3: a
// flow sequence of zeros, ended by a line feed when size is even, and the
// values it decodes to.
func yamlZeros(size int64) (string, int64) {
	k := (size - 1) / 2
	doc := "[" + strings.Repeat("0,", int(k-1)) + "0]"
	if size%2 == 0 {
		doc += "\n"
	}
	return doc, k + 1
}

// yamlZerosAtMost is yamlZeros of size bytes, holding at most values
// values: the array and values-1 zeros, padded with spaces to size.
func yamlZerosAtMost(size, values int64) (string, int64) {
	doc, v := yamlZeros(size)
	if v <= values {
		return doc, v
	}
	zeros := values - 1
	return "[" + strings.Repeat("0,", int(zeros-1)) + "0" + strings.Repeat(" ", int(size-2*zeros-1)) + "]", values
}

// notRefusedLoading is a reached that wants the inputs loaded whole: no
// data limit refused them. The validation that follows may still be
// refused by schema-work, which a loading case's values pass.
func notRefusedLoading(r *calResult, _ calLimits) error {
	for _, f := range r.Findings {
		switch f.Params["limit"] {
		case "input-values", "yaml-size", "inputs-total", "input-size", "csv-rows", "collection-items":
			return fmt.Errorf("the inputs were refused by %v at %s%s", f.Params["limit"], f.Location.File, f.Location.Pointer)
		}
	}
	return nil
}

// jointCase is ADR-009 Amendment A4 item 3's joint case for form: the
// form's costliest shape decoded to the values input-values leaves, the
// rest of inputs-total as long strings in plain text files of input-size
// at most, and a last YAML document at yaml-size, the values of all of
// them together exactly input-values. One value more is refused. Since
// Amendment A6 item 4 halved input-values, a document at yaml-size holds
// more zeros than input-values allows, so the last document holds at most
// half of input-values, padded to yaml-size.
func jointCase(form string) calCase {
	return calCase{
		name: "input-values-" + form, limit: "input-values",
		lowered: calLimits{"input-values": 30_000, "yaml-size": 64 << 10, "inputs-total": 1 << 20, "input-size": 256 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			last, lastValues := yamlZerosAtMost(lim["yaml-size"], lim["input-values"]/2)
			z := loadInput{name: "z", form: "yaml", files: map[string]string{"z.yaml": last}}
			// The form's shape, first sized for the values the strings
			// leave, then the strings sized for the bytes the shape leaves.
			shape := func(values int64) (loadInput, int64, error) {
				switch form {
				case "json":
					// [{},{},…]: an object for every three bytes.
					if values < 2 {
						return loadInput{}, 0, fmt.Errorf("%d values make no array of objects", values)
					}
					doc := "[" + strings.TrimSuffix(strings.Repeat("{},", int(values-1)), ",") + "]"
					return loadInput{name: "a", form: "json", files: map[string]string{"a.json": doc}}, int64(len(doc)), nil
				case "csv":
					// A header of two columns and rows of one field each,
					// the second unquoted and empty, so absent: an object
					// and a member for every three bytes. An odd count
					// takes a last row of both.
					rows := (values - 1) / 2
					doc := "a,b\n" + strings.Repeat("0,\n", int(rows))
					if (values-1)%2 == 1 {
						if rows == 0 {
							return loadInput{}, 0, fmt.Errorf("%d values make no table", values)
						}
						doc = "a,b\n" + strings.Repeat("0,\n", int(rows-1)) + "0,0\n"
					}
					return loadInput{name: "a", form: "csv", files: map[string]string{"a.csv": doc}}, int64(len(doc)), nil
				case "yaml":
					// A collection of documents at yaml-size, the last
					// smaller, each a flow sequence of zeros.
					in := loadInput{name: "a", form: "yaml", collection: true, files: map[string]string{}}
					left, size := values-1, int64(0)
					for i := 0; left > 0; i++ {
						docSize := lim["yaml-size"]
						if _, v := yamlZeros(docSize); v > left {
							docSize = 2*left - 1
						}
						if docSize < 3 {
							return loadInput{}, 0, fmt.Errorf("%d values make no last document", left)
						}
						doc, v := yamlZeros(docSize)
						in.files[fmt.Sprintf("a/%06d.yaml", i)] = doc
						left -= v
						size += int64(len(doc))
					}
					return in, size, nil
				}
				return loadInput{}, 0, fmt.Errorf("no shape for %s", form)
			}
			// The strings come first: plain text files, a value each.
			_, guess, err := shape(lim["input-values"] - lastValues)
			if err != nil {
				return nil, err
			}
			rest := lim["inputs-total"] - guess - lim["yaml-size"]
			texts := (rest + lim["input-size"] - 1) / lim["input-size"]
			a, size, err := shape(lim["input-values"] - lastValues - texts)
			if err != nil {
				return nil, err
			}
			rest = lim["inputs-total"] - size - lim["yaml-size"]
			ins := []loadInput{a}
			for i := range texts {
				n := min(rest, lim["input-size"])
				rest -= n
				ins = append(ins, loadInput{name: fmt.Sprintf("s%d", i), form: "text", files: map[string]string{fmt.Sprintf("s%d.txt", i): strings.Repeat("x", int(n))}})
			}
			return loadInputs(dir, append(ins, z))
		},
		reached: notRefusedLoading,
		beyond:  refusedBy("input-values"),
	}
}
