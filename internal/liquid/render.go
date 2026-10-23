// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"fmt"
	"io"
	"runtime/debug"
	"strconv"

	"metais.org/metais/internal/liquid/ast"
	"metais.org/metais/internal/liquid/value"
)

// Partials is where a render tag finds its partial (ADR-004 D8). Partial
// returns the parsed template named name — the tag's literal, matched
// byte for byte — or the error that stops the render. internal/template
// implements it over the package's templates/ entries, read in place; the
// engine has no other source of templates.
type Partials interface {
	Partial(name string) (*Template, *Error)
}

// DefaultPartialDepth is the render depth when RenderLimits names none:
// ADR-009's default for partial-depth.
const DefaultPartialDepth = 32

// RenderLimits are the bounds a render holds itself, beside its Meter's.
// The values are the caller's — internal/template passes ADR-009's — and
// this package knows no other.
type RenderLimits struct {
	// PartialDepth bounds the render calls active at once (ADR-009
	// partial-depth, ADR-004 D8 item 6): with a depth of n, n nested
	// render tags render and the one inside them is refused with
	// CodeLimit. Zero means DefaultPartialDepth.
	PartialDepth int
	// ValueSize bounds, in bytes, every string a render builds: a capture,
	// an ifchanged body and a filter's result (ADR-009 value-size). Zero
	// means no bound.
	ValueSize int64
	// ArraySize bounds the elements of every array a render builds: a
	// range made a value, and a filter's result (ADR-009 array-size). Zero
	// means no bound.
	ArraySize int64
}

// Env is everything a render reads (ADR-010 D4 item 3). Nothing else
// reaches a template: no host, no clock, no global state (ADR-004 D7 item
// 1).
type Env struct {
	// Dialect defines the filters and added tags a render applies; it is
	// the dialect the template was parsed with. Nil is the bare grammar.
	Dialect *Dialect
	// Globals are the names bound beneath every template and partial
	// (ADR-004 D7), made by Data. internal/template binds exactly the
	// globals ADR-004 D7 item 10 lists for the context.
	Globals value.Object
	// Partials resolves render tags. A render tag with no Partials is an
	// internal error.
	Partials Partials
	// Escape, when set, escapes the text of every value written by an
	// output, an echo or a cycle that is not safe (ADR-004 D9): it appends
	// s, escaped, to dst. Nil writes text unescaped.
	Escape func(dst []byte, s string) []byte
	// Meter counts the render's work (ADR-009 D11). It is required: a
	// render without one is refused, since nothing would bound it.
	Meter Meter
	// Limits are the render's own bounds.
	Limits RenderLimits
	// UnorderedObjects, when set, makes every iteration of an object — for
	// and tablerow over it, render … for, and the first and last
	// shorthands — visit its members in the order of a Go map built from
	// them, which may differ from one iteration to the next, rather than in
	// canonical order (ADR-004 D4 item 5). It exists so that a determinism
	// detector can be shown to catch a render whose bytes vary (ADR-010
	// Amendment A2); no render of the tool sets it.
	UnorderedObjects bool
}

// SafeFilter is a FilterDef whose result is safe from escaping: in an
// output with an Escape, a value whose last filter marks it safe is
// written as it is (ADR-004 D9 item 2). escape, escape_once, safe and
// json_script are such filters.
type SafeFilter interface {
	FilterDef
	MarksSafe() bool
}

// LastFilter is a FilterDef that may be applied only as the last filter of
// an output expression, "{{ }}" or echo: METAIS's safe (ADR-004 D9 item
// 3). The parser refuses one applied anywhere else with CodeSafePosition.
type LastFilter interface {
	FilterDef
	OnlyLast() bool
}

// Render renders t to w (ADR-004 D2 to D6, D8). It stops at the first
// error and returns it, located at the "{{" or "{%" of the statement that
// failed, in the template or partial it is written in, with the render
// calls that led there (ADR-004 D16 item 2). What was written before the
// error stays written; internal/template writes nothing of an output with
// an error (ADR-004 D16 item 4).
//
// A panic inside the render is recovered as one CodeInternal error.
func Render(w io.Writer, t *Template, env *Env) (err *Error) {
	defer recoverInternal(&err)
	r, err := newRenderer(env)
	if err != nil {
		return err
	}
	if t == nil || t.Root == nil {
		return &Error{Code: CodeInternal, Detail: "render of a nil template"}
	}
	r.sinks = []*sink{{w: w}}
	r.fr = newRenderFrame(t.Name)
	_, err = r.block(t.Root, false)
	return err
}

// Eval evaluates a value expression, as ParseValue returns one, with
// env's globals bound (ADR-005 D3, ADR-004 D16 item 5). An error is
// located at the node of e that failed; its Source is empty, for the
// caller to name.
func Eval(e *ast.Expr, env *Env) (v value.Value, err *Error) {
	defer recoverInternal(&err)
	r, err := newRenderer(env)
	if err != nil {
		return nil, err
	}
	r.fr = newRenderFrame("")
	v, _, err = r.expr(e)
	return v, err
}

// EvalCondition evaluates a condition, as ParseCondition returns one,
// with env's globals bound, and reports whether it holds. An error is
// located at the node of c that failed; its Source is empty, for the
// caller to name.
func EvalCondition(c *ast.Cond, env *Env) (ok bool, err *Error) {
	defer recoverInternal(&err)
	r, err := newRenderer(env)
	if err != nil {
		return false, err
	}
	r.fr = newRenderFrame("")
	return r.cond(c)
}

