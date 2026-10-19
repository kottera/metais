// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package ast is the syntax tree of METAIS's Liquid dialect (ADR-004 D1 to
// D3, ADR-010 D4 item 2): what internal/liquid's parser produces, what its
// renderer executes, and what ADR-007's cross-reference walks without a
// second parser.
//
// Every node carries the Span of the source text it was parsed from. A
// template is a Block of nodes: Text, Output, and one node type per tag.
// Every expression is a tree: a Primary (a Literal, a Path or a Range),
// optionally followed by a chain of Filters in an Expr, and the conditions
// of if, elsif, unless and when are a Cond tree.
//
// The nodes that bind a name — assign, capture, increment, decrement, for,
// tablerow and render's parameters and alias — implement Binder, which
// says what each binds and in which scope (ADR-004 D7 item 8).
//
// Walk visits every node, in source order. A tree is immutable once the
// parser returns it: nothing in this package or in the renderer changes a
// node, so one tree is safe to render any number of times.
//
// The package is part of internal/liquid, so it imports nothing outside
// the standard library and no other internal package but value (ADR-010
// D2 item 2).
package ast

import "github.com/kottera/metais/internal/liquid/value"

// Span is where a node's text lies in its source: the bytes [Start, End),
// and the line and column of Start. Line and Col are 1-based; Col counts
// Unicode code points from the start of the line, a tab counting as one,
// and a line ends at LF, CR LF or a lone CR (ADR-006 D5 item 3).
type Span struct {
	Start, End int
	Line, Col  int
}

// Node is a node of the tree.
type Node interface {
	// Pos returns where the node's text lies in its source.
	Pos() Span
	node()
}

// Scope is how far a binding reaches (ADR-004 D7 item 8).
type Scope uint8

// The scopes.
const (
	// ScopeTemplate is a binding visible from where it is made to the end
	// of the template: assign and capture, and increment's and decrement's
	// counters. It may not take a reserved name.
	ScopeTemplate Scope = iota + 1
	// ScopeBlock is a binding visible only inside the block that makes it:
	// a for or tablerow variable. It may shadow any name.
	ScopeBlock
	// ScopePartial is a binding visible only inside the partial render
	// renders: a render parameter, or the name its with or for value is
	// bound to. It may shadow any name.
	ScopePartial
)

// String names the scope.
func (s Scope) String() string {
	switch s {
	case ScopeTemplate:
		return "template"
	case ScopeBlock:
		return "block"
	case ScopePartial:
		return "partial"
	}
	return "unknown"
}

// Binding is one name a Binder binds: where the name is written, its
// scope, and whether it is a counter, which lives in its own namespace
// (increment and decrement, ADR-004 D2).
type Binding struct {
	Name    string
	Pos     Span
	Scope   Scope
	Counter bool
}

// Binder is a node that binds names. Bindings returns them in source
// order; the values they are bound to are the node's own fields.
type Binder interface {
	Node
	Bindings() []Binding
}

// Block is a sequence of nodes: a template's body, or one branch of a tag.
type Block struct {
	Span  Span
	Nodes []Node
}

// Text is template text written as it is, except for whitespace control:
// TrimLeft is set when the token before it ends with "-}}" or "-%}", and
// TrimRight when the token after it starts with "{{-" or "{%-"; each
// removes the whitespace ADR-004 D3 item 1 names from that end. Text is
// the source text exactly, before either is applied.
type Text struct {
	Span                Span
	Text                string
	TrimLeft, TrimRight bool
}

// Trimmed returns the text with its whitespace control applied.
func (t *Text) Trimmed() string {
	s := t.Text
	if t.TrimLeft {
		i := 0
		for i < len(s) && IsSpace(s[i]) {
			i++
		}
		s = s[i:]
	}
	if t.TrimRight {
		i := len(s)
		for i > 0 && IsSpace(s[i-1]) {
			i--
		}
		s = s[:i]
	}
	return s
}

// IsSpace reports whether b is whitespace as ADR-004 D3 item 1 and D5 item
// 2 define it: U+0020, U+0009, U+000A, U+000B, U+000C or U+000D.
func IsSpace(b byte) bool {
	switch b {
	case ' ', '\t', '\n', '\v', '\f', '\r':
		return true
	}
	return false
}

// Output is "{{ … }}". Value is nil for an output with no expression,
// which writes nothing.
type Output struct {
	Span  Span
	Value *Expr
}

