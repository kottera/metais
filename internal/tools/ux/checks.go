// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"

	"metais.org/metais/internal/tools/ux/score"
	"metais.org/metais/internal/tools/ux/streamjson"
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
	checkNoMetais        = "metais-not-on-path"
	checkCredentialName  = "placeholder-credential"
	checkAutoMemory      = "auto-memory-off"
	checkProjectSettings = "project-settings-only"
	checkStrictMCPName   = "strict-mcp-config"
	checkNoPersistence   = "no-session-persistence"
	checkReported        = "session-as-started"
	checkSkill           = "skill-as-declared"
	checkNoMemory        = "no-memory-read"
	checkInsideRun       = "writes-inside-run"
	// checkCredentialNotSeen is ADR-009 A12's, made only with the
	// credential route on.
	checkCredentialNotSeen = "credential-not-seen"
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

// listFiles lists every file beneath dir, sorted by path. Anything but a
// regular file or a directory is listed with size -1 and no digest, so a
// comparison of two listings sees it.
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
	// A walk lists a directory's files before a sibling that sorts
	// between its name and its files' paths (a/b before a-c).
	slices.SortFunc(files, func(a, b fileEntry) int { return strings.Compare(a.Path, b.Path) })
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

// checkMetaisNotOnPath: an install scenario's session finds no metais on
// its PATH, so a metais it runs is one it installed (ADR-013 D9). PATH
// holds no relative directory, which would find a metais the session
// writes into its own directory.
func checkMetaisNotOnPath(inv invocation) check {
	path, _ := lookup(inv.env, "PATH")
	for _, dir := range filepath.SplitList(path) {
		if !filepath.IsAbs(dir) {
			return fail(checkNoMetais, "PATH holds the relative directory %q", dir)
		}
		if st, err := os.Stat(filepath.Join(dir, "metais")); err == nil && !st.IsDir() {
			return fail(checkNoMetais, "PATH finds %s: an install scenario starts with no metais on PATH", filepath.Join(dir, "metais"))
		}
	}
	return pass(checkNoMetais)
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
	case !t.Init.Found:
		return fail(checkReported, "the transcript has no init event")
	case t.Init.Cwd != s.work:
		return fail(checkReported, "the session reports it ran in %s, not %s", t.Init.Cwd, s.work)
	case t.Init.Version == "" || !strings.HasPrefix(version, t.Init.Version):
		return fail(checkReported, "the session reports version %q, not the recorded %q", t.Init.Version, version)
	}
	return pass(checkReported)
}

// checkNoMCPServers: the transcript's opening lists no MCP server.
func checkNoMCPServers(t transcript) check {
	if !t.Init.Found || t.Init.MCPServers != 0 {
		return fail(checkStrictMCPName, "the session reports %d MCP servers", t.Init.MCPServers)
	}
	return pass(checkStrictMCPName)
}

