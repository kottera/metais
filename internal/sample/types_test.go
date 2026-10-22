// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"math/big"
	"regexp"
	"slices"
	"strings"
	"testing"
	"unicode/utf8"

	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/schema"
)

// each generates schema's value under seeds 0 to n−1, each validated.
func each(t *testing.T, schemaText string, n int, f func(seed int, v value.Value)) {
	t.Helper()
	for seed := range n {
		f(seed, mustOne(t, schemaText, Options{Seed: uint64(seed), Items: 3}))
	}
}

// exact is the exact rational v's literal writes.
func exact(v value.Value) *big.Rat {
	return rat(v.(value.Number))
}

// TestNumbers is item 17: the range is the bounds, [1, 100] when
// unbounded, 100 from the bound when bounded on one side; integers are
// drawn from the integers in range that are multiples of multipleOf;
// numbers as n/100, refining to n/10^6 when the range holds no hundredth,
// or as multiples of multipleOf; written as plain decimals.
func TestNumbers(t *testing.T) {
	for _, tc := range []struct {
		name, schema string
		lo, hi       string // the range, inclusive
		step         string // every value is a multiple of it
	}{
		{"an unbounded integer", `{"type":"integer"}`, "1", "100", "1"},
		{"an unbounded number", `{"type":"number"}`, "1", "100", "1/100"},
		{"bounded below", `{"type":"integer","minimum":1000}`, "1000", "1100", "1"},
		{"bounded above", `{"type":"number","maximum":-5}`, "-105", "-5", "1/100"},
		{"exclusive bounds", `{"type":"integer","exclusiveMinimum":3,"exclusiveMaximum":6}`, "4", "5", "1"},
		{"the tightest of several", `{"type":"integer","minimum":0,"maximum":50,"allOf":[{"minimum":40},{"exclusiveMaximum":45}]}`, "40", "44", "1"},
		{"integer multiples", `{"type":"integer","multipleOf":7,"minimum":0,"maximum":30}`, "0", "28", "7"},
		{"integer multiples of a fraction", `{"type":"integer","multipleOf":2.5,"minimum":0,"maximum":30}`, "0", "30", "5"},
		{"the least common multiple", `{"type":"integer","multipleOf":4,"allOf":[{"multipleOf":6}],"minimum":1,"maximum":100}`, "12", "96", "12"},
		{"number multiples", `{"type":"number","multipleOf":0.25,"minimum":1,"maximum":2}`, "1", "2", "1/4"},
		{"refined to millionths", `{"type":"number","exclusiveMinimum":0.001,"exclusiveMaximum":0.009}`, "1001/1000000", "8999/1000000", "1/1000000"},
		{"near the safe bound", `{"type":"integer","minimum":9007199254740900}`, "9007199254740900", "9007199254740991", "1"},
		{"inferred from minimum", `{"minimum":2,"maximum":3}`, "2", "3", "1/100"},
	} {
		seen := map[string]bool{}
		each(t, tc.schema, 12, func(seed int, v value.Value) {
			n, ok := v.(value.Number)
			if !ok {
				t.Fatalf("%s: %s is no number", tc.name, text(v))
			}
			r := exact(v)
			lo, _ := new(big.Rat).SetString(tc.lo)
			hi, _ := new(big.Rat).SetString(tc.hi)
			step, _ := new(big.Rat).SetString(tc.step)
			if r.Cmp(lo) < 0 || r.Cmp(hi) > 0 || !new(big.Rat).Quo(r, step).IsInt() {
				t.Errorf("%s, seed %d: %s is not a multiple of %s in [%s, %s]", tc.name, seed, n.Literal(), tc.step, tc.lo, tc.hi)
			}
			if strings.ContainsAny(n.Literal(), "eE") {
				t.Errorf("%s: %s has an exponent", tc.name, n.Literal())
			}
			seen[n.Literal()] = true
		})
		if len(seen) < 2 {
			t.Errorf("%s: 12 seeds gave only %v", tc.name, seen)
		}
	}
}

