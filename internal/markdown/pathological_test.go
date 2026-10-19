// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// cmarkFile is cmark's pathological suite, vendored from its source with
// its licence (testdata/cmark/SOURCE).
const cmarkFile = "testdata/cmark/pathological_tests.py"

// pathologicalCPU is the CPU time, user and system, that one case may cost
// its process, start-up included. cmark gives each case five seconds of
// wall time; a parser that is linear on these inputs needs a fraction of
// one second even under the race detector, and one that is quadratic on
// them needs minutes.
const pathologicalCPU = 5 * time.Second

// pathologicalAtOnce is how many cases run at once. Each is CPU-bound, and
// a crowd of them would starve the wall clocks of the other packages'
// tests; one at a time spends most of the run starting processes.
const pathologicalAtOnce = 4

// pathologicalKill ends a child that has run far past pathologicalCPU.
const pathologicalKill = 6 * pathologicalCPU

// tail is the end of a child's output, where its failure is.
func tail(out []byte) []byte { return out[max(0, len(out)-2048):] }

// childCase names the case a child process runs; the parent sets it.
const childCase = "METAIS_MARKDOWN_PATHOLOGICAL_CASE"

// TestPathological runs every case of cmark's pathological suite through
// Outline, as the file and as a heading's content, and through
// SplitFrontmatter as the file, as the text of a
// closed frontmatter block and after an unclosed opening line (ADR-009 D10
// item 4, D19 item 10). Each case runs in its own process, re-executing
// this test binary, so that its cost is measured as that process's CPU
// time (ADR-010 D14 item 7) without a platform-specific call. The bound is
// on CPU time; the child's -test.timeout only ends a run already far past
// it, so a quadratic case fails in minutes rather than hours.
func TestPathological(t *testing.T) {
	if name := os.Getenv(childCase); name != "" {
		runPathological(t, name)
		return
	}
	cases := pathologicalCases(t)
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Dir(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != childCase {
			env = append(env, kv)
		}
	}
	slots := make(chan struct{}, pathologicalAtOnce)
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			t.Parallel()
			slots <- struct{}{}
			defer func() { <-slots }()
			cmd := exec.Command(exe, "-test.run=^TestPathological$", "-test.count=1", "-test.v",
				"-test.timeout="+(pathologicalKill).String())
			cmd.Env = slices.Concat(env, []string{childCase + "=" + c.name})
			out, err := cmd.CombinedOutput()
			cpu := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Fatalf("case %q: %v after %v of CPU time (bound %v)\n%s", c.name, err, cpu, pathologicalCPU, tail(out))
			}
			if !strings.Contains(string(out), "pathological case ran: "+strconv.Quote(c.name)) {
				t.Fatalf("case %q: the child did not run it:\n%s", c.name, out)
			}
			t.Logf("%s: %v of CPU time", c.name, cpu)
			if cpu > pathologicalCPU {
				t.Errorf("case %q cost %v of CPU time, over the bound of %v", c.name, cpu, pathologicalCPU)
			}
		})
	}
}

// runPathological is the child's half: it runs the one case named.
func runPathological(t *testing.T, name string) {
	var in string
	for _, c := range pathologicalCases(t) {
		if c.name == name {
			in = c.input(t)
		}
	}
	if in == "" {
		t.Fatalf("no pathological case %q", name)
	}
	src := []byte(in)
	checkOutline(t, src)
	// As the content of a heading, so that the inline pass reads all of
	// it: on one ATX line, and as the paragraph a setext underline makes a
	// heading of.
	checkOutline(t, []byte("# "+strings.ReplaceAll(in, "\n", " ")+"\n"))
	checkOutline(t, []byte(in+"\n===\n"))
	for _, s := range [][]byte{src, []byte("---\n" + in + "\n---\n"), []byte("---\n" + in)} {
		front, frontLine, _, fs := SplitFrontmatter(s, testSource, limits.Defaults())
		if len(fs) == 0 && frontLine == 0 && front != nil {
			t.Fatalf("no frontmatter, but front = %d bytes", len(front))
		}
	}
	fmt.Printf("pathological case ran: %q\n", name)
}

