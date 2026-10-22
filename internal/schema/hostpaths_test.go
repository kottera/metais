// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schema

import (
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/testenv"
)

// The tests in this file back the reasons TestNoHostFacts' dependency
// allow-list gives for the library's host-reaching code (internal/guarantees,
// dependencyAllowList): each shows the path NewCompiler controls never
// reaches it.

// builtInFormats are the formats the library validates itself (ADR-010
// Context, "JSON Schema 2020-12").
var builtInFormats = []string{
	"regex", "ipv4", "ipv6", "json-pointer", "relative-json-pointer", "uuid", "duration", "period",
	"hostname", "email", "date", "time", "date-time", "uri", "iri", "uri-reference", "iri-reference",
	"uri-template", "semver",
}

// TestBuiltInFormatsNeverRun backs the format.go time.Parse entry: every
// built-in format is either one METAIS registers its own validator for,
// which a compiled schema dispatches to (TestFormatsAreMETAIS), or one the
// pre-check refuses before the library sees the schema. So the library's
// own date validator, the one that calls time.Parse, never runs.
func TestBuiltInFormatsNeverRun(t *testing.T) {
	for _, name := range builtInFormats {
		if _, ours := formats[name]; ours {
			continue
		}
		_, fs := compileFiles(t, limits.Defaults(), map[string]string{"inputs/s.json": `{"format": "` + name + `"}`})
		if len(fs) != 1 || fs[0].Code() != CodeFormatUnsupported {
			t.Errorf("%s: neither METAIS's nor refused: %s", name, describe(fs))
		}
	}
	// date in particular: a validator that refuses everything, put in
	// METAIS's place, decides; the library's would accept this date.
	real := formats["date"]
	formats["date"] = func(string) bool { return false }
	defer func() { formats["date"] = real }()
	s := mustCompile(t, map[string]string{"inputs/s.json": `{"format": "date"}`}, "inputs/s.json")
	if fails, err := s.Validate(mustValue(t, `"2026-10-19"`)); err != nil || len(fails) == 0 {
		t.Error("the compiled schema did not use METAIS's date validator")
	}
}

// TestNoLoadReachesFileLoader backs the loader.go os.Open and runtime.GOOS
// entries: a load through the compiler NewCompiler configured, of a file
// that exists on the host, reaches METAIS's refusing loader, never
// FileLoader. The control is the bare library, whose FileLoader reads it.
func TestNoLoadReachesFileLoader(t *testing.T) {
	dir := testenv.Dir(t)
	host := filepath.Join(dir, "host.json")
	if err := os.WriteFile(host, []byte(`{"type": "string"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	u := (&url.URL{Scheme: "file", Path: filepath.ToSlash(host)}).String()
	if !strings.HasPrefix(u, "file:///") {
		u = "file:///" + strings.TrimPrefix(u, "file://")
	}
	if _, err := jsonschema.NewCompiler().Compile(u); err != nil {
		t.Fatalf("control: the bare library's FileLoader did not read the host file: %v", err)
	}
	c := compiled(t)
	_, err := c.lib.Compile(u)
	var le *jsonschema.LoadURLError
	if !errors.As(err, &le) || !errors.Is(le.Err, ErrRefused) {
		t.Errorf("a load through NewCompiler's compiler gave %v, want the refusing loader's refusal", err)
	}
}

// TestNoRelativeLocation backs the util.go filepath.Abs and runtime.GOOS
// entries: the library takes absolute()'s relative path, through
// filepath.Abs, only for a location that is not an absolute URI, and every
// location NewCompiler adds or compiles is one. A relative $ref resolves
// against the absolute URI of its file, and every failure names a
// metais:/// schema. The control: a relative location given the library
// directly does go through filepath.Abs, to a file:// URL of the working
// directory.
func TestNoRelativeLocation(t *testing.T) {
	c, fs := compileFiles(t, limits.Defaults(), map[string]string{
		"inputs/a.json":     `{"properties": {"d": {"$ref": "sub/d.json"}}}`,
		"inputs/sub/d.json": `{"type": "integer"}`,
	})
	if len(fs) > 0 {
		t.Fatal(describe(fs))
	}
	for loc := range c.roots {
		if u, err := url.Parse(loc); err != nil || !u.IsAbs() {
			t.Errorf("NewCompiler added %q, not an absolute URI", loc)
		}
	}
	s, _ := c.Schema("inputs/a.json")
	err := s.lib.Validate(toLibrary(mustValue(t, `{"d": "x"}`)))
	var ve *jsonschema.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("validation gave %v", err)
	}
	var walk func(*jsonschema.ValidationError)
	walk = func(e *jsonschema.ValidationError) {
		if !strings.HasPrefix(e.SchemaURL, "metais:///") {
			t.Errorf("a failure names %q", e.SchemaURL)
		}
		for _, c := range e.Causes {
			walk(c)
		}
	}
	walk(ve)

	_, err = c.lib.Compile("inputs/a.json")
	var le *jsonschema.LoadURLError
	if !errors.As(err, &le) || !strings.HasPrefix(le.URL, "file:///") {
		t.Errorf("control: a relative location did not go through filepath.Abs to a file URL: %v", err)
	}
}
