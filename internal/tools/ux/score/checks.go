// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"encoding/json"
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"metais.org/metais/internal/tools/ux/streamjson"
)

// check is one scenario's outcome check (plan §5's table), and what it
// cannot see, which a scorecard states beside its results.
type check struct {
	run    func(r *run) Result
	limits string
	// judged marks a check that reads the session's words: a recorded
	// verdict, when the run carries one, decides it.
	judged bool
}

// checks are the outcome checks, by the name a scenario's declaration
// gives as its check.
var checks = map[string]check{
	"either-reading": {eitherReading,
		"Whether the session said which reading it took, and whether it presented invented facts as real, is decided from its words by phrase: a statement of its reading (\"I took this to mean…\", \"I interpreted…\", \"assuming you meant…\") naming what it did, and a word that marks its data as made up (fictional, sample, placeholder…). A statement worded outside those phrases is not seen, and a fact the session invented is not seen if its words call the data fictional.", true},
	"asks-not-invents": {asksNotInvents,
		"Invention is seen as a rendered document: a session that renders nothing passes however it fills a data file it leaves behind, which the check counts as a scaffold. Asking is seen as a question mark or a request (\"please provide\", \"send me\", \"let me know\") in the session's last words.", true},
	"new-fictional-render": {newFictionalRender,
		"New data is data that is not a shipped example's: equal to one, input by input, or rendering one of its expected outputs byte for byte. A copy with a value changed is new.", false},
	"generated-preview": {generatedPreview,
		"Generated data is metais's: render --sample, or a data directory metais sample wrote. Data the session wrote by hand is not generated.", false},
	"names-every-input": {namesEveryInput,
		"An input or a required field is named when its name, spelt with underscores, spaces or hyphens, or its schema title appears in the session's last words. A field is also named by a line of them that holds every word of its name, each read by its first six letters or by a word of the line, plural or not, of five letters or more that is its end, so \"Primary and second contact: name, email and telephone\" names six fields, \"detection\" names detected_at and \"phone\" names telephone; or that holds the number its title begins with, alone or inside a range such as 1.7–1.9, beside a word of that title. A line that holds a field's words for another reason is read as naming it, as is one whose word ends a word of the name by chance (\"count\" ends account), and a field named only by a synonym is not seen. The required fields are each input schema's top-level required list; a field the stage makes mandatory through a condition is not counted.", false},
	"script-renders": {scriptRenders,
		"A script validates and renders through metais when its text names metais and render: metais render validates the inputs as metais validate does and refuses on any error, so a separate validate is not required. Its test run passes when the session runs it, the harness reports no error, and the run renders: render's listing of the files it wrote is in its output, or a file named as one of the package's outputs is left in the session's directory. The harness reports one status for a Bash call, its last command's, so a run of the script that other commands of its call follow passes on render's listing in the call's output alone, whatever the call's status: a script that renders and then fails in a later step of its own is not told apart there from one that ends cleanly. The scorer does not run the script again.", false},
	"new-package": {newPackage,
		"check passing is seen through pack, which checks and tests the package before it writes it. A render --sample is seen when it lists the files it wrote or leaves them in the run's directory; when its listing went elsewhere, such as to /dev/null, and its files are gone, it is seen when a later command of its own Bash call that prints a file (cat, head, sed, grep…) names a file beneath its output directory and the call's output holds no refusal and no missing file. A file its output directory held before it rendered is read there as its own. A for loop's body is read once for each word the loop names, its variable expanded; a loop over a command substitution or another variable is read once, its variable unexpanded.", false},
	"checks-tests-packs": {checksTestsPacks,
		"A command passes by its own report: check's 0 errors, test's pass with no example failing, pack's Packed line. Whether the package holds every part the task asks for, whether it is the worked package with its names changed, and what the session read are not checked: SC9's round judges those from the package and the transcript.", false},
	"schema-and-template": {schemaAndTemplate,
		"A change is seen in the files the session left, compared with the package's source, or in what its Write and Edit calls wrote: a schema and a template that each gain a due date. A change made by a command line whose files are gone is not seen.", false},
	"data-to-js": {dataToJS,
		"The page's data parses when a script element of a rendered HTML output holds a JSON array: a page the render wrote that the run leaves, or a file the run leaves with the bytes the render's listing reports writing, such as an example's expected output metais test --update wrote from the same data. A rendered page the session deleted with no copy of its bytes left is not read, even when the call's output shows its script. Whether the array is the numbers given is not checked. A command's directory follows the session's cd, $PWD read as the directory it runs in, and a cd carries to the session's next call only when its call exits 0, as the harness records the directory then alone; within a call, a cd that failed is still followed.", false},
}

