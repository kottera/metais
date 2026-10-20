// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"github.com/kottera/metais/internal/manifest"
	"github.com/kottera/metais/internal/specdata"
)

// Readme is the README text of the specification version spec (ADR-001
// A1 item 3): the bytes every packed package declaring it begins with, as
// README.md. Specification 1.0 defines the first text; a minor that
// defines none has the text of the latest minor before it that does.
func Readme(spec manifest.Spec) []byte {
	return specdata.Readme10
}
