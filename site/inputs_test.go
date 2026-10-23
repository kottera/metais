// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package site

import (
	"bytes"
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the site package's generated input schemas from docs/cli/schemas")

// cliResultSchemas are the site package's input schemas that are the result
// of a command's JSON document, each generated from that document's schema
// under docs/cli/schemas, so the package validates exactly what the
// command's own schema says it prints: the input's schema in the package,
// and the schema it is generated from.
var cliResultSchemas = []struct{ input, cli string }{
	{"inputs/help.schema.json", "metais.help.1.json"},
	{"inputs/tool.schema.json", "metais.version.1.json"},
}

// resultSchema is the schema of the result of the document whose schema is
// doc: doc's $defs/result as the root, beside its other $defs and the
// envelope's that it names, each $ref into the envelope made a $ref into the same file.
// A package's schema is self-contained, so it cannot name the envelope's
// file.
func resultSchema(doc, envelope []byte) ([]byte, error) {
	var d, e struct {
		Title string                     `json:"title"`
		Defs  map[string]json.RawMessage `json:"$defs"`
	}
	if err := json.Unmarshal(doc, &d); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(envelope, &e); err != nil {
		return nil, err
	}
	var root map[string]any
	if err := json.Unmarshal(d.Defs["result"], &root); err != nil {
		return nil, err
	}
	defs := map[string]json.RawMessage{}
	for name, def := range e.Defs {
		if bytes.Contains(doc, []byte(`"metais.envelope.1.json#/$defs/`+name+`"`)) {
			defs[name] = def
		}
	}
	for name, def := range d.Defs {
		if name != "result" {
			defs[name] = def
		}
	}
	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	root["title"] = "The result of a " + d.Title + " document"
	root["description"] = "Generated from docs/cli/schemas by site/inputs_test.go's TestCLIResultSchemas; do not edit."
	if len(defs) > 0 {
		root["$defs"] = defs
	}
	b, err := json.MarshalIndent(root, "", "  ")
	if err != nil {
		return nil, err
	}
	b = bytes.ReplaceAll(b, []byte(`"metais.envelope.1.json#/$defs/`), []byte(`"#/$defs/`))
	if bytes.Contains(b, []byte(`.json#`)) || bytes.Contains(b, []byte(`"metais.envelope.1.json"`)) {
		return nil, os.ErrInvalid
	}
	return append(b, '\n'), nil
}

// TestCLIResultSchemas holds each of cliResultSchemas to the schema it is
// generated from. Run with -update to rewrite them.
func TestCLIResultSchemas(t *testing.T) {
	envelope, err := os.ReadFile("../docs/cli/schemas/metais.envelope.1.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range cliResultSchemas {
		doc, err := os.ReadFile(filepath.Join("../docs/cli/schemas", s.cli))
		if err != nil {
			t.Fatal(err)
		}
		want, err := resultSchema(doc, envelope)
		if err != nil {
			t.Fatalf("%s: %v", s.cli, err)
		}
		path := filepath.Join("package", filepath.FromSlash(s.input))
		if *update {
			if err := os.WriteFile(path, want, 0o644); err != nil {
				t.Fatal(err)
			}
			continue
		}
		got, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("site/package/%s differs from the result of docs/cli/schemas/%s; rewrite it with\n"+
				"  go test ./site -run '^TestCLIResultSchemas$' -args -update", s.input, s.cli)
		}
	}
}
