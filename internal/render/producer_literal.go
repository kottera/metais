// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"bytes"
	"io"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/zipfmt"
)

// literal is the producer of a profile's own mimetype entry (ADR-005 D9
// items 2 to 4), whose bytes are the profile's media type.
type literal struct{}

// Record is a profile's: its mimetype has no kind (D7 item 3).
func (literal) Record() (string, bool) { return "profile", false }

// Reach is none: the mimetype renders no template.
func (literal) Reach(*File) []string { return nil }

// Produce is the profile's media type, measured.
func (literal) Produce(_ *Producing, f *File) []finding.Finding {
	m, _ := f.decl.Container.ToolMimetype()
	f.bytes = []byte(m)
	f.size, f.digest = measureBytes(f.bytes)
	return nil
}

func (literal) Open(f *File) (io.Reader, func(error) error) {
	return bytes.NewReader(f.bytes), func(error) error { return nil }
}

// Method is stored, always (D8 item 7).
func (literal) Method(*File) zipfmt.Method { return zipfmt.Store }
