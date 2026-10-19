// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package finding

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/specdata"
)

// spec/findings/registry.json is generated from the tool's table by
//
//	go generate ./internal/finding
//
// which runs TestGenerateRegistry with -finding.generate (ADR-006 A2).
// go generate ./internal/specdata then copies it for the binary, and go
// generate ./internal/guarantees regenerates the catalog from it.

//go:generate go test -count=1 -run=^TestGenerateRegistry$ -args -finding.generate

var generateRegistry = flag.Bool("finding.generate", false, "rewrite spec/findings/registry.json from the tool's table")

// registryPath is spec/findings/registry.json, from this package's
// directory, where go test runs.
var registryPath = filepath.Join("..", "..", "spec", "findings", "registry.json")

// The registry file's members, in the order the file writes them.
type fileParam struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Items   string `json:"items,omitempty"`
	Meaning string `json:"meaning"`
	Always  bool   `json:"always"`
}

type fileEntry struct {
	Code       Code        `json:"code"`
	Severity   string      `json:"severity"`
	Class      string      `json:"class"`
	Subject    string      `json:"subject"`
	Params     []fileParam `json:"params"`
	Message    string      `json:"message"`
	Hint       string      `json:"hint"`
	Location   string      `json:"location"`
	Since      string      `json:"since"`
	Retired    string      `json:"retired,omitempty"`
	Statements []string    `json:"statements"`
}

// registryFile is spec/findings/registry.json for entries: two-space
// indented, one member per line, ending in one LF.
func registryFile(entries []Entry) ([]byte, error) {
	var r struct {
		Codes []fileEntry `json:"codes"`
	}
	r.Codes = []fileEntry{}
	for _, e := range entries {
		f := fileEntry{
			Code:       e.Code,
			Severity:   e.Severity.String(),
			Class:      e.Class.String(),
			Subject:    e.Subject.String(),
			Params:     []fileParam{},
			Message:    e.Message,
			Hint:       e.Hint,
			Location:   e.Span.String(),
			Since:      e.Since,
			Retired:    e.Retired,
			Statements: append([]string{}, e.Statements...),
		}
		for _, p := range e.Params {
			fp := fileParam{Name: p.Name, Type: p.Type.String(), Meaning: p.Meaning, Always: p.Always}
			if p.Items != 0 {
				fp.Items = p.Items.String()
			}
			f.Params = append(f.Params, fp)
		}
		r.Codes = append(r.Codes, f)
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(r); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// TestGenerateRegistry is the generator go generate runs. Without
// -finding.generate it does nothing.
func TestGenerateRegistry(t *testing.T) {
	if !*generateRegistry {
		t.Skip("run by go generate ./internal/finding")
	}
	b, err := registryFile(Registry())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(registryPath, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// registryFileProblem describes how got, a committed registry file,
// differs from what the table generates, or returns "".
func registryFileProblem(got []byte, entries []Entry) string {
	want, err := registryFile(entries)
	if err != nil {
		return err.Error()
	}
	if !bytes.Equal(got, want) {
		return "spec/findings/registry.json differs from what the tool's table generates; run go generate ./internal/finding ./internal/specdata ./internal/guarantees"
	}
	return ""
}

// TestRegistryFileGenerated is the drift test of the generated registry
// file: spec/findings/registry.json, and the copy the binary embeds, are
// byte for byte what the table generates, so a hand edit of either fails.
func TestRegistryFileGenerated(t *testing.T) {
	committed, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatal(err)
	}
	if p := registryFileProblem(committed, Registry()); p != "" {
		t.Error(p)
	}
	if p := registryFileProblem(specdata.FindingRegistry, Registry()); p != "" {
		t.Error("the embedded copy: " + p)
	}
}

// TestRegistryFileProblemCatchesEdits is the control: an edit of the
// file that leaves its meaning unchanged, and a change to the table, are
// both caught.
func TestRegistryFileProblemCatchesEdits(t *testing.T) {
	file, err := registryFile(Registry())
	if err != nil {
		t.Fatal(err)
	}
	if p := registryFileProblem(file, Registry()); p != "" {
		t.Fatalf("the generated file is not current: %s", p)
	}
	for name, edited := range map[string]string{
		"reindented":       strings.Replace(string(file), "\n    {", "\n  {", 1),
		"a trailing space": strings.Replace(string(file), "\n", " \n", 1),
		"no final LF":      strings.TrimSuffix(string(file), "\n"),
		"a member edited":  strings.Replace(string(file), `"since": "1.0"`, `"since": "1.1"`, 1),
	} {
		if registryFileProblem([]byte(edited), Registry()) == "" {
			t.Errorf("%s: not caught", name)
		}
	}
	changed := Registry()
	changed[0].Hint += "."
	if registryFileProblem(file, changed) == "" {
		t.Error("a change to the table is not caught")
	}
}
