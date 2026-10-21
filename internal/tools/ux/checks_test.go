// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// testSession is a session in fresh directories under the test's own, and
// a claude harness that is never run.
func testSession(t *testing.T) (*claudeCode, session) {
	t.Helper()
	dir := testenv.Dir(t)
	s := session{
		work:   filepath.Join(dir, "work"),
		home:   filepath.Join(dir, "home"),
		tmp:    filepath.Join(dir, "tmp"),
		bin:    filepath.Join(dir, "bin"),
		model:  "haiku",
		prompt: "render this file: report.metais",
	}
	for _, d := range []string{s.work, s.home, s.tmp, s.bin} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	return &claudeCode{hostPath: "/usr/bin:/bin", baseURL: "http://127.0.0.1:9", binary: "/opt/claude/bin/claude"}, s
}

// without is env with every variable named name removed.
func without(env []string, name string) []string {
	return slices.DeleteFunc(slices.Clone(env), func(kv string) bool { return strings.HasPrefix(kv, name+"=") })
}

// withoutArg is args with arg, and n values after it, removed.
func withoutArg(args []string, arg string, n int) []string {
	i := slices.Index(args, arg)
	if i < 0 {
		return args
	}
	return slices.Concat(args[:i], args[i+1+n:])
}

// replaceArg is args with the value after arg replaced.
func replaceArg(args []string, arg, value string) []string {
	out := slices.Clone(args)
	out[slices.Index(out, arg)+1] = value
	return out
}

