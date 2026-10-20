// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
)

// TestSizeNamesTheLimit is ADR-009 D3's division of the limits for one
// text: an input's is bounded by input-size, an output's by output-size,
// and the package's own content by the limit Source.Size names, which a
// lowered data limit does not reach.
func TestSizeNamesTheLimit(t *testing.T) {
	const src = `{"type": "object"}`
	lim := lowered(t, "input-size=8", "output-size=8", "schema-size=8")
	for _, c := range []struct {
		from Source
		want string
	}{
		{vendor, "input-size"},
		{Source{Input: vendor.Input, At: vendor.At, Output: true}, "output-size"},
		{Source{Input: vendor.Input, At: vendor.At, Size: limits.SchemaSize}, "schema-size"},
		{Source{Input: vendor.Input, At: vendor.At, Size: limits.ManifestSize}, ""},
		{Source{Input: vendor.Input, At: vendor.At, Size: limits.EntrySize}, ""},
	} {
		_, fs := Parse([]byte(src), c.from, lim)
		var got string
		if len(fs) == 1 && fs[0].Code() == finding.CodeLimitExceeded {
			v, _ := fs[0].Param("limit")
			got, _ = v.Str()
		} else if len(fs) > 0 {
			t.Errorf("%+v: findings %v", c.from, fs)
			continue
		}
		if got != c.want {
			t.Errorf("size %v, output %v: refused by %q, want %q", c.from.Size, c.from.Output, got, c.want)
		}
	}
}
