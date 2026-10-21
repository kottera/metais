// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
)

// check is one isolation check's result on one run. A run that fails any
// check is recorded invalid and is not scored.
type check struct {
	Name   string `json:"name"`
	Pass   bool   `json:"pass"`
	Detail string `json:"detail,omitempty"`
}

func pass(name string) check { return check{Name: name, Pass: true} }

func fail(name, format string, args ...any) check {
	return check{Name: name, Detail: fmt.Sprintf(format, args...)}
}

// The checks' names, as every record carries them.
const (
	checkFresh           = "fresh-directory"
	checkOutsideRepo     = "outside-repository"
	checkMetaisOnPath    = "metais-first-on-path"
	checkCredentialName  = "placeholder-credential"
	checkAutoMemory      = "auto-memory-off"
	checkProjectSettings = "project-settings-only"
	checkStrictMCPName   = "strict-mcp-config"
	checkNoPersistence   = "no-session-persistence"
	checkReported        = "session-as-started"
	checkSkill           = "skill-as-declared"
	checkNoMemory        = "no-memory-read"
)

// fileEntry is one regular file of a directory: its path, slash-separated
// and relative to the directory, its size and its digest.
type fileEntry struct {
	Path   string `json:"path"`
	Size   int64  `json:"size"`
	SHA256 string `json:"sha256"`
	// Change is how the session changed it: added, modified, deleted or
	// unchanged. It is empty in a listing of fixtures.
	Change string `json:"change,omitempty"`
}

// listFiles lists every file beneath dir. Anything but a regular file or a
// directory is listed with size -1 and no digest, so a comparison of two
// listings sees it.
func listFiles(dir string) ([]fileEntry, error) {
	var files []fileEntry
	err := fs.WalkDir(os.DirFS(dir), ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || p == "." || d.IsDir() {
			return err
		}
		if !d.Type().IsRegular() {
			files = append(files, fileEntry{Path: p, Size: -1})
			return nil
		}
		b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(p)))
		if err != nil {
			return err
		}
		sum := sha256.Sum256(b)
		files = append(files, fileEntry{Path: p, Size: int64(len(b)), SHA256: hex.EncodeToString(sum[:])})
		return nil
	})
	return files, err
}

// checkFreshDirectory: the session's directory holds exactly the files
// the runner put there, its fixtures, and nothing else.
func checkFreshDirectory(work string, want []fileEntry) check {
	got, err := listFiles(work)
	if err != nil {
		return fail(checkFresh, "listing the directory: %v", err)
	}
	if !slices.Equal(got, want) {
		return fail(checkFresh, "the directory holds %s, want only the fixtures %s", paths(got), paths(want))
	}
	return pass(checkFresh)
}

func paths(files []fileEntry) string {
	var ps []string
	for _, f := range files {
		ps = append(ps, f.Path)
	}
	return "[" + strings.Join(ps, " ") + "]"
}

// checkOutsideRepository: the session's directory is an absolute path, and
// neither it nor any directory above it is a repository's top level, so a
// harness that reads a repository's instructions and settings from the
// directories above it finds none.
func checkOutsideRepository(work string) check {
	if !filepath.IsAbs(work) {
		return fail(checkOutsideRepo, "%s is not an absolute path", work)
	}
	for dir := filepath.Clean(work); ; dir = filepath.Dir(dir) {
		for _, marker := range []string{".git", ".hg", ".jj", ".svn"} {
			if _, err := os.Lstat(filepath.Join(dir, marker)); err == nil {
				return fail(checkOutsideRepo, "%s is inside the repository at %s", work, dir)
			}
		}
		if dir == filepath.Dir(dir) {
			return pass(checkOutsideRepo)
		}
	}
}

// checkMetaisFirstOnPath: the first metais the session's PATH finds is the
// one built from the commit under test, whose digest is sha.
func checkMetaisFirstOnPath(inv invocation, bin, sha string) check {
	path, _ := lookup(inv.env, "PATH")
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			return fail(checkMetaisOnPath, "PATH holds the relative directory %q before any metais", dir)
		}
		p := filepath.Join(dir, "metais")
		st, err := os.Stat(p)
		if err != nil || !st.Mode().IsRegular() || st.Mode().Perm()&0o111 == 0 {
			continue
		}
		if dir != bin {
			return fail(checkMetaisOnPath, "the first metais on PATH is %s, not the one under test in %s", p, bin)
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return fail(checkMetaisOnPath, "reading %s: %v", p, err)
		}
		if sum := sha256.Sum256(b); hex.EncodeToString(sum[:]) != sha {
			return fail(checkMetaisOnPath, "%s is not the binary built from the commit under test", p)
		}
		return pass(checkMetaisOnPath)
	}
	return fail(checkMetaisOnPath, "no metais on PATH")
}

