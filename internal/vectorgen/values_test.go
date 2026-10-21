// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package vectorgen

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
	"github.com/kottera/metais/internal/zipfmt"
)

// This file writes every number of a vector's expectations that the bytes
// of its package fix: a finding's offset into the package, a size or a
// length it reports, and a limit a case lowers to the package's own size
// or one below it. The rest of each file is hand-written, and stays so: a
// vector's codes, locations, severities and the values its case chose are
// what a reviewer reads and what the specification's text decides. Only
// the numbers that follow from where vectorgen placed each record are
// vectorgen's, so a change to the README, or to any bytes ahead of an
// entry, is regenerated rather than edited across the suite:
//
//	go test ./internal/vectorgen -run TestValues -args -update
//
// Each number is derived from the layout archive.write recorded, never by
// reading the package back: a hostile package is the input under test.

// tmpl is the base package's template, the entry most cases break.
const tmpl = "templates/report.md.liquid"

// innerPrefix is what pkg/second-end-record's inner archive begins with,
// ahead of the end record's signature it hides.
const innerPrefix = "an inner archive: "

// zip64RecordLen is the length of the zip64 end of central directory
// record zip64End writes, ahead of its locator.
const zip64RecordLen = 56

// value is one number in a vector's JSON that vectorgen writes.
type value struct {
	file string // the file, from the vector's directory
	ptr  string // the number's JSON Pointer within it
	of   func(b built) int
}

// param is the parameter name of the i-th finding of
// expected/findings.json.
func param(i int, name string, of func(built) int) value {
	return value{"expected/findings.json", fmt.Sprintf("/%d/params/%s", i, name), of}
}

// limit is the limit name the first step of the case c of vector.json
// lowers.
func limit(c int, name string, of func(built) int) value {
	return value{"vector.json", fmt.Sprintf("/cases/%d/steps/0/args/limits/%s", c, name), of}
}

// The derivations: each is where vectorgen put something, or a size it
// wrote.
func localHeader(name string) func(built) int { return localField(name, 0) }
func localField(name string, f int) func(built) int {
	return func(b built) int { return b.at.entry(name).local + f }
}
func fileData(name string) func(built) int { return func(b built) int { return b.at.entry(name).data } }

// dataEnd is the end of the file data of the entry name.
func dataEnd(name string) func(built) int {
	return func(b built) int { e := b.at.entry(name); return e.data + int(e.lh.csize) }
}
func centralHeader(name string) func(built) int { return centralField(name, 0) }
func centralField(name string, f int) func(built) int {
	return func(b built) int { return b.at.entry(name).central + f }
}

// centralExtra is the extra field of the central directory header of the
// entry name.
func centralExtra(name string) func(built) int {
	return func(b built) int { e := b.at.entry(name); return e.central + cfName + int(e.ch.nameLen) }
}

// central is a field of the central directory header of the entry name,
// as written.
func central(name string, f func(h header) int) func(built) int {
	return func(b built) int { return f(b.at.entry(name).ch) }
}
func declaredSize(h header) int  { return int(h.usize) }
func extraLength(h header) int   { return int(h.extraLen) }
func commentLength(h header) int { return int(h.commentLen) }

// endRecord is the end record, and endField one of its fields.
var endRecord = endField(0)

func endField(f int) func(built) int { return func(b built) int { return b.at.end + f } }

// start is the package's first byte, where a prefix ahead of its first
// entry begins.
func start(built) int { return 0 }

// afterEnd is the first byte after the end record and its comment.
func afterEnd(b built) int       { return b.at.suffix }
func packageSize(b built) int    { return b.at.size }
func entryCount(b built) int     { return len(b.at.entries) }
func archiveComment(b built) int { return int(b.at.eocd.commentLen) }
func onDisk(b built) int         { return int(b.at.eocd.onDisk) }

// uncompressed is the sum of every entry's uncompressed size, as its
// central directory header declares it (chapter 14 §14.2).
func uncompressed(b built) int {
	n := 0
	for _, e := range b.at.entries {
		n += int(e.ch.usize)
	}
	return n
}

