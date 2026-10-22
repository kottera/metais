// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

// The SC7 harness's additions to the conformance run (ADR-009 D16 item 1),
// which a runner with sc7 set makes to every case: environment A's further
// variables and decoys, the two path modes, and every command over each
// package the case names. TestHostIsolation runs them over every vector;
// TestConformance runs ADR-011 D12's environments alone.

// pathMode is how the paths a step names reach the implementation.
type pathMode int

const (
	// relPaths gives them relative to the working directory, as the
	// vector wrote them.
	relPaths pathMode = iota
	// absPaths gives them as absolute paths into the work directory, from
	// a working directory outside it, so a tool that prints a path it was
	// given, or makes one absolute, prints the sandbox's sentinel.
	absPaths
)

// metaisVars is how many variables named METAIS_<n> environment A
// carries.
const metaisVars = 32

// harness prepares sb for the SC7 harness in mode: in the absolute-path
// mode, a working directory outside the work directory; and in
// environment A, ADR-009 D16 item 1's further variables, each valued with
// a sentinel naming it, the two that hold a path as a path with a
// sentinel component. Nothing is inherited in either environment.
func (sb *sandbox) harness(mode pathMode) error {
	if mode == absPaths {
		sb.abs = true
		sb.cwd = filepath.Join(sb.root, "away")
		if err := os.MkdirAll(sb.cwd, 0o755); err != nil {
			return err
		}
	}
	if sb.env != envA {
		return nil
	}
	sb.vars["PWD"] = sb.cwd
	sb.vars["OLDPWD"] = filepath.Join(sb.root, "old", sentinel+"OLDPWD")
	sb.vars["SHELL"] = filepath.Join(sb.root, "bin", sentinel+"BIN", sentinel+"SHELL")
	sb.vars["SOURCE_DATE_EPOCH"] = sentinel + "EPOCH"
	sb.vars["CI"] = sentinel + "CI"
	sb.vars["GITHUB_TOKEN"] = sentinel + "GHTOKEN"
	sb.vars["AWS_SECRET_ACCESS_KEY"] = sentinel + "AWSKEY"
	for n := 1; n <= metaisVars; n++ {
		sb.vars["METAIS_"+strconv.Itoa(n)] = sentinel + "METAIS" + strconv.Itoa(n)
	}
	return nil
}

// carelessFiles are the files a careless tool might read from a home, a
// working or a temporary directory (ADR-009 D16 item 1).
var carelessFiles = []string{
	".config/metais/config.yaml",
	".metais/config.yaml",
	".ssh/id_ed25519",
	".netrc",
	".gitconfig",
}

// reachPrefix begins the name of every directory the harness's commands
// write into, in the work directory.
const reachPrefix = "metais-reach-"

// plantHarnessDecoys writes the SC7 harness's decoys in environment A
// (ADR-009 D16 item 1): in the working directory, the home directory and
// the temporary directory, with sentinel content naming the directory, a
// decoy of every name the case's packages refer to and of every careless
// file, unless the directory already holds something of that name. A
// name a step's own arguments begin with is left alone, so no decoy
// stands where a step reads or writes.
func (sb *sandbox) plantHarnessDecoys(v *vector, c *vcase) error {
	var taken []string
	for _, st := range c.Steps {
		x := st.args
		for _, p := range []string{x.Package, x.Source, x.Data, x.Out, x.To, x.InputsJSON} {
			if first, _, _ := strings.Cut(path.Clean(p), "/"); p != "" {
				taken = append(taken, first)
			}
		}
	}
	free := func(name string) bool {
		first, _, _ := strings.Cut(name, "/")
		return !slices.Contains(taken, first) && !strings.HasPrefix(first, reachPrefix)
	}
	dirs := []struct{ dir, token string }{
		{sb.cwd, sentinel + "CWD"},
		{sb.home, sentinel + "HOME"},
		{sb.tmp, sentinel + "TMPDECOY"},
	}
	plant := func(name string, careless bool) error {
		for _, d := range dirs {
			p := filepath.Join(d.dir, filepath.FromSlash(name))
			if _, err := os.Lstat(p); err == nil {
				continue
			}
			content := d.token + "\n"
			if ext := path.Ext(name); ext == ".json" || ext == ".yaml" || ext == ".yml" {
				content = strconv.Quote(d.token) + "\n"
			}
			err := os.MkdirAll(filepath.Dir(p), 0o755)
			if err == nil {
				err = os.WriteFile(p, []byte(content), 0o644)
			}
			// A name of a package's may be one this file system cannot
			// hold, or collide with another as a path, a file where the
			// other needs a directory; such a name has no decoy. A
			// careless file always has one.
			if err != nil && careless {
				return fmt.Errorf("the decoy %s: %w", name, err)
			}
		}
		return nil
	}
	for _, n := range carelessFiles {
		if err := plant(n, true); err != nil {
			return err
		}
	}
	for _, n := range decoyNames(v, c) {
		if !free(n) {
			continue
		}
		if err := plant(n, false); err != nil {
			return err
		}
	}
	return nil
}

