// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid_test

import (
	"github.com/kottera/metais/internal/dates"
	"github.com/kottera/metais/internal/liquid"
)

// Golden Liquid's date cases run with the date filter of internal/dates,
// the one the dialect registers. This external test file is linked into
// the package's test binary, so it can import internal/dates, which
// imports internal/liquid, and hand the filter to the internal tests.
func init() { liquid.RegisterGoldenFilter(dates.Filter()) }
