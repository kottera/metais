// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

// The fake implementation is this test binary re-executed in a directory
// named metais-fake-<fault> (ADR-011 D18 item 1). It serves either
// adapter: with arguments it is a metais-compatible command line, and
// with none it is a driver reading one request. What it does is scripted
// by the fixture package's fake.json; the fault breaks exactly one thing
// the runner checks. The fault "clean" breaks nothing.

// fakeTool is the identity the fake reports.
var fakeTool = toolID{Name: "metais-fake", Version: "0.0.0-fake"}

// fakeScript is a fixture package's fake.json: what a correct
// implementation gives for the package.
type fakeScript struct {
	OK         bool             `json:"ok"`
	Exit       int              `json:"exit"`
	Findings   []map[string]any `json:"findings"`
	Tree       string           `json:"tree"`
	Executable []string         `json:"executable"`
	Zips       []fakeZip        `json:"zips"`
	Listing    []any            `json:"listing"`
}

// fakeZip is a container the fake writes: the files below Dir in the
// package, as the entries of Path, with the entries Stored stored and the
// rest deflated.
type fakeZip struct {
	Path   string   `json:"path"`
	Dir    string   `json:"dir"`
	Stored []string `json:"stored"`
}

// fakeCall is one operation the fake performs.
type fakeCall struct {
	op      string
	pkg     string
	out     string // the output directory, or ""
	archive bool   // stream every output as a tar
	only    string // stream this one output
	dryRun  bool
}

// fakeResult is what the operation gave.
type fakeResult struct {
	ok       bool
	exit     int
	findings []any
	listing  []any
	stream   []byte
	note     string // a free member of the document, which nothing compares
}

// fakeMain runs the fake with fault and the command-line arguments args.
func fakeMain(fault string, args []string) int {
	if len(args) == 0 {
		return fakeDriver(fault)
	}
	cmd := args[0]
	switch cmd {
	case "version":
		return fakeEnvelope(os.Stdout, "metais.version/1", true, []any{}, map[string]any{
			"tool": fakeTool, "specification": map[string]any{"implements": []string{"1.0"}},
		}, 0)
	case "help":
		var cmds []map[string]any
		for _, n := range []string{"inspect", "ls", "schema", "validate", "render", "unpack"} {
			cmds = append(cmds, map[string]any{"name": n, "opens_package": true})
		}
		if fault == "help-unmapped" {
			cmds = append(cmds, map[string]any{"name": "peek", "opens_package": true})
		}
		for _, n := range []string{"version", "help", "limits"} {
			cmds = append(cmds, map[string]any{"name": n, "opens_package": false})
		}
		return fakeEnvelope(os.Stdout, "metais.help/1", true, []any{}, map[string]any{"commands": cmds}, 0)
	}
	call := fakeCall{op: cmd}
	if cmd == "inspect" || cmd == "ls" || cmd == "schema" {
		call.op = "open"
	}
	for i := 1; i < len(args); i++ {
		switch a := args[i]; {
		case a == "--out" || a == "--to":
			i++
			if args[i] == "-" {
				call.archive = true
			} else {
				call.out = args[i]
			}
		case a == "--only":
			i++
			call.only, call.archive = args[i], false
		case a == "--dry-run":
			call.dryRun = true
		case a == "--data" || a == "--set" || a == "--set-json" || a == "--limit" || a == "--inputs-json":
			i++
		case strings.HasPrefix(a, "-"):
		case call.pkg == "":
			call.pkg = a
		}
	}
	res, err := fakeRun(fault, call)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}
	doc := io.Writer(os.Stdout)
	if res.stream != nil {
		os.Stdout.Write(res.stream)
		doc = os.Stderr
	}
	result := map[string]any{"note": res.note}
	if call.dryRun {
		result["files"] = res.listing
	}
	exit := res.exit
	switch fault {
	case "ok":
		res.ok = !res.ok
		if res.ok {
			exit = 0
		}
	case "exit":
		exit = 1
		if res.exit != 0 {
			exit = 5
		}
	}
	return fakeEnvelope(doc, "metais."+cmd+"/1", res.ok, res.findings, result, exit)
}

// fakeEnvelope prints ADR-008 D10's envelope and returns exit.
func fakeEnvelope(w io.Writer, format string, ok bool, findings []any, result any, exit int) int {
	if findings == nil {
		findings = []any{}
	}
	b, _ := json.Marshal(map[string]any{
		"format": format, "ok": ok, "findings": findings,
		"summary": map[string]any{"errors": 0}, "result": result, "next": []any{},
	})
	w.Write(append(b, '\n'))
	return exit
}

