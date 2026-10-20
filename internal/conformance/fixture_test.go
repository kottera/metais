// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The self-proof's fixture suite: a spec tree of its own, with a front
// page, a registry of fixture codes, the suite's real schemas, and vectors
// whose packages script the fake. It is written to a temporary directory,
// so it never sits under spec/conformance/, and generated rather than
// committed, so its containers and streams are built by the same writers
// the fake uses.

// fixtureBounds are fix/bounded's lowered bounds, which the fake's cpu
// and memory faults exceed.
var fixtureBounds = bounds{CPUms: 50, RSSMiB: 32}

// fixtureRegistry is the fixture tree's finding-code registry.
var fixtureRegistry = map[string]any{"codes": []any{
	map[string]any{"code": "package.fake_refused", "severity": "error", "class": "refused"},
	map[string]any{"code": "package.fake_digest", "severity": "error", "class": "refused"},
	map[string]any{"code": "input.fake_invalid", "severity": "error", "class": "content"},
	map[string]any{"code": "input.fake_unused", "severity": "warning", "class": "content"},
	map[string]any{"code": "output.fake_write_failed", "severity": "error", "class": "destination"},
}}

// The fixture vectors' ids.
const (
	fixRenderTree   = "fix/render-tree"
	fixRefuse       = "fix/refuse"
	fixValidate     = "fix/validate"
	fixRenderStream = "fix/render-stream"
	fixRenderDry    = "fix/render-dry"
	fixBounded      = "fix/bounded"
	fixRepeatable   = "fix/repeatable"
	fixUnwritable   = "fix/unwritable"
	fixPack         = "fix/pack"
)

// fixtureTree is the tree fix/render-tree's package renders.
var fixtureTree = map[string]string{
	"a.txt":     "alpha\n",
	"sub/b.txt": "beta\n",
	"bin/run":   "#!/bin/sh\necho run\n",
	"about.txt": "made by metais-fake 0.0.0-fake\n",
}

// fixtureZip is the container fix/render-tree's package renders.
var fixtureZip = map[string]string{
	"mimetype":      "application/x-fake",
	"content/c.txt": padding + "chapter one\n",
}

