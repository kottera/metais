// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package scrub makes a UX run's files fit to publish (the Wave 1 brief's
// ruling, PLAN-2026-10-metais-ux W1-2): it takes every credential out of
// them, the runner's placeholder token included, every path of the host
// outside the run's own directory, and the user and group names ls -l
// prints (Ed's ruling, 2026-10-21). It is the step a scorer's
// committed fixtures were made by, and the one a baseline's transcripts
// are put through before they are kept: the runner's -scrub runs it over
// a set.
//
// The package works on bytes it is handed and refers to no host symbol
// (ADR-009 Amendment A8 item 4).
package scrub

import (
	"regexp"
	"slices"
	"strings"
)

// The placeholders a scrubbed file holds.
const (
	// RunDir stands for the run's own directory: the run's work/, home/
	// and tmp/ are beneath it.
	RunDir = "/ux/run"
	// SetDir stands for the set's directory, which holds the metais under
	// test in bin/.
	SetDir = "/ux/set"
	// HostPath stands for any other path of the host.
	HostPath = "/ux/host"
	// Removed stands for a credential.
	Removed = "[scrubbed]"
)

// PlaceholderToken is the runner's ANTHROPIC_AUTH_TOKEN (R-UX-7), declared
// here alone so the runner sets the very token the scrub removes. It is
// no secret, but it is removed anyway, so a later reader never has to
// decide whether a token in a fixture is one.
const PlaceholderToken = "metais-ux-placeholder"

// credentials match a credential's value. Each is replaced whole.
var credentials = []*regexp.Regexp{
	// A credential variable's value, however it is written: an
	// environment entry, an export, a JSON member.
	regexp.MustCompile(`((?:ANTHROPIC_(?:AUTH_TOKEN|API_KEY)|CLAUDE_CODE_OAUTH_TOKEN|GH_TOKEN|GITHUB_TOKEN|AWS_SECRET_ACCESS_KEY|AWS_SESSION_TOKEN)["']?\s*[=:]\s*["']?)[^\s"'\\]+`),
	regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{8,}`),
	regexp.MustCompile(`\bgh[pousr]_[A-Za-z0-9]{20,}`),
	regexp.MustCompile(`\bgithub_pat_[A-Za-z0-9_]{20,}`),
	regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`\bxox[abprs]-[A-Za-z0-9-]{10,}`),
	regexp.MustCompile(`(?i)(\bbearer\s+)[A-Za-z0-9._~+/=-]{16,}`),
	regexp.MustCompile(regexp.QuoteMeta(PlaceholderToken)),
}

// hostRoot matches an absolute path beneath a directory where a host
// keeps its users', volumes' or temporary files, which names the host,
// its user or its job. A system path such as /usr/bin/env or
// /etc/bash.bashrc names none of them and is kept.
var hostRoot = regexp.MustCompile(`(^|\\[nrt]|[^A-Za-z0-9._~/-])((?:/(?:var/lib|var/folders|home|Users|private|tmp|root|opt|nix|mnt|srv|Volumes|workspace|data|scratch)\b(?:/[^\s"'\\:;,|()<>{}\[\]` + "`" + `*?]*)?)|(?:\b[A-Za-z]:\\(?:Users|ProgramData)\\[^\s"']*))`)

// hostMangled matches a host path as Claude Code names a project's
// directory after it, every character that is not a letter or a digit
// made a hyphen: -var-lib-runner-volumes-….
var hostMangled = regexp.MustCompile(`(^|[^A-Za-z0-9-])(-(?:var-lib|var-folders|home|Users|private|tmp|root|opt|nix|mnt|srv|Volumes|workspace|data|scratch)-[A-Za-z0-9-]*)`)

// mangled is p as Claude Code names a project's directory after it.
func mangled(p string) string {
	return regexp.MustCompile(`[^A-Za-z0-9]`).ReplaceAllString(p, "-")
}

// lsLong matches a line of ls -l: the mode, the link count, then the
// owner and the group, which name the host's user. Owner and group are
// replaced, the rest kept.
var lsLong = regexp.MustCompile(`([-dlcbps][rwxsStT-]{9}[@.+]?[ \t]+[0-9]+[ \t]+)([^ \t\\"]+)([ \t]+)([^ \t\\"]+)([ \t]+[0-9])`)

// The placeholders of an ls -l line's owner and group.
const (
	User  = "user"
	Group = "group"
)

// Scrubber replaces the paths of one run and every credential.
type Scrubber struct {
	dirs []dir
}

type dir struct{ from, to string }

// New is the scrubber of the run whose directory is runDir, in the set
// whose directory is setDir, both absolute as the runner made them.
func New(runDir, setDir string) *Scrubber {
	s := &Scrubber{}
	for _, d := range []dir{{runDir, RunDir}, {setDir, SetDir}} {
		if d.from = strings.TrimRight(d.from, "/"); d.from != "" {
			s.dirs = append(s.dirs, d, dir{mangled(d.from), mangled(d.to)})
		}
	}
	// The longer first: the run's directory is beneath the set's.
	slices.SortFunc(s.dirs, func(a, b dir) int { return len(b.from) - len(a.from) })
	return s
}

// Bytes is b scrubbed: the run's and the set's directories by their
// placeholders, every other host path by HostPath, and every credential
// by Removed. A scrubbed file is scrubbed again to itself.
func (s *Scrubber) Bytes(b []byte) []byte {
	text := string(b)
	for _, d := range s.dirs {
		text = strings.ReplaceAll(text, d.from, d.to)
	}
	text, _ = scrub(text)
	return []byte(text)
}

// Check reports what in a file a scrub would remove: each credential, and
// each host path outside the placeholders. A scrubbed file reports
// nothing.
func Check(b []byte) []string {
	_, found := scrub(string(b))
	return found
}

// scrub replaces every host path and credential of text, and lists each
// it replaced.
func scrub(text string) (string, []string) {
	var found []string
	text = hostRoot.ReplaceAllStringFunc(text, func(m string) string {
		sub := hostRoot.FindStringSubmatch(m)
		if kept(sub[2]) {
			return m
		}
		found = append(found, "a host path: "+sub[2])
		return sub[1] + HostPath
	})
	text = lsLong.ReplaceAllStringFunc(text, func(m string) string {
		sub := lsLong.FindStringSubmatch(m)
		if sub[2] == User && sub[4] == Group {
			return m
		}
		found = append(found, "a user name: "+sub[2]+" "+sub[4])
		return sub[1] + User + sub[3] + Group + sub[5]
	})
	text = hostMangled.ReplaceAllStringFunc(text, func(m string) string {
		sub := hostMangled.FindStringSubmatch(m)
		found = append(found, "a host path: "+sub[2])
		return sub[1] + mangled(HostPath)
	})
	for _, re := range credentials {
		text = re.ReplaceAllStringFunc(text, func(m string) string {
			// A variable's name is kept, and its value removed.
			out := Removed
			if sub := re.FindStringSubmatch(m); len(sub) > 1 {
				out = sub[1] + Removed
			}
			if out != m {
				found = append(found, "a credential: "+m)
			}
			return out
		})
	}
	return text, found
}

// kept reports whether a path is one of the placeholders, or beneath one.
func kept(p string) bool {
	for _, d := range []string{RunDir, SetDir, HostPath} {
		if p == d || strings.HasPrefix(p, d+"/") {
			return true
		}
	}
	return false
}
