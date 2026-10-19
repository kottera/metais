// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/digest"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

// The benchmarks below are the commands ADR-010 D15 item 2's CPU budgets
// are set for, each run in-process as the binary runs it:
//
//   - BenchmarkInspect: inspect of a packed package of 10,000 entries,
//     whole-package verification included (the 500 ms budget);
//   - BenchmarkValidate: validate of a 10,000-item collection, read from
//     a data directory (the 500 ms budget).
//
// The table's 200 ms budget is internal/liquid's BenchmarkRender/table.
// Each reports cpu-ns/op beside ns/op, since a budget is CPU time.
// testdata/bench/baseline.txt is a sandbox figure, not a budget;
// docs/dev/benchmarks.md is how the module's benchmarks are run.

// benchManifest declares the one collection the benchmarks validate.
const benchManifest = `metais: "1.0"
name: bench
version: 1.0.0
title: Bench
description: The package ADR-010 D15's command benchmarks open.
guide: Supply the items.
inputs:
  - name: items
    title: Items
    schema: inputs/items.schema.json
    collection: true
outputs:
  - id: list
    template: list.md.liquid
`

// benchItemsSchema is the collection's schema: an array of objects of
// every kind of member, reached by a $ref, with no pattern.
const benchItemsSchema = `{
  "type": "array",
  "items": {"$ref": "#/$defs/item"},
  "$defs": {
    "item": {
      "type": "object",
      "required": ["id", "name", "status", "amount", "when"],
      "additionalProperties": false,
      "properties": {
        "id": {"type": "integer", "minimum": 0},
        "name": {"type": "string", "minLength": 1, "maxLength": 200},
        "status": {"enum": ["open", "held", "closed"]},
        "amount": {"type": "number", "exclusiveMinimum": -1},
        "when": {"type": "string", "format": "date"},
        "tags": {"type": "array", "items": {"type": "string"}, "uniqueItems": true, "maxItems": 8}
      }
    }
  }
}
`

// benchSources is the benchmark package's source files: the manifest,
// the schema, a template, and references enough that, packed with its
// compiled files and its digest file, the package has entries entries.
func benchSources(t testing.TB, entries int) map[string][]byte {
	t.Helper()
	files := map[string][]byte{
		"metais.yaml":              []byte(benchManifest),
		"inputs/items.schema.json": []byte(benchItemsSchema),
		"templates/list.md.liquid": []byte("{% for i in inputs.items %}- {{ i.name }}\n{% endfor %}"),
	}
	p, fs := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	if p == nil {
		t.Fatalf("the benchmark package does not open: %v", fs)
	}
	for i := range entries - len(files) - len(p.Compiled) - 1 {
		files[fmt.Sprintf("references/%02d/%04d.md", i%37, i)] = []byte(fmt.Sprintf("# Reference %d\n\n%s", i,
			strings.Repeat(fmt.Sprintf("Line of reference %d, which says what it says.\n", i), 8+i%32)))
	}
	return files
}