// Expr is a value expression: what may appear inside "{{ … }}", a Primary
// followed by any number of filters, applied left to right.
type Expr struct {
	Span    Span
	Base    Primary
	Filters []*Filter
}

// Filter is one application of a filter: "| name" with its arguments in
// source order. An argument with a Name is a keyword argument, "name:
// value"; one without is positional.
type Filter struct {
	Span Span
	Name string
	Args []*Arg
}

// Arg is one argument of a filter or of a tag. Name is empty for a
// positional argument.
type Arg struct {
	Span  Span
	Name  string
	Value Primary
}

// Primary is a value without filters: a Literal, a Path or a Range.
type Primary interface {
	Node
	primary()
}

// LiteralKind says which literal a Literal is.
type LiteralKind uint8

// The literal kinds.
const (
	LitString LiteralKind = iota + 1
	LitInteger
	LitFloat
	LitTrue
	LitFalse
	LitNil
	// LitEmpty and LitBlank are Shopify's empty and blank, which are not
	// values of the data model (ADR-004 D5 item 2): their Value is nil.
	LitEmpty
	LitBlank
)

// Literal is a literal of the template's own text. Text is the literal as
// written, without the quotes of a string. Value is the data-model value
// it denotes: a value.String, a value.Number whose literal records whether
// it was written as an integer or a float (ADR-004 D4 item 2), a
// value.Bool or value.Null; nil for empty and blank.
type Literal struct {
	Span  Span
	Kind  LiteralKind
	Text  string
	Value value.Value
}

// Path is a variable lookup: a root name followed by segments. Root is
// empty when the path begins with an index ("['a b'].c"), and then its
// first segment is that index.
type Path struct {
	Span     Span
	Root     string
	Segments []*Segment
}

// Segment is one step of a Path: ".name" when Index is nil, "[expr]"
// otherwise. "[\"name\"]" is an Index whose value is a string literal, so
// a static lookup is either form.
type Segment struct {
	Span  Span
	Name  string
	Index Primary
}

// Range is "(from..to)".
type Range struct {
	Span     Span
	From, To Primary
}

// CondKind says which form a Cond is.
type CondKind uint8

// The condition forms.
const (
	// CondValue is a value tested for truthiness: Left.
	CondValue CondKind = iota + 1
	// CondCompare is "Left Op Right".
	CondCompare
	// CondAnd and CondOr join two conditions, X and Y. They group from the
	// right, as Shopify's do: "a or b and c" is a or (b and c), and "a and b
	// or c" is a and (b or c) (ADR-004 D2). X is evaluated first.
	CondAnd
	CondOr
)

// CompareOp is a comparison operator.
type CompareOp string

// The comparison operators of ADR-004 D4 item 6.
const (
	OpEq       CompareOp = "=="
	OpNe       CompareOp = "!="
	OpLtGt     CompareOp = "<>"
	OpLt       CompareOp = "<"
	OpGt       CompareOp = ">"
	OpLe       CompareOp = "<="
	OpGe       CompareOp = ">="
	OpContains CompareOp = "contains"
)

// Cond is a condition: what follows if, elsif and unless, and ParseCondition's
// result.
type Cond struct {
	Span        Span
	Kind        CondKind
	Left, Right Primary   // CondValue: Left; CondCompare: Left Op Right
	Op          CompareOp // CondCompare
	X, Y        *Cond     // CondAnd, CondOr
}

// Branch is one condition and the body it guards.
type Branch struct {
	Span Span
	Cond *Cond
	Body *Block
}

// If is "{% if %}…{% elsif %}…{% else %}…{% endif %}". Else is nil when
// absent.
type If struct {
	Span     Span
	Branches []*Branch
	Else     *Block
}

// Unless is "{% unless %}" with its elsif and else branches. The first
// branch's body runs when its condition is falsy; each later branch is an
// elsif, as in If.
type Unless struct {
	Span     Span
	Branches []*Branch
	Else     *Block
}

// When is one clause of a case: a "{% when %}" with its values, any of
// which equal to the case's subject selects Body, or, when Else is set, an
// "{% else %}", whose Values are nil. Shopify evaluates a case's clauses
// in order and renders every one that is selected: each when whose values
// match, and each else that no earlier when matched.
type When struct {
	Span   Span
	Values []Primary
	Else   bool
	Body   *Block
}

