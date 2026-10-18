// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"strings"
	"testing"
)

func mustPanic(t *testing.T, name string, want string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		r := recover()
		if r == nil {
			t.Errorf("%s: New did not refuse", name)
			return
		}
		if msg, _ := r.(string); !strings.Contains(msg, want) {
			t.Errorf("%s: refused with %q, want it to mention %q", name, r, want)
		}
	}()
	f()
}

// TestNewRefuses is ADR-006 D1 item 4: a finding is built only from a
// registered code and the parameters it declares.
func TestNewRefuses(t *testing.T) {
	withTestCodes(t)
	loc := dataDir("a.csv", 1, 1, "/0")
	ml := func(p Params) func() { return func() { New(codeInputMinLength, input("a"), loc, p) } }
	mustPanic(t, "unregistered code", "not registered", func() { New("input.no_such_code", input("a"), loc, nil) })
	mustPanic(t, "empty code", "not registered", func() { New("", input("a"), loc, nil) })
	mustPanic(t, "undeclared parameter", `"value" is not declared`, ml(Params{"limit": Int(1), "length": Int(0), "value": Int(3)}))
	mustPanic(t, "wrong type", `"limit" is a string, not a number`, ml(Params{"limit": String("1"), "length": Int(0)}))
	mustPanic(t, "missing parameter", `"length" is always present`, ml(Params{"limit": Int(1)}))
	mustPanic(t, "wrong element type", `holds a number`, func() {
		New(codeInputType, input("a"), loc, Params{"expected": List(Int(1)), "actual": String("x")})
	})
	mustPanic(t, "wrong subject", "not a template", func() {
		New(codeInputMinLength, Subject{Kind: SubjectTemplate}, loc, Params{"limit": Int(1), "length": Int(0)})
	})
	mustPanic(t, "a location for a code with none", "span none", func() { New(codeUsageFlag, Subject{Kind: SubjectInvocation}, loc, nil) })
	mustPanic(t, "no location for a code with a span", "span value", func() {
		New(codeInputMinLength, input("a"), nil, Params{"limit": Int(1), "length": Int(0)})
	})
	bad := Location{Origin: OriginDataDir}.WithPointer("a/b")
	mustPanic(t, "a pointer without its slash", "not a JSON Pointer", func() { New(codeInputMinLength, input("a"), &bad, Params{"limit": Int(1), "length": Int(0)}) })
	bad = bad.WithPointer("/a~2")
	mustPanic(t, "a bad escape", "not a JSON Pointer", func() { New(codeInputMinLength, input("a"), &bad, Params{"limit": Int(1), "length": Int(0)}) })
	// ADR-006 D4 items 2 and 3: a name exactly where the kind has one, a
	// title only for an input or a schema.
	mustPanic(t, "an input with no name", "needs a name", func() {
		New(codeInputRequired, Subject{Kind: SubjectInput}, loc, Params{"property": String("x")})
	})
	mustPanic(t, "an output with no name", "needs a name", func() {
		New(codeOutputPath, Subject{Kind: SubjectOutput}, loc, nil)
	})
	mustPanic(t, "a named package", "takes no name", func() {
		New(codePackageName, Subject{Kind: SubjectPackage, Name: "p"}, loc, nil)
	})
	mustPanic(t, "a titled template", "takes no title", func() {
		New(codeTemplateParse, Subject{Kind: SubjectTemplate, Name: "t", Title: "T"}, loc, nil)
	})
	// Every pointer a finding carries is an RFC 6901 pointer.
	for name, opt := range map[string]Option{
		"related":           WithRelated(Related{Location: Location{Origin: OriginDataDir}.WithPointer("a~2b"), Note: "n"}),
		"schema pointer":    WithSchema(Schema{Entry: "e", Pointer: "properties/x", Keyword: "k", KeywordLocation: "/k"}),
		"keyword location":  WithSchema(Schema{Entry: "e", Pointer: "/k", Keyword: "k", KeywordLocation: "k"}),
		"via":               WithVia(Via{Keyword: "anyOf", SchemaPointer: "anyOf"}),
		"condition pointer": WithVia(Via{Keyword: "then", SchemaPointer: "/then", Condition: &Condition{SchemaPointer: "~"}}),
	} {
		mustPanic(t, name, "not a JSON Pointer", func() {
			New(codeInputMinLength, input("a"), loc, Params{"limit": Int(1), "length": Int(0)}, opt)
		})
	}
	mustPanic(t, "a number past 2^53", "no exact canonical", func() { Int(1<<53 + 1) })
	mustPanic(t, "an array of arrays", "holds an array", func() { List(List()) })
}

