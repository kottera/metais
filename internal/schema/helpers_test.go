// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/jsonx"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/liquid/value"
)

// schemaFile is the source a test's schema text is read as.
var schemaFile = jsonx.Source{
	Input: finding.Subject{Kind: finding.SubjectSchema, Name: "inputs/vendor.schema.json"},
	At:    finding.Location{Origin: finding.OriginPackage, File: "inputs/vendor.schema.json"},
}

// parse returns the value of the JSON text src, failing t on any finding.
func parse(t *testing.T, src string) value.Value {
	t.Helper()
	doc, fs := jsonx.Parse([]byte(src), schemaFile, limits.Defaults())
	if len(fs) > 0 {
		t.Fatalf("parse %s: %v", src, fs)
	}
	return doc.Value
}

// canonical returns v's canonical JSON text.
func canonical(v value.Value) string { return string(jsonx.Canonical(v)) }

// ptr returns a pointer to s.
func ptr(s string) *string { return &s }
