// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package vectorgen produces the conformance suite's committed packages:
// every package.metais under spec/conformance/vectors/ (ADR-011 D9 item 7,
// D16 item 5). Its tests are the program. With -update they write each
// package's bytes into its vector; without it they rebuild every package
// and fail on any difference from the committed bytes, and on a committed
// package no case produces:
//
//	go test ./internal/vectorgen -run TestPackages -args -update
//
// A package is built from source files the way pack builds one: its
// compiled files are derived by internal/loader, the code every open
// re-derives them with, its digest file by internal/digest, and its zip
// is written field by field, so that each hostile package can break one
// rule of the profile and keep a correct .compiled/ and correct digests
// around its single defect (ADR-011 D16 item 5). Deflated entries are
// written with compress/flate at level 9, whose bytes the module's pinned
// toolchain fixes (ADR-001 D4 item 6); most entries are stored, so most
// packages do not depend on it.
package vectorgen