// checkCredential: the session's only credential is R-UX-7's placeholder,
// so no real credential of the host's reaches it or its transcript.
func checkCredential(inv invocation) check {
	for _, kv := range inv.env {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case k == "ANTHROPIC_AUTH_TOKEN" && v != placeholderToken:
			return fail(checkCredentialName, "ANTHROPIC_AUTH_TOKEN is not the placeholder")
		case k != "ANTHROPIC_AUTH_TOKEN" && (strings.Contains(k, "API_KEY") || strings.Contains(k, "TOKEN") || strings.HasPrefix(k, "CLAUDE_CODE_OAUTH")):
			return fail(checkCredentialName, "the environment carries %s", k)
		}
	}
	if v, ok := lookup(inv.env, "ANTHROPIC_AUTH_TOKEN"); !ok || v != placeholderToken {
		return fail(checkCredentialName, "ANTHROPIC_AUTH_TOKEN is not set to the placeholder")
	}
	return pass(checkCredentialName)
}

// flagValue is the value following the last occurrence of flag in args.
func flagValue(args []string, flag string) (string, bool) {
	for i := len(args) - 2; i >= 0; i-- {
		if args[i] == flag {
			return args[i+1], true
		}
	}
	return "", false
}

// checkAutoMemoryOff: auto-memory is off both ways, in the environment and
// in the settings the session is given.
func checkAutoMemoryOff(inv invocation) check {
	if v, _ := lookup(inv.env, "CLAUDE_CODE_DISABLE_AUTO_MEMORY"); v != "1" {
		return fail(checkAutoMemory, "CLAUDE_CODE_DISABLE_AUTO_MEMORY is not 1")
	}
	s, ok := flagValue(inv.args, "--settings")
	if !ok {
		return fail(checkAutoMemory, "no --settings")
	}
	var settings struct {
		AutoMemoryEnabled *bool `json:"autoMemoryEnabled"`
	}
	if err := json.Unmarshal([]byte(s), &settings); err != nil || settings.AutoMemoryEnabled == nil || *settings.AutoMemoryEnabled {
		return fail(checkAutoMemory, "--settings does not set autoMemoryEnabled to false")
	}
	return pass(checkAutoMemory)
}

// checkProjectSettingsOnly: settings are read from the project only, never
// the user's or the local ones, and the session's HOME holds no settings
// before it starts.
func checkProjectSettingsOnly(inv invocation) check {
	if v, ok := flagValue(inv.args, "--setting-sources"); !ok || v != "project" {
		return fail(checkProjectSettings, "--setting-sources is %q, want project", v)
	}
	home, _ := lookup(inv.env, "HOME")
	if home == "" {
		return fail(checkProjectSettings, "HOME is not set")
	}
	if files, err := listFiles(home); err != nil || len(files) > 0 {
		return fail(checkProjectSettings, "the session's HOME is not empty: %s %v", paths(files), err)
	}
	return pass(checkProjectSettings)
}

// checkStrictMCP: the session loads no MCP server but those --mcp-config
// names, and it names none.
func checkStrictMCP(inv invocation) check {
	if !slices.Contains(inv.args, "--strict-mcp-config") {
		return fail(checkStrictMCPName, "no --strict-mcp-config")
	}
	if slices.Contains(inv.args, "--mcp-config") {
		return fail(checkStrictMCPName, "--mcp-config names a server")
	}
	return pass(checkStrictMCPName)
}

// checkNoSessionPersistence: the session is not saved to disk.
func checkNoSessionPersistence(inv invocation) check {
	if !slices.Contains(inv.args, "--no-session-persistence") {
		return fail(checkNoPersistence, "no --no-session-persistence")
	}
	return pass(checkNoPersistence)
}

