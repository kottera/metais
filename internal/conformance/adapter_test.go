// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// toolID is the implementation's identity: ADR-002 D8's tool object.
type toolID struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// invocation is one process an adapter ran for a step, and what it gave.
type invocation struct {
	label    string // the command, or the driver's op
	stdout   []byte
	stderr   []byte
	stream   []byte // the stream, or nil when the step streams nothing
	listing  any    // the dry-run listing, or nil
	ok       bool
	findings []any
	proc     *proc
}

// adapter drives an implementation (ADR-011 D11 items 5 and 6).
type adapter interface {
	// name is the adapter's flag value.
	name() string
	// run performs step st of v in sb, one process per invocation.
	run(r *runner, sb *sandbox, v *vector, st *step) ([]*invocation, []problem)
	// identity asks the implementation for its tool object, running it
	// in a plain sandbox below base.
	identity(base string) (toolID, error)
	// race reports whether the program the adapter runs is race-built.
	race() bool
	// excluded is a work-directory path the nothing-else-changes check
	// skips for st, or "".
	excluded(st *step) string
}

// plainSandbox makes a sandbox of environment B below base, which the
// identity and the opening set are asked in.
func plainSandbox(base string) (*sandbox, error) {
	if err := os.MkdirAll(base, 0o755); err != nil {
		return nil, err
	}
	return newSandbox(base, envB)
}

// ---- The cli adapter ----

// cliAdapter drives a metais-compatible command line (ADR-011 D11 item 5).
type cliAdapter struct {
	path string
	reg  map[string]codeInfo

	raceOnce sync.Once
	isRace   bool

	openOnce sync.Once
	openSet  []string
	openErr  error
}

func (a *cliAdapter) name() string          { return "cli" }
func (a *cliAdapter) excluded(*step) string { return "" }

func (a *cliAdapter) race() bool {
	a.raceOnce.Do(func() { a.isRace = raceBuilt(a.path) })
	return a.isRace
}

// command is one command line for a step, and the role ADR-006 D12 item 2
// gives it.
type command struct {
	args      []string
	stdin     []byte
	streaming bool
	authoring bool
}

// openingCommands are the command lines the adapter gives each
// package-opening command for a hostile open step (ADR-011 D16 item 3).
var openingCommands = map[string][]string{
	"inspect":  {"inspect", "--json", "{package}"},
	"ls":       {"ls", "--json", "--recursive", "{package}"},
	"read":     {"read", "--json", "{package}", "metais.yaml"},
	"outline":  {"outline", "--json", "{package}", "metais.yaml"},
	"grep":     {"grep", "--json", "{package}", "metais"},
	"schema":   {"schema", "--json", "{package}"},
	"extract":  {"extract", "--json", "{package}", "**", "--to", "extracted"},
	"check":    {"check", "--json", "{package}"},
	"validate": {"validate", "--json", "{package}"},
	"render":   {"render", "--json", "{package}", "--out", "out"},
	"unpack":   {"unpack", "--json", "{package}", "--to", "unpacked"},
}

// openPlan returns the command lines a hostile open step runs: one for
// every command the tool declares as opening a package. A declared
// command the adapter has no command line for is an error, so the step
// cannot quietly skip it.
func openPlan(set []string, pkg string) ([]command, error) {
	if len(set) == 0 {
		return nil, fmt.Errorf("the tool declares no command that opens a package, so a hostile open step cannot run")
	}
	var plan []command
	for _, name := range set {
		tmpl, ok := openingCommands[name]
		if !ok {
			return nil, fmt.Errorf("the tool declares %q as opening a package, and the cli adapter has no command line for it", name)
		}
		args := slices.Clone(tmpl)
		for i, a := range args {
			if a == "{package}" {
				args[i] = pkg
			}
		}
		plan = append(plan, command{args: args, authoring: name == "unpack" || name == "check"})
	}
	return plan, nil
}

// planCovers reports the opening commands of set that plan does not run.
func planCovers(set []string, plan []command) []string {
	var missing []string
	for _, name := range set {
		if !slices.ContainsFunc(plan, func(c command) bool { return len(c.args) > 0 && c.args[0] == name }) {
			missing = append(missing, name)
		}
	}
	return missing
}

