// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package manifest

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/liquid/value"
	"github.com/kottera/metais/internal/version"
)

// TestGate is ADR-002 D7 items 1, 7, 8 and 11: each refusal of the version
// gate, its code, its class and the implemented list it carries (D8 item
// 2), and the versions it accepts.
func TestGate(t *testing.T) {
	isolate(t)
	cases := []struct {
		name    string
		metais  any
		code    finding.Code
		params  map[string]string // the string params, beside implemented
		ptr     string
		declare Spec
	}{
		{name: "missing", metais: del, code: CodeVersionMissing, ptr: ""},
		{name: "number 1.0", metais: json.RawMessage("1.0"), code: CodeVersionNotString,
			params: map[string]string{"actual": "number", "written": "1.0"}, ptr: "/metais"},
		{name: "number 1.10 quoted as written", metais: json.RawMessage("1.10"), code: CodeVersionNotString,
			params: map[string]string{"actual": "number", "written": "1.10"}, ptr: "/metais"},
		{name: "boolean", metais: true, code: CodeVersionNotString, params: map[string]string{"actual": "boolean"}, ptr: "/metais"},
		{name: "null", metais: nil, code: CodeVersionNotString, params: map[string]string{"actual": "null"}, ptr: "/metais"},
		{name: "array", metais: []any{"1.0"}, code: CodeVersionNotString, params: map[string]string{"actual": "array"}, ptr: "/metais"},
		{name: "1.00", metais: "1.00", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "1", metais: "1", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "01.0", metais: "01.0", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "1.0.0", metais: "1.0.0", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "1.0-rc", metais: "1.0-rc", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "v1.0", metais: "v1.0", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "five-digit major", metais: "10000.0", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "empty", metais: "", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "spaces", metais: " 1.0", code: CodeVersionMalformed, ptr: "/metais"},
		{name: "2.0", metais: "2.0", code: CodeVersionMajor, params: map[string]string{"declared": "2.0"}, ptr: "/metais"},
		{name: "0.9", metais: "0.9", code: CodeVersionMajor, params: map[string]string{"declared": "0.9"}, ptr: "/metais"},
		{name: "9999.9999", metais: "9999.9999", code: CodeVersionMajor, params: map[string]string{"declared": "9999.9999"}, ptr: "/metais"},
		{name: "1.1", metais: "1.1", code: CodeVersionMinor, params: map[string]string{"declared": "1.1"}, ptr: "/metais"},
		{name: "1.99", metais: "1.99", code: CodeVersionMinor, params: map[string]string{"declared": "1.99"}, ptr: "/metais"},
		{name: "1.0", metais: "1.0", declare: Spec{1, 0}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			m := base()
			set(t, m, "", "metais", c.metais)
			text := encode(t, m)
			root, pos := parse(t, text)
			spec, fs := Gate(root, Source{File: ManifestFile, Positions: pos})
			if c.code == "" {
				if len(fs) != 0 || spec != c.declare {
					t.Fatalf("Gate = %v, %v; want %v and no finding", spec, codes(fs), c.declare)
				}
				return
			}
			if len(fs) != 1 || fs[0].Code() != c.code {
				t.Fatalf("Gate reported %v, want [%s]", codes(fs), c.code)
			}
			f := fs[0]
			if f.Class() != finding.ClassRefused || f.Subject().Kind != finding.SubjectPackage {
				t.Errorf("class %s, subject %s; want refused, package", f.Class(), f.Subject().Kind)
			}
			loc, _ := f.Location()
			if loc.Pointer != c.ptr || loc.File != ManifestFile || loc.Origin != finding.OriginPackage || loc.Line != 1 {
				t.Errorf("location %+v, want %q in %s on line 1", loc, c.ptr, ManifestFile)
			}
			// The implemented list is an array, which the hint renders
			// element by element (D8 item 2).
			impl, ok := f.Param("implemented")
			if want := strings.Join(version.Implements(), ", "); !ok || impl.Type() != finding.TypeArray || !strings.Contains(f.Hint()+f.Message(), want) {
				t.Errorf("implemented = %v, want the list %s; message %q, hint %q", impl, want, f.Message(), f.Hint())
			}
			for k, want := range c.params {
				if v, _ := f.Param(k); func() bool { s, ok := v.Str(); return !ok || s != want }() {
					t.Errorf("%s = %v, want %q", k, v, want)
				}
			}
			if _, ok := f.Param("written"); ok && c.params["written"] == "" {
				t.Error("written is set for a value that is not a number")
			}
		})
	}
}

