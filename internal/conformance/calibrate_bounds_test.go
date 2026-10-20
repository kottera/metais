// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"archive/zip"
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ADR-009 Amendment A6's calibration cases: the joint render case of item
// 4, with each of its two last outputs; item 5's YAML schema file; and a
// case for each path item 2 bounds: the digest file, example.yaml read as
// JSON, the title the open takes of a reference, and a manifest whose
// validation would pass schema-work.

// renderJointCase is item 4's joint render case, its last output of kind
// form: inputs of exactly input-values values, the most of them in A4 item
// 3's costliest shape, [{},{},…], nested in an array so that the input's
// schema, which applies to its root alone, is evaluated and the library
// validates its own copy of the whole value; the rest of inputs-total as
// plain text files of input-size at most, a value each; then outputs of
// output-size, exactly output-total together with the last output, which
// is validated after they are rendered and held: a JSON array of empty
// objects, exactly output-values values, or a YAML flow sequence of zeros
// of exactly yaml-size bytes, as many zeros as output-values allows. One
// value or byte less refuses the last output.
func renderJointCase(form string) calCase {
	limit := map[string]string{"json": "output-values", "yaml": "yaml-size"}[form]
	return calCase{
		name: "render-joint-" + form, limit: limit,
		lowered: calLimits{"input-values": 60_000, "inputs-total": 1 << 20, "input-size": 256 << 10,
			"output-total": 1 << 20, "output-size": 256 << 10, "output-values": 25_000, "yaml-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			// The last output, validated.
			var last, lastPath string
			var lastSize int64
			switch form {
			case "json":
				// The array and v-1 empty objects, the costliest value
				// for its count (A4 item 3).
				v := lim["output-values"]
				last = fmt.Sprintf("[{%% for i in (1..%d) %%}{},{%% endfor %%}{}]", v-2)
				lastPath, lastSize = "z.json", 3*v-2
			case "yaml":
				// A flow sequence of zeros, as many as output-values lets
				// it hold, padded to yaml-size.
				_, values := yamlZeros(lim["yaml-size"])
				k := min(values, lim["output-values"]) - 1
				pad := lim["yaml-size"] - (2*k + 1)
				last = fmt.Sprintf("[{%% for i in (1..%d) %%}0,{%% endfor %%}0%s]", k-1, strings.Repeat(" ", int(pad)))
				lastPath, lastSize = "z.yaml", lim["yaml-size"]
			}
			// The outputs before it: output-size each, and one of what is
			// left.
			var outs strings.Builder
			templates := map[string]string{"pkg/templates/full.txt.liquid": sizedText(lim["output-size"]), "pkg/templates/z.liquid": last,
				"pkg/references/s.json": `{"type":"array"}`}
			rest := lim["output-total"] - lastSize
			for i := 0; rest > 0; i++ {
				tmpl := "full.txt.liquid"
				if rest < lim["output-size"] {
					tmpl = "rest.txt.liquid"
					templates["pkg/templates/"+tmpl] = sizedText(rest)
				}
				fmt.Fprintf(&outs, "  - id: o%d\n    template: %s\n    path: o%d.txt\n", i, tmpl, i)
				rest -= min(rest, lim["output-size"])
			}
			fmt.Fprintf(&outs, "  - id: z\n    template: z.liquid\n    path: %s\n    validate: references/s.json\n", lastPath)

			// The inputs: the strings first, a value each, then the array
			// of objects of the values they leave: the array, the array it
			// holds, and an object for every three bytes.
			texts := (lim["inputs-total"] + lim["input-size"] - 1) / lim["input-size"]
			objects := lim["input-values"] - texts - 2
			doc := "[[" + strings.TrimSuffix(strings.Repeat("{},", int(objects)), ",") + "]]"
			left := lim["inputs-total"] - int64(len(doc))
			texts = (left + lim["input-size"] - 1) / lim["input-size"]
			objects = lim["input-values"] - texts - 2
			doc = "[[" + strings.TrimSuffix(strings.Repeat("{},", int(objects)), ",") + "]]"
			left = lim["inputs-total"] - int64(len(doc))
			var m strings.Builder
			m.WriteString(manifestHead + "inputs:\n  - name: a\n    title: A\n    schema: inputs/a.schema.json\n    required: false\n    forms: [json]\n")
			files := map[string]string{"pkg/inputs/a.schema.json": `{"type":"array"}`, "pkg/inputs/any.schema.json": "{}", "data/a.json": doc}
			for i := range texts {
				n := min(left, lim["input-size"])
				left -= n
				fmt.Fprintf(&m, "  - name: s%d\n    title: S%d\n    schema: inputs/any.schema.json\n    required: false\n    forms: [text]\n", i, i)
				files[fmt.Sprintf("data/s%d.txt", i)] = strings.Repeat("x", int(n))
			}
			m.WriteString("outputs:\n" + outs.String())
			files["pkg/metais.yaml"] = m.String()
			for k, v := range templates {
				files[k] = v
			}
			if err := writeFiles(dir, files); err != nil {
				return nil, err
			}
			return []string{"render", "--json", "pkg", "--data", "data", "--out", "out"}, nil
		},
		reached: func(r *calResult, lim calLimits) error {
			// Every input is loaded and validated, every output rendered,
			// and the last one parsed whole: nothing is refused but by
			// schema-work, which the last output's values may pass.
			for _, f := range r.Findings {
				if l := f.Params["limit"]; l != nil && l != "schema-work" {
					return fmt.Errorf("refused by %v at %s%s", l, f.Location.File, f.Location.Pointer)
				}
				if f.Params["limit"] == nil {
					return fmt.Errorf("%s at %s%s", f.Code, f.Location.File, f.Location.Pointer)
				}
			}
			return nil
		},
		beyond: func(r *calResult, lim calLimits) error { return r.refusedBy(limit, lim[limit], "") },
	}
}

