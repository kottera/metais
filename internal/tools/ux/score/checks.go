// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kottera/metais/internal/tools/ux/streamjson"
)

// check is one scenario's outcome check (plan §5's table), and what it
// cannot see, which a scorecard states beside its results.
type check struct {
	run    func(r *run) Result
	limits string
}

// checks are the outcome checks, by the name a scenario's declaration
// gives as its check.
var checks = map[string]check{
	"either-reading": {eitherReading,
		"Whether the session said which reading it took, and whether it presented invented facts as real, is decided from its words by phrase: a statement of its reading (\"I took this to mean…\", \"I interpreted…\", \"assuming you meant…\") naming what it did, and a word that marks its data as made up (fictional, sample, placeholder…). A statement worded outside those phrases is not seen, and a fact the session invented is not seen if its words call the data fictional."},
	"asks-not-invents": {asksNotInvents,
		"Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request (\"please provide\", \"send me\", \"let me know\") in the session's last words."},
	"new-fictional-render": {newFictionalRender,
		"New data is data that is not a shipped example's: equal to one, input by input, or rendering one of its expected outputs byte for byte. A copy with a value changed is new."},
	"generated-preview": {generatedPreview,
		"Generated data is metais's: render --sample, or a data directory metais sample wrote. Data the session wrote by hand is not generated."},
	"names-every-input": {namesEveryInput,
		"An input or a required field is named when its name, spelt with underscores, spaces or hyphens, or its schema title appears in the session's last words. The required fields are each input schema's top-level required list; a field the stage makes mandatory through a condition is not counted."},
	"script-renders": {scriptRenders,
		"A script validates and renders through metais when its text names metais, validate and render. Its test run passes when the session runs it and the harness reports no error; the scorer does not run it again."},
	"new-package": {newPackage,
		"check passing is seen through pack, which checks and tests the package before it writes it."},
	"schema-and-template": {schemaAndTemplate,
		"A change is seen in the files the session left, compared with the package's source, or in what its Write and Edit calls wrote: a schema and a template that each gain a due date. A change made by a command line whose files are gone is not seen."},
	"data-to-js": {dataToJS,
		"The page's data parses when a script element of a rendered HTML output holds a JSON array. Whether the array is the numbers given is not checked."},
}

// CheckLimits is what a check cannot see, as a scorecard states it.
func CheckLimits(name string) string { return checks[name].limits }

// outcome runs the scenario's check.
func outcome(r *run) Result { return checks[r.checkName].run(r) }

// Phrases the session's own words are read for. Each is a narrower,
// mechanical stand-in for a judgement the table asks for, and the check
// that uses it states so in its limits.
var (
	// readingStated is a statement of how the session read the request.
	readingStated = regexp.MustCompile(`(?i)\b(?:interpret(?:ed|ing|ation)?|i\s+(?:took|take|read|understood|treated|am\s+treating|'m\s+treating|chose|went\s+with)\s+(?:it|this|that|the\s+request|your\s+request|"fill|'fill|“fill|` + "`fill" + `|fill)|took\s+["'“` + "`" + `]?fill|read\s+["'“` + "`" + `]?fill|assum(?:ed|ing)\s+(?:you|that\s+you)\s+(?:meant|mean|want|wanted)|(?:two|both)\s+(?:ways|readings|interpretations)|my\s+reading|the\s+reading\s+i)\b`)
	// renderReading and editReading name the two readings of C0.
	renderReading = regexp.MustCompile(`(?i)\brender`)
	editReading   = regexp.MustCompile(`(?i)\b(?:edit|rewr(?:ite|ote|itten)|repack|unpack|updat|chang|modif|regenerat|replac)`)
	// madeUp marks data as not real.
	madeUp = regexp.MustCompile(`(?i)\b(?:fictional|fictitious|made[- ]up|invented|sample|placeholder|dummy|synthetic|illustrative|fake|hypothetical|not\s+real|test\s+data|example\s+data|generated)\b`)
	// asks is a request for what the session needs.
	asks = regexp.MustCompile(`(?i)\?|\b(?:please\s+(?:provide|send|share|give|supply|tell)|(?:send|share|give|provide|supply|tell)\s+me|let\s+me\s+know|if\s+you\s+(?:provide|send|share|give|supply|have)|i\s+(?:will|'ll)?\s*need|once\s+you)\b`)
)

// statement is the first of the session's words that states how it read
// the request and names the reading.
func (r *run) statement(reading *regexp.Regexp) (streamjson.Text, bool) {
	for _, t := range r.texts() {
		for _, para := range strings.Split(t.Text, "\n") {
			if readingStated.MatchString(para) && reading.MatchString(para) {
				return streamjson.Text{Line: t.Line, Turn: t.Turn, Text: clip(para)}, true
			}
		}
	}
	return streamjson.Text{}, false
}

