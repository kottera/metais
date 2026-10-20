// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/user"
	"path/filepath"
	"slices"
	"strings"
	"sync"
)

// Check 2 of the SC7 harness (ADR-009 D16 item 1): no stream holds a real
// fact of the machine the test runs on — its host name or user name, when
// longer than three characters, or the absolute path of the test's
// temporary root, of the runner's real home directory or of the checkout.
// A sentinel cannot be planted in these: the tool would have to ask the
// operating system for them, which the static tripwire refuses, so check 2
// is the dynamic half of that rule.

// hostFact is one real fact of the host, lower-cased, and the source a
// hit on it names.
type hostFact struct {
	what  string
	value string
	path  bool // a path: matched where no path character continues it
}

// hostFacts are the host's facts, gathered once, and the text a host or
// user name may share with what the tool and the suite write themselves.
type hostFacts struct {
	names []hostFact // the host name and the user name
	paths []hostFact // the real home directory and the checkout
	texts sync.Map   // a vector's directory: the text of its files
}

// minNameLength is the length a host or user name must exceed to be
// looked for: a shorter one is a word of too many texts.
const minNameLength = 3

// newHostFacts gathers this host's facts. A host or user name that is a
// word of vocab, the texts the implementation and the suite write of
// their own — its help, the registry's messages — is not looked for: a
// scan cannot tell it from the tool's own text. Nor, in a case, is one
// that is a word of the vector's own files, which the tool may echo.
func newHostFacts(vocab ...[]byte) *hostFacts {
	h := &hostFacts{}
	text := bytes.Join(vocab, []byte{0})
	name := func(what, v string) {
		f := hostFact{what: what, value: strings.ToLower(v)}
		if len(f.value) > minNameLength && factAt(text, []hostFact{f}) == "" &&
			!slices.ContainsFunc(h.names, func(g hostFact) bool { return g.value == f.value }) {
			h.names = append(h.names, f)
		}
	}
	if host, err := os.Hostname(); err == nil {
		name("the host name", host)
		if label, _, ok := strings.Cut(host, "."); ok {
			name("the host name", label)
		}
	}
	homes := []string{}
	if u, err := user.Current(); err == nil {
		name("the user name", u.Username)
		homes = append(homes, u.HomeDir)
	}
	for _, k := range []string{"USER", "LOGNAME"} {
		name("the user name", os.Getenv(k))
	}
	if home, err := os.UserHomeDir(); err == nil {
		homes = append(homes, home)
	}
	for _, home := range homes {
		h.paths = addPath(h.paths, "the real home directory", home)
	}
	if root, err := filepath.Abs(moduleRoot); err == nil {
		h.paths = addPath(h.paths, "the checkout", root)
	}
	return h
}

// addPath adds the absolute path p, and the path its links resolve to,
// to facts as what. The root, or a path that is not absolute, is no fact.
func addPath(facts []hostFact, what, p string) []hostFact {
	ps := []string{p}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		ps = append(ps, r)
	}
	for _, p := range ps {
		p = strings.ToLower(filepath.Clean(p))
		if !filepath.IsAbs(p) || len(p) <= 1 || slices.ContainsFunc(facts, func(f hostFact) bool { return f.value == p }) {
			continue
		}
		facts = append(facts, hostFact{what: what, value: p, path: true})
	}
	return facts
}

// vectorText is the text of every file of v's directory, each file's
// first MiB, read once.
func (h *hostFacts) vectorText(v *vector) []byte {
	if b, ok := h.texts.Load(v.dir); ok {
		return b.([]byte)
	}
	var text []byte
	filepath.WalkDir(v.dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || !d.Type().IsRegular() {
			return nil
		}
		text = append(append(text, 0), d.Name()...)
		if f, err := os.Open(p); err == nil {
			b, _ := io.ReadAll(io.LimitReader(f, 1<<20))
			f.Close()
			text = append(append(text, 0), b...)
		}
		return nil
	})
	h.texts.Store(v.dir, text)
	return text
}

// finder is check 2's finder for a case of v whose sandboxes are below
// tempRoot, the case's temporary root.
func (h *hostFacts) finder(v *vector, tempRoot string) finder {
	text := h.vectorText(v)
	var facts []hostFact
	for _, f := range h.names {
		if factAt(text, []hostFact{f}) == "" {
			facts = append(facts, f)
		}
	}
	facts = append(facts, h.paths...)
	facts = addPath(facts, "the test's temporary root", tempRoot)
	return finder{check: checkHost, holds: "the host's", find: func(b []byte) string { return factAt(b, facts) }}
}

// factAt names the first of facts that b holds, case-insensitively, or
// gives "". A name counts only as a whole word, and a path only where no
// path character continues it on either side, so that a word or a path
// the fact begins or ends is no hit.
func factAt(b []byte, facts []hostFact) string {
	low := bytes.ToLower(b)
	for _, f := range facts {
		v := []byte(f.value)
		for off := 0; off < len(low); {
			i := bytes.Index(low[off:], v)
			if i < 0 {
				break
			}
			i += off
			if f.bounded(low, i, i+len(v)) {
				return f.what
			}
			off = i + 1
		}
	}
	return ""
}

// bounded reports whether the match of f at b[i:j] stands alone.
func (f hostFact) bounded(b []byte, i, j int) bool {
	cont := func(c byte) bool {
		return 'a' <= c && c <= 'z' || '0' <= c && c <= '9' || c == '_' || c == '-' || f.path && (c == '.' || c == '~')
	}
	if i > 0 && cont(b[i-1]) {
		return false
	}
	return j == len(b) || !cont(b[j])
}

// toolVocabulary is the tool's own text: help --json, and help --json of
// every command it lists, run in a plain sandbox below base.
func toolVocabulary(program, base string) ([][]byte, error) {
	sb, err := plainSandbox(base)
	if err != nil {
		return nil, err
	}
	help := func(args ...string) ([]byte, error) {
		p, err := runProc(program, append([]string{"help", "--json"}, args...), sb.work, sb.environ(), nil, defaultWall)
		if err != nil {
			return nil, err
		}
		if p.exit != 0 {
			return nil, fmt.Errorf("help --json %s exited %d", strings.Join(args, " "), p.exit)
		}
		return p.stdout, nil
	}
	top, err := help()
	if err != nil {
		return nil, err
	}
	var doc struct {
		Result struct {
			Commands []struct {
				Name string `json:"name"`
			} `json:"commands"`
		} `json:"result"`
	}
	if err := json.Unmarshal(top, &doc); err != nil {
		return nil, fmt.Errorf("help --json: %w", err)
	}
	out := [][]byte{top}
	for _, c := range doc.Result.Commands {
		b, err := help(c.Name)
		if err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, nil
}
