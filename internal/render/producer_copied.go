// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/template"
	"metais.org/metais/internal/zipfmt"
)

// copied is the producer of an asset's file (ADR-005 D5): its bytes are
// the package entry's, never held, and streamed from the package as they
// are written (D13 item 4).
type copied struct{}

// Record is an asset's: it has no kind (D7 item 3).
func (copied) Record() (string, bool) { return "asset", false }

// Reach is none: an asset renders no template.
func (copied) Reach(*File) []string { return nil }

// Produce charges the entry's size, which the plan took, against the
// output limits, without reading it.
func (copied) Produce(c *Producing, f *File) []finding.Finding {
	o := template.Output{ID: f.owner, Path: f.path, In: f.in}
	if fd := c.r.ChargeAsset(o, f.size); fd != nil {
		f.failed = true
		return []finding.Finding{*fd}
	}
	return nil
}

// Open streams the entry from the package, its digest taken on the first
// whole read.
func (copied) Open(f *File) (io.Reader, func(error) error) {
	rc, err := f.tree.Open(f.asset)
	if err != nil {
		panic("render: opening " + f.asset + ", which the plan found: " + err.Error())
	}
	h := sha256.New()
	return io.TeeReader(rc, h), func(read error) error {
		rc.Close()
		if read == nil && f.digest == "" {
			f.digest = "sha256:" + hex.EncodeToString(h.Sum(nil))
		}
		return nil
	}
}

func (copied) Method(f *File) zipfmt.Method { return declaredMethod(f) }

// placement is where the asset tag places f's entry, by its name beneath
// assets/ (ADR-005 D5 item 6), when f is copied from one.
func placement(f *File) (string, template.Placement, bool) {
	if _, ok := f.producer.(copied); !ok {
		return "", template.Placement{}, false
	}
	return strings.TrimPrefix(f.asset, template.AssetDir), template.Placement{Tree: f.in, Path: f.path}, true
}

// digestAssets reads every file of files whose digest is not yet taken,
// for its digest (ADR-005 D16 item 2). Every producer but copied measures
// its file in phase 4; an asset's digest is taken as it is first read,
// and a dry run, the archive and --only read it after the records are
// made, or never.
func digestAssets(files []*File) {
	for _, pf := range files {
		if pf.digest == "" {
			r, done := pf.Open()
			_, err := io.Copy(io.Discard, r)
			if derr := done(err); err != nil || derr != nil {
				panic("render: reading an asset the plan found: " + errors.Join(err, derr).Error())
			}
		}
	}
}
