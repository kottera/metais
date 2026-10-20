// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package specdata holds byte-for-byte copies of the spec files the binary
// embeds (ADR-010 D2 item 4). The canonical files live under spec/, which
// //go:embed cannot reach from here; go generate ./internal/specdata
// refreshes the copies, and the gate fails if that changes anything.
//
// Each copy keeps its path below spec/: spec/findings/registry.json is
// copied to findings/registry.json in this directory. The files copied are
// those named by specFiles in generate_test.go.
//
// The copier is a test, run by go generate with -specdata.generate, so this
// package's non-test files hold no file access.
package specdata

import _ "embed"

//go:generate go test -count=1 -run=^TestGenerate$ -args -specdata.generate

// FindingRegistry is spec/findings/registry.json, the finding-code
// registry (ADR-006 D3 item 3).
//
//go:embed findings/registry.json
var FindingRegistry []byte

// FindingRegistrySchema is spec/schemas/1.0/finding-registry.schema.json,
// the schema the registry validates against.
//
//go:embed schemas/1.0/finding-registry.schema.json
var FindingRegistrySchema []byte

// FindingSchema is spec/schemas/1.0/finding.schema.json, the finding
// object's schema (ADR-006 D10 item 5).
//
//go:embed schemas/1.0/finding.schema.json
var FindingSchema []byte

// ManifestSchema10 is spec/schemas/1.0/manifest.schema.json, the schema
// of a manifest that declares METAIS 1.0 (ADR-002 D10).
//
//go:embed schemas/1.0/manifest.schema.json
var ManifestSchema10 []byte

// Readme10 is spec/readme/1.0/README.md, the README.md every packed
// package declaring specification 1.0 begins with (ADR-001 A1 item 3).
// Chapter 2 carries the same bytes, which TestReadmeMatchesChapter holds.
//
//go:embed readme/1.0/README.md
var Readme10 []byte

// files are the embedded copies by their path below spec/, which is the
// list generate_test.go's specFiles names.
var files = map[string][]byte{
	"findings/registry.json":                   FindingRegistry,
	"schemas/1.0/finding-registry.schema.json": FindingRegistrySchema,
	"schemas/1.0/finding.schema.json":          FindingSchema,
	"schemas/1.0/manifest.schema.json":         ManifestSchema10,
	"readme/1.0/README.md":                     Readme10,
}
