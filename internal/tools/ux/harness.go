// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"metais.org/metais/internal/tools/ux/scrub"
	"metais.org/metais/internal/tools/ux/streamjson"
)

// harness is the agent harness a session runs in (R-UX-6): the runner
// takes it as a parameter, and Claude Code is the only one implemented.
type harness interface {
	// name is the harness's name, as -harness gives it and every record
	// carries it.
	name() string
	// version asks the harness's own binary for its version.
	version() (string, error)
	// invocation is the command that runs one session.
	invocation(s session) invocation
	// checks are the harness's isolation checks of the command before it
	// runs (checks.go).
	checks(inv invocation) []check
	// read reads a session's transcript.
	read(transcript []byte) transcript
	// after are the harness's isolation checks of a finished session: its
	// transcript, and its HOME.
	after(s session, version string, t transcript) []check
}

// session is what the runner prepared for one run.
type session struct {
	work   string // the session's directory, holding only its fixtures
	home   string // its HOME
	tmp    string // its TMPDIR
	bin    string // the directory holding the metais under test, first on PATH
	model  string // the model, as the harness names it
	prompt string
	skill  bool // metais skill was installed for it
	// install marks a session of an install scenario (ADR-013 D9): bin is
	// left off its PATH, so it starts with no metais.
	install bool
}

// invocation is one command, exactly as the runner starts it.
type invocation struct {
	path string
	args []string
	env  []string
	dir  string
}

// lookup is the value the last kv of env names gives name, as the process
// sees it, or "" and false.
func lookup(env []string, name string) (string, bool) {
	for _, kv := range slices.Backward(env) {
		if k, v, ok := strings.Cut(kv, "="); ok && k == name {
			return v, true
		}
	}
	return "", false
}

// placeholderToken is R-UX-7's non-secret credential. A dispatch's
// ANTHROPIC_BASE_URL is a loopback proxy that authenticates upstream; a
// nested claude refuses on the client side unless some credential variable
// is set, so the runner sets this one. The scrub declares it, so what a
// session is given is what a scrub removes.
const placeholderToken = scrub.PlaceholderToken

// claudeCodeName is the Claude Code harness's name.
const claudeCodeName = "claude-code"

// defaultModels are R-UX-5's three, by Claude Code's aliases.
var defaultModels = []string{"opus", "sonnet", "haiku"}

// harnessByName is the harness -harness names, configured from environ.
func harnessByName(name string, environ []string) (harness, error) {
	if name != claudeCodeName {
		return nil, fmt.Errorf("harness %q is not implemented; %s is the only one (R-UX-6)", name, claudeCodeName)
	}
	path, ok := lookup(environ, "PATH")
	if !ok {
		return nil, errors.New("PATH is not set")
	}
	base, _ := lookup(environ, "ANTHROPIC_BASE_URL")
	return &claudeCode{hostPath: path, baseURL: base}, nil
}

// claudeCode is the Claude Code harness: claude -p, its transcript in
// stream-json.
type claudeCode struct {
	hostPath string // the runner's PATH, after the metais under test
	baseURL  string // the dispatch's loopback proxy
	binary   string // claude's absolute path, found on hostPath
}

func (c *claudeCode) name() string { return claudeCodeName }

// find resolves claude on the runner's PATH once.
func (c *claudeCode) find() (string, error) {
	if c.binary != "" {
		return c.binary, nil
	}
	for _, dir := range filepath.SplitList(c.hostPath) {
		if !filepath.IsAbs(dir) {
			continue
		}
		if p, err := exec.LookPath(filepath.Join(dir, "claude")); err == nil {
			c.binary = p
			return p, nil
		}
	}
	return "", errors.New("claude is not on an absolute directory of PATH")
}

func (c *claudeCode) version() (string, error) {
	if c.baseURL == "" {
		return "", errors.New("ANTHROPIC_BASE_URL is not set: the runs need the dispatch's loopback proxy (R-UX-7)")
	}
	p, err := c.find()
	if err != nil {
		return "", err
	}
	out, err := exec.Command(p, "--version").Output()
	if err != nil {
		return "", fmt.Errorf("%s --version: %w", p, err)
	}
	return strings.TrimSpace(string(out)), nil
}

// sessionSettings is the settings JSON every session is given: auto-memory
// off in settings, as CLAUDE_CODE_DISABLE_AUTO_MEMORY turns it off in the
// environment.
const sessionSettings = `{"autoMemoryEnabled":false}`

// invocation is claude -p with the flags confirmed against Claude Code
// 2.1.299: the prompt in print mode, its transcript in stream-json, no
// session saved, no MCP server but --mcp-config's (none), project settings
// only, auto-memory off, and permissions bypassed, as nobody is there to
// answer them. Its environment holds nothing of the host's but PATH and
// the proxy's address. An install scenario's session gets the runner's
// PATH alone, with no metais under test before it, and GOFLAGS
// -modcacherw, so a module cache go install makes in its HOME is one the
// runner can remove when it runs the run again.
func (c *claudeCode) invocation(s session) invocation {
	path := s.bin + string(filepath.ListSeparator) + c.hostPath
	if s.install {
		path = c.hostPath
	}
	inv := invocation{
		path: c.binary,
		args: []string{
			"-p", s.prompt,
			"--model", s.model,
			"--output-format", "stream-json", "--verbose",
			"--no-session-persistence",
			"--strict-mcp-config",
			"--setting-sources", "project",
			"--settings", sessionSettings,
			"--permission-mode", "bypassPermissions",
		},
		env: []string{
			"PATH=" + path,
			"HOME=" + s.home,
			"TMPDIR=" + s.tmp,
			"CLAUDE_CODE_TMPDIR=" + s.tmp,
			"ANTHROPIC_BASE_URL=" + c.baseURL,
			"ANTHROPIC_AUTH_TOKEN=" + placeholderToken,
			"CLAUDE_CODE_DISABLE_AUTO_MEMORY=1",
			"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
			"DISABLE_AUTOUPDATER=1",
		},
		dir: s.work,
	}
	if s.install {
		inv.env = append(inv.env, "GOFLAGS=-modcacherw")
	}
	return inv
}

func (c *claudeCode) checks(inv invocation) []check {
	return []check{
		checkCredential(inv),
		checkAutoMemoryOff(inv),
		checkProjectSettingsOnly(inv),
		checkStrictMCP(inv),
		checkNoSessionPersistence(inv),
	}
}

func (c *claudeCode) after(s session, version string, t transcript) []check {
	return []check{
		checkSessionReported(s, version, t),
		checkNoMCPServers(t),
		checkSkillAsDeclared(s, t),
		checkNoSessionSaved(s.home),
		checkNoMemoryRead(s, t),
	}
}

// transcript is the runner's reading of a session's transcript, the one
// the scorer reads too.
type transcript = *streamjson.Transcript

// read reads a stream-json transcript.
func (c *claudeCode) read(b []byte) transcript { return streamjson.Read(b) }

// quotaPause reports whether a session ended on a quota or rate limit
// rather than finishing: the seat's quota ran out, and the run is to be
// run again, not recorded as the session's behaviour (R-UX-7).
func quotaPause(t transcript, stderr []byte) bool {
	if t.Result.Found && !t.Result.IsError {
		return false
	}
	if t.Result.APIStatus == 429 {
		return true
	}
	text := strings.ToLower(t.Result.Text + "\n" + string(stderr))
	for _, s := range []string{"usage limit", "rate limit", "rate_limit", "quota", "429"} {
		if strings.Contains(text, s) {
			return true
		}
	}
	return false
}
