// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"maps"
	"slices"
)

// sigKind is a filter's analysis signature (ADR-007 D3 item 10): what the
// analysis takes its result to be.
type sigKind uint8

const (
	// sigPreserve gives its input.
	sigPreserve sigKind = iota + 1
	// sigDefault gives the union of its input and its first argument.
	sigDefault
	// sigConcat gives an array of the union of its input's element values
	// and its argument's.
	sigConcat
	// sigFirst and sigLast give the element at [0] and [-1]: of an array,
	// its element; of a string, a string.
	sigFirst
	sigLast
	// sigProject gives an array of its property-name argument resolved on
	// its input's element.
	sigProject
	// sigScalar gives a scalar of a stated kind.
	sigScalar
	// sigUnknown gives unknown.
	sigUnknown
)

// signature is one filter's analysis signature.
type signature struct {
	kind sigKind
	// scalar is the kind sigScalar gives.
	scalar scalar
	// property is whether a string literal as the first positional
	// argument is a property name, resolved as .p on the input's element
	// and checked like any segment.
	property bool
}

// signatures is the cross-reference's own filter-signature table (ADR-007
// D3 item 10): every filter of the standard catalog, liquid.Core()'s,
// which is ADR-010 D4 item 2's list of Shopify's 5.14.0 filters without
// date, and with escape's alias h, which ADR-004's Context lists. A filter the
// table does not name — date and METAIS's own (ADR-004 D10, D11), which
// internal/template adds to the dialect — gives unknown: its arguments
// are still analysed.
var signatures = map[string]signature{
	// Preserves its input.
	"compact":      {kind: sigPreserve},
	"reverse":      {kind: sigPreserve},
	"slice":        {kind: sigPreserve},
	"sort":         {kind: sigPreserve, property: true},
	"sort_natural": {kind: sigPreserve, property: true},
	"uniq":         {kind: sigPreserve},
	"where":        {kind: sigPreserve, property: true},
	"default":      {kind: sigDefault},
	"concat":       {kind: sigConcat},
	// Gives the element.
	"first": {kind: sigFirst},
	"last":  {kind: sigLast},
	// Projects a property.
	"map": {kind: sigProject, property: true},
	// Gives a number.
	"abs":        {kind: sigScalar, scalar: sNumber},
	"at_least":   {kind: sigScalar, scalar: sNumber},
	"at_most":    {kind: sigScalar, scalar: sNumber},
	"ceil":       {kind: sigScalar, scalar: sNumber},
	"divided_by": {kind: sigScalar, scalar: sNumber},
	"floor":      {kind: sigScalar, scalar: sNumber},
	"minus":      {kind: sigScalar, scalar: sNumber},
	"modulo":     {kind: sigScalar, scalar: sNumber},
	"plus":       {kind: sigScalar, scalar: sNumber},
	"round":      {kind: sigScalar, scalar: sNumber},
	"size":       {kind: sigScalar, scalar: sNumber},
	"sum":        {kind: sigScalar, scalar: sNumber, property: true},
	"times":      {kind: sigScalar, scalar: sNumber},
	// Gives a boolean.
	"has": {kind: sigScalar, scalar: sBool},
	// Gives a string.
	"append":                 {kind: sigScalar, scalar: sString},
	"base64_decode":          {kind: sigScalar, scalar: sString},
	"base64_encode":          {kind: sigScalar, scalar: sString},
	"base64_url_safe_decode": {kind: sigScalar, scalar: sString},
	"base64_url_safe_encode": {kind: sigScalar, scalar: sString},
	"capitalize":             {kind: sigScalar, scalar: sString},
	"downcase":               {kind: sigScalar, scalar: sString},
	"escape":                 {kind: sigScalar, scalar: sString},
	"escape_once":            {kind: sigScalar, scalar: sString},
	"h":                      {kind: sigScalar, scalar: sString},
	"join":                   {kind: sigScalar, scalar: sString},
	"lstrip":                 {kind: sigScalar, scalar: sString},
	"newline_to_br":          {kind: sigScalar, scalar: sString},
	"prepend":                {kind: sigScalar, scalar: sString},
	"remove":                 {kind: sigScalar, scalar: sString},
	"remove_first":           {kind: sigScalar, scalar: sString},
	"remove_last":            {kind: sigScalar, scalar: sString},
	"replace":                {kind: sigScalar, scalar: sString},
	"replace_first":          {kind: sigScalar, scalar: sString},
	"replace_last":           {kind: sigScalar, scalar: sString},
	"rstrip":                 {kind: sigScalar, scalar: sString},
	"squish":                 {kind: sigScalar, scalar: sString},
	"strip":                  {kind: sigScalar, scalar: sString},
	"strip_html":             {kind: sigScalar, scalar: sString},
	"strip_newlines":         {kind: sigScalar, scalar: sString},
	"truncate":               {kind: sigScalar, scalar: sString},
	"truncatewords":          {kind: sigScalar, scalar: sString},
	"upcase":                 {kind: sigScalar, scalar: sString},
	"url_decode":             {kind: sigScalar, scalar: sString},
	"url_encode":             {kind: sigScalar, scalar: sString},
	// Gives a value the analysis cannot tell: an element found by a
	// value, its index or nil, an array of strings, or a filtered array.
	"find":       {kind: sigUnknown},
	"find_index": {kind: sigUnknown},
	"reject":     {kind: sigUnknown},
	"split":      {kind: sigUnknown},
}

// Filters returns the names of the filters the signature table states a
// signature for, sorted.
func Filters() []string {
	return slices.Sorted(maps.Keys(signatures))
}
