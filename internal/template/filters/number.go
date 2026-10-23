// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package filters

import (
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"unicode/utf8"

	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// numberFilter is number or amount (ADR-004 A2 decision items 1 to 3,
// MTS-FLT-015 to MTS-FLT-020): a number, or a decimal text, written for a
// reader with a chosen count of decimals, a point and a group separator,
// and, for amount, with its ISO 4217 code. Neither reads a locale: every
// separator is an argument, and what an argument left out means is the
// same in every package. nil gives nil.
//
// A literal argument is checked when the template is parsed, amount's
// code included. Every refusal is a *liquid.Error: CodeFilterArgument for
// an argument count, a keyword or an argument's value it never takes,
// CodeAmountCurrency for a code, and CodeNumberInput for an input.
//
// Its work is charged as MTS-TPL-063 says: the shallow sizes of the input
// and of every argument before anything else, then the result's byte
// length, or nil's shallow size, as work units, once the result is made.
// Formatting is linear in the input's text and in the result, both
// charged, so no work goes uncharged.
type numberFilter struct{ amount bool }

var (
	numberFilterDef = numberFilter{}
	amountFilterDef = numberFilter{amount: true}
)

// maxDecimals is the most decimals: takes (MTS-FLT-017).
const maxDecimals = 20

func (f numberFilter) Name() string {
	if f.amount {
		return "amount"
	}
	return "number"
}

// knows reports whether the filter takes the keyword k.
func (f numberFilter) knows(k string) bool {
	switch k {
	case "decimals", "group", "point":
		return true
	case "code":
		return f.amount
	}
	return false
}

// shape refuses pos positional arguments and keywords the filter does not
// take: number takes none, amount one, the code.
func (f numberFilter) shape(pos int, keywords []string) *liquid.Error {
	name := f.Name()
	seen := map[string]bool{}
	for _, k := range keywords {
		if !f.knows(k) {
			return argErr(name, k, fmt.Sprintf("%s takes no keyword %q", name, k))
		}
		if seen[k] {
			return argErr(name, k, fmt.Sprintf("keyword %q given twice", k))
		}
		seen[k] = true
	}
	switch {
	case !f.amount && pos > 0:
		return argErr(name, strconv.Itoa(pos), fmt.Sprintf("number takes no positional argument, not %d", pos))
	case f.amount && pos == 0:
		return argErr(name, "1", "amount takes one argument, the code, not 0")
	case f.amount && pos > 1:
		return argErr(name, strconv.Itoa(pos), fmt.Sprintf("amount takes one argument, the code, not %d", pos))
	}
	return nil
}

// Check refuses an application that is wrong whatever the data: an
// argument count it never takes, a keyword it does not know or given
// twice, a literal argument it refuses, and a literal code amount
// refuses (MTS-FLT-020).
func (f numberFilter) Check(app *ast.Filter) *liquid.Error {
	var pos int
	var keywords []string
	for _, a := range app.Args {
		if a.Name == "" {
			pos++
		} else {
			keywords = append(keywords, a.Name)
		}
	}
	if err := f.shape(pos, keywords); err != nil {
		return err
	}
	lits := map[string]value.Value{}
	var code value.Value
	for _, a := range app.Args {
		lit, ok := a.Value.(*ast.Literal)
		if !ok {
			if a.Name != "" {
				lits[a.Name] = nil
			}
			continue
		}
		v := lit.Value
		if lit.Kind == ast.LitEmpty || lit.Kind == ast.LitBlank {
			// As a value outside a comparison, each is "" (ADR-004 D5
			// item 2).
			v = value.String{}
		}
		if a.Name == "" {
			code = v
		} else {
			lits[a.Name] = v
		}
		if v == nil {
			continue
		}
		if err := f.checkArg(a.Name, v); err != nil {
			return err
		}
	}
	if err := f.checkGroup(lits); err != nil {
		return err
	}
	if s, ok := code.(value.String); ok {
		_, decimals := lits["decimals"]
		if _, err := f.currency(s.Text(), decimals); err != nil {
			return err
		}
	}
	return nil
}

// checkArg refuses the argument v, the code when keyword is empty.
func (f numberFilter) checkArg(keyword string, v value.Value) *liquid.Error {
	name := f.Name()
	switch keyword {
	case "":
		if _, ok := v.(value.String); !ok {
			return argErr(name, "1", "the code is a "+liquid.KindName(v)+", not a string")
		}
	case "decimals":
		if _, ok := decimalsOf(v); !ok {
			return argErr(name, keyword, fmt.Sprintf("decimals is an integer from 0 to %d, not this %s", maxDecimals, liquid.KindName(v)))
		}
	case "point":
		if _, ok := separator(v, false); !ok {
			return argErr(name, keyword, "point is one character other than a digit and -")
		}
	case "group":
		if _, ok := separator(v, true); !ok {
			return argErr(name, keyword, "group is empty or one character other than a digit and -")
		}
	case "code":
		if s, ok := v.(value.String); !ok || s.Text() != "before" && s.Text() != "after" {
			return argErr(name, keyword, `code is "before" or "after"`)
		}
	}
	return nil
}

// checkGroup refuses a group separator equal to the point, which is "."
// when point: is not given (MTS-FLT-017). kw holds the keywords given,
// each with its value, or nil when it is not known; one not known is not
// judged.
func (f numberFilter) checkGroup(kw map[string]value.Value) *liquid.Error {
	g, ok := kw["group"].(value.String)
	if !ok || g.Text() == "" {
		return nil
	}
	point := "."
	if v, given := kw["point"]; given {
		p, ok := v.(value.String)
		if !ok {
			return nil
		}
		point = p.Text()
	}
	if g.Text() == point {
		return argErr(f.Name(), "group", "group is not the point")
	}
	return nil
}

// currency is the count of decimals the code gives, by Appendix C, or the
// refusal of MTS-FLT-020: a code that is not three ASCII capital letters,
// and, when decimals: is not given, one Appendix C does not list or lists
// with no minor units. With decimals: given, units is not read. The error
// never holds the code (ADR-004 D16 item 3).
func (f numberFilter) currency(code string, decimals bool) (units int, err *liquid.Error) {
	if len(code) != 3 || !capital(code[0]) || !capital(code[1]) || !capital(code[2]) {
		return 0, liquid.FilterError(f.Name(), liquid.CodeAmountCurrency, map[string]string{"reason": "malformed"},
			"a currency code is three ASCII capital letters")
	}
	if decimals {
		return 0, nil
	}
	u, ok := minorUnits[code]
	if !ok || u == noMinorUnit {
		return 0, liquid.FilterError(f.Name(), liquid.CodeAmountCurrency, map[string]string{"reason": "no_minor_unit"},
			"Appendix C gives the code no minor units, so decimals: gives the count")
	}
	return u, nil
}

func capital(c byte) bool { return 'A' <= c && c <= 'Z' }

// decimalsOf is decimals:'s value, an integer from 0 to maxDecimals.
func decimalsOf(v value.Value) (int, bool) {
	n, ok := v.(value.Number)
	if !ok {
		return 0, false
	}
	i, ok := liquid.Integer(n)
	if !ok || i < 0 || i > maxDecimals {
		return 0, false
	}
	return int(i), true
}

// separator is point:'s or group:'s value: one code point other than an
// ASCII digit and -, or, for group: (empty true), the empty string.
func separator(v value.Value, empty bool) (string, bool) {
	s, ok := v.(value.String)
	if !ok {
		return "", false
	}
	t := s.Text()
	if t == "" {
		return "", empty
	}
	r, n := utf8.DecodeRuneInString(t)
	if n != len(t) || r == '-' || '0' <= r && r <= '9' {
		return "", false
	}
	return t, true
}

func (f numberFilter) Apply(c liquid.FilterContext, in value.Value, args []liquid.Arg) (value.Value, *liquid.Error) {
	cl := &call{name: f.Name(), m: c.Meter(), kw: map[string]value.Value{}}
	var keywords []string
	work := liquid.Shallow(in)
	for _, a := range args {
		work += liquid.Shallow(a.Value)
		if a.Name == "" {
			cl.pos = append(cl.pos, a.Value)
			continue
		}
		keywords = append(keywords, a.Name)
		cl.kw[a.Name] = a.Value
	}
	if err := f.shape(len(cl.pos), keywords); err != nil {
		return nil, err
	}
	if err := cl.units(work); err != nil {
		return nil, err
	}
	for _, a := range args {
		if err := f.checkArg(a.Name, a.Value); err != nil {
			return nil, err
		}
	}
	if err := f.checkGroup(cl.kw); err != nil {
		return nil, err
	}
	decimals := -1
	if v, ok := cl.kw["decimals"]; ok {
		decimals, _ = decimalsOf(v)
	}
	var code string
	if f.amount {
		code = cl.pos[0].(value.String).Text()
		u, err := f.currency(code, decimals >= 0)
		if err != nil {
			return nil, err
		}
		if decimals < 0 {
			decimals = u
		}
	}
	d, isNil, err := f.read(in, decimals)
	if err != nil {
		return nil, err
	}
	if isNil {
		if err := cl.units(1); err != nil {
			return nil, err
		}
		return value.Null{}, nil
	}
	point, group := ".", ""
	if v, ok := cl.kw["point"]; ok {
		point, _ = separator(v, false)
	}
	if v, ok := cl.kw["group"]; ok {
		group, _ = separator(v, true)
	}
	num := d.write(point, group)
	s := num
	if f.amount {
		s = code + " " + num
		if v, ok := cl.kw["code"]; ok && v.(value.String).Text() == "after" {
			s = num + " " + code
		}
	}
	return cl.text(int64(len(s)), func() string { return s })
}

// digits is a decimal value as it is to be written: below zero or not,
// and the digits of its magnitude's integer part, with no leading zero but
// a lone 0, and of its fraction.
type digits struct {
	neg         bool
	whole, frac string
}

// read is the value of in (MTS-FLT-015, MTS-FLT-016), rounded to decimals
// digits after the point, or with its own digits when decimals is below
// zero. isNil is true for nil; any other input that is neither a number
// nor a decimal text is CodeNumberInput.
func (f numberFilter) read(in value.Value, decimals int) (d digits, isNil bool, err *liquid.Error) {
	switch v := in.(type) {
	case nil, value.Null:
		return digits{}, true, nil
	case value.Number:
		return numberDigits(v.Float64(), decimals), false, nil
	case value.String:
		if d, ok := textDigits(v.Text(), decimals); ok {
			return d, false, nil
		}
	}
	return digits{}, false, liquid.FilterError(f.Name(), liquid.CodeNumberInput, map[string]string{"type": liquid.KindName(in)},
		f.Name()+" takes a number or a decimal text, not this "+liquid.KindName(in))
}

// numberDigits is the value of the number x: the shortest decimal that
// reads back as x, the decimal round computes on (MTS-TPL-018), rounded
// on big.Rat. Rat.FloatString rounds its last digit to nearest, halves
// away from zero, as round does; so 2.675 at two decimals is 2.68, where
// FormatFloat(x, 'f', 2, 64), rounding x's binary value, writes 2.67.
func numberDigits(x float64, decimals int) digits {
	g := strconv.FormatFloat(x, 'g', -1, 64)
	r, ok := new(big.Rat).SetString(g)
	if !ok {
		panic("filters: unreadable float text " + g)
	}
	places := decimals
	if places < 0 {
		places = fractionDigits(g)
	}
	neg := r.Sign() < 0
	whole, frac, _ := strings.Cut(r.Abs(r).FloatString(places), ".")
	return digits{neg: neg, whole: whole, frac: frac}
}

// fractionDigits is the count of digits after the point of the decimal g,
// as FormatFloat's 'g' writes it: the mantissa's less the exponent.
func fractionDigits(g string) int {
	m, e, _ := strings.Cut(g, "e")
	n := 0
	if _, f, ok := strings.Cut(m, "."); ok {
		n = len(f)
	}
	if e != "" {
		exp, err := strconv.Atoi(e)
		if err != nil {
			panic("filters: unreadable exponent " + g)
		}
		n -= exp
	}
	return max(n, 0)
}

// textDigits is the value of s when it is a decimal text (MTS-FLT-015):
// an optional -, then 0 or a digit 1 to 9 followed by any digits, then
// optionally . and one or more digits, and nothing else; ok is false for
// any other string. The decimal is the one s spells, exactly, at any
// length, so it is rounded on its digits, in time linear in s, which the
// input's shallow size has charged: a digit from 5 up after the last one
// kept rounds the magnitude up, which is half away from zero.
func textDigits(s string, decimals int) (d digits, ok bool) {
	t := s
	if strings.HasPrefix(t, "-") {
		d.neg, t = true, t[1:]
	}
	d.whole, d.frac, _ = strings.Cut(t, ".")
	switch {
	case d.whole == "" || !allDigits(d.whole) || len(d.whole) > 1 && d.whole[0] == '0':
		return digits{}, false
	case strings.Contains(t, ".") && (d.frac == "" || !allDigits(d.frac)):
		return digits{}, false
	}
	switch {
	case decimals < 0:
		d.frac = strings.TrimRight(d.frac, "0")
	case decimals >= len(d.frac):
		d.frac += strings.Repeat("0", decimals-len(d.frac))
	default:
		up := d.frac[decimals] >= '5'
		d.frac = d.frac[:decimals]
		if up {
			d.whole, d.frac = increment(d.whole, d.frac)
		}
	}
	return d, true
}

// increment adds one in the last place of the decimal whole.frac.
func increment(whole, frac string) (string, string) {
	b := []byte(whole + frac)
	i := len(b) - 1
	for ; i >= 0 && b[i] == '9'; i-- {
		b[i] = '0'
	}
	if i < 0 {
		b = append([]byte{'1'}, b...)
	} else {
		b[i]++
	}
	n := len(b) - len(frac)
	return string(b[:n]), string(b[n:])
}

// write is d's text (MTS-FLT-016, MTS-FLT-017): - when the value written
// is below zero, the integer part with group between two of its digits
// wherever the digits to the right number a multiple of three, and the
// point and the fraction when a digit follows the point. It never writes
// an exponent.
func (d digits) write(point, group string) string {
	var b strings.Builder
	if d.neg && strings.Trim(d.whole+d.frac, "0") != "" {
		b.WriteByte('-')
	}
	for i := 0; i < len(d.whole); i++ {
		if group != "" && i > 0 && (len(d.whole)-i)%3 == 0 {
			b.WriteString(group)
		}
		b.WriteByte(d.whole[i])
	}
	if d.frac != "" {
		b.WriteString(point)
		b.WriteString(d.frac)
	}
	return b.String()
}
