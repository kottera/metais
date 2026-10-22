// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package args is the metais argument parser (ADR-010 D10 item 3), with the
// syntax ADR-008 D3 states: long flags written --name, a value given as
// --name value or --name=value, a closed list of one-letter aliases that
// never group, flags interspersed with operands until the first --, no
// abbreviation, and a refusal of a single-valued flag given twice.
//
// The parser reads only the declarations it is given, so a flag that is not
// declared cannot be parsed, and the help printer reads the same
// declarations (ADR-008 D15).
package args

import (
	"slices"
	"strconv"
	"strings"
)

// Flag declares one flag.
type Flag struct {
	// Name is the long name, without its leading --.
	Name string
	// Short is the one-letter alias, written -<Short>, or 0 when the flag
	// has none.
	Short byte
	// Value is the placeholder of the flag's value in help, such as <n>;
	// it is empty for a boolean flag, which takes no value.
	Value string
	// Values, when not nil, is every value the flag accepts.
	Values []string
	// Default is the value in force when the flag is absent, as help
	// shows it; empty when there is none to show.
	Default string
	// Repeatable is set for a flag that may be given more than once.
	Repeatable bool
	// Required is set for a flag that must be given, such as unpack's
	// --out. Its absence is a MissingOperand that names the flag as the
	// usage line writes it, --name <value> (ADR-008 D13: a missing
	// operand or required flag).
	Required bool
	// Description is the flag's one-sentence help.
	Description string
}

// Boolean reports whether f takes no value.
func (f Flag) Boolean() bool { return f.Value == "" }

// Operand declares one positional argument.
type Operand struct {
	Name        string // the placeholder in help, such as <topic>
	Description string
	Required    bool
	Repeatable  bool // only the last operand may be repeatable
}

// Spec is everything one command line, or one part of one, may hold.
type Spec struct {
	Flags    []Flag
	Operands []Operand
}

// Kind is why an argument is refused.
type Kind uint8

const (
	// UnknownFlag is an argument that looks like a flag and names none
	// the spec declares.
	UnknownFlag Kind = iota + 1
	// AbbreviatedFlag is a long flag written as a prefix of one the spec
	// declares; no flag may be abbreviated (ADR-008 D1 item 3).
	AbbreviatedFlag
	// GroupedFlags is two or more one-letter aliases written together, as
	// -ri; aliases are never grouped (ADR-008 D3 item 2).
	GroupedFlags
	// MissingValue is a flag that takes a value written last, with none
	// after it.
	MissingValue
	// BooleanValue is a boolean flag written with =value.
	BooleanValue
	// ValueNotAllowed is a value outside the flag's Values.
	ValueNotAllowed
	// RepeatedFlag is a single-valued flag given a second time.
	RepeatedFlag
	// ExtraOperand is an operand beyond those the spec declares.
	ExtraOperand
	// MissingOperand is a required operand, or a required flag, not
	// given.
	MissingOperand
)

var kindNames = [...]string{
	UnknownFlag: "unknown_flag", AbbreviatedFlag: "abbreviated_flag", GroupedFlags: "grouped_flags",
	MissingValue: "missing_value", BooleanValue: "boolean_value", ValueNotAllowed: "value_not_allowed",
	RepeatedFlag: "repeated_flag", ExtraOperand: "extra_operand", MissingOperand: "missing_operand",
}

func (k Kind) String() string {
	if k >= UnknownFlag && k <= MissingOperand {
		return kindNames[k]
	}
	return "Kind(" + strconv.Itoa(int(k)) + ")"
}

// Error is one refused argument. It carries no text of the argument
// itself: only where it is and, when it concerns a declared flag or
// operand, that declaration's name, so a caller's argument is never
// echoed (ADR-006 D9 item 4, ADR-008 D11 item 5).
type Error struct {
	Kind Kind
	// Arg is the index of the refused argument in the slice Parse was
	// given; for MissingOperand it is the slice's length.
	Arg int
	// Flag is the long name of the declared flag the argument names or
	// abbreviates, or "" when it names none.
	Flag string
	// Operand is the name of the operand missing, for MissingOperand; a
	// required flag's is its usage, such as --out <dir>.
	Operand string
}

// Parsed is a parsed command line.
type Parsed struct {
	values   map[string][]string // by long name; a boolean flag holds one "" per occurrence
	first    map[string]int      // the index of each flag's first occurrence
	at       map[string][]int    // the index of each occurrence, in order
	Operands []string
	// OperandArgs holds the index of each operand in the slice Parse was
	// given.
	OperandArgs []int
}

// Has reports whether the flag called name was given.
func (p Parsed) Has(name string) bool {
	_, ok := p.values[name]
	return ok
}

// Value returns the value of the flag called name, and false when it was
// not given. For a repeatable flag it is the first value.
func (p Parsed) Value(name string) (string, bool) {
	v, ok := p.values[name]
	if !ok {
		return "", false
	}
	return v[0], true
}

// Values returns every value of the flag called name, in command-line
// order.
func (p Parsed) Values(name string) []string { return slices.Clone(p.values[name]) }

// Arg returns the index of the first occurrence of the flag called name,
// and false when it was not given.
func (p Parsed) Arg(name string) (int, bool) {
	i, ok := p.first[name]
	return i, ok
}

// ArgsOf returns the index of each occurrence of the flag called name,
// in command-line order: the occurrences Values holds the values of.
func (p Parsed) ArgsOf(name string) []int { return slices.Clone(p.at[name]) }