// commands maps st to command lines (ADR-011 D11 item 5's table). An open
// step of a security vector runs every package-opening command.
func (a *cliAdapter) commands(v *vector, sb *sandbox, st *step) ([]command, error) {
	x := st.args
	supply := func(args []string) ([]string, []byte, error) {
		if x.Data != "" {
			args = append(args, "--data", x.Data)
		}
		for _, s := range x.Set {
			args = append(args, "--set", s)
		}
		for _, s := range x.SetJSON {
			args = append(args, "--set-json", s)
		}
		var stdin []byte
		if x.InputsJSON != "" {
			b, err := os.ReadFile(filepath.Join(sb.work, filepath.FromSlash(x.InputsJSON)))
			if err != nil {
				return nil, nil, fmt.Errorf("inputs_json: %w", err)
			}
			args, stdin = append(args, "--inputs-json", "-"), b
		}
		return args, stdin, nil
	}
	limitFlags := func(args []string) []string {
		for _, name := range sortedKeys(x.Limits) {
			val := strings.Trim(string(x.Limits[name]), `"`)
			args = append(args, "--limit", name+"="+val)
		}
		return args
	}
	sampleFlags := func(args []string, optional, seed, items json.RawMessage) []string {
		for _, f := range []struct {
			flag string
			v    json.RawMessage
		}{{"--optional", optional}, {"--seed", seed}, {"--items", items}} {
			if len(f.v) > 0 && string(f.v) != "null" {
				args = append(args, f.flag, strings.Trim(string(f.v), `"`))
			}
		}
		return args
	}
	switch st.Op {
	case "open":
		if slices.Contains(v.Tags, "security") {
			set, err := a.opening(filepath.Join(filepath.Dir(sb.root), "opening"))
			if err != nil {
				return nil, err
			}
			plan, err := openPlan(set, x.Package)
			for i := range plan {
				plan[i].args = limitFlags(plan[i].args)
			}
			return plan, err
		}
		return []command{{args: limitFlags([]string{"inspect", "--json", x.Package})}}, nil
	case "validate":
		args, stdin, err := supply([]string{"validate", "--json", x.Package})
		if err != nil {
			return nil, err
		}
		return []command{{args: limitFlags(args), stdin: stdin}}, nil
	case "render":
		args := []string{"render", "--json", x.Package}
		archive, only, streaming := x.stream()
		switch {
		case archive:
			args = append(args, "--out", "-")
		case streaming:
			args = append(args, "--out", "-", "--only", only)
		default:
			args = append(args, "--out", x.Out)
		}
		for _, o := range x.Only {
			args = append(args, "--only", o)
		}
		if x.Overwrite {
			args = append(args, "--overwrite")
		}
		if x.DryRun {
			args = append(args, "--dry-run")
		}
		if x.Now != "" {
			args = append(args, "--now", x.Now)
		}
		if x.Sample != nil {
			args = sampleFlags(append(args, "--sample"), x.Sample.Optional, x.Sample.Seed, x.Sample.Items)
		}
		args, stdin, err := supply(args)
		if err != nil {
			return nil, err
		}
		return []command{{args: limitFlags(args), stdin: stdin, streaming: streaming, authoring: x.Sample != nil}}, nil
	case "check":
		return []command{{args: limitFlags([]string{"check", "--json", x.Source}), authoring: true}}, nil
	case "test":
		args := []string{"test", "--json", x.Source}
		for _, e := range x.Examples {
			args = append(args, "--example", e)
		}
		return []command{{args: limitFlags(args), authoring: true}}, nil
	case "pack":
		return []command{{args: []string{"pack", "--json", x.Source, "--out", x.To}, authoring: true}}, nil
	case "unpack":
		return []command{{args: []string{"unpack", "--json", x.Package, "--to", x.To}, authoring: true}}, nil
	case "sample":
		args := []string{"sample", "--json", x.Source}
		if x.Example != "" {
			args = append(args, "--example", x.Example)
		} else {
			args = append(args, "--to", x.To)
		}
		args = sampleFlags(args, x.Optional, x.Seed, x.Items)
		args, stdin, err := supply(args)
		if err != nil {
			return nil, err
		}
		return []command{{args: args, stdin: stdin, authoring: true}}, nil
	}
	return nil, fmt.Errorf("the cli adapter has no command line for op %q", st.Op)
}

// opening reads the set of package-opening commands from the tool's own
// command declarations, through help --json, once, in a plain sandbox
// below base.
func (a *cliAdapter) opening(base string) ([]string, error) {
	a.openOnce.Do(func() { a.openSet, a.openErr = a.readOpening(base) })
	return a.openSet, a.openErr
}