// Case is "{% case subject %}" with its clauses in source order. Before is
// what lies between the case tag and its first clause: it is parsed, as
// Shopify parses it, and never rendered.
type Case struct {
	Span    Span
	Subject Primary
	Before  *Block
	Clauses []*When
}

// For is "{% for var in collection … %}…{% else %}…{% endfor %}".
// Reversed, Limit and Offset are the loop's modifiers, applied as offset,
// then limit, then reversed (ADR-004 D2). OffsetContinue is "offset:
// continue", and then Offset is nil. Name is Shopify's forloop.name,
// "var-collection" with the collection as written.
type For struct {
	Span           Span
	Var            string
	VarSpan        Span
	Collection     Primary
	Reversed       bool
	Limit, Offset  Primary
	OffsetContinue bool
	Name           string
	Body           *Block
	Else           *Block
}

// Tablerow is "{% tablerow var in collection … %}…{% endtablerow %}".
type Tablerow struct {
	Span                Span
	Var                 string
	VarSpan             Span
	Collection          Primary
	Cols, Limit, Offset Primary
	Name                string
	Body                *Block
}

// Assign is "{% assign name = value %}".
type Assign struct {
	Span     Span
	Name     string
	NameSpan Span
	Value    *Expr
}

// Capture is "{% capture name %}…{% endcapture %}".
type Capture struct {
	Span     Span
	Name     string
	NameSpan Span
	Body     *Block
}

// Counter is "{% increment name %}" or, when Decrement, "{% decrement
// name %}".
type Counter struct {
	Span      Span
	Name      string
	NameSpan  Span
	Decrement bool
}

// Cycle is "{% cycle [group:] value, … %}". Group is nil when absent.
type Cycle struct {
	Span   Span
	Group  Primary
	Values []Primary
}

// Echo is "{% echo value %}", an output expression.
type Echo struct {
	Span  Span
	Value *Expr
}

// Break is "{% break %}".
type Break struct{ Span Span }

// Continue is "{% continue %}".
type Continue struct{ Span Span }

// Ifchanged is "{% ifchanged %}…{% endifchanged %}".
type Ifchanged struct {
	Span Span
	Body *Block
}

// Comment is "{% comment %}…{% endcomment %}", "{% doc %}…{% enddoc %}"
// or an inline "{% # … %}". Text is the comment's body as written; it
// renders nothing.
type Comment struct {
	Span Span
	Kind CommentKind
	Text string
}

// CommentKind says which comment a Comment is.
type CommentKind uint8

// The comment kinds.
const (
	CommentBlock CommentKind = iota + 1
	CommentInline
	CommentDoc
)

// Raw is "{% raw %}…{% endraw %}". Text is its body, written verbatim:
// whitespace control on the raw and endraw tags trims the text outside
// them, never the body, as in Shopify.
type Raw struct {
	Span     Span
	Text     string
	TextSpan Span
}

// Liquid is "{% liquid … %}": one tag per line, its body parsed as a
// block.
type Liquid struct {
	Span Span
	Body *Block
}

// RenderMode says how a render passes its value.
type RenderMode uint8

// The render modes.
const (
	RenderPlain RenderMode = iota + 1 // no with or for
	RenderWith                        // with value [as alias]
	RenderFor                         // for collection [as alias], binding forloop in the partial
)

// Render is "{% render "name" … %}". Name is the partial's literal name
// (ADR-004 D8 item 1). Value is the with or for value, nil for
// RenderPlain. Alias is the name the value is bound to inside the
// partial: the as name, or else what follows the partial name's last "/",
// as Shopify binds it; empty when Mode is RenderPlain. Params are the name: value pairs.
type Render struct {
	Span      Span
	Name      string
	NameSpan  Span
	Mode      RenderMode
	Value     Primary
	Alias     string
	AliasSpan Span
	Params    []*Arg
}

// Tag is a tag a Dialect adds (internal/liquid's TagDef): "{% name args
// %}", its markup parsed as an argument list, positional arguments before
// keyword arguments.
type Tag struct {
	Span Span
	Name string
	Args []*Arg
}