// ratioBound is the lowest entry-ratio the entry name is within: its
// uncompressed size divided by its compressed size, rounded up.
func ratioBound(name string) func(built) int {
	return func(b built) int { h := b.at.entry(name).ch; return int((h.usize + h.csize - 1) / h.csize) }
}

// ratio is the entry name's ratio as a finding reports it, rounded down.
func ratio(name string) func(built) int {
	return func(b built) int { h := b.at.entry(name).ch; return int(h.usize / h.csize) }
}

// below is one less than of: the bound a package at of is one past.
func below(of func(built) int) func(built) int { return func(b built) int { return of(b) - 1 } }

// packageValues is every vector's generated numbers, by vector.
var packageValues = map[string][]value{
	"pkg/archive-comment":            {param(0, "offset", endRecord), param(0, "length", archiveComment)},
	"pkg/central-order":              {param(0, "offset", localHeader("metais.yaml")), param(0, "declared", localHeader("inputs/report.schema.json")), param(1, "offset", localHeader(tmpl)), param(1, "declared", localHeader("metais.yaml")), param(2, "offset", localHeader("inputs/report.schema.json")), param(2, "declared", localHeader(tmpl))},
	"pkg/count-disk":                 {param(0, "offset", endField(efOnDisk)), param(0, "value", onDisk), param(0, "expected", entryCount)},
	"pkg/count-fewer":                {param(0, "offset", centralHeader(tmpl))},
	"pkg/count-more":                 {param(0, "offset", endRecord)},
	"pkg/crc-mismatch":               {param(0, "offset", fileData(tmpl))},
	"pkg/crc-wrong":                  {param(0, "offset", fileData(tmpl))},
	"pkg/data-descriptor":            {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/deflate-stream":             {param(0, "offset", fileData(tmpl))},
	"pkg/deflate-truncated":          {param(0, "offset", fileData(tmpl))},
	"pkg/directory-entry-data":       {param(0, "offset", centralField("assets/dir/", cfExternal)), param(1, "offset", centralHeader("assets/dir/"))},
	"pkg/directory-entry-mode":       {param(0, "offset", centralField("assets/dir/", cfExternal)), param(1, "offset", centralHeader("assets/dir/"))},
	"pkg/disk-number-start":          {param(0, "offset", centralField(tmpl, cfDisk))},
	"pkg/encrypted":                  {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/end-disk-number":            {param(0, "offset", endField(efCDDisk)), param(1, "offset", endField(efDisk))},
	"pkg/entry-order":                {param(0, "offset", centralHeader("inputs/report.schema.json"))},
	"pkg/external-attributes":        {param(0, "offset", centralField(tmpl, cfExternal))},
	"pkg/extra-field":                {param(0, "offset", centralHeader(tmpl)), param(0, "length", central(tmpl, extraLength))},
	"pkg/extra-field-unicode-path":   {param(0, "offset", centralHeader(tmpl)), param(0, "length", central(tmpl, extraLength))},
	"pkg/field-value":                {param(0, "offset", centralField(tmpl, cfExternal))},
	"pkg/file-comment":               {param(0, "offset", centralField(tmpl, cfCommentLen)), param(0, "value", central(tmpl, commentLength))},
	"pkg/flag-bit-13":                {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/flag-bit-6":                 {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/flag-reserved":              {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/flag-utf8-clear":            {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/flags":                      {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/gap":                        {param(0, "offset", dataEnd("inputs/report.schema.json")), param(0, "declared", localHeader("metais.yaml"))},
	"pkg/header-mismatch":            {param(0, "offset", localField(tmpl, lfCRC))},
	"pkg/internal-attributes":        {param(0, "offset", centralField(tmpl, cfInternal))},
	"pkg/local-compressed-size":      {param(0, "offset", localField(tmpl, lfCSize))},
	"pkg/local-flags":                {param(0, "offset", localField(tmpl, lfFlags))},
	"pkg/local-method":               {param(0, "offset", localField(tmpl, lfMethod))},
	"pkg/local-name":                 {param(0, "offset", localField(tmpl, lfName))},
	"pkg/local-uncompressed-size":    {param(0, "offset", localField(tmpl, lfUSize))},
	"pkg/method":                     {param(0, "offset", centralField(tmpl, cfMethod))},
	"pkg/method-aes":                 {param(0, "offset", centralField(tmpl, cfFlags))},
	"pkg/method-deflate64":           {param(0, "offset", centralField(tmpl, cfMethod))},
	"pkg/method-lzma":                {param(0, "offset", centralField(tmpl, cfMethod))},
	"pkg/method-zstd":                {param(0, "offset", centralField(tmpl, cfMethod))},
	"pkg/mod-date":                   {param(0, "offset", centralField(tmpl, cfDate))},
	"pkg/mod-time":                   {param(0, "offset", centralField(tmpl, cfTime))},
	"pkg/mode-block-device":          {param(0, "offset", centralField("assets/disk", cfExternal))},
	"pkg/mode-character-device":      {param(0, "offset", centralField("assets/tty", cfExternal))},
	"pkg/mode-executable":            {param(0, "offset", centralField("assets/run", cfExternal))},
	"pkg/mode-fifo":                  {param(0, "offset", centralField("assets/fifo", cfExternal))},
	"pkg/mode-setuid":                {param(0, "offset", centralField("assets/run", cfExternal))},
	"pkg/mode-socket":                {param(0, "offset", centralField("assets/socket", cfExternal))},
	"pkg/name-absolute":              {param(0, "offset", centralHeader("/etc/x"))},
	"pkg/name-asterisk":              {param(0, "offset", centralHeader("assets/a*b"))},
	"pkg/name-backslash-parent":      {param(0, "offset", centralHeader("a\\..\\..\\x"))},
	"pkg/name-case-collision":        {param(0, "offset", centralHeader("assets/straße.txt"))},
	"pkg/name-case-collision-ascii":  {param(0, "offset", centralHeader("assets/a.txt"))},
	"pkg/name-case-collision-prefix": {param(0, "offset", centralHeader("assets/docs/b"))},
	"pkg/name-character":             {param(0, "offset", centralHeader("assets/a\u202etxt.exe"))},
	"pkg/name-control":               {param(0, "offset", centralHeader("assets/a\x01b"))},
	"pkg/name-directory-entry":       {param(0, "offset", centralHeader("assets/dir/"))},
	"pkg/name-dot":                   {param(0, "offset", centralHeader("assets/./x"))},
	"pkg/name-drive-backslash":       {param(0, "offset", centralHeader(`C:\x`)), param(1, "offset", centralHeader(`C:\x`))},
	"pkg/name-drive-relative":        {param(0, "offset", centralHeader("C:x"))},
	"pkg/name-duplicate":             {param(0, "offset", func(b built) int { return b.at.nth("assets/a.txt", 1).central })},
	"pkg/name-empty":                 {param(0, "offset", centralHeader(""))},
	"pkg/name-empty-segment":         {param(0, "offset", centralHeader("assets//b"))},
	"pkg/name-encoding":              {param(0, "offset", centralHeader("assets/a\xff.txt"))},
	"pkg/name-file-directory":        {param(0, "offset", centralHeader("assets/a/b"))},
	"pkg/name-git-joiner":            {param(0, "offset", centralHeader("assets/.g\u200cit/config")), param(1, "offset", centralHeader("assets/.g\u200cit/config"))},
	"pkg/name-git-short":             {param(0, "offset", centralHeader("assets/git~1/config"))},
	"pkg/name-git-upper":             {param(0, "offset", centralHeader("assets/.GIT/config"))},
	"pkg/name-kelvin":                {param(0, "offset", centralHeader("assets/K.txt"))},
	"pkg/name-not-nfc":               {param(0, "offset", centralHeader("assets/café.txt"))},
	"pkg/name-not-portable":          {param(0, "offset", centralHeader("assets/.git/config"))},
	"pkg/name-nul":                   {param(0, "offset", centralHeader("assets/a\x00b"))},
	"pkg/name-parent":                {param(0, "offset", centralHeader("../x"))},
	"pkg/name-parent-nested":         {param(0, "offset", centralHeader("a/../../x"))},
	"pkg/name-private-use":           {param(0, "offset", centralHeader("assets/a\ue000b"))},
	"pkg/name-reserved":              {param(0, "offset", centralHeader("assets/CON"))},
	"pkg/name-reserved-extension":    {param(0, "offset", centralHeader("assets/nul.txt"))},
	"pkg/name-reserved-port":         {param(0, "offset", centralHeader("assets/com1"))},
	"pkg/name-reserved-spaced":       {param(0, "offset", centralHeader("assets/Con .md"))},
	"pkg/name-reserved-superscript":  {param(0, "offset", centralHeader("assets/COM¹.txt"))},
	"pkg/name-segment-length":        {param(0, "offset", centralHeader("assets/"+strings.Repeat("s", 256)))},
	"pkg/name-shape":                 {param(0, "offset", centralHeader("assets/../evil.txt"))},
	"pkg/name-stream":                {param(0, "offset", centralHeader("assets/a:b"))},
	"pkg/name-trailing-dot":          {param(0, "offset", centralHeader("assets/name."))},
	"pkg/name-trailing-space":        {param(0, "offset", centralHeader("assets/name "))},
	"pkg/name-unassigned":            {param(0, "offset", centralHeader("assets/a\u0378b"))},
	"pkg/name-unc":                   {param(0, "offset", centralHeader("\\\\server\\share\\x"))},
	"pkg/name-zero-width-space":      {param(0, "offset", centralHeader("assets/a\u200bb"))},
	"pkg/overlap":                    {param(0, "offset", dataEnd("assets/a.txt")), param(0, "declared", localHeader("assets/a.txt")), param(1, "offset", dataEnd("assets/a.txt")), param(1, "declared", localHeader("inputs/report.schema.json"))},
	"pkg/overlap-quoted":             {param(0, "offset", dataEnd("assets/a.bin")), param(0, "declared", localHeader("assets/b.txt"))},
	"pkg/second-end-record":          {param(0, "offset", func(b built) int { return fileData("assets/inner.zip")(b) + len(innerPrefix) })},
	"pkg/size-less-data":             {param(0, "offset", fileData(tmpl)), param(0, "declared", central(tmpl, declaredSize))},
	"pkg/size-mismatch":              {param(0, "offset", centralField(tmpl, cfCSize)), param(0, "declared", central(tmpl, declaredSize))},
	"pkg/size-more-data":             {param(0, "offset", fileData(tmpl)), param(0, "declared", central(tmpl, declaredSize))},
	"pkg/symlink-inside":             {param(0, "offset", centralField("assets/link", cfExternal))},
	"pkg/symlink-outside":            {param(0, "offset", centralField("assets/link", cfExternal))},
	"pkg/trailing-data":              {param(0, "offset", afterEnd)},
	"pkg/version-made-by":            {param(0, "offset", centralField(tmpl, cfMadeBy))},
	"pkg/version-needed":             {param(0, "offset", centralField(tmpl, cfNeeded))},
	"pkg/zip64":                      {param(0, "offset", endRecord)},
	"pkg/zip64-end-record":           {param(0, "offset", func(b built) int { return b.at.cdTail + zip64RecordLen })},
	"pkg/zip64-extra-field":          {param(0, "offset", centralHeader(tmpl)), param(0, "length", central(tmpl, extraLength)), param(1, "offset", centralExtra(tmpl))},
	"sec/package-size-packed":        {limit(0, "package-size", packageSize), limit(1, "package-size", below(packageSize)), param(0, "bound", below(packageSize)), param(0, "observed", packageSize)},
	"sec/package-uncompressed":       {limit(0, "package-uncompressed", uncompressed), limit(1, "package-uncompressed", below(uncompressed)), param(0, "bound", below(uncompressed)), param(0, "observed", uncompressed)},
	"pkg/prepended-data":             {param(0, "offset", start), param(0, "declared", localHeader(zipfmt.ReadmeName))},
	"pkg/digest-values":              {limit(0, "entries", func(b built) int { return entryCount(b) + 1 }), limit(1, "entries", entryCount)},
	"sec/entries-packed":             {limit(0, "entries", entryCount), limit(1, "entries", below(entryCount)), param(0, "bound", below(entryCount)), param(0, "observed", entryCount)},
	"sec/entry-size":                 {limit(0, "entry-size", central("assets/k.bin", declaredSize)), limit(1, "entry-size", below(central("assets/k.bin", declaredSize))), param(0, "bound", below(central("assets/k.bin", declaredSize))), param(0, "observed", central("assets/k.bin", declaredSize))},
	"sec/entry-ratio":                {limit(0, "entry-ratio", ratioBound("assets/k.bin")), limit(1, "entry-ratio", below(ratioBound("assets/k.bin"))), param(0, "bound", below(ratioBound("assets/k.bin"))), param(0, "observed", ratio("assets/k.bin"))},
	"sec/entry-ratio-default":        {param(0, "observed", ratio("assets/k.bin"))},
}

// TestValues is the program and its reproduction test for every vector's
// generated numbers: each is derived from its case's package and compared
// with the committed number, or, with -update, written in its place. It
// also fails on a byte-derived number of a vectorgen package's vector
// that no value writes, and on a file with written numbers that
// .gitattributes does not give the metais-generated-values merge driver.
func TestValues(t *testing.T) {
	testenv.Home(t)
	cases := map[string]pkgCase{}
	for _, c := range packageCases() {
		cases[c.vector] = c
	}
	owned := map[string]bool{} // <vector>/<file>#<pointer>
	files := map[string]bool{} // spec/conformance/vectors/<vector>/<file>
	for _, v := range slices.Sorted(maps.Keys(packageValues)) {
		c, ok := cases[v]
		if !ok {
			t.Errorf("%s: values for a vector no case builds", v)
			continue
		}
		b := c.build(t)
		docs := map[string][]byte{}
		changed := map[string]bool{}
		for _, x := range packageValues[v] {
			key := v + "/" + x.file + "#" + x.ptr
			if owned[key] {
				t.Errorf("%s: two values", key)
			}
			owned[key] = true
			files["spec/conformance/vectors/"+v+"/"+x.file] = true
			path := filepath.Join(vectorsDir, filepath.FromSlash(v), filepath.FromSlash(x.file))
			doc, ok := docs[path]
			if !ok {
				var err error
				if doc, err = os.ReadFile(path); err != nil {
					t.Fatal(err)
				}
			}
			from, to, err := numberAt(doc, x.ptr)
			if err != nil {
				t.Errorf("%s: %v", key, err)
				continue
			}
			want := strconv.Itoa(x.of(b))
			if got := string(doc[from:to]); got != want {
				if !*update {
					t.Errorf("%s is %s, and the package's bytes make it %s (run go test ./internal/vectorgen -run TestValues -args -update)", key, got, want)
					continue
				}
				doc = slices.Concat(doc[:from], []byte(want), doc[to:])
				changed[path] = true
			}
			docs[path] = doc
		}
		for _, path := range slices.Sorted(maps.Keys(changed)) {
			if err := os.WriteFile(path, docs[path], 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, c := range packageCases() {
		for _, key := range byteDerived(t, c.vector) {
			if !owned[key] {
				t.Errorf("%s: a number the package's bytes fix, which no value in values_test.go writes", key)
			}
		}
	}
	attrs, err := os.ReadFile("../../.gitattributes")
	if err != nil {
		t.Fatal(err)
	}
	driven := map[string]bool{}
	for _, l := range strings.Split(string(attrs), "\n") {
		if f := strings.Fields(l); len(f) == 2 && f[1] == "merge=metais-generated-values" {
			driven[f[0]] = true
		}
	}
	for _, f := range slices.Sorted(maps.Keys(files)) {
		if !driven[f] {
			t.Errorf(".gitattributes lacks the line %q", f+" merge=metais-generated-values")
		}
	}
	for _, f := range slices.Sorted(maps.Keys(driven)) {
		if !files[f] {
			t.Errorf(".gitattributes gives %s the metais-generated-values driver, but vectorgen writes no number in it", f)
		}
	}
}

// byteLimits are the limits whose bound a package's own bytes decide.
var byteLimits = []string{"package-size", "package-uncompressed", "entries", "entry-size", "entry-ratio"}

// byteDerived is every number of the vector's vector.json and of the
// findings files its steps expect that the package's bytes fix, as
// <vector>/<file>#<pointer>: a finding's numeric offset, declared and
// length, the observed of a finding of a limit of byteLimits and its bound
// when a step lowered it, and every limit of byteLimits a step lowers.
func byteDerived(t *testing.T, vector string) []string {
	dir := filepath.Join(vectorsDir, filepath.FromSlash(vector))
	var v struct {
		Cases []struct {
			Steps []struct {
				Args struct {
					Limits map[string]json.RawMessage `json:"limits"`
				} `json:"args"`
				Expect struct {
					Findings string `json:"findings"`
				} `json:"expect"`
			} `json:"steps"`
		} `json:"cases"`
	}
	read := func(name string, into any) {
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
		if err == nil {
			err = json.Unmarshal(b, into)
		}
		if err != nil {
			t.Fatalf("%s/%s: %v", vector, name, err)
		}
	}
	read("vector.json", &v)
	var out []string
	findings := map[string]bool{}
	for i, c := range v.Cases {
		for j, s := range c.Steps {
			for _, l := range slices.Sorted(maps.Keys(s.Args.Limits)) {
				if slices.Contains(byteLimits, l) {
					out = append(out, fmt.Sprintf("%s/vector.json#/cases/%d/steps/%d/args/limits/%s", vector, i, j, l))
				}
			}
			if s.Expect.Findings != "" {
				findings[s.Expect.Findings] = true
			}
		}
	}
	for _, name := range slices.Sorted(maps.Keys(findings)) {
		var fs []struct {
			Params map[string]any `json:"params"`
		}
		read(name, &fs)
		for i, f := range fs {
			keys := []string{"offset", "declared", "length"}
			if l, _ := f.Params["limit"].(string); slices.Contains(byteLimits, l) {
				keys = append(keys, "observed")
				if f.Params["lowered"] == true {
					keys = append(keys, "bound")
				}
			}
			for _, k := range keys {
				if _, ok := f.Params[k].(float64); ok {
					out = append(out, fmt.Sprintf("%s/%s#/%d/params/%s", vector, name, i, k))
				}
			}
		}
	}
	return out
}

// numberAt is where the number at the JSON Pointer ptr (RFC 6901) is in
// doc: its first byte and the byte after its last.
func numberAt(doc []byte, ptr string) (int, int, error) {
	if !strings.HasPrefix(ptr, "/") {
		return 0, 0, fmt.Errorf("%q is not a JSON Pointer to a member", ptr)
	}
	var want []string
	for _, s := range strings.Split(ptr[1:], "/") {
		want = append(want, strings.ReplaceAll(strings.ReplaceAll(s, "~1", "/"), "~0", "~"))
	}
	// Each open array or object: an array's index of the value being read,
	// or an object's member name and whether a name comes next.
	type frame struct {
		array   bool
		index   int
		name    string
		wantKey bool
	}
	var stack []frame
	next := func() {
		if len(stack) > 0 {
			if f := &stack[len(stack)-1]; f.array {
				f.index++
			} else {
				f.wantKey = true
			}
		}
	}
	at := func() bool {
		if len(stack) != len(want) {
			return false
		}
		for i, f := range stack {
			if s := f.name; f.array && strconv.Itoa(f.index) != want[i] || !f.array && s != want[i] {
				return false
			}
		}
		return true
	}
	dec := json.NewDecoder(bytes.NewReader(doc))
	dec.UseNumber()
	for {
		tok, err := dec.Token()
		if errors.Is(err, io.EOF) {
			return 0, 0, fmt.Errorf("no number at %s", ptr)
		}
		if err != nil {
			return 0, 0, err
		}
		if d, ok := tok.(json.Delim); ok {
			switch d {
			case '{':
				stack = append(stack, frame{wantKey: true})
			case '[':
				stack = append(stack, frame{array: true})
			default:
				stack = stack[:len(stack)-1]
				next()
			}
			continue
		}
		if n := len(stack); n > 0 && !stack[n-1].array && stack[n-1].wantKey {
			stack[n-1].name, stack[n-1].wantKey = tok.(string), false
			continue
		}
		if at() {
			n, ok := tok.(json.Number)
			if !ok {
				return 0, 0, fmt.Errorf("the value at %s is not a number", ptr)
			}
			end := int(dec.InputOffset())
			return end - len(n), end, nil
		}
		next()
	}
}
