// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package ooxml breaks never-depends once, through loader.
package ooxml

import "metais.org/metais/internal/loader"

// Scan reaches the manifest through the loader.
func Scan() { _ = loader.Of }
