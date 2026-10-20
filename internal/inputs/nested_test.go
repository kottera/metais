// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// listedWithin is how many findings f lists: itself, and every finding
// within its alternatives at any depth (MTS-FND-020).
func listedWithin(f finding.Finding) int {
	n := 1
	for _, a := range f.Alternatives() {
		for _, g := range a.Findings {
			n += listedWithin(g)
		}
	}
	return n
}

// TestNestedFindingsBounded is MTS-FND-020 and MTS-FND-021 as the memory
// a failing anyOf or oneOf holds (W7-NB3): the finding validation builds
// for it lists at most the findings limit, itself and the findings within
// its alternatives together, before any report caps it, so no one finding
// holds more than a report could list. Each branch of the choices here
// fails eight properties, so each lists seventeen findings unbounded; the
// count of findings held is the measure, not the time or memory spent.
func TestNestedFindingsBounded(t *testing.T) {
	testenv.Home(t)
	var names, members []string
	for c := 'a'; c <= 'h'; c++ {
		names = append(names, fmt.Sprintf("%q", string(c)))
		members = append(members, fmt.Sprintf("%q: 0", string(c)))
	}
	branch := func(typ string) string {
		var props []string
		for _, n := range names {
			props = append(props, fmt.Sprintf("%s: {\"type\": %q}", n, typ))
		}
		return `{"properties": {` + strings.Join(props, ", ") + `}}`
	}
	for _, keyword := range []string{"anyOf", "oneOf"} {
		t.Run(keyword, func(t *testing.T) {
			l := layer(t, pkg{
				inputs: []map[string]any{slot("doc", "inputs/doc.json", "required", true)},
				schemas: map[string]string{
					"inputs/doc.json": `{"` + keyword + `": [` + branch("string") + `, ` + branch("boolean") + `]}`,
				},
			})
			doc := "--inputs-json={\"doc\": {" + strings.Join(members, ", ") + "}}"
			unbounded := run(t, l, limits.Defaults(), args{flags: []string{doc}})
			if len(unbounded.Findings) != 1 || listedWithin(unbounded.Findings[0]) != 17 {
				t.Fatalf("at the default findings limit: %d findings, want one listing 17:\n%s",
					len(unbounded.Findings), describe(unbounded.Findings))
			}
			for _, bound := range []int{1, 2, 5, 16} {
				lim := limits.Defaults()
				if f, ok := lim.Lower(fmt.Sprintf("findings=%d", bound)); !ok {
					t.Fatalf("findings=%d: %s", bound, f.Code())
				}
				res := run(t, l, lim, args{flags: []string{doc}})
				if len(res.Findings) != 1 {
					t.Fatalf("findings=%d: %d findings, want one:\n%s", bound, len(res.Findings), describe(res.Findings))
				}
				if got := listedWithin(res.Findings[0]); got != bound {
					t.Errorf("findings=%d: the %s finding holds %d findings, want %d", bound, keyword, got, bound)
				}
			}
		})
	}
}
