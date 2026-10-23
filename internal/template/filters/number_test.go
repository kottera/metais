// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"bufio"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// TestNumber is MTS-FLT-015 to MTS-FLT-018: number's reading of its
// input, its rounding and its separators, each refusal at parse when its
// argument is a literal.
func TestNumber(t *testing.T) {
	testenv.Home(t)
	const d = `{"big": 1e21, "tiny": 5e-324, "f": 2.675, "neg": -0.004, "half": -2.5, "z": -0.0, "b": true, "a": [1], "o": {}, "n": null,
		"g": "|", "p": 2, "dec": 2, "code": "EUR"}`
	ok := func(name, src, want string) filterCase { return filterCase{name: name, src: src, data: d, want: want} }
	arg := func(name, src, argument string, atParse bool) filterCase {
		return filterCase{name: name, src: src, data: d, code: liquid.CodeFilterArgument, args: map[string]string{"filter": "number", "argument": argument}, atParse: atParse}
	}
	input := func(name, src, kind string) filterCase {
		return filterCase{name: name, src: src, data: d, code: liquid.CodeNumberInput, args: map[string]string{"filter": "number", "type": kind}}
	}
	runCases(t, []filterCase{
		// MTS-FLT-016: the shortest decimal, rounded half away from zero.
		ok("2.675 is the decimal 2.675", `{{ f | number: decimals: 2 }}`, "2.68"),
		ok("a literal 2.675", `{{ 2.675 | number: decimals: 2 }}`, "2.68"),
		ok("1.005", `{{ 1.005 | number: decimals: 2 }}`, "1.01"),
		ok("0.125", `{{ 0.125 | number: decimals: 2 }}`, "0.13"),
		ok("-2.5 away from zero", `{{ half | number: decimals: 0 }}`, "-3"),
		ok("-0.004 is written 0.00", `{{ neg | number: decimals: 2 }}`, "0.00"),
		ok("-0.004 with its own digits", `{{ neg | number }}`, "-0.004"),
		ok("-0 is 0", `{{ z | number }}`, "0"),
		ok("1e21 has 22 digits", `{{ big | number }}`, "1000000000000000000000"),
		ok("5e-324 written positionally", `{{ tiny | number }}`, "0."+strings.Repeat("0", 323)+"5"),
		ok("5e-324 at 20 decimals", `{{ tiny | number: decimals: 20 }}`, "0.00000000000000000000"),
		ok("trailing zeros removed", `{{ 2.50 | number }}`, "2.5"),
		ok("an integer", `{{ 48920000 | number }}`, "48920000"),
		ok("decimals pad", `{{ 3 | number: decimals: 3 }}`, "3.000"),
		ok("no point at 0 decimals", `{{ 2.4 | number: decimals: 0 }}`, "2"),
		ok("a carry", `{{ 9.995 | number: decimals: 2 }}`, "10.00"),
		ok("decimals from a variable", `{{ f | number: decimals: dec }}`, "2.68"),
		// MTS-FLT-015: a decimal text, exactly, at any length.
		ok("a decimal text", `{{ "-1234.50" | number }}`, "-1234.5"),
		ok("a decimal text beyond binary64", `{{ "12345678901234567.891" | number: decimals: 2, group: "," }}`, "12,345,678,901,234,567.89"),
		ok("a text halfway", `{{ "2.5" | number: decimals: 0 }}`, "3"),
		ok("a negative text halfway", `{{ "-0.125" | number: decimals: 2 }}`, "-0.13"),
		ok("a text below a half", `{{ "0.1249999999999999999999" | number: decimals: 2 }}`, "0.12"),
		ok("a text carrying into a new digit", `{{ "999.96" | number: decimals: 1, group: "," }}`, "1,000.0"),
		ok("a negative text rounding to zero", `{{ "-0.004" | number: decimals: 2 }}`, "0.00"),
		ok("-0 as text", `{{ "-0" | number }}`, "0"),
		ok("0 as text", `{{ "0.000" | number }}`, "0"),
		ok("nil gives nil", `{{ n | number }}|`, "|"),
		// MTS-FLT-017 and MTS-FLT-018: the separators.
		ok("a group", `{{ 48920000 | number: decimals: 2, group: "," }}`, "48,920,000.00"),
		ok("a space group and a comma point", `{{ 1234.5 | number: group: " ", point: "," }}`, "1 234,5"),
		ok("a group of two bytes", `{{ 1234567 | number: group: "’" }}`, "1’234’567"),
		ok("no group in three digits", `{{ 123 | number: group: "," }}`, "123"),
		ok("no group in the fraction", `{{ "1.234567" | number: group: "," }}`, "1.234567"),
		ok("an empty group", `{{ 1234 | number: group: "" }}`, "1234"),
		ok("a group from a variable", `{{ 1234 | number: group: g }}`, "1|234"),
		ok("the group the point would be, when point is given", `{{ 1234.5 | number: group: ".", point: "," }}`, "1.234,5"),
		ok("-2.5 to 0 decimals", `{{ -2.5 | number: decimals: 0 }}`, "-3"),
		// The refusals.
		input("a boolean", `{{ b | number }}`, "boolean"),
		input("an array", `{{ a | number }}`, "array"),
		input("an object", `{{ o | number }}`, "object"),
		input("a space", `{{ " 1" | number }}`, "string"),
		input("a plus", `{{ "+1" | number }}`, "string"),
		input("a leading zero", `{{ "01" | number }}`, "string"),
		input("an exponent", `{{ "1e3" | number }}`, "string"),
		input("a separator", `{{ "1,000" | number }}`, "string"),
		input("no digit after the point", `{{ "1." | number }}`, "string"),
		input("no digit before the point", `{{ ".5" | number }}`, "string"),
		input("a lone minus", `{{ "-" | number }}`, "string"),
		input("an empty string", `{{ "" | number }}`, "string"),
		arg("decimals 21", `{{ 1 | number: decimals: 21 }}`, "decimals", true),
		arg("decimals -1", `{{ 1 | number: decimals: -1 }}`, "decimals", true),
		arg("decimals a float", `{{ 1 | number: decimals: 2.0 }}`, "decimals", true),
		arg("decimals a string", `{{ 1 | number: decimals: "2" }}`, "decimals", true),
		arg("decimals at render", `{{ 1 | number: decimals: code }}`, "decimals", false),
		arg("a two-character point", `{{ 1 | number: point: ".." }}`, "point", true),
		arg("an empty point", `{{ 1 | number: point: "" }}`, "point", true),
		arg("a digit point", `{{ 1 | number: point: "1" }}`, "point", true),
		arg("a minus point", `{{ 1 | number: point: "-" }}`, "point", true),
		arg("a minus group", `{{ 1 | number: group: "-" }}`, "group", true),
		arg("a digit group", `{{ 1 | number: group: "0" }}`, "group", true),
		arg("a group that is the point", `{{ 1 | number: group: "." }}`, "group", true),
		arg("a group that is the point given", `{{ 1 | number: group: ",", point: "," }}`, "group", true),
		arg("a group that is the point at render", `{{ 1 | number: group: g, point: "|" }}`, "group", false),
		arg("a group not a string", `{{ 1 | number: group: p }}`, "group", false),
		arg("a positional argument", `{{ 1 | number: 2 }}`, "1", true),
		arg("an unknown keyword", `{{ 1 | number: code: "after" }}`, "code", true),
		arg("a keyword twice", `{{ 1 | number: decimals: 1, decimals: 2 }}`, "decimals", true),
	})
}

