// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package render

import (
	"os"
	"path/filepath"
	"testing"
)

// writeSourcePackage writes a source directory below root, its manifest's
// outputs section being outputs and its files those of files, and returns
// its path.
func writeSourcePackage(t *testing.T, root, outputs string, files map[string]string) string {
	t.Helper()
	pkg := filepath.Join(root, "pkg")
	files["metais.yaml"] = "metais: \"1.0\"\nname: example\nversion: 1.0.0\ntitle: Example\n" +
		"description: An example package.\nguide: Render it.\ninputs: []\noutputs:\n" + outputs
	for name, text := range files {
		p := filepath.Join(pkg, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return pkg
}
