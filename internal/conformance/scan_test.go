// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
)

// sentinelSources names where each sentinel suffix was planted, a suffix
// before every other it begins, so a match takes the most specific
// (ADR-011 D12 item 3, and the SC7 harness's additions of ADR-009 D16
// item 1).
var sentinelSources = []struct{ suffix, source string }{
	{"NOCOLOR", "NO_COLOR"},
	{"PATH", "the sandbox path"},
	{"HOME", "HOME"},
	{"USER", "USER"},
	{"HOST", "HOSTNAME"},
	{"TERM", "TERM"},
	{"CWD", "a working-directory decoy"},
	{"ENV", "METAIS_SENTINEL"},
	{"TMPDECOY", "a temporary-directory decoy"},
	{"TMP", "TMPDIR"},
	{"BIN", "PATH"},
	{"XDG", "an XDG directory"},
	{"OLDPWD", "OLDPWD"},
	{"SHELL", "SHELL"},
	{"EPOCH", "SOURCE_DATE_EPOCH"},
	{"CI", "CI"},
	{"GHTOKEN", "GITHUB_TOKEN"},
	{"AWSKEY", "AWS_SECRET_ACCESS_KEY"},
	{"METAIS", "a METAIS_<n> variable"},
}

// asciiUpper upper-cases the ASCII letters of b and nothing else, so an
// index into the result is an index into b.
func asciiUpper(b []byte) []byte {
	out := make([]byte, len(b))
	for i, c := range b {
		if 'a' <= c && c <= 'z' {
			c -= 'a' - 'A'
		}
		out[i] = c
	}
	return out
}

// sentinelAt returns the source of the sentinel in b, searched for
// case-insensitively, or "" when b holds none. Every planted path carries
// the sandbox's token first, so a hit naming a more specific source
// anywhere in b wins over the sandbox path.
func sentinelAt(b []byte) string {
	up := asciiUpper(b)
	found := ""
	for i := bytes.Index(up, []byte(sentinel)); i >= 0; {
		rest := string(up[i+len(sentinel) : min(len(up), i+len(sentinel)+8)])
		src := "an unknown source"
		for _, s := range sentinelSources {
			if strings.HasPrefix(rest, s.suffix) {
				src = s.source
				break
			}
		}
		if found == "" || found == "the sandbox path" {
			found = src
		}
		next := bytes.Index(up[i+len(sentinel):], []byte(sentinel))
		if next < 0 {
			break
		}
		i += len(sentinel) + next
	}
	return found
}

// zipDepth is how deep the scan inflates zips inside zips (ADR-011 D12
// item 3).
const zipDepth = 3

// finder is one kind of hit the scan looks for: the sentinel (check 1),
// or a real fact of the host (ADR-009 D16 item 1, check 2). find names the
// source of a hit in b, or gives "" when b holds none.
type finder struct {
	check string
	find  func(b []byte) string
	holds string // what a hit's message says the artefact holds
}

// sentinels is check 1's finder.
var sentinels = finder{check: checkSentinel, find: sentinelAt, holds: "the sentinel from"}

// scanBytes scans one artefact for the sentinel (scanFor).
func scanBytes(name string, b []byte, depth int) string {
	return scanFor(sentinels, name, b, depth)
}

// scanFor scans one artefact for f's hits: its bytes, and when it is a
// zip, each entry inflated, recursively to depth levels. It returns a hit
// as "<artefact> holds <source>", or "".
func scanFor(f finder, name string, b []byte, depth int) string {
	if src := f.find(b); src != "" {
		return fmt.Sprintf("%s holds %s %s", name, f.holds, src)
	}
	if depth == 0 || !isZip(b) {
		return ""
	}
	z, err := parseZip(b)
	if err != nil {
		return fmt.Sprintf("%s reads as a zip that cannot be opened, so the scan cannot see inside it: %v", name, err)
	}
	for _, h := range z.headers {
		if src := f.find([]byte(h.name)); src != "" {
			return fmt.Sprintf("%s: the entry name %q holds %s %s", name, h.name, f.holds, src)
		}
		content, err := h.inflate(b)
		if err != nil {
			return fmt.Sprintf("%s: the entry %q cannot be inflated, so the scan cannot see inside it: %v", name, h.name, err)
		}
		if hit := scanFor(f, name+": entry "+h.name, content, depth-1); hit != "" {
			return hit
		}
	}
	return ""
}

