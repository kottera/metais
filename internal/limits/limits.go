// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package limits is ADR-009 D3's table of resource limits, with their
// exact defaults, and D4's rule for lowering one. A limit is never raised:
// no flag, file or environment variable can take one above its default
// (RULINGS R1-015).
package limits

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kottera/metais/internal/finding"
)

// ID names one limit.
type ID int

// The limits, in ADR-009 D3's table order.
const (
	PackageSize ID = iota
	PackageUncompressed
	Entries
	EntrySize
	EntryRatio
	PathLength
	PathDepth
	ManifestSize
	SchemaSize
	CombinedSchemaSize
	SchemaDepth
	RefDepth
	PatternLength
	RegexProgram
	TemplateSize
	TemplateNesting
	PartialDepth
	ValueDepth
	NumberLength

	InputSize
	InputsTotal
	CSVRows
	CSVColumns
	CSVCell
	CollectionItems
	FrontmatterSize

	RenderWork
	LoopIterations
	ValueSize
	ArraySize
	AssignBytes
	SchemaWork
	RegexWork
	Outputs
	ContainerDepth
	OutputSize
	OutputTotal
	Findings
	GrepMatches
	GrepContext

	count
)

// Unit is what a limit counts.
type Unit uint8

const (
	// Bytes is a size. A size limit's value may carry a KiB, MiB or GiB
	// suffix (ADR-009 D4 item 1).
	Bytes Unit = iota + 1
	// Count is a number of things: entries, path components, levels of
	// nesting, iterations, work units, findings.
	Count
	// Ratio is a quotient: uncompressed size over compressed size.
	Ratio
)

func (u Unit) String() string {
	switch u {
	case Bytes:
		return "bytes"
	case Count:
		return "count"
	case Ratio:
		return "ratio"
	}
	return fmt.Sprintf("Unit(%d)", uint8(u))
}

// Class is which of ADR-009 D3's tables a limit is in.
type Class uint8

const (
	// ClassPackage bounds what a package may use; a package within every
	// one is portable.
	ClassPackage Class = iota + 1
	// ClassData bounds the inputs of one command.
	ClassData
	// ClassWork bounds what one command computes and writes.
	ClassWork
)

func (c Class) String() string {
	switch c {
	case ClassPackage:
		return "package"
	case ClassData:
		return "data"
	case ClassWork:
		return "work"
	}
	return fmt.Sprintf("Class(%d)", uint8(c))
}

const (
	kib = 1 << 10
	mib = 1 << 20
	gib = 1 << 30
)

// Limit is one row of the table.
type Limit struct {
	ID      ID
	Name    string
	Default int64
	Unit    Unit
	Class   Class
}