// CheckLimits is what an outcome check, or a pass rule, cannot see, as a
// scorecard states it: "" for a rule that states nothing.
func CheckLimits(name string) string {
	if c, ok := checks[name]; ok {
		return c.limits
	}
	for _, ru := range rules {
		if ru.name == name {
			return ru.limits
		}
	}
	return ""
}

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
	editReading   = regexp.MustCompile(`(?i)\b(?:edit|rewr(?:ite|ote|itten)|repack|unpack|updat|chang|modif|regenerat|replac|the\s+package\s+itself|its\s+examples?\b)`)
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

// sourceDirs are the package source directories the session worked in,
// relative to the run's directory: what it unpacked to or made with init,
// and what it checked, tested or packed.
func (r *run) sourceDirs() []string {
	var dirs []string
	add := func(p, cwd string) {
		if d := r.resolve(p, cwd); d != "" && !strings.HasSuffix(d, ".metais") && !slices.Contains(dirs, d) {
			dirs = append(dirs, d)
		}
	}
	for _, c := range r.calls {
		switch c.sub {
		case "unpack":
			for _, d := range outDirs(c.args) {
				add(d, c.cwd)
			}
		case "init", "check", "test", "pack":
			if op := operand(c.args); op != "" {
				add(op, c.cwd)
			} else if c.sub != "init" {
				add(".", c.cwd)
			}
		}
	}
	return dirs
}

// outDirs are the directories a command line names for unpack, extract
// or sample to write: --out, which every command that writes a
// directory takes (ADR-008 A4), and --to, which those three took
// when the baseline's runs were recorded.
func outDirs(args []string) []string {
	return append(flagValues(args, "--out"), flagValues(args, "--to")...)
}

// operand is a command's first word that is not a flag, or "".
func operand(args []string) string {
	for _, a := range args {
		if !strings.HasPrefix(a, "-") {
			return a
		}
	}
	return ""
}

// inSource reports whether a file, relative to the run's directory, is in
// one of dirs, or in any directory when dirs is empty.
func inSource(file string, dirs []string) bool {
	if len(dirs) == 0 {
		return true
	}
	for _, d := range dirs {
		if d == "." || file == d || strings.HasPrefix(file, d+"/") {
			return true
		}
	}
	return false
}

// lastWrite is the place, among the session's commands and writes, of
// its last Write or Edit of a file of the package's source, or of its
// last command that edits one in place, or -1.
func (r *run) lastWrite() int {
	dirs := r.sourceDirs()
	last := -1
	for _, w := range r.writes {
		if inSource(w.file, dirs) {
			last = max(last, w.seq)
		}
	}
	for _, c := range r.cmds {
		if (c.name == "sed" || c.name == "perl") && slices.ContainsFunc(c.rest, func(w string) bool { return strings.HasPrefix(w, "-i") }) {
			for _, w := range c.rest {
				if !strings.HasPrefix(w, "-") && inSource(r.resolve(w, c.cwd), dirs) {
					last = max(last, c.seq)
				}
			}
		}
	}
	return last
}

