// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/version"
)

const sgr = "\x1b["

// alwaysTerminal makes isTerminal report every stream as a terminal until
// t ends.
func alwaysTerminal(t *testing.T) {
	saved := isTerminal
	t.Cleanup(func() { isTerminal = saved })
	isTerminal = func(any) bool { return true }
}

// TestColourPolicy is ADR-008 D12 item 1 over the human refusal, whose
// severity word is what may be coloured: auto colours standard error when
// it is a terminal and NO_COLOR is unset or empty; always colours whatever
// NO_COLOR says; never colours nothing. Standard output's being a terminal
// does not colour standard error.
func TestColourPolicy(t *testing.T) {
	for _, c := range []struct {
		colour      string
		noColor     bool
		stderrIsTTY bool
		stdoutIsTTY bool
		want        bool
	}{
		{"auto", false, true, false, true},
		{"auto", true, true, false, false},
		{"auto", false, false, false, false},
		{"auto", false, false, true, false},
		{"always", true, false, false, true},
		{"always", false, false, false, true},
		{"never", false, true, true, false},
		{"never", true, true, true, false},
	} {
		var stdout, stderr bytes.Buffer
		saved := isTerminal
		isTerminal = func(s any) bool {
			return (s == any(&stderr) && c.stderrIsTTY) || (s == any(&stdout) && c.stdoutIsTTY)
		}
		status := Run(t.Context(), Invocation{
			Args: []string{"version", "--bogus", "--color", c.colour}, Stdin: failingReader{t}, Stdout: &stdout, Stderr: &stderr,
			Options: Options{Display: hostenv.Display{NoColor: c.noColor}},
		})
		isTerminal = saved
		if status != finding.ExitUsage {
			t.Fatalf("status %d", status)
		}
		coloured := strings.Contains(stderr.String(), sgr+"31merror"+sgr+"0m")
		if coloured != c.want {
			t.Errorf("--color %s, NO_COLOR %v, stderr a terminal %v, stdout %v: coloured %v, want %v\n%q", c.colour, c.noColor, c.stderrIsTTY, c.stdoutIsTTY, coloured, c.want, stderr.String())
		}
		if strings.Count(stderr.String(), sgr) > 2 {
			t.Errorf("more than the severity word is coloured: %q", stderr.String())
		}
	}
}

// TestNeverColoured is ADR-008 D12 item 3: a JSON document and help are
// never coloured, and nothing this wave's commands print is, whatever
// --color says and whatever the terminal.
func TestNeverColoured(t *testing.T) {
	alwaysTerminal(t)
	for _, argv := range [][]string{
		{"version", "--bogus", "--json"}, {"bogus", "--json"}, {"--json"}, {"help", "nope", "--json"},
		{"version", "--json"}, {"limits", "--json"}, {"help", "--json"},
		{"help"}, {"--help"}, {"version", "--help"}, {"help", "globs"},
		{"version"}, {"limits"},
	} {
		for _, colour := range []string{"always", "auto"} {
			r := runWith(t, hostenv.Display{}, append(slices.Clone(argv), "--color", colour)...)
			if strings.Contains(r.stdout, sgr) {
				t.Errorf("metais %q --color %s coloured standard output", argv, colour)
			}
		}
	}
	// The control: the same terminal colours a human refusal.
	if r := runWith(t, hostenv.Display{}, "bogus", "--color", "auto"); !strings.Contains(r.stderr, sgr) {
		t.Error("a human refusal on a terminal is not coloured: the policy above proves nothing")
	}
}

// TestPresentationNeverReachesJSON is ADR-008 D12 item 6 in-process: the
// documents and their exit statuses are the same bytes whatever NO_COLOR,
// --color and the terminal say. The pseudo-terminal run of D16 item 6 is
// Wave 6's.
func TestPresentationNeverReachesJSON(t *testing.T) {
	for _, argv := range [][]string{{"version", "--json"}, {"limits", "--json"}, {"help", "--json"}, {"version", "--bogus", "--json"}, {"--json"}} {
		// --color is given, as a caller would, before the line's own
		// arguments, so every variant refuses the same arguments at the
		// same positions.
		base := runTool(t, append([]string{"--color", "never"}, argv...)...)
		for _, noColor := range []bool{false, true} {
			for _, tty := range []bool{false, true} {
				saved := isTerminal
				isTerminal = func(any) bool { return tty }
				for _, colour := range []string{"auto", "always", "never"} {
					r := runWith(t, hostenv.Display{NoColor: noColor}, append([]string{"--color", colour}, argv...)...)
					if r.stdout != base.stdout || r.status != base.status {
						t.Errorf("metais %q differs under NO_COLOR %v, a terminal %v, --color %s", argv, noColor, tty, colour)
					}
				}
				isTerminal = saved
			}
		}
	}
}

