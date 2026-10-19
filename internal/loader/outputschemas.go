// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"slices"
	"strings"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/pkgtree"
	"github.com/kottera/metais/internal/regexprofile"
	"github.com/kottera/metais/internal/schema"
)

// OutputSchemas compiles every output validation schema the declarations
// outs name, at every depth of a container (ADR-005 D11 items 3 and 5),
// and returns the compiler holding each, by its validate path. The files
// beneath one top-level directory are compiled together, that directory
// read as ADR-003 D5 item 5 reads inputs/; beneath inputs/, every schema
// file there is loaded with them (ADR-003 D3 item 1). A validate path that
// names no entry of tree is the manifest paths' finding, and is left out.
// Patterns are charged to meter, the command's matching budget. Every
// finding of every file is reported, and the map is nil when there is
// any.
func OutputSchemas(tree pkgtree.Tree, outs []manifest.Output, lim limits.Set, meter *regexprofile.Meter) (map[string]*schema.Compiler, []finding.Finding) {
	byScope := map[string][]string{}
	var walk func([]manifest.Output)
	walk = func(outs []manifest.Output) {
		for _, d := range outs {
			if d.Validate != "" {
				if _, err := tree.Stat(d.Validate); err == nil {
					scope := ""
					if i := strings.IndexByte(d.Validate, '/'); i >= 0 {
						scope = d.Validate[:i+1]
					}
					byScope[scope] = append(byScope[scope], d.Validate)
				}
			}
			if d.Container != nil {
				walk(d.Container.Entries)
			}
		}
	}
	walk(outs)
	if len(byScope) == 0 {
		return nil, nil
	}
	scopes := make([]string, 0, len(byScope))
	for s := range byScope {
		scopes = append(scopes, s)
	}
	slices.Sort(scopes)
	out := map[string]*schema.Compiler{}
	var fs []finding.Finding
	for _, scope := range scopes {
		roots := byScope[scope]
		if scope == schema.InputsDir {
			for _, e := range tree.List() {
				if schema.IsSchemaFile(e) {
					roots = append(roots, e)
				}
			}
		}
		c, cfs := schema.NewOutputCompiler(tree, scope, roots, lim, decoder(lim), meter)
		fs = append(fs, cfs...)
		for _, r := range byScope[scope] {
			out[r] = c
		}
	}
	if len(fs) > 0 {
		return nil, fs
	}
	return out, nil
}