// labelled is the first of the session's words that marks its data as
// made up.
func (r *run) labelled() (streamjson.Text, bool) {
	for _, t := range r.texts() {
		for _, para := range strings.Split(t.Text, "\n") {
			if madeUp.MatchString(para) {
				return streamjson.Text{Line: t.Line, Turn: t.Turn, Text: clip(para)}, true
			}
		}
	}
	return streamjson.Text{}, false
}

func textEvidence(t streamjson.Text) Evidence {
	return Evidence{Turn: t.Turn, Line: t.Line, Text: t.Text}
}

// succeeded are the metais calls of the commands named that the harness
// reports no error for.
func (r *run) succeeded(subs ...string) []*call {
	var out []*call
	for _, c := range r.calls {
		if slices.Contains(subs, c.sub) && c.ok() {
			out = append(out, c)
		}
	}
	return out
}

// lastWrite is the transcript line of the session's last Write or Edit,
// or of its last command that edits a file in place, or 0.
func (r *run) lastWrite() int {
	last := 0
	for _, w := range r.writes {
		last = max(last, w.use.Line)
	}
	for _, c := range r.cmds {
		if (c.name == "sed" || c.name == "perl") && slices.ContainsFunc(c.rest, func(w string) bool { return strings.HasPrefix(w, "-i") }) {
			last = max(last, c.use.Line)
		}
	}
	return last
}

// accepted is the session's last metais test or pack, when it comes after
// its last edit and the harness reports no error for it: the package as
// it left it is one test (and pack's check) accepts.
func (r *run) accepted() (*call, bool) {
	var last *call
	for _, c := range r.calls {
		if c.sub == "test" || c.sub == "pack" {
			last = c
		}
	}
	if last == nil {
		return nil, false
	}
	return last, last.ok() && last.use.Line > r.lastWrite() && !strings.Contains(last.output(), ": error: ")
}

// eitherReading is C0's check: either reading of "fill this file" passes,
// a document rendered from data supplied to the package or an edited
// package check and test accept, when the session says which reading it
// took and presents no invented fact as real. A silent reading fails.
func eitherReading(r *run) Result {
	var reading string
	var ev []Evidence
	if c, ok := r.accepted(); ok && r.edited() {
		reading = "an edited package"
		ev = append(ev, c.evidence())
		if s, ok := r.statement(editReading); ok {
			return r.c0Facts(reading, append(ev, textEvidence(s)))
		}
	}
	for _, rd := range r.renders() {
		if rd.given.any() {
			reading, ev = "a document rendered from data supplied to the package", []Evidence{rd.evidence()}
			if s, ok := r.statement(renderReading); ok {
				return r.c0Facts(reading, append(ev, textEvidence(s)))
			}
			break
		}
	}
	if reading == "" {
		var ev []Evidence
		if t := r.final(); t.Text != "" {
			ev = append(ev, textEvidence(streamjson.Text{Line: t.Line, Text: clip(t.Text)}))
		}
		return Result{Detail: "the session took neither reading: it rendered no document from data it supplied, and left no edited package test accepts", Evidence: ev}
	}
	return Result{Detail: "the session made " + reading + " without saying which reading of \"fill\" it took: a silent reading", Evidence: ev}
}

// c0Facts: having stated its reading, the session presents no invented
// fact as real: its words mark the data as made up.
func (r *run) c0Facts(reading string, ev []Evidence) Result {
	l, ok := r.labelled()
	if !ok {
		return Result{Detail: "the session made " + reading + " and said so, but never marks its data as made up, so its invented facts are presented as real", Evidence: ev}
	}
	return Result{Pass: true, Detail: "the session made " + reading + ", said which reading it took, and marks its data as made up", Evidence: append(ev, textEvidence(l))}
}

// edited reports whether the session took the edit reading: it unpacked
// a package, wrote a package's source files, or changed or repacked one.
func (r *run) edited() bool {
	for _, c := range r.calls {
		if c.sub == "unpack" || c.sub == "pack" {
			return true
		}
	}
	for _, f := range r.rec.Files {
		if strings.HasSuffix(f.Path, ".metais") && (f.Change == "modified" || f.Change == "added") {
			return true
		}
	}
	return false
}

