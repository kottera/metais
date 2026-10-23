// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquid

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
)

// TestStripHTMLIsTheExpressions holds strip_html's scanner to Shopify's
// two regular expressions, applied in turn. The texts are built from the
// pieces the expressions turn on, so openings without closes, closes
// before openings, blocks inside tags, tags made by removing a block, and
// unclosed tags all occur, beside the cases written out.
func TestStripHTMLIsTheExpressions(t *testing.T) {
	blocks := regexp.MustCompile(`(?s)<script.*?</script>|<!--.*?-->|<style.*?</style>`)
	tags := regexp.MustCompile(`(?s)<.*?>`)
	want := func(s string) string {
		return tags.ReplaceAllLiteralString(blocks.ReplaceAllLiteralString(s, ""), "")
	}
	texts := []string{
		"", "plain", "<", ">", "<>", "a<b>c", "<p>x</p>", "<a", "a>b<c",
		"<script>x</script>y", "<script</script>", "<scrip></script>", "<script>",
		"<!---->", "<!-->", "<!--->", "<!-- a --> b <!-- c",
		"<style>s</style><script>t</script>", "<<!-- x -->a>", "<sc<!---->ript>x</script>",
		"<script>a</script></script>", "<!--<script>-->x</script>", "<style><!--</style>-->",
		"é<b>ü</b>\n<script>\n</script>",
	}
	pieces := []string{"<", ">", "<script", "</script>", "<!--", "-->", "<style", "</style>", "x", "-", "!", "/", "s", "\n", "é"}
	r := rand.New(rand.NewPCG(3, 4))
	for range 20_000 {
		var b strings.Builder
		for range r.IntN(12) {
			b.WriteString(pieces[r.IntN(len(pieces))])
		}
		texts = append(texts, b.String())
	}
	for _, s := range texts {
		if got := stripTags(stripBlocks(s)); got != want(s) {
			t.Fatalf("strip_html of %q is %q, not %q", s, got, want(s))
		}
	}
}
