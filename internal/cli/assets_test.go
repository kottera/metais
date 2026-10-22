// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
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

// TestRenderAssetRecords is ADR-005 D7 item 3 and D16 item 2 in render's
// result (ADR-008 D8 item 9): a file copied from an asset is recorded with
// source asset, no kind, its size and the digest of its bytes, beside a
// template file's record; and the human form lists both in path order.
func TestRenderAssetRecords(t *testing.T) {
	root := testenv.Dir(t)
	logo := "\x89PNG\r\n\x1a\n\x00\xff"
	pkg := writeSourcePackage(t, root, "  - id: logo\n    asset: logo.png\n    executable: true\n  - id: page\n    template: page.html.liquid\n",
		map[string]string{"assets/logo.png": logo, "templates/page.html.liquid": `<img src="{% asset "logo.png" %}">`})
	r := runTool(t, "render", pkg, "--out", filepath.Join(root, "out"), "--json")
	if r.status != 0 {
		t.Fatalf("metais render --json: status %d, %s", r.status, r.stdout)
	}
	res := readRender(t, r)
	if len(res.Files) != 2 {
		t.Fatalf("files %+v", res.Files)
	}
	sum := sha256.Sum256([]byte(logo))
	a, p := res.Files[0], res.Files[1]
	if a.Path != "logo.png" || a.Output != "logo" || a.Source != "asset" || a.Kind != nil || a.Size != int64(len(logo)) ||
		a.Digest != "sha256:"+hex.EncodeToString(sum[:]) || !a.Executable || a.Status != "create" {
		t.Errorf("the asset's record %+v", a)
	}
	if p.Path != "page.html" || p.Source != "template" || p.Kind == nil || *p.Kind != "html" {
		t.Errorf("the template's record %+v", p)
	}
	if !strings.Contains(r.stdout, `"kind":null`) {
		t.Errorf("the asset's kind is not null: %s", r.stdout)
	}
	got, err := os.ReadFile(filepath.Join(root, "out", "logo.png"))
	if err != nil || string(got) != logo {
		t.Errorf("logo.png is %q, %v", got, err)
	}

	r = runTool(t, "render", pkg, "--out", filepath.Join(root, "out2"))
	want := "10\t" + a.Digest + "\tlogo.png\n" + "20\t" + p.Digest + "\tpage.html\n"
	if r.status != 0 || r.stdout != want {
		t.Errorf("metais render: status %d, stdout %q, want %q", r.status, r.stdout, want)
	}
}