// sizedText is a template whose output is n bytes: lines of 1,024 bytes
// written by a loop, then what is left.
func sizedText(n int64) string {
	return kibLines(n/1024*1024) + strings.Repeat("x", int(n%1024))
}

// a6Cases are the cases of ADR-009 Amendment A6 items 2 and 5.
var a6Cases = []calCase{
	{
		// One YAML schema file of exactly schema-yaml-size bytes, in the
		// shape item 5 names: const: [0, …], a value for every two bytes.
		// It compiles, and a byte less is refused before it is read.
		name: "schema-yaml-size", limit: "schema-yaml-size",
		lowered: calLimits{"schema-yaml-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			doc, _ := yamlZeros(lim["schema-yaml-size"] - int64(len("const: ")))
			files := map[string]string{
				"pkg/metais.yaml": manifestHead + "inputs:\n  - name: report\n    title: Report\n    schema: inputs/report.schema.yaml\n" +
					"outputs:\n  - id: a\n    template: a.txt.liquid\n",
				"pkg/templates/a.txt.liquid":    "a",
				"pkg/inputs/report.schema.yaml": "const: " + doc,
			}
			if err := writeFiles(dir, files); err != nil {
				return nil, err
			}
			return []string{"inspect", "--json", "pkg"}, nil
		},
		reached: succeeded,
		beyond:  refusedBy("schema-yaml-size"),
	},
	{
		// A packed package whose digest file is entry-size bytes of
		// values, a zero for every two bytes: it is refused as not in its
		// form once it has decoded entries plus 4 values, and decodes no
		// more.
		name: "digest-values", limit: "entries",
		lowered: calLimits{"entries": 100, "entry-size": 1 << 20},
		build: func(dir string, lim calLimits) ([]string, error) {
			const head = `{"algorithm":"sha256","entries":{},"package":[`
			k := (lim["entry-size"] - int64(len(head)) - 2) / 2
			digests := head + strings.Repeat("0,", int(k-1)) + "0]}"
			return packedWithDigests(dir, digests)
		},
		reached: func(r *calResult, lim calLimits) error {
			for _, f := range r.Findings {
				if f.Code == "package.digest.not_canonical" && f.Params["reason"] == "shape" {
					return nil
				}
			}
			return fmt.Errorf("the digest file was not refused as not in its form, reason shape: %v", r.Findings)
		},
	},
	{
		// example.yaml in its JSON form, exactly manifest-size bytes of
		// values, a zero for every two bytes, read whole: its extra
		// member is example.metadata after it is parsed, and a byte less
		// refuses it before it is read.
		name: "example-json", limit: "manifest-size",
		lowered: calLimits{"manifest-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			const head = `{"title":"x","n":[`
			k := (lim["manifest-size"] - int64(len(head)) - 2) / 2
			doc := head + strings.Repeat("0,", int(k-1)) + "0]}"
			doc += strings.Repeat(" ", int(lim["manifest-size"])-len(doc))
			files := map[string]string{
				"pkg/metais.yaml":               manifestHead + "inputs: []\noutputs:\n  - id: a\n    template: a.txt.liquid\n",
				"pkg/templates/a.txt.liquid":    "a",
				"pkg/examples/e/example.yaml":   doc,
				"pkg/examples/e/expected/a.txt": "a",
			}
			if err := writeFiles(dir, files); err != nil {
				return nil, err
			}
			return []string{"test", "--json", "pkg"}, nil
		},
		reached: exampleMetadata("member"),
		beyond:  exampleMetadata("syntax"),
	},
	{
		// A reference of exactly entry-size bytes: a title, then empty
		// level-2 headings, a heading for every three bytes. The open
		// takes the title and keeps no heading after it.
		name: "reference-title", limit: "entry-size",
		lowered: calLimits{"entry-size": 1 << 20},
		build: func(dir string, lim calLimits) ([]string, error) {
			const title = "# Title\n"
			n := lim["entry-size"] - int64(len(title))
			ref := title + strings.Repeat("##\n", int(n/3)) + strings.Repeat("\n", int(n%3))
			if err := writePackage(dir, "references/r.md", ref); err != nil {
				return nil, err
			}
			return []string{"inspect", "--json", "pkg"}, nil
		},
		reached: succeeded,
		beyond:  refusedBy("entry-size"),
	},
	{
		// A manifest of exactly manifest-size bytes whose unknown member
		// holds an array of a zero for every two bytes: at the defaults
		// its validation passes schema-work, and it is refused naming
		// that limit, never as a defect in the tool. A byte less is
		// refused by manifest-size.
		name: "manifest-schema-work", limit: "manifest-size",
		lowered: calLimits{"manifest-size": 64 << 10},
		build: func(dir string, lim calLimits) ([]string, error) {
			head := manifestHead + "inputs: []\noutputs:\n  - id: a\n    template: a.txt.liquid\nx: ["
			k := (lim["manifest-size"] - int64(len(head)) - 2) / 2
			m := head + strings.Repeat("0,", int(k-1)) + "0]\n"
			m += strings.Repeat("\n", int(lim["manifest-size"])-len(m))
			if err := writeFiles(dir, map[string]string{"pkg/metais.yaml": m, "pkg/templates/a.txt.liquid": "a"}); err != nil {
				return nil, err
			}
			return []string{"inspect", "--json", "pkg"}, nil
		},
		reached: func(r *calResult, lim calLimits) error {
			if r.OK {
				return fmt.Errorf("the package opened")
			}
			for _, f := range r.Findings {
				if f.Code == "internal.error" || f.Params["limit"] == "manifest-size" {
					return fmt.Errorf("%s %v", f.Code, f.Params)
				}
			}
			return nil
		},
		beyond: refusedBy("manifest-size"),
	},
}