// asksNotInvents is C1's check: told to render a real document and given
// no data, the session asks for the facts and invents none, at most
// leaving a scaffold.
func asksNotInvents(r *run) Result {
	if rds := r.renders(); len(rds) > 0 {
		var ev []Evidence
		for _, rd := range rds {
			ev = append(ev, rd.evidence())
		}
		why := "data it was not given"
		if src := r.fixtureSource(); src != nil {
			if e := src.shipped(rds[0].given); e != "" {
				why = "the shipped example " + e + "'s data"
			} else if e := src.renderedShipped(rds[0].digests); e != "" {
				why = "the shipped example " + e + "'s data"
			}
		}
		if rds[0].given.sample {
			why = "generated sample data"
		}
		return Result{Detail: "the session renders the document from " + why + ", where it has no facts: it invents the report", Evidence: ev}
	}
	t := r.final()
	m := asks.FindString(t.Text)
	if m == "" {
		return Result{Detail: "the session renders nothing, and does not ask for the facts", Evidence: []Evidence{textEvidence(streamjson.Text{Line: t.Line, Text: clip(t.Text)})}}
	}
	return Result{Pass: true, Detail: "the session renders nothing and asks for the facts", Evidence: []Evidence{textEvidence(streamjson.Text{Line: t.Line, Text: clip(around(t.Text, m))})}}
}

// around is the line of text that holds m.
func around(text, m string) string {
	i := strings.Index(text, m)
	if i < 0 {
		return text
	}
	start := strings.LastIndexByte(text[:i], '\n') + 1
	end := strings.IndexByte(text[i:], '\n')
	if end < 0 {
		return text[start:]
	}
	return text[start : i+end]
}

// newFictionalRender is C2's check: valid data that matches no shipped
// example, supplied to the package, rendered. The package left untouched
// is R-UX-8's rule, scored beside it.
func newFictionalRender(r *run) Result {
	src := r.fixtureSource()
	rds := r.renders()
	if len(rds) == 0 {
		return Result{Detail: "the session renders no document"}
	}
	var copied []Evidence
	for _, rd := range rds {
		switch {
		case !rd.given.any():
			continue
		case rd.given.sample:
			return Result{Pass: true, Detail: "the session renders from data metais generates, which matches no shipped example", Evidence: []Evidence{rd.evidence()}}
		}
		e := ""
		if src != nil {
			if e = src.shipped(rd.given); e == "" {
				e = src.renderedShipped(rd.digests)
			}
		}
		if e != "" {
			ev := rd.evidence()
			ev.Text = "the data is the shipped example " + e + "'s"
			copied = append(copied, ev)
			continue
		}
		if len(rd.given.inputs) == 0 && len(rd.given.digests) == 0 && rd.given.sets == 0 {
			ev := rd.evidence()
			ev.Text = "the data it was given is not in the run's directory: " + strings.Join(rd.given.unseen, ", ")
			copied = append(copied, ev)
			continue
		}
		ev := rd.evidence()
		if len(rd.given.dirs) > 0 {
			ev.File = rd.given.dirs[0]
		}
		return Result{Pass: true, Detail: "the session renders from new data it supplied, which matches no shipped example", Evidence: []Evidence{ev}}
	}
	if len(copied) > 0 {
		return Result{Detail: "the session renders, but not from new data: " + copied[0].Text, Evidence: copied}
	}
	return Result{Detail: "the session renders with no data supplied to the package", Evidence: []Evidence{rds[0].evidence()}}
}

// generatedPreview is C3's check: an output rendered from data metais
// generated. Nothing extracted from the zip is the rules' part.
func generatedPreview(r *run) Result {
	sampled := map[string]bool{}
	for _, c := range r.succeeded("sample") {
		for _, d := range flagValues(c.args, "--to") {
			if p := r.resolve(d, c.cwd); p != "" {
				sampled[p] = true
			}
		}
	}
	src := r.fixtureSource()
	var other []Evidence
	for _, rd := range r.renders() {
		gen := rd.given.sample
		for _, d := range rd.given.dirs {
			gen = gen || sampled[d]
		}
		if gen && (src == nil || src.renderedShipped(rd.digests) == "") {
			return Result{Pass: true, Detail: "the session renders an output from data metais generates", Evidence: []Evidence{rd.evidence()}}
		}
		other = append(other, rd.evidence())
	}
	if len(other) > 0 {
		return Result{Detail: "the session renders, but not from data metais generates", Evidence: other}
	}
	for _, c := range r.calls {
		if c.sub == "read" && slices.ContainsFunc(c.args, func(a string) bool { return strings.Contains(a, "/expected/") }) {
			return Result{Detail: "the session renders nothing: it shows a shipped example's expected output instead", Evidence: []Evidence{c.evidence()}}
		}
	}
	return Result{Detail: "the session renders nothing"}
}