// TestSafeBound pins item 17's clamp at its exact edge: the generator
// makes no integer beyond ±(2^53−1), so a range that starts at 2^53 holds
// none it may make, and a range that reaches 2^53 is cut back to 2^53−1.
// Each range here holds one value the generator may make, or none, so
// every seed gives the same answer. 2^53 is spelled with an exponent,
// since an integer literal beyond 2^53−1 does not open (ADR-003 D8 item
// 5).
func TestSafeBound(t *testing.T) {
	for _, sch := range []string{
		`{"type":"integer","minimum":9.007199254740992e15}`,
		`{"type":"integer","maximum":-9.007199254740992e15}`,
	} {
		_, fs := one(t, sch, Options{Items: 3})
		if got := reasons(t, fs); !slices.Equal(got, []string{ReasonRange}) {
			t.Errorf("%s: reasons %v, want %s%s", sch, got, ReasonRange, describe(fs))
		}
	}
	for _, tc := range []struct{ schema, want string }{
		{`{"type":"integer","minimum":9007199254740991,"maximum":9.007199254740992e15}`, "9007199254740991"},
		{`{"type":"integer","minimum":-9.007199254740992e15,"maximum":-9007199254740991}`, "-9007199254740991"},
	} {
		each(t, tc.schema, 12, func(seed int, v value.Value) {
			if n, ok := v.(value.Number); !ok || n.Literal() != tc.want {
				t.Errorf("%s, seed %d: %s, want %s", tc.schema, seed, text(v), tc.want)
			}
		})
	}
}

// TestBoolean: a boolean is drawn by its value stream.
func TestBoolean(t *testing.T) {
	seen := map[string]bool{}
	each(t, `{"type":"boolean"}`, 8, func(_ int, v value.Value) { seen[text(v)] = true })
	if !seen["true"] || !seen["false"] {
		t.Errorf("8 seeds gave %v", seen)
	}
}

// TestTypeChoice is item 14: the first type of the set, in the order
// written; with no type, the type the keywords imply.
func TestTypeChoice(t *testing.T) {
	for _, tc := range []struct{ schema, kind string }{
		{`{"type":["integer","string"]}`, "number"},
		{`{"type":["string","integer"]}`, "string"},
		{`{"type":["number","integer"],"allOf":[{"type":"integer"}]}`, "number"},
		{`{"properties":{"a":{"type":"integer"}}}`, "object"},
		{`{"required":["a"]}`, "object"},
		{`{"items":{"type":"integer"}}`, "array"},
		{`{"minLength":3}`, "string"},
		{`{"multipleOf":3}`, "number"},
		{`{}`, "string"},
		{`{"type":"null"}`, "null"},
	} {
		if got := mustOne(t, tc.schema, Options{Items: 3}).Kind().String(); got != tc.kind {
			t.Errorf("%s: %s, want %s", tc.schema, got, tc.kind)
		}
	}
	v := mustOne(t, `{"type":["number","integer"],"allOf":[{"type":"integer"}]}`, Options{})
	if !v.(value.Number).Integral() {
		t.Errorf("number meeting integer gave %s", text(v))
	}
}

