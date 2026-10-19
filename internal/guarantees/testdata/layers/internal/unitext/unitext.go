// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package unitext keeps only-importers' exception for golang.org/x/text.
package unitext

import "golang.org/x/text/unicode/norm"

// IsNFC is norm.NFC.IsNormalString.
var IsNFC = norm.NFC.IsNormalString