// checkOutline runs Outline over src and checks that every heading is at
// a line of src, in order, at a level of 1 to 6.
func checkOutline(t *testing.T, src []byte) {
	t.Helper()
	last := 1
	n := lineCount(src)
	for _, hd := range Outline(src) {
		if hd.Level < 1 || hd.Level > 6 || hd.Line < last || hd.Line > n {
			t.Fatalf("heading %+v of a %d-line file, after line %d", hd, n, last)
		}
		last = hd.Line
	}
}

// lineCount is ADR-008 D5's number of lines: the LFs, and one more for a
// non-empty file that does not end with one.
func lineCount(src []byte) int {
	n := strings.Count(string(src), "\n")
	if len(src) > 0 && src[len(src)-1] != '\n' {
		n++
	}
	return n
}

// A pathological case is one entry of the suite's dictionaries: its name
// and the expression that builds its input.
type pathologicalCase struct {
	name string
	expr pyExpr
	file string // the vendored file, for hash_collisions
}

func (c pathologicalCase) input(t *testing.T) string {
	v, err := c.expr.eval(pyEnv{file: c.file})
	if err != nil {
		t.Fatalf("case %q: %v", c.name, err)
	}
	switch v := v.(type) {
	case string:
		return v
	case []any: // (input, regex): the regex is for cmark's HTML, which METAIS never renders
		if s, ok := v[0].(string); ok && len(v) == 2 {
			return s
		}
	}
	t.Fatalf("case %q: its value is %T, not (input, regex)", c.name, v)
	return ""
}

// pathologicalCases reads every case of the vendored file's two
// dictionaries, pathological and pathological_cmark.
//
// cmark states its cases only as Python: each input is an expression of
// string literals, repetition and concatenation, and a join over a map of
// a lambda over a range; and one is the value of hash_collisions(). The
// expressions are evaluated here as data, by an evaluator of exactly that
// subset, which refuses anything outside it, so a case upstream writes
// some other way fails here rather than being skipped. hash_collisions is
// a function, not an expression, and pyHashCollisions mirrors it; its text
// is pinned by digest, so a change to it upstream fails here too.
func pathologicalCases(t *testing.T) []pathologicalCase {
	b, err := os.ReadFile(cmarkFile)
	if err != nil {
		t.Fatal(err)
	}
	file := string(b)
	var cases []pathologicalCase
	for _, dict := range []string{"pathological", "pathological_cmark"} {
		open := "\n" + dict + " = {"
		i := strings.Index(file, open)
		if i < 0 {
			t.Fatalf("%s has no dictionary %s", cmarkFile, dict)
		}
		toks, err := pyTokens(file[i+len(open):])
		if err != nil {
			t.Fatalf("%s: dictionary %s: %v", cmarkFile, dict, err)
		}
		p := &pyParser{toks: toks}
		entries, err := p.dict()
		if err != nil {
			t.Fatalf("%s: dictionary %s: %v", cmarkFile, dict, err)
		}
		for _, e := range entries {
			cases = append(cases, pathologicalCase{name: e.key, expr: e.value, file: file})
		}
	}
	if len(cases) == 0 {
		t.Fatal("no pathological cases")
	}
	return cases
}

