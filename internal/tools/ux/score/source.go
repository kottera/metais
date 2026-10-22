// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"slices"
	"strings"

	"metais.org/metais/internal/finding"
	"metais.org/metais/internal/jsonx"
	"metais.org/metais/internal/limits"
	"metais.org/metais/internal/yamlx"
)

// source is what the scorer knows of a fixture from its source directory:
// the package's inputs and their required fields, and its shipped
// examples.
type source struct {
	inputs   []input
	examples []example
	// files are the source's files' bytes, by path.
	files map[string][]byte
}

type input struct {
	name string
	// required are the input's required top-level fields, each with the
	// names a session may call it by: its property name and its title.
	required [][]string
}

type example struct {
	name string
	// inputs are the example's input values, by input.
	inputs map[string]any
	// expected are the digests of the outputs it should render.
	expected map[string]bool
}

// manifest is what the scorer reads of a metais.yaml.
type manifest struct {
	Inputs []struct {
		Name   string `json:"name"`
		Schema string `json:"schema"`
	} `json:"inputs"`
	Outputs []struct {
		Path     string `json:"path"`
		Template string `json:"template"`
		Escape   string `json:"escape"`
	} `json:"outputs"`
}

// readManifest reads a metais.yaml, as metais reads one.
func readManifest(b []byte) (manifest, bool) {
	var m manifest
	v, ok := readYAML(b, finding.SubjectManifest, limits.ManifestSize)
	if !ok {
		return m, false
	}
	j, _ := json.Marshal(v)
	return m, json.Unmarshal(j, &m) == nil
}

// readYAML reads a YAML document through metais's own reading of YAML,
// internal/yamlx (ADR-010 D6 item 5), as the value JSON would decode it
// to.
func readYAML(b []byte, kind finding.SubjectKind, size limits.ID) (any, bool) {
	doc, fs := yamlx.Parse(bytes.TrimPrefix(b, []byte("\uFEFF")), yamlx.Source{Subject: finding.Subject{Kind: kind}, Size: size}, limits.Defaults())
	if len(fs) > 0 {
		return nil, false
	}
	var v any
	if json.Unmarshal(jsonx.Canonical(doc.Value), &v) != nil {
		return nil, false
	}
	return v, true
}

func readSource(fsys fs.FS) (*source, error) {
	s := &source{files: map[string][]byte{}}
	err := fs.WalkDir(fsys, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return err
		}
		b, err := fs.ReadFile(fsys, p)
		s.files[p] = b
		return err
	})
	if err != nil {
		return nil, err
	}
	m, ok := readManifest(s.files["metais.yaml"])
	if !ok {
		return nil, fmt.Errorf("metais.yaml is not a manifest metais reads")
	}
	for _, in := range m.Inputs {
		i := input{name: in.Name}
		var schema struct {
			Required   []string `json:"required"`
			Properties map[string]struct {
				Title string `json:"title"`
			} `json:"properties"`
		}
		if b, ok := s.files[in.Schema]; ok {
			if err := json.Unmarshal(b, &schema); err != nil {
				return nil, fmt.Errorf("%s: %w", in.Schema, err)
			}
		}
		for _, f := range schema.Required {
			names := []string{f}
			if t := schema.Properties[f].Title; t != "" {
				names = append(names, t)
			}
			i.required = append(i.required, names)
		}
		s.inputs = append(s.inputs, i)
	}
	names := map[string]bool{}
	for p := range s.files {
		if parts := strings.Split(p, "/"); len(parts) > 2 && parts[0] == "examples" {
			names[parts[1]] = true
		}
	}
	for _, n := range slices.Sorted(func(yield func(string) bool) {
		for n := range names {
			if !yield(n) {
				return
			}
		}
	}) {
		e := example{name: n, inputs: map[string]any{}, expected: map[string]bool{}}
		for p, b := range s.files {
			switch {
			case strings.HasPrefix(p, "examples/"+n+"/inputs/"):
				name := strings.TrimSuffix(path.Base(p), path.Ext(p))
				if v, ok := parseData(path.Ext(p), b); ok {
					e.inputs[name] = v
				}
			case strings.HasPrefix(p, "examples/"+n+"/expected/"):
				e.expected[digest(b)] = true
			}
		}
		s.examples = append(s.examples, e)
	}
	return s, nil
}

