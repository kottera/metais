// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package schemafail

import (
	"encoding/json"
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/finding"
)

// docs is a set of schema entries, decoded, that Schemas reads.
type docs map[string]any

func newDocs(t *testing.T, entries map[string]string) docs {
	t.Helper()
	d := docs{}
	for name, src := range entries {
		var v any
		if err := json.Unmarshal([]byte(src), &v); err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		d[name] = v
	}
	return d
}

func (d docs) object(entry, ptr string) map[string]any {
	v := d[entry]
	for _, tok := range tokens(ptr) {
		switch c := v.(type) {
		case map[string]any:
			v = c[tok]
		case []any:
			i, err := strconv.Atoi(tok)
			if err != nil || i < 0 || i >= len(c) {
				return nil
			}
			v = c[i]
		default:
			return nil
		}
	}
	o, _ := v.(map[string]any)
	return o
}

func (d docs) Annotations(entry, ptr string) Annotations {
	o := d.object(entry, ptr)
	title, _ := o["title"].(string)
	desc, _ := o["description"].(string)
	wo, _ := o["writeOnly"].(bool)
	pat, _ := o["pattern"].(string)
	return Annotations{Title: title, Description: desc, WriteOnly: wo, Pattern: pat}
}

// Ref resolves a $ref of the form "<entry>#<pointer>", the entry empty
// for the same one.
func (d docs) Ref(entry, ptr string) (string, string, bool) {
	ref, ok := d.object(entry, ptr)["$ref"].(string)
	if !ok {
		return "", "", false
	}
	e, frag, _ := strings.Cut(ref, "#")
	if e == "" {
		e = entry
	}
	return e, frag, true
}

func (d docs) Properties(entry, ptr string) []string {
	props, _ := d.object(entry, ptr)["properties"].(map[string]any)
	var out []string
	for k := range props {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// locate is a Target's Locate: the line is one more than the pointer's
// depth, and a member name is at column 3, its value at column 1.
func locate(p *finding.Path, key bool) finding.Location {
	col := 1
	if key {
		col = 3
	}
	return finding.Location{Origin: finding.OriginDataDir, File: "vendor.json", Line: p.Depth() + 1, Column: col}
}

var vendor = finding.Subject{Kind: finding.SubjectInput, Name: "vendor", Title: "Vendor"}

func target(d docs) Target {
	return Target{Family: FamilyInput, Subject: vendor, Schemas: d, Locate: locate}
}

// mapAll maps fs for tg and returns the findings in report order.
func mapAll(t *testing.T, tg Target, fs ...Failure) []finding.Finding {
	t.Helper()
	got, err := Map(tg, fs...)
	if err != nil {
		t.Fatal(err)
	}
	r := finding.NewReport(10000)
	for _, f := range got {
		r.Add(f)
	}
	return r.Findings()
}

// briefs describes each finding by brief.
func briefs(fs []finding.Finding) []string {
	var out []string
	for _, f := range fs {
		out = append(out, brief(f))
	}
	return out
}

// brief is a finding's normative members in one line: its code and
// pointer, its params, and its field, via and alternatives when it has
// them.
func brief(f finding.Finding) string {
	var j map[string]any
	if err := json.Unmarshal(finding.AppendJSON(nil, f)[1:len(finding.AppendJSON(nil, f))-1], &j); err != nil {
		panic(err)
	}
	return briefJSON(j)
}

func briefJSON(j map[string]any) string {
	var b strings.Builder
	b.WriteString(j["code"].(string))
	if l, ok := j["location"].(map[string]any); ok {
		fmt.Fprintf(&b, " %q", l["pointer"])
	}
	params, _ := json.Marshal(j["params"])
	b.WriteString(" " + string(params))
	if fd, ok := j["field"].(map[string]any); ok {
		fmt.Fprintf(&b, " field=%v|%v", orEmpty(fd["title"]), orEmpty(fd["description"]))
	}
	if vs, ok := j["via"].([]any); ok {
		var parts []string
		for _, v := range vs {
			v := v.(map[string]any)
			s := fmt.Sprintf("%v@%v", v["keyword"], v["schema_pointer"])
			if br, ok := v["branch"]; ok {
				s += fmt.Sprintf("#%v", br)
			}
			if t, ok := v["title"]; ok {
				s += fmt.Sprintf("(%v)", t)
			}
			if c, ok := v["condition"].(map[string]any); ok {
				s += fmt.Sprintf("<%v:%v>", c["schema_pointer"], orEmpty(c["title"]))
			}
			parts = append(parts, s)
		}
		b.WriteString(" via=" + strings.Join(parts, ","))
	}
	if as, ok := j["alternatives"].([]any); ok {
		var parts []string
		for _, a := range as {
			a := a.(map[string]any)
			var fs []string
			for _, f := range a["findings"].([]any) {
				fs = append(fs, briefJSON(f.(map[string]any)))
			}
			s := fmt.Sprintf("%v", a["branch"])
			if t, ok := a["title"]; ok {
				s += fmt.Sprintf("(%v)", t)
			}
			if o, ok := a["omitted"]; ok && o != 0.0 {
				s += fmt.Sprintf("+%v", o)
			}
			parts = append(parts, s+": "+strings.Join(fs, "; "))
		}
		b.WriteString(" alts=[" + strings.Join(parts, " | ") + "]")
	}
	return b.String()
}

func orEmpty(v any) any {
	if v == nil {
		return ""
	}
	return v
}

func checkBriefs(t *testing.T, name string, got []finding.Finding, want ...string) {
	t.Helper()
	if g := briefs(got); !slices.Equal(g, want) {
		t.Errorf("%s:\n got  %s\n want %s", name, strings.Join(g, "\n      "), strings.Join(want, "\n      "))
	}
}

// at builds a keyword failure in the vendor schema at pointer ptr, its
// evaluation path the same, on the instance at inst.
func at(kw, ptr string, inst []string, data Data, children ...Failure) Failure {
	return Failure{Keyword: kw, KeywordLocation: ptr, Entry: "vendor.schema.json", SchemaPointer: ptr, InstancePointer: inst, Data: data, Children: children}
}

func path(toks ...string) []string { return toks }
