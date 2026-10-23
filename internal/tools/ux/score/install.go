// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"path"
	"regexp"
	"slices"
	"strings"
)

// An install scenario (ADR-013 D9) starts with no metais on PATH, so a
// metais the session runs is one it installed. The scorer reads how it
// installed it from its commands: the routes of D6 item 1, and a build
// from the source, which is none of them.

// The routes, as an outcome names them.
const (
	routeBrew    = "brew install kottera/tap/metais"
	routeScript  = "the install script from metais.org"
	routeGo      = "go install metais.org/metais/cmd/metais"
	routeRelease = "the GitHub release's archives"
	routeBuild   = "a build from the source"
)

// goldenRoutes are D6's golden paths; the other routes of D6 are its
// escape hatches.
var goldenRoutes = []string{routeBrew, routeScript}

// shells are the commands that run a script they are given.
var shells = []string{"sh", "bash", "dash", "zsh", "ksh"}

// install is a command of the session's that installs metais by a route,
// in a Bash call the harness reports no error for.
type install struct {
	cmd   *cmd
	route string
}

func (i install) golden() bool { return slices.Contains(goldenRoutes, i.route) }

func (i install) evidence() Evidence {
	e := i.cmd.evidence()
	e.Text = "installs metais by " + i.route
	return e
}

// installs are the session's installs of metais, in order:
//   - brew install (or reinstall) of kottera/tap/metais, or of metais
//     once the session has tapped kottera/tap;
//   - curl or wget of metais.org/install.sh, in a call that pipes it to
//     a shell or runs a shell on it, or a later shell, or the script
//     itself, run on the file it was saved as;
//   - go install metais.org/metais/cmd/metais;
//   - curl, wget or gh of github.com/kottera/metais's releases, or gh
//     release download of kottera/metais;
//   - go build of cmd/metais.
func (r *run) installs() []install {
	var (
		out     []install
		tapped  bool
		scripts []string // the files the install script was saved as
	)
	add := func(c *cmd, route string) {
		if c.ok() {
			out = append(out, install{c, route})
		}
	}
	for _, c := range r.cmds {
		args := slices.Concat(c.rest, c.s.outs)
		switch {
		case c.name == "brew":
			ops := operands(c.rest)
			switch {
			case len(ops) > 1 && ops[0] == "tap" && ops[1] == "kottera/tap":
				tapped = true
			case len(ops) > 1 && (ops[0] == "install" || ops[0] == "reinstall") &&
				(slices.Contains(ops[1:], "kottera/tap/metais") || tapped && slices.Contains(ops[1:], "metais")):
				add(c, routeBrew)
			}
		case (c.name == "curl" || c.name == "wget") && slices.ContainsFunc(args, isInstallScript):
			if slices.ContainsFunc(r.cmds, func(o *cmd) bool { return o.use == c.use && slices.Contains(shells, o.name) }) {
				add(c, routeScript)
			} else if f := savedAs(c); f != "" {
				scripts = append(scripts, f)
			}
		case slices.Contains(shells, c.name) && slices.ContainsFunc(operands(c.rest), func(w string) bool { return slices.Contains(scripts, path.Base(w)) }),
			slices.Contains(scripts, c.name):
			add(c, routeScript)
		case c.name == "go" && len(c.rest) > 0 && c.rest[0] == "install" && slices.ContainsFunc(c.rest, func(w string) bool { return strings.HasPrefix(w, "metais.org/metais/cmd/metais") }):
			add(c, routeGo)
		case (c.name == "curl" || c.name == "wget" || c.name == "gh") && slices.ContainsFunc(args, func(w string) bool { return strings.Contains(w, "github.com/kottera/metais/releases") }),
			c.name == "gh" && len(c.rest) > 1 && c.rest[0] == "release" && c.rest[1] == "download" && slices.ContainsFunc(c.rest, func(w string) bool { return strings.Contains(w, "kottera/metais") }):
			add(c, routeRelease)
		case c.name == "go" && len(c.rest) > 0 && c.rest[0] == "build" && slices.ContainsFunc(c.rest, func(w string) bool { return strings.Contains(w, "cmd/metais") }):
			add(c, routeBuild)
		}
	}
	return out
}

// isInstallScript reports whether a word names install.sh at metais.org.
func isInstallScript(w string) bool { return strings.Contains(w, "metais.org/install.sh") }

// savedAs is the name of the file curl or wget saved what it fetched as:
// curl's -o or --output, its -O or --remote-name, which keep the URL's
// name, wget's -O or --output-document, a short flag of either among
// others (-fsSLo, -qO), or a redirection of its output; wget saves under
// the URL's name by default. It is "" for standard output.
func savedAs(c *cmd) string {
	name := func(v string) string {
		if v == "-" || v == "" {
			return ""
		}
		return path.Base(v)
	}
	for i, w := range c.rest {
		next := ""
		if i+1 < len(c.rest) {
			next = c.rest[i+1]
		}
		short := strings.HasPrefix(w, "-") && !strings.HasPrefix(w, "--")
		switch {
		case w == "--output" || w == "--output-document":
			return name(next)
		case strings.HasPrefix(w, "--output=") || strings.HasPrefix(w, "--output-document="):
			_, v, _ := strings.Cut(w, "=")
			return name(v)
		case w == "--remote-name":
			return "install.sh"
		case short && c.name == "curl" && strings.HasSuffix(w, "o"):
			return name(next)
		case short && c.name == "curl" && strings.Contains(w, "O"):
			return "install.sh"
		case short && c.name == "wget" && strings.Contains(w, "O"):
			if _, v, _ := strings.Cut(w, "O"); v != "" {
				return name(v)
			}
			return name(next)
		}
	}
	if len(c.s.outs) > 0 {
		return name(c.s.outs[len(c.s.outs)-1])
	}
	if c.name == "wget" {
		return "install.sh"
	}
	return ""
}

