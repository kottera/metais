// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"errors"

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

// validateStage is ADR-005 D11: a file whose declaration names a schema
// in validate has its final bytes validated against it.
var validateStage = Stage{
	Name:    "validate",
	Applies: func(d manifest.Output) bool { return d.Validate != "" },
	Run: func(c *Staging, f *File) []finding.Finding {
		return validateOutput(c.pkg, f, c.schemas.byName, c.lim)
	},
}

// bom is the UTF-8 byte order mark a YAML output may begin with.
var bom = []byte("\uFEFF")

// validateOutput is ADR-005 D11 items 2 to 4 for pf: its kind is json or
// yaml, its bytes parse by that kind's rules, and the value validates
// against the schema validate names. Each finding is about the output,
// located by its path (a container's entry by the container's path and
// its own), the JSON Pointer into it, and its line where the parser gives
// one.
func validateOutput(pkg *loader.Package, pf *File, schemas map[string]*schema.Compiler, lim limits.Set) []finding.Finding {
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
		return []finding.Finding{finding.New(finding.CodeDeclarationValidateKind, subj, &l, finding.Params{"kind": finding.String(kind)})}
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
	// The values the output decodes to are counted against output-values,
	// its own and no other output's, since each output's value is released
	// before the next is validated; a YAML output is bounded by yaml-size
	// before it is parsed (ADR-009 Amendment A6 item 3).
	values := lim.Counter(limits.OutputValues)
	if kind == "json" {
		doc, fs := jsonx.Parse(pf.bytes, jsonx.Source{Input: reader, At: at, Output: true, Values: values}, lim)
		if len(fs) > 0 {
			return outputParseFindings(fs, subj, kind)
		}
		v, pos = doc.Value, doc.Positions()
	} else {
		if n := int64(len(pf.bytes)); n > lim.Value(limits.YAMLSize) {
			return []finding.Finding{lim.ExceededOutput(limits.YAMLSize, subj, at, n, true)}
		}
		doc, fs := yamlx.Parse(bytes.TrimPrefix(pf.bytes, bom), yamlx.Source{Subject: reader, At: at, Size: limits.YAMLSize, Values: values}, lim)
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
		panic("render: validating an output: " + err.Error())
	}
	fs, err := schemafail.Map(schemafail.Target{
		Family:  schemafail.FamilyOutput,
		Subject: subj,
		Schemas: c,
		Listed:  int(min(lim.Value(limits.Findings), int64(^uint(0)>>1))),
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
		panic("render: mapping an output's failures: " + err.Error())
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