// recoverInternal turns a panic into a CodeInternal error (ADR-010 D4
// item 6).
func recoverInternal(err **Error) {
	if p := recover(); p != nil {
		*err = &Error{Code: CodeInternal, Detail: fmt.Sprintf("panic: %v\n%s", p, debug.Stack())}
	}
}

// renderer is one call of Render, Eval or EvalCondition.
type renderer struct {
	env     *Env
	dialect *Dialect
	meter   Meter
	lim     RenderLimits
	// sinks is where writes go: the output, beneath any capture or
	// ifchanged buffers being filled. Eval and EvalCondition write nothing.
	sinks []*sink
	// fr is the template or partial being rendered.
	fr *renderFrame
	// stmt is the statement being executed, where an error is located;
	// nil in Eval and EvalCondition, which locate at the failing node.
	stmt ast.Node
	// depth is the render calls active.
	depth int
	// blank memoizes the blank-body rule per block tag (ADR-004 D3 item
	// 3), so a tag's subtree is examined once per render.
	blank map[ast.Node]bool
	// cycleKeys memoizes each group-less cycle's literal key; "" for a
	// cycle that has none.
	cycleKeys map[*ast.Cycle]string
	// scratch is reused for escaping.
	scratch []byte
}

func newRenderer(env *Env) (*renderer, *Error) {
	if env == nil || env.Meter == nil {
		return nil, &Error{Code: CodeInternal, Detail: "a render needs an Env with a Meter"}
	}
	d := env.Dialect
	if d == nil {
		d = &Dialect{}
	}
	lim := env.Limits
	if lim.PartialDepth <= 0 {
		lim.PartialDepth = DefaultPartialDepth
	}
	return &renderer{env: env, dialect: d, meter: env.Meter, lim: lim,
		blank: map[ast.Node]bool{}, cycleKeys: map[*ast.Cycle]string{}}, nil
}

// binding is a bound name's value, and whether it is safe from escaping.
type binding struct {
	v    value.Value
	safe bool
}

// renderFrame is the state of one template or partial being rendered. A
// partial gets its own, which is how it is isolated (ADR-004 D8 item 4):
// it sees its parameters, its own bindings and the globals, and changes
// nothing its caller sees.
type renderFrame struct {
	source string
	// scopes are the block- and partial-scoped bindings, innermost last
	// (ADR-004 D7 item 8): a partial's parameters at the bottom, then one
	// scope per loop open.
	scopes []map[string]binding
	// vars are the template-scoped bindings: assign and capture.
	vars map[string]binding
	// counters are increment's and decrement's, in their own namespace.
	counters map[string]int64
	// cycles is each cycle's position, by its key.
	cycles map[any]int
	// offsets is where each for loop stopped, by its name, for "offset:
	// continue".
	offsets map[string]int64
	// changed is the last body an ifchanged wrote.
	changed    string
	hasChanged bool
	// loops are the forloop objects of the for loops open, innermost
	// last, for parentloop.
	loops []value.Object
}

func newRenderFrame(source string) *renderFrame {
	return &renderFrame{
		source:   source,
		vars:     map[string]binding{},
		counters: map[string]int64{},
		cycles:   map[any]int{},
		offsets:  map[string]int64{},
	}
}

// ctl is how a block ended: normally, or interrupted by break or continue,
// which stops every enclosing block up to the nearest loop.
type ctl uint8

const (
	ctlNone ctl = iota
	ctlBreak
	ctlContinue
)

// sink is where writes go: the output writer, or a buffer.
type sink struct {
	w   io.Writer
	buf []byte
	// held is set on a capture's buffer, whose bytes are charged to
	// assign-bytes as they are written.
	held bool
}

// at makes n the statement errors are located at.
func (r *renderer) at(n ast.Node) { r.stmt = n }

// locate fills in e's position: the statement's, or, in Eval and
// EvalCondition, n's.
func (r *renderer) locate(e *Error, n ast.Node) *Error {
	sp := n.Pos()
	if r.stmt != nil {
		sp = r.stmt.Pos()
	}
	e.Source, e.Line, e.Col, e.Offset = r.fr.source, sp.Line, sp.Col, sp.Start
	return e
}

// fail returns an error located at n.
func (r *renderer) fail(n ast.Node, code Code, args map[string]string, detail string) *Error {
	return r.locate(&Error{Code: code, Args: args, Detail: detail}, n)
}

// adopt locates an error a definition, a Partials or the Meter returned,
// copying it so a shared error value is never changed.
func (r *renderer) adopt(e *Error, n ast.Node) *Error {
	c := *e
	return r.locate(&c, n)
}

// charge charges n units of w to the meter, for node at.
func (r *renderer) charge(w Work, n int64, at ast.Node) *Error {
	if e := r.meter.Charge(w, n); e != nil {
		return r.adopt(e, at)
	}
	return nil
}

// nodeCost is what each node evaluated costs (ADR-009 D11 item 2).
const nodeCost = 16

// limitErr returns a CodeLimit error naming limit.
func (r *renderer) limitErr(n ast.Node, limit string, detail string) *Error {
	return r.fail(n, CodeLimit, map[string]string{"limit": limit}, detail)
}

// checkString refuses a built string longer than value-size.
func (r *renderer) checkString(n int, at ast.Node) *Error {
	if r.lim.ValueSize > 0 && int64(n) > r.lim.ValueSize {
		return r.limitErr(at, "value-size", fmt.Sprintf("a string of %d bytes, above %d", n, r.lim.ValueSize))
	}
	return nil
}

