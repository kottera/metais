// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package template

import (
	"bytes"
	"testing"

	"metais.org/metais/internal/inputs"
	"metais.org/metais/internal/liquid"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
)

// renderTagsWith renders text in a renderer whose package declares decls,
// supplied as records, and returns what it wrote or the error that
// stopped it.
func renderTagsWith(t *testing.T, text string, decls []manifest.Input, records []inputs.Record) (string, *liquid.Error) {
	t.Helper()
	r := &Renderer{
		b:   &Bindings{pkg: Package{Manifest: &manifest.Manifest{Name: "p", Version: "1.0.0", Inputs: decls}}, records: records},
		out: Output{ID: "o", Path: "o.txt", Kind: "text"},
	}
	r.dialect = newDialect(r)
	tpl, errs := liquid.Parse(r.dialect, liquid.Source{Name: "templates/t", Text: []byte(text)}, liquid.ParseLimits{})
	if len(errs) > 0 {
		t.Fatalf("%q does not parse: %v", text, errs[0])
	}
	var buf bytes.Buffer
	e := liquid.Render(&buf, tpl, &liquid.Env{Dialect: r.dialect, Globals: obj(), Meter: liquid.NewMeter(liquid.MeterLimits{})})
	return buf.String(), e
}

// TestEmbedNotUTF8 is ADR-004 D12 item 2's refusal, which no supply can
// reach in 1.0, every form refusing text that is not UTF-8: a raw form
// that is not UTF-8, once its byte order mark is dropped, is
// template.invalid_utf8 naming the tag, and nothing of it is written.
func TestEmbedNotUTF8(t *testing.T) {
	decls := []manifest.Input{{Name: "x"}}
	records := []inputs.Record{{Name: "x", Value: str("ignored"), Raw: "\xEF\xBB\xBFok \xFF", HasRaw: true}}
	out, e := renderTagsWith(t, `a{% embed "x" %}b`, decls, records)
	if e == nil || e.Code != liquid.CodeInvalidUTF8 || e.Args["tag"] != "embed" {
		t.Fatalf("embed of bytes that are not UTF-8 wrote %q and gave %v, want invalid_utf8 naming embed", out, e)
	}
	r := &Renderer{}
	f := r.renderFinding(Output{ID: "o", Path: "o.txt"}, e)
	if f.Code() != CodeInvalidUTF8 {
		t.Fatalf("the finding is %s, want %s", f.Code(), CodeInvalidUTF8)
	}
	v, _ := f.Param("tag")
	if s, ok := v.Str(); !ok || s != "embed" {
		t.Errorf("the finding's tag is %v, want embed", v)
	}
	if _, ok := f.Param("filter"); ok {
		t.Error("the finding names a filter")
	}

	// The control: the same bytes made UTF-8 are written, without their
	// byte order mark.
	records[0].Raw = "\xEF\xBB\xBFok \xC3\xBF"
	if out, e := renderTagsWith(t, `a{% embed "x" %}b`, decls, records); e != nil || out != "aok ÿb" {
		t.Errorf("embed wrote %q and gave %v, want %q", out, e, "aok ÿb")
	}
}

// TestEncodeValue is ADR-004 D14 item 3: every byte outside the set is
// %XX in upper-case hex, and in the html style a - after a - is %2D, so
// that no value holds --.
func TestEncodeValue(t *testing.T) {
	for _, tc := range []struct {
		in         string
		html, want string
	}{
		{"az-AZ_09.+:/@", "az-AZ_09.+:/@", "az-AZ_09.+:/@"},
		{"a b=c\"é", "a%20b%3Dc%22%C3%A9", "a%20b%3Dc%22%C3%A9"},
		{"--x---", "-%2Dx-%2D%2D", "--x---"},
		{"-x-", "-x-", "-x-"},
		{"\x00\xff", "%00%FF", "%00%FF"},
	} {
		if got := encodeValue(tc.in, true); got != tc.html {
			t.Errorf("encodeValue(%q, html) = %q, want %q", tc.in, got, tc.html)
		}
		if got := encodeValue(tc.in, false); got != tc.want {
			t.Errorf("encodeValue(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// TestProvenanceWritesNothingOfTheHost pins ADR-004 D14 item 2's closed
// fields false first: whatever parts are named, and in every style, a
// line holds only the fields the parts name, and the time only when --now
// gave one. Built with the pid canary, which adds the process id to
// every line, it fails.
func TestProvenanceWritesNothingOfTheHost(t *testing.T) {
	decls := []manifest.Input{{Name: "x"}}
	records := []inputs.Record{{Name: "x", Value: value.Null{}}}
	for style, want := range map[string]string{
		"hash": "# metais-provenance package=p version=1.0.0 package_digest=sha256:0000000000000000000000000000000000000000000000000000000000000000 input.x=absent tool=",
		"json": `{"inputs":{"x":null},"package":"p","package_digest":"sha256:0000000000000000000000000000000000000000000000000000000000000000","tool":"","version":"1.0.0"}`,
	} {
		out, e := renderTagsWith(t, `{% provenance all style: "`+style+`" %}`, decls, records)
		if e != nil || out != want {
			t.Errorf("provenance all in %s wrote %q and gave %v, want %q", style, out, e, want)
		}
	}
}
