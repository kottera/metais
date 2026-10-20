// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"errors"
	"math"
	"regexp"
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// pos is n's position in the document: the library's line, and its
// column, which it counts in characters (ADR-006 D5 item 4). The library
// places an empty node that ends the text at the start of a line the text
// does not have; that node is at the end of the text.
func (r *reader) pos(n *yaml.Node) finding.Pos {
	if r.offsetOf(n.Line, n.Column) < 0 {
		return r.posAt(len(r.data))
	}
	return finding.Pos{Line: n.Line, Column: n.Column}
}

// atPtr is the finding location of n, about the value or member at pl.
func (r *reader) atPtr(n *yaml.Node, pl *place) finding.Location {
	return r.at(r.pos(n)).WithPath(pl.pointer())
}

// node checks n, the value at pl, nested in depth-1 arrays and objects,
// and returns its value. A refused node is reported and its value is
// null: once anything is reported the document is refused, so the value
// is never used.
func (r *reader) node(n *yaml.Node, ptr *place, depth int) value.Value {
	if r.counted {
		return value.Null{}
	}
	ptr.value, ptr.hasValue = r.inFile(r.pos(n)), true
	if !r.src.Values.Add(1) {
		r.counted = true
		r.findings = append(r.findings, r.limit(r.src.Values.ID(), r.atPtr(n, ptr), 0, false))
		return value.Null{}
	}
	r.properties(n, ptr)
	switch n.Kind {
	case yaml.AliasNode:
		r.add(ruleAlias, r.atPtr(n, ptr), nil)
		return value.Null{}
	case yaml.ScalarNode:
		return r.scalar(n, ptr)
	}
	if bound := r.lim.Value(limits.ValueDepth); int64(depth) > bound {
		r.findings = append(r.findings, r.limit(limits.ValueDepth, r.atPtr(n, ptr), 0, false))
		return value.Null{}
	}
	if n.Kind == yaml.SequenceNode {
		elems := make([]value.Value, len(n.Content))
		for i, c := range n.Content {
			elems[i] = r.node(c, ptr.child(strconv.Itoa(i)), depth+1)
		}
		a, err := value.NewArray(elems...)
		if err != nil {
			panic("yamlx: " + err.Error())
		}
		return a
	}
	if n.Kind != yaml.MappingNode || len(n.Content)%2 != 0 {
		panic("yamlx: a node of kind " + strconv.Itoa(int(n.Kind)) + " in a document")
	}
	return r.mapping(n, ptr, depth)
}

// mapping checks the members of n, the mapping at ptr (ADR-002 D2 items 6
// to 8), and returns its object.
func (r *reader) mapping(n *yaml.Node, ptr *place, depth int) value.Value {
	var members []value.Member
	// first is where each name was first given, kept as it is seen so a
	// duplicate costs no walk back along its line.
	first := map[string]finding.Pos{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		r.properties(k, ptr)
		switch {
		case k.Kind == yaml.AliasNode:
			r.add(ruleAlias, r.atPtr(k, ptr), nil)
			continue
		case k.Kind != yaml.ScalarNode:
			r.add(ruleKeyNotScalar, r.atPtr(k, ptr), nil)
			continue
		case plain(k) && k.Value == "":
			r.add(ruleKeyEmpty, r.atPtr(k, ptr), nil)
			continue
		case plain(k) && k.Value == "<<":
			r.add(ruleMergeKey, r.atPtr(k, ptr), nil)
			continue
		}
		name := k.Value
		at := ptr.child(name)
		r.text(k, at, name)
		if f, dup := first[name]; dup {
			r.add(ruleDuplicateKey, r.atPtr(k, at), nil,
				finding.WithRelated(finding.Related{Location: r.at(f).WithPath(at.pointer()), Note: noteFirst}))
			r.node(v, at, depth+1)
			continue
		}
		first[name] = r.pos(k)
		at.key, at.hasKey = r.inFile(first[name]), true
		members = append(members, value.Member{Name: name, Value: r.node(v, at, depth+1)})
	}
	o, err := value.NewObject(members...)
	if err != nil {
		panic("yamlx: " + err.Error())
	}
	return o
}

// properties refuses n's anchor and its tag (ADR-002 D2 items 4 and 5).
// The library marks an explicit tag in n's style, except the non-specific
// tag `!`, which it resolves without a trace; so a tag is also found in
// the text, where n begins with its properties.
func (r *reader) properties(n *yaml.Node, ptr *place) {
	if n.Anchor != "" {
		r.add(ruleAnchor, r.atPtr(n, ptr), nil)
	}
	if n.Style&yaml.TaggedStyle != 0 || r.tagAt(n) && !startsWithKey(n) {
		r.add(ruleTag, r.atPtr(n, ptr), nil)
	}
}

// startsWithKey reports whether n is a block mapping that begins where
// its first key does, so the text there is the key's, properties and
// all.
func startsWithKey(n *yaml.Node) bool {
	return n.Kind == yaml.MappingNode && len(n.Content) > 0 && n.Content[0].Line == n.Line && n.Content[0].Column == n.Column
}