// readOpening runs help --json in a plain sandbox below base and returns
// every command whose declaration marks it as opening a package, in the
// tool's order.
func (a *cliAdapter) readOpening(base string) ([]string, error) {
	sb, err := plainSandbox(base)
	if err != nil {
		return nil, err
	}
	p, err := runProc(a.path, []string{"help", "--json"}, sb.work, sb.environ(), nil, defaultWall)
	if err != nil {
		return nil, err
	}
	if p.exit != 0 {
		return nil, fmt.Errorf("help --json exited %d", p.exit)
	}
	var doc struct {
		Result struct {
			Commands []struct {
				Name         string `json:"name"`
				OpensPackage bool   `json:"opens_package"`
			} `json:"commands"`
		} `json:"result"`
	}
	if err := json.Unmarshal(p.stdout, &doc); err != nil {
		return nil, fmt.Errorf("help --json: %w", err)
	}
	var set []string
	for _, c := range doc.Result.Commands {
		if c.OpensPackage {
			set = append(set, c.Name)
		}
	}
	return set, nil
}

func (a *cliAdapter) identity(base string) (toolID, error) {
	sb, err := plainSandbox(base)
	if err != nil {
		return toolID{}, err
	}
	p, err := runProc(a.path, []string{"version", "--json"}, sb.work, sb.environ(), nil, defaultWall)
	if err != nil {
		return toolID{}, err
	}
	return readIdentity(p)
}

// readIdentity reads the tool object from what version --json printed.
func readIdentity(p *proc) (toolID, error) {
	if p.exit != 0 {
		return toolID{}, fmt.Errorf("version --json exited %d", p.exit)
	}
	var doc struct {
		Result struct {
			Tool toolID `json:"tool"`
		} `json:"result"`
	}
	if err := json.Unmarshal(p.stdout, &doc); err != nil {
		return toolID{}, fmt.Errorf("version --json: %w", err)
	}
	if doc.Result.Tool.Name == "" || doc.Result.Tool.Version == "" {
		return toolID{}, errors.New("version --json names no tool")
	}
	return doc.Result.Tool, nil
}