// checkArray refuses a built array longer than array-size.
func (r *renderer) checkArray(n int64, at ast.Node) *Error {
	if r.lim.ArraySize > 0 && n > r.lim.ArraySize {
		return r.limitErr(at, "array-size", fmt.Sprintf("an array of %d elements, above %d", n, r.lim.ArraySize))
	}
	return nil
}

// write writes b to the current sink: to the output, charged as output
// bytes, or to a buffer, held to value-size. A capture's buffer is
// charged as held bytes as it fills, so the bytes of captures not yet
// ended are counted too (ADR-009 D2 item 2).
func (r *renderer) write(b []byte, at ast.Node) *Error {
	if len(b) == 0 {
		return nil
	}
	s := r.sinks[len(r.sinks)-1]
	if s.w == nil {
		if e := r.checkString(len(s.buf)+len(b), at); e != nil {
			return e
		}
		if s.held {
			if e := r.charge(WorkHeldBytes, int64(len(b)), at); e != nil {
				return e
			}
		}
		s.buf = append(s.buf, b...)
		return nil
	}
	if e := r.charge(WorkOutputBytes, int64(len(b)), at); e != nil {
		return e
	}
	if _, err := s.w.Write(b); err != nil {
		return r.fail(at, CodeWrite, nil, err.Error())
	}
	return nil
}

func (r *renderer) writeString(s string, at ast.Node) *Error {
	return r.write([]byte(s), at)
}

// writeValue writes v's text (ADR-004 D4 item 4), escaped when the render
// escapes and v is not safe (ADR-004 D9). An array writes its elements'
// texts in turn, and costs a unit per element; an object is refused.
func (r *renderer) writeValue(v value.Value, safe bool, at ast.Node) *Error {
	switch v := v.(type) {
	case value.Array:
		// Writing an array visits each element: one unit each, beside the
		// output bytes its texts are charged.
		if err := r.charge(WorkUnits, int64(v.Len()), at); err != nil {
			return err
		}
		for _, e := range v.All() {
			if err := r.writeValue(e, safe, at); err != nil {
				return err
			}
		}
		return nil
	case value.Object:
		return r.fail(at, CodeObjectOutput, map[string]string{"use": "json"}, "an object has no text; write it with json")
	}
	s := scalarText(v)
	if r.env.Escape == nil || safe {
		return r.writeString(s, at)
	}
	r.scratch = r.env.Escape(r.scratch[:0], s)
	return r.write(r.scratch, at)
}

// scalarText is the text of a value that is not an array or object
// (ADR-004 D4 item 4).
func scalarText(v value.Value) string {
	switch v := v.(type) {
	case value.Null:
		return ""
	case value.Bool:
		if v {
			return "true"
		}
		return "false"
	case value.Number:
		return numberText(v)
	case value.String:
		return v.Text()
	}
	panic(fmt.Sprintf("scalarText of %T", v))
}

// pushBuffer starts capturing writes, charged as held bytes when held is
// set; popBuffer ends it and returns them.
func (r *renderer) pushBuffer(held bool) { r.sinks = append(r.sinks, &sink{held: held}) }

func (r *renderer) popBuffer() []byte {
	s := r.sinks[len(r.sinks)-1]
	r.sinks = r.sinks[:len(r.sinks)-1]
	return s.buf
}

// block executes a block's nodes in order. dropText is set by a block tag
// the blank-body rule applies to, whose bodies' text is not written
// (ADR-004 D3 item 3).
func (r *renderer) block(b *ast.Block, dropText bool) (ctl, *Error) {
	if b == nil {
		return ctlNone, nil
	}
	for _, n := range b.Nodes {
		if t, ok := n.(*ast.Text); ok {
			if dropText {
				continue
			}
			if e := r.writeString(t.Trimmed(), t); e != nil {
				return ctlNone, e
			}
			continue
		}
		c, e := r.node(n)
		if e != nil || c != ctlNone {
			return c, e
		}
	}
	return ctlNone, nil
}

// node executes one statement. Every statement but text is a node
// evaluated, and costs nodeCost units (ADR-009 D11 item 2).
func (r *renderer) node(n ast.Node) (ctl, *Error) {
	r.at(n)
	if e := r.charge(WorkUnits, nodeCost, n); e != nil {
		return ctlNone, e
	}
	switch n := n.(type) {
	case *ast.Output:
		return ctlNone, r.output(n.Value, n)
	case *ast.Echo:
		return ctlNone, r.output(n.Value, n)
	case *ast.If:
		return r.ifTag(n.Branches, n.Else, false, n)
	case *ast.Unless:
		return r.ifTag(n.Branches, n.Else, true, n)
	case *ast.Case:
		return r.caseTag(n)
	case *ast.For:
		return r.forTag(n)
	case *ast.Tablerow:
		return r.tablerow(n)
	case *ast.Assign:
		return ctlNone, r.assign(n)
	case *ast.Capture:
		return r.capture(n)
	case *ast.Counter:
		return ctlNone, r.counter(n)
	case *ast.Cycle:
		return ctlNone, r.cycle(n)
	case *ast.Break:
		return ctlBreak, nil
	case *ast.Continue:
		return ctlContinue, nil
	case *ast.Ifchanged:
		return r.ifchanged(n)
	case *ast.Comment:
		return ctlNone, nil
	case *ast.Raw:
		return ctlNone, r.writeString(n.Text, n)
	case *ast.Liquid:
		return r.block(n.Body, false)
	case *ast.Render:
		return ctlNone, r.render(n)
	case *ast.Tag:
		return ctlNone, r.addedTag(n)
	}
	return ctlNone, r.fail(n, CodeInternal, nil, fmt.Sprintf("no renderer for %T", n))
}

