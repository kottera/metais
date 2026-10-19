// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package args

import (
	"slices"
	"strings"
	"testing"
)

// testSpec has every shape a declaration can take: booleans with and
// without an alias, a valued flag with an alias, one with a closed set of
// values, a repeatable one, and two operands, the second optional.
var testSpec = Spec{
	Flags: []Flag{
		{Name: "json", Description: "JSON."},
		{Name: "help", Short: 'h', Description: "Help."},
		{Name: "recursive", Short: 'r', Description: "Recurse."},
		{Name: "context", Short: 'C', Value: "<n>", Description: "Context."},
		{Name: "color", Value: "<when>", Values: []string{"auto", "always", "never"}, Default: "auto", Description: "Colour."},
		{Name: "colour-depth", Value: "<n>", Description: "Shares the prefix colo with color."},
		{Name: "set", Value: "<pointer>=<text>", Repeatable: true, Description: "Set."},
	},
	Operands: []Operand{{Name: "<package>", Required: true}, {Name: "<path>"}},
}

type parseCase struct {
	name     string
	args     []string
	operands []string
	flags    map[string][]string
	errs     []Error
}

func TestParse(t *testing.T) {
	cases := []parseCase{
		// Interspersed flags: before, between and after operands.
		{name: "flags after operands", args: []string{"pkg", "dir", "--json", "-r"}, operands: []string{"pkg", "dir"}, flags: map[string][]string{"json": {""}, "recursive": {""}}},
		{name: "flags before operands", args: []string{"--json", "-r", "pkg", "dir"}, operands: []string{"pkg", "dir"}, flags: map[string][]string{"json": {""}, "recursive": {""}}},
		{name: "flags between operands", args: []string{"pkg", "--context", "3", "dir"}, operands: []string{"pkg", "dir"}, flags: map[string][]string{"context": {"3"}}},

		// -- ends the flags.
		{name: "-- makes a flag an operand", args: []string{"pkg", "--", "--json"}, operands: []string{"pkg", "--json"}, flags: map[string][]string{}},
		{name: "-- makes dash words operands", args: []string{"--", "-x", "-r"}, operands: []string{"-x", "-r"}, flags: map[string][]string{}},
		{name: "a second -- is an operand", args: []string{"pkg", "--", "--"}, operands: []string{"pkg", "--"}, flags: map[string][]string{}},
		{name: "flags before -- still count", args: []string{"--json", "--", "pkg"}, operands: []string{"pkg"}, flags: map[string][]string{"json": {""}}},
		{name: "- is an operand", args: []string{"-", "-r"}, operands: []string{"-"}, flags: map[string][]string{"recursive": {""}}},

		// The = and space forms.
		{name: "space form", args: []string{"pkg", "--color", "never"}, operands: []string{"pkg"}, flags: map[string][]string{"color": {"never"}}},
		{name: "= form", args: []string{"pkg", "--color=never"}, operands: []string{"pkg"}, flags: map[string][]string{"color": {"never"}}},
		{name: "= form with an empty value", args: []string{"pkg", "--context="}, operands: []string{"pkg"}, flags: map[string][]string{"context": {""}}},
		{name: "= form splits at the first =", args: []string{"pkg", "--set=/a=b=c"}, operands: []string{"pkg"}, flags: map[string][]string{"set": {"/a=b=c"}}},
		{name: "space form takes a dash value", args: []string{"pkg", "--context", "-3"}, operands: []string{"pkg"}, flags: map[string][]string{"context": {"-3"}}},
		{name: "space form takes --", args: []string{"pkg", "--context", "--"}, operands: []string{"pkg"}, flags: map[string][]string{"context": {"--"}}},
		{name: "a missing value", args: []string{"pkg", "--context"}, errs: []Error{{Kind: MissingValue, Arg: 1, Flag: "context"}}},
		{name: "a boolean with a value", args: []string{"pkg", "--json=true"}, errs: []Error{{Kind: BooleanValue, Arg: 1, Flag: "json"}}},
		{name: "a boolean with an empty value", args: []string{"pkg", "--json="}, errs: []Error{{Kind: BooleanValue, Arg: 1, Flag: "json"}}},
		{name: "a value outside the set", args: []string{"pkg", "--color", "purple"}, errs: []Error{{Kind: ValueNotAllowed, Arg: 1, Flag: "color"}}},
		{name: "a value outside the set, = form", args: []string{"pkg", "--color=Never"}, errs: []Error{{Kind: ValueNotAllowed, Arg: 1, Flag: "color"}}},

		// Short aliases.
		{name: "-h", args: []string{"-h"}, flags: map[string][]string{"help": {""}}, errs: []Error{{Kind: MissingOperand, Arg: 1, Operand: "<package>"}}},
		{name: "-C 3", args: []string{"pkg", "-C", "3"}, operands: []string{"pkg"}, flags: map[string][]string{"context": {"3"}}},
		{name: "-C3", args: []string{"pkg", "-C3"}, operands: []string{"pkg"}, flags: map[string][]string{"context": {"3"}}},
		{name: "-C with no value", args: []string{"pkg", "-C"}, errs: []Error{{Kind: MissingValue, Arg: 1, Flag: "context"}}},
		{name: "an undeclared alias", args: []string{"pkg", "-x"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},
		{name: "a long name with one dash", args: []string{"pkg", "-json"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},

		// Refused grouping.
		{name: "grouped aliases", args: []string{"pkg", "-rh"}, errs: []Error{{Kind: GroupedFlags, Arg: 1, Flag: "recursive"}}},
		{name: "a boolean alias with a letter after it", args: []string{"pkg", "-hr"}, errs: []Error{{Kind: GroupedFlags, Arg: 1, Flag: "help"}}},

		// Refused abbreviation.
		{name: "an abbreviation", args: []string{"pkg", "--rec"}, errs: []Error{{Kind: AbbreviatedFlag, Arg: 1, Flag: "recursive"}}},
		{name: "an abbreviation with a value", args: []string{"pkg", "--cont=3"}, errs: []Error{{Kind: AbbreviatedFlag, Arg: 1, Flag: "context"}}},
		{name: "an ambiguous abbreviation", args: []string{"pkg", "--colo"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},
		{name: "a longer name", args: []string{"pkg", "--jsonx"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},
		{name: "an unknown flag", args: []string{"pkg", "--bogus"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},
		{name: "--= names nothing", args: []string{"pkg", "--=x"}, errs: []Error{{Kind: UnknownFlag, Arg: 1}}},

		// Refused repetition.
		{name: "a repeated boolean", args: []string{"pkg", "--json", "--json"}, errs: []Error{{Kind: RepeatedFlag, Arg: 2, Flag: "json"}}},
		{name: "a repeated boolean through its alias", args: []string{"pkg", "--help", "-h"}, errs: []Error{{Kind: RepeatedFlag, Arg: 2, Flag: "help"}}},
		{name: "a repeated value, same value", args: []string{"pkg", "--color", "never", "--color=never"}, errs: []Error{{Kind: RepeatedFlag, Arg: 3, Flag: "color"}}},
		{name: "a repeated value through its alias", args: []string{"-C3", "pkg", "--context", "4"}, errs: []Error{{Kind: RepeatedFlag, Arg: 2, Flag: "context"}}},
		{name: "a repeatable flag", args: []string{"pkg", "--set", "/a=1", "--set=/b=2"}, operands: []string{"pkg"}, flags: map[string][]string{"set": {"/a=1", "/b=2"}}},

		// Operands.
		{name: "an extra operand", args: []string{"pkg", "dir", "more", "--json", "again"}, errs: []Error{{Kind: ExtraOperand, Arg: 2}, {Kind: ExtraOperand, Arg: 4}}},
		{name: "a missing operand", args: []string{"--json"}, errs: []Error{{Kind: MissingOperand, Arg: 1, Operand: "<package>"}}},
		{name: "no arguments", args: nil, errs: []Error{{Kind: MissingOperand, Arg: 0, Operand: "<package>"}}},

		// Every refusal at once, operand counts only when nothing else is refused.
		{name: "several refusals", args: []string{"--bogus", "pkg", "--json", "-ri", "--json", "a", "b", "c"}, errs: []Error{{Kind: UnknownFlag, Arg: 0}, {Kind: GroupedFlags, Arg: 3, Flag: "recursive"}, {Kind: RepeatedFlag, Arg: 4, Flag: "json"}}},
	}
	for _, c := range cases {
		p, errs := Parse(testSpec, c.args)
		if !slices.Equal(errs, c.errs) {
			t.Errorf("%s: Parse(%q) refused %+v, want %+v", c.name, c.args, errs, c.errs)
		}
		if c.errs != nil && c.flags == nil {
			continue
		}
		if !slices.Equal(p.Operands, c.operands) {
			t.Errorf("%s: operands %q, want %q", c.name, p.Operands, c.operands)
		}
		for _, f := range testSpec.Flags {
			want, given := c.flags[f.Name]
			if p.Has(f.Name) != given || !slices.Equal(p.Values(f.Name), want) {
				t.Errorf("%s: --%s is %q (given %v), want %q (given %v)", c.name, f.Name, p.Values(f.Name), p.Has(f.Name), want, given)
			}
		}
	}
}

func TestParsedAccessors(t *testing.T) {
	p, errs := Parse(testSpec, []string{"pkg", "--set", "/a=1", "--json", "--set=/b=2"})
	if errs != nil {
		t.Fatal(errs)
	}
	if v, ok := p.Value("set"); !ok || v != "/a=1" {
		t.Errorf("Value(set) = %q, %v", v, ok)
	}
	if _, ok := p.Value("color"); ok {
		t.Error("Value(color) found a flag not given")
	}
	if i, ok := p.Arg("set"); !ok || i != 1 {
		t.Errorf("Arg(set) = %d, %v, want 1", i, ok)
	}
	if i, ok := p.Arg("json"); !ok || i != 3 {
		t.Errorf("Arg(json) = %d, %v, want 3", i, ok)
	}
	vs := p.Values("set")
	vs[0] = "changed"
	if v, _ := p.Value("set"); v != "/a=1" {
		t.Error("Values returned the parse's own slice")
	}
	if at := p.ArgsOf("set"); !slices.Equal(at, []int{1, 4}) {
		t.Errorf("ArgsOf(set) = %v, want [1 4]", at)
	}
	if at := p.ArgsOf("color"); at != nil {
		t.Errorf("ArgsOf(color) = %v for a flag not given", at)
	}
}

// TestErrorsCarryNoArgumentText holds Parse to never putting a refused
// argument's text into what it returns: an Error names only a position and
// a declaration (ADR-006 D9 item 4).
func TestErrorsCarryNoArgumentText(t *testing.T) {
	const sentinel = "SENTINEL-c0ffee"
	for _, args := range [][]string{
		{"--" + sentinel}, {"-" + sentinel}, {"--json=" + sentinel}, {"--color", sentinel},
		{"pkg", "dir", sentinel}, {"--rec" + sentinel},
	} {
		_, errs := Parse(testSpec, args)
		if len(errs) == 0 {
			t.Errorf("Parse(%q) refused nothing", args)
		}
		for _, e := range errs {
			if strings.Contains(e.Flag+e.Operand, "SENTINEL") {
				t.Errorf("Parse(%q) echoed the argument: %+v", args, e)
			}
		}
	}
}

func TestRequestsJSON(t *testing.T) {
	for _, c := range []struct {
		args []string
		want bool
	}{
		{[]string{"--json"}, true},
		{[]string{"version", "--bogus", "--json", "-ri"}, true},
		{[]string{"x", "--color", "--json"}, true},
		{[]string{"--", "--json"}, false},
		{[]string{"--json=1"}, false},
		{[]string{"-json"}, false},
		{nil, false},
	} {
		if got := RequestsJSON(c.args); got != c.want {
			t.Errorf("RequestsJSON(%q) = %v, want %v", c.args, got, c.want)
		}
	}
}

func TestKindString(t *testing.T) {
	for k := UnknownFlag; k <= MissingOperand; k++ {
		if s := k.String(); s == "" || strings.HasPrefix(s, "Kind(") {
			t.Errorf("Kind %d has no name", k)
		}
	}
	if s := Kind(99).String(); s != "Kind(99)" {
		t.Errorf("Kind(99) = %q", s)
	}
}