// Where a package refers to a name: a partial, an asset or an embed in a
// tag, a manifest field naming a file, and an input's name.
var (
	tagRef      = regexp.MustCompile(`\{%-?\s*(?:render|include|asset|embed)\s+["']([^"'\n]+)["']`)
	manifestRef = regexp.MustCompile(`(?m)^\s*-?\s*(?:template|schema|path):\s*["']?([^"'\s#]+)`)
	inputName   = regexp.MustCompile(`(?m)^\s*-\s*name:\s*["']?([A-Za-z0-9_.-]+)`)
)

// decoyNames are the names case c's packages refer to (ADR-009 D16 item
// 1): every entry, every name a tag or the manifest refers to, as given
// and below templates/ and assets/, and <input>.json and <input>.yaml for
// every input, in order. A package is read as the vector holds it; one
// an earlier step writes has no decoys of its own. Only a name that is a
// plain relative path, and stays one on every platform, has a decoy.
func decoyNames(v *vector, c *vcase) []string {
	seen := map[string]bool{}
	add := func(n string) {
		if n == path.Clean(n) && fs.ValidPath(n) && n != "." && len(n) < 256 && utf8.ValidString(n) &&
			!strings.ContainsAny(n, "\\:*?\"<>|{}\x00") {
			seen[n] = true
		}
	}
	refs := func(name string, b []byte) {
		if len(b) > 1<<20 || !utf8.Valid(b) {
			return
		}
		for _, m := range tagRef.FindAllSubmatch(b, -1) {
			ref := string(m[1])
			add(ref)
			add("templates/" + ref)
			add("assets/" + ref)
		}
		if name != "metais.yaml" {
			return
		}
		for _, m := range manifestRef.FindAllSubmatch(b, -1) {
			ref := string(m[1])
			add(ref)
			add("templates/" + ref)
		}
		for _, m := range inputName.FindAllSubmatch(b, -1) {
			add(string(m[1]) + ".json")
			add(string(m[1]) + ".yaml")
		}
	}
	for _, op := range caseOperands(c) {
		for _, e := range packageEntries(filepath.Join(v.dir, filepath.FromSlash(op))) {
			add(e.name)
			refs(e.name, e.content)
		}
	}
	return sortedKeys(seen)
}

// caseOperands are the package and source operands c's steps name, each
// once, in order.
func caseOperands(c *vcase) []string {
	var ops []string
	for _, st := range c.Steps {
		for _, p := range []string{st.args.Package, st.args.Source} {
			if p != "" && p != "-" && !slices.Contains(ops, p) {
				ops = append(ops, p)
			}
		}
	}
	return ops
}

// pkgEntry is one entry of a package as the harness reads it.
type pkgEntry struct {
	name    string
	content []byte // nil when not read
}

// packageEntries reads the package at p as the harness sees it, without
// the tool: a directory's regular files below it, or a zip's central
// directory, each entry inflated when it can be. A package the harness
// cannot read has no entries, and a hostile package's entries are what
// its central directory claims. The entries are in byte order.
func packageEntries(p string) []pkgEntry {
	fi, err := os.Lstat(p)
	if err != nil {
		return nil
	}
	var out []pkgEntry
	if fi.IsDir() {
		filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
			if err != nil || !d.Type().IsRegular() {
				return nil
			}
			rel, err := filepath.Rel(p, q)
			if err != nil {
				return nil
			}
			e := pkgEntry{name: filepath.ToSlash(rel)}
			if info, err := d.Info(); err == nil && info.Size() <= 1<<20 {
				e.content, _ = os.ReadFile(q)
			}
			out = append(out, e)
			return nil
		})
	} else if fi.Mode().IsRegular() && fi.Size() <= 64<<20 {
		b, err := os.ReadFile(p)
		if err != nil || !isZip(b) {
			return nil
		}
		z, err := parseZip(b)
		if err != nil {
			return nil
		}
		for _, h := range z.headers {
			e := pkgEntry{name: h.name}
			if h.usize <= 1<<20 {
				e.content, _ = h.inflate(b)
			}
			out = append(out, e)
		}
	}
	slices.SortFunc(out, func(a, b pkgEntry) int { return strings.Compare(a.name, b.name) })
	return out
}

