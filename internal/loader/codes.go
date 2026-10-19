// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/unitext"
	"github.com/kottera/metais/internal/zipfmt"
)

// The codes the loader reports beside its libraries' (ADR-001 Interfaces,
// ADR-002 D1, ADR-003 D1 item 4, ADR-008 D2 item 1). The package family's
// are class refused, about the package; the manifest family's are class
// content, about the manifest.
const (
	CodeNotFound           finding.Code = "package.not_found"
	CodeNotReadable        finding.Code = "package.not_readable"
	CodeUnknownTopLevel    finding.Code = "package.layout.unknown_top_level"
	CodeMissingManifest    finding.Code = "package.layout.missing_manifest"
	CodeCompiledInSource   finding.Code = "package.layout.compiled_in_source"
	CodeEmptyDirectory     finding.Code = "package.layout.empty_directory"
	CodeNonRegularFile     finding.Code = "package.layout.non_regular_file"
	CodeCompiledMissing    finding.Code = "package.compiled.missing"
	CodeCompiledUnknown    finding.Code = "package.compiled.unknown_file"
	CodeCompiledMismatch   finding.Code = "package.compiled.mismatch"
	CodeManifestWrongName  finding.Code = "manifest.file.wrong_name"
	CodeManifestPathAbsent finding.Code = "manifest.path_not_found"
	CodeSchemaSyntax       finding.Code = "schema.syntax"
)

var (
	pkgSubject      = finding.Subject{Kind: finding.SubjectPackage}
	manifestSubject = finding.Subject{Kind: finding.SubjectManifest}
)

// entryLocation is the location of the package entry name as a whole.
func entryLocation(name string) finding.Location {
	return finding.Location{Origin: finding.OriginPackage, File: name}
}

// entryFinding is a finding of a package-family code about the entry name.
func entryFinding(code finding.Code, name string, p finding.Params) finding.Finding {
	l := entryLocation(name)
	return finding.New(code, pkgSubject, &l, p)
}

// sourceFindings maps hostfs's refusal of a source directory to the
// findings a packed package with the same defects is refused with (ADR-001
// D5 items 5 and 6, D2): each problem in the walk's order, then the names
// the portable-path predicate refuses.
func sourceFindings(e *hostfs.SourceError, lim limits.Set) []finding.Finding {
	var out []finding.Finding
	for _, p := range e.Problems {
		switch p.Kind {
		case hostfs.ProblemNonRegularFile:
			out = append(out, entryFinding(CodeNonRegularFile, p.Path, nil))
		case hostfs.ProblemEmptyDirectory:
			out = append(out, entryFinding(CodeEmptyDirectory, p.Path, nil))
		case hostfs.ProblemCompiledInSource:
			out = append(out, entryFinding(CodeCompiledInSource, p.Path, nil))
		case hostfs.ProblemMissingManifest:
			out = append(out, finding.New(CodeMissingManifest, pkgSubject, nil, nil))
		case hostfs.ProblemNameShape:
			out = append(out, zipfmt.NameFindings(&unitext.TreeError{
				Paths: []unitext.PathProblems{{Path: p.Path, Problems: unitext.CheckPath(p.Path, lim)}},
			}, lim)...)
		case hostfs.ProblemLimit:
			out = append(out, lim.Breached(p.Limit, entryLocation(p.Path), 0, false))
		case hostfs.ProblemUnreadable:
			out = append(out, notReadable(p.Reason))
		}
	}
	var te *unitext.TreeError
	if e.Names != nil {
		if t, ok := e.Names.(*unitext.TreeError); ok {
			te = t
		}
	}
	if te != nil {
		out = append(out, zipfmt.NameFindings(te, lim)...)
	}
	return out
}

// wrongManifestName reports whether the top-level name is a spelling of
// the manifest's name that is not its one name (ADR-002 D1 item 3):
// metais.json, metais.yml, or metais.yaml in any other letter case,
// compared byte for byte.
func wrongManifestName(name string) bool {
	if name == "metais.yaml" {
		return false
	}
	return name == "metais.json" || name == "metais.yml" || strings.EqualFold(name, "metais.yaml")
}

func sortStrings(s []string) { slices.Sort(s) }