// table is ADR-009 D3's table, exact (D3 item 2), indexed by ID.
var table = [count]Limit{
	{PackageSize, "package-size", 256 * mib, Bytes, ClassPackage},
	{PackageUncompressed, "package-uncompressed", 512 * mib, Bytes, ClassPackage},
	{Entries, "entries", 10_000, Count, ClassPackage},
	{EntrySize, "entry-size", 64 * mib, Bytes, ClassPackage},
	{EntryRatio, "entry-ratio", 200, Ratio, ClassPackage},
	{PathLength, "path-length", 512, Bytes, ClassPackage},
	{PathDepth, "path-depth", 32, Count, ClassPackage},
	{ManifestSize, "manifest-size", 1 * mib, Bytes, ClassPackage},
	{SchemaSize, "schema-size", 4 * mib, Bytes, ClassPackage},
	{CombinedSchemaSize, "combined-schema-size", 16 * mib, Bytes, ClassPackage},
	{SchemaDepth, "schema-depth", 64, Count, ClassPackage},
	{RefDepth, "ref-depth", 32, Count, ClassPackage},
	{PatternLength, "pattern-length", 4_096, Bytes, ClassPackage},
	{RegexProgram, "regex-program", 4_096, Count, ClassPackage},
	{TemplateSize, "template-size", 8 * mib, Bytes, ClassPackage},
	{TemplateNesting, "template-nesting", 100, Count, ClassPackage},
	{PartialDepth, "partial-depth", 32, Count, ClassPackage},
	{ValueDepth, "value-depth", 64, Count, ClassPackage},
	{NumberLength, "number-length", 1_024, Bytes, ClassPackage},

	{InputSize, "input-size", 32 * mib, Bytes, ClassData},
	{InputsTotal, "inputs-total", 128 * mib, Bytes, ClassData},
	{CSVRows, "csv-rows", 1_000_000, Count, ClassData},
	{CSVColumns, "csv-columns", 1_024, Count, ClassData},
	{CSVCell, "csv-cell", 1 * mib, Bytes, ClassData},
	{CollectionItems, "collection-items", 100_000, Count, ClassData},
	{FrontmatterSize, "frontmatter-size", 1 * mib, Bytes, ClassData},

	{RenderWork, "render-work", 1 << 32, Count, ClassWork},
	{LoopIterations, "loop-iterations", 10_000_000, Count, ClassWork},
	{ValueSize, "value-size", 64 * mib, Bytes, ClassWork},
	{ArraySize, "array-size", 1_000_000, Count, ClassWork},
	{AssignBytes, "assign-bytes", 512 * mib, Bytes, ClassWork},
	{SchemaWork, "schema-work", 100_000_000, Count, ClassWork},
	{RegexWork, "regex-work", 1 << 33, Count, ClassWork},
	{Outputs, "outputs", 100_000, Count, ClassWork},
	{ContainerDepth, "container-depth", 3, Count, ClassWork},
	{OutputSize, "output-size", 256 * mib, Bytes, ClassWork},
	{OutputTotal, "output-total", 1 * gib, Bytes, ClassWork},
	{Findings, "findings", 100_000, Count, ClassWork},
	{GrepMatches, "grep-matches", 10_000, Count, ClassWork},
	{GrepContext, "grep-context", 100, Count, ClassWork},
}

// All returns every limit, in ADR-009 D3's table order.
func All() []Limit { return append([]Limit{}, table[:]...) }

// Get returns the limit id names.
func Get(id ID) Limit {
	if id < 0 || id >= count {
		panic(fmt.Sprintf("limits: no limit %d", id))
	}
	return table[id]
}

// Lookup returns the limit called name, and false when no limit has it.
func Lookup(name string) (Limit, bool) {
	for _, l := range table {
		if l.Name == name {
			return l, true
		}
	}
	return Limit{}, false
}

// Set is the limits one command runs under: every default, with any the
// caller lowered (ADR-009 D4 item 1). The zero Set is not valid; start
// from Defaults.
type Set struct {
	value   [count]int64
	lowered [count]bool
	valid   bool
}

// Defaults returns the Set of every default.
func Defaults() Set {
	s := Set{valid: true}
	for i, l := range table {
		s.value[i] = l.Default
	}
	return s
}

// Value returns the bound id has in s.
func (s Set) Value(id ID) int64 {
	s.check(id)
	return s.value[id]
}

// Lowered reports whether the caller lowered id, which a limit finding
// states (ADR-009 D2 item 5).
func (s Set) Lowered(id ID) bool {
	s.check(id)
	return s.lowered[id]
}

func (s Set) check(id ID) {
	if !s.valid {
		panic("limits: a Set not made by Defaults")
	}
	Get(id)
}

// The reasons limit.invalid gives.
const (
	ReasonUnknownName    = "unknown_name"
	ReasonMalformedValue = "malformed_value"
	ReasonAboveDefault   = "above_default"
)