// reachTarget is one package the case names, and the step whose
// arguments the harness's commands take for it.
type reachTarget struct {
	operand string
	supply  *step // the first step that supplies inputs to it, or the first naming it
}

// reachTargets are the packages case c names that exist in the work
// directory once its steps have run.
func reachTargets(sb *sandbox, c *vcase) []reachTarget {
	var ts []reachTarget
	for _, op := range caseOperands(c) {
		if _, err := os.Lstat(filepath.Join(sb.work, filepath.FromSlash(op))); err != nil {
			continue
		}
		var first, supplying *step
		for i := range c.Steps {
			st := &c.Steps[i]
			x := st.args
			if x.Package != op && x.Source != op {
				continue
			}
			if first == nil {
				first = st
			}
			if supplying == nil && (x.Data != "" || len(x.Set) > 0 || len(x.SetJSON) > 0 || x.InputsJSON != "") {
				supplying = st
			}
		}
		if supplying == nil {
			supplying = first
		}
		ts = append(ts, reachTarget{operand: op, supply: supplying})
	}
	return ts
}

// claim reports whether the case owner runs the reach commands that key
// names. The first case to ask for a key runs them, and every run of that
// case, in either environment, asks as the same owner, so its
// environments run the same commands; every other case leaves them out,
// since they would give the same bytes again.
func (r *runner) claim(key, owner string) bool {
	got, _ := r.reached.LoadOrStore(key, owner)
	return got == owner
}

// treeDigest is the hexadecimal SHA-256 of what is at p: a directory's
// every entry below it, by path, with its type, its link target and its
// content; a file's content; or nothing.
func treeDigest(p string) string {
	h := sha256.New()
	filepath.WalkDir(p, func(q string, d fs.DirEntry, err error) error {
		if err != nil {
			fmt.Fprintf(h, "error\x00%s\x00", err)
			return nil
		}
		rel, _ := filepath.Rel(p, q)
		fmt.Fprintf(h, "%s\x00%v\x00", filepath.ToSlash(rel), d.Type())
		switch {
		case d.Type()&fs.ModeSymlink != 0:
			target, _ := os.Readlink(q)
			fmt.Fprintf(h, "%s\x00", target)
		case d.Type().IsRegular():
			if f, err := os.Open(q); err == nil {
				io.Copy(h, f)
				f.Close()
			}
			h.Write([]byte{0})
		}
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))
}

// reachCommand is one command line the harness runs over a package.
type reachCommand struct {
	args    []string
	stdin   []byte
	json    bool
	stream  bool   // standard output is render's archive
	extract string // the work-directory path extract writes into, or ""
	twin    int    // the index of the other form's command
}

// The usage codes that say the harness's own command line is wrong for
// the tool, not that the package or its arguments are refused.
var driftCodes = []string{
	"usage.missing_command", "usage.missing_flag", "usage.missing_operand", "usage.unexpected_operand",
	"usage.unknown_command", "usage.unknown_flag", "usage.flag_excluded", "usage.flag_requires",
	"usage.repeated_flag", "usage.conflicting_flags", "usage.bad_glob",
}

