// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/testenv"
)

// FuzzYAMLProfile is ADR-010 D14 item 6's target: whatever the bytes,
// Parse does not panic, gives a document or findings and never both, and
// every position it gives is within the input. A document the manifest or
// an input accepts is written by Write and read back under the same
// profile as the same value. Its seed corpus, under testdata/fuzz, runs
// in every go test.
func FuzzYAMLProfile(f *testing.F) {
	testenv.Home(f)
	frontmatter := inputSrc
	frontmatter.Size, frontmatter.FirstLine = limits.FrontmatterSize, 3
	sets := []limits.Set{limits.Defaults(), lowered(f, "value-depth=3", "number-length=8")}
	f.Fuzz(func(t *testing.T, data []byte) {
		for _, src := range []Source{manifestSrc, inputSrc, frontmatter} {
			for _, lim := range sets {
				doc, fs := parseLim(t, src, string(data), lim)
				if doc.Value == nil {
					continue
				}
				back := inputSrc
				if src.Subject.Kind == manifestSrc.Subject.Kind {
					back = manifestSrc
				}
				text := Write(doc.Value)
				again, fs := Parse(text, back, limits.Defaults())
				if fs != nil || !value.Equal(again.Value, doc.Value) {
					t.Fatalf("%q was written as %q, which reads back as %v: %s", data, text, again, describe(fs))
				}
			}
		}
	})
}