// TestPathologicalCasesRead pins the reading of the vendored file: every
// case of both dictionaries, by name, and the inputs of a few whose size
// the file states plainly. The commented-out "many references" is not a
// case.
func TestPathologicalCasesRead(t *testing.T) {
	cases := pathologicalCases(t)
	var names []string
	for _, c := range cases {
		names = append(names, c.name)
	}
	want := []string{
		"nested strong emph", "many emph closers with no openers", "many emph openers with no closers",
		"many link closers with no openers", "many link openers with no closers",
		"mismatched openers and closers", "issue #389", "openers and closers multiple of 3",
		"link openers and emph closers", "pattern [ (]( repeated", "pattern ![[]() repeated",
		"hard link/emph case", "nested brackets", "nested block quotes", "deeply nested lists",
		"U+0000 in input", "backticks", "unclosed links A", "unclosed links B", "unclosed <!--",
		"empty lines in deeply nested lists", "empty lines in deeply nested lists in blockquote",
		"emph in deep blockquote", "reference collisions",
		"nested inlines",
	}
	if strings.Join(names, "|") != strings.Join(want, "|") {
		t.Fatalf("cases %q, want %q", names, want)
	}
	inputs := map[string]string{}
	for _, c := range cases {
		inputs[c.name] = c.input(t)
	}
	for name, want := range map[string]string{
		"nested strong emph":      strings.Repeat("*a **a ", 32500) + "b" + strings.Repeat(" a** a*", 32500),
		"pattern ![[]() repeated": strings.Repeat("![[]()", 160000),
		"hard link/emph case":     "**x [a*b**c*](d)",
		"U+0000 in input":         "abc\x00de\x00",
		"unclosed links A":        strings.Repeat("[a](<b", 30000),
		"unclosed <!--":           "</" + strings.Repeat("<!--", 300000),
		"emph in deep blockquote": strings.Repeat(">", 100000) + strings.Repeat("a*", 100000),
		"nested inlines":          strings.Repeat("*", 20000) + "a" + strings.Repeat("*", 20000),
	} {
		if inputs[name] != want {
			t.Errorf("case %q: input of %d bytes, want %d", name, len(inputs[name]), len(want))
		}
	}
	var lists strings.Builder
	for x := range 500 {
		lists.WriteString(strings.Repeat("  ", x) + "* a\n")
	}
	if inputs["deeply nested lists"] != lists.String() {
		t.Error("case \"deeply nested lists\": the join over a map over a range is misread")
	}
	rc := inputs["reference collisions"]
	if n := strings.Count(rc, "]: /url\n\n["); n != 24999 {
		t.Errorf("reference collisions: %d definitions, want 24999", n)
	}
	if !strings.HasPrefix(rc, "[x") {
		t.Errorf("reference collisions begins %q", rc[:min(len(rc), 20)])
	}
}

// TestPyEvaluatorRefuses pins that the evaluator refuses what it does not
// know, rather than skipping it.
func TestPyEvaluatorRefuses(t *testing.T) {
	for _, src := range []string{
		`"a": ("x" - 1, re.compile("x")) }`,
		`"a": ("x" * "y", re.compile("x")) }`,
		`"a": (f"x", re.compile("x")) }`,
		`"a": (other(), re.compile("x")) }`,
		`"a": ("x" * y, re.compile("x")) }`,
		`"a": ("""x""", re.compile("x")) }`,
		`"a": ("x"[0], re.compile("x")) }`,
	} {
		toks, err := pyTokens(src)
		if err == nil {
			var entries []pyEntry
			p := &pyParser{toks: toks}
			if entries, err = p.dict(); err == nil {
				_, err = entries[0].value.eval(pyEnv{})
			}
		}
		if err == nil {
			t.Errorf("%s: evaluated", src)
		}
	}
}

// hashCollisionsDigest is the SHA-256 of the vendored hash_collisions
// function, from its def line to the blank lines that end it. A change to
// it upstream is a change to pyHashCollisions.
const hashCollisionsDigest = "cb79875d257c9bac9bb1fbc81a9820f022470ced0a4765b87bd22154df90bce5"

// pyHashCollisions is the document the vendored hash_collisions() returns:
// count link reference definitions whose labels collide in a reference
// map of refmapSize buckets under cmark's old hash, each followed by a
// reference to the first of them.
func pyHashCollisions(file string) (string, error) {
	start := strings.Index(file, "def hash_collisions():")
	end := strings.Index(file, "\n\n\n# list of pairs")
	if start < 0 || end < start {
		return "", fmt.Errorf("hash_collisions is not where it was")
	}
	sum := sha256.Sum256([]byte(file[start:end]))
	if got := hex.EncodeToString(sum[:]); got != hashCollisionsDigest {
		return "", fmt.Errorf("hash_collisions changed (sha256 %s); update pyHashCollisions to the new text", got)
	}
	const refmapSize, count = 16, 25000 // REFMAP_SIZE and COUNT
	badhash := func(ref string) bool {
		var h uint32
		for _, c := range ref {
			h = uint32(c) + h<<6 + h<<16 - h
		}
		return h%refmapSize == 0
	}
	var keys []string
	for i := 0; len(keys) < count; i++ {
		if k := "x" + strconv.Itoa(i); badhash(k) {
			keys = append(keys, k)
		}
	}
	var b strings.Builder
	for _, k := range keys[1:] {
		fmt.Fprintf(&b, "[%s]: /url\n\n[%s]\n\n", k, keys[0])
	}
	return b.String(), nil
}