func digest(b []byte) string {
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])
}

// parseData parses a data file by its extension, JSON or YAML, into the
// values JSON would decode it to, so the two compare.
func parseData(ext string, b []byte) (any, bool) {
	var v any
	switch ext {
	case ".json":
		if json.Unmarshal(b, &v) != nil {
			return nil, false
		}
	case ".yaml", ".yml":
		var ok bool
		if v, ok = readYAML(b, finding.SubjectInput, limits.InputSize); !ok {
			return nil, false
		}
	default:
		return nil, false
	}
	return v, true
}

// supplied is what a render or a validate was given: its data
// directory's inputs, its --inputs-json and --set values, or --sample.
type supplied struct {
	sample bool
	// inputs are the values it was given, by input, as far as the run's
	// directory still holds them.
	inputs map[string]any
	// digests are the digests of the data files it was given.
	digests []string
	// dirs are the data directories, relative to the run's directory.
	dirs []string
	// sets counts --set and --set-json values.
	sets int
	// unseen names data the scorer cannot read: a data directory outside
	// the run's, or gone from it.
	unseen []string
}

func (s *supplied) any() bool {
	return s.sample || len(s.inputs) > 0 || len(s.digests) > 0 || s.sets > 0 || len(s.unseen) > 0
}

// supply reads what a metais call was given.
func (r *run) supply(c *call) *supplied {
	s := &supplied{sample: hasFlag(c.args, "--sample"), inputs: map[string]any{}}
	for _, d := range flagValues(c.args, "--data") {
		rel := r.resolve(d, c.cwd)
		if rel == "" {
			s.unseen = append(s.unseen, d)
			continue
		}
		s.dirs = append(s.dirs, rel)
		files := r.filesUnder(rel)
		for _, f := range files {
			b, _ := r.file(f)
			if v, ok := parseData(path.Ext(f), b); ok {
				s.inputs[strings.TrimSuffix(path.Base(f), path.Ext(f))] = v
			}
		}
		for _, f := range r.rec.Files {
			if p := path.Join("work", f.Path); strings.HasPrefix(p, rel+"/") {
				s.digests = append(s.digests, f.SHA256)
			}
		}
		if len(files) == 0 && len(s.digests) == 0 {
			s.unseen = append(s.unseen, d)
		}
	}
	for _, j := range flagValues(c.args, "--inputs-json") {
		var m map[string]any
		if json.Unmarshal([]byte(j), &m) == nil {
			for k, v := range m {
				s.inputs[k] = v
			}
		} else {
			s.unseen = append(s.unseen, "--inputs-json "+j)
		}
	}
	s.sets = len(flagValues(c.args, "--set")) + len(flagValues(c.args, "--set-json"))
	return s
}

// shipped is the shipped example the data copies, or "": every input
// supplied equal to the example's, or a data file byte for byte one of
// its inputs.
func (src *source) shipped(s *supplied) string {
	for _, e := range src.examples {
		if len(s.inputs) > 0 && s.sets == 0 {
			same := true
			for k, v := range s.inputs {
				if !sameValue(e.inputs[k], v) {
					same = false
					break
				}
			}
			if same {
				return e.name
			}
		}
		for p, b := range src.files {
			if strings.HasPrefix(p, "examples/"+e.name+"/inputs/") && slices.Contains(s.digests, digest(b)) && len(s.inputs) == 0 {
				return e.name
			}
		}
	}
	return ""
}

// sameValue reports whether two decoded JSON values are equal: their
// encodings are, as encoding/json writes an object's members in key order.
func sameValue(a, b any) bool {
	ja, errA := json.Marshal(a)
	jb, errB := json.Marshal(b)
	return errA == nil && errB == nil && string(ja) == string(jb)
}

// renderedShipped is the shipped example whose expected output a render
// wrote, byte for byte, or "".
func (src *source) renderedShipped(digests []string) string {
	for _, e := range src.examples {
		for _, d := range digests {
			if e.expected[d] {
				return e.name
			}
		}
	}
	return ""
}