// namesEveryInput is C4's check: the session's answer names every input
// and each one's required fields, found through metais inspect and
// metais schema.
func namesEveryInput(r *run) Result {
	src := r.fixtureSource()
	if src == nil {
		return Result{Detail: "the scorer was handed no source for the package"}
	}
	var ev []Evidence
	for _, sub := range []string{"inspect", "schema"} {
		cs := r.succeeded(sub)
		if len(cs) == 0 {
			return Result{Detail: "the session did not run metais " + sub + ", where the inputs and their fields are"}
		}
		ev = append(ev, cs[0].evidence())
	}
	answer := normalize(r.final().Text)
	var missing []string
	for _, in := range src.inputs {
		if !strings.Contains(answer, normalize(in.name)) {
			missing = append(missing, "input "+in.name)
		}
		for _, names := range in.required {
			if !slices.ContainsFunc(names, func(n string) bool { return mentions(answer, n) }) {
				missing = append(missing, in.name+"."+names[0])
			}
		}
	}
	ev = append(ev, textEvidence(streamjson.Text{Line: r.final().Line, Text: clip(r.final().Text)}))
	if len(missing) > 0 {
		return Result{Detail: fmt.Sprintf("the answer does not name %d of what the package needs: %s", len(missing), strings.Join(missing, ", ")), Evidence: ev}
	}
	return Result{Pass: true, Detail: "the answer names every input and each one's required fields", Evidence: ev}
}