// TestAmount is MTS-FLT-019 and MTS-FLT-020: the code and the minor units
// of Appendix C, and the code's refusals, a literal's at parse.
func TestAmount(t *testing.T) {
	testenv.Home(t)
	const d = `{"eur": "EUR", "xau": "XAU", "low": "eur", "zzz": "ZZZ", "two": 2, "n": null}`
	ok := func(name, src, want string) filterCase { return filterCase{name: name, src: src, data: d, want: want} }
	cur := func(name, src, reason string, atParse bool) filterCase {
		return filterCase{name: name, src: src, data: d, code: liquid.CodeAmountCurrency, args: map[string]string{"filter": "amount", "reason": reason}, atParse: atParse}
	}
	arg := func(name, src, argument string, atParse bool) filterCase {
		return filterCase{name: name, src: src, data: d, code: liquid.CodeFilterArgument, args: map[string]string{"filter": "amount", "argument": argument}, atParse: atParse}
	}
	runCases(t, []filterCase{
		ok("EUR", `{{ 48920000 | amount: "EUR", group: "," }}`, "EUR 48,920,000.00"),
		ok("JPY", `{{ 5000 | amount: "JPY" }}`, "JPY 5000"),
		ok("BHD", `{{ 1.5 | amount: "BHD" }}`, "BHD 1.500"),
		ok("CLF", `{{ 1.23456 | amount: "CLF" }}`, "CLF 1.2346"),
		ok("XAU with decimals", `{{ 2.5 | amount: "XAU", decimals: 3 }}`, "XAU 2.500"),
		ok("a code Appendix C lacks, with decimals", `{{ 2 | amount: "ZZZ", decimals: 1 }}`, "ZZZ 2.0"),
		ok("decimals over the table", `{{ 2.675 | amount: "JPY", decimals: 2 }}`, "JPY 2.68"),
		ok("after", `{{ -5 | amount: "EUR", code: "after" }}`, "-5.00 EUR"),
		ok("before", `{{ -5 | amount: "EUR", code: "before" }}`, "EUR -5.00"),
		ok("a code from the data", `{{ 2.675 | amount: eur }}`, "EUR 2.68"),
		ok("a point and a group", `{{ "1234567.891" | amount: eur, group: ".", point: "," }}`, "EUR 1.234.567,89"),
		ok("nil gives nil", `{{ n | amount: "EUR" }}|`, "|"),
		cur("lower case", `{{ 1 | amount: "eur" }}`, "malformed", true),
		cur("four letters", `{{ 1 | amount: "EURO" }}`, "malformed", true),
		cur("malformed even with decimals", `{{ 1 | amount: "E1R", decimals: 2 }}`, "malformed", true),
		cur("empty", `{{ 1 | amount: "" }}`, "malformed", true),
		cur("not listed", `{{ 1 | amount: "ZZZ" }}`, "no_minor_unit", true),
		cur("no minor unit", `{{ 1 | amount: "XAU" }}`, "no_minor_unit", true),
		ok("any three capitals with decimals from the data", `{{ 1 | amount: "XXX", decimals: two }}`, "XXX 1.00"),
		cur("lower case at render", `{{ 1 | amount: low }}`, "malformed", false),
		cur("not listed at render", `{{ 1 | amount: zzz }}`, "no_minor_unit", false),
		cur("no minor unit at render", `{{ 1 | amount: xau }}`, "no_minor_unit", false),
		cur("the code is judged before the input", `{{ n | amount: xau }}`, "no_minor_unit", false),
		arg("no code", `{{ 1 | amount }}`, "1", true),
		arg("two codes", `{{ 1 | amount: "EUR", "USD" }}`, "2", true),
		arg("a code not a string", `{{ 1 | amount: 978 }}`, "1", true),
		arg("a code not a string at render", `{{ 1 | amount: two }}`, "1", false),
		arg("code neither before nor after", `{{ 1 | amount: "EUR", code: "first" }}`, "code", true),
		arg("decimals 21", `{{ 1 | amount: "EUR", decimals: 21 }}`, "decimals", true),
		{name: "a string not a decimal text", src: `{{ "1 000" | amount: "EUR" }}`, data: d, code: liquid.CodeNumberInput, args: map[string]string{"filter": "amount", "type": "string"}},
	})
}