// operands are a command's words that are not flags.
func operands(words []string) []string {
	return slices.DeleteFunc(slices.Clone(words), func(w string) bool { return strings.HasPrefix(w, "-") })
}

// notFound is a shell's report that it found no metais to run.
var notFound = regexp.MustCompile(`metais: (?:command )?not found|metais: No such file or directory`)

// installed is the session's first metais command that ran, the harness
// reporting no error for its call and the shell finding the command, and
// the last install before it: the route the metais that ran came by. It
// is nil when the session ran no metais, or ran one with no install the
// scorer sees before it.
func (r *run) installed() (*install, *call) {
	var ran *call
	for _, c := range r.calls {
		if c.ok() && !notFound.MatchString(c.output()) {
			ran = c
			break
		}
	}
	if ran == nil {
		return nil, nil
	}
	var last *install
	for _, i := range r.installs() {
		if i.cmd.seq < ran.seq {
			last = &i
		}
	}
	return last, ran
}

// installLimits is what the rule and the check that read an install cannot
// see.
const installLimits = "An install is read from the session's commands, in a Bash call the harness reports no error for: brew install kottera/tap/metais, or metais once kottera/tap is tapped; curl or wget of metais.org/install.sh piped to a shell in the same call, or saved and then run; go install metais.org/metais/cmd/metais; a download of github.com/kottera/metais's releases; go build of cmd/metais. The metais that ran is credited to the last such install before the session's first metais command that ran, because the session started with no metais on PATH: the scorer does not check which binary ran, nor that the install itself succeeded rather than its call, so curl of the script failing into a shell that then reads nothing is read as an install when a metais runs after it."

// goldenInstall is the rule of an install scenario that gates (I1): the
// session installs metais by a golden path of ADR-013 D6, brew install
// kottera/tap/metais or the install script from metais.org, before the
// first metais it runs.
func goldenInstall(r *run) Result {
	inst, ran := r.installed()
	var tried []Evidence
	for _, i := range r.installs() {
		tried = append(tried, i.evidence())
	}
	switch {
	case ran == nil:
		return Result{Detail: "the session never runs a metais it installed", Evidence: tried}
	case inst == nil:
		return Result{Detail: "metais runs, but the session installs it by no route the scorer sees before it", Evidence: []Evidence{ran.evidence()}}
	case !inst.golden():
		return Result{Detail: "the session installs metais by " + inst.route + ", not by a golden path of ADR-013 D6 (" + strings.Join(goldenRoutes, ", or ") + ")",
			Evidence: []Evidence{inst.evidence(), ran.evidence()}}
	}
	return Result{Pass: true, Detail: "the session installs metais by " + inst.route + ", a golden path of ADR-013 D6, and then runs it",
		Evidence: []Evidence{inst.evidence(), ran.evidence()}}
}

// findsAndInstalls is I2's check: given A1's task with nothing installed,
// the session finds METAIS, installs it by any route, and runs it. The
// result records the route, and A1's own check as a note: I2 is evidence,
// not a gate (ADR-013 D9).
func findsAndInstalls(r *run) Result {
	inst, ran := r.installed()
	a1 := newPackage(r)
	note := Evidence{Text: "A1's check (new-package) " + map[bool]string{true: "passes", false: "fails"}[a1.Pass] + ": " + a1.Detail}
	if len(a1.Evidence) > 0 {
		e := a1.Evidence[len(a1.Evidence)-1]
		note.Turn, note.Line, note.Tool, note.Command = e.Turn, e.Line, e.Tool, e.Command
	}
	switch {
	case ran == nil:
		var tried []Evidence
		for _, i := range r.installs() {
			tried = append(tried, i.evidence())
		}
		return Result{Detail: "the session never runs metais: it does not find and install it", Evidence: tried, Notes: []Evidence{note}}
	case inst == nil:
		return Result{Pass: true, Detail: "the session runs a metais it installed by a route the scorer does not name", Evidence: []Evidence{ran.evidence()}, Notes: []Evidence{note}}
	}
	kind := "an escape hatch of ADR-013 D6"
	switch {
	case inst.golden():
		kind = "a golden path of ADR-013 D6"
	case inst.route == routeBuild:
		kind = "no route of ADR-013 D6"
	}
	return Result{Pass: true, Detail: "the session finds METAIS, installs it by " + inst.route + ", " + kind + ", and runs it",
		Evidence: []Evidence{inst.evidence(), ran.evidence()}, Notes: []Evidence{note}}
}