// fakeDriver serves one driver request (ADR-011 D11 item 6).
func fakeDriver(fault string) int {
	var req struct {
		Op         string          `json:"op"`
		Args       json.RawMessage `json:"args"`
		StreamFile string          `json:"stream_file"`
	}
	if err := json.NewDecoder(os.Stdin).Decode(&req); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	switch fault {
	case "driver-malformed":
		os.Stdout.WriteString(`{"ok": true, "findings": [`)
		return 0
	case "driver-exit":
		return 1
	}
	if req.Op == "version" {
		b, _ := json.Marshal(map[string]any{"tool": fakeTool})
		os.Stdout.Write(b)
		return 0
	}
	var a stepArgs
	if err := json.Unmarshal(req.Args, &a); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	call := fakeCall{op: req.Op, pkg: a.Package, out: a.Out, dryRun: a.DryRun}
	call.archive, call.only, _ = a.stream()
	res, err := fakeRun(fault, call)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if res.stream != nil {
		if err := os.WriteFile(req.StreamFile, res.stream, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}
	if fault == "ok" {
		res.ok = !res.ok
	}
	var listing any
	if call.dryRun {
		listing = res.listing
	}
	if res.findings == nil {
		res.findings = []any{}
	}
	b, _ := json.Marshal(map[string]any{"ok": res.ok, "findings": res.findings, "listing": listing, "tool": fakeTool, "note": res.note})
	os.Stdout.Write(b)
	return 0
}

// fakeRun performs call as the package's script says, then breaks what
// fault names.
func fakeRun(fault string, call fakeCall) (*fakeResult, error) {
	b, err := os.ReadFile(filepath.Join(call.pkg, "fake.json"))
	if err != nil {
		return nil, err
	}
	var sc fakeScript
	if err := json.Unmarshal(b, &sc); err != nil {
		return nil, err
	}
	res := &fakeResult{ok: sc.OK, exit: sc.Exit}
	for _, f := range sc.Findings {
		res.findings = append(res.findings, f)
	}
	res.listing = sc.Listing

	// The bound faults break environment A's run only, which is where
	// the runner must catch them; B's run then differs from A's only in
	// what the bounds measure.
	inA := os.Getenv("METAIS_SENTINEL") != ""
	switch {
	case fault == "cpu" && inA:
		burnCPU(fakeCPUBound() * 3 / 2)
	case fault == "rss" && inA:
		holdMemory(fakeRSSBound() * 3 / 2)
	case fault == "wall" && inA:
		time.Sleep(time.Minute)
	}
	switch fault {
	case "finding-order":
		res.findings[0], res.findings[1] = res.findings[1], res.findings[0]
	case "finding-member":
		f := cloneMap(res.findings[0].(map[string]any))
		f["subject"] = map[string]any{"kind": "input", "name": "other"}
		res.findings[0] = f
	case "finding-extra":
		res.findings = append(res.findings, res.findings[0])
	case "finding-missing":
		res.findings = res.findings[:len(res.findings)-1]
	case "listing":
		l := cloneMap(res.listing[0].(map[string]any))
		l["size"] = 999
		res.listing = append([]any{l}, res.listing[1:]...)
	case "sentinel-stdout":
		res.note = os.Getenv("HOME")
	case "sentinel-stderr":
		fmt.Fprintln(os.Stderr, "home is", os.Getenv("HOME"))
	case "write-outside":
		os.WriteFile("stray.txt", []byte("stray\n"), 0o644)
	case "tmp-left":
		os.WriteFile(filepath.Join(os.Getenv("TMPDIR"), "left.txt"), []byte("left\n"), 0o644)
	}
	if !sc.OK || call.op != "render" {
		return res, nil
	}
	files, exec, err := fakeOutputs(fault, call.pkg, &sc)
	if err != nil {
		return nil, err
	}
	switch {
	case call.dryRun:
	case call.archive:
		res.stream = writeTar(files, exec, fault == "sentinel-tar")
		if fault == "stream" {
			res.stream = writeTar(files, append(exec, "a.txt"), false)
		}
	case call.only != "":
		res.stream = files[call.only]
		if fault == "stream" {
			res.stream = append(slices.Clone(res.stream), '!')
		}
	case call.out != "":
		if err := writeOutputs(call.out, files, exec, fault); err != nil {
			return nil, err
		}
	}
	return res, nil
}

// fakeOutputs reads the outputs the script names, with fault applied:
// the tree's files and the containers built from the package.
func fakeOutputs(fault, pkg string, sc *fakeScript) (map[string][]byte, []string, error) {
	files := map[string][]byte{}
	if sc.Tree != "" {
		root := filepath.Join(pkg, sc.Tree)
		err := filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(root, p)
			b, err := os.ReadFile(p)
			files[filepath.ToSlash(rel)] = b
			return err
		})
		if err != nil {
			return nil, nil, err
		}
	}
	for _, z := range sc.Zips {
		entries, err := zipEntriesFrom(filepath.Join(pkg, z.Dir), z.Stored)
		if err != nil {
			return nil, nil, err
		}
		entries = breakZip(fault, entries)
		files[z.Path] = writeZip(entries, fault == "zip-header")
	}
	exec := slices.Clone(sc.Executable)
	switch fault {
	case "tree-extra":
		files["extra.txt"] = []byte("extra\n")
	case "tree-missing":
		delete(files, "sub/b.txt")
	case "tree-changed":
		files["a.txt"] = append(slices.Clone(files["a.txt"]), '!')
	case "executable":
		exec = append(exec, "a.txt")
	case "sentinel-file":
		files["a.txt"] = append(slices.Clone(files["a.txt"]), os.Getenv("METAIS_SENTINEL")...)
	case "repeatable":
		files["a.txt"] = append(slices.Clone(files["a.txt"]), fmt.Sprintf("run %d\n", os.Getpid())...)
	case "ab":
		if os.Getenv("TZ") == "UTC" {
			files["a.txt"] = append(slices.Clone(files["a.txt"]), "utc\n"...)
		}
	}
	return files, exec, nil
}