// TestNumberAgreesWithRound is ADR-004 A2 ruling 7: x | round: n and
// x | number: decimals: n never disagree, since both round the shortest
// decimal of x half away from zero. The values are drawn near halves,
// where a rounding of the binary value parts from it.
func TestNumberAgreesWithRound(t *testing.T) {
	testenv.Home(t)
	rng := rand.New(rand.NewPCG(4217, 2026))
	for range 2000 {
		n := rng.IntN(7)
		// k + 0.5, at n places, scaled down: 2.675 is 2675·10⁻³.
		k := rng.Int64N(2_000_000) - 1_000_000
		x, _ := strconv.ParseFloat(strconv.FormatInt(k*10+5, 10)+"e-"+strconv.Itoa(n+1), 64)
		src := "{{ x | round: " + strconv.Itoa(n) + " }} {{ x | number: decimals: " + strconv.Itoa(n) + " }}"
		out, err, _ := render(t, src, `{"x": `+strconv.FormatFloat(x, 'g', -1, 64)+`}`)
		if err != nil {
			t.Fatalf("%v: %v", x, err)
		}
		r, num, _ := strings.Cut(out, " ")
		a, _ := strconv.ParseFloat(r, 64)
		b, _ := strconv.ParseFloat(num, 64)
		if a != b {
			t.Errorf("%v at %d: round writes %s, number %s", x, n, r, num)
		}
	}
}

