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

// sentinelAt returns the source of the first sentinel in b, searched for
// case-insensitively, or "" when b holds none.
func sentinelAt(b []byte) string {
	i := bytes.Index(bytes.ToUpper(b), []byte(sentinel))
	if i < 0 {
		return ""
	}
	rest := strings.ToUpper(string(b[i+len(sentinel) : min(len(b), i+len(sentinel)+8)]))
	for _, s := range sentinelSources {
		if strings.HasPrefix(rest, s.suffix) {
			return s.source
		}
	}
	return "an unknown source"
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
	entries, err := readZip(b)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		if src := sentinelAt([]byte(e.name)); src != "" {
			return fmt.Sprintf("%s: the entry name %q holds the sentinel from %s", name, e.name, src)
		}
		if hit := scanBytes(name+": entry "+e.name, e.content, depth-1); hit != "" {
			return hit
		}
	}
	return ""
}

// scanTar scans each member of a tar stream: its name and its content,
// and each zip inside it.
func scanTar(name string, b []byte) string {
	members, err := readTar(b)
	if err != nil {
		return ""
	}
	for _, m := range members {
		if src := sentinelAt([]byte(m.name + "\x00" + m.uname + "\x00" + m.gname)); src != "" {
			return fmt.Sprintf("%s: the member header %q holds the sentinel from %s", name, m.name, src)
		}
		if hit := scanBytes(name+": member "+m.name, m.content, zipDepth); hit != "" {
			return hit
		}
	}
	return ""
}

// scanInvocation scans everything one invocation of environment A
// produced: standard output and standard error (which hold a driver's
// response), every file of the step's declared trees and its zips, and
// the stream and its members.
func scanInvocation(inv *invocation, trees map[string]*tree) []problem {
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
		add(scanTar("the stream", inv.stream))
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
