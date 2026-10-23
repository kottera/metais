// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"encoding/json"
	"maps"
	"math/big"
	"slices"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/liquid/value"
)

// The rules vocabulary holds the keywords this dialect defines beyond
// 2020-12's, which state rules across the items of an array (ADR-003
// Amendment A2, chapter 5 §5.4). It is registered with the library by
// RegisterVocabulary, and the compiler's AssertVocabs applies it to every
// schema, none of which declares $vocabulary (MTS-DAT-003). Each keyword
// has a grammar the pre-check holds it to (A2 item 14), a compiled form
// the vocabulary's Compile builds, and an error kind the adapter maps.

// rulesVocabularyURL identifies the vocabulary to the library. No schema
// names it: a schema that declares $vocabulary is refused.
const rulesVocabularyURL = "https://metais.org/vocab/rules"

// maxKeyPointers is the most key pointers uniqueKeys holds (MTS-DAT-015).
const maxKeyPointers = 8

// ruleKeywords are the rules vocabulary's keywords, each with the check of
// its grammar: whether a value is one the keyword allows.
var ruleKeywords = map[string]func(value.Value) bool{
	"uniqueKeys": validUniqueKeys,
	"coverage":   validCoverage,
}

// rulesVocabulary is the vocabulary the compiler registers. It has no
// meta-schema of its own: the pre-check refuses a keyword that breaks its
// grammar before the library sees it, with schema.invalid naming the
// keyword rather than a meta-schema's.
func rulesVocabulary() *jsonschema.Vocabulary {
	return &jsonschema.Vocabulary{URL: rulesVocabularyURL, Compile: compileRules}
}

// compileRules compiles the rules vocabulary's keywords of one schema
// object, obj as the library holds it, or returns nil when it has none.
func compileRules(_ *jsonschema.CompilerContext, obj map[string]any) (jsonschema.SchemaExt, error) {
	var r rules
	if v, ok := obj["uniqueKeys"]; ok {
		// The pre-check held it to its grammar: strings, each a pointer.
		for _, p := range v.([]any) {
			r.uniqueKeys = append(r.uniqueKeys, pointerTokens(p.(string)))
			r.keyTexts = append(r.keyTexts, p.(string))
		}
	}
	if v, ok := obj["coverage"]; ok {
		// The pre-check held it to its grammar: a key pointer and values.
		m := v.(map[string]any)
		c := &coverage{key: pointerTokens(m["key"].(string)), keyText: m["key"].(string), values: m["values"].([]any)}
		var b strings.Builder
		for _, e := range c.values {
			b.Reset()
			encodeKey(&b, e)
			c.encoded = append(c.encoded, b.String())
		}
		r.coverage = c
	}
	if r.uniqueKeys == nil && r.coverage == nil {
		return nil, nil
	}
	return &r, nil
}

// validUniqueKeys reports whether v is uniqueKeys' value by MTS-DAT-015:
// an array of 1 to 8 key pointers, no two of them equal.
func validUniqueKeys(v value.Value) bool {
	a, ok := v.(value.Array)
	if !ok || a.Len() < 1 || a.Len() > maxKeyPointers {
		return false
	}
	var seen []string
	for _, e := range a.All() {
		s, ok := e.(value.String)
		if !ok || !keyPointer(s.Text()) || slices.Contains(seen, s.Text()) {
			return false
		}
		seen = append(seen, s.Text())
	}
	return true
}

// validCoverage reports whether v is coverage's value by MTS-DAT-016: an
// object of exactly two members, key, a key pointer, and values, an array
// of one or more values no two of which are equal as MTS-DAT-013 compares
// them.
func validCoverage(v value.Value) bool {
	o, ok := v.(value.Object)
	if !ok || o.Len() != 2 {
		return false
	}
	key, ok := o.Get("key")
	if s, isString := key.(value.String); !ok || !isString || !keyPointer(s.Text()) {
		return false
	}
	values, ok := o.Get("values")
	a, isArray := values.(value.Array)
	if !ok || !isArray || a.Len() < 1 {
		return false
	}
	seen := make(map[string]bool, a.Len())
	var b strings.Builder
	for _, e := range a.All() {
		b.Reset()
		encodeKey(&b, toLibrary(e))
		if seen[b.String()] {
			return false
		}
		seen[b.String()] = true
	}
	return true
}

// keyPointer reports whether p is a key pointer (§5.4): an RFC 6901
// pointer other than the empty one. RFC 6901 gives each token one
// spelling, so two key pointers are equal exactly when their texts are.
func keyPointer(p string) bool {
	_, ok := finding.ParsePath(p)
	return ok && p != ""
}

// rules is the rules vocabulary's keywords of one schema object, compiled.
type rules struct {
	uniqueKeys [][]string // each key pointer's tokens, in order
	keyTexts   []string   // each key pointer as the schema writes it
	coverage   *coverage
}

// coverage is coverage's value, compiled.
type coverage struct {
	key     []string // the key pointer's tokens
	keyText string   // the key pointer as the schema writes it
	values  []any    // values' members, as the library holds them
	encoded []string // each member's encoding by encodeKey, in order
}

// Validate evaluates the rules against v, which the library calls with
// every instance the schema object applies to; an instance that is not an
// array is not judged (MTS-DAT-017).
func (r *rules) Validate(ctx *jsonschema.ValidatorContext, v any) {
	items, ok := v.([]any)
	if !ok {
		return
	}
	if r.uniqueKeys != nil {
		r.unique(ctx, items)
	}
	if r.coverage != nil {
		r.coverage.cover(ctx, items)
	}
}