// TestNumberNotFormatFloat pins the property false first: the values the
// specification names, each of which a rounding of the binary64's exact
// value, FormatFloat(x, 'f', n, 64), writes otherwise.
func TestNumberNotFormatFloat(t *testing.T) {
	for _, c := range []struct {
		x    float64
		n    int
		want string
	}{{2.675, 2, "2.68"}, {1.005, 2, "1.01"}, {0.125, 2, "0.13"}, {-2.5, 0, "-3"}} {
		got := numberDigits(c.x, c.n).write(".", "")
		if got != c.want {
			t.Errorf("%v at %d: %s, want %s", c.x, c.n, got, c.want)
		}
		if f := strconv.FormatFloat(c.x, 'f', c.n, 64); f == c.want {
			t.Errorf("%v at %d: FormatFloat writes %s too, so the case proves nothing", c.x, c.n, f)
		}
	}
}

// TestTextDigitsMatchNumberDigits holds the two readings to one value:
// the decimal text of a number's shortest decimal, written positionally,
// rounds as the number does, at every count of decimals.
func TestTextDigitsMatchNumberDigits(t *testing.T) {
	rng := rand.New(rand.NewPCG(15, 16))
	xs := []float64{0, 1, -1, 2.675, 1.005, 0.125, -2.5, -0.004, 1e21, 5e-324, math.MaxFloat64, 9.995, 0.5}
	for range 500 {
		xs = append(xs, (rng.Float64()-0.5)*math.Pow(10, float64(rng.IntN(40)-20)))
	}
	for _, x := range xs {
		own := numberDigits(x, -1)
		text := own.whole
		if own.frac != "" {
			text += "." + own.frac
		}
		if own.neg {
			text = "-" + text
		}
		for n := -1; n <= maxDecimals; n++ {
			want := numberDigits(x, n).write(".", ",")
			d, ok := textDigits(text, n)
			if !ok {
				t.Fatalf("%v: %q is not a decimal text", x, text)
			}
			if got := d.write(".", ","); got != want {
				t.Errorf("%v at %d: the text %s writes %s, the number %s", x, n, text, got, want)
			}
		}
	}
}