// TestVersion is W2-1.1: metais version --json validates against
// metais.version/1 (runTool checks every document) and carries ADR-002 D8
// item 1's content, from internal/version.
func TestVersion(t *testing.T) {
	version.Override(t, "1.2.3-rc.1+build.5")
	r := runTool(t, "version", "--json")
	var res versionResult
	if err := json.Unmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	want := versionResult{toolVersion{"metais", "1.2.3-rc.1+build.5"}, specificationVersion{version.Implements()}}
	if !reflect.DeepEqual(res, want) {
		t.Errorf("version --json result %+v, want %+v", res, want)
	}
	if !r.doc.OK || len(r.doc.Findings) != 0 || len(r.doc.Next) != 0 {
		t.Errorf("version --json: %+v", r.doc)
	}
	h := runTool(t, "version")
	if h.stdout != "metais 1.2.3-rc.1+build.5\nspecification 1.0\n" {
		t.Errorf("metais version printed %q", h.stdout)
	}
	// The document's bytes, member for member.
	const doc = `{"format":"metais.version/1","ok":true,"findings":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}},"result":{"tool":{"name":"metais","version":"1.2.3-rc.1+build.5"},"specification":{"implements":["1.0"]}},"next":[]}` + "\n"
	if r.stdout != doc {
		t.Errorf("version --json is\n%s\nwant\n%s", r.stdout, doc)
	}
}

// TestLimits is W2-1.2: metais limits --json lists ADR-009 D3's table
// exactly, in its order, each limit with its name, default, unit and
// class taken from internal/limits, and the table is read from the ADR's
// own text, not from a copy of it.
func TestLimits(t *testing.T) {
	r := runTool(t, "limits", "--json")
	var res limitsResult
	if err := json.Unmarshal(r.doc.Result, &res); err != nil {
		t.Fatal(err)
	}
	var fromPackage []limitRow
	for _, l := range limits.All() {
		fromPackage = append(fromPackage, limitRow{l.Name, l.Default, l.Unit.String(), l.Class.String()})
	}
	if !slices.Equal(res.Limits, fromPackage) {
		t.Errorf("limits --json differs from internal/limits:\n%v\n%v", res.Limits, fromPackage)
	}
	adr := adr009Table(t)
	if len(adr) != len(res.Limits) {
		t.Fatalf("limits --json lists %d limits, ADR-009 D3 %d", len(res.Limits), len(adr))
	}
	for i, want := range adr {
		got := res.Limits[i]
		if got.Name != want.Name || got.Default != want.Default || got.Class != want.Class || (want.Unit == "bytes") != (got.Unit == "bytes") {
			t.Errorf("limit %d is %+v, ADR-009 D3 has %+v", i, got, want)
		}
	}
	// The human table holds the same rows, a size in its largest whole unit.
	h := runTool(t, "limits").stdout
	lines := strings.Split(strings.TrimSuffix(h, "\n"), "\n")
	if len(lines) != len(adr)+1 || strings.Fields(lines[0])[0] != "NAME" {
		t.Fatalf("metais limits printed %d lines:\n%s", len(lines), h)
	}
	for i, l := range limits.All() {
		f := strings.Fields(lines[i+1])
		if len(f) != 4 || f[0] != l.Name || f[1] != humanDefault(l) || f[2] != l.Unit.String() || f[3] != l.Class.String() {
			t.Errorf("line %d is %q", i+2, lines[i+1])
		}
	}
}

func TestHumanDefault(t *testing.T) {
	for _, c := range []struct {
		l    limits.Limit
		want string
	}{
		{limits.Limit{Default: 256 << 20, Unit: limits.Bytes}, "256MiB"},
		{limits.Limit{Default: 1 << 30, Unit: limits.Bytes}, "1GiB"},
		{limits.Limit{Default: 4096, Unit: limits.Bytes}, "4KiB"},
		{limits.Limit{Default: 512, Unit: limits.Bytes}, "512"},
		{limits.Limit{Default: 1536, Unit: limits.Bytes}, "1536"},
		{limits.Limit{Default: 3072, Unit: limits.Bytes}, "3KiB"},
		{limits.Limit{Default: 0, Unit: limits.Bytes}, "0"},
		{limits.Limit{Default: 4096, Unit: limits.Count}, "4096"},
	} {
		if got := humanDefault(c.l); got != c.want {
			t.Errorf("humanDefault(%d %s) = %q, want %q", c.l.Default, c.l.Unit, got, c.want)
		}
	}
}