// checkSessionReported: the transcript's opening says the session ran in
// the directory the runner made, under the harness version recorded.
func checkSessionReported(s session, version string, t transcript) check {
	switch {
	case !t.init.found:
		return fail(checkReported, "the transcript has no init event")
	case t.init.cwd != s.work:
		return fail(checkReported, "the session reports it ran in %s, not %s", t.init.cwd, s.work)
	case t.init.version == "" || !strings.HasPrefix(version, t.init.version):
		return fail(checkReported, "the session reports version %q, not the recorded %q", t.init.version, version)
	}
	return pass(checkReported)
}

// checkNoMCPServers: the transcript's opening lists no MCP server.
func checkNoMCPServers(t transcript) check {
	if !t.init.found || t.init.mcpServers != 0 {
		return fail(checkStrictMCPName, "the session reports %d MCP servers", t.init.mcpServers)
	}
	return pass(checkStrictMCPName)
}

// checkSkillAsDeclared: the session loaded metais's skill exactly when the
// run installed it.
func checkSkillAsDeclared(s session, t transcript) check {
	if loaded := slices.Contains(t.init.skills, "metais"); loaded != s.skill {
		return fail(checkSkill, "the metais skill loaded is %v, installed is %v", loaded, s.skill)
	}
	return pass(checkSkill)
}

// checkNoSessionSaved: after the session, its HOME holds no saved session.
func checkNoSessionSaved(home string) check {
	files, err := listFiles(home)
	if err != nil {
		return fail(checkNoPersistence, "listing HOME: %v", err)
	}
	for _, f := range files {
		if strings.HasPrefix(f.Path, ".claude/projects/") && strings.HasSuffix(f.Path, ".jsonl") {
			return fail(checkNoPersistence, "the session was saved as %s", f.Path)
		}
	}
	return pass(checkNoPersistence)
}

// memoryFile matches a reference to a memory file: an instructions file a
// harness loads as memory, an auto-memory index or directory, or a saved
// project's directory.
var memoryFile = regexp.MustCompile(`(?:^|[^A-Za-z0-9_])(?:CLAUDE(?:\.local)?\.md|MEMORY\.md|AGENTS\.md)|\.claude/(?:projects|memory)\b|/memory/`)

// checkNoMemoryRead: no tool call of the session names a memory file.
func checkNoMemoryRead(t transcript) check {
	for i, u := range t.toolUses {
		var input any
		if err := json.Unmarshal(u.input, &input); err != nil {
			continue
		}
		for _, s := range stringsIn(input) {
			if m := memoryFile.FindString(s); m != "" {
				return fail(checkNoMemory, "tool call %d (%s) names a memory file: %q", i+1, u.name, strings.TrimSpace(m))
			}
		}
	}
	return pass(checkNoMemory)
}

// stringsIn is every string in a decoded JSON value, keys included.
func stringsIn(v any) []string {
	switch v := v.(type) {
	case string:
		return []string{v}
	case []any:
		var out []string
		for _, e := range v {
			out = append(out, stringsIn(e)...)
		}
		return out
	case map[string]any:
		var out []string
		for k, e := range v {
			out = append(out, k)
			out = append(out, stringsIn(e)...)
		}
		return out
	}
	return nil
}

// changes lists the files of after against the fixtures before: each
// added, modified, unchanged or, from before, deleted.
func changes(before, after []fileEntry) []fileEntry {
	was := map[string]fileEntry{}
	for _, f := range before {
		was[f.Path] = f
	}
	var out []fileEntry
	for _, f := range after {
		prev, ok := was[f.Path]
		switch {
		case !ok:
			f.Change = "added"
		case prev.SHA256 != f.SHA256 || prev.Size != f.Size:
			f.Change = "modified"
		default:
			f.Change = "unchanged"
		}
		delete(was, f.Path)
		out = append(out, f)
	}
	for _, f := range before {
		if _, ok := was[f.Path]; ok {
			f.Change = "deleted"
			out = append(out, f)
		}
	}
	slices.SortFunc(out, func(a, b fileEntry) int { return strings.Compare(a.Path, b.Path) })
	return out
}

// valid reports whether every check passed.
func valid(checks []check) bool {
	return !slices.ContainsFunc(checks, func(c check) bool { return !c.Pass })
}
