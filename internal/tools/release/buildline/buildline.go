// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package buildline is ADR-010 D12 items 1 and 2's static build of
// ./cmd/metais: its go build arguments and the environment it builds
// under. The release builder builds with it, and the SC7 harness builds
// the release binary it judges with it, so the binary the harness clears
// is the one a release ships.
package buildline

// Package is the package a release builds.
const Package = "./cmd/metais"

// versionVar is the variable a release build sets to its version
// (ADR-010 D12 item 3).
const versionVar = "metais.org/metais/internal/version.release"

// Target is one GOOS/GOARCH pair.
type Target struct {
	OS, Arch string
}

func (t Target) String() string { return t.OS + "/" + t.Arch }

// Shipped are the targets a release ships, in the order of ADR-010 D12
// item 2.
var Shipped = []Target{
	{"darwin", "amd64"},
	{"darwin", "arm64"},
	{"linux", "amd64"},
	{"linux", "arm64"},
}

// Settings are the variables every build sets explicitly, whatever the
// caller's environment holds: no cgo, and the baseline microarchitecture of
// each shipped architecture (ADR-010 D12 items 1 and 2).
var Settings = []string{"CGO_ENABLED=0", "GOAMD64=v1", "GOARM64=v8.0"}

// Args are go's arguments that build Package to out as a release of
// version, a semantic version without a leading v.
func Args(version, out string) []string {
	return []string{
		"build", "-trimpath", "-buildvcs=true",
		"-ldflags", LDFlags(version),
		"-o", out, Package,
	}
}

// LDFlags is the -ldflags value Args passes, as go version -m reports it.
func LDFlags(version string) string {
	return "-s -w -X " + versionVar + "=" + version
}
