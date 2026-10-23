// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import "github.com/yuin/goldmark"

// Markdown is the site's Markdown library outside the site's converter,
// which breaks only-importers.
var Markdown = goldmark.New