func TestNewRendersTemplates(t *testing.T) {
	withTestCodes(t)
	f := New(codeInputRequired, input("vendor"), dataDir("v.yaml", 1, 1, "/vendor/name"), Params{"property": String("name")},
		WithField(Field{Title: "Name"}))
	if got, want := f.Message(), "Name is required."; got != want {
		t.Errorf("message %q, want %q", got, want)
	}
	if got, want := f.Hint(), "Add `name` to the input vendor."; got != want {
		t.Errorf("hint %q, want %q", got, want)
	}
	// {field} falls back to the pointer, then to the input itself.
	f = New(codeInputRequired, input("vendor"), dataDir("v.yaml", 1, 1, "/vendor/name"), Params{"property": String("name")})
	if got, want := f.Message(), "`/vendor/name` is required."; got != want {
		t.Errorf("message %q, want %q", got, want)
	}
	f = New(codeInputRequired, input("vendor"), dataDir("v.yaml", 1, 1, ""), Params{"property": String("name")})
	if got, want := f.Message(), "the input `vendor` is required."; got != want {
		t.Errorf("message %q, want %q", got, want)
	}
	// {package} and literal braces.
	f = New(codeManifestField, Subject{Kind: SubjectManifest}, dataDir("metais.yaml", 1, 1, "/x"), Params{"member": String("x"), "known": Bool(false)})
	if got, want := f.Hint(), "Remove it from <package>."; got != want {
		t.Errorf("hint %q, want %q", got, want)
	}
	f = New(codeWarnUnused, Subject{Kind: SubjectTemplate, Name: "t"}, dataDir("t", 1, 1, ""), nil)
	if got, want := f.Hint(), "Use it or remove it {literally}."; got != want {
		t.Errorf("hint %q, want %q", got, want)
	}
	if v, ok := f.Param("anything"); ok {
		t.Errorf("Param of an absent parameter = %v", v)
	}
}

func TestRenderRefusesBadTemplates(t *testing.T) {
	withTestCodes(t)
	f := New(codeUsageFlag, Subject{Kind: SubjectInvocation}, nil, nil)
	for _, tmpl := range []string{"{unclosed", "a } b", "{nope}"} {
		if _, err := f.render(tmpl); err == nil {
			t.Errorf("render(%q) succeeded", tmpl)
		}
	}
}

func TestNewCleansText(t *testing.T) {
	withTestCodes(t)
	f := New(codeInputRequired, Subject{Kind: SubjectInput, Name: "a\xffb"}, dataDir("f\xfe", 1, 1, "/x"), Params{"property": String("p\xff")})
	l, _ := f.Location()
	p, _ := f.Param("property")
	ps, _ := p.Str()
	for _, s := range []string{f.Subject().Name, l.File, ps, f.Hint()} {
		if strings.ContainsRune(s, 0xff) || !strings.Contains(s, "�") {
			t.Errorf("%q is not cleaned to valid UTF-8", s)
		}
	}
}

func TestLookupAndRegistryCopies(t *testing.T) {
	e, ok := Lookup(CodeLimitInvalid)
	if !ok || e.Code != CodeLimitInvalid || e.Class != ClassUsage {
		t.Fatalf("Lookup(limit.invalid) = %+v, %v", e, ok)
	}
	e.Params[0].Name = "changed"
	if again, _ := Lookup(CodeLimitInvalid); again.Params[0].Name == "changed" {
		t.Error("Lookup returned the registry's own slice")
	}
	r := Registry()
	r[0].Statements = append(r[0].Statements, "x")
	if Registry()[0].Statements != nil && len(Registry()[0].Statements) != 0 {
		t.Error("Registry returned the registry's own slice")
	}
	if _, ok := Lookup("no.such"); ok {
		t.Error("Lookup found an unregistered code")
	}
}