// checkSkillAsDeclared: the session loaded metais's skill exactly when the
// run installed it.
func checkSkillAsDeclared(s session, t transcript) check {
	if loaded := slices.Contains(t.Init.Skills, "metais"); loaded != s.skill {
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
var memoryFile = regexp.MustCompile(`(?:^|[\s/"'=~])(?:CLAUDE(?:\.local)?|MEMORY|AGENTS)\.md(?:$|[^A-Za-z0-9_-])|\.claude/(?:projects|memory)\b|/memory/`)

// spilledOutput matches a tool output Claude Code spilled to a file of the
// session's project beneath home, which a session reads back when a result
// is too large to show whole, or the directory of them. The session's HOME
// starts empty, so such a file is the session's own output and no memory.
// The project and the session may be globs, and the file a glob of its
// name: a project's directory begins with a dash, and a glob of the
// session's is of a session's id, so neither reaches the project's memory
// directory or a file beside it. A path that ends at the tool-results
// directory is the session's output too; one that leaves it is not.
func spilledOutput(home string) *regexp.Regexp {
	return regexp.MustCompile(`(?:` + regexp.QuoteMeta(home) + `|~|\$HOME|\$\{HOME\})/\.claude/projects/[-*][^/\s"'\\]*/` +
		`(?:[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}|[0-9a-f-]*\*[0-9a-f*-]*)/tool-results` +
		`(?:/[A-Za-z0-9_*-][A-Za-z0-9_.*-]*|/?(?:$|[\s"';&|)]))`)
}

// checkNoMemoryRead: no tool call of the session names a memory file. A
// tool output the session spilled to its own HOME is not one, and nor is
// a file of a repository the session cloned or created itself
// (ownRepositories): the session finding METAIS by cloning its
// repository reads that repository's CLAUDE.md as it reads its README,
// and no harness put it there. A name is excused only where the check
// can resolve it to a path inside such a repository; every other mention
// of a memory file, resolved or not, still fails.
func checkNoMemoryRead(s session, t transcript) check {
	spilled := spilledOutput(s.home)
	own := ownRepositories{session: s, cwd: s.work}
	for i, u := range t.ToolUses {
		var input any
		if err := json.Unmarshal(u.Input, &input); err != nil {
			continue
		}
		if u.Name == "Bash" {
			if m := own.bash(u, spilled.ReplaceAllString(u.Command(), " ")); m != "" {
				return fail(checkNoMemory, "tool call %d (%s) names a memory file: %q", i+1, u.Name, m)
			}
			continue
		}
		fields, _ := input.(map[string]any)
		keys := slices.Sorted(maps.Keys(fields))
		for _, k := range keys {
			v := fields[k]
			str, ok := v.(string)
			if !ok {
				for _, str := range stringsIn(v) {
					if m := memoryFile.FindString(spilled.ReplaceAllString(str, " ")); m != "" {
						return fail(checkNoMemory, "tool call %d (%s) names a memory file: %q", i+1, u.Name, strings.TrimSpace(m))
					}
				}
				continue
			}
			m := memoryFile.FindString(spilled.ReplaceAllString(str, " "))
			if m == "" || own.toolPath(fields, k, str) {
				continue
			}
			return fail(checkNoMemory, "tool call %d (%s) names a memory file: %q", i+1, u.Name, strings.TrimSpace(m))
		}
		for _, k := range keys {
			if memoryFile.MatchString(k) {
				return fail(checkNoMemory, "tool call %d (%s) names a memory file: %q", i+1, u.Name, k)
			}
		}
	}
	return pass(checkNoMemory)
}

// ownRepositories follows a session's Bash calls for the repositories it
// makes: each directory a git clone, gh repo clone or git init of its
// made, in a call the harness reports no error for. A clone succeeds
// only into a directory that is new or empty, so what is in it is the
// clone's. A directory a harness reads memory or instructions from is
// never one: the session's directory or one above it, its HOME or one
// above it, or anything beneath HOME's .claude.
//
// It follows the shell's directory as Claude Code keeps it from one Bash
// call to the next: each cd, and the harness's own notice when it puts
// the shell back ("Shell cwd was reset to …"). A cd it cannot resolve,
// through a variable or a glob, leaves the directory unknown, and a
// relative name is then resolved nowhere.
type ownRepositories struct {
	session session
	cwd     string
	dirs    []string
}

// cwdReset is the harness's notice that it put the shell's directory
// back.
var cwdReset = regexp.MustCompile(`(?m)Shell cwd was reset to (/\S*)`)

// bash reads one Bash call: it returns the first mention of a memory
// file in command that is not a path inside one of the session's own
// repositories, and records the repositories and the directory the call
// leaves.
func (o *ownRepositories) bash(u streamjson.ToolUse, command string) string {
	ok := u.Result != nil && !u.Result.IsError
	cwd := o.cwd
	var excused []string
	mention := ""
	for _, c := range score.Commands(command) {
		name, rest := c.Name()
		switch {
		case name == "cd" && len(c.Words) > 0 && c.Words[0] == "cd":
			cwd = o.cd(cwd, rest)
			continue
		case ok && name == "git", ok && name == "gh":
			if d := o.repository(cwd, name, rest); d != "" {
				o.dirs = append(o.dirs, d)
			}
		}
		for _, w := range slices.Concat(c.Words, c.Ins, c.Outs) {
			if !memoryFile.MatchString(w) {
				continue
			}
			if !o.inside(cwd, w) {
				if mention == "" {
					mention = strings.TrimSpace(memoryFile.FindString(w))
				}
				continue
			}
			excused = append(excused, w)
		}
	}
	o.cwd = cwd
	if u.Result != nil {
		if m := cwdReset.FindAllStringSubmatch(u.Result.Text, -1); len(m) > 0 {
			o.cwd = m[len(m)-1][1]
		}
	}
	if mention != "" {
		return mention
	}
	// A mention the parse does not read as a word, such as a
	// here-document's or a comment's, is not excused.
	for _, w := range excused {
		command = strings.ReplaceAll(command, w, " ")
	}
	return strings.TrimSpace(memoryFile.FindString(command))
}

// cd is the directory a cd of args leaves the shell in, from cwd, or ""
// when it cannot be told.
func (o *ownRepositories) cd(cwd string, args []string) string {
	ops := slices.DeleteFunc(slices.Clone(args), func(a string) bool { return a == "-P" || a == "-L" || a == "--" })
	if len(ops) == 0 {
		return o.session.home
	}
	p, ok := o.resolve(cwd, ops[0])
	if !ok {
		return ""
	}
	return p
}

// gitValued are git clone's options that take the next word as their
// value.
var gitValued = []string{"-b", "--branch", "-o", "--origin", "-c", "--config", "--depth", "--reference",
	"--reference-if-able", "--separate-git-dir", "-j", "--jobs", "--filter", "--shallow-since",
	"--shallow-exclude", "--template", "-u", "--upload-pack", "--server-option", "--bundle-uri"}

// repository is the directory a git clone, gh repo clone or git init
// makes, from cwd, or "" for any other command or one the check cannot
// resolve.
func (o *ownRepositories) repository(cwd, name string, args []string) string {
	if name == "git" {
		for len(args) > 1 && (args[0] == "-C" || args[0] == "-c") {
			if args[0] == "-C" {
				p, ok := o.resolve(cwd, args[1])
				if !ok {
					return ""
				}
				cwd = p
			}
			args = args[2:]
		}
	}
	var ops []string
	switch {
	case name == "git" && len(args) > 0 && (args[0] == "clone" || args[0] == "init"),
		name == "gh" && len(args) > 1 && args[0] == "repo" && args[1] == "clone":
		if name == "gh" {
			args = args[1:]
		}
		for i := 1; i < len(args); i++ {
			switch a := args[i]; {
			case a == "--":
				if name == "gh" {
					// What follows gh's -- is git clone's own flags.
					i = len(args)
					continue
				}
				ops = append(ops, args[i+1:]...)
				i = len(args)
			case slices.Contains(gitValued, a):
				i++
			case !strings.HasPrefix(a, "-"):
				ops = append(ops, a)
			}
		}
	default:
		return ""
	}
	var dir string
	switch {
	case args[0] == "init" && len(ops) == 0:
		dir = "."
	case args[0] == "init" || len(ops) > 1:
		dir = ops[len(ops)-1]
	case len(ops) == 1:
		// The repository's last path element, without .git.
		src := strings.TrimRight(ops[0], "/")
		dir = strings.TrimSuffix(src[strings.LastIndexAny(src, "/:")+1:], ".git")
	default:
		return ""
	}
	p, ok := o.resolve(cwd, dir)
	if !ok || p == "/" {
		return ""
	}
	home, work := o.session.home, o.session.work
	within := func(p, dir string) bool { return p == dir || strings.HasPrefix(p, dir+"/") }
	if within(work, p) || within(home, p) || within(p, filepath.Join(home, ".claude")) {
		return ""
	}
	return p
}

// inside reports whether w, a word of a command run in cwd, is a path
// inside one of the session's own repositories.
func (o *ownRepositories) inside(cwd, w string) bool {
	if strings.HasPrefix(w, "-") {
		_, w, _ = strings.Cut(w, "=")
	}
	p, ok := o.resolve(cwd, w)
	if !ok {
		return false
	}
	return slices.ContainsFunc(o.dirs, func(d string) bool { return strings.HasPrefix(p, d+"/") })
}

// toolPath reports whether a tool's field k, holding str, names a path
// inside one of the session's own repositories: a file_path, path or
// notebook_path, or a Glob's pattern beneath its path. A tool resolves a
// relative path from the session's directory.
func (o *ownRepositories) toolPath(fields map[string]any, k, str string) bool {
	switch k {
	case "file_path", "path", "notebook_path":
		return o.inside(o.session.work, str)
	case "pattern":
		dir, _ := fields["path"].(string)
		if _, glob := fields["glob"]; dir == "" || glob {
			return false
		}
		if strings.HasPrefix(str, "**/") {
			str = str[3:]
		}
		return !strings.ContainsAny(dir, "*?[") && o.inside(o.session.work, filepath.Join(dir, str))
	}
	return false
}

// resolve is p as an absolute, clean path, from cwd: ~ and $HOME are the
// session's HOME. A path through any other variable, a substitution or a
// glob, or a relative one from an unknown directory, is not resolved.
func (o *ownRepositories) resolve(cwd, p string) (string, bool) {
	for _, h := range []string{"~", "$HOME", "${HOME}"} {
		if p == h || strings.HasPrefix(p, h+"/") {
			p = o.session.home + p[len(h):]
			break
		}
	}
	switch {
	case p == "" || strings.ContainsAny(p, "$`*?[{"):
		return "", false
	case filepath.IsAbs(p):
		return filepath.Clean(p), true
	case cwd == "":
		return "", false
	}
	return filepath.Join(cwd, p), true
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

// outsideRun lists the files of the set's directory out that are not in
// the run's own directory, runDir beneath it.
func outsideRun(out, runDir string) ([]fileEntry, error) {
	files, err := listFiles(out)
	rel := strings.TrimPrefix(filepath.ToSlash(strings.TrimPrefix(runDir, out)), "/") + "/"
	return slices.DeleteFunc(files, func(f fileEntry) bool { return strings.HasPrefix(f.Path, rel) }), err
}

// writeTools are the tools that write the file their input names.
var writeTools = []string{"Write", "Edit", "MultiEdit", "NotebookEdit"}

// checkWritesInsideRun: the session wrote nothing outside its own run
// directory (Ed's ruling, 2026-10-21). The set's directory outside the
// run's is listed before the session and after it, and must be as it
// was; and no Write or Edit tool call names a file outside the run's
// directory. A command line that writes outside the set's directory is
// not seen: the dispatch's sandbox refuses most such paths.
func checkWritesInsideRun(runDir, work string, before, after []fileEntry, t transcript) check {
	for _, f := range changes(before, after) {
		if f.Change != "unchanged" {
			return fail(checkInsideRun, "the set's %s is %s outside the run's directory", f.Path, f.Change)
		}
	}
	for i, u := range t.ToolUses {
		if !slices.Contains(writeTools, u.Name) {
			continue
		}
		p := u.Field("file_path")
		if p == "" {
			p = u.Field("notebook_path")
		}
		if p == "" {
			continue
		}
		if !filepath.IsAbs(p) {
			p = filepath.Join(work, p)
		}
		if p = filepath.Clean(p); p != runDir && !strings.HasPrefix(p, runDir+string(filepath.Separator)) {
			return fail(checkInsideRun, "tool call %d (%s) writes %s, outside the run's directory", i+1, u.Name, p)
		}
	}
	return pass(checkInsideRun)
}

// valid reports whether every check passed.
func valid(checks []check) bool {
	return !slices.ContainsFunc(checks, func(c check) bool { return !c.Pass })
}
