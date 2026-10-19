// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ast

// Walk visits n and every node beneath it, depth first and in source
// order: a node before its children, and children in the order their text
// appears. visit returns whether to descend into the node it was given; a
// nil node, and a nil field of a node, is not visited.
//
// Every node of the tree is reached: a Block's nodes; a tag's conditions,
// values, filters, arguments, segments and bodies; and the Branches, Whens
// and Segments that hold them.
func Walk(n Node, visit func(Node) bool) {
	if isNil(n) || !visit(n) {
		return
	}
	switch n := n.(type) {
	case *Block:
		for _, c := range n.Nodes {
			Walk(c, visit)
		}
	case *Text, *Literal, *Break, *Continue, *Comment, *Raw:
	case *Output:
		walkExpr(n.Value, visit)
	case *Expr:
		Walk(n.Base, visit)
		for _, f := range n.Filters {
			Walk(f, visit)
		}
	case *Filter:
		walkArgs(n.Args, visit)
	case *Arg:
		Walk(n.Value, visit)
	case *Path:
		for _, s := range n.Segments {
			Walk(s, visit)
		}
	case *Segment:
		Walk(n.Index, visit)
	case *Range:
		Walk(n.From, visit)
		Walk(n.To, visit)
	case *Cond:
		switch n.Kind {
		case CondAnd, CondOr:
			walkCond(n.X, visit)
			walkCond(n.Y, visit)
		default:
			Walk(n.Left, visit)
			Walk(n.Right, visit)
		}
	case *Branch:
		walkCond(n.Cond, visit)
		walkBlock(n.Body, visit)
	case *If:
		for _, b := range n.Branches {
			Walk(b, visit)
		}
		walkBlock(n.Else, visit)
	case *Unless:
		for _, b := range n.Branches {
			Walk(b, visit)
		}
		walkBlock(n.Else, visit)
	case *Case:
		Walk(n.Subject, visit)
		walkBlock(n.Before, visit)
		for _, w := range n.Clauses {
			Walk(w, visit)
		}
	case *When:
		for _, v := range n.Values {
			Walk(v, visit)
		}
		walkBlock(n.Body, visit)
	case *For:
		Walk(n.Collection, visit)
		// The modifiers in the order Shopify's grammar admits them.
		Walk(n.Limit, visit)
		Walk(n.Offset, visit)
		walkBlock(n.Body, visit)
		walkBlock(n.Else, visit)
	case *Tablerow:
		Walk(n.Collection, visit)
		Walk(n.Cols, visit)
		Walk(n.Limit, visit)
		Walk(n.Offset, visit)
		walkBlock(n.Body, visit)
	case *Assign:
		walkExpr(n.Value, visit)
	case *Capture:
		walkBlock(n.Body, visit)
	case *Counter:
	case *Cycle:
		Walk(n.Group, visit)
		for _, v := range n.Values {
			Walk(v, visit)
		}
	case *Echo:
		walkExpr(n.Value, visit)
	case *Ifchanged:
		walkBlock(n.Body, visit)
	case *Liquid:
		walkBlock(n.Body, visit)
	case *Render:
		Walk(n.Value, visit)
		walkArgs(n.Params, visit)
	case *Tag:
		walkArgs(n.Args, visit)
	}
}

// The helpers below keep a nil pointer of a concrete type from becoming a
// non-nil Node.

func walkExpr(e *Expr, visit func(Node) bool) {
	if e != nil {
		Walk(e, visit)
	}
}

func walkCond(c *Cond, visit func(Node) bool) {
	if c != nil {
		Walk(c, visit)
	}
}

func walkBlock(b *Block, visit func(Node) bool) {
	if b != nil {
		Walk(b, visit)
	}
}

func walkArgs(args []*Arg, visit func(Node) bool) {
	for _, a := range args {
		Walk(a, visit)
	}
}

// isNil reports whether n is nil or holds a nil pointer.
func isNil(n Node) bool {
	switch n := n.(type) {
	case nil:
		return true
	case *Literal:
		return n == nil
	case *Path:
		return n == nil
	case *Range:
		return n == nil
	}
	return false
}
