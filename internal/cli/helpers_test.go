// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/cli/args"
	"github.com/kottera/metais/internal/fault"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostenv"
	"github.com/kottera/metais/internal/specdata"
)

// schemaDir is docs/cli/schemas, from this package's directory.
const schemaDir = "../../docs/cli/schemas"

const (
	cliSchemaBase   = "https://metais.org/cli/schemas/"
	findingSchemaID = "https://metais.org/schemas/1.0/finding.schema.json"
)

// noLoader refuses every URL: every schema is given to the compiler, and
// nothing is fetched.
type noLoader struct{}

func (noLoader) Load(url string) (any, error) { return nil, errors.New("not loaded: " + url) }

var (
	schemasOnce sync.Once
	schemas     map[string]*jsonschema.Schema // by format, such as metais.version/1
	schemasErr  error
)

// loadSchemas compiles every committed schema under docs/cli/schemas, with
// the finding schema they refer to, keyed by the format each defines.
func loadSchemas() (map[string]*jsonschema.Schema, error) {
	schemasOnce.Do(func() {
		schemas, schemasErr = compileSchemas(schemaDir)
	})
	return schemas, schemasErr
}

func compileSchemas(dir string) (map[string]*jsonschema.Schema, error) {
	c := jsonschema.NewCompiler()
	c.UseLoader(noLoader{})
	add := func(id string, b []byte) error {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
		if err != nil {
			return fmt.Errorf("%s: %w", id, err)
		}
		return c.AddResource(id, doc)
	}
	if err := add(findingSchemaID, specdata.FindingSchema); err != nil {
		return nil, err
	}
	names, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		return nil, err
	}
	for _, n := range names {
		b, err := os.ReadFile(n)
		if err != nil {
			return nil, err
		}
		if err := add(cliSchemaBase+filepath.Base(n), b); err != nil {
			return nil, err
		}
	}
	out := map[string]*jsonschema.Schema{}
	for _, n := range names {
		base := filepath.Base(n)
		s, err := c.Compile(cliSchemaBase + base)
		if err != nil {
			return nil, fmt.Errorf("%s does not compile: %w", base, err)
		}
		if base == "metais.envelope.1.json" {
			continue
		}
		// metais.version.1.json defines metais.version/1.
		name, ok := strings.CutSuffix(base, ".1.json")
		if !ok {
			return nil, fmt.Errorf("%s is not named metais.<command>.1.json", base)
		}
		out[name+"/1"] = s
	}
	return out, nil
}

// envelopeMembers are the envelope's members in their declared order
// (ADR-008 D10 item 1, with summary after findings).
var envelopeMembers = []string{"format", "ok", "findings", "summary", "result", "next"}

// checkDocument holds b, one document a run printed with --json, to
// ADR-008 D10 and D16 item 3: one compact line ending in one LF, the
// envelope's members in order, and valid against the committed schema of
// its format. A format with no schema fails.
func checkDocument(b []byte) error {
	if !bytes.HasSuffix(b, []byte("\n")) || bytes.Count(b, []byte("\n")) != 1 {
		return fmt.Errorf("the document is not one line ending in one LF: %q", b)
	}
	var compact bytes.Buffer
	if err := json.Compact(&compact, b); err != nil {
		return fmt.Errorf("the document is not JSON: %w", err)
	}
	if !bytes.Equal(compact.Bytes(), bytes.TrimSuffix(b, []byte("\n"))) {
		return errors.New("the document is not compact")
	}
	keys, err := topLevelKeys(b)
	if err != nil {
		return err
	}
	if !slices.Equal(keys, envelopeMembers) {
		return fmt.Errorf("the members are %q, want %q", keys, envelopeMembers)
	}
	var head struct{ Format string }
	if err := json.Unmarshal(b, &head); err != nil {
		return err
	}
	all, err := loadSchemas()
	if err != nil {
		return err
	}
	s, ok := all[head.Format]
	if !ok {
		return fmt.Errorf("the format %q has no schema under docs/cli/schemas", head.Format)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return err
	}
	if err := s.Validate(inst); err != nil {
		return fmt.Errorf("the document does not validate against %s: %v", head.Format, err)
	}
	return nil
}