// Parse parses args against spec and returns every argument it refuses, in
// the order of the arguments, with the parse of everything it accepted.
//
// The first -- ends the flags: every argument after it is an operand, so
// a lone - is an operand wherever it is. Before it, an argument that
// begins with - and is longer than one byte is a flag. A flag that takes a
// value consumes the next argument whatever it is, as --name value writes
// it, so a value may itself begin with -.
//
// After an argument it cannot read, Parse goes on to the next one, so
// every refusal on a line is reported at once. The count of operands is
// checked only on a line with no other refusal: an unknown flag may have
// been meant to take the operand after it.
func Parse(spec Spec, args []string) (Parsed, []Error) {
	p := Parsed{values: map[string][]string{}, first: map[string]int{}, at: map[string][]int{}}
	var errs []Error
	add := func(f Flag, i int, v string) {
		if _, seen := p.values[f.Name]; seen && !f.Repeatable {
			errs = append(errs, Error{Kind: RepeatedFlag, Arg: i, Flag: f.Name})
			return
		}
		if f.Values != nil && !f.Boolean() && !slices.Contains(f.Values, v) {
			errs = append(errs, Error{Kind: ValueNotAllowed, Arg: i, Flag: f.Name})
			return
		}
		if _, seen := p.first[f.Name]; !seen {
			p.first[f.Name] = i
		}
		p.values[f.Name] = append(p.values[f.Name], v)
		p.at[f.Name] = append(p.at[f.Name], i)
	}
	ended := false
	for i := 0; i < len(args); i++ {
		a := args[i]
		switch {
		case ended || a == "-" || !strings.HasPrefix(a, "-"):
			p.Operands = append(p.Operands, a)
			p.OperandArgs = append(p.OperandArgs, i)
		case a == "--":
			ended = true
		case strings.HasPrefix(a, "--"):
			name, value, hasValue := strings.Cut(a[2:], "=")
			f, ok := lookupLong(spec, name)
			if !ok {
				e := Error{Kind: UnknownFlag, Arg: i}
				if full, ok := abbreviates(spec, name); ok {
					e = Error{Kind: AbbreviatedFlag, Arg: i, Flag: full}
				}
				errs = append(errs, e)
				continue
			}
			switch {
			case f.Boolean() && hasValue:
				errs = append(errs, Error{Kind: BooleanValue, Arg: i, Flag: f.Name})
			case f.Boolean():
				add(f, i, "")
			case hasValue:
				add(f, i, value)
			case i+1 < len(args):
				i++
				add(f, i-1, args[i])
			default:
				errs = append(errs, Error{Kind: MissingValue, Arg: i, Flag: f.Name})
			}
		default:
			f, ok := lookupShort(spec, a[1])
			switch {
			case !ok:
				errs = append(errs, Error{Kind: UnknownFlag, Arg: i})
			case f.Boolean() && len(a) > 2:
				// -ri: an alias that takes no value, with more after it.
				errs = append(errs, Error{Kind: GroupedFlags, Arg: i, Flag: f.Name})
			case f.Boolean():
				add(f, i, "")
			case len(a) > 2:
				// -C3: the value written against the alias.
				add(f, i, a[2:])
			case i+1 < len(args):
				i++
				add(f, i-1, args[i])
			default:
				errs = append(errs, Error{Kind: MissingValue, Arg: i, Flag: f.Name})
			}
		}
	}
	if len(errs) > 0 {
		return p, errs
	}
	n := len(spec.Operands)
	repeatable := n > 0 && spec.Operands[n-1].Repeatable
	if !repeatable && len(p.Operands) > n {
		for _, i := range p.OperandArgs[n:] {
			errs = append(errs, Error{Kind: ExtraOperand, Arg: i})
		}
	}
	for j := len(p.Operands); j < n; j++ {
		if spec.Operands[j].Required {
			errs = append(errs, Error{Kind: MissingOperand, Arg: len(args), Operand: spec.Operands[j].Name})
		}
	}
	for _, f := range spec.Flags {
		if f.Required && !p.Has(f.Name) {
			errs = append(errs, Error{Kind: MissingOperand, Arg: len(args), Operand: "--" + f.Name + " " + f.Value})
		}
	}
	return p, errs
}

func lookupLong(spec Spec, name string) (Flag, bool) {
	if name == "" {
		return Flag{}, false
	}
	for _, f := range spec.Flags {
		if f.Name == name {
			return f, true
		}
	}
	return Flag{}, false
}

func lookupShort(spec Spec, c byte) (Flag, bool) {
	for _, f := range spec.Flags {
		if f.Short != 0 && f.Short == c {
			return f, true
		}
	}
	return Flag{}, false
}

// abbreviates returns the one declared flag name is a proper prefix of,
// and false when it is a prefix of none or of more than one.
func abbreviates(spec Spec, name string) (string, bool) {
	if name == "" {
		return "", false
	}
	full := ""
	for _, f := range spec.Flags {
		if strings.HasPrefix(f.Name, name) {
			if full != "" {
				return "", false
			}
			full = f.Name
		}
	}
	return full, full != ""
}

// RequestsJSON reports whether --json is among args before the first --,
// which is how a line that fails to parse still gets its usage error as
// JSON (ADR-008 D10 item 7).
func RequestsJSON(args []string) bool {
	for _, a := range args {
		switch a {
		case "--":
			return false
		case "--json":
			return true
		}
	}
	return false
}