// TestGateRootNotObject: a root that is not an object has no metais, so
// the gate refuses it before anything judges its shape.
func TestGateRootNotObject(t *testing.T) {
	isolate(t)
	for _, text := range []string{`[]`, `"1.0"`, `1`, `null`} {
		root, _ := parse(t, []byte(text))
		if fs := Check(root, Source{File: ManifestFile}); !slices.Equal(codes(fs), []finding.Code{CodeVersionMissing}) {
			t.Errorf("%s: Check reported %v", text, codes(fs))
		}
		if _, err := Build(root); !errors.Is(err, ErrInvalid) {
			t.Errorf("%s: Build: %v", text, err)
		}
	}
}

// TestGateFirst is ADR-002 D7 item 8 and W2-16.4: a declared minor newer
// than the tool's highest is refused before every other check. The
// manifest breaks a rule of every kind Check and Build judge; with a
// version the tool implements each is reported, and with 1.99 only the
// gate's refusal is.
func TestGateFirst(t *testing.T) {
	isolate(t)
	broken := func(metais string) value.Value {
		m := base()
		set(t, m, "", "metais", metais)
		set(t, m, "", "license", "MIT and Apache-2.0")
		set(t, m, "inputs/1", "name", "product")
		set(t, m, "outputs/1", "id", "report")
		set(t, m, "", "title", " Untrimmed")
		set(t, m, "", "x-extra", true)
		root, _ := parse(t, encode(t, m))
		return root
	}

	// The control: under 1.0 every rule is judged.
	fs := Check(broken("1.0"), Source{File: ManifestFile})
	want := []finding.Code{CodeLicenseExpression, CodeInputDuplicateName, CodeOutputDuplicateID}
	if !slices.Equal(codes(fs), want) {
		t.Fatalf("under 1.0, Check reported %v, want %v", codes(fs), want)
	}
	_, err := Build(broken("1.0"))
	var inv *InvalidError
	if !errors.As(err, &inv) || len(inv.Violations) != 5 {
		t.Fatalf("under 1.0, Build: %v, want five violations", err)
	}

	// The detector: under 1.99 the gate alone speaks.
	fs = Check(broken("1.99"), Source{File: ManifestFile})
	if !slices.Equal(codes(fs), []finding.Code{CodeVersionMinor}) {
		t.Fatalf("under 1.99, Check reported %v, want only %s", codes(fs), CodeVersionMinor)
	}
	_, err = Build(broken("1.99"))
	if !errors.As(err, &inv) || !slices.Equal(inv.Violations, []Violation{{"/metais", string(CodeVersionMinor)}}) {
		t.Fatalf("under 1.99, Build: %v, want only the gate's refusal", err)
	}
}

// TestParseSpec is D7 item 1's grammar.
func TestParseSpec(t *testing.T) {
	for text, want := range map[string]Spec{"1.0": {1, 0}, "0.0": {0, 0}, "12.345": {12, 345}, "9999.9999": {9999, 9999}} {
		if got, ok := ParseSpec(text); !ok || got != want || got.String() != text {
			t.Errorf("ParseSpec(%q) = %v, %v", text, got, ok)
		}
	}
	for _, text := range []string{"", "1", "1.", ".1", "1.0.0", "01.0", "1.01", "+1.0", "1.0 ", "١.٠", "10000.0", "1.10000"} {
		if _, ok := ParseSpec(text); ok {
			t.Errorf("ParseSpec(%q) accepted it", text)
		}
	}
}