// output writes an output's or an echo's value; nil writes nothing.
func (r *renderer) output(e *ast.Expr, at ast.Node) *Error {
	if e == nil {
		return nil
	}
	v, safe, err := r.expr(e)
	if err != nil {
		return err
	}
	return r.writeValue(v, safe, at)
}

// ifTag executes if, or unless, whose first branch runs when its
// condition is falsy (ADR-004 D2).
func (r *renderer) ifTag(branches []*ast.Branch, els *ast.Block, unless bool, n ast.Node) (ctl, *Error) {
	drop := r.isBlank(n)
	for i, b := range branches {
		r.at(b)
		ok, err := r.cond(b.Cond)
		if err != nil {
			return ctlNone, err
		}
		if i == 0 && unless {
			ok = !ok
		}
		if ok {
			return r.block(b.Body, drop)
		}
	}
	return r.block(els, drop)
}

// caseTag executes case: its clauses in order, rendering each when one of
// whose values equals the subject — once per value that does — and each
// else that no earlier when matched, as Shopify does.
func (r *renderer) caseTag(n *ast.Case) (ctl, *Error) {
	drop := r.isBlank(n)
	subject, err := r.operand(n.Subject)
	if err != nil {
		return ctlNone, err
	}
	elseLive := true
	for _, w := range n.Clauses {
		if w.Else {
			if elseLive {
				if c, e := r.block(w.Body, drop); e != nil || c != ctlNone {
					return c, e
				}
			}
			continue
		}
		for _, v := range w.Values {
			r.at(w)
			o, err := r.operand(v)
			if err != nil {
				return ctlNone, err
			}
			eq, err := r.compare(ast.OpEq, subject, o, w)
			if err != nil {
				return ctlNone, err
			}
			if eq {
				elseLive = false
				if c, e := r.block(w.Body, drop); e != nil || c != ctlNone {
					return c, e
				}
			}
		}
	}
	return ctlNone, nil
}

// seq is a collection a loop iterates: n items, the i-th given by at.
type seq struct {
	n  int64
	at func(i int64) value.Value
	// isRange is set for a range literal, whose span loop-iterations is
	// charged before its first iteration (ADR-009 D11 item 1).
	isRange bool
	// isString is set for a string, which Shopify's loops visit once
	// whatever their offset and limit (Utils.slice_collection).
	isString bool
}

// sequence returns what a for, tablerow or render … for iterates over p:
// a range literal's integers, lazily; an array's elements; an object's
// members as [name, value] pairs in canonical order; a non-empty string
// once; anything else not at all, as Shopify's loops do.
func (r *renderer) sequence(p ast.Primary, at ast.Node) (seq, value.Value, *Error) {
	if rg, ok := p.(*ast.Range); ok {
		from, to, err := r.rangeBounds(rg, at)
		if err != nil {
			return seq{}, nil, err
		}
		n := max(to-from+1, 0)
		return seq{n: n, isRange: true, at: func(i int64) value.Value { return intValue(from + i) }}, nil, nil
	}
	v, _, err := r.primary(p)
	if err != nil {
		return seq{}, nil, err
	}
	return r.valueSeq(v), v, nil
}

// valueSeq is the sequence a loop iterates over a value.
func (r *renderer) valueSeq(v value.Value) seq {
	switch v := v.(type) {
	case value.Array:
		return seq{n: int64(v.Len()), at: func(i int64) value.Value { return v.At(int(i)) }}
	case value.Object:
		member := func(i int) value.Member { return v.At(i) }
		if r.env.UnorderedObjects {
			order := unorderedMembers(v)
			member = func(i int) value.Member { return v.At(order[i]) }
		}
		return seq{n: int64(v.Len()), at: func(i int64) value.Value {
			m := member(int(i))
			return newArray([]value.Value{newString(m.Name), m.Value})
		}}
	case value.String:
		if v.Text() == "" {
			return seq{}
		}
		return seq{n: 1, isString: true, at: func(int64) value.Value { return v }}
	}
	return seq{}
}

// member returns an object's i-th member in the order every template
// iteration visits them: canonical order, ascending by the UTF-16 code
// units of the name (ADR-004 D4 item 5). It is the one place the engine
// orders an object's members for a template — for and tablerow over an
// object, render … for, and the first and last shorthands — and it reads
// the object's own ordered members, never a Go map, unless the render's
// Env sets UnorderedObjects (ADR-010 Amendment A2). A loop takes one
// order for its whole iteration (valueSeq).
func (r *renderer) member(o value.Object, i int) value.Member {
	if r.env.UnorderedObjects {
		return o.At(unorderedMembers(o)[i])
	}
	return o.At(i)
}

// unorderedMembers is the order of o's members that a Go map built from
// them iterates in: the engine's one iteration of a Go map, which only
// Env.UnorderedObjects reaches (ADR-010 D4 item 2, Amendment A2).
func unorderedMembers(o value.Object) []int {
	index := make(map[string]int, o.Len())
	for i := range o.Len() {
		index[o.At(i).Name] = i
	}
	order := make([]int, 0, len(index))
	for _, i := range index {
		order = append(order, i)
	}
	return order
}