// topLevelKeys returns the member names of the JSON object b, in the
// order they are written.
func topLevelKeys(b []byte) ([]string, error) {
	dec := json.NewDecoder(bytes.NewReader(b))
	if t, err := dec.Token(); err != nil || t != json.Delim('{') {
		return nil, fmt.Errorf("the document is not an object")
	}
	var keys []string
	for dec.More() {
		t, err := dec.Token()
		if err != nil {
			return nil, err
		}
		keys = append(keys, t.(string))
		var skip json.RawMessage
		if err := dec.Decode(&skip); err != nil {
			return nil, err
		}
	}
	return keys, nil
}

// document is an envelope as a test reads it.
type document struct {
	Format   string
	OK       bool
	Findings []struct {
		Code   string
		Params map[string]any
	}
	Summary json.RawMessage
	Result  json.RawMessage
	Next    []Next
}

// ran is one in-process run of the tool.
type ran struct {
	args           []string
	status         int
	stdout, stderr string
	doc            *document // the document, when the line asked for JSON
}

func (r ran) codes() []string {
	if r.doc == nil {
		return nil
	}
	var out []string
	for _, f := range r.doc.Findings {
		out = append(out, f.Code)
	}
	return out
}

// failingReader fails the test if the tool reads standard input, which
// no command of this wave does (ADR-008 D11 item 6).
type failingReader struct{ t testing.TB }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("standard input was read")
	return 0, errors.New("standard input is not to be read")
}

// runTool runs the tool in-process over the real command table, with
// NO_COLOR set, and checks what every run must satisfy.
func runTool(t testing.TB, argv ...string) ran {
	t.Helper()
	return runWith(t, hostenv.Display{NoColor: true}, argv...)
}

// runWith is runTool with the display d. Every run is held to:
//   - with JSON asked for, one document on standard output, valid against
//     its committed schema, and standard error empty (ADR-008 D11 item 2);
//   - every finding's code is a refusal the line's command declares, or
//     internal.error (ADR-008 D16 item 2);
//   - every next step's command line parses against the declarations;
//   - a refusal names at least one next step (ADR-008 D14).
func runWith(t testing.TB, d hostenv.Display, argv ...string) ran {
	t.Helper()
	return runIn(t, d, failingReader{t}, 0, argv...)
}

// runIn is runWith with stdin as standard input and faults injected.
func runIn(t testing.TB, d hostenv.Display, stdin io.Reader, faults fault.Set, argv ...string) ran {
	t.Helper()
	var stdout, stderr bytes.Buffer
	status := Run(context.Background(), Invocation{
		Args: argv, Stdin: stdin, Stdout: &stdout, Stderr: &stderr,
		Options: Options{Display: d, Faults: faults},
	})
	r := ran{args: argv, status: status, stdout: stdout.String(), stderr: stderr.String()}
	l := readLine(table, argv)
	if l.json {
		// When standard output carries bytes, the document is on standard
		// error (ADR-008 D11 item 2).
		docOut, other := &stdout, &stderr
		if l.streamsBytes() {
			docOut, other = &stderr, nil
		}
		if err := checkDocument(docOut.Bytes()); err != nil {
			t.Errorf("metais %q: %v", argv, err)
		}
		if other != nil && other.Len() != 0 {
			t.Errorf("metais %q --json wrote standard error: %q", argv, other.String())
		}
		var doc document
		if err := json.Unmarshal(docOut.Bytes(), &doc); err != nil {
			t.Fatalf("metais %q: %v", argv, err)
		}
		r.doc = &doc
		refusals := toolRefusals
		if l.cmd != nil {
			refusals = l.cmd.Refusals
		}
		// A refusal leaves no result, and its codes are the ones a
		// command declares (ADR-008 D16 item 2); a report's findings are
		// its result.
		refused := string(doc.Result) == "null"
		for _, code := range r.codes() {
			if !refused {
				break
			}
			if code != string(finding.CodeInternalError) && !slices.ContainsFunc(refusals, func(rf Refusal) bool { return string(rf.Code) == code }) {
				t.Errorf("metais %q emitted %s, which its declaration does not list", argv, code)
			}
		}
		for _, n := range doc.Next {
			if err := parsesAsLine(n.Run); err != nil {
				t.Errorf("metais %q: next %q: %v", argv, n.Run, err)
			}
		}
		if !doc.OK && len(doc.Next) == 0 {
			t.Errorf("metais %q refused with no next step", argv)
		}
	}
	return r
}