// TestPlainText is item 18's plain text: corpus words to a target of 24
// code points within the bounds; under 40, words joined by spaces, the
// first capitalised, no punctuation; from 40, sentences of 6 to 12
// words, each capitalised and ending in a full stop; cut at a word
// boundary to fit maxLength, extended by words to reach minLength.
func TestPlainText(t *testing.T) {
	words := regexp.MustCompile(`^[A-Z][a-z]*( [a-z]+)*$`)
	sentences := regexp.MustCompile(`^([A-Z][a-z]*( [a-z]+){5,11}\.)( [A-Z][a-z]*( [a-z]+){5,11}\.)*`)
	corpus := map[string]bool{}
	for _, w := range strings.Fields(strings.ToLower(strings.NewReplacer(",", "", ".", "").Replace(Corpus))) {
		corpus[w] = true
	}
	for _, tc := range []struct {
		schema   string
		min, max int
		re       *regexp.Regexp
	}{
		{`{"type":"string"}`, 24, 40, words},
		{`{"type":"string","maxLength":10}`, 1, 10, words},
		{`{"type":"string","minLength":30,"maxLength":35}`, 30, 35, words},
		{`{"type":"string","minLength":60}`, 60, 200, sentences},
		{`{"type":"string","minLength":45,"maxLength":50}`, 45, 50, nil},
		{`{"type":"string","maxLength":2}`, 1, 2, nil},
	} {
		each(t, tc.schema, 6, func(seed int, v value.Value) {
			s := v.(value.String).Text()
			n := utf8.RuneCountInString(s)
			if n < tc.min || n > tc.max || tc.re != nil && !tc.re.MatchString(s) {
				t.Errorf("%s, seed %d: %q (%d)", tc.schema, seed, s, n)
			}
			// Every word is the corpus's; the last may be cut, when the
			// bounds leave no word boundary that fits.
			ws := strings.Fields(strings.ToLower(strings.ReplaceAll(s, ".", "")))
			for i, w := range ws {
				if !corpus[w] && !(i == len(ws)-1 && n == tc.max && prefixOfWord(corpus, w)) {
					t.Errorf("%s: %q is not a corpus word", tc.schema, w)
				}
			}
		})
	}
	if got := text(mustOne(t, `{"type":"string","maxLength":0}`, Options{})); got != `""` {
		t.Errorf("maxLength 0: %s", got)
	}
}

// prefixOfWord reports whether w begins a word of corpus.
func prefixOfWord(corpus map[string]bool, w string) bool {
	for c := range corpus {
		if strings.HasPrefix(c, w) {
			return true
		}
	}
	return false
}

// TestProse is item 18's prose: a whole text-form input, a collection
// item's body or a contentMediaType text/plain or text/markdown string
// targets 400 code points within its bounds, as paragraphs of three to
// five sentences separated by a blank line; Markdown adds a level-one
// heading of three title-cased words and a three-item bullet list.
func TestProse(t *testing.T) {
	sentence := `[A-Z][a-z]*( [a-z]+){5,11}\.`
	para := sentence + `( ` + sentence + `){2,4}`
	plain := regexp.MustCompile(`^` + para + `(\n\n` + para + `)*$`)
	md := regexp.MustCompile(`^# [A-Z][a-z]* [A-Z][a-z]* [A-Z][a-z]*\n\n` + para + `(\n\n` + para + `)*\n\n- [A-Z][a-z]* [a-z]+ [a-z]+\n- [A-Z][a-z]* [a-z]+ [a-z]+\n- [A-Z][a-z]* [a-z]+ [a-z]+$`)
	p := compile(t, map[string]string{
		entry:                  `{"type":"string"}`,
		"inputs/notes.json":    `{"type":"object","properties":{"md":{"type":"string","contentMediaType":"text/markdown"},"txt":{"type":"string","contentMediaType":"text/plain","maxLength":120}}}`,
		"inputs/articles.json": `{"type":"array","items":{"type":"object","required":["title"],"properties":{"title":{"type":"string"},"body":{"type":"string"}}}}`,
		"inputs/pages.json":    `{"type":"array","items":{"type":"object","properties":{"title":{"type":"string"}}}}`,
	})
	gen := func(in Input) value.Value {
		v, _, fs, err := p.gen(Options{Items: 2}).Generate(in)
		if err != nil || fs != nil {
			t.Fatal(err, describe(fs))
		}
		return v
	}
	check := func(what string, s string, re *regexp.Regexp, lo, hi int) {
		if n := utf8.RuneCountInString(s); n < lo || n > hi || !re.MatchString(s) {
			t.Errorf("%s: %q (%d)", what, s, n)
		}
	}
	check("a text input", gen(Input{Name: "summary", Schema: entry, Forms: []Form{FormText}, Required: true}).(value.String).Text(), plain, 400, 600)
	notes := gen(Input{Name: "notes", Schema: "inputs/notes.json", Forms: []Form{FormJSON}, Required: true})
	check("text/markdown", member(notes, "md").(value.String).Text(), md, 400, 800)
	check("text/plain within maxLength", member(notes, "txt").(value.String).Text(), regexp.MustCompile(`^[A-Z]`), 60, 120)
	articles := gen(Input{Name: "articles", Schema: "inputs/articles.json", Forms: []Form{FormJSON}, Collection: true, Required: true})
	check("a collection item's body", member(articles.(value.Array).At(0), "body").(value.String).Text(), plain, 400, 600)
	if title := member(articles.(value.Array).At(0), "title").(value.String).Text(); utf8.RuneCountInString(title) > 40 {
		t.Errorf("a title is not prose: %q", title)
	}
	pages := gen(Input{Name: "pages", Schema: "inputs/pages.json", Forms: []Form{FormMarkdown}, Collection: true, Required: true})
	check("a Markdown item's body", member(pages.(value.Array).At(1), "body").(value.String).Text(), md, 400, 800)
}