// window is the part of s a loop visits, from offset from to before to
// (when hasTo), as Shopify slices a collection: item i is visited when
// from <= i and, with a limit, i < to. A string is visited whole.
func window(s seq, from int64, to int64, hasTo bool) (lo, hi int64) {
	if s.isString {
		return 0, s.n
	}
	n := s.n
	lo, hi = max(from, 0), n
	if hasTo {
		hi = min(hi, to)
	}
	if hi < lo {
		hi = lo
	}
	return lo, hi
}

// chargeLoop charges a loop's iterations before its first one: a range
// literal's whole span (ADR-009 D11 item 1), and otherwise the items the
// loop will visit.
func (r *renderer) chargeLoop(s seq, count int64, at ast.Node) *Error {
	n := count
	if s.isRange {
		n = s.n
	}
	return r.charge(WorkIterations, n, at)
}

// forTag executes for (ADR-004 D2): its offset, collection and limit,
// evaluated in that order, then offset, limit and reversed applied in
// that order, as Shopify applies them.
func (r *renderer) forTag(n *ast.For) (ctl, *Error) {
	drop := r.isBlank(n)
	var from int64
	switch {
	case n.OffsetContinue:
		from = r.fr.offsets[n.Name]
	case n.Offset != nil:
		v, err := r.loopInteger(n.Offset, "for", "offset", n)
		if err != nil {
			return ctlNone, err
		}
		from = v
	}
	s, _, err := r.sequence(n.Collection, n)
	if err != nil {
		return ctlNone, err
	}
	var to int64
	hasTo := false
	if n.Limit != nil {
		r.at(n)
		lv, _, err := r.primary(n.Limit)
		if err != nil {
			return ctlNone, err
		}
		if _, isNil := lv.(value.Null); !isNil {
			l, err := r.loopInt(lv, "for", "limit", n)
			if err != nil {
				return ctlNone, err
			}
			to, hasTo = l+from, true
		}
	}
	lo, hi := window(s, from, to, hasTo)
	count := hi - lo
	r.fr.offsets[n.Name] = from + count
	if count == 0 {
		return r.block(n.Else, drop)
	}
	if err := r.chargeLoop(s, count, n); err != nil {
		return ctlNone, err
	}
	var parent value.Value = value.Null{}
	if len(r.fr.loops) > 0 {
		parent = r.fr.loops[len(r.fr.loops)-1]
	}
	scope := map[string]binding{}
	r.fr.scopes = append(r.fr.scopes, scope)
	defer func() { r.fr.scopes = r.fr.scopes[:len(r.fr.scopes)-1] }()
	r.fr.loops = append(r.fr.loops, value.Object{})
	defer func() { r.fr.loops = r.fr.loops[:len(r.fr.loops)-1] }()
	for k := int64(0); k < count; k++ {
		i := lo + k
		if n.Reversed {
			i = hi - 1 - k
		}
		fl := forloop(n.Name, count, k, parent)
		r.fr.loops[len(r.fr.loops)-1] = fl
		scope[n.Var] = binding{v: s.at(i)}
		scope["forloop"] = binding{v: fl}
		c, err := r.block(n.Body, drop)
		if err != nil {
			return ctlNone, err
		}
		if c == ctlBreak {
			break
		}
	}
	return ctlNone, nil
}

// forloop is the forloop object of iteration k of count (ADR-004 D2).
func forloop(name string, count, k int64, parent value.Value) value.Object {
	return newObject([]value.Member{
		{Name: "first", Value: value.Bool(k == 0)},
		{Name: "index", Value: intValue(k + 1)},
		{Name: "index0", Value: intValue(k)},
		{Name: "last", Value: value.Bool(k == count-1)},
		{Name: "length", Value: intValue(count)},
		{Name: "name", Value: newString(name)},
		{Name: "parentloop", Value: parent},
		{Name: "rindex", Value: intValue(count - k)},
		{Name: "rindex0", Value: intValue(count - k - 1)},
	})
}

// loopInteger evaluates a for loop's offset: nil is 0.
func (r *renderer) loopInteger(p ast.Primary, tag, arg string, at ast.Node) (int64, *Error) {
	r.at(at)
	v, _, err := r.primary(p)
	if err != nil {
		return 0, err
	}
	if _, isNil := v.(value.Null); isNil {
		return 0, nil
	}
	return r.loopInt(v, tag, arg, at)
}

// loopInt is Shopify's Utils.to_integer, which for loops apply to their
// limit and offset, as the catalog's toInteger reads a count: an integer
// is itself, a string must be one written in decimal, and anything else
// — a float, a boolean, an array — is refused.
func (r *renderer) loopInt(v value.Value, tag, arg string, at ast.Node) (int64, *Error) {
	i, err := toInteger(tag, arg, v)
	switch {
	case err == nil:
		return i, nil
	case err.Code == CodeNumberRange:
		return 0, r.fail(at, CodeNumberRange, nil, err.Detail)
	}
	return 0, r.argErr(tag, arg, v, at)
}

// stringToI reads a string as Ruby's String#to_i does, as the catalog's
// leadingInteger reads it: 0 when it has no leading digits, and
// CodeNumberRange beyond ±(2^53−1).
func (r *renderer) stringToI(s value.String, at ast.Node) (int64, *Error) {
	n, err := leadingInteger("", s.Text())
	if err != nil {
		return 0, r.fail(at, CodeNumberRange, nil, err.Detail)
	}
	return n.i, nil
}