// cover reports each member of values that no item's value at key equals
// (MTS-DAT-019); an item with no value there covers nothing. Each item's
// value is encoded once and each member looked up once, so the evaluation
// is linear in the items and the package's own values (A2 item 18).
func (c *coverage) cover(ctx *jsonschema.ValidatorContext, items []any) {
	held := make(map[string]bool, len(items))
	var b strings.Builder
	for _, item := range items {
		v, ok := resolve(item, c.key)
		if !ok {
			continue
		}
		b.Reset()
		encodeKey(&b, v)
		held[b.String()] = true
	}
	for i, k := range c.encoded {
		if !held[k] {
			ctx.AddError(&coverageKind{Missing: canonicalText(c.values[i]), Key: c.keyText})
		}
	}
}

// unique reports each item whose key under uniqueKeys equals an earlier
// item's, naming the first item with that key (MTS-DAT-018). Each item's
// key is encoded once and looked up once, so the evaluation is linear in
// the items and the size of their keys (A2 item 18).
func (r *rules) unique(ctx *jsonschema.ValidatorContext, items []any) {
	first := make(map[string]int, len(items))
	var at []string
	if len(r.uniqueKeys) == 1 {
		at = r.uniqueKeys[0]
	}
	var b strings.Builder
	for i, item := range items {
		b.Reset()
		if !r.key(&b, item) {
			continue
		}
		k := b.String()
		if j, ok := first[k]; ok {
			ctx.AddError(&uniqueKeysKind{Duplicates: [2]int{j, i}, At: at, Keys: r.keyTexts})
			continue
		}
		first[k] = i
	}
}

// key writes item's key to b, and reports whether it has one: whether
// every key pointer resolves to a value within it.
func (r *rules) key(b *strings.Builder, item any) bool {
	for _, toks := range r.uniqueKeys {
		v, ok := resolve(item, toks)
		if !ok {
			return false
		}
		encodeKey(b, v)
	}
	return true
}

// resolve returns the value toks name within v, as RFC 6901 evaluates a
// pointer, and false when they name nothing.
func resolve(v any, toks []string) (any, bool) {
	for _, t := range toks {
		switch c := v.(type) {
		case map[string]any:
			m, ok := c[t]
			if !ok {
				return nil, false
			}
			v = m
		case []any:
			i, ok := arrayIndex(t)
			if !ok || i >= len(c) {
				return nil, false
			}
			v = c[i]
		default:
			return nil, false
		}
	}
	return v, true
}

// arrayIndex reads an RFC 6901 array index: 0, or digits with no leading
// zero.
func arrayIndex(t string) (int, bool) {
	if t == "" || (len(t) > 1 && t[0] == '0') {
		return 0, false
	}
	for _, c := range t {
		if c < '0' || c > '9' {
			return 0, false
		}
	}
	i, err := strconv.Atoi(t)
	return i, err == nil
}

// encodeKey writes v, a value as the library holds it, to b so that two
// values are written alike exactly when MTS-DAT-013 has them equal: a
// number by its mathematical value, an object whatever its members'
// order. Every value is tagged with its type and every string with its
// length, so no two values' encodings run into each other.
func encodeKey(b *strings.Builder, v any) {
	str := func(s string) {
		b.WriteString(strconv.Itoa(len(s)))
		b.WriteByte(':')
		b.WriteString(s)
	}
	switch v := v.(type) {
	case nil:
		b.WriteByte('z')
	case bool:
		if v {
			b.WriteByte('t')
		} else {
			b.WriteByte('f')
		}
	case string:
		b.WriteByte('s')
		str(v)
	case json.Number:
		r, ok := new(big.Rat).SetString(string(v))
		if !ok {
			// The data model holds only numbers RFC 8259 writes.
			panic("schema: a number that is not a decimal: " + string(v))
		}
		b.WriteByte('n')
		str(r.RatString())
	case []any:
		b.WriteByte('a')
		b.WriteString(strconv.Itoa(len(v)))
		b.WriteByte(':')
		for _, e := range v {
			encodeKey(b, e)
		}
	case map[string]any:
		b.WriteByte('o')
		b.WriteString(strconv.Itoa(len(v)))
		b.WriteByte(':')
		for _, name := range slices.Sorted(maps.Keys(v)) {
			str(name)
			encodeKey(b, v[name])
		}
	default:
		panic("schema: a value of the library's that is not of the data model")
	}
}

// uniqueKeysKind is the library's error kind for one item whose key under
// uniqueKeys repeats an earlier item's: Duplicates holds the earlier
// item's index, then its own. At is the tokens of the keyword's one
// pointer, where the finding is located beneath the item, and nil when the
// keyword has several; Keys is every pointer as the schema writes it, which
// the hint names. It embeds *kind.Group only for the text method the
// library's ErrorKind requires, which the adapter never calls (ADR-010 D5
// item 5).
type uniqueKeysKind struct {
	*kind.Group
	Duplicates [2]int
	At         []string
	Keys       []string
}

// KeywordPath is the keyword's path within its schema object.
func (*uniqueKeysKind) KeywordPath() []string { return []string{"uniqueKeys"} }

// coverageKind is the library's error kind for one member of coverage's
// values that no item holds at the key: Missing is its RFC 8785 text, and
// Key the key pointer as the schema writes it, which the hint names. It
// embeds *kind.Group as uniqueKeysKind does, for the same reason.
type coverageKind struct {
	*kind.Group
	Missing string
	Key     string
}

// KeywordPath is the keyword's path within its schema object.
func (*coverageKind) KeywordPath() []string { return []string{"coverage"} }