// packBench writes files as pack writes them, into a file of its own, and
// returns the file's path: every entry, the compiled files the loader
// derives and the digest file, in byte order of their names, each method
// auto (ADR-001 D4).
func packBench(t testing.TB, files map[string][]byte) string {
	t.Helper()
	p, fs := loader.OpenFiles(files, loader.Options{Limits: limits.Defaults()})
	if p == nil {
		t.Fatalf("the benchmark package does not open: %v", fs)
	}
	all := map[string][]byte{}
	for n, b := range files {
		all[n] = b
	}
	for _, f := range p.Compiled {
		all[f.Name] = f.Content
	}
	var list []digest.Entry
	for n, b := range all {
		list = append(list, digest.Entry{Name: n, Sum: sha256.Sum256(b)})
	}
	all[digest.File] = digest.Encode(list)
	var es []zipfmt.Entry
	for n, b := range all {
		es = append(es, zipfmt.Entry{Name: n, Content: b, Method: zipfmt.Auto})
	}
	slices.SortFunc(es, func(a, b zipfmt.Entry) int { return strings.Compare(a.Name, b.Name) })
	var buf bytes.Buffer
	if err := zipfmt.Write(&buf, es); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(testenv.Dir(t), "bench.metais")
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

// benchData writes a data directory holding the collection items, n
// items each a JSON file of its own, valid against benchItemsSchema, and
// returns its path.
func benchData(t testing.TB, n int) string {
	t.Helper()
	dir := filepath.Join(testenv.Dir(t), "data")
	if err := os.MkdirAll(filepath.Join(dir, "items"), 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range n {
		item := fmt.Sprintf(`{"id": %d, "name": "Item %d", "status": %q, "amount": %d.%02d, "when": "2026-%02d-%02d", "tags": ["t%d", "u%d"]}`+"\n",
			i, i, []string{"open", "held", "closed"}[i%3], i%5000, i%100, i%12+1, i%28+1, i%13, i%17)
		if err := os.WriteFile(filepath.Join(dir, "items", fmt.Sprintf("%05d.json", i)), []byte(item), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// benchRun runs the command line argv in-process, its streams discarded,
// as the binary runs it, and fails b unless it exits 0.
func benchRun(b *testing.B, argv ...string) {
	status := Run(context.Background(), Invocation{
		Args: argv, Stdin: strings.NewReader(""), Stdout: io.Discard, Stderr: io.Discard,
		Options: Options{Display: hostenv.Display{NoColor: true}},
	})
	if status != 0 {
		b.Fatalf("metais %q exited %d", argv, status)
	}
}

// BenchmarkInspect is inspect of a packed package of 10,000 entries, the
// entries limit's default, whole-package verification included: ADR-010
// D15 item 2's 500 ms budget.
func BenchmarkInspect(b *testing.B) {
	testenv.Home(b)
	pkg := packBench(b, benchSources(b, int(limits.Defaults().Value(limits.Entries))))
	argv := []string{"inspect", "--json", pkg}
	benchRun(b, argv...)
	b.ReportAllocs()
	cpu := startCPU()
	for b.Loop() {
		benchRun(b, argv...)
	}
	cpu.report(b)
}

// BenchmarkValidate is validate of a 10,000-item collection, each item a
// file of the data directory: ADR-010 D15 item 2's 500 ms budget.
func BenchmarkValidate(b *testing.B) {
	testenv.Home(b)
	pkg := filepath.Join(testenv.Dir(b), "pkg")
	writeBenchSources(b, pkg, benchSources(b, 4))
	argv := []string{"validate", "--json", pkg, "--data", benchData(b, 10_000)}
	benchRun(b, argv...)
	b.ReportAllocs()
	cpu := startCPU()
	for b.Loop() {
		benchRun(b, argv...)
	}
	cpu.report(b)
}

// writeBenchSources writes files as a source directory at dir.
func writeBenchSources(t testing.TB, dir string, files map[string][]byte) {
	t.Helper()
	for n, b := range files {
		p := filepath.Join(dir, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// cpuSpan is the process's CPU time at the start of a benchmark's loop.
type cpuSpan struct {
	at time.Duration
	ok bool
}

func startCPU() cpuSpan {
	at, ok := selfCPU()
	return cpuSpan{at, ok}
}

// report reports the CPU time each iteration of b's loop took, as
// cpu-ns/op, where the platform measures it: the unit of ADR-010 D15
// item 2's budgets.
func (s cpuSpan) report(b *testing.B) {
	if now, ok := selfCPU(); ok && s.ok && b.N > 0 {
		b.ReportMetric(float64(now-s.at)/float64(b.N), "cpu-ns/op")
	}
}

// TestBenchCommands holds the command benchmarks to the work they time,
// in every go test, through runTool's checks of each run: the packed
// package holds exactly the entries asked for, and inspect opens it as
// packed and lists its every reference; validate finds the items valid,
// and reports one made invalid, so the validation it times looks. The
// test builds 300 of each, since the race-detecting gate need not pack
// and write 10,000; the benchmarks build their 10,000 with the same
// functions.
func TestBenchCommands(t *testing.T) {
	testenv.Home(t)
	const entries = 300
	files := benchSources(t, entries)
	pkg := packBench(t, files)
	b, err := os.ReadFile(pkg)
	if err != nil {
		t.Fatal(err)
	}
	a, fs, err := zipfmt.Open(bytes.NewReader(b), int64(len(b)), limits.Defaults())
	if err != nil || len(fs) > 0 || len(a.List()) != entries {
		t.Fatalf("the benchmark package does not hold %d entries: %v %v", entries, err, fs)
	}
	r := runTool(t, "inspect", "--json", pkg)
	if r.status != 0 || !r.doc.OK {
		t.Fatalf("inspect of the benchmark package: %s", r.stdout)
	}
	var res struct {
		Package struct {
			Source string `json:"source"`
		} `json:"package"`
		References struct {
			Count int `json:"count"`
		} `json:"references"`
	}
	result(t, r, &res)
	if refs := len(files) - 3; res.Package.Source != "packed" || res.References.Count != refs {
		t.Errorf("inspect saw a %s package of %d references, want a packed one of %d", res.Package.Source, res.References.Count, refs)
	}

	src := filepath.Join(testenv.Dir(t), "pkg")
	writeBenchSources(t, src, benchSources(t, 4))
	data := benchData(t, entries)
	if r := runTool(t, "validate", "--json", src, "--data", data); r.status != 0 || !r.doc.OK {
		t.Fatalf("validate of the benchmark collection: %s", r.stdout[:min(len(r.stdout), 2000)])
	}
	if err := os.WriteFile(filepath.Join(data, "items", "00150.json"), []byte(`{"id": 1, "name": "", "status": "gone", "amount": 1, "when": "2026-01-01"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if r := runTool(t, "validate", "--json", src, "--data", data); r.status == 0 {
		t.Error("an invalid item is not reported: the benchmark's validation does not look")
	}
}