// argErr refuses a core tag's argument of the wrong type.
func (r *renderer) argErr(tag, arg string, v value.Value, at ast.Node) *Error {
	return r.fail(at, CodeArgument, map[string]string{"tag": tag, "argument": arg, "kind": kindName(v)},
		fmt.Sprintf("%s's %s is not an integer", tag, arg))
}

// toI is Ruby's to_i, which tablerow applies to its cols, limit and
// offset: nil is 0, a float is truncated, a string is read as String#to_i
// reads it, and a boolean, array or object is refused.
func (r *renderer) toI(v value.Value, arg string, at ast.Node) (int64, *Error) {
	switch v := v.(type) {
	case value.Null:
		return 0, nil
	case value.Number:
		if i, ok := templateInt(v); ok {
			return i, nil
		}
		f := v.Float64()
		if f != f || f > value.MaxSafeInteger || f < -value.MaxSafeInteger {
			return 0, r.fail(at, CodeNumberRange, nil, "a number beyond ±(2^53-1) made an integer")
		}
		return int64(f), nil
	case value.String:
		return r.stringToI(v, at)
	}
	return 0, r.argErr("tablerow", arg, v, at)
}

// tablerow executes tablerow (ADR-004 D2): an HTML table of its
// collection, its markup exactly as Shopify writes it. Its collection,
// offset, limit and cols are evaluated in that order; a collection that
// is nil or false writes nothing.
func (r *renderer) tablerow(n *ast.Tablerow) (ctl, *Error) {
	s, v, err := r.sequence(n.Collection, n)
	if err != nil {
		return ctlNone, err
	}
	if v != nil && !truthy(v) {
		return ctlNone, nil
	}
	attr := func(p ast.Primary, arg string) (int64, bool, *Error) {
		if p == nil {
			return 0, false, nil
		}
		r.at(n)
		v, _, err := r.primary(p)
		if err != nil {
			return 0, false, err
		}
		i, err := r.toI(v, arg, n)
		return i, true, err
	}
	from, _, err := attr(n.Offset, "offset")
	if err != nil {
		return ctlNone, err
	}
	limit, hasTo, err := attr(n.Limit, "limit")
	if err != nil {
		return ctlNone, err
	}
	lo, hi := window(s, from, from+limit, hasTo)
	count := hi - lo
	cols, hasCols, err := attr(n.Cols, "cols")
	if err != nil {
		return ctlNone, err
	}
	if !hasCols {
		cols = count
	}
	if cols == 0 && count > 0 {
		return ctlNone, r.fail(n, CodeArgument, map[string]string{"tag": "tablerow", "argument": "cols", "kind": "integer"},
			"tablerow's cols is zero")
	}
	if err := r.writeString("<tr class=\"row1\">\n", n); err != nil {
		return ctlNone, err
	}
	if count > 0 {
		if err := r.chargeLoop(s, count, n); err != nil {
			return ctlNone, err
		}
	}
	scope := map[string]binding{}
	r.fr.scopes = append(r.fr.scopes, scope)
	defer func() { r.fr.scopes = r.fr.scopes[:len(r.fr.scopes)-1] }()
	for k := int64(0); k < count; k++ {
		col0 := floorMod(k, cols)
		row := floorDiv(k, cols) + 1
		last := k == count-1
		colLast := col0+1 == cols
		scope[n.Var] = binding{v: s.at(lo + k)}
		scope["tablerowloop"] = binding{v: newObject([]value.Member{
			{Name: "col", Value: intValue(col0 + 1)},
			{Name: "col0", Value: intValue(col0)},
			{Name: "col_first", Value: value.Bool(col0 == 0)},
			{Name: "col_last", Value: value.Bool(colLast)},
			{Name: "first", Value: value.Bool(k == 0)},
			{Name: "index", Value: intValue(k + 1)},
			{Name: "index0", Value: intValue(k)},
			{Name: "last", Value: value.Bool(last)},
			{Name: "length", Value: intValue(count)},
			{Name: "rindex", Value: intValue(count - k)},
			{Name: "rindex0", Value: intValue(count - k - 1)},
			{Name: "row", Value: intValue(row)},
		})}
		r.at(n)
		if err := r.writeString("<td class=\"col"+strconv.FormatInt(col0+1, 10)+"\">", n); err != nil {
			return ctlNone, err
		}
		c, err := r.block(n.Body, false)
		if err != nil {
			return ctlNone, err
		}
		r.at(n)
		if err := r.writeString("</td>", n); err != nil {
			return ctlNone, err
		}
		if c == ctlBreak {
			break
		}
		if colLast && !last {
			if err := r.writeString("</tr>\n<tr class=\"row"+strconv.FormatInt(row+1, 10)+"\">", n); err != nil {
				return ctlNone, err
			}
		}
	}
	r.at(n)
	return ctlNone, r.writeString("</tr>\n", n)
}

// floorDiv and floorMod are Ruby's integer / and %, which floor; cols, by
// which tablerow divides, is never zero here.
func floorDiv(a, b int64) int64 {
	q := a / b
	if (a%b != 0) && ((a < 0) != (b < 0)) {
		q--
	}
	return q
}

func floorMod(a, b int64) int64 { return a - floorDiv(a, b)*b }

