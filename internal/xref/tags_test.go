// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package xref

import (
	"testing"

	"github.com/kottera/metais/internal/manifest"
)

// TestMetaisTags holds embed, asset and provenance to ADR-007 D3 item 12:
// embed names an input by literal, asset an asset path, and provenance
// takes keywords only.
func TestMetaisTags(t *testing.T) {
	fx := fixture{
		inputs:    []fxInput{in("vendor", `{"type": "object"}`), in("notes", `{"type": "array"}`)},
		templates: map[string]string{"main.liquid": `{% embed "vendor" %}{% embed "vendr" %}{% embed "notes", index: inputs.notes.size %}{% asset "site.css" %}{% asset "logo.svg", as: "path" %}{% provenance package version inputs style: "html" %}`},
		assets:    []string{"site.css"},
		outputs:   []manifest.Output{tpl("main", "main.liquid")},
	}
	expect(t, fx.run(t),
		`template.xref.missing_asset templates/main.liquid:1:107 asset="logo.svg"`,
		`template.xref.unknown_input templates/main.liquid:1:21 name="vendr" suggestion="vendor"`,
	)
	// embed is a use: with neither embed, both inputs are unused.
	fx.templates["main.liquid"] = `{% embed "nobody" %}{% provenance inputs %}`
	expect(t, fx.run(t),
		`template.xref.unknown_input templates/main.liquid:1:1 name="nobody"`,
		`template.xref.unused_input metais.yaml#/inputs/0`,
		`template.xref.unused_input metais.yaml#/inputs/1`,
	)
}
