// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"errors"
	"maps"
	"slices"

	"github.com/kottera/metais/internal/liquid"
	"github.com/kottera/metais/internal/liquid/ast"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/template"
)

// HeldOut begins the message of every compile the server holds out of
// the run: one whose template, or a file of its file system, applies
// date. date is Wave 5's (plan §8), so until it is registered its cases
// are out of the delegated run, as goldenDateSelector holds Golden
// Liquid's out, and TestDelegatedRun counts them apart from the
// departures. A held-out compile is answered as a parse error, so
// liquid-spec reports it among its failures with this text in it.
const HeldOut = "held out of the run: the template applies date, which the dialect does not yet register"

// errDateRegistered is the refusal to serve once date is registered: its
// cases are then in scope, and the hold-out must go rather than keep
// them out.
var errDateRegistered = errors.New("the dialect registers date, so liquid-spec's date cases are in scope: remove the hold-out of heldout.go")

// dateProbe is the dialect with a stub date, under which a template that
// applies date parses, so its tree can be walked for the application.
func dateProbe() (*liquid.Dialect, error) {
	d := template.Dialect()
	if _, ok := d.Filter("date"); ok {
		return nil, errDateRegistered
	}
	return d.Extend(liquid.Spec{Filters: []liquid.FilterDef{stubFilter("date")}})
}

// appliesDate reports whether src, or a file of fs, applies date under
// probe. The selection is mechanical: it names no case.
func appliesDate(probe *liquid.Dialect, fs map[string]string, src string) bool {
	srcs := []string{src}
	for _, name := range slices.Sorted(maps.Keys(fs)) {
		srcs = append(srcs, fs[name])
	}
	for _, s := range srcs {
		t, _ := liquid.Parse(probe, liquid.Source{Text: []byte(s)}, liquid.ParseLimits{})
		if t == nil {
			continue
		}
		found := false
		ast.Walk(t.Root, func(n ast.Node) bool {
			if f, ok := n.(*ast.Filter); ok && f.Name == "date" {
				found = true
			}
			return !found
		})
		if found {
			return true
		}
	}
	return false
}

// stubFilter is a filter that only has a name.
type stubFilter string

func (f stubFilter) Name() string                  { return string(f) }
func (stubFilter) Check(*ast.Filter) *liquid.Error { return nil }
func (stubFilter) Apply(liquid.FilterContext, value.Value, []liquid.Arg) (value.Value, *liquid.Error) {
	return nil, &liquid.Error{Code: liquid.CodeInternal, Detail: "a stub filter does not render"}
}