// tagAt reports whether the text at n's position is a tag, or an anchor
// followed by one. No other node begins with `!`: a plain scalar may not
// (YAML 1.2.2 §7.3.3), and every other node begins with its indicator,
// except a block mapping, which begins with its first key.
func (r *reader) tagAt(n *yaml.Node) bool {
	off := r.offsetOf(n.Line, n.Column)
	if off < 0 || off >= len(r.data) {
		return false
	}
	if r.data[off] == '&' {
		for off < len(r.data) && !separator(r.data[off]) {
			off++
		}
		for off < len(r.data) {
			switch c := r.data[off]; {
			case c == ' ' || c == '\t' || c == '\n' || c == '\r':
				off++
			case c == '#':
				for off < len(r.data) && r.data[off] != '\n' && r.data[off] != '\r' {
					off++
				}
			default:
				return c == '!'
			}
		}
		return false
	}
	return r.data[off] == '!'
}

// separator reports whether c ends an anchor's name: white space, a line
// break or a flow indicator (YAML 1.2.2 §6.9.2).
func separator(c byte) bool {
	switch c {
	case ' ', '\t', '\n', '\r', ',', '[', ']', '{', '}':
		return true
	}
	return false
}

// plain reports whether n is a plain scalar: neither quoted nor a block
// scalar.
func plain(n *yaml.Node) bool {
	return n.Style&(yaml.DoubleQuotedStyle|yaml.SingleQuotedStyle|yaml.LiteralStyle|yaml.FoldedStyle) == 0
}

// scalar resolves n, the scalar at ptr. Quoted and block scalars are
// strings; a plain scalar is resolved by ADR-002 D2 item 9's table.
func (r *reader) scalar(n *yaml.Node, ptr *place) value.Value {
	if !plain(n) {
		return r.text(n, ptr, n.Value)
	}
	s := n.Value
	switch {
	case s == "" || s == "null":
		return value.Null{}
	case s == "true" || s == "false":
		return value.Bool(s == "true")
	case jsonNumber.MatchString(s):
		return r.number(n, ptr, s)
	}
	if form := refusedForm(s); form != "" {
		r.add(ruleScalarRefused, r.atPtr(n, ptr), finding.Params{"form": finding.String(form)})
		return value.Null{}
	}
	return r.text(n, ptr, s)
}

var (
	// jsonNumber is RFC 8259's number grammar, which ADR-002 D2 item 9
	// resolves to a number.
	jsonNumber = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?([eE][-+]?[0-9]+)?$`)
	// coreNumber is every int and float form of YAML 1.2.2's Core Schema
	// (§10.3.2); a plain scalar it matches that jsonNumber does not is
	// refused rather than resolved.
	coreNumber = regexp.MustCompile(`^(?:[-+]?[0-9]+|0o[0-7]+|0x[0-9a-fA-F]+|[-+]?(?:\.[0-9]+|[0-9]+(?:\.[0-9]*)?)(?:[eE][-+]?[0-9]+)?|[-+]?\.(?:inf|Inf|INF)|\.(?:nan|NaN|NAN))$`)
)

// refusedForm is the form of s, a plain scalar that resolves to no null,
// boolean or number, when ADR-002 D2 item 9 refuses it, or "" when it is
// a string.
func refusedForm(s string) string {
	switch s {
	case "~", "Null", "NULL":
		return formNull
	case "True", "TRUE", "False", "FALSE":
		return formBoolean
	}
	if coreNumber.MatchString(s) {
		return formNumber
	}
	return ""
}

// number is the number lit, the plain scalar n at ptr, held to
// number-length and to the data model's range: ADR-002 D4 item 3 for the
// manifest, which refuses any integer value beyond 2^53-1, and ADR-003 D8
// item 5 for an input, which refuses one written without fraction or
// exponent.
func (r *reader) number(n *yaml.Node, ptr *place, lit string) value.Value {
	if l := int64(len(lit)); l > r.lim.Value(limits.NumberLength) {
		r.findings = append(r.findings, r.limit(limits.NumberLength, r.atPtr(n, ptr), l, true))
		return value.Null{}
	}
	num, err := value.ParseNumber(lit)
	switch {
	case errors.Is(err, value.ErrNumberRange):
		r.add(ruleNumberRange, r.atPtr(n, ptr), finding.Params{"reason": finding.String(reasonOverflow)})
		return value.Null{}
	case err != nil:
		panic("yamlx: " + err.Error())
	}
	whole := num.Integral()
	if !r.manifest {
		whole = num.WrittenAsInteger()
	}
	if whole && math.Abs(num.Float64()) > value.MaxSafeInteger {
		r.add(ruleNumberRange, r.atPtr(n, ptr), finding.Params{"reason": finding.String(reasonUnsafeInteger)})
		return value.Null{}
	}
	return num
}

// text is the string s, the text of n at ptr. An input's strings hold no
// noncharacter (ADR-003 D8 item 4); the first one is reported.
func (r *reader) text(n *yaml.Node, ptr *place, s string) value.Value {
	if !r.manifest {
		for _, c := range s {
			if noncharacter(c) {
				r.add(ruleCharacter, r.atPtr(n, ptr), finding.Params{
					"codepoint": finding.String(codepoint(c)), "reason": finding.String(reasonNoncharacter)})
				break
			}
		}
	}
	v, err := value.NewString(s)
	if err != nil {
		panic("yamlx: " + err.Error())
	}
	return v
}

// noncharacter reports whether c is one of Unicode's 66 noncharacters.
func noncharacter(c rune) bool {
	return 0xFDD0 <= c && c <= 0xFDEF || c&0xFFFE == 0xFFFE
}