// The evaluator. pyTokens and pyParser read the subset; eval computes it.

type pyTok struct {
	kind byte // 's' string, 'n' integer, 'i' name, or the punctuation itself
	text string
}

// pyTokens splits src, the text after a dictionary's "{", into tokens up
// to and including the "}" that closes it, dropping comments and
// whitespace.
func pyTokens(src string) ([]pyTok, error) {
	var toks []pyTok
	depth := 0
	for i := 0; i < len(src); {
		c := src[i]
		switch {
		case c == ' ' || c == '\t' || c == '\n' || c == '\r':
			i++
		case c == '#':
			for i < len(src) && src[i] != '\n' {
				i++
			}
		case c == '"' || c == '\'' || ((c == 'r' || c == 'R') && i+1 < len(src) && (src[i+1] == '"' || src[i+1] == '\'')):
			s, n, err := pyString(src[i:])
			if err != nil {
				return nil, err
			}
			toks = append(toks, pyTok{'s', s})
			i += n
		case c >= '0' && c <= '9':
			j := i
			for j < len(src) && src[j] >= '0' && src[j] <= '9' {
				j++
			}
			toks = append(toks, pyTok{'n', src[i:j]})
			i = j
		case c == '_' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z':
			j := i
			for j < len(src) && (src[j] == '_' || src[j] >= 'a' && src[j] <= 'z' || src[j] >= 'A' && src[j] <= 'Z' || src[j] >= '0' && src[j] <= '9') {
				j++
			}
			if j < len(src) && (src[j] == '"' || src[j] == '\'') {
				return nil, fmt.Errorf("string prefix %q is not read", src[i:j])
			}
			toks = append(toks, pyTok{'i', src[i:j]})
			i = j
		case strings.IndexByte("(){},:*+.", c) >= 0:
			toks = append(toks, pyTok{c, string(c)})
			i++
			switch c {
			case '(', '{':
				depth++
			case ')', '}':
				if depth--; depth < 0 {
					return toks, nil
				}
			}
		default:
			return nil, fmt.Errorf("character %q is not read", c)
		}
	}
	return nil, fmt.Errorf("the dictionary is not closed")
}

// pyString reads one string literal at the start of src, a raw one when
// it has the r prefix, and returns its value and its length in src.
func pyString(src string) (string, int, error) {
	i := 0
	raw := src[0] == 'r' || src[0] == 'R'
	if raw {
		i++
	}
	q := src[i]
	if strings.HasPrefix(src[i:], strings.Repeat(string(q), 3)) {
		return "", 0, fmt.Errorf("a triple-quoted string is not read")
	}
	i++
	var b strings.Builder
	for i < len(src) {
		c := src[i]
		switch {
		case c == q:
			return b.String(), i + 1, nil
		case c == '\n':
			return "", 0, fmt.Errorf("unterminated string")
		case c == '\\' && i+1 < len(src):
			if raw {
				b.WriteString(src[i : i+2])
				i += 2
				continue
			}
			e := src[i+1]
			i += 2
			switch e {
			case '\\', '\'', '"':
				b.WriteByte(e)
			case 'n':
				b.WriteByte('\n')
			case 't':
				b.WriteByte('\t')
			case 'r':
				b.WriteByte('\r')
			case 'u':
				if i+4 > len(src) {
					return "", 0, fmt.Errorf("short \\u escape")
				}
				r, err := strconv.ParseUint(src[i:i+4], 16, 32)
				if err != nil {
					return "", 0, fmt.Errorf("bad \\u escape: %v", err)
				}
				b.WriteString(string(rune(r)))
				i += 4
			default: // Python keeps an unknown escape as written
				b.WriteByte('\\')
				b.WriteByte(e)
			}
		default:
			b.WriteByte(c)
			i++
		}
	}
	return "", 0, fmt.Errorf("unterminated string")
}

