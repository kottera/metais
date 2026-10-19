// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"fmt"
	"strings"
)

// sentinelSources names where each sentinel suffix was planted, longest
// first so a match takes the most specific (ADR-011 D12 item 3).
var sentinelSources = []struct{ suffix, source string }{
	{"NOCOLOR", "NO_COLOR"},
	{"PATH", "the sandbox path"},
	{"HOME", "HOME"},
	{"USER", "USER"},
	{"HOST", "HOSTNAME"},
	{"TERM", "TERM"},
	{"CWD", "a working-directory decoy"},
	{"ENV", "METAIS_SENTINEL"},
	{"TMP", "TMPDIR"},
	{"BIN", "PATH"},
	{"XDG", "an XDG directory"},
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

// scanBytes scans one artefact: its bytes, and when it is a zip, each
// entry inflated, recursively to depth levels. It returns a hit as
// "<artefact>: <source>", or "".
func scanBytes(name string, b []byte, depth int) string {
	if src := sentinelAt(b); src != "" {
		return fmt.Sprintf("%s holds the sentinel from %s", name, src)
	}
	if depth == 0 || !isZip(b) {
		return ""
	}
	z, err := parseZip(b)
	if err != nil {
		return fmt.Sprintf("%s reads as a zip that cannot be opened, so the scan cannot see inside it: %v", name, err)
	}
	for _, h := range z.headers {
		if src := sentinelAt([]byte(h.name)); src != "" {
			return fmt.Sprintf("%s: the entry name %q holds the sentinel from %s", name, h.name, src)
		}
		content, err := h.inflate(b)
		if err != nil {
			return fmt.Sprintf("%s: the entry %q cannot be inflated, so the scan cannot see inside it: %v", name, h.name, err)
		}
		if hit := scanBytes(name+": entry "+h.name, content, depth-1); hit != "" {
			return hit
		}
	}
	return ""
}

// scanTar scans each member of a tar stream: its name and its content,
// and each zip inside it. A stream that does not read as a tar to its end
// cannot be scanned, which is itself a hit.
func scanTar(name string, b []byte) string {
	members, err := readTar(b)
	for _, m := range members {
		if src := sentinelAt([]byte(m.name + "\x00" + m.uname + "\x00" + m.gname)); src != "" {
			return fmt.Sprintf("%s: the member header %q holds the sentinel from %s", name, m.name, src)
		}
		if hit := scanBytes(name+": member "+m.name, m.content, zipDepth); hit != "" {
			return hit
		}
	}
	if err != nil {
		return fmt.Sprintf("%s does not read as a tar to its end, so the scan cannot see inside it: %v", name, err)
	}
	return ""
}

// scanInvocation scans everything one invocation of environment A
// produced: standard output and standard error (which hold a driver's
// response), every file of the step's declared trees and its zips, and
// the stream and its members.
func scanInvocation(inv *invocation, archive bool, trees map[string]*tree) []problem {
	var ps []problem
	add := func(hit string) {
		if hit != "" {
			ps = append(ps, problem{check: checkSentinel, msg: hit})
		}
	}
	add(scanBytes("standard output", inv.stdout, 0))
	add(scanBytes("standard error", inv.stderr, 0))
	if inv.stream != nil {
		add(scanBytes("the stream", inv.stream, zipDepth))
		if archive {
			add(scanTar("the stream", inv.stream))
		}
	}
	for root, t := range trees {
		for _, n := range sortedKeys(t.files) {
			p := root + "/" + n
			if src := sentinelAt([]byte(p)); src != "" {
				add(fmt.Sprintf("the written path %s holds the sentinel from %s", p, src))
			}
			add(scanBytes("the written file "+p, t.files[n].content, zipDepth))
		}
	}
	return ps
}
