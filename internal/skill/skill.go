// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package skill is metais's agent skill and the one writer of it
// (ADR-008 A2, ADR-009 A5): the SKILL.md a coding agent's harness loads
// before its first command, embedded in the binary and versioned with
// it, and Install, which writes it beneath the directory a caller names.
//
// The written bytes are the embedded text, the tool's version and the
// SHA-256 of the text: constants of the binary. Nothing a package or an
// input holds reaches them, and the package imports nothing that reads
// either (internal/guarantees' layers.json holds that).
package skill

import (
	"bytes"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"regexp"
	"slices"
	"strings"

	"metais.org/metais/internal/hostfs"
)

// source is the skill's text, without its stamp.
//
//go:embed SKILL.md
var source []byte

// Name is the skill's name, its frontmatter's name and its directory's.
const Name = "metais"

// MaxExisting is the most of an existing SKILL.md Install reads: a file
// larger than this is not a copy metais wrote (ADR-009 A5 item 2).
const MaxExisting = 64 << 10

// Harnesses are the harness names --harness accepts, in help's order.
var Harnesses = []string{"claude", "codex", "gemini", "cursor", "opencode"}

// The two directories the harnesses read a project's skills from, and the
// harness directories detection looks for (ADR-008 A2 items 3 and 6).
const (
	claudeDir = ".claude"
	agentsDir = ".agents"
)

// dirOf is the top-level directory harness h reads skills beneath.
func dirOf(h string) string {
	if h == "claude" {
		return claudeDir
	}
	return agentsDir
}

// PathOf is the path, relative to the directory, of the SKILL.md beneath
// the top-level directory top.
func PathOf(top string) string { return top + "/skills/" + Name + "/SKILL.md" }

// Source returns the skill's text without its stamp.
func Source() []byte { return slices.Clone(source) }

// Text returns the skill as Install writes it and --print prints it: the
// embedded text, then the stamp line of version (ADR-008 A2 item 5).
func Text(version string) []byte {
	return append(slices.Clone(source), stamp(version, source)...)
}

// stamp is the last line of a written SKILL.md: the version of the tool
// that wrote it and the SHA-256 of every byte before the line.
func stamp(version string, body []byte) string {
	sum := sha256.Sum256(body)
	return "<!-- metais skill: version " + version + ", sha256 " + hex.EncodeToString(sum[:]) + " -->\n"
}

// stampLine is a well-formed stamp line. A version is the characters a
// semantic version holds, so a version read from a file whose digest
// matched is text some metais wrote, never a control character.
var stampLine = regexp.MustCompile(`^<!-- metais skill: version ([0-9A-Za-z.+-]{1,64}), sha256 ([0-9a-f]{64}) -->\n$`)

// own reports whether b is a SKILL.md metais wrote and nobody edited: its
// last line is a well-formed stamp whose digest is that of every byte
// before it. It returns the stamp's version.
func own(b []byte) (version string, ok bool) {
	if len(b) == 0 || b[len(b)-1] != '\n' {
		return "", false
	}
	i := strings.LastIndexByte(string(b[:len(b)-1]), '\n') + 1
	m := stampLine.FindSubmatch(b[i:])
	if m == nil {
		return "", false
	}
	sum := sha256.Sum256(b[:i])
	if hex.EncodeToString(sum[:]) != string(m[2]) {
		return "", false
	}
	return string(m[1]), true
}

// Action is what Install did, or with DryRun would do, to one target.
type Action string

const (
	Installed    Action = "installed"
	Unchanged    Action = "unchanged"
	Updated      Action = "updated"
	Edited       Action = "edited"
	Replaced     Action = "replaced"
	WouldInstall Action = "would_install"
	WouldUpdate  Action = "would_update"
	WouldReplace Action = "would_replace"
)

// Options is what a caller asks of Install.
type Options struct {
	// Harnesses are the --harness values as given, repeats included.
	// When empty, Install detects the harness from the directory.
	Harnesses []string
	// Overwrite replaces a copy someone edited.
	Overwrite bool
	// DryRun decides every target and writes nothing.
	DryRun bool
	// Version is the tool's version, written into the stamp.
	Version string
}

// Target is one SKILL.md Install decided.
type Target struct {
	// Path is the file's path relative to the directory.
	Path string
	// Harnesses are the harnesses that read it: every name given that
	// shares its directory, or under detection every one that does.
	Harnesses []string
	// Because is why it is a target: "flag", or "<dir>/ exists".
	Because string
	// Action is what was done, or would be. It is empty when Err is set.
	Action Action
	// From is the version in the stamp of the copy found, when its
	// digest matched.
	From string
	// Err is a refusal or failed write of this target, a *hostfs.Error.
	// A target's file is renamed into place as its last step, so a target
	// with Err was not written.
	Err error
}

// Result is what Install did.
type Result struct {
	// Chosen is "flag" when --harness chose the targets, "detected"
	// when the directory did.
	Chosen  string
	Targets []Target
	// Repository reports whether the directory holds a .git entry.
	Repository bool
}

// NeedHarnessError is Install's refusal when no harness is given and the
// directory does not settle one: it holds neither .claude/ nor .agents/,
// or it holds both (ADR-008 A2 item 3). Nothing is written.
type NeedHarnessError struct {
	// Found is "neither" or "both".
	Found string
}

func (e *NeedHarnessError) Error() string {
	return "skill: no harness given, and the directory holds " + e.Found + " of .claude/ and .agents/"
}

