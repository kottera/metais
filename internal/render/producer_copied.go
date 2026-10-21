// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"errors"
	"io"
)

// digestAssets reads every asset of files whose digest is not yet taken,
// for its digest (ADR-005 D16 item 2). An asset's digest is taken as it is
// first read; a dry run, the archive and --only read it after the records
// are made, or never.
func digestAssets(files []*File) {
	for _, pf := range files {
		if pf.asset != "" && pf.digest == "" {
			r, done := pf.Open()
			_, err := io.Copy(io.Discard, r)
			if derr := done(err); err != nil || derr != nil {
				panic("render: reading an asset the plan found: " + errors.Join(err, derr).Error())
			}
		}
	}
}