// reachCommands are the command lines ADR-009 D16 item 1 runs over the
// package of t, the target with index i: ls --recursive, read of every
// entry, outline of every Markdown entry, grep over every entry, schema,
// extract, and render with --dry-run and to standard output, each in JSON
// and in human form. Every command takes the limits of the step t's
// arguments come from, and render its inputs and render time.
//
// Reading and rendering are each run once for what they depend on, by
// the first case to claim it: reading takes the package and the limits,
// rendering the inputs and the render time as well (claim).
func (r *runner) reachCommands(sb *sandbox, t reachTarget, i int, owner string) ([]reachCommand, error) {
	x := t.supply.args
	pkg := sb.path(t.operand)
	var limits []string
	for _, name := range sortedKeys(x.Limits) {
		limits = append(limits, "--limit", name+"="+strings.Trim(string(x.Limits[name]), `"`))
	}
	var supply []string
	var stdin []byte
	if x.Data != "" {
		supply = append(supply, "--data", sb.path(x.Data))
	}
	for _, s := range x.Set {
		supply = append(supply, "--set", s)
	}
	for _, s := range x.SetJSON {
		supply = append(supply, "--set-json", s)
	}
	if x.InputsJSON != "" {
		b, err := os.ReadFile(filepath.Join(sb.work, filepath.FromSlash(x.InputsJSON)))
		if err != nil {
			return nil, fmt.Errorf("inputs_json: %w", err)
		}
		supply, stdin = append(supply, "--inputs-json", "-"), b
	}
	if x.Now != "" {
		supply = append(supply, "--now", x.Now)
	}
	type line struct {
		args    []string
		stdin   []byte
		stream  bool
		extract string
	}
	op := filepath.Join(sb.work, filepath.FromSlash(t.operand))
	bounds, _ := json.Marshal(t.supply.Bounds)
	key := treeDigest(op) + "\x00" + strings.Join(limits, "\x00") + "\x00" + string(bounds)
	var lines []line
	if r.claim("read\x00"+key, owner) {
		lines = append(lines, line{args: []string{"ls", pkg, "--recursive"}})
		for _, e := range packageEntries(op) {
			if e.name == "" || strings.HasPrefix(e.name, "-") || strings.ContainsRune(e.name, 0) {
				continue // no command line can name it as an operand
			}
			lines = append(lines, line{args: []string{"read", pkg, e.name}})
			if ext := strings.ToLower(path.Ext(e.name)); ext == ".md" || ext == ".markdown" {
				lines = append(lines, line{args: []string{"outline", pkg, e.name}})
			}
		}
		lines = append(lines,
			line{args: []string{"grep", pkg, ".", "**", ".compiled/**"}},
			line{args: []string{"schema", pkg}},
			line{args: []string{"extract", pkg, "**", "--out"}, extract: "extract"},
		)
	}
	data := ""
	if x.Data != "" {
		data = treeDigest(filepath.Join(sb.work, filepath.FromSlash(x.Data)))
	}
	if r.claim("render\x00"+key+"\x00"+data+"\x00"+digest(stdin)+"\x00"+strings.Join(supply, "\x00"), owner) {
		lines = append(lines,
			line{args: append([]string{"render", pkg, "--out", sb.path(fmt.Sprintf("%s%d-out", reachPrefix, i)), "--dry-run"}, supply...), stdin: stdin},
			line{args: append([]string{"render", pkg, "--out", "-"}, supply...), stdin: stdin, stream: true},
		)
	}
	var cmds []reachCommand
	for _, l := range lines {
		for _, asJSON := range []bool{true, false} {
			form := "human"
			args := slices.Clone(l.args)
			if asJSON {
				form = "json"
				args = slices.Insert(args, 1, "--json")
			}
			c := reachCommand{args: append(args, limits...), stdin: l.stdin, json: asJSON, stream: l.stream}
			if l.extract != "" {
				c.extract = fmt.Sprintf("%s%d-%s-%s", reachPrefix, i, l.extract, form)
				c.args = slices.Insert(c.args, len(args), sb.path(c.extract))
			}
			c.twin = len(cmds) + 1
			if !asJSON {
				c.twin = len(cmds) - 1
			}
			cmds = append(cmds, c)
		}
	}
	return cmds, nil
}

// label names a reach command in a failure: the command and its
// arguments after the package, in its form.
func (c reachCommand) label() string {
	args := slices.DeleteFunc(slices.Clone(c.args[1:]), func(a string) bool { return a == "--json" })
	form := "human"
	if c.json {
		form = "JSON"
	}
	return fmt.Sprintf("%s %s, %s", c.args[0], strings.Join(args[1:], " "), form)
}