// host is the directory Install opened, as the rest of the package sees
// it: plain functions, so that Install is the one place that names
// hostfs (ADR-009 A5 item 4).
type host struct {
	// isDir reports whether name is a directory, not a link to one.
	isDir func(name string) bool
	// exists reports whether there is an entry called name, of any kind.
	exists func(name string) bool
	// read returns the bytes of the regular file name, found false when
	// there is none, and nil bytes for a file over MaxExisting.
	read func(name string) (b []byte, found bool, err error)
	// write puts text at path, replacing a regular file when replace is
	// set and refusing any entry there otherwise.
	write func(path string, text []byte, replace bool) error
}

// Install decides and writes the skill beneath dir, the caller's own name
// for an existing directory, for the harnesses opt names or, when it
// names none, the one the directory shows (ADR-008 A2, ADR-009 A5). It is
// the package's one way to the host's files.
//
// Each target is decided on its own: a refused target is reported in its
// Err and does not stop another's write. An error is returned only when
// dir cannot be opened, as a *hostfs.Error, or no harness is settled, as
// a *NeedHarnessError.
func Install(dir string, opt Options) (Result, error) {
	w, err := hostfs.OpenWriter(dir)
	if err != nil {
		return Result{}, err
	}
	defer w.Close()
	h := host{
		isDir: func(name string) bool {
			e, err := w.Lstat(name)
			return err == nil && e.Kind == hostfs.KindDir
		},
		exists: func(name string) bool {
			_, err := w.Lstat(name)
			return err == nil
		},
		read: func(name string) ([]byte, bool, error) {
			b, err := w.ReadFile(name, MaxExisting)
			switch hostfs.ReasonOf(err) {
			case hostfs.NotFound:
				return nil, false, nil
			case hostfs.TooLarge:
				return nil, true, nil
			}
			return b, err == nil, err
		},
		// write creates the missing directories above path, writes a file
		// beside it under a fixed temporary name, and renames that over
		// path, so a reader never sees a partial file and an existing one
		// is replaced, never written into (ADR-009 D8 item 3). A failure
		// removes what the write created and is still empty.
		write: func(path string, text []byte, replace bool) error {
			dir := path[:strings.LastIndexByte(path, '/')]
			tmp := dir + "/" + hostfs.TempName("SKILL.md")
			err := w.MkdirAll(dir)
			if err == nil {
				err = w.Create(tmp, false, bytes.NewReader(text))
			}
			if err == nil && replace {
				err = w.RenameReplace(tmp, path)
			} else if err == nil {
				err = w.RenameNoReplace(tmp, path)
			}
			if err != nil {
				w.RemoveCreated()
			}
			return err
		},
	}
	return installIn(h, opt)
}

// installIn is Install over the directory h.
func installIn(h host, opt Options) (Result, error) {
	res := Result{Repository: h.exists(".git")}
	tops, err := targets(h, opt.Harnesses)
	if err != nil {
		return Result{}, err
	}
	res.Chosen = "detected"
	if len(opt.Harnesses) > 0 {
		res.Chosen = "flag"
	}
	text := Text(opt.Version)
	for _, top := range tops {
		t := Target{Path: PathOf(top), Because: "flag"}
		if res.Chosen == "detected" {
			t.Because = top + "/ exists"
		}
		for _, hn := range Harnesses {
			if dirOf(hn) == top && (res.Chosen == "detected" || slices.Contains(opt.Harnesses, hn)) {
				t.Harnesses = append(t.Harnesses, hn)
			}
		}
		t.Action, t.From, t.Err = decide(h, t.Path, text, opt)
		if t.Err == nil && !opt.DryRun && writes(t.Action) {
			if t.Err = h.write(t.Path, text, t.Action != Installed); t.Err != nil {
				t.Action = ""
			}
		}
		res.Targets = append(res.Targets, t)
	}
	return res, nil
}

// targets are the top-level directories the skill is written beneath:
// one per directory the named harnesses share, or, with none named, the
// one of .claude/ and .agents/ the directory holds.
func targets(h host, named []string) ([]string, error) {
	var tops []string
	if len(named) > 0 {
		for _, top := range []string{claudeDir, agentsDir} {
			if slices.ContainsFunc(named, func(hn string) bool { return dirOf(hn) == top }) {
				tops = append(tops, top)
			}
		}
		return tops, nil
	}
	for _, top := range []string{claudeDir, agentsDir} {
		if h.isDir(top) {
			tops = append(tops, top)
		}
	}
	switch len(tops) {
	case 0:
		return nil, &NeedHarnessError{Found: "neither"}
	case 2:
		return nil, &NeedHarnessError{Found: "both"}
	}
	return tops, nil
}

// decide reads the SKILL.md at path, if any, and decides what to do to it
// (ADR-008 A2 item 5). from is the version of a copy whose digest matched.
func decide(h host, path string, text []byte, opt Options) (a Action, from string, err error) {
	b, found, err := h.read(path)
	switch {
	case err != nil:
		return "", "", err
	case !found:
		return pick(opt.DryRun, Installed, WouldInstall), "", nil
	}
	if v, ok := own(b); ok {
		if string(b) == string(text) {
			return Unchanged, v, nil
		}
		return pick(opt.DryRun, Updated, WouldUpdate), v, nil
	}
	if opt.Overwrite {
		return pick(opt.DryRun, Replaced, WouldReplace), "", nil
	}
	return Edited, "", nil
}

func pick(dryRun bool, done, would Action) Action {
	if dryRun {
		return would
	}
	return done
}

// writes reports whether a reaches the file.
func writes(a Action) bool { return a == Installed || a == Updated || a == Replaced }
