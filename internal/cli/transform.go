// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"errors"

	"github.com/kottera/metais/internal/author"
	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/loader"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/schemafail"
	"github.com/kottera/metais/internal/yamlx"
)

// outputSchemas compiles every output validation schema of pkg's
// declarations (ADR-005 D11 item 3), charged to the command's matching
// budget. A schema that does not compile refuses the render with its
// findings.
func outputSchemas(pkg *loader.Package, lim limits.Set) (map[string]*schema.Compiler, []finding.Finding) {
	var meter *regexprofile.Meter
	if pkg.Schemas != nil {
		meter = pkg.Schemas.Meter()
	}
	return loader.OutputSchemas(pkg.Tree, pkg.Manifest.Outputs, lim, meter)
}

// transform is ADR-005 D13 item 4's transforms of pf, a file that
// rendered, top-level or a container's entry: D10's CRLF conversion, then
// D11's validation of its final bytes against the declaration's schema,
// every finding of which is reported (D11 item 4). It runs before a
// container holding pf is measured, so the container holds the bytes
// transformed.
func transform(pkg *loader.Package, pf *planned, schemas map[string]*schema.Compiler, lim limits.Set) []finding.Finding {
	if pf.decl.Newline == "crlf" {
		pf.bytes = crlf(pf.bytes)
	}
	if pf.decl.Validate == "" {
		return nil
	}
	return validateOutput(pkg, pf, schemas, lim)
}

// crlf is ADR-005 D10 item 1: every LF not immediately preceded by CR
// becomes CR LF. A CR LF and a lone CR are left as they are, so the
// conversion is idempotent.
func crlf(b []byte) []byte {
	n := 0
	for i, c := range b {
		if c == '\n' && (i == 0 || b[i-1] != '\r') {
			n++
		}
	}
	if n == 0 {
		return b
	}
	out := make([]byte, 0, len(b)+n)
	for i, c := range b {
		if c == '\n' && (i == 0 || b[i-1] != '\r') {
			out = append(out, '\r')
		}
		out = append(out, c)
	}
	return out
}

// bom is the UTF-8 byte order mark a YAML output may begin with.
var bom = []byte("\uFEFF")

// validateOutput is ADR-005 D11 items 2 to 4 for pf: its kind is json or
// yaml, its bytes parse by that kind's rules, and the value validates
// against the schema validate names. Each finding is about the output,
// located by its path (a container's entry by the container's path and
// its own), the JSON Pointer into it, and its line where the parser gives
// one.
func validateOutput(pkg *loader.Package, pf *planned, schemas map[string]*schema.Compiler, lim limits.Set) []finding.Finding {
	subj := finding.Subject{Kind: finding.SubjectOutput, Name: pf.owner}
	kind := kindOf(pf.path, pf.decl.Kind)
	if kind != "json" && kind != "yaml" {
		// The kind a path template gives is the render's to judge
		// (ADR-005 D11 item 2).
		l := finding.Location{Origin: finding.OriginPackage, File: manifest.ManifestFile}.WithPointer(pf.decl.Pointer + "/validate")
		if pkg.Positions != nil {
			if pos, ok := pkg.Positions.Value(pf.decl.Pointer + "/validate"); ok {
				l = l.At(pos)
			}
		}
		return []finding.Finding{finding.New(author.CodeValidateKind, subj, &l, finding.Params{"kind": finding.String(kind)})}
	}
	c, ok := schemas[pf.decl.Validate]
	if !ok {
		// The schema did not compile, which refused the render already.
		return nil
	}
	file := pf.path
	if pf.in != "" {
		file = pf.in + "/" + pf.path
	}
	at := finding.Location{Origin: finding.OriginOutput, File: file}
	// The readers report in an input's codes, which outputParseFindings makes
	// the output's.
	reader := finding.Subject{Kind: finding.SubjectInput, Name: pf.owner}
	var v value.Value
	var pos finding.Positions
	if kind == "json" {
		doc, fs := jsonx.Parse(pf.bytes, jsonx.Source{Input: reader, At: at, Output: true}, lim)
		if len(fs) > 0 {
			return outputParseFindings(fs, subj, kind)
		}
		v, pos = doc.Value, doc.Positions()
	} else {
		doc, fs := yamlx.Parse(bytes.TrimPrefix(pf.bytes, bom), yamlx.Source{Subject: reader, At: at, Size: limits.OutputSize}, lim)
		if len(fs) > 0 {
			return outputParseFindings(fs, subj, kind)
		}
		v, pos = doc.Value, doc.Positions()
	}
	s, _ := c.Schema(pf.decl.Validate)
	failures, err := s.Validate(v)
	var le *regexprofile.LimitError
	switch {
	case errors.As(err, &le):
		// The command's matching budget or its schema-work is spent, so
		// the output's validation has no answer.
		return []finding.Finding{lim.ExceededOutput(le.Limit, subj, at.WithPointer(""), 0, false)}
	case err != nil:
		panic("cli: validating an output: " + err.Error())
	}
	fs, err := schemafail.Map(schemafail.Target{
		Family:  schemafail.FamilyOutput,
		Subject: subj,
		Schemas: c,
		Locate: func(p *finding.Path, key bool) finding.Location {
			l := at
			got, ok := finding.ValueAt(pos, p)
			if key {
				got, ok = finding.KeyAt(pos, p)
			}
			if ok {
				l = l.At(got)
			}
			return l
		},
	}, failures...)
	if err != nil {
		panic("cli: mapping an output's failures: " + err.Error())
	}
	return fs
}

// outputParseFindings makes the findings a reader gave an output's text the
// output's (ADR-005 D11 item 4): a breach of a limit is
// limit.exceeded.output, and every other refusal output.parse, naming the
// code an input breaking the same rule gets. Each keeps its location in
// the output and its related locations.
func outputParseFindings(fs []finding.Finding, subj finding.Subject, form string) []finding.Finding {
	out := make([]finding.Finding, 0, len(fs))
	for _, f := range fs {
		loc, _ := f.Location()
		opts := []finding.Option{finding.WithRelated(f.RelatedAt()...)}
		if f.Code() == finding.CodeLimitExceeded {
			p := finding.Params{}
			for _, n := range []string{"limit", "bound", "lowered", "observed"} {
				if v, ok := f.Param(n); ok {
					p[n] = v
				}
			}
			out = append(out, finding.New(finding.CodeLimitExceededOutput, subj, &loc, p, opts...))
			continue
		}
		out = append(out, finding.New(codeOutputParse, subj, &loc, finding.Params{
			"form": finding.String(form),
			"rule": finding.String(string(f.Code())),
		}, opts...))
	}
	return out
}

// codeOutputParse is an output whose text its kind's reader refuses
// (ADR-005's interfaces name it).
const codeOutputParse finding.Code = "output.parse"
