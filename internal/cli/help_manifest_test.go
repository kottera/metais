// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/manifest"
	"metais.org/metais/internal/specdata"
	"metais.org/metais/internal/yamlx"
)

// entryPrefix is the path of a container entry's fields. An entry takes
// the fields of outputs[], so a field beneath it with no row of its own
// is covered by the outputs[] row of the same key.
const entryPrefix = "outputs[].container.entries[]."

// schemaFields is every field path the manifest schema declares, sorted:
// a property k of an object at path p is p.k (k at the root), an array's
// items at p are p[], and $ref, oneOf, anyOf and allOf stay at their
// path. A $ref already on the walk's stack is not expanded again, which
// ends the cycle of a container's entries holding a container. Neither
// if/then/else nor dependentSchemas is walked: in the 1.0 schema they
// re-declare properties the object already lists, and name no new key.
func schemaFields(schema []byte) ([]string, error) {
	var root map[string]any
	if err := json.Unmarshal(schema, &root); err != nil {
		return nil, err
	}
	defs, _ := root["$defs"].(map[string]any)
	set := map[string]bool{}
	var walk func(node any, path string, stack []string) error
	walk = func(node any, path string, stack []string) error {
		m, ok := node.(map[string]any)
		if !ok {
			return nil
		}
		if ref, ok := m["$ref"].(string); ok {
			name, local := strings.CutPrefix(ref, "#/$defs/")
			def, found := defs[name]
			if !local || !found {
				return fmt.Errorf("at %q: $ref %q names no $defs member", path, ref)
			}
			if !slices.Contains(stack, name) {
				if err := walk(def, path, append(slices.Clone(stack), name)); err != nil {
					return err
				}
			}
		}
		for _, k := range []string{"oneOf", "anyOf", "allOf"} {
			subs, _ := m[k].([]any)
			for _, s := range subs {
				if err := walk(s, path, stack); err != nil {
					return err
				}
			}
		}
		if props, ok := m["properties"].(map[string]any); ok {
			keys := make([]string, 0, len(props))
			for k := range props {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				p := k
				if path != "" {
					p = path + "." + k
				}
				set[p] = true
				if err := walk(props[k], p, stack); err != nil {
					return err
				}
			}
		}
		if items, ok := m["items"]; ok {
			return walk(items, path+"[]", stack)
		}
		return nil
	}
	if err := walk(root, "", nil); err != nil {
		return nil, err
	}
	out := make([]string, 0, len(set))
	for p := range set {
		out = append(out, p)
	}
	sort.Strings(out)
	return out, nil
}

// manifestDrift is every disagreement between the schema's field paths
// and the topic's rows: a schema path with no row, a row whose path the
// schema does not declare, and a path with more than one row.
func manifestDrift(schemaPaths []string, rows []manifestField) []string {
	var out []string
	count := map[string]int{}
	for _, r := range rows {
		count[r.Path]++
	}
	for _, p := range schemaPaths {
		if count[p] > 0 {
			continue
		}
		if rest, ok := strings.CutPrefix(p, entryPrefix); ok && count["outputs[]."+rest] > 0 {
			continue
		}
		out = append(out, "no row for the schema's "+p)
	}
	for _, r := range rows {
		if !slices.Contains(schemaPaths, r.Path) {
			out = append(out, "the row "+r.Path+" names no field of the schema")
		}
	}
	for _, r := range rows {
		if count[r.Path] > 1 {
			out = append(out, "the row "+r.Path+" is repeated")
			count[r.Path] = 0
		}
	}
	return out
}

