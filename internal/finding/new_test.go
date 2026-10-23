// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"strings"
	"testing"
	"unicode/utf8"
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
		for _, write := range []bool{false, true} {
			if _, err := f.render(tmpl, write); err == nil {
				t.Errorf("render(%q, %v) succeeded", tmpl, write)
			}
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
	n := len(r[0].Statements)
	r[0].Statements = append(r[0].Statements[:n:n], "x")
	if len(r[0].Statements) > 0 {
		r[0].Statements[0] = "changed"
	}
	if again := Registry()[0].Statements; len(again) != n || n > 0 && again[0] == "changed" {
		t.Error("Registry returned the registry's own slice")
	}
	if _, ok := Lookup("no.such"); ok {
		t.Error("Lookup found an unregistered code")
	}
}

// TestWithHint is a command's declared hint (ADR-008 D14 item 4): it
// replaces the code's hint, renders its placeholders as the registry's
// templates do, leaves every member of the finding as it was, and is
// refused when it names a placeholder the finding cannot fill.
func TestWithHint(t *testing.T) {
	withTestCodes(t)
	f := minLength("name", "inputs/a.json", 2, "/name", 3)
	g := f.WithHint("Write at least {limit} characters at {field}.")
	if want := "Write at least 10 characters at `/name`."; g.Hint() != want {
		t.Errorf("hint %q, want %q", g.Hint(), want)
	}
	if f.Hint() == g.Hint() || g.Message() != f.Message() || Compare(f, g) != 0 {
		t.Errorf("WithHint changed more than the hint: %q, %q", g.Message(), g.Hint())
	}
	mustPanic(t, "an unknown placeholder", "declared hint", func() { f.WithHint("Use {nothing}.") })
}

// TestWithHintValues is a declared hint with values of its own: text the
// package declares, such as a field's description, that no parameter
// carries. Each fills its placeholder in the hint and nowhere else, is
// shortened as a parameter's value is so that the hint stays within 240
// code points, and leaves every other member as it was.
func TestWithHintValues(t *testing.T) {
	withTestCodes(t)
	f := minLength("name", "inputs/a.json", 2, "/name", 3)
	g := f.WithHintValues("Write at least {limit} characters: {description}", map[string]string{"description": "The name, as it is signed."})
	if want := "Write at least 10 characters: The name, as it is signed."; g.Hint() != want {
		t.Errorf("hint %q, want %q", g.Hint(), want)
	}
	if g.Message() != f.Message() || Compare(f, g) != 0 {
		t.Errorf("WithHintValues changed more than the hint: %q", g.Message())
	}
	long := g.WithHintValues("Write it as {description} says.", map[string]string{"description": strings.Repeat("d", 400)})
	if n := utf8.RuneCountInString(long.Hint()); n != 240 || !strings.HasSuffix(long.Hint(), "… says.") {
		t.Errorf("a long value gives a hint of %d code points: %q", n, long.Hint())
	}
	mustPanic(t, "a value no placeholder names", "declared hint", func() { f.WithHintValues("Use {limit}.", map[string]string{"limit": "x"}) })
	mustPanic(t, "an unknown placeholder", "declared hint", func() { f.WithHintValues("Use {nothing}.", map[string]string{"other": "x"}) })
}