// Lower lowers one limit by arg, the NAME=VALUE text of one --limit
// (ADR-009 D4). VALUE is a non-negative decimal integer, 0 or without a
// leading zero, and for a size limit it may end in KiB, MiB or GiB, powers
// of 1024. A value no higher than the default replaces the limit's bound
// for this command, a later one replacing an earlier; the bound counts as
// lowered while it is below the default.
//
// An unknown name, a malformed value or a value above the default is
// refused: Lower returns a limit.invalid finding, class usage, naming the
// limit's default where arg names a limit, and false; s is unchanged.
// Nothing arg holds that names no limit is echoed into the finding.
func (s *Set) Lower(arg string) (finding.Finding, bool) {
	s.check(0)
	name, value, hasValue := strings.Cut(arg, "=")
	l, known := Lookup(name)
	if !known {
		return invalid(ReasonUnknownName, nil, 0, false), false
	}
	if !hasValue {
		return invalid(ReasonMalformedValue, &l, 0, false), false
	}
	n, err := parseValue(value, l.Unit)
	switch {
	case err == errMalformed:
		return invalid(ReasonMalformedValue, &l, 0, false), false
	case err == errTooLarge:
		return invalid(ReasonAboveDefault, &l, 0, false), false
	case n > l.Default:
		return invalid(ReasonAboveDefault, &l, n, true), false
	}
	s.value[l.ID], s.lowered[l.ID] = n, n < l.Default
	return finding.Finding{}, true
}

// maxEchoed is the largest value a limit.invalid finding states: the
// largest integer with an exact canonical JSON number form.
const maxEchoed = 1 << 53

func invalid(reason string, l *Limit, value int64, hasValue bool) finding.Finding {
	p := finding.Params{"reason": finding.String(reason)}
	if l != nil {
		p["name"] = finding.String(l.Name)
		p["default"] = finding.Int(l.Default)
		p["unit"] = finding.String(l.Unit.String())
		if hasValue && value <= maxEchoed {
			p["value"] = finding.Int(value)
		}
	}
	return finding.New(finding.CodeLimitInvalid, finding.Subject{Kind: finding.SubjectInvocation}, nil, p)
}

type valueError string

func (e valueError) Error() string { return string(e) }

const (
	errMalformed valueError = "malformed"
	errTooLarge  valueError = "too large"
)

// parseValue reads a --limit VALUE in unit u: digits, then a size suffix
// when u is Bytes. A well-formed value too large for an int64 is
// errTooLarge, which is above every default.
func parseValue(v string, u Unit) (int64, error) {
	mult := int64(1)
	if u == Bytes {
		for _, sfx := range []struct {
			s string
			m int64
		}{{"KiB", kib}, {"MiB", mib}, {"GiB", gib}} {
			if strings.HasSuffix(v, sfx.s) {
				v, mult = strings.TrimSuffix(v, sfx.s), sfx.m
				break
			}
		}
	}
	if v == "" || (v[0] == '0' && len(v) > 1) {
		return 0, errMalformed
	}
	for i := 0; i < len(v); i++ {
		if v[i] < '0' || v[i] > '9' {
			return 0, errMalformed
		}
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return 0, errTooLarge
	}
	if n > (1<<63-1)/mult {
		return 0, errTooLarge
	}
	return n * mult, nil
}

// Breached returns the limit.package finding (ADR-009 D19): the package's
// own content breaches id, a limit of s. loc is where, in the package. It
// carries the fields of ADR-009 D2 item 5: the limit's name, its bound in
// s, whether the caller lowered it, and observed, the value found, when
// hasObserved is set; a reading that stopped at the bound has none.
func (s Set) Breached(id ID, loc finding.Location, observed int64, hasObserved bool) finding.Finding {
	s.check(id)
	p := finding.Params{
		"limit":   finding.String(table[id].Name),
		"bound":   finding.Int(s.value[id]),
		"lowered": finding.Bool(s.lowered[id]),
	}
	if hasObserved && observed <= maxEchoed {
		p["observed"] = finding.Int(observed)
	}
	return finding.New(finding.CodeLimitPackage, finding.Subject{Kind: finding.SubjectPackage}, &loc, p)
}
