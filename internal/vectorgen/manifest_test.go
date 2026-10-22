// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/loader"
	"metais.org/metais/internal/testenv"
)

// This file is the program of the hostile corpus's manifests too large to
// write by hand (ADR-009 D17 item 5): each is a source directory whose
// metais.yaml is a valid manifest with one defect appended. Their bytes
// depend on nothing but this file, so no merge can leave them stale.

// hostileManifest is the valid manifest each case appends its defect to,
// the manifest of the corpus's hand-written cases.
const hostileManifest = `metais: "1.0"
name: hostile
version: 1.0.0
title: Hostile
description: A package that breaks one rule.
guide: Supply a report.
inputs:
  - name: report
    title: Report
    schema: inputs/report.schema.json
outputs:
  - id: report
    template: report.md.liquid
`

// manifestSources returns each case's source directory, by vector.
func manifestSources() map[string]map[string][]byte {
	source := func(defect string) map[string][]byte {
		return map[string][]byte{
			"metais.yaml":                []byte(hostileManifest + defect),
			"inputs/report.schema.json":  []byte(`{"type": "object", "required": ["title"], "properties": {"title": {"type": "string"}}}` + "\n"),
			"templates/report.md.liquid": []byte("# {{ inputs.report.title }}\n"),
		}
	}
	return map[string]map[string][]byte{
		// 10,000 arrays nested in one flow sequence: 9,936 deeper than
		// value-depth allows.
		"sec/manifest-deep-nesting": source("deep: " + strings.Repeat("[", 10000) + strings.Repeat("]", 10000) + "\n"),
		// An integer of a million digits, in a manifest under
		// manifest-size.
		"sec/manifest-million-digit-number": source("n: 1" + strings.Repeat("0", 999_999) + "\n"),
	}
}

// TestManifestSources is the program and its reproduction test (ADR-011
// D16 item 5): every case's source directory is built and compared with
// the committed files, or, with -update, written; a committed file no
// case builds fails it too.
func TestManifestSources(t *testing.T) {
	testenv.Home(t)
	for _, vector := range slices.Sorted(maps.Keys(manifestSources())) {
		files := manifestSources()[vector]
		dir := filepath.Join(vectorsDir, filepath.FromSlash(vector), "package")
		if *update {
			if err := os.RemoveAll(dir); err != nil {
				t.Fatal(err)
			}
		}
		for _, name := range slices.Sorted(maps.Keys(files)) {
			path := filepath.Join(dir, filepath.FromSlash(name))
			if *update {
				if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(path, files[name], 0o644); err != nil {
					t.Fatal(err)
				}
				continue
			}
			got, err := os.ReadFile(path)
			if err != nil {
				t.Errorf("%s: %v (run go test ./internal/vectorgen -run TestManifestSources -args -update)", vector, err)
				continue
			}
			if !bytes.Equal(got, files[name]) {
				t.Errorf("%s: the committed %s is not what its case builds", vector, name)
			}
		}
		err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, _ := filepath.Rel(dir, p)
			if _, ok := files[filepath.ToSlash(rel)]; !ok {
				t.Errorf("%s: a committed %s that no case builds", vector, filepath.ToSlash(rel))
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}

// TestManifestSourcesHoldOneDefect proves each case has one defect
// (R2-178): its sources without the appended line open, and with it they
// are refused with one limit.package finding.
func TestManifestSourcesHoldOneDefect(t *testing.T) {
	opt := loader.Options{Limits: limits.Defaults()}
	for _, vector := range slices.Sorted(maps.Keys(manifestSources())) {
		files := manifestSources()[vector]
		if !bytes.HasPrefix(files["metais.yaml"], []byte(hostileManifest)) {
			t.Fatalf("%s: the manifest does not begin with the valid manifest", vector)
		}
		valid := maps.Clone(files)
		valid["metais.yaml"] = []byte(hostileManifest)
		if p, fs := loader.OpenFiles(valid, opt); p == nil {
			t.Errorf("%s: without its defect the package is refused: %v", vector, fs)
		}
		p, fs := loader.OpenFiles(files, opt)
		if p != nil || len(fs) != 1 || fs[0].Code() != "limit.package" {
			t.Errorf("%s: want one limit.package finding, got %v", vector, fs)
		}
	}
}
