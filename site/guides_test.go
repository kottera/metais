// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"html"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// invoiceDir is the example package the guide "Writing a package" writes
// (Ed's M5, W2-OOB5): its files are the ones the guide shows.
const invoiceDir = "../examples/invoice"

// codeBlock matches one code block of a built page.
var codeBlock = regexp.MustCompile(`(?s)<pre><code>(.*?)</code></pre>`)

// codeBlocks returns the text of every code block of page, a path in the
// built site, as a reader copies it: its markup removed and its
// references decoded.
func codeBlocks(t *testing.T, page string) []string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(builtSite(t), filepath.FromSlash(page)))
	if err != nil {
		t.Fatal(err)
	}
	tag := regexp.MustCompile(`<[^>]*>`)
	var out []string
	for _, m := range codeBlock.FindAllSubmatch(b, -1) {
		out = append(out, html.UnescapeString(tag.ReplaceAllString(string(m[1]), "")))
	}
	return out
}

// TestWritingGuideInvoice holds the guide "Writing a package" and
// examples/invoice identical: its schema and its template are the files
// the guide shows, byte for byte but the final newline, and its manifest
// holds the guide's inputs and outputs verbatim. The example package's
// own checks and tests are TestExamplePackages', in internal/cli.
func TestWritingGuideInvoice(t *testing.T) {
	blocks := codeBlocks(t, "guide/writing-a-package/index.html")
	for _, c := range []struct {
		file, start string
		whole       bool
	}{
		{file: "inputs/invoice.schema.json", start: "{\n  \"$schema\"", whole: true},
		{file: "templates/invoice.html.liquid", start: "<h1>Invoice ", whole: true},
		{file: "metais.yaml", start: "inputs:\n"},
	} {
		b, err := os.ReadFile(filepath.Join(invoiceDir, filepath.FromSlash(c.file)))
		if err != nil {
			t.Fatal(err)
		}
		var found []string
		for _, block := range blocks {
			if strings.HasPrefix(block, c.start) {
				found = append(found, block)
			}
		}
		if len(found) != 1 {
			t.Errorf("the guide has %d code blocks starting %q, want the one that shows %s", len(found), c.start, c.file)
			continue
		}
		block := strings.TrimSuffix(found[0], "\n") + "\n"
		switch {
		case c.whole && block != string(b):
			t.Errorf("examples/invoice/%s differs from the guide's block:\n--- the guide\n%s--- the example\n%s", c.file, block, b)
		case !c.whole && !strings.Contains(string(b), block):
			t.Errorf("examples/invoice/%s does not hold the guide's block:\n%s", c.file, block)
		}
	}
}

// TestGettingStartedRedirect requires /getting-started/, the page the
// guides replaced, to redirect to "Using a package" (ADR-013 D1 item 5,
// Ed's M7 in W2-OOB5). TestInternalLinks holds the target to a page the
// site serves.
func TestGettingStartedRedirect(t *testing.T) {
	b, err := os.ReadFile(filepath.Join(builtSite(t), "getting-started", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	m := refreshURL.FindAllSubmatch(b, -1)
	if len(m) != 1 || string(m[0][1]) != "/guide/using-a-package/" {
		t.Errorf("getting-started/index.html redirects to %q, want once to /guide/using-a-package/", m)
	}
}

// teraWord matches tera as a word, in any case.
var teraWord = regexp.MustCompile(`(?i)\btera\b`)

// TestTeraOnlyWhoMakes requires the built site to name tera on no page but
// the landing page, and there only in "Who makes METAIS" (ADR-013 D2 item
// 1, the brief's vendor-neutrality line). TestVendorNeutral holds the help
// and the skill the reference pages come from.
func TestTeraOnlyWhoMakes(t *testing.T) {
	site := tree(t, builtSite(t))
	for p, b := range site {
		if !strings.HasSuffix(p, ".html") {
			continue
		}
		text := string(b)
		if p == "index.html" {
			who := strings.Index(text, `<h2 id="who-makes-metais">`)
			if who < 0 || !teraWord.MatchString(text[who:]) {
				t.Errorf("index.html has no \"Who makes METAIS\" naming tera")
				continue
			}
			text = text[:who]
		}
		if loc := teraWord.FindStringIndex(text); loc != nil {
			t.Errorf("%s names tera outside the landing page's \"Who makes METAIS\": %q", p, text[max(0, loc[0]-40):min(len(text), loc[1]+40)])
		}
	}
}