// TestFormats is item 18's formats: each asserted format, from fixed,
// reserved or placeholder sources, valid by the validator's own check.
func TestFormats(t *testing.T) {
	want := map[string]*regexp.Regexp{
		"date":                  regexp.MustCompile(`^202[0-9]-[01][0-9]-[0-3][0-9]$`),
		"date-time":             regexp.MustCompile(`^202[0-9]-[01][0-9]-[0-3][0-9]T[0-2][0-9]:[0-5][0-9]:[0-5][0-9]Z$`),
		"time":                  regexp.MustCompile(`^[0-2][0-9]:[0-5][0-9]:[0-5][0-9]Z$`),
		"duration":              regexp.MustCompile(`^(P[0-9]+D|PT[0-9]+H|P[0-9]+DT[0-9]+H)$`),
		"email":                 regexp.MustCompile(`^[a-z]+\.[a-z]+@example\.com$`),
		"hostname":              regexp.MustCompile(`^[a-z]+\.example\.com$`),
		"ipv4":                  regexp.MustCompile(`^(192\.0\.2|198\.51\.100|203\.0\.113)\.[0-9]+$`),
		"ipv6":                  regexp.MustCompile(`^2001:db8(:[0-9a-f]{1,4}){6}$`),
		"uri":                   regexp.MustCompile(`^https://example\.com/[a-z]+/[a-z]+$`),
		"uri-reference":         regexp.MustCompile(`^/[a-z]+/[a-z]+$`),
		"uri-template":          regexp.MustCompile(`^https://example\.com/\{[a-z]+\}$`),
		"uuid":                  regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`),
		"json-pointer":          regexp.MustCompile(`^/[a-z]+/[0-9]$`),
		"relative-json-pointer": regexp.MustCompile(`^0/[a-z]+$`),
	}
	for _, f := range schema.Formats() {
		re, ok := want[f]
		if !ok {
			t.Errorf("the asserted format %s has no generator", f)
			continue
		}
		each(t, `{"type":"string","format":"`+f+`"}`, 20, func(seed int, v value.Value) {
			if s := v.(value.String).Text(); !re.MatchString(s) {
				t.Errorf("%s, seed %d: %q", f, seed, s)
			}
		})
	}
	// Every day of the decade is reachable, and none outside it.
	st := NewSplitMix64(1)
	for range 5000 {
		if d := date(st); d < "2020-01-01" || d > "2029-12-31" {
			t.Fatalf("date %s", d)
		}
	}
}

// TestPatterns is item 18's pattern walk: literals as written; a class
// yields from a–z, A–Z, 0–9, space, printable ASCII, else its lowest
// character; alternation picks a branch; a repetition repeats min plus 0
// to 2, capped at max; anchors yield nothing; a result shorter than
// minLength is padded with corpus words after it unless end-anchored,
// else before it unless start-anchored, else regenerated with
// repetitions raised.
func TestPatterns(t *testing.T) {
	for _, tc := range []struct {
		pattern string
		extra   string
		want    string // the whole value matches it
	}{
		{`^abc$`, ``, `^abc$`},
		{`^[a-z]$`, ``, `^[a-z]$`},
		{`^[A-Z0-9]$`, ``, `^[A-Z]$`},
		{`^[0-9_]$`, ``, `^[0-9]$`},
		{`^[ \t]$`, ``, `^ $`},
		{`^[!-/]$`, ``, `^[!-/]$`},
		{`^[éê]$`, ``, `^é$`},
		{`^\d{3}$`, ``, `^[0-9]{3}$`},
		{`^x{2,}$`, ``, `^x{2,4}$`},
		{`^y{1,2}$`, ``, `^y{1,2}$`},
		{`^(cat|dog)s?$`, ``, `^(cat|dog)s?$`},
		{`^a.b$`, ``, `^a[a-z]b$`},
		{`id`, `,"minLength":10`, `^id( [a-z]+)+$`},
		{`id$`, `,"minLength":10`, `^([a-z]+ )+id$`},
		{`^q+$`, `,"minLength":6`, `^q{6,}$`},
		{`^\p{Lu}{2}$`, ``, `^[A-Z]{2}$`},
	} {
		re := regexp.MustCompile(tc.want)
		each(t, `{"type":"string","pattern":`+quoteJSON(tc.pattern)+tc.extra+`}`, 10, func(seed int, v value.Value) {
			if s := v.(value.String).Text(); !re.MatchString(s) {
				t.Errorf("%s, seed %d: %q does not match %s", tc.pattern, seed, s, tc.want)
			}
		})
	}
}

// quoteJSON is s as a JSON string.
func quoteJSON(s string) string {
	return `"` + strings.NewReplacer(`\`, `\\`, `"`, `\"`).Replace(s) + `"`
}

// TestArrays is item 16: the length is --items clamped to minItems and
// maxItems, maxItems lowered to the prefixItems length when items is
// false, the prefixItems length the default target when prefixItems is
// present; position i takes prefixItems[i], else items; the first
// minContains positions are made from contains as well.
func TestArrays(t *testing.T) {
	for _, tc := range []struct {
		schema string
		items  int
		want   string
	}{
		{`{"type":"array","items":{"const":1}}`, 3, `[1,1,1]`},
		{`{"type":"array","items":{"const":1}}`, 0, `[]`},
		{`{"type":"array","items":{"const":1},"minItems":4}`, 2, `[1,1,1,1]`},
		{`{"type":"array","items":{"const":1},"maxItems":1}`, 5, `[1]`},
		{`{"type":"array","prefixItems":[{"const":"a"},{"const":"b"}],"items":{"const":0}}`, 5, `["a","b"]`},
		{`{"type":"array","prefixItems":[{"const":"a"},{"const":"b"}],"items":{"const":0},"minItems":4}`, 5, `["a","b",0,0]`},
		{`{"type":"array","prefixItems":[{"const":"a"}],"items":false,"minItems":0}`, 5, `["a"]`},
		{`{"type":"array","items":{"type":"integer","maximum":3},"contains":{"const":3},"minContains":2}`, 3, ``},
	} {
		v := mustOne(t, tc.schema, Options{Items: tc.items})
		if tc.want != "" && text(v) != tc.want {
			t.Errorf("%s with %d items: %s, want %s", tc.schema, tc.items, text(v), tc.want)
		}
	}
	v := mustOne(t, `{"type":"array","items":{"type":"integer","maximum":3},"contains":{"const":3},"minContains":2}`, Options{Items: 3})
	if a := v.(value.Array); text(a.At(0)) != "3" || text(a.At(1)) != "3" {
		t.Errorf("contains: %s", text(v))
	}
	v = mustOne(t, `{"type":"array","uniqueItems":true,"minItems":3,"items":{"type":"integer","minimum":1,"maximum":3}}`, Options{Items: 3})
	if a := v.(value.Array); a.Len() != 3 {
		t.Errorf("uniqueItems: %s", text(v))
	}
}

// TestConditionals is item 9: the if of every schema in C is evaluated
// against the value, the selected then or else is added, and what it
// newly requires is generated; dependentRequired and dependentSchemas
// take effect when their trigger is present.
func TestConditionals(t *testing.T) {
	v := mustOne(t, `{"type":"object","required":["kind"],"properties":{"kind":{"const":"person"}},
		"if":{"properties":{"kind":{"const":"person"}}},"then":{"required":["age"],"properties":{"age":{"type":"integer","minimum":18,"maximum":18}}},
		"else":{"required":["founded"]}}`, Options{Optional: None})
	if text(v) != `{"age":18,"kind":"person"}` {
		t.Errorf("then: %s", text(v))
	}
	v = mustOne(t, `{"type":"object","required":["a"],"properties":{"a":{"type":"string"}},"dependentRequired":{"a":["b"]},
		"dependentSchemas":{"a":{"properties":{"c":{"const":1}},"required":["c"]}}}`, Options{Optional: None})
	if member(v, "b") == nil || text(member(v, "c")) != "1" {
		t.Errorf("dependencies: %s", text(v))
	}
	v = mustOne(t, `{"type":"integer","minimum":1,"maximum":100,"if":{"minimum":50},"then":{"maximum":10},"else":{"minimum":1}}`, Options{})
	if n := v.(value.Number).Float64(); n >= 50 {
		t.Errorf("a scalar's then: %v", n)
	}
}

// TestOneOf is item 8: anyOf and oneOf pick a branch by the branch
// stream; a oneOf value matching other than one branch tries the next.
func TestOneOf(t *testing.T) {
	seen := map[string]bool{}
	each(t, `{"anyOf":[{"type":"string","maxLength":3},{"type":"integer"},{"type":"boolean"}]}`, 12, func(_ int, v value.Value) {
		seen[v.Kind().String()] = true
	})
	if len(seen) != 3 {
		t.Errorf("anyOf branches taken: %v", seen)
	}
	// Every value of the first branch matches the second too, so a pick
	// of it is retried to the second, whose example only it matches.
	each(t, `{"oneOf":[{"type":"integer","minimum":1},{"type":"integer","examples":[0]}]}`, 8, func(seed int, v value.Value) {
		if text(v) != "0" {
			t.Errorf("oneOf, seed %d: %s", seed, text(v))
		}
	})
	// A branch no value satisfies is passed over for the next.
	each(t, `{"anyOf":[{"type":"integer","minimum":5,"maximum":1},{"const":"ok"}]}`, 8, func(seed int, v value.Value) {
		if text(v) != `"ok"` {
			t.Errorf("anyOf, seed %d: %s", seed, text(v))
		}
	})
}

// TestRecursion is item 20: a $ref target entered for the fourth time on
// the path stops growing, so optional properties are omitted and arrays
// take minItems.
func TestRecursion(t *testing.T) {
	v := mustOne(t, `{"$ref":"#/$defs/node","$defs":{"node":{"type":"object","required":["name"],"properties":{"name":{"type":"string","const":"n"},"children":{"type":"array","items":{"$ref":"#/$defs/node"}}}}}}`, Options{Items: 1})
	depth := 0
	for cur := v; cur != nil; depth++ {
		kids, _ := member(cur, "children").(value.Array)
		if kids.Len() == 0 {
			break
		}
		cur = kids.At(0)
	}
	if depth != 3 {
		t.Errorf("the tree is %d deep: %s", depth, text(v))
	}
}

// TestNot is item 10: a value matching a not is made again from attempt
// streams.
func TestNot(t *testing.T) {
	each(t, `{"type":"integer","minimum":1,"maximum":3,"not":{"const":2}}`, 10, func(seed int, v value.Value) {
		if text(v) == "2" {
			t.Errorf("seed %d: 2", seed)
		}
	})
}