// parsesAsLine reports whether line, a metais command line from help or
// a next step, reads cleanly against the declarations, with each role
// placeholder in angle brackets substituted as a harness substitutes it
// (ADR-008 D14 item 2) and each single-quoted word unquoted as a POSIX
// shell unquotes it.
func parsesAsLine(line string) error {
	// A redirection of standard output into a file is the shell's, not
	// metais's: render --out - --only <path> > <file> (ADR-008 A3).
	if i := strings.LastIndex(line, " > "); i >= 0 && !strings.ContainsAny(line[i+3:], " <>") {
		line = line[:i]
	}
	sub := strings.NewReplacer("<package>", "pkg.metais", "<data dir>", "data",
		"<your --set, --set-json and --inputs-json flags>", "--set /a=1 --set-json /b=2 --inputs-json {}", "<dir>", "some-dir", "<name>", "some-name", "<pattern>", "x",
		"<path>", "some/path", "<glob>", "some/**", "<total>", "3")
	fields, err := shellFields(sub.Replace(line))
	if err != nil {
		return fmt.Errorf("%q: %v", line, err)
	}
	if len(fields) == 0 || fields[0] != "metais" {
		return fmt.Errorf("%q is not a metais command line", line)
	}
	for _, f := range fields {
		if strings.ContainsAny(f, "<>\"") {
			return fmt.Errorf("%q holds a placeholder or quoting this test cannot substitute", line)
		}
	}
	l := readLine(table, fields[1:])
	if len(l.findings) > 0 {
		var codes []string
		for _, f := range l.findings {
			codes = append(codes, string(f.Code()))
		}
		return fmt.Errorf("%q does not parse: %v", line, codes)
	}
	if l.cmd == nil && !l.help {
		return fmt.Errorf("%q names no command", line)
	}
	if l.cmd != nil && l.cmd.Name == "help" && len(l.parsed.Operands) == 1 {
		name := l.parsed.Operands[0]
		if _, ok := lookupTopic(name); !ok && lookup(table, name) == nil {
			return fmt.Errorf("%q names no command or topic", line)
		}
	}
	if l.cmd != nil && l.cmd.Name == "help" && len(l.parsed.Operands) == 2 {
		if l.parsed.Operands[0] != "refusals" || lookup(table, l.parsed.Operands[1]) == nil {
			return fmt.Errorf("%q names no command after the topic refusals", line)
		}
	}
	return nil
}

// shellFields splits line into words as a POSIX shell does for words of
// plain characters and single-quoted runs, the only quoting a next step
// writes.
func shellFields(line string) ([]string, error) {
	var out []string
	var w strings.Builder
	in, quoted := false, false
	for i := 0; i < len(line); i++ {
		c := line[i]
		switch {
		case quoted && c == '\'':
			quoted = false
		case quoted:
			w.WriteByte(c)
		case c == '\'':
			quoted, in = true, true
		case c == ' ':
			if in {
				out = append(out, w.String())
				w.Reset()
				in = false
			}
		default:
			w.WriteByte(c)
			in = true
		}
	}
	if quoted {
		return nil, errors.New("an unclosed quote")
	}
	if in {
		out = append(out, w.String())
	}
	return out, nil
}

// allFlags is every flag declared anywhere, the lead's included.
func allFlags() []args.Flag {
	fs := slices.Clone(leadSpec.Flags)
	for _, c := range table {
		fs = append(fs, c.Flags...)
	}
	return fs
}

func jsonUnmarshal(b []byte, v any) error { return json.Unmarshal(b, v) }
