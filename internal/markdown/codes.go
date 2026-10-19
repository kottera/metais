// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package markdown

import "github.com/kottera/metais/internal/finding"

// The codes SplitFrontmatter reports (ADR-003 Interfaces, ADR-009 D19).
// Each is class content, about an input.
const (
	CodeFrontmatterUnclosed finding.Code = "markdown.frontmatter_unclosed"
	CodeLimitExceeded       finding.Code = "limit.exceeded"
)