// scanTar scans each member of a tar stream for f's hits: its name and
// its content, and each zip inside it. A stream that does not read as a
// tar to its end cannot be scanned, which is itself a hit.
func scanTar(f finder, name string, b []byte) string {
	members, err := readTar(b)
	for _, m := range members {
		if src := f.find([]byte(m.name + "\x00" + m.uname + "\x00" + m.gname)); src != "" {
			return fmt.Sprintf("%s: the member header %q holds %s %s", name, m.name, f.holds, src)
		}
		if hit := scanFor(f, name+": member "+m.name, m.content, zipDepth); hit != "" {
			return hit
		}
	}
	if err != nil {
		return fmt.Sprintf("%s does not read as a tar to its end, so the scan cannot see inside it: %v", name, err)
	}
	return ""
}

// scanInvocation scans everything one invocation produced for f's hits:
// standard output and standard error (which hold a driver's response),
// every file of the step's declared trees and its zips, and the stream
// and its members.
func scanInvocation(f finder, inv *invocation, archive bool, trees map[string]*tree) []problem {
	var ps []problem
	add := func(hit string) {
		if hit != "" {
			ps = append(ps, problem{check: f.check, msg: hit})
		}
	}
	add(scanFor(f, "standard output", inv.stdout, 0))
	add(scanFor(f, "standard error", inv.stderr, 0))
	if inv.stream != nil {
		add(scanFor(f, "the stream", inv.stream, zipDepth))
		if archive {
			add(scanTar(f, "the stream", inv.stream))
		}
	}
	for _, root := range sortedKeys(trees) {
		t := trees[root]
		for _, n := range sortedKeys(t.files) {
			p := root + "/" + n
			if src := f.find([]byte(p)); src != "" {
				add(fmt.Sprintf("the written path %s holds %s %s", p, f.holds, src))
			}
			add(scanFor(f, "the written file "+p, t.files[n].content, zipDepth))
		}
	}
	return ps
}

// scanWritten scans every regular file a step created or changed anywhere
// in the sandbox outside its declared trees, which scanInvocation scans,
// for f's hits: a file the step should not have written may still carry
// a host fact, and ADR-009 D16 item 1 scans every written file. Paths are
// relative to the sandbox's root, as snapshot keys them; trees are
// relative to the work directory. A written path is not scanned for the
// sandbox's own path, which every path below the root carries.
func scanWritten(f finder, sb *sandbox, before, after map[string]node, trees []string) []problem {
	var ps []problem
	for _, p := range sortedKeys(after) {
		n := after[p]
		if n.kind != "file" || before[p] == n {
			continue
		}
		if rel, ok := strings.CutPrefix(p, "work/"); ok && slices.ContainsFunc(trees, func(t string) bool {
			t = path.Clean(t)
			return t == "." || rel == t || strings.HasPrefix(rel, t+"/")
		}) {
			continue
		}
		if src := f.find([]byte(p)); src != "" && src != "the sandbox path" {
			ps = append(ps, problem{check: f.check, msg: fmt.Sprintf("the written path %s holds %s %s", p, f.holds, src)})
		}
		b, err := os.ReadFile(filepath.Join(sb.root, filepath.FromSlash(p)))
		if err != nil {
			ps = append(ps, problem{check: checkHarness, harness: true, msg: err.Error()})
			continue
		}
		if hit := scanFor(f, "the written file "+p, b, zipDepth); hit != "" {
			ps = append(ps, problem{check: f.check, msg: hit})
		}
	}
	return ps
}