func (a *cliAdapter) run(r *runner, sb *sandbox, v *vector, st *step) ([]*invocation, []problem) {
	cmds, err := a.commands(v, sb, st)
	if err != nil {
		return nil, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	lim := stepLimits(st.Bounds, a.race(), r.wall)
	// A hostile open step runs every package-opening command, each of
	// which reads the package and must write nothing (ADR-011 D16 item 3),
	// so they run at once: the step takes as long as its slowest command
	// rather than their sum. Every other step runs one command.
	invs := make([]*invocation, len(cmds))
	found := make([][]problem, len(cmds))
	var wg sync.WaitGroup
	for i, c := range cmds {
		wg.Go(func() { invs[i], found[i] = a.invoke(sb, c, lim, st.args.DryRun) })
	}
	wg.Wait()
	var ps []problem
	for i, cps := range found {
		for j := range cps {
			cps[j].where = invs[i].label
		}
		ps = append(ps, cps...)
	}
	return invs, ps
}

// invoke runs one command line and reads ADR-008's envelope from it.
func (a *cliAdapter) invoke(sb *sandbox, c command, lim limits, dryRun bool) (*invocation, []problem) {
	inv := &invocation{label: c.args[0]}
	p, err := runProc(a.path, c.args, sb.work, sb.environ(), c.stdin, lim.wall)
	if err != nil {
		return inv, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	inv.proc, inv.stdout, inv.stderr = p, p.stdout, p.stderr
	ps := boundProblems(p, lim)
	if p.killed {
		return inv, ps
	}
	switch p.exit {
	case 2:
		return inv, append(ps, problem{check: checkHarness, harness: true,
			msg: fmt.Sprintf("exit 2: the tool crashed or reported an internal error\n%s", tail(p.stderr))})
	case 3:
		return inv, append(ps, problem{check: checkHarness, harness: true,
			msg: fmt.Sprintf("exit 3: the adapter's command line %q has drifted from the tool's", strings.Join(c.args, " "))})
	}
	doc := p.stdout
	if c.streaming {
		// Standard output is the stream, so an empty one is a stream of
		// no bytes.
		inv.stream, doc = append([]byte{}, p.stdout...), p.stderr
	}
	env, err := parseEnvelope(doc)
	if err != nil {
		return inv, append(ps, problem{check: checkHarness, harness: true, msg: "the tool's JSON document: " + err.Error()})
	}
	inv.ok, inv.findings = env.ok, env.findings
	if dryRun {
		inv.listing = env.files
	}
	want, err := exitFor(a.reg, inv.ok, inv.findings, c.authoring)
	switch {
	case err != nil:
		ps = append(ps, problem{check: checkExit, msg: err.Error()})
	case p.exit != want:
		ps = append(ps, problem{check: checkExit,
			msg: fmt.Sprintf("exit %d is inconsistent with ok %t and the findings' classes, which give %d (ADR-008 D13)", p.exit, inv.ok, want)})
	}
	return inv, ps
}

// envelope is what the adapter reads of ADR-008 D10's document.
type envelope struct {
	ok       bool
	findings []any
	files    any // result.files, render's listing
}

// parseEnvelope reads one JSON document carrying ok, findings and result.
func parseEnvelope(b []byte) (*envelope, error) {
	v, err := decodeJSON(b)
	if err != nil {
		return nil, err
	}
	m, isObj := v.(map[string]any)
	if !isObj {
		return nil, errors.New("not an object")
	}
	ok, isBool := m["ok"].(bool)
	findings, isArr := m["findings"].([]any)
	if !isBool || !isArr {
		return nil, errors.New("no ok and findings members")
	}
	if _, has := m["result"]; !has {
		return nil, errors.New("no result member")
	}
	e := &envelope{ok: ok, findings: findings}
	if res, _ := m["result"].(map[string]any); res != nil {
		e.files = res["files"]
	}
	return e, nil
}

// exitFor is the exit code ADR-008 D13 maps a result onto: 0 exactly when
// ok, and otherwise the first of 2, 3, 4, 1 and 5 that the error findings'
// classes give, a package's own content error counting as refused under a
// consumer command and as content under an authoring one (ADR-006 D12).
// Each code's class is the registry's.
func exitFor(reg map[string]codeInfo, ok bool, findings []any, authoring bool) (int, error) {
	if ok {
		return 0, nil
	}
	classes := map[string]bool{}
	for _, f := range findings {
		m, _ := f.(map[string]any)
		code, _ := m["code"].(string)
		info, known := reg[code]
		if !known {
			return 0, fmt.Errorf("the finding code %q is not in the registry", code)
		}
		if info.Severity != "error" {
			continue
		}
		kind := ""
		if s, _ := m["subject"].(map[string]any); s != nil {
			kind, _ = s["kind"].(string)
		}
		if info.Class == "content" && slices.Contains([]string{"package", "manifest", "schema", "template", "asset", "reference"}, kind) {
			if authoring {
				classes["content"] = true
			} else {
				classes["refused"] = true
			}
			continue
		}
		classes[info.Class] = true
	}
	for _, c := range []struct {
		class string
		code  int
	}{{"internal", 2}, {"usage", 3}, {"refused", 4}, {"content", 1}, {"destination", 5}} {
		if classes[c.class] {
			return c.code, nil
		}
	}
	return 0, errors.New("ok is false, but no error finding explains it")
}

// ---- The exec adapter ----

// driverProtocol is the protocol name a driver request carries.
const driverProtocol = "metais.conformance-driver/1"

// streamFile is where a driver writes stream bytes, below the work
// directory.
const streamFile = ".stream"

// driverRequest is one request of ADR-011 D11 item 6.
type driverRequest struct {
	Protocol   string          `json:"protocol"`
	Op         string          `json:"op"`
	Args       json.RawMessage `json:"args"`
	StreamFile string          `json:"stream_file,omitempty"`
}

// driverResponse is a driver's answer.
type driverResponse struct {
	OK       *bool   `json:"ok"`
	Findings []any   `json:"findings"`
	Listing  any     `json:"listing"`
	Tool     *toolID `json:"tool"`
}

// execAdapter drives an implementation through the driver protocol.
type execAdapter struct {
	path     string
	validate func([]byte) error // the driver-response schema

	raceOnce sync.Once
	isRace   bool
}

func (a *execAdapter) name() string { return "exec" }

// excluded is the stream file on a streaming step: the driver writes it,
// and the runner reads and removes it. On any other step it is a file
// like another.
func (a *execAdapter) excluded(st *step) string {
	if _, _, streaming := st.args.stream(); streaming {
		return streamFile
	}
	return ""
}

func (a *execAdapter) race() bool {
	a.raceOnce.Do(func() { a.isRace = raceBuilt(a.path) })
	return a.isRace
}

// call sends one request to a fresh driver process in sb and reads its
// response.
func (a *execAdapter) call(sb *sandbox, req driverRequest, lim limits) (*proc, *driverResponse, []problem) {
	body, err := json.Marshal(req)
	if err != nil {
		return nil, nil, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	p, err := runProc(a.path, nil, sb.work, sb.environ(), body, lim.wall)
	if err != nil {
		return nil, nil, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
	}
	ps := boundProblems(p, lim)
	if p.killed {
		return p, nil, ps
	}
	if p.exit != 0 {
		return p, nil, append(ps, problem{check: checkDriver, harness: true,
			msg: fmt.Sprintf("the driver exited %d\n%s", p.exit, tail(p.stderr))})
	}
	var resp driverResponse
	if err := a.validate(p.stdout); err != nil {
		return p, nil, append(ps, problem{check: checkDriver, harness: true, msg: "the driver's response is malformed: " + err.Error()})
	}
	dec := json.NewDecoder(bytes.NewReader(p.stdout))
	dec.UseNumber()
	if err := dec.Decode(&resp); err != nil {
		return p, nil, append(ps, problem{check: checkDriver, harness: true, msg: "the driver's response is malformed: " + err.Error()})
	}
	return p, &resp, ps
}

func (a *execAdapter) run(r *runner, sb *sandbox, v *vector, st *step) ([]*invocation, []problem) {
	lim := stepLimits(st.Bounds, a.race(), r.wall)
	_, _, streaming := st.args.stream()
	sp := filepath.Join(sb.work, streamFile)
	if streaming {
		// A stream file an earlier step left would otherwise pass for
		// this step's stream.
		if err := os.Remove(sp); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return nil, []problem{{check: checkHarness, harness: true, msg: err.Error()}}
		}
	}
	req := driverRequest{Protocol: driverProtocol, Op: st.Op, Args: st.Args, StreamFile: streamFile}
	p, resp, ps := a.call(sb, req, lim)
	inv := &invocation{label: st.Op, proc: p}
	if p != nil {
		inv.stdout, inv.stderr = p.stdout, p.stderr
	}
	for i := range ps {
		ps[i].where = inv.label
	}
	if resp == nil {
		return []*invocation{inv}, ps
	}
	if resp.OK == nil {
		return []*invocation{inv}, append(ps, problem{check: checkDriver, harness: true, where: inv.label, msg: "the driver's response has no ok"})
	}
	inv.ok, inv.findings, inv.listing = *resp.OK, resp.Findings, resp.Listing
	if inv.findings == nil {
		inv.findings = []any{}
	}
	if streaming {
		b, err := os.ReadFile(sp)
		switch {
		case errors.Is(err, fs.ErrNotExist):
		case err != nil:
			return []*invocation{inv}, append(ps, problem{check: checkHarness, harness: true, where: inv.label, msg: err.Error()})
		default:
			inv.stream = b
			if err := os.Remove(sp); err != nil {
				return []*invocation{inv}, append(ps, problem{check: checkHarness, harness: true, where: inv.label, msg: err.Error()})
			}
		}
	}
	return []*invocation{inv}, ps
}

func (a *execAdapter) identity(base string) (toolID, error) {
	sb, err := plainSandbox(base)
	if err != nil {
		return toolID{}, err
	}
	_, resp, ps := a.call(sb, driverRequest{Protocol: driverProtocol, Op: "version", Args: json.RawMessage("{}")}, limits{cpu: defaultCPU, rss: defaultRSS, wall: defaultWall})
	if len(ps) > 0 {
		return toolID{}, errors.New(ps[0].msg)
	}
	if resp.Tool == nil || resp.Tool.Name == "" || resp.Tool.Version == "" {
		return toolID{}, errors.New("the driver's version response names no tool")
	}
	return *resp.Tool, nil
}

// tail is the end of a process's standard error, for a harness error.
func tail(b []byte) string {
	if len(b) > 2048 {
		b = b[len(b)-2048:]
	}
	return string(b)
}

// sortedKeys returns m's keys in order.
func sortedKeys[V any](m map[string]V) []string {
	return slices.Sorted(maps.Keys(m))
}