// assign binds a template-scoped name (ADR-004 D7 item 8). A string it
// binds is charged as held bytes (assign-bytes), once: a filter that
// built it charged none.
func (r *renderer) assign(n *ast.Assign) *Error {
	v, safe, err := r.expr(n.Value)
	if err != nil {
		return err
	}
	if s, ok := v.(value.String); ok {
		if err := r.charge(WorkHeldBytes, int64(len(s.Text())), n); err != nil {
			return err
		}
	}
	r.fr.vars[n.Name] = binding{v: v, safe: safe}
	return nil
}

// capture binds a template-scoped name to what its body writes. In a
// render that escapes, the captured text is already escaped, and is safe
// (ADR-004 D9 item 2). Its bytes are charged as held bytes as its body
// writes them.
func (r *renderer) capture(n *ast.Capture) (ctl, *Error) {
	r.pushBuffer(true)
	c, err := r.block(n.Body, false)
	b := r.popBuffer()
	if err != nil {
		return ctlNone, err
	}
	r.at(n)
	s, verr := value.NewString(string(b))
	if verr != nil {
		return ctlNone, r.fail(n, CodeInternal, nil, "a capture is not Unicode text")
	}
	r.fr.vars[n.Name] = binding{v: s, safe: r.env.Escape != nil}
	return c, nil
}

// counter executes increment, which writes its counter and then adds one,
// or decrement, which subtracts one and then writes it. Counters start at
// zero, in their own namespace (ADR-004 D2).
func (r *renderer) counter(n *ast.Counter) *Error {
	v := r.fr.counters[n.Name]
	if n.Decrement {
		v--
		r.fr.counters[n.Name] = v
	} else {
		r.fr.counters[n.Name] = v + 1
	}
	return r.writeString(strconv.FormatInt(v, 10), n)
}

// cycleKey is a grouped cycle's key: its group's kind and text.
type cycleKey struct{ kind, text string }

// cycle writes the value at its position and advances it (ADR-004 D2).
// Cycles share a position when they have the same group value, or, with
// no group, when they are written with the same literals; a cycle with no
// group and a variable among its values has a position of its own, as in
// Shopify. A position beyond a shorter list writes nothing.
func (r *renderer) cycle(n *ast.Cycle) *Error {
	var key any = n
	if n.Group != nil {
		g, _, err := r.primary(n.Group)
		if err != nil {
			return err
		}
		if !composite(g) {
			key = cycleKey{kind: kindName(g), text: scalarText(g)}
		}
		// An array or object group keeps the cycle's own position.
	} else if lits, ok := r.literalKey(n); ok {
		key = lits
	}
	i := r.fr.cycles[key]
	next := i + 1
	if next >= len(n.Values) {
		next = 0
	}
	r.fr.cycles[key] = next
	if i >= len(n.Values) {
		return nil
	}
	v, safe, err := r.primary(n.Values[i])
	if err != nil {
		return err
	}
	return r.writeValue(v, safe, n)
}

// literalKey is the key of a group-less cycle whose values are all
// literals: their kinds and texts. It is made once per render, so a cycle
// in a loop costs no more per iteration than its value.
func (r *renderer) literalKey(n *ast.Cycle) (string, bool) {
	if k, ok := r.cycleKeys[n]; ok {
		return k, k != ""
	}
	var b []byte
	for _, v := range n.Values {
		l, ok := v.(*ast.Literal)
		if !ok {
			r.cycleKeys[n] = ""
			return "", false
		}
		b = strconv.AppendInt(b, int64(l.Kind), 10)
		b = append(b, ':')
		b = strconv.AppendQuote(b, l.Text)
		b = append(b, ',')
	}
	r.cycleKeys[n] = string(b)
	return string(b), true
}

// ifchanged writes its body when it differs from the last body an
// ifchanged of this template wrote. The body is held while it is
// compared, to value-size as every buffer is; it is not stored by assign
// or capture, so it charges no held bytes (ADR-009 Amendment A10 item 4).
func (r *renderer) ifchanged(n *ast.Ifchanged) (ctl, *Error) {
	r.pushBuffer(false)
	c, err := r.block(n.Body, false)
	b := r.popBuffer()
	if err != nil {
		return ctlNone, err
	}
	r.at(n)
	if s := string(b); !r.fr.hasChanged || s != r.fr.changed {
		r.fr.changed, r.fr.hasChanged = s, true
		if err := r.write(b, n); err != nil {
			return ctlNone, err
		}
	}
	return c, nil
}

