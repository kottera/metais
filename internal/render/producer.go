// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"io"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/zipfmt"
)

// Producer is where a planned file's bytes come from (ADR-010 A7 item 5).
// Every phase after planning asks a file's producer, never its form.
type Producer interface {
	// Record is the record's source of a file it produces, and whether
	// the file has a kind (ADR-005 D16 item 2, D7 item 3).
	Record() (source string, hasKind bool)
	// Reach is the templates f renders from, each parsed, with every
	// template it reaches, before anything renders.
	Reach(f *File) []string
	// Produce is phase 4 for f: render it and run its stages, charge it
	// against the output limits, or measure it. Its findings are f's.
	Produce(c *Producing, f *File) []finding.Finding
	// Open is the reader of f's bytes as they are emitted, and done
	// reports the error of their writing, once the reader is finished
	// with.
	Open(f *File) (r io.Reader, done func(read error) error)
	// Method is how f is written as a container's entry (ADR-005 D8 item
	// 7).
	Method(f *File) zipfmt.Method
}

// The producers, which the form that expands a file sets.
type production int

const (
	byRendering production = iota
	byCopying
	byAssembling
	byLiteral
)

// producerOf is the producer of a production: the table of producers.
func producerOf(p production) Producer {
	switch p {
	case byRendering:
		return rendered{}
	case byCopying:
		return copied{}
	case byAssembling:
		return assembled{}
	case byLiteral:
		return literal{}
	}
	panic("render: no producer")
}

// declaredMethod is the method of an entry its declaration sets: stored
// when it says stored: true, and otherwise ADR-001 D4's auto (ADR-005 D8
// item 7).
func declaredMethod(f *File) zipfmt.Method {
	if f.decl.Stored {
		return zipfmt.Store
	}
	return zipfmt.Auto
}