type pyEntry struct {
	key   string
	value pyExpr
}

type pyParser struct {
	toks []pyTok
	pos  int
}

func (p *pyParser) peek() pyTok {
	if p.pos < len(p.toks) {
		return p.toks[p.pos]
	}
	return pyTok{}
}

func (p *pyParser) next() pyTok { t := p.peek(); p.pos++; return t }

func (p *pyParser) want(kind byte) (pyTok, error) {
	t := p.next()
	if t.kind != kind {
		return t, fmt.Errorf("token %d: %q, want %q", p.pos, t.text, kind)
	}
	return t, nil
}

// dict reads `"key": expr, ...}`, the body of a dictionary after its "{".
func (p *pyParser) dict() ([]pyEntry, error) {
	var entries []pyEntry
	for p.peek().kind != '}' {
		k, err := p.want('s')
		if err != nil {
			return nil, err
		}
		if _, err := p.want(':'); err != nil {
			return nil, err
		}
		v, err := p.expr()
		if err != nil {
			return nil, fmt.Errorf("%q: %v", k.text, err)
		}
		entries = append(entries, pyEntry{k.text, v})
		if p.peek().kind == ',' {
			p.next()
		}
	}
	return entries, nil
}

// pyExpr is a parsed expression of the subset.
type pyExpr interface {
	eval(env pyEnv) (any, error)
}

type pyEnv struct {
	file string         // the vendored file, for hash_collisions
	vars map[string]any // a lambda's parameter
}

type (
	pyLit    struct{ v any }
	pyName   struct{ name string }
	pyBinary struct {
		op   byte
		l, r pyExpr
	}
	pyTuple struct{ elems []pyExpr }
	pyAttr  struct {
		x    pyExpr
		name string
	}
	pyCall struct {
		fn   pyExpr
		args []pyExpr
	}
	pyLambda struct {
		param string
		body  pyExpr
	}
)

// expr := lambda name ":" expr | sum
func (p *pyParser) expr() (pyExpr, error) {
	if t := p.peek(); t.kind == 'i' && t.text == "lambda" {
		p.next()
		name, err := p.want('i')
		if err != nil {
			return nil, err
		}
		if _, err := p.want(':'); err != nil {
			return nil, err
		}
		body, err := p.expr()
		return pyLambda{name.text, body}, err
	}
	return p.binary('+', p.product)
}

func (p *pyParser) product() (pyExpr, error) { return p.binary('*', p.postfix) }

func (p *pyParser) binary(op byte, operand func() (pyExpr, error)) (pyExpr, error) {
	l, err := operand()
	for err == nil && p.peek().kind == op {
		p.next()
		var r pyExpr
		r, err = operand()
		l = pyBinary{op, l, r}
	}
	return l, err
}

// postfix := primary ("." name | "(" args ")")*
func (p *pyParser) postfix() (pyExpr, error) {
	x, err := p.primary()
	for err == nil {
		switch p.peek().kind {
		case '.':
			p.next()
			var name pyTok
			if name, err = p.want('i'); err == nil {
				x = pyAttr{x, name.text}
			}
		case '(':
			p.next()
			var args []pyExpr
			if args, err = p.list(')'); err == nil {
				x = pyCall{x, args}
			}
		default:
			return x, nil
		}
	}
	return nil, err
}

// primary := string | integer | name | "(" expr ("," expr)* ")"
func (p *pyParser) primary() (pyExpr, error) {
	t := p.next()
	switch t.kind {
	case 's':
		return pyLit{t.text}, nil
	case 'n':
		n, err := strconv.Atoi(t.text)
		return pyLit{n}, err
	case 'i':
		return pyName{t.text}, nil
	case '(':
		elems, err := p.list(')')
		if err != nil {
			return nil, err
		}
		if len(elems) == 1 && p.toks[p.pos-2].kind != ',' {
			return elems[0], nil
		}
		return pyTuple{elems}, nil
	}
	return nil, fmt.Errorf("token %d: %q begins no expression", p.pos, t.text)
}