// render renders a partial (ADR-004 D8): with no value once; with a
// "with" value once, the value bound to the alias; with a "for" value
// once per item of an array, an object or a range, the item bound to the
// alias and forloop bound, and otherwise once with the value bound. The
// alias is bound even to nil, as the partial-scoped binding ADR-004 D7
// item 8 makes it, where Shopify leaves it unbound. Parameters are
// evaluated once, in the caller's scope, and the alias shadows a
// parameter of its name.
func (r *renderer) render(n *ast.Render) *Error {
	if r.env.Partials == nil {
		return r.fail(n, CodeInternal, nil, "a render tag with no Partials")
	}
	if r.depth >= r.lim.PartialDepth {
		return r.limitErr(n, "partial-depth", fmt.Sprintf("more than %d render calls active at once", r.lim.PartialDepth))
	}
	scope := map[string]binding{}
	for _, p := range n.Params {
		v, safe, err := r.primary(p.Value)
		if err != nil {
			return err
		}
		scope[p.Name] = binding{v: v, safe: safe}
	}
	var s seq
	var v value.Value
	safe := false
	switch n.Mode {
	case ast.RenderWith:
		var err *Error
		if v, safe, err = r.primary(n.Value); err != nil {
			return err
		}
	case ast.RenderFor:
		var err *Error
		if s, v, err = r.sequence(n.Value, n); err != nil {
			return err
		}
	}
	r.at(n)
	tpl, perr := r.env.Partials.Partial(n.Name)
	if perr != nil {
		return r.adopt(perr, n)
	}
	if tpl == nil || tpl.Root == nil {
		return r.fail(n, CodeInternal, nil, "Partials returned no template and no error")
	}
	iterate := n.Mode == ast.RenderFor && (s.isRange || composite(v))
	if !iterate {
		if n.Mode != ast.RenderPlain {
			scope[n.Alias] = binding{v: v, safe: safe}
		}
		return r.callPartial(n, tpl, scope)
	}
	if s.n > 0 {
		if err := r.chargeLoop(s, s.n, n); err != nil {
			return err
		}
	}
	item := map[string]binding{}
	for k := int64(0); k < s.n; k++ {
		item["forloop"] = binding{v: forloop(n.Name, s.n, k, value.Null{})}
		item[n.Alias] = binding{v: s.at(k)}
		if err := r.callPartial(n, tpl, scope, item); err != nil {
			return err
		}
	}
	return nil
}

// callPartial renders tpl in a frame of its own, with scopes as its
// partial-scoped bindings, innermost last. An error inside it carries
// this call's site.
func (r *renderer) callPartial(n *ast.Render, tpl *Template, scopes ...map[string]binding) *Error {
	r.at(n)
	if err := r.charge(WorkPartials, 1, n); err != nil {
		return err
	}
	site := Site{Source: r.fr.source, Line: n.Span.Line, Col: n.Span.Col, Offset: n.Span.Start}
	caller := r.fr
	r.fr = newRenderFrame(tpl.Name)
	r.fr.scopes = scopes
	r.depth++
	_, err := r.block(tpl.Root, false)
	r.depth--
	r.fr = caller
	r.at(n)
	if err != nil {
		err.Chain = append([]Site{site}, err.Chain...)
	}
	return err
}

// addedTag executes a tag the dialect adds, through its definition.
func (r *renderer) addedTag(n *ast.Tag) *Error {
	def, ok := r.dialect.Tag(n.Name)
	if !ok {
		return r.fail(n, CodeSyntax, nil, fmt.Sprintf("the render's dialect has no tag %q", n.Name))
	}
	if err := def.Render(tagContext{r: r, n: n}, n); err != nil {
		return r.adopt(err, n)
	}
	return nil
}

// tagContext is the TagContext a TagDef renders through.
type tagContext struct {
	r *renderer
	n *ast.Tag
}

func (c tagContext) Eval(e ast.Primary) (value.Value, *Error) {
	c.r.at(c.n)
	v, _, err := c.r.primary(e)
	return v, err
}

func (c tagContext) Write(b []byte) *Error {
	c.r.at(c.n)
	return c.r.write(b, c.n)
}

func (c tagContext) Meter() Meter { return c.r.meter }

// isBlank reports whether the blank-body rule applies to a block tag:
// whether all its bodies are blank (ADR-004 D3 item 3, Context).
func (r *renderer) isBlank(n ast.Node) bool {
	if b, ok := r.blank[n]; ok {
		return b
	}
	var b bool
	switch n := n.(type) {
	case *ast.If:
		b = r.blankBranches(n.Branches, n.Else)
	case *ast.Unless:
		b = r.blankBranches(n.Branches, n.Else)
	case *ast.Case:
		b = r.blankBlock(n.Before)
		for _, w := range n.Clauses {
			b = b && r.blankBlock(w.Body)
		}
	case *ast.For:
		b = r.blankBlock(n.Body) && r.blankBlock(n.Else)
	case *ast.Tablerow:
		b = r.blankBlock(n.Body)
	case *ast.Ifchanged:
		b = r.blankBlock(n.Body)
	}
	r.blank[n] = b
	return b
}

func (r *renderer) blankBranches(bs []*ast.Branch, els *ast.Block) bool {
	for _, br := range bs {
		if !r.blankBlock(br.Body) {
			return false
		}
	}
	return r.blankBlock(els)
}

// blankBlock reports whether a body holds only whitespace and blank tags:
// assign, capture, a comment, an empty raw or doc, and block tags whose
// own bodies are blank. Every other tag, and every output, is not blank.
func (r *renderer) blankBlock(b *ast.Block) bool {
	if b == nil {
		return true
	}
	for _, n := range b.Nodes {
		switch n := n.(type) {
		case *ast.Text:
			for i := 0; i < len(n.Text); i++ {
				if !ast.IsSpace(n.Text[i]) {
					return false
				}
			}
		case *ast.Assign, *ast.Capture:
		case *ast.Comment:
			if n.Kind == ast.CommentDoc && n.Text != "" {
				return false
			}
		case *ast.Raw:
			if n.Text != "" {
				return false
			}
		case *ast.If, *ast.Unless, *ast.Case, *ast.For, *ast.Tablerow, *ast.Ifchanged:
			if !r.isBlank(n) {
				return false
			}
		case *ast.Liquid:
			if !r.blankBlock(n.Body) {
				return false
			}
		default:
			return false
		}
	}
	return true
}