// adr009Table reads ADR-009 D3's three tables from the ADR: each row's
// name, its default as a number in its unit, and its class by the table
// it is in.
func adr009Table(t *testing.T) []limitRow {
	t.Helper()
	f, err := os.Open("../../docs/adr/ADR-009-the-security-model.md")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	row := regexp.MustCompile("^\\| `([a-z0-9-]+)` \\| ([^|]+) \\|")
	var out []limitRow
	class, inD3 := "", false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "### D3 "):
			inD3 = true
		case strings.HasPrefix(line, "### ") && inD3:
			return out
		case !inD3:
		case strings.HasPrefix(line, "**Package limits**"):
			class = "package"
		case strings.HasPrefix(line, "**Data limits**"):
			class = "data"
		case strings.HasPrefix(line, "**Work and output limits**"):
			class = "work"
		default:
			if m := row.FindStringSubmatch(line); m != nil {
				def, unit := parseADRDefault(t, strings.TrimSpace(m[2]))
				out = append(out, limitRow{m[1], def, unit, class})
			}
		}
	}
	t.Fatal("ADR-009 has no D3")
	return nil
}

// parseADRDefault reads a default as ADR-009 writes it: 256 MiB, 10,000,
// 2³² units, 512 bytes, 1,000,000 elements. The unit it returns is bytes
// for a size and count otherwise.
func parseADRDefault(t *testing.T, s string) (int64, string) {
	t.Helper()
	num, word, _ := strings.Cut(s, " ")
	var n int64
	switch num {
	case "2³²":
		n = 1 << 32
	case "2³³":
		n = 1 << 33
	default:
		v, err := strconv.ParseInt(strings.ReplaceAll(num, ",", ""), 10, 64)
		if err != nil {
			t.Fatalf("ADR-009 default %q: %v", s, err)
		}
		n = v
	}
	switch word {
	case "KiB":
		return n << 10, "bytes"
	case "MiB":
		return n << 20, "bytes"
	case "GiB":
		return n << 30, "bytes"
	case "bytes":
		return n, "bytes"
	}
	return n, "count"
}

// TestEnvelopeMemberOrder is W2-1.6: the envelope's members are format,
// ok, findings, summary, result and next, in that order, in the type that
// writes it and in metais.envelope.1.json, and summary is always present.
func TestEnvelopeMemberOrder(t *testing.T) {
	var tags []string
	et := reflect.TypeFor[envelope]()
	for i := range et.NumField() {
		tag := et.Field(i).Tag.Get("json")
		if strings.Contains(tag, ",") {
			t.Errorf("envelope member %s has options %q: every member is always present", et.Field(i).Name, tag)
		}
		tags = append(tags, strings.Split(tag, ",")[0])
	}
	if !slices.Equal(tags, envelopeMembers) {
		t.Errorf("envelope members %q, want %q", tags, envelopeMembers)
	}
	b, err := os.ReadFile(schemaDir + "/metais.envelope.1.json")
	if err != nil {
		t.Fatal(err)
	}
	var schema struct {
		Description string
		Required    []string
		Properties  map[string]json.RawMessage
	}
	if err := json.Unmarshal(b, &schema); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(schema.Required, envelopeMembers) || len(schema.Properties) != len(envelopeMembers) {
		t.Errorf("metais.envelope.1.json requires %q", schema.Required)
	}
	if !strings.Contains(schema.Description, "format, ok, findings, summary, result, next") {
		t.Error("metais.envelope.1.json does not state the member order")
	}
	var raw struct{ Properties json.RawMessage }
	if err := json.Unmarshal(b, &raw); err != nil {
		t.Fatal(err)
	}
	props, err := topLevelKeys(raw.Properties)
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(props, envelopeMembers) {
		t.Errorf("metais.envelope.1.json lists its properties as %q", props)
	}
	// ADR-008's implementation log records the reconciliation.
	adr, err := os.ReadFile("../../docs/adr/ADR-008-the-command-line-and-its-machine-output.md")
	if err != nil {
		t.Fatal(err)
	}
	if _, log, _ := strings.Cut(string(adr), "## Implementation log"); !strings.Contains(log, "`summary`") {
		t.Error("ADR-008's implementation log does not record the summary reconciliation")
	}
}