// list reads expressions separated by commas up to the closing token.
func (p *pyParser) list(end byte) ([]pyExpr, error) {
	var elems []pyExpr
	for p.peek().kind != end {
		e, err := p.expr()
		if err != nil {
			return nil, err
		}
		elems = append(elems, e)
		if p.peek().kind != ',' {
			break
		}
		p.next()
	}
	_, err := p.want(end)
	return elems, err
}

func (e pyLit) eval(pyEnv) (any, error) { return e.v, nil }

func (e pyName) eval(env pyEnv) (any, error) {
	if v, ok := env.vars[e.name]; ok {
		return v, nil
	}
	return nil, fmt.Errorf("name %q is not bound", e.name)
}

func (e pyBinary) eval(env pyEnv) (any, error) {
	l, err := e.l.eval(env)
	if err != nil {
		return nil, err
	}
	r, err := e.r.eval(env)
	if err != nil {
		return nil, err
	}
	switch e.op {
	case '+':
		if a, ok := l.(string); ok {
			if b, ok := r.(string); ok {
				return a + b, nil
			}
		}
		if a, ok := l.(int); ok {
			if b, ok := r.(int); ok {
				return a + b, nil
			}
		}
	case '*':
		if s, ok := l.(string); ok {
			if n, ok := r.(int); ok && n >= 0 {
				return strings.Repeat(s, n), nil
			}
		}
		if n, ok := l.(int); ok {
			if s, ok := r.(string); ok && n >= 0 {
				return strings.Repeat(s, n), nil
			}
		}
	}
	return nil, fmt.Errorf("%T %c %T is not read", l, e.op, r)
}

func (e pyTuple) eval(env pyEnv) (any, error) {
	var vs []any
	for _, x := range e.elems {
		v, err := x.eval(env)
		if err != nil {
			return nil, err
		}
		vs = append(vs, v)
	}
	return vs, nil
}

func (e pyAttr) eval(pyEnv) (any, error) {
	return nil, fmt.Errorf("attribute %q outside a call is not read", e.name)
}

func (e pyLambda) eval(pyEnv) (any, error) { return e, nil }

// pyRegex is the value of re.compile: cmark's check of its HTML, which
// METAIS has no use for.
type pyRegex struct{}

func (e pyCall) eval(env pyEnv) (any, error) {
	args := make([]any, len(e.args))
	for i, a := range e.args {
		v, err := a.eval(env)
		if err != nil {
			return nil, err
		}
		args[i] = v
	}
	switch fn := e.fn.(type) {
	case pyAttr:
		if n, ok := fn.x.(pyName); ok && n.name == "re" && fn.name == "compile" && len(args) == 1 {
			if _, ok := args[0].(string); ok {
				return pyRegex{}, nil
			}
		}
		if sep, ok := fn.x.(pyLit); ok && fn.name == "join" && len(args) == 1 {
			s, ok1 := sep.v.(string)
			items, ok2 := args[0].([]any)
			if ok1 && ok2 {
				parts := make([]string, len(items))
				for i, it := range items {
					if parts[i], ok = it.(string); !ok {
						return nil, fmt.Errorf("join of a %T", it)
					}
				}
				return strings.Join(parts, s), nil
			}
		}
	case pyName:
		switch {
		case fn.name == "range" && len(args) == 2:
			lo, ok1 := args[0].(int)
			hi, ok2 := args[1].(int)
			if ok1 && ok2 {
				var vs []any
				for i := lo; i < hi; i++ {
					vs = append(vs, i)
				}
				return vs, nil
			}
		case fn.name == "map" && len(args) == 2:
			f, ok1 := args[0].(pyLambda)
			items, ok2 := args[1].([]any)
			if ok1 && ok2 {
				vs := make([]any, len(items))
				for i, it := range items {
					v, err := f.body.eval(pyEnv{file: env.file, vars: map[string]any{f.param: it}})
					if err != nil {
						return nil, err
					}
					vs[i] = v
				}
				return vs, nil
			}
		case fn.name == "hash_collisions" && len(args) == 0:
			doc, err := pyHashCollisions(env.file)
			return []any{doc, pyRegex{}}, err
		}
	}
	return nil, fmt.Errorf("a call of %#v is not read", e.fn)
}