// runReach runs every reach command over each package of c in run's
// sandbox, after the case's steps, whose last snapshot is before, as one
// more step: all at once, since
// each only reads the package, and extract writes only into its own
// directory. It checks each command's process (crash, bounds, an exit
// consistent with its JSON document and the same in both forms) and the
// sandbox (check 4: only the extract directories change), and scans
// everything with finders (checks 1 and 2). The step's invocations and
// trees join run's steps, so check 3 compares them between A and B.
func (r *runner) runReach(run *envRun, v *vector, c *vcase, finders []finder, before map[string]node) {
	owner := v.ID + "\x00" + c.Name
	sb := run.sb
	si := len(c.Steps) + 1
	add := func(p problem) {
		p.step = si
		run.problems = append(run.problems, p)
	}
	a, ok := r.adapter.(*cliAdapter)
	if !ok {
		add(problem{check: checkHarness, harness: true, msg: "the SC7 harness runs only through the cli adapter"})
		return
	}
	var cmds []reachCommand
	var lims []limits
	for i, t := range reachTargets(sb, c) {
		cs, err := r.reachCommands(sb, t, i, owner)
		if err != nil {
			add(problem{check: checkHarness, harness: true, msg: err.Error()})
			return
		}
		for j := range cs {
			cs[j].twin += len(cmds)
		}
		cmds = append(cmds, cs...)
		for range cs {
			lims = append(lims, stepLimits(t.supply.Bounds, a.race(), r.wall))
		}
	}
	if len(cmds) == 0 {
		return
	}
	invs := make([]*invocation, len(cmds))
	found := make([][]problem, len(cmds))
	var wg sync.WaitGroup
	for i, c := range cmds {
		wg.Go(func() { invs[i], found[i] = a.reachInvoke(sb, c, lims[i]) })
	}
	wg.Wait()
	sr := stepRun{invs: invs, trees: map[string]*tree{}}
	var extracts []string
	for i, c := range cmds {
		for _, p := range found[i] {
			p.where = invs[i].label
			add(p)
		}
		if c.extract != "" {
			t, err := readTree(filepath.Join(sb.work, c.extract))
			if err != nil {
				add(problem{check: checkHarness, harness: true, where: invs[i].label, msg: err.Error()})
				continue
			}
			sr.trees[c.extract] = t
			extracts = append(extracts, c.extract)
		}
		// The two forms of one command exit alike (ADR-008 D13), and
		// the JSON form's document says what the exit must be.
		if tw := invs[c.twin]; c.json && invs[i].proc != nil && tw.proc != nil && tw.proc.exit != invs[i].proc.exit {
			add(problem{check: checkExit, where: tw.label,
				msg: fmt.Sprintf("the human form exits %d, and the JSON form %d", tw.proc.exit, invs[i].proc.exit)})
		}
	}
	run.steps = append(run.steps, sr)
	after, err := sb.snapshot()
	if err != nil {
		add(problem{check: checkHarness, harness: true, msg: err.Error()})
		return
	}
	for _, ch := range changesOutside(before, after, extracts, nil) {
		add(problem{check: checkSnapshot, msg: ch + " outside the directory an extract writes"})
	}
	for _, f := range finders {
		for _, p := range scanWritten(f, sb, before, after, extracts) {
			add(p)
		}
		for i, c := range cmds {
			var trees map[string]*tree
			if c.extract != "" {
				trees = map[string]*tree{c.extract: sr.trees[c.extract]}
			}
			archive := c.stream && invs[i].proc != nil && invs[i].proc.exit == 0
			for _, p := range scanInvocation(f, invs[i], archive, trees) {
				p.where = invs[i].label
				add(p)
			}
		}
	}
}

// reachInvoke runs one reach command and reads what it gave.
func (a *cliAdapter) reachInvoke(sb *sandbox, c reachCommand, lim limits) (*invocation, []problem) {
	inv := &invocation{label: "reach: " + c.label(), args: c.args}
	path, args, err := sb.limited(a.path, c.args)
	if err != nil {
		return inv, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	p, err := runProc(path, args, sb.cwd, sb.environ(), c.stdin, lim.wall)
	if err != nil {
		return inv, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	inv.proc, inv.stdout, inv.stderr = p, p.stdout, p.stderr
	ps := boundProblems(p, lim)
	if p.killed {
		return inv, ps
	}
	if h := crashed(p.stderr); h != "" {
		return inv, append(ps, crashProblem(h))
	}
	if backstopped(p) {
		return inv, append(ps, backstopProblem())
	}
	doc := p.stdout
	if c.stream {
		if len(p.stdout) > 0 {
			inv.stream = p.stdout
		}
		inv.stdout, doc = nil, p.stderr
	}
	if !c.json {
		return inv, ps
	}
	env, err := parseEnvelope(doc)
	if err != nil {
		return inv, append(ps, problem{check: checkHarness, harness: true, msg: "the tool's JSON document: " + err.Error()})
	}
	inv.ok, inv.findings = env.ok, env.findings
	for _, f := range env.findings {
		m, _ := f.(map[string]any)
		if code, _ := m["code"].(string); slices.Contains(driftCodes, code) {
			return inv, append(ps, problem{check: checkHarness, harness: true,
				msg: fmt.Sprintf("%s: the harness's command line %q has drifted from the tool's", code, strings.Join(c.args, " "))})
		}
	}
	want, err := exitFor(a.reg, inv.ok, inv.findings, false, false)
	switch {
	case err != nil:
		ps = append(ps, problem{check: checkExit, msg: err.Error()})
	case p.exit != want:
		ps = append(ps, problem{check: checkExit,
			msg: fmt.Sprintf("exit %d is inconsistent with ok %t and the findings' classes, which give %d (ADR-008 D13)", p.exit, inv.ok, want)})
	}
	return inv, ps
}