// TestCheckDocumentRefuses holds the document check to failing what
// ADR-008 D10 and D16 item 3 refuse: a format with no schema, members out
// of order or missing, indentation, two lines, a result of the wrong
// shape.
func TestCheckDocumentRefuses(t *testing.T) {
	good := runTool(t, "version", "--json").stdout
	if err := checkDocument([]byte(good)); err != nil {
		t.Fatal(err)
	}
	var indented bytes.Buffer
	if err := json.Indent(&indented, []byte(good), "", "  "); err != nil {
		t.Fatal(err)
	}
	for name, bad := range map[string]string{
		"a format with no schema": strings.Replace(good, "metais.version/1", "metais.nope/1", 1),
		"a format of version 2":   strings.Replace(good, "metais.version/1", "metais.version/2", 1),
		"no summary":              regexp.MustCompile(`"summary":\{[^}]*\{\}\},`).ReplaceAllString(good, ""),
		"summary last":            strings.Replace(regexp.MustCompile(`"summary":\{[^}]*\{\}\},`).ReplaceAllString(good, ""), `"next":[]}`, `"next":[],"summary":{"errors":0,"warnings":0,"info":0,"listed":0,"omitted":0,"by_code":{}}}`, 1),
		"indented":                indented.String() + "\n",
		"no final LF":             strings.TrimSuffix(good, "\n"),
		"two documents":           good + good,
		"a wrong result":          strings.Replace(good, `"name":"metais"`, `"name":"other"`, 1),
		"an extra member":         strings.Replace(good, `"next":[]}`, `"next":[],"extra":1}`, 1),
		"ok with no result":       regexp.MustCompile(`"result":\{.*\},"next"`).ReplaceAllString(good, `"result":null,"next"`),
		"refused with no next":    strings.Replace(runTool(t, "bogus", "--json").stdout, regexp.MustCompile(`"next":\[.*\]`).FindString(runTool(t, "bogus", "--json").stdout), `"next":[]`, 1),
		"a next that is no line":  strings.Replace(runTool(t, "bogus", "--json").stdout, `"run":"metais help"`, `"run":"rm -rf /"`, 1),
	} {
		if checkDocument([]byte(bad)) == nil {
			t.Errorf("%s: checkDocument accepts\n%s", name, bad)
		}
	}
}

// machineTypes are the types metais writes as JSON: the envelope and every
// result.
var machineTypes = []reflect.Type{
	reflect.TypeFor[envelope](), reflect.TypeFor[versionResult](), reflect.TypeFor[limitsResult](), reflect.TypeFor[helpResult](),
}

// mapsIn returns the path of every map type reachable from t.
func mapsIn(t reflect.Type, path string, seen map[reflect.Type]bool) []string {
	if seen[t] {
		return nil
	}
	seen[t] = true
	switch t.Kind() {
	case reflect.Map:
		return []string{path}
	case reflect.Pointer, reflect.Slice, reflect.Array:
		return mapsIn(t.Elem(), path+"[]", seen)
	case reflect.Struct:
		var out []string
		for i := range t.NumField() {
			f := t.Field(i)
			out = append(out, mapsIn(f.Type, path+"."+f.Name, seen)...)
		}
		return out
	}
	return nil
}

// TestNoMapsInMachineOutput is ADR-010 D7 item 3's tripwire: no map is
// marshalled in machine output, so member order is always the declared
// order and never map iteration's.
func TestNoMapsInMachineOutput(t *testing.T) {
	for _, mt := range machineTypes {
		if m := mapsIn(mt, mt.Name(), map[reflect.Type]bool{}); len(m) > 0 {
			t.Errorf("%s holds a map at %v", mt, m)
		}
	}
	// The control: a type holding a map, nested, is caught.
	type inner struct{ M map[string]int }
	type outer struct{ I []*inner }
	if m := mapsIn(reflect.TypeFor[outer](), "outer", map[reflect.Type]bool{}); len(m) != 1 {
		t.Errorf("the tripwire misses a nested map: %v", m)
	}
	// Every result a command returns is one of the machine types.
	for _, argv := range [][]string{{"version"}, {"limits"}, {"help"}, {"help", "version"}, {"help", "globs"}} {
		l := readLine(table, argv)
		o := l.cmd.run(&session{table: table, line: l}, l.parsed)
		if !slices.Contains(machineTypes, reflect.TypeOf(o.result).Elem()) {
			t.Errorf("metais %q returns a %T, which the tripwire does not walk", argv, o.result)
		}
	}
}

// TestHTMLNotEscaped is ADR-008 D10 item 2: <, > and & are written as
// they are.
func TestHTMLNotEscaped(t *testing.T) {
	var b bytes.Buffer
	writeJSON(&b, Next{Run: "metais help <topic>", Why: "a & b"})
	if got := b.String(); got != `{"run":"metais help <topic>","why":"a & b"}`+"\n" {
		t.Errorf("writeJSON wrote %q", got)
	}
	r := runTool(t, "help", "--json")
	if strings.Contains(r.stdout, `\u003c`) || !strings.Contains(r.stdout, "<command>") {
		t.Error("help --json escapes HTML")
	}
}