// caseTool is the program a case's build runs: -conformance.tool, or the
// binary TestMain built.
func caseTool() string {
	if *flagTool != "" {
		return *flagTool
	}
	return builtTool
}

// exampleMetadata is a reached that wants the example refused with
// example.metadata for reason.
func exampleMetadata(reason string) func(*calResult, calLimits) error {
	return func(r *calResult, _ calLimits) error {
		for _, f := range r.Findings {
			if f.Code == "example.metadata" && f.Params["reason"] == reason {
				return nil
			}
		}
		return fmt.Errorf("no example.metadata for %s: %v", reason, r.Findings)
	}
}

// packedWithDigests packs a package of one output with the tool, beneath
// dir, and rewrites it with its digest file replaced by digests, stored,
// every other entry and header as the tool wrote it. It returns the
// command that opens the rewritten package.
func packedWithDigests(dir, digests string) ([]string, error) {
	files := map[string]string{
		"src/metais.yaml":            manifestHead + "inputs: []\noutputs:\n  - id: a\n    template: a.txt.liquid\n",
		"src/templates/a.txt.liquid": "a",
	}
	if err := writeFiles(dir, files); err != nil {
		return nil, err
	}
	p, err := runProc(caseTool(), []string{"pack", "src", "--out", "packed.metais"}, dir, nil, nil, defaultWall)
	if err != nil {
		return nil, err
	}
	if p.exit != 0 {
		return nil, fmt.Errorf("metais pack exited %d: %s", p.exit, p.stdout)
	}
	zr, err := zip.OpenReader(filepath.Join(dir, "packed.metais"))
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	var out bytes.Buffer
	zw := zip.NewWriter(&out)
	for _, f := range zr.File {
		h := f.FileHeader
		var content io.Reader
		if f.Name == ".compiled/digests.json" {
			h.Method, h.ReaderVersion, h.CRC32 = zip.Store, 10, crc32.ChecksumIEEE([]byte(digests))
			h.CompressedSize64, h.UncompressedSize64 = uint64(len(digests)), uint64(len(digests))
			content = strings.NewReader(digests)
		} else if content, err = f.OpenRaw(); err != nil {
			return nil, err
		}
		w, err := zw.CreateRaw(&h)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(w, content); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(dir, "pkg.metais"), out.Bytes(), 0o644); err != nil {
		return nil, err
	}
	return []string{"inspect", "--json", "pkg.metais"}, nil
}