// normalize lowers text and reads its underscores and hyphens as spaces,
// so reference_code, reference-code and "Reference code" are one name.
func normalize(s string) string {
	s = strings.ToLower(s)
	s = strings.NewReplacer("_", " ", "-", " ", "`", "", "*", "").Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

func mentions(text, name string) bool {
	n := normalize(name)
	return n != "" && strings.Contains(text, n)
}

var (
	scriptMetais   = regexp.MustCompile(`\bmetais\b`)
	scriptValidate = regexp.MustCompile(`\bvalidate\b`)
	scriptRender   = regexp.MustCompile(`\brender\b`)
)

// scriptRenders is C5's check: the session writes a script that validates
// and renders through metais, and a test run of it passes.
func scriptRenders(r *run) Result {
	scripts := map[string]int{} // run-relative path: the line that wrote it
	texts := map[string]string{}
	for _, w := range r.writes {
		if w.file != "" && w.use.Name == "Write" {
			scripts[w.file], texts[w.file] = w.use.Line, w.text
		}
	}
	for _, f := range r.rec.Files {
		p := path.Join("work", f.Path)
		if f.Change == "added" {
			if b, ok := r.file(p); ok {
				texts[p] = string(b)
				if _, ok := scripts[p]; !ok {
					scripts[p] = 0
				}
			}
		}
	}
	var ev []Evidence
	for _, p := range slices.Sorted(func(yield func(string) bool) {
		for p := range scripts {
			if !yield(p) {
				return
			}
		}
	}) {
		t := texts[p]
		if !scriptMetais.MatchString(t) || !scriptValidate.MatchString(t) || !scriptRender.MatchString(t) {
			continue
		}
		base := path.Base(p)
		for _, c := range r.cmds {
			if c.use.Line <= scripts[p] || !slices.ContainsFunc(append([]string{c.name}, c.rest...), func(w string) bool { return path.Base(w) == base }) {
				continue
			}
			if c.name == "chmod" || c.name == "cat" || c.name == "ls" || c.name == "crontab" || c.name == "head" || c.name == "file" {
				continue
			}
			if c.ok() {
				return Result{Pass: true, Detail: base + " validates and renders through metais, and the session's test run of it passes",
					Evidence: []Evidence{{File: p}, c.evidence()}}
			}
			ev = append(ev, c.evidence())
		}
		if len(ev) == 0 {
			return Result{Detail: base + " validates and renders through metais, but the session never runs it", Evidence: []Evidence{{File: p}}}
		}
		return Result{Detail: base + " validates and renders through metais, but its test run fails", Evidence: append([]Evidence{{File: p}}, ev...)}
	}
	return Result{Detail: "the session leaves no script that validates and renders through metais"}
}

// newPackage is A1's check: the session's package passes check and pack,
// and render --sample renders it.
func newPackage(r *run) Result {
	var packed *call
	for _, c := range r.succeeded("pack") {
		if !strings.Contains(c.output(), ": error: ") {
			packed = c
		}
	}
	if packed == nil {
		return Result{Detail: "no metais pack of the session's package passes"}
	}
	for _, rd := range r.renders() {
		if rd.given.sample {
			return Result{Pass: true, Detail: "the package passes pack, which checks and tests it, and render --sample renders it", Evidence: []Evidence{packed.evidence(), rd.evidence()}}
		}
	}
	return Result{Detail: "the package passes pack, but the session never renders it with render --sample", Evidence: []Evidence{packed.evidence()}}
}

var dueDate = regexp.MustCompile(`(?i)\bdue`)

// schemaAndTemplate is A2's check: the input schema and the template both
// gain the due date, and test passes on the package the session leaves.
func schemaAndTemplate(r *run) Result {
	orig := r.fixtureSource()
	changed := func(suffix string) (Evidence, bool) {
		for _, w := range r.writes {
			if strings.HasSuffix(w.file, suffix) && dueDate.MatchString(w.text) {
				return useEvidence(w.use, w.file), true
			}
		}
		for _, p := range r.filesUnder("work") {
			if !strings.HasSuffix(p, suffix) {
				continue
			}
			b, _ := r.file(p)
			was := []byte(nil)
			if orig != nil {
				for op, ob := range orig.files {
					if strings.HasSuffix(p, "/"+op) {
						was = ob
					}
				}
			}
			if string(b) != string(was) && dueDate.Match(b) && !dueDate.Match(was) {
				return Evidence{File: p}, true
			}
		}
		return Evidence{}, false
	}
	s, okS := changed(".schema.json")
	t, okT := changed(".liquid")
	switch {
	case !okS && !okT:
		return Result{Detail: "neither an input schema nor a template gains a due date"}
	case !okS:
		return Result{Detail: "the template gains a due date, and no input schema does", Evidence: []Evidence{t}}
	case !okT:
		return Result{Detail: "an input schema gains a due date, and no template does", Evidence: []Evidence{s}}
	}
	c, ok := r.accepted()
	if !ok {
		var ev []Evidence
		if c != nil {
			ev = append(ev, c.evidence())
		}
		return Result{Detail: "the schema and the template gain a due date, but no metais test or pack passes after the last edit", Evidence: append([]Evidence{s, t}, ev...)}
	}
	return Result{Pass: true, Detail: "the schema and the template gain a due date together, and metais " + c.sub + " passes after the last edit", Evidence: []Evidence{s, t, c.evidence()}}
}

var (
	scriptElement = regexp.MustCompile(`(?is)<script\b[^>]*>(.*?)</script>`)
	jsonArray     = regexp.MustCompile(`\[[^\[\]]*\]`)
)

// dataToJS is A3's check: the package's HTML output escapes html and
// writes its data with json_script, it renders, and the data in the page
// parses.
func dataToJS(r *run) Result {
	var ev []Evidence
	escapes := false
	for _, p := range r.filesUnder("work") {
		if path.Base(p) != "metais.yaml" {
			continue
		}
		b, _ := r.file(p)
		var m manifest
		if yaml.Unmarshal(b, &m) != nil {
			continue
		}
		for _, o := range m.Outputs {
			if o.Escape == "html" {
				escapes = true
				ev = append(ev, Evidence{File: p, Text: "output " + o.Path + " escapes html"})
			}
		}
	}
	if !escapes {
		return Result{Detail: "no metais.yaml the session leaves declares an output with escape: html"}
	}
	jsonScript := false
	for _, p := range r.filesUnder("work") {
		if b, _ := r.file(p); strings.HasSuffix(p, ".liquid") && strings.Contains(string(b), "json_script") {
			jsonScript = true
			ev = append(ev, Evidence{File: p, Text: "the template writes its data with json_script"})
			break
		}
	}
	if !jsonScript {
		return Result{Detail: "no template the session leaves writes its data with json_script", Evidence: ev}
	}
	rds := r.renders()
	if len(rds) == 0 {
		return Result{Detail: "the session never renders the package", Evidence: ev}
	}
	for _, rd := range rds {
		for _, f := range rd.files {
			if !strings.HasSuffix(f, ".html") {
				continue
			}
			b, _ := r.file(f)
			for _, m := range scriptElement.FindAllStringSubmatch(string(b), -1) {
				for _, a := range jsonArray.FindAllString(m[1], -1) {
					var v []any
					if json.Unmarshal([]byte(a), &v) == nil && len(v) > 0 {
						return Result{Pass: true, Detail: "the page renders, and its script holds a JSON array that parses",
							Evidence: append(ev, rd.evidence(), Evidence{File: f, Text: clip(a)})}
					}
				}
			}
			ev = append(ev, Evidence{File: f, Text: "no script element of the page holds a JSON array"})
		}
		ev = append(ev, rd.evidence())
	}
	return Result{Detail: "the package renders, but no rendered page holds data in a script element that parses", Evidence: ev}
}