// TestRefusalNextPerCause is W5-V1: when a command declares one refusal
// code for more than one cause, a finding of that code names the next
// step declared for its own cause, so help and the emitted next agree
// (ADR-008 D14 item 4, D15 item 3). Each line triggers one declared
// cause, named by its code and its cause; every cause of every code a
// command declares more than once has a line here.
func TestRefusalNextPerCause(t *testing.T) {
	root := testenv.Dir(t)
	pkg := samplePackage(t, root)
	to := filepath.Join(root, "out")
	const (
		lineBadValue   = "A flag's value is missing, or is not one the flag accepts, or a flag that takes no value is given one."
		renderBadValue = "--now is not an RFC 3339 date-time with an offset, or --seed or --items is not a whole number in range."
		sampleBadValue = "--seed or --items is not a whole number in range, or --example is not a name an example can have."
		renderSample   = "--data is given with --sample, whose generated data takes the data directory's place."
		renderArchive  = "--overwrite is given with --out -, and an archive on standard output replaces nothing."
	)
	cases := []struct {
		line  []string
		code  finding.Code
		cause string
	}{
		{[]string{"render", pkg, "--out", to, "--now"}, finding.CodeUsageBadValue, lineBadValue},
		{[]string{"render", pkg, "--out", to, "--json=x"}, finding.CodeUsageBadValue, lineBadValue},
		{[]string{"render", pkg, "--out", to, "--sample", "--optional", "some"}, finding.CodeUsageBadValue, lineBadValue},
		{[]string{"render", pkg, "--out", to, "--now", "2026-10-19"}, finding.CodeUsageBadValue, renderBadValue},
		{[]string{"render", pkg, "--out", to, "--sample", "--seed", "x"}, finding.CodeUsageBadValue, renderBadValue},
		{[]string{"render", pkg, "--out", to, "--sample", "--items", "-1"}, finding.CodeUsageBadValue, renderBadValue},
		{[]string{"render", pkg, "--sample", "--data", to, "--out", to}, finding.CodeUsageFlagExcluded, renderSample},
		{[]string{"render", pkg, "--out", "-", "--overwrite"}, finding.CodeUsageFlagExcluded, renderArchive},
		{[]string{"render", pkg, "--out", "-", "--overwrite", "--data", to}, finding.CodeUsageFlagExcluded, renderArchive},
		{[]string{"sample", pkg, "--to", to, "--optional", "some"}, finding.CodeUsageBadValue, lineBadValue},
		{[]string{"sample", pkg, "--to", ""}, finding.CodeUsageBadValue, lineBadValue},
		{[]string{"sample", pkg, "--to", to, "--seed=+1"}, finding.CodeUsageBadValue, sampleBadValue},
		{[]string{"sample", pkg, "--to", to, "--items", "-1"}, finding.CodeUsageBadValue, sampleBadValue},
		{[]string{"sample", pkg, "--example", "Not-A-Name"}, finding.CodeUsageBadValue, sampleBadValue},
	}
	covered := map[string]bool{}
	for _, tc := range cases {
		c := lookup(table, tc.line[0])
		i := slices.IndexFunc(c.Refusals, func(r Refusal) bool { return r.Code == tc.code && r.Cause == tc.cause })
		if i < 0 {
			t.Fatalf("metais %s declares no refusal %s for %q", c.Name, tc.code, tc.cause)
		}
		covered[c.Name+" "+string(tc.code)+" "+tc.cause] = true
		r := runTool(t, slices.Insert(slices.Clone(tc.line), 1, "--json")...)
		if !slices.Equal(r.codes(), []string{string(tc.code)}) {
			t.Errorf("metais %q: codes %q; want %s", tc.line, r.codes(), tc.code)
			continue
		}
		if want := []Next{c.Refusals[i].Next}; !reflect.DeepEqual(r.doc.Next, want) {
			t.Errorf("metais %q: next %v; its cause declares %v", tc.line, r.doc.Next, want)
		}
	}
	for _, c := range table {
		for _, r := range c.Refusals {
			n := 0
			for _, o := range c.Refusals {
				if o.Code == r.Code {
					n++
				}
			}
			if n > 1 && !covered[c.Name+" "+string(r.Code)+" "+r.Cause] {
				t.Errorf("metais %s declares %s %d times, and no line here triggers its cause %q", c.Name, r.Code, n, r.Cause)
			}
		}
	}
}
