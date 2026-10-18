// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package version states what this tool is: its own version, and the
// specification versions it implements (ADR-002 D8, ADR-010 D12 items 3
// and 7). The two are independent: a tool release may implement a new
// minor of the specification without a new tool major.
package version

import (
	"runtime/debug"
	"strings"
)

// release is set by a release build with
// -ldflags "-X github.com/kottera/metais/internal/version.release=<semver>",
// to the release's tag without its leading v. It is empty in every other
// build.
var release string

// override, when non-empty, is the version an in-process test fixed with
// Override.
var override string

// devel is the version of a build that is neither a release nor a module
// download of a tagged version.
const devel = "0.0.0-devel"

// implements is every specification version this tool implements,
// ascending by major then minor (ADR-002 D7 item 2, D8 item 1).
var implements = []string{"1.0"}

// String returns the tool's version, a semantic version without a leading
// v (ADR-010 D12 item 3):
//
//   - the release variable, when a release build set it;
//   - otherwise the main module's version from the build information, without
//     its leading v, when it is a semantic version with no +dirty suffix, so
//     go install github.com/kottera/metais/cmd/metais@v1.0.0 reports 1.0.0;
//   - otherwise 0.0.0-devel.
//
// No flag or environment variable sets it.
func String() string {
	if override != "" {
		return override
	}
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return resolve(release, "")
	}
	return resolve(release, info.Main.Version)
}

// resolve is String's rule over its two sources: the release variable and
// the main module's build-information version.
func resolve(release, module string) string {
	if release != "" {
		return release
	}
	v, ok := strings.CutPrefix(module, "v")
	if !ok || strings.HasSuffix(v, "+dirty") || !isSemver(v) {
		return devel
	}
	return v
}

// Implements returns the specification versions this tool implements,
// ascending by major then minor, each listed explicitly so a caller tests
// membership without knowing that a minor implies every earlier one
// (ADR-002 D8 item 1). The slice is the caller's own.
func Implements() []string {
	return append([]string(nil), implements...)
}

// Override fixes the version String returns until t ends, for in-process
// tests whose output records the tool's version. Only test files may call
// it, which a tripwire checks; t is a testing.TB.
//
// Override sets process-wide state, so a test that calls it must not call
// t.Parallel.
func Override(t interface {
	Helper()
	Cleanup(func())
}, v string) {
	t.Helper()
	prev := override
	override = v
	t.Cleanup(func() { override = prev })
}

// isSemver reports whether v is a semantic version as Semantic Versioning
// 2.0.0 defines it: MAJOR.MINOR.PATCH, each a decimal without a leading
// zero, then an optional pre-release of dot-separated identifiers and an
// optional build metadata of dot-separated identifiers.
func isSemver(v string) bool {
	v, build, hasBuild := strings.Cut(v, "+")
	if hasBuild && !identifiers(build, false) {
		return false
	}
	core, pre, hasPre := strings.Cut(v, "-")
	if hasPre && !identifiers(pre, true) {
		return false
	}
	parts := strings.Split(core, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if !numeric(p) || (len(p) > 1 && p[0] == '0') {
			return false
		}
	}
	return true
}

// identifiers reports whether s is a non-empty, dot-separated list of
// non-empty identifiers of ASCII letters, digits and hyphens. In a
// pre-release, a numeric identifier has no leading zero.
func identifiers(s string, pre bool) bool {
	for _, id := range strings.Split(s, ".") {
		if id == "" {
			return false
		}
		for i := 0; i < len(id); i++ {
			c := id[i]
			if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c == '-') {
				return false
			}
		}
		if pre && numeric(id) && len(id) > 1 && id[0] == '0' {
			return false
		}
	}
	return true
}

// numeric reports whether s is a non-empty string of ASCII digits.
func numeric(s string) bool {
	if s == "" {
		return false
	}
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
