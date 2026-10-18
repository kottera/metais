// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import "fmt"

// Role is what a command is to the package it opens (ADR-006 D12 item 2,
// ADR-008 D13).
type Role uint8

const (
	// RoleConsumer is a command whose caller received the package and
	// cannot fix it: inspect, ls, read, outline, grep, schema, extract,
	// validate and render.
	RoleConsumer Role = iota + 1
	// RoleAuthoring is a command whose caller is the package's author:
	// init, check, test, pack, unpack, sample and render --sample.
	RoleAuthoring
	// RoleNoPackage is a command that opens no package: version, help and
	// limits.
	RoleNoPackage
)

// The exit statuses of ADR-006 D12 item 1.
const (
	ExitOK          = 0
	ExitFindings    = 1
	ExitInternal    = 2
	ExitUsage       = 3
	ExitRefused     = 4
	ExitDestination = 5
)

// packageSubject reports whether a finding about k is about the package's
// own content, which fails a consumer command as a refused package
// (ADR-006 D12 item 2).
func packageSubject(k SubjectKind) bool {
	switch k {
	case SubjectPackage, SubjectManifest, SubjectSchema, SubjectTemplate, SubjectAsset, SubjectReference:
		return true
	}
	return false
}

// Exit returns the exit status of a command in role whose findings are
// the report's (ADR-006 D12), counting every distinct finding, listed or
// not, so the cap never changes the status (ADR-009 D2 item 6). failOn is
// SeverityError, or SeverityWarning when a warning fails the command as an
// error does (ADR-006 D2 item 5). When several statuses apply, the first
// of 2, 3, 4, 1 and 5 is returned (D12 item 3). Info never changes the
// status.
func (r *Report) Exit(role Role, failOn Severity) int {
	if role < RoleConsumer || role > RoleNoPackage {
		panic(fmt.Sprintf("finding: Exit with role %d", role))
	}
	if failOn != SeverityError && failOn != SeverityWarning {
		panic(fmt.Sprintf("finding: Exit failing on %s", failOn))
	}
	c := r.sum.classes
	switch {
	case c[ClassInternal]:
		return ExitInternal
	case c[ClassUsage]:
		return ExitUsage
	case c[ClassRefused], r.sum.packageContent && role == RoleConsumer:
		return ExitRefused
	case c[ClassContent], r.sum.packageContent, failOn == SeverityWarning && r.sum.warnings > 0:
		return ExitFindings
	case c[ClassDestination]:
		return ExitDestination
	}
	return ExitOK
}