// Each command's own report that it passed. A Bash call's output is all
// its commands' together, so a call is judged by its own lines, not by
// whether some other command of the call reported an error.
var (
	packed     = regexp.MustCompile(`(?m)^Packed .*: \d+ bytes, \d+ entries|"format":"metais\.pack/1","ok":true`)
	testPassed = regexp.MustCompile(`(?m)^[^\s:]+: pass$|"format":"metais\.test/1","ok":true`)
	testFailed = regexp.MustCompile(`(?m)^[^\s:]+: fail\b|"format":"metais\.test/1","ok":false`)
	checked    = regexp.MustCompile(`(?m)^check: 0 errors|"format":"metais\.check/1","ok":true`)
)

// passed reports whether a metais test, pack or check reports it passed.
func passed(c *call) bool {
	out := c.output()
	switch c.sub {
	case "pack":
		return packed.MatchString(out)
	case "test":
		return testPassed.MatchString(out) && !testFailed.MatchString(out)
	case "check":
		return checked.MatchString(out)
	}
	return c.ok()
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
	return last, passed(last) && last.seq > r.lastWrite()
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
		for _, d := range outDirs(c.args) {
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
	answer := normalize(r.final().Text)
	lines := answerLines(r.final().Text)
	var missing []string
	for _, in := range src.inputs {
		if !strings.Contains(answer, normalize(in.name)) {
			missing = append(missing, "input "+in.name)
		}
		for _, names := range in.required {
			if !slices.ContainsFunc(names, func(n string) bool { return mentions(answer, n) }) && !slices.ContainsFunc(lines, func(l answerLine) bool { return l.names(names) }) {
				missing = append(missing, in.name+"."+names[0])
			}
		}
	}
	var ev []Evidence
	for _, sub := range []string{"inspect", "schema"} {
		cs := r.succeeded(sub)
		if len(cs) == 0 {
			return Result{Detail: fmt.Sprintf("the session did not run metais %s, where the inputs and their fields are; its answer leaves %d of them unnamed", sub, len(missing))}
		}
		ev = append(ev, cs[0].evidence())
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

// answerLine is one line of an answer, read for the fields it names in
// its own words: the stems of its words, and the field numbers it holds.
type answerLine struct {
	stems   map[string]bool
	words   []string
	numbers []string
	ranges  [][2]string
}

var (
	answerWord = regexp.MustCompile(`[a-z]+`)
	// fieldNumber is a numbered field, such as 1.7 or 3.15: a title that
	// begins with one, and an answer that names it so.
	fieldNumber = regexp.MustCompile(`\d+(?:\.\d+)+`)
	// fieldRange is a run of numbered fields: 1.7–1.9, 1.10-1.12, 2.1 to 2.4.
	fieldRange = regexp.MustCompile(`(\d+(?:\.\d+)+)\s*(?:–|—|-|to)\s*(\d+(?:\.\d+)+)`)
	// joining are the words a field's name joins its words with, which a
	// line need not repeat: detected_at is named by "detection".
	joining = []string{"a", "an", "and", "at", "by", "for", "in", "is", "of", "on", "or", "the", "to"}
)

// stem is the part of a word a line and a field's name are compared by:
// its first six letters, or a shorter word without its plural s, so that
// detected and detection, and name and names, are one word.
func stem(w string) string {
	if len(w) > 6 {
		return w[:6]
	}
	if len(w) > 3 {
		return strings.TrimSuffix(w, "s")
	}
	return w
}

// answerLines reads each line of an answer.
func answerLines(text string) []answerLine {
	var out []answerLine
	for _, l := range strings.Split(strings.ToLower(text), "\n") {
		al := answerLine{stems: map[string]bool{}, numbers: fieldNumber.FindAllString(l, -1)}
		for _, w := range answerWord.FindAllString(l, -1) {
			al.stems[stem(w)] = true
			al.words = append(al.words, w)
		}
		for _, m := range fieldRange.FindAllStringSubmatch(l, -1) {
			al.ranges = append(al.ranges, [2]string{m[1], m[2]})
		}
		out = append(out, al)
	}
	return out
}

// names reports whether the line names a field, given its property name
// and title: every word of its name is in the line, by its stem, or the
// line holds the number its title begins with, alone or inside a range,
// beside a word of that title.
func (l answerLine) names(names []string) bool {
	if l.holdsAll(strings.ReplaceAll(names[0], "_", " ")) {
		return true
	}
	if len(names) < 2 {
		return false
	}
	num := fieldNumber.FindString(names[1])
	if num == "" || !strings.HasPrefix(names[1], num) || !l.holdsAny(strings.TrimPrefix(names[1], num)) {
		return false
	}
	if slices.Contains(l.numbers, num) {
		return true
	}
	return slices.ContainsFunc(l.ranges, func(rg [2]string) bool { return inRange(num, rg[0], rg[1]) })
}

// words are the words of a name or title the line is read for, without
// the joining words.
func words(text string) []string {
	return slices.DeleteFunc(answerWord.FindAllString(strings.ToLower(text), -1), func(w string) bool { return slices.Contains(joining, w) })
}

// holdsAll reports whether the line holds every word of text, by its
// stem or by its end.
func (l answerLine) holdsAll(text string) bool {
	ws := words(text)
	return len(ws) > 0 && !slices.ContainsFunc(ws, func(w string) bool { return !l.stems[stem(w)] && !l.ends(w) })
}

// ends reports whether a word of the line of five letters or more, or
// that word without its plural s, is the end of w, as phone and phones
// are of telephone: the short form a person writes for a word.
func (l answerLine) ends(w string) bool {
	return slices.ContainsFunc(l.words, func(lw string) bool {
		for _, f := range []string{lw, strings.TrimSuffix(lw, "s")} {
			if len(f) >= 5 && len(f) < len(w) && strings.HasSuffix(w, f) {
				return true
			}
		}
		return false
	})
}

// holdsAny reports whether the line holds a word of text, by its stem.
func (l answerLine) holdsAny(text string) bool {
	return slices.ContainsFunc(words(text), func(w string) bool { return l.stems[stem(w)] })
}

// inRange reports whether a field number falls in a range of them that
// differ only in their last part: 1.8 is in 1.7–1.9.
func inRange(num, lo, hi string) bool {
	cut := func(s string) (string, int) {
		i := strings.LastIndexByte(s, '.')
		n, err := strconv.Atoi(s[i+1:])
		if err != nil {
			return "", -1
		}
		return s[:i], n
	}
	p, n := cut(num)
	lp, l := cut(lo)
	hp, h := cut(hi)
	return n >= 0 && p == lp && p == hp && l <= n && n <= h
}

var (
	scriptMetais = regexp.MustCompile(`\bmetais\b`)
	scriptRender = regexp.MustCompile(`\brender\b`)
)

// scriptRenders is C5's check: the session writes a script that validates
// and renders through metais, and a test run of it passes. metais render
// validates its inputs as metais validate does, and renders nothing when
// any is in error, so a script that renders through metais validates
// through it.
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
		if !scriptMetais.MatchString(t) || !scriptRender.MatchString(t) {
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
			// The harness reports one status for a Bash call, its last
			// command's. A run of the script that other commands follow
			// is judged by render's own listing in the call's output.
			if c.ok() && r.rendered(c) || !c.ok() && !r.lastOfCall(c) && reportedDigest.MatchString(c.output()) {
				return Result{Pass: true, Detail: base + " validates and renders through metais, and the session's test run of it renders",
					Evidence: []Evidence{{File: p}, c.evidence()}}
			}
			ev = append(ev, c.evidence())
		}
		if len(ev) == 0 {
			return Result{Detail: base + " validates and renders through metais, but the session never runs it", Evidence: []Evidence{{File: p}}}
		}
		return Result{Detail: base + " validates and renders through metais, but no test run of it renders", Evidence: append([]Evidence{{File: p}}, ev...)}
	}
	return Result{Detail: "the session leaves no script that validates and renders through metais"}
}

// lastOfCall reports whether c is the last command of its Bash call.
func (r *run) lastOfCall(c *cmd) bool {
	return !slices.ContainsFunc(r.cmds, func(o *cmd) bool { return o.use == c.use && o.seq > c.seq })
}

// rendered reports whether a command rendered the package: its output
// lists a file render wrote, or the session's directory holds, added, a
// file named as one of the package's outputs.
func (r *run) rendered(c *cmd) bool {
	if reportedDigest.MatchString(c.output()) {
		return true
	}
	src := r.fixtureSource()
	if src == nil {
		return false
	}
	names := map[string]bool{}
	for p := range src.files {
		if strings.HasPrefix(p, "examples/") && strings.Contains(p, "/expected/") {
			names[path.Base(p)] = true
		}
	}
	for _, f := range r.rec.Files {
		if f.Change == "added" && names[path.Base(f.Path)] {
			return true
		}
	}
	return false
}

// newPackage is A1's check: the session's package passes check and pack,
// and render --sample renders it.
func newPackage(r *run) Result {
	var pack *call
	for _, c := range r.calls {
		if c.sub == "pack" && passed(c) {
			pack = c
		}
	}
	if pack == nil {
		return Result{Detail: "no metais pack of the session's package passes"}
	}
	for _, rd := range r.renders() {
		if rd.given.sample {
			return Result{Pass: true, Detail: "the package passes pack, which checks and tests it, and render --sample renders it", Evidence: []Evidence{pack.evidence(), rd.evidence()}}
		}
	}
	return Result{Detail: "the package passes pack, but the session never renders it with render --sample", Evidence: []Evidence{pack.evidence()}}
}

// checksTestsPacks is A4's check, SC9's bar as a run shows it: the
// session's last metais check, its last metais test and its last metais
// pack each pass, and each comes after its last edit of the package's
// source.
func checksTestsPacks(r *run) Result {
	edit := r.lastWrite()
	var ev []Evidence
	for _, sub := range []string{"check", "test", "pack"} {
		var last *call
		for _, c := range r.calls {
			if c.sub == sub {
				last = c
			}
		}
		switch {
		case last == nil:
			return Result{Detail: "the session never runs metais " + sub, Evidence: ev}
		case !passed(last):
			return Result{Detail: "the session's last metais " + sub + " does not pass", Evidence: append(ev, last.evidence())}
		case last.seq < edit:
			return Result{Detail: "the session edits its package after its last metais " + sub, Evidence: append(ev, last.evidence())}
		}
		ev = append(ev, last.evidence())
	}
	return Result{Pass: true, Detail: "the session's last metais check, test and pack each pass, after its last edit of the package", Evidence: ev}
}

var dueDate = regexp.MustCompile(`(?i)\bdue`)

// schemaAndTemplate is A2's check: the input schema and the template both
// gain the due date, and test passes on the package the session leaves.
func schemaAndTemplate(r *run) Result {
	orig := r.fixtureSource()
	changed := func(suffix string) (Evidence, bool) {
		// A session may edit a source outside its own directory, so a
		// write is read by the path it names.
		for _, w := range r.writes {
			if strings.HasSuffix(w.use.Field("file_path"), suffix) && dueDate.MatchString(w.text) {
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
		m, ok := readManifest(b)
		if !ok {
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
		// A file the run leaves with bytes the render reported writing is
		// the rendered page, wherever the session's later commands put it.
		files := slices.Clone(rd.files)
		for _, f := range r.rec.Files {
			if p := path.Join("work", f.Path); f.Change != "deleted" && slices.Contains(rd.digests, f.SHA256) && !slices.Contains(files, p) {
				files = append(files, p)
			}
		}
		for _, f := range files {
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
