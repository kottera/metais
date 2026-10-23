// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"html"
	"maps"
	"net/url"
	"path"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// linkAttr matches every attribute of a page that names another resource:
// an href or a src, quoted as the site's templates and the converted
// Markdown quote them.
var linkAttr = regexp.MustCompile(`\s(?:href|src)="([^"]*)"`)

// refreshURL matches the target of a meta refresh.
var refreshURL = regexp.MustCompile(`<meta http-equiv="refresh" content="[0-9]+; url=([^"]*)">`)

// idAttr matches every id an element of a page carries, which a link's
// fragment names.
var idAttr = regexp.MustCompile(`\sid="([^"]*)"`)

// links is every link of page, a slash-separated path in the published
// tree, that is internal to the site, each resolved to the path of the
// file it names in that tree and its fragment, or "" for none. A link
// with a scheme is internal when it names the site's origin; any other is
// not the site's to hold.
func links(page string, b []byte) (out []link, bad []string) {
	base, _ := url.Parse(origin + page)
	var raws []string
	for _, m := range linkAttr.FindAllSubmatch(b, -1) {
		raws = append(raws, html.UnescapeString(string(m[1])))
	}
	for _, m := range refreshURL.FindAllSubmatch(b, -1) {
		raws = append(raws, html.UnescapeString(string(m[1])))
	}
	for _, raw := range raws {
		u, err := url.Parse(raw)
		if err != nil || raw == "" {
			bad = append(bad, raw)
			continue
		}
		u = base.ResolveReference(u)
		if u.Scheme+"://"+u.Host+"/" != origin {
			continue
		}
		p := strings.TrimPrefix(u.Path, "/")
		if p == "" || strings.HasSuffix(p, "/") {
			p += "index.html"
		}
		if path.Clean("/"+p) != "/"+p {
			bad = append(bad, raw)
			continue
		}
		out = append(out, link{raw: raw, file: p, fragment: u.Fragment})
	}
	return out, bad
}

// link is one internal link: as the page writes it, the file it names in
// the published tree, and its fragment.
type link struct {
	raw, file, fragment string
}

// checkLinks requires every internal link of every page of site, a
// published tree by path, to name a file the tree holds, and every
// fragment to name an id that page carries.
func checkLinks(t errorer, where string, site map[string][]byte) {
	t.Helper()
	ids := map[string][]string{}
	n := 0
	for _, p := range slices.Sorted(maps.Keys(site)) {
		if !strings.HasSuffix(p, ".html") {
			continue
		}
		ls, bad := links(p, site[p])
		for _, raw := range bad {
			t.Errorf("%s: %s links to %q, which is not a path the site serves", where, p, raw)
		}
		for _, l := range ls {
			n++
			target, ok := site[l.file]
			if !ok {
				t.Errorf("%s: %s links to %q, and the site has no %s", where, p, l.raw, l.file)
				continue
			}
			if l.fragment == "" || !strings.HasSuffix(l.file, ".html") {
				continue
			}
			if _, ok := ids[l.file]; !ok {
				ids[l.file] = []string{}
				for _, m := range idAttr.FindAllSubmatch(target, -1) {
					ids[l.file] = append(ids[l.file], html.UnescapeString(string(m[1])))
				}
			}
			if !slices.Contains(ids[l.file], l.fragment) {
				t.Errorf("%s: %s links to %q, and %s has no id %q", where, p, l.raw, l.file, l.fragment)
			}
		}
	}
	if n == 0 {
		t.Errorf("%s: no page links to another; the check is blind", where)
	}
}

// errorer is the part of a testing.T that checkLinks reports through.
type errorer interface {
	Helper()
	Errorf(format string, args ...any)
}

// recorder counts what checkLinks reports, so a test can require a
// refusal without failing.
type recorder struct{ errors int }

func (r *recorder) Helper()               {}
func (r *recorder) Errorf(string, ...any) { r.errors++ }

// TestInternalLinks requires every internal link of every page the site
// builds to go to a page or a file the site serves, and every fragment to
// an id of that page (ADR-013 D8 item 4). It walks the whole built tree,
// so a page a later change adds is held by it as it lands.
func TestInternalLinks(t *testing.T) {
	checkLinks(t, "the built site", tree(t, builtSite(t)))
}

// TestLinks pins the link check false first: a link to a page the site
// does not have, to a fragment its page lacks, or out of the tree is
// refused, and a link elsewhere, or relative, is resolved as a browser
// resolves it.
func TestLinks(t *testing.T) {
	good := map[string][]byte{
		"index.html":     []byte(`<a href="/a/">a</a><a href="a/#x">x</a><a href="https://metais.org/f.json">f</a><a href="https://example.org/nowhere">e</a><a href="#top">t</a><p id="top"></p>`),
		"a/index.html":   []byte(`<h2 id="x">x</h2><a href="../">home</a><a href="b.css">b</a><meta http-equiv="refresh" content="0; url=/">`),
		"a/b.css":        []byte(``),
		"f.json":         []byte(`{}`),
		"not-a-page.txt": []byte(`<a href="/missing/">not read</a>`),
	}
	ls, bad := links("a/index.html", good["a/index.html"])
	var files []string
	for _, l := range ls {
		files = append(files, l.file+"#"+l.fragment)
	}
	if want := []string{"index.html#", "a/b.css#", "index.html#"}; len(bad) != 0 || !slices.Equal(files, want) {
		t.Errorf("a/index.html's links resolve to %q, %q; want %q", files, bad, want)
	}
	checkLinks(t, "a site whose links all resolve", good)

	for name, page := range map[string]string{
		"a missing page":     `<a href="/b/">b</a>`,
		"a missing file":     `<img src="/a/c.png">`,
		"a missing fragment": `<a href="/a/#y">y</a>`,
		"a missing id here":  `<a href="#nowhere">n</a>`,
		"a missing redirect": `<meta http-equiv="refresh" content="0; url=/spec/1.0/">`,
		"the origin's page":  `<a href="https://metais.org/b/">b</a>`,
		"a path out":         `<a href="/a/../../x">x</a>`,
	} {
		site := map[string][]byte{"a/index.html": []byte(`<h2 id="x"></h2>`), "index.html": []byte(page)}
		var rec recorder
		checkLinks(&rec, name, site)
		if rec.errors == 0 {
			t.Errorf("a page with %s passes the link check", name)
		}
	}
}