// zipEntriesFrom reads the files below dir as zip entries, sorted by name.
func zipEntriesFrom(dir string, stored []string) ([]zipEntry, error) {
	var entries []zipEntry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(dir, p)
		name := filepath.ToSlash(rel)
		b, err := os.ReadFile(p)
		method := uint16(methodDeflate)
		if slices.Contains(stored, name) || len(b) == 0 {
			method = methodStore
		}
		entries = append(entries, zipEntry{name: name, method: method, content: b})
		return err
	})
	slices.SortFunc(entries, func(a, b zipEntry) int { return strings.Compare(a.name, b.name) })
	return entries, err
}

// breakZip applies a container fault to a container's entries.
func breakZip(fault string, entries []zipEntry) []zipEntry {
	entries = slices.Clone(entries)
	switch fault {
	case "zip-order":
		slices.Reverse(entries)
	case "zip-content":
		e := &entries[len(entries)-1]
		e.content = append(slices.Clone(e.content), '!')
	case "zip-method":
		for i := range entries {
			if entries[i].method == methodStore && len(entries[i].content) > 0 {
				entries[i].method = methodDeflate
			}
		}
	case "sentinel-zip":
		for i := range entries {
			if entries[i].method == methodDeflate {
				entries[i].content = append(slices.Clone(entries[i].content), os.Getenv("METAIS_SENTINEL")...)
				break
			}
		}
	case "sentinel-zip-nested":
		deep := writeZip([]zipEntry{{name: "deep.txt", method: methodDeflate, content: []byte(padding + os.Getenv("METAIS_SENTINEL") + padding)}}, false)
		inner := writeZip([]zipEntry{{name: "inner.zip", method: methodDeflate, content: deep}}, false)
		entries = append(entries, zipEntry{name: "nested.zip", method: methodDeflate, content: inner})
	}
	return entries
}

// padding makes a deflated entry compress, so the token inside it is
// coded and only inflating it finds it.
const padding = "padding padding padding padding padding padding padding padding\n"

// writeTar writes files, by path, as a tar stream; sentinel adds the
// sentinel to the first member.
func writeTar(files map[string][]byte, exec []string, sentinelled bool) []byte {
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	for i, n := range sortedKeys(files) {
		content := files[n]
		if sentinelled && i == 0 {
			content = append(slices.Clone(content), os.Getenv("METAIS_SENTINEL")...)
		}
		mode := int64(0o644)
		if slices.Contains(exec, n) {
			mode = 0o755
		}
		tw.WriteHeader(&tar.Header{Name: n, Mode: mode, Size: int64(len(content)), Typeflag: tar.TypeReg,
			ModTime: time.Unix(0, 0), Format: tar.FormatUSTAR})
		tw.Write(content)
	}
	tw.Close()
	return buf.Bytes()
}

// writeOutputs writes files below out, with the tree faults that are not
// a matter of content.
func writeOutputs(out string, files map[string][]byte, exec []string, fault string) error {
	for _, n := range sortedKeys(files) {
		p := filepath.Join(out, filepath.FromSlash(n))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return err
		}
		mode := fs.FileMode(0o644)
		if slices.Contains(exec, n) {
			mode = 0o755
		}
		if err := os.WriteFile(p, files[n], mode); err != nil {
			return err
		}
	}
	switch fault {
	case "tree-emptydir":
		return os.Mkdir(filepath.Join(out, "empty"), 0o755)
	case "tree-link":
		return os.Symlink("a.txt", filepath.Join(out, "link.txt"))
	}
	return nil
}

func cloneMap(m map[string]any) map[string]any {
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// The bounds of the fixture vector the fake breaks them on, as the
// runner applies them to this binary.
func fakeCPUBound() time.Duration { return stepLimits(&fixtureBounds, raceEnabled, defaultWall).cpu }
func fakeRSSBound() int64         { return stepLimits(&fixtureBounds, raceEnabled, defaultWall).rss }

// burnCPU spins until this process has used d of CPU time.
func burnCPU(d time.Duration) {
	start := selfCPU()
	x := 0
	for selfCPU()-start < int64(d) {
		for i := range 100000 {
			x += i
		}
	}
	_ = x
}

// holdMemory makes n bytes resident, touching every page.
func holdMemory(n int64) {
	b := make([]byte, n)
	for i := 0; i < len(b); i += 4096 {
		b[i] = 1
	}
	if b[len(b)-1] != 0 {
		os.Exit(3)
	}
}