func (b *Block) Pos() Span     { return b.Span }
func (t *Text) Pos() Span      { return t.Span }
func (o *Output) Pos() Span    { return o.Span }
func (e *Expr) Pos() Span      { return e.Span }
func (f *Filter) Pos() Span    { return f.Span }
func (a *Arg) Pos() Span       { return a.Span }
func (l *Literal) Pos() Span   { return l.Span }
func (p *Path) Pos() Span      { return p.Span }
func (s *Segment) Pos() Span   { return s.Span }
func (r *Range) Pos() Span     { return r.Span }
func (c *Cond) Pos() Span      { return c.Span }
func (b *Branch) Pos() Span    { return b.Span }
func (n *If) Pos() Span        { return n.Span }
func (n *Unless) Pos() Span    { return n.Span }
func (n *When) Pos() Span      { return n.Span }
func (n *Case) Pos() Span      { return n.Span }
func (n *For) Pos() Span       { return n.Span }
func (n *Tablerow) Pos() Span  { return n.Span }
func (n *Assign) Pos() Span    { return n.Span }
func (n *Capture) Pos() Span   { return n.Span }
func (n *Counter) Pos() Span   { return n.Span }
func (n *Cycle) Pos() Span     { return n.Span }
func (n *Echo) Pos() Span      { return n.Span }
func (n *Break) Pos() Span     { return n.Span }
func (n *Continue) Pos() Span  { return n.Span }
func (n *Ifchanged) Pos() Span { return n.Span }
func (n *Comment) Pos() Span   { return n.Span }
func (n *Raw) Pos() Span       { return n.Span }
func (n *Liquid) Pos() Span    { return n.Span }
func (n *Render) Pos() Span    { return n.Span }
func (n *Tag) Pos() Span       { return n.Span }

func (*Block) node()     {}
func (*Text) node()      {}
func (*Output) node()    {}
func (*Expr) node()      {}
func (*Filter) node()    {}
func (*Arg) node()       {}
func (*Literal) node()   {}
func (*Path) node()      {}
func (*Segment) node()   {}
func (*Range) node()     {}
func (*Cond) node()      {}
func (*Branch) node()    {}
func (*If) node()        {}
func (*Unless) node()    {}
func (*When) node()      {}
func (*Case) node()      {}
func (*For) node()       {}
func (*Tablerow) node()  {}
func (*Assign) node()    {}
func (*Capture) node()   {}
func (*Counter) node()   {}
func (*Cycle) node()     {}
func (*Echo) node()      {}
func (*Break) node()     {}
func (*Continue) node()  {}
func (*Ifchanged) node() {}
func (*Comment) node()   {}
func (*Raw) node()       {}
func (*Liquid) node()    {}
func (*Render) node()    {}
func (*Tag) node()       {}

func (*Literal) primary() {}
func (*Path) primary()    {}
func (*Range) primary()   {}

// Bindings returns the name assign binds, template-scoped.
func (n *Assign) Bindings() []Binding {
	return []Binding{{Name: n.Name, Pos: n.NameSpan, Scope: ScopeTemplate}}
}

// Bindings returns the name capture binds, template-scoped.
func (n *Capture) Bindings() []Binding {
	return []Binding{{Name: n.Name, Pos: n.NameSpan, Scope: ScopeTemplate}}
}

// Bindings returns the counter increment or decrement binds.
func (n *Counter) Bindings() []Binding {
	return []Binding{{Name: n.Name, Pos: n.NameSpan, Scope: ScopeTemplate, Counter: true}}
}

// Bindings returns the loop variable, block-scoped. forloop is bound too,
// and is Liquid's own: it is not reported.
func (n *For) Bindings() []Binding {
	return []Binding{{Name: n.Var, Pos: n.VarSpan, Scope: ScopeBlock}}
}

// Bindings returns the loop variable, block-scoped.
func (n *Tablerow) Bindings() []Binding {
	return []Binding{{Name: n.Var, Pos: n.VarSpan, Scope: ScopeBlock}}
}

// Bindings returns what the partial sees bound: its alias, when the
// render passes a value, then each parameter, all partial-scoped.
func (n *Render) Bindings() []Binding {
	var bs []Binding
	if n.Mode != RenderPlain {
		bs = append(bs, Binding{Name: n.Alias, Pos: n.AliasSpan, Scope: ScopePartial})
	}
	for _, p := range n.Params {
		bs = append(bs, Binding{Name: p.Name, Pos: p.Span, Scope: ScopePartial})
	}
	return bs
}