// TestManifestTopicFields holds help manifest to the manifest schema: a
// row for every field the schema declares, none for a field it does not,
// and each row printed. It is pinned false with an invented field at the
// root, in an input and in a template entry alone, and with a row the
// schema lacks and a repeated row.
func TestManifestTopicFields(t *testing.T) {
	paths, err := schemaFields(specdata.ManifestSchema10)
	if err != nil {
		t.Fatal(err)
	}
	// The walk itself: a field at each depth, an entry's own field and
	// one it shares with outputs[], and the cycle cut at the nested
	// container's key.
	for _, want := range []string{"metais", "authors[].email", "inputs[].csv.delimiter",
		"outputs[].container.profile", "outputs[].container.entries[].first",
		"outputs[].container.entries[].template", "outputs[].container.entries[].container"} {
		if !slices.Contains(paths, want) {
			t.Errorf("schemaFields does not give %s; it gives %v", want, paths)
		}
	}
	if slices.ContainsFunc(paths, func(p string) bool { return strings.Contains(p, "entries[].container.") }) {
		t.Errorf("schemaFields expands the container cycle: %v", paths)
	}
	for _, d := range manifestDrift(paths, manifestFields) {
		t.Error(d)
	}
	text := runTool(t, "help", "manifest").stdout
	for _, f := range manifestFields {
		if !strings.Contains(text, "\n  "+f.Path+"\n") {
			t.Errorf("help manifest does not list %s on a line of its own", f.Path)
		}
	}

	// Pinned false: an invented field at the root, in an input and in a
	// template entry alone is drift, and so are a row the schema lacks
	// and a repeated row.
	var doc map[string]any
	if err := json.Unmarshal(specdata.ManifestSchema10, &doc); err != nil {
		t.Fatal(err)
	}
	invent := func(obj any) {
		obj.(map[string]any)["properties"].(map[string]any)["zz_new"] = map[string]any{"type": "string"}
	}
	defs := doc["$defs"].(map[string]any)
	invent(doc)
	invent(defs["input"])
	invent(defs["template_entry"])
	b, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	invented, err := schemaFields(b)
	if err != nil {
		t.Fatal(err)
	}
	rows := append(slices.Clone(manifestFields), manifestField{Path: "zz_gone"}, manifestFields[0])
	drift := manifestDrift(invented, rows)
	for _, want := range []string{
		"no row for the schema's zz_new",
		"no row for the schema's inputs[].zz_new",
		"no row for the schema's " + entryPrefix + "zz_new",
		"the row zz_gone names no field of the schema",
		"the row " + manifestFields[0].Path + " is repeated",
	} {
		if !slices.Contains(drift, want) {
			t.Errorf("the drift check does not report %q; it reports %q", want, drift)
		}
	}
	if base := manifestDrift(paths, manifestFields); len(drift) != len(base)+5 {
		t.Errorf("the drift check reports %q, want five findings beyond %q", drift, base)
	}
}

// buildManifest is the manifest the YAML text src is, as metais check
// reads it: parsed under the YAML profile and judged by every rule of the
// manifest schema and beside it.
func buildManifest(t *testing.T, src string) error {
	t.Helper()
	doc, fds := yamlx.Parse([]byte(src), yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectManifest}}, limits.Defaults())
	if len(fds) > 0 {
		return fmt.Errorf("parse: %s", fds[0].Message())
	}
	_, err := manifest.Build(doc.Value)
	return err
}

// TestManifestTopicExample holds every manifest help manifest shows to
// the rules it states: each example builds, and leads the topic; and each
// row's example is one YAML line setting the row's own key, printed
// within the help's width.
func TestManifestTopicExample(t *testing.T) {
	for name, outputs := range map[string]string{"leading": topicManifestOutputs, "fan-out": topicManifestFanOut} {
		if err := buildManifest(t, topicManifestHead+outputs); err != nil {
			t.Errorf("the %s example is not a valid manifest: %v", name, err)
		}
	}
	// The pin: the check refuses a manifest the rules refuse.
	if buildManifest(t, topicManifestHead+strings.Replace(topicManifestOutputs, "escape: html", "escape: xml", 1)) == nil {
		t.Error("buildManifest accepts escape: xml")
	}

	text := runTool(t, "help", "manifest").stdout
	// The help indents each example's lines by two spaces, and a blank
	// line not at all.
	indent := func(s string) string {
		lines := strings.Split(strings.TrimSuffix(s, "\n"), "\n")
		for i, l := range lines {
			if l != "" {
				lines[i] = "  " + l
			}
		}
		return strings.Join(lines, "\n") + "\n"
	}
	lead := strings.Index(text, indent(topicManifestHead+topicManifestOutputs))
	first := strings.Index(text, "\n  "+manifestFields[0].Path+"\n")
	if lead < 0 || first < 0 || lead > first {
		t.Errorf("help manifest does not open with the leading example before its fields:\n%s", text)
	}
	if i := strings.Index(text, indent(topicManifestFanOut)); i < 0 || i > first {
		t.Errorf("help manifest does not show the fan-out example before its fields")
	}

	for _, f := range manifestFields {
		if f.Text == "" || f.Example == "" {
			t.Errorf("the row %s has no text or no example", f.Path)
			continue
		}
		if n := len("      Example: " + f.Example); n > helpWidth {
			t.Errorf("the row %s's example is %d columns", f.Path, n)
		}
		doc, fds := yamlx.Parse([]byte(f.Example+"\n"), yamlx.Source{Subject: finding.Subject{Kind: finding.SubjectManifest}}, limits.Defaults())
		if len(fds) > 0 {
			t.Errorf("the row %s's example %q is not YAML: %s", f.Path, f.Example, fds[0].Message())
			continue
		}
		obj, _ := doc.Value.(value.Object)
		key := f.Path[strings.LastIndexAny(f.Path, ".]")+1:]
		if _, ok := obj.Get(key); !ok || obj.Len() != 1 {
			t.Errorf("the row %s's example %q does not set %s alone", f.Path, f.Example, key)
		}
	}
}