// TestHarnessChecks: the invocation the Claude Code harness builds passes
// every one of its checks, and each check fails on an invocation that
// lacks what it checks.
func TestHarnessChecks(t *testing.T) {
	h, s := testSession(t)
	inv := h.invocation(s)
	for _, c := range h.checks(inv) {
		if !c.Pass {
			t.Errorf("the harness's own invocation fails %s: %s", c.Name, c.Detail)
		}
	}
	cases := []struct {
		name  string
		check string
		edit  func(inv *invocation)
	}{
		{"no auto-memory variable", checkAutoMemory, func(inv *invocation) { inv.env = without(inv.env, "CLAUDE_CODE_DISABLE_AUTO_MEMORY") }},
		{"the auto-memory variable 0", checkAutoMemory, func(inv *invocation) { inv.env = append(inv.env, "CLAUDE_CODE_DISABLE_AUTO_MEMORY=0") }},
		{"no settings", checkAutoMemory, func(inv *invocation) { inv.args = withoutArg(inv.args, "--settings", 1) }},
		{"auto-memory on in settings", checkAutoMemory, func(inv *invocation) {
			inv.args = replaceArg(inv.args, "--settings", `{"autoMemoryEnabled":true}`)
		}},
		{"settings silent on auto-memory", checkAutoMemory, func(inv *invocation) { inv.args = replaceArg(inv.args, "--settings", `{}`) }},
		{"user settings too", checkProjectSettings, func(inv *invocation) { inv.args = replaceArg(inv.args, "--setting-sources", "user,project") }},
		{"no setting sources", checkProjectSettings, func(inv *invocation) { inv.args = withoutArg(inv.args, "--setting-sources", 1) }},
		{"settings already in HOME", checkProjectSettings, func(inv *invocation) {
			home, _ := lookup(inv.env, "HOME")
			if err := os.MkdirAll(filepath.Join(home, ".claude"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(home, ".claude", "settings.json"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
		}},
		{"no strict MCP config", checkStrictMCPName, func(inv *invocation) { inv.args = withoutArg(inv.args, "--strict-mcp-config", 0) }},
		{"an MCP config", checkStrictMCPName, func(inv *invocation) { inv.args = append(inv.args, "--mcp-config", `{"mcpServers":{}}`) }},
		{"session persistence", checkNoPersistence, func(inv *invocation) { inv.args = withoutArg(inv.args, "--no-session-persistence", 0) }},
		{"no placeholder token", checkCredentialName, func(inv *invocation) { inv.env = without(inv.env, "ANTHROPIC_AUTH_TOKEN") }},
		{"a real token", checkCredentialName, func(inv *invocation) { inv.env = append(inv.env, "ANTHROPIC_AUTH_TOKEN=sk-real") }},
		{"an API key", checkCredentialName, func(inv *invocation) { inv.env = append(inv.env, "ANTHROPIC_API_KEY=sk-real") }},
		{"an OAuth token", checkCredentialName, func(inv *invocation) { inv.env = append(inv.env, "CLAUDE_CODE_OAUTH_TOKEN=x") }},
	}
	for _, c := range cases {
		_, s := testSession(t)
		inv := h.invocation(s)
		c.edit(&inv)
		got := h.checks(inv)
		i := slices.IndexFunc(got, func(ch check) bool { return ch.Name == c.check })
		if i < 0 || got[i].Pass {
			t.Errorf("%s: %s passed", c.name, c.check)
		}
		for _, ch := range got {
			if ch.Name != c.check && !ch.Pass {
				t.Errorf("%s: %s failed too: %s", c.name, ch.Name, ch.Detail)
			}
		}
	}
}

func TestCheckFreshDirectory(t *testing.T) {
	dir := testenv.Dir(t)
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("report.metais", "package")
	want := []fileEntry{{Path: "report.metais", Size: 7, SHA256: digest([]byte("package"))}}
	if c := checkFreshDirectory(dir, want); !c.Pass {
		t.Fatalf("only the fixture: %s", c.Detail)
	}
	write("report.metais", "changed")
	if c := checkFreshDirectory(dir, want); c.Pass {
		t.Error("a changed fixture passed")
	}
	write("report.metais", "package")
	write(".claude/settings.json", "{}")
	if c := checkFreshDirectory(dir, want); c.Pass || !strings.Contains(c.Detail, ".claude/settings.json") {
		t.Errorf("a settings file beside the fixture: %v", c)
	}
	if c := checkFreshDirectory(dir, nil); c.Pass {
		t.Error("a directory that should be empty passed")
	}
	if err := os.Symlink("report.metais", filepath.Join(dir, "link")); err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(dir, ".claude")); err != nil {
		t.Fatal(err)
	}
	if c := checkFreshDirectory(dir, want); c.Pass {
		t.Error("a symbolic link beside the fixture passed")
	}
}

func TestCheckOutsideRepository(t *testing.T) {
	dir := testenv.Dir(t)
	work := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(work, 0o755); err != nil {
		t.Fatal(err)
	}
	if c := checkOutsideRepository(work); !c.Pass {
		t.Fatalf("a directory outside any repository: %s", c.Detail)
	}
	if c := checkOutsideRepository("a/b"); c.Pass {
		t.Error("a relative directory passed")
	}
	for _, marker := range []string{".git", ".hg", ".jj"} {
		p := filepath.Join(dir, "a", marker)
		if err := os.WriteFile(p, []byte("gitdir: x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if c := checkOutsideRepository(work); c.Pass || !strings.Contains(c.Detail, filepath.Join(dir, "a")) {
			t.Errorf("a directory beneath a %s: %v", marker, c)
		}
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCheckMetaisFirstOnPath(t *testing.T) {
	dir := testenv.Dir(t)
	bin, other := filepath.Join(dir, "bin"), filepath.Join(dir, "other")
	for _, d := range []string{bin, other} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	under := []byte("metais under test")
	if err := os.WriteFile(filepath.Join(bin, "metais"), under, 0o755); err != nil {
		t.Fatal(err)
	}
	sha := digest(under)
	path := func(dirs ...string) invocation {
		return invocation{env: []string{"PATH=" + strings.Join(dirs, string(filepath.ListSeparator))}}
	}
	if c := checkMetaisFirstOnPath(path(bin, other), bin, sha); !c.Pass {
		t.Fatalf("the metais under test first: %s", c.Detail)
	}
	if err := os.WriteFile(filepath.Join(other, "metais"), []byte("another metais"), 0o755); err != nil {
		t.Fatal(err)
	}
	if c := checkMetaisFirstOnPath(path(other, bin), bin, sha); c.Pass {
		t.Error("another metais first passed")
	}
	if c := checkMetaisFirstOnPath(path(bin, other), bin, digest([]byte("another commit"))); c.Pass {
		t.Error("a metais of another commit passed")
	}
	if c := checkMetaisFirstOnPath(path(".", bin), bin, sha); c.Pass {
		t.Error("a relative directory first passed")
	}
	if c := checkMetaisFirstOnPath(path(filepath.Join(dir, "none")), bin, sha); c.Pass {
		t.Error("no metais passed")
	}
	if err := os.WriteFile(filepath.Join(dir, "plain", "metais"), nil, 0o644); err == nil {
		t.Fatal("wrote beneath a missing directory")
	}
}

// initLine is a transcript's opening for s under version.
func initLine(s session, version string, mcp, skills string) string {
	return `{"type":"system","subtype":"init","cwd":"` + s.work + `","model":"claude-haiku-5-5","claude_code_version":"` +
		version + `","mcp_servers":` + mcp + `,"skills":` + skills + `}`
}

const resultLine = `{"type":"result","subtype":"success","is_error":false,"num_turns":3,"total_cost_usd":0.01,"duration_ms":1200,"result":"ok","modelUsage":{"claude-haiku-5-5":{}}}`

func toolLine(name, input string) string {
	return `{"type":"assistant","message":{"content":[{"type":"tool_use","name":"` + name + `","input":` + input + `}]}}`
}

// TestAfterChecks: a clean transcript passes every check of a finished
// session, and each check fails on a transcript or HOME that shows what it
// checks for.
func TestAfterChecks(t *testing.T) {
	const version = "2.1.299 (Claude Code)"
	clean := func(s session) string {
		return strings.Join([]string{initLine(s, "2.1.299", "[]", `["debug"]`),
			toolLine("Bash", `{"command":"metais inspect report.metais"}`), resultLine}, "\n")
	}
	h, s := testSession(t)
	for _, c := range h.after(s, version, h.read([]byte(clean(s)))) {
		if !c.Pass {
			t.Errorf("a clean session fails %s: %s", c.Name, c.Detail)
		}
	}
	cases := []struct {
		name  string
		check string
		edit  func(s *session) string
	}{
		{"another directory", checkReported, func(s *session) string {
			return strings.Replace(clean(*s), s.work, "/elsewhere", 1)
		}},
		{"another version", checkReported, func(s *session) string { return strings.Replace(clean(*s), "2.1.299", "2.0.0", 1) }},
		{"no init", checkReported, func(s *session) string { return resultLine }},
		{"an MCP server", checkStrictMCPName, func(s *session) string {
			return strings.Replace(clean(*s), `"mcp_servers":[]`, `"mcp_servers":[{"name":"x","status":"connected"}]`, 1)
		}},
		{"the skill loaded uninstalled", checkSkill, func(s *session) string {
			return strings.Replace(clean(*s), `["debug"]`, `["debug","metais"]`, 1)
		}},
		{"the skill installed and not loaded", checkSkill, func(s *session) string { s.skill = true; return clean(*s) }},
		{"a saved session", checkNoPersistence, func(s *session) string {
			p := filepath.Join(s.home, ".claude", "projects", "-work")
			if err := os.MkdirAll(p, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(p, "6a19.jsonl"), []byte("{}"), 0o644); err != nil {
				t.Fatal(err)
			}
			return clean(*s)
		}},
		{"a CLAUDE.md read", checkNoMemory, func(s *session) string {
			return clean(*s) + "\n" + toolLine("Read", `{"file_path":"`+s.home+`/.claude/CLAUDE.md"}`)
		}},
		{"a memory directory listed", checkNoMemory, func(s *session) string {
			return clean(*s) + "\n" + toolLine("Bash", `{"command":"ls ~/.claude/projects/-work/memory/"}`)
		}},
		{"a MEMORY.md searched", checkNoMemory, func(s *session) string {
			return clean(*s) + "\n" + toolLine("Grep", `{"pattern":"x","path":"MEMORY.md"}`)
		}},
		{"an AGENTS.md read", checkNoMemory, func(s *session) string {
			return clean(*s) + "\n" + toolLine("Read", `{"file_path":"../AGENTS.md"}`)
		}},
	}
	for _, c := range cases {
		_, s := testSession(t)
		got := h.after(s, version, h.read([]byte(c.edit(&s))))
		i := slices.IndexFunc(got, func(ch check) bool { return ch.Name == c.check })
		if i < 0 || got[i].Pass {
			t.Errorf("%s: %s passed", c.name, c.check)
		}
	}
	// A name that merely holds a memory file's name is no memory read.
	_, s = testSession(t)
	tr := h.read([]byte(clean(s) + "\n" + toolLine("Write", `{"file_path":"report-CLAUDE.mdx","content":"x"}`)))
	if c := checkNoMemoryRead(tr); !c.Pass {
		t.Errorf("report-CLAUDE.mdx: %s", c.Detail)
	}
}

func TestRead(t *testing.T) {
	h, s := testSession(t)
	tr := h.read([]byte(strings.Join([]string{initLine(s, "2.1.299", "[]", "[]"), "not json",
		toolLine("Bash", `{"command":"metais inspect report.metais"}`), resultLine, ""}, "\n")))
	if !tr.init.found || tr.init.model != "claude-haiku-5-5" || tr.init.cwd != s.work {
		t.Errorf("init = %+v", tr.init)
	}
	if !tr.result.found || tr.result.turns != 3 || tr.result.costUSD != 0.01 || tr.result.duration != 1200 ||
		!slices.Equal(tr.result.models, []string{"claude-haiku-5-5"}) {
		t.Errorf("result = %+v", tr.result)
	}
	if len(tr.toolUses) != 1 || tr.toolUses[0].name != "Bash" || tr.malformed != 1 {
		t.Errorf("tool uses %v, malformed %d", tr.toolUses, tr.malformed)
	}
}

func TestQuotaPause(t *testing.T) {
	h := &claudeCode{}
	for _, c := range []struct {
		name, transcript, stderr string
		want                     bool
	}{
		{"a finished session", resultLine, "", false},
		{"a finished session that mentions a quota", strings.Replace(resultLine, `"ok"`, `"the quota is 3"`, 1), "", false},
		{"a 429", `{"type":"result","is_error":true,"api_error_status":429,"result":"API Error"}`, "", true},
		{"a usage limit", `{"type":"result","is_error":true,"result":"Claude usage limit reached"}`, "", true},
		{"no result, a rate limit on stderr", "", "Error: rate limit exceeded", true},
		{"no result, another error", "", "Error: ENOENT", false},
		{"an error result", `{"type":"result","is_error":true,"result":"tool failed"}`, "", false},
	} {
		if got := quotaPause(h.read([]byte(c.transcript)), []byte(c.stderr)); got != c.want {
			t.Errorf("%s: quotaPause = %v, want %v", c.name, got, c.want)
		}
	}
}

func TestChanges(t *testing.T) {
	before := []fileEntry{{Path: "a.metais", Size: 1, SHA256: "1"}, {Path: "b.metais", Size: 1, SHA256: "2"}}
	after := []fileEntry{{Path: "a.metais", Size: 1, SHA256: "1"}, {Path: "out/report.html", Size: 3, SHA256: "3"}}
	got := changes(before, after)
	want := []string{"a.metais unchanged", "b.metais deleted", "out/report.html added"}
	var g []string
	for _, f := range got {
		g = append(g, f.Path+" "+f.Change)
	}
	if !slices.Equal(g, want) {
		t.Errorf("changes = %q, want %q", g, want)
	}
	after[0].SHA256 = "9"
	if got := changes(before, after); got[0].Change != "modified" {
		t.Errorf("a changed fixture is %s", got[0].Change)
	}
}