// render is one metais render the session ran that wrote its outputs.
type render struct {
	*call
	given *supplied
	// digests are the digests of the files it reports it wrote.
	digests []string
	// out is its output directory relative to the run's directory, or ""
	// for standard output or a directory outside the run's.
	out string
	// stdout marks a render to standard output (--out -) that wrote
	// there: its bytes are in the call's output, not in a file.
	stdout bool
	// files are the files of the run's directory it wrote, as far as the
	// directory still holds them.
	files []string
	// readBack is the command of its own Bash call that read back a file
	// it wrote, when the render is seen to have written by that alone.
	readBack *cmd
}

// reportedDigest is a file render lists as written: a line of its human
// listing, or a file of its JSON result.
var reportedDigest = regexp.MustCompile(`(?m)(?:^\d+\t|"digest":")sha256:([0-9a-f]{64})`)

// refused is a metais refusal in a command's output: its human report's
// error line, or its JSON result's ok of false.
var refused = regexp.MustCompile(`(?m)^metais: error:|"ok":false`)

// renders are the renders that wrote their outputs: metais render, not a
// dry run, that lists the files it wrote or left them in the run's
// directory, or that rendered to standard output (--out -) and wrote
// there. A refused render writes nothing and lists nothing, and to
// standard output it writes nothing at all (MTS-RND-011), so a render
// there wrote when the harness reports no error for its call and the
// call's output is not empty and holds no refusal.
func (r *run) renders() []*render {
	var out []*render
	for _, c := range r.calls {
		if c.sub != "render" || hasFlag(c.args, "--dry-run") {
			continue
		}
		rd := &render{call: c, given: r.supply(c)}
		for _, m := range reportedDigest.FindAllStringSubmatch(c.output(), -1) {
			rd.digests = append(rd.digests, m[1])
		}
		if o := flagValues(c.args, "--out"); len(o) > 0 && o[0] != "-" {
			rd.out = r.resolve(o[0], c.cwd)
		} else {
			for _, f := range c.s.outs {
				if p := r.resolve(f, c.cwd); p != "" {
					rd.files = append(rd.files, p)
				}
			}
			text := c.output()
			rd.stdout = len(o) > 0 && c.ok() && strings.TrimSpace(text) != "" && !refused.MatchString(text)
		}
		if rd.out != "" {
			rd.files = append(rd.files, r.filesUnder(rd.out)...)
			for _, f := range r.rec.Files {
				if p := path.Join("work", f.Path); strings.HasPrefix(p, rd.out+"/") && f.Change == "added" {
					rd.digests = append(rd.digests, f.SHA256)
				}
			}
		}
		if len(rd.digests) == 0 && len(rd.files) == 0 && !rd.stdout {
			if rd.readBack = r.readBack(c.cmd, rd.out); rd.readBack == nil {
				continue
			}
		}
		out = append(out, rd)
	}
	return out
}

// readers are the commands that print what a file holds.
var readers = []string{"cat", "head", "tail", "sed", "grep", "awk", "less", "more", "wc", "diff", "cmp", "od", "xxd", "hexdump", "jq", "shasum", "sha256sum", "md5sum"}

// missing is a reader's report that a file it was given is not there.
var missing = regexp.MustCompile(`(?i)no such file`)

// readBack is the later command of a render's own Bash call that read a
// file beneath the render's output directory, out, when the call's output
// shows the read found it: it holds no refusal and no report of a missing
// file. A render whose listing went elsewhere and whose files the session
// deleted is seen to have written by that read, and only by it.
func (r *run) readBack(c *cmd, out string) *cmd {
	text := c.output()
	if out == "" || refused.MatchString(text) || missing.MatchString(text) {
		return nil
	}
	for _, o := range r.cmds {
		if o.use != c.use || o.seq <= c.seq || !slices.Contains(readers, o.name) {
			continue
		}
		for _, w := range append(slices.Clone(o.rest), o.s.ins...) {
			if p := r.resolve(w, o.cwd); strings.HasPrefix(p, out+"/") {
				return o
			}
		}
	}
	return nil
}

// fixtureSource is the source of the scenario's first fixture, or nil.
func (r *run) fixtureSource() *source {
	for _, f := range r.rec.Fixtures {
		if s := r.sources[f.Path]; s != nil {
			return s
		}
	}
	for _, s := range r.sources {
		return s
	}
	return nil
}