// TestCurrencyTable is ADR-004 A2's implementation note: the Go table is
// Appendix C, read from the appendix, code for code, and no code more.
func TestCurrencyTable(t *testing.T) {
	f, err := os.Open(filepath.Join("..", "..", "..", "spec", "C-currency-minor-units.md"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	row := regexp.MustCompile("^\\| `([A-Z]{3})` \\| ([0-9]|—) \\| [^|]+ \\|$")
	appendix := map[string]int{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if !strings.HasPrefix(l, "| `") {
			continue
		}
		m := row.FindStringSubmatch(l)
		if m == nil {
			t.Errorf("the appendix's row %q is not a code, its minor units and a name", l)
			continue
		}
		u := noMinorUnit
		if m[2] != "—" {
			u, _ = strconv.Atoi(m[2])
		}
		if _, dup := appendix[m[1]]; dup {
			t.Errorf("the appendix lists %s twice", m[1])
		}
		appendix[m[1]] = u
	}
	if err := sc.Err(); err != nil {
		t.Fatal(err)
	}
	if len(appendix) != 178 {
		t.Errorf("the appendix lists %d codes, and ADR-004 A2 ruling 5 counts 178", len(appendix))
	}
	for code, u := range appendix {
		if got, ok := minorUnits[code]; !ok || got != u {
			t.Errorf("%s: the table gives %d (listed %v), the appendix %d", code, got, ok, u)
		}
	}
	for code := range minorUnits {
		if _, ok := appendix[code]; !ok {
			t.Errorf("%s is in the table and not in the appendix", code)
		}
	}
}

// TestNumberCharges is MTS-TPL-063's row for number and amount, counted
// on the meter: the shallow sizes of the input and the arguments, then
// the result's length, or nil's shallow size; one unit fewer is refused
// at the last charge.
func TestNumberCharges(t *testing.T) {
	testenv.Home(t)
	str := func(s string) value.Value { v, _ := value.NewString(s); return v }
	num := func(s string) value.Value { v, _ := value.ParseNumber(s); return v }
	cases := []struct {
		filter string
		in     value.Value
		args   []liquid.Arg
		units  int64
	}{
		// 1 + 1 + 1 + 13 for "48,920,000.00"
		{"number", num("48920000"), []liquid.Arg{{Name: "decimals", Value: num("2")}, {Name: "group", Value: str(",")}}, 1 + 1 + 1 + 13},
		// 21 for the text + 21 for "12345678901234567.891"
		{"number", str("12345678901234567.891"), nil, 21 + 21},
		// 1 + 3 + 17 for "EUR 48,920,000.00"
		{"amount", num("48920000"), []liquid.Arg{{Value: str("EUR")}, {Name: "group", Value: str(",")}}, 1 + 3 + 1 + 17},
		// 1 for nil, 3 for the code, and 1 for nil, the result
		{"amount", value.Null{}, []liquid.Arg{{Value: str("EUR")}}, 1 + 3 + 1},
		{"number", value.Null{}, nil, 1 + 1},
	}
	for i, c := range cases {
		f, _ := dialect.Filter(c.filter)
		m := liquid.NewMeter(liquid.MeterLimits{})
		if _, err := f.Apply(meter{m}, c.in, c.args); err != nil {
			t.Errorf("%d %s: %v", i, c.filter, err)
			continue
		}
		if got := m.Used(liquid.WorkUnits); got != c.units {
			t.Errorf("%d %s: %d units, want %d", i, c.filter, got, c.units)
		}
		if got := m.Used(liquid.WorkHeldBytes); got != 0 {
			t.Errorf("%d %s: %d held bytes, want 0", i, c.filter, got)
		}
		m = liquid.NewMeter(liquid.MeterLimits{RenderWork: c.units - 1})
		if _, err := f.Apply(meter{m}, c.in, c.args); err == nil || err.Code != liquid.CodeLimit || err.Args["limit"] != "render-work" {
			t.Errorf("%d %s with render-work %d: %v, want the limit", i, c.filter, c.units-1, err)
		}
	}
}