// writeFixture writes the fixture spec tree below root and returns the
// suite directory, root/conformance.
func writeFixture(t testing.TB, root string) string {
	t.Helper()
	put := func(rel string, b []byte) {
		t.Helper()
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, b, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	putJSON := func(rel string, v any) {
		t.Helper()
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		put(rel, append(b, '\n'))
	}
	put("README.md", []byte("```\nVersion: 1.0\nStatus: Draft\nLatest release: none\n```\n\n# A fixture specification\n"))
	putJSON("findings/registry.json", fixtureRegistry)
	putJSON("statements.json", map[string]any{"statements": []any{
		map[string]any{"id": "MTS-FIX-001", "chapter": "fixture", "section": "fixture", "kind": "requirement",
			"since": "1.0", "observable": true, "status": "active"},
		map[string]any{"id": "MTS-OTH-001", "chapter": "fixture", "section": "fixture", "kind": "requirement",
			"since": "1.0", "observable": true, "status": "active"},
	}})
	for _, n := range []string{vectorSchema, findingsSchema, driverRequestSchema, driverResponseSchema, recordSchema} {
		b, err := os.ReadFile(filepath.Join(moduleRoot, "spec", "schemas", "conformance", "1", n))
		if err != nil {
			t.Fatal(err)
		}
		put("schemas/conformance/1/"+n, b)
	}

	vec := func(id string) string { return "conformance/vectors/" + id + "/" }
	vector := func(id string, extra map[string]any, cases ...any) {
		v := map[string]any{
			"$schema":    schemaBase + vectorSchema,
			"id":         id,
			"title":      "a fixture of the runner's self-proof",
			"statements": []string{"MTS-FIX-001"},
			"since":      "1.0",
			"class":      "processor",
			"tags":       []string{},
			"requires":   []string{},
			"notes":      "",
			"cases":      cases,
		}
		for k, x := range extra {
			v[k] = x
		}
		putJSON(vec(id)+"vector.json", v)
	}
	step := func(op string, args, expect map[string]any) map[string]any {
		return map[string]any{"op": op, "args": args, "expect": expect}
	}
	tcase := func(name string, steps ...any) map[string]any {
		return map[string]any{"name": name, "steps": steps}
	}
	putTree := func(dir string, files map[string]string) {
		for n, c := range files {
			put(dir+n, []byte(c))
		}
	}
	zipEntries := func() []zipEntry {
		return []zipEntry{
			{name: "content/c.txt", method: methodDeflate, content: []byte(fixtureZip["content/c.txt"])},
			{name: "mimetype", method: methodStore, content: []byte(fixtureZip["mimetype"])},
		}
	}
	renderScript := map[string]any{
		"ok": true, "exit": 0, "findings": []any{}, "tree": "tree",
		"executable": []string{"bin/run"},
		"zips":       []any{map[string]any{"path": "book.zip", "dir": "zip", "stored": []string{"mimetype"}}},
	}
	renderTreeExpect := map[string]any{"ok": true, "trees": []any{map[string]any{
		"path": "out", "equals": "expected/out",
		"containers": []any{map[string]any{"path": "book.zip", "methods": []string{"mimetype"}}},
		"executable": []string{"bin/run"},
		"substitute": []string{"about.txt"},
	}}}
	expectedTree := map[string]string{}
	for n, c := range fixtureTree {
		expectedTree[n] = c
	}
	expectedTree["about.txt"] = "made by @@TOOL_NAME@@ @@TOOL_VERSION@@\n"

	// fix/render-tree: a render into a directory, with a container, an
	// executable file and a substituted file, reading a data tree that
	// gets decoys.
	for _, id := range []string{fixRenderTree, fixBounded} {
		putJSON(vec(id)+"package/fake.json", renderScript)
		putTree(vec(id)+"package/tree/", fixtureTree)
		putTree(vec(id)+"package/zip/", fixtureZip)
		putTree(vec(id)+"data/", map[string]string{"input.json": "{}\n"})
		putTree(vec(id)+"expected/out/", expectedTree)
		put(vec(id)+"expected/out/book.zip", writeZip(zipEntries(), false))
	}
	vector(fixRenderTree, nil, tcase("render",
		step("render", map[string]any{"package": "package", "data": "data", "out": "out"}, renderTreeExpect)))
	bounded := step("render", map[string]any{"package": "package", "data": "data", "out": "out"}, renderTreeExpect)
	bounded["bounds"] = map[string]any{"cpu_ms": fixtureBounds.CPUms, "rss_mib": fixtureBounds.RSSMiB}
	vector(fixBounded, nil, tcase("render", bounded))

	// fix/refuse: a hostile package every opening command refuses.
	refused := []any{
		map[string]any{"code": "package.fake_digest", "severity": "error", "message": "m", "hint": "h",
			"subject": map[string]any{"kind": "package"}, "params": map[string]any{"entry": "a.txt"}},
		map[string]any{"code": "package.fake_refused", "severity": "error", "message": "m", "hint": "h",
			"subject": map[string]any{"kind": "package"}, "params": map[string]any{"reason": "fixture"}},
	}
	putJSON(vec(fixRefuse)+"package/fake.json", map[string]any{"ok": false, "exit": 4, "findings": refused})
	putJSON(vec(fixRefuse)+"expected/findings.json", projections(refused))
	vector(fixRefuse, map[string]any{"tags": []string{"security"}}, tcase("open",
		step("open", map[string]any{"package": "package"}, map[string]any{"ok": false, "findings": "expected/findings.json"})))

	// fix/validate: inputs refused with two findings, in order.
	invalid := []any{
		map[string]any{"code": "input.fake_invalid", "severity": "error", "message": "m", "hint": "h",
			"subject": map[string]any{"kind": "input", "name": "first"}, "params": map[string]any{"length": 3}},
		map[string]any{"code": "input.fake_unused", "severity": "warning", "message": "m", "hint": "h",
			"subject": map[string]any{"kind": "input", "name": "second"}, "params": map[string]any{}},
	}
	putJSON(vec(fixValidate)+"package/fake.json", map[string]any{"ok": false, "exit": 1, "findings": invalid})
	putTree(vec(fixValidate)+"data/", map[string]string{"first.json": "[1, 2, 3]\n"})
	putJSON(vec(fixValidate)+"expected/findings.json", projections(invalid))
	vector(fixValidate, nil, tcase("validate",
		step("validate", map[string]any{"package": "package", "data": "data"}, map[string]any{"ok": false, "findings": "expected/findings.json"})))

	// fix/render-stream: the whole render as a tar, and one output alone.
	streamTree := map[string]string{"a.txt": fixtureTree["a.txt"], "sub/b.txt": fixtureTree["sub/b.txt"]}
	putJSON(vec(fixRenderStream)+"package/fake.json", map[string]any{"ok": true, "exit": 0, "findings": []any{}, "tree": "tree"})
	putTree(vec(fixRenderStream)+"package/tree/", streamTree)
	streamFiles := map[string][]byte{}
	for n, c := range streamTree {
		streamFiles[n] = []byte(c)
	}
	put(vec(fixRenderStream)+"expected/stream.tar", writeTar(streamFiles, nil, false))
	put(vec(fixRenderStream)+"expected/a.txt", []byte(streamTree["a.txt"]))
	vector(fixRenderStream, nil,
		tcase("archive", step("render", map[string]any{"package": "package", "stream": "archive"},
			map[string]any{"ok": true, "stream": map[string]any{"equals": "expected/stream.tar"}})),
		tcase("only", step("render", map[string]any{"package": "package", "stream": map[string]any{"only": "a.txt"}},
			map[string]any{"ok": true, "stream": map[string]any{"equals": "expected/a.txt"}})))

	// fix/render-dry: a dry run's listing, and nothing written.
	// Of a container, only the path is compared: its size and digest are
	// the encoder's.
	listing := []any{
		map[string]any{"path": "a.txt", "size": 6, "digest": "sha256:" + digest([]byte(fixtureTree["a.txt"]))},
		map[string]any{"path": "book.zip", "size": 210, "digest": "sha256:" + digest([]byte("one encoder"))},
		map[string]any{"path": "sub/b.txt", "size": 5, "digest": "sha256:" + digest([]byte(fixtureTree["sub/b.txt"]))},
	}
	expectedListing := slices.Clone(listing)
	expectedListing[1] = map[string]any{"path": "book.zip", "size": 199, "digest": "sha256:" + digest([]byte("another encoder"))}
	putJSON(vec(fixRenderDry)+"package/fake.json", map[string]any{"ok": true, "exit": 0, "findings": []any{}, "tree": "tree", "listing": listing})
	putTree(vec(fixRenderDry)+"package/tree/", streamTree)
	putJSON(vec(fixRenderDry)+"expected/listing.json", expectedListing)
	vector(fixRenderDry, nil, tcase("dry-run",
		step("render", map[string]any{"package": "package", "out": "out", "dry_run": true},
			map[string]any{"ok": true, "listing": map[string]any{"equals": "expected/listing.json", "containers": []string{"book.zip"}}})))

	// fix/repeatable: bytes the implementation chooses, which must be the
	// same in every run.
	putJSON(vec(fixRepeatable)+"package/fake.json", map[string]any{"ok": true, "exit": 0, "findings": []any{}, "tree": "tree"})
	putTree(vec(fixRepeatable)+"package/tree/", streamTree)
	repeat := step("render", map[string]any{"package": "package", "out": "out"},
		map[string]any{"ok": true, "trees": []any{map[string]any{"path": "out", "files": []string{"a.txt", "sub/b.txt"}}}})
	repeat["repeatable"] = true
	vector(fixRepeatable, nil, tcase("render", repeat))

	// fix/unwritable: a render into a directory beneath one the setup
	// makes unwritable (ADR-011 D9 item 6, A2). The fake writes as the
	// package says and reports the refusal its script names when the
	// write fails, as a correct implementation does.
	writeFailed := []any{
		map[string]any{"code": "output.fake_write_failed", "severity": "error", "message": "m", "hint": "h",
			"subject": map[string]any{"kind": "invocation"}, "params": map[string]any{"reason": "permission_denied"}},
	}
	putJSON(vec(fixUnwritable)+"package/fake.json", map[string]any{"ok": true, "exit": 0, "findings": []any{}, "tree": "tree",
		"write_failed": writeFailed, "write_failed_exit": 5})
	putTree(vec(fixUnwritable)+"package/tree/", streamTree)
	putJSON(vec(fixUnwritable)+"expected/findings.json", projections(writeFailed))
	unwritable := tcase("locked", step("render", map[string]any{"package": "package", "out": "locked/out"},
		map[string]any{"ok": false, "findings": "expected/findings.json"}))
	unwritable["setup"] = map[string]any{"unwritable": []string{"locked"}}
	vector(fixUnwritable, map[string]any{"requires": []string{"unwritable-dirs"}}, unwritable)

	// fix/pack: a source directory packed into a directory, as the
	// package its script names, whose deflated entry hides what it holds
	// from a scan of its raw bytes.
	putJSON(vec(fixPack)+"package/fake.json", map[string]any{"ok": true, "exit": 0, "findings": []any{},
		"pack": map[string]any{"path": fixturePackName, "dir": "src", "stored": []string{"mimetype"}}})
	putTree(vec(fixPack)+"package/src/", fixtureZip)
	put(vec(fixPack)+"expected/out/"+fixturePackName, writeZip(zipEntries(), false))
	pack := tcase("pack", step("pack", map[string]any{"source": "package", "to": "out"},
		map[string]any{"ok": true, "trees": []any{map[string]any{
			"path": "out", "equals": "expected/out",
			"containers": []any{map[string]any{"path": fixturePackName, "methods": []string{"mimetype"}}},
		}}}))
	pack["setup"] = map[string]any{"dirs": []string{"out"}}
	vector(fixPack, map[string]any{"class": "authoring"}, pack)

	return filepath.Join(root, "conformance")
}

// fixturePackName is the package fix/pack's source packs into.
const fixturePackName = "fake-1.0.0.metais"

// projections keeps of each finding the members a vector may expect
// (ADR-011 D10 item 2): never message or hint.
func projections(findings []any) []any {
	var out []any
	for _, f := range findings {
		p := cloneMap(f.(map[string]any))
		delete(p, "message")
		delete(p, "hint")
		out = append(out, p)
	}
	return out
}
