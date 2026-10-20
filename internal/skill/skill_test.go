// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package skill

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/hostfs"
	"github.com/kottera/metais/internal/testenv"
)

const v1, v2 = "1.2.3", "1.3.0"

// install runs Install over dir with the harnesses named, failing t on an
// error.
func install(t *testing.T, dir string, opt Options, harnesses ...string) Result {
	t.Helper()
	opt.Harnesses = harnesses
	if opt.Version == "" {
		opt.Version = v1
	}
	res, err := Install(dir, opt)
	if err != nil {
		t.Fatalf("Install: %v", err)
	}
	return res
}

// mkdirs creates each directory beneath dir.
func mkdirs(t *testing.T, dir string, names ...string) {
	t.Helper()
	for _, n := range names {
		if err := os.MkdirAll(filepath.Join(dir, n), 0o755); err != nil {
			t.Fatal(err)
		}
	}
}

// put writes b at name beneath dir, creating its directories.
func put(t *testing.T, dir, name string, b []byte) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

// read returns the bytes at name beneath dir, or nil when there is none.
func read(t *testing.T, dir, name string) []byte {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// files lists every regular file beneath dir, relative and "/"-separated.
func files(t *testing.T, dir string) []string {
	t.Helper()
	var out []string
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() {
			rel, _ := filepath.Rel(dir, p)
			out = append(out, filepath.ToSlash(rel))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

const (
	claudeFile = ".claude/skills/metais/SKILL.md"
	agentsFile = ".agents/skills/metais/SKILL.md"
)

// TestHarnessPaths is ADR-008 A2 item 3's table: each harness writes the
// documented path, and nothing else.
func TestHarnessPaths(t *testing.T) {
	for h, want := range map[string]string{
		"claude":   claudeFile,
		"codex":    agentsFile,
		"gemini":   agentsFile,
		"cursor":   agentsFile,
		"opencode": agentsFile,
	} {
		t.Run(h, func(t *testing.T) {
			dir := testenv.Dir(t)
			res := install(t, dir, Options{}, h)
			if len(res.Targets) != 1 || res.Targets[0].Path != want || res.Targets[0].Action != Installed {
				t.Fatalf("targets %+v, want one %s installed", res.Targets, want)
			}
			if got := files(t, dir); !slices.Equal(got, []string{want}) {
				t.Errorf("wrote %q, want %q", got, want)
			}
			if !bytes.Equal(read(t, dir, want), Text(v1)) {
				t.Errorf("%s is not the skill's text", want)
			}
			if res.Chosen != "flag" || res.Targets[0].Because != "flag" || !slices.Equal(res.Targets[0].Harnesses, []string{h}) {
				t.Errorf("chosen %q, target %+v", res.Chosen, res.Targets[0])
			}
		})
	}
}

// TestSharedDirectoryOnce is item 3's first rule: harnesses that share a
// directory write it once, and both directories only when both are named.
func TestSharedDirectoryOnce(t *testing.T) {
	dir := testenv.Dir(t)
	res := install(t, dir, Options{}, "codex", "gemini", "codex")
	if len(res.Targets) != 1 || !slices.Equal(res.Targets[0].Harnesses, []string{"codex", "gemini"}) {
		t.Fatalf("targets %+v", res.Targets)
	}
	dir = testenv.Dir(t)
	res = install(t, dir, Options{}, "codex", "claude")
	if got := files(t, dir); !slices.Equal(got, []string{agentsFile, claudeFile}) {
		t.Errorf("wrote %q", got)
	}
	if res.Targets[0].Path != claudeFile || res.Targets[1].Path != agentsFile {
		t.Errorf("targets out of order: %+v", res.Targets)
	}
}

// TestDetection is item 3's detection: exactly one harness directory
// chooses, and neither or both is a refusal that writes nothing.
func TestDetection(t *testing.T) {
	for _, c := range []struct {
		name  string
		dirs  []string
		want  string // the path written, or ""
		found string // NeedHarnessError's Found, or ""
	}{
		{"claude", []string{".claude"}, claudeFile, ""},
		{"agents", []string{".agents"}, agentsFile, ""},
		{"neither", nil, "", "neither"},
		{"both", []string{".claude", ".agents"}, "", "both"},
	} {
		t.Run(c.name, func(t *testing.T) {
			dir := testenv.Dir(t)
			mkdirs(t, dir, c.dirs...)
			res, err := Install(dir, Options{Version: v1})
			var need *NeedHarnessError
			if c.found != "" {
				if !errors.As(err, &need) || need.Found != c.found {
					t.Fatalf("err %v, want a NeedHarnessError of %s", err, c.found)
				}
				if got := files(t, dir); len(got) != 0 {
					t.Errorf("wrote %q", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			top := c.dirs[0]
			if res.Chosen != "detected" || res.Targets[0].Because != top+"/ exists" || res.Targets[0].Path != c.want {
				t.Errorf("result %+v", res)
			}
			if top == ".agents" && !slices.Equal(res.Targets[0].Harnesses, []string{"codex", "gemini", "cursor", "opencode"}) {
				t.Errorf("harnesses %q", res.Targets[0].Harnesses)
			}
		})
	}
	// A file named .claude is no harness directory.
	dir := testenv.Dir(t)
	put(t, dir, ".claude", []byte("x"))
	if _, err := Install(dir, Options{Version: v1}); !errors.As(err, new(*NeedHarnessError)) {
		t.Errorf("a file .claude: err %v", err)
	}
}

// TestScope is item 4: the skill goes beneath the directory named, and
// nothing is written outside it.
func TestScope(t *testing.T) {
	root := testenv.Dir(t)
	project := filepath.Join(root, "project")
	mkdirs(t, root, "project")
	install(t, project, Options{}, "claude")
	if got := files(t, root); !slices.Equal(got, []string{"project/" + claudeFile}) {
		t.Errorf("wrote %q", got)
	}
}

// TestRepository is item 4's warning's input: whether the directory holds
// a .git entry, of any kind.
func TestRepository(t *testing.T) {
	dir := testenv.Dir(t)
	if install(t, dir, Options{}, "claude").Repository {
		t.Error("no .git, and Repository")
	}
	put(t, dir, ".git", []byte("gitdir: elsewhere\n"))
	if !install(t, dir, Options{}, "claude").Repository {
		t.Error("a .git file, and no Repository")
	}
}

// TestRefresh is item 5's refresh rule: a copy metais wrote is unchanged
// at the same version and replaced at another.
func TestRefresh(t *testing.T) {
	dir := testenv.Dir(t)
	install(t, dir, Options{}, "claude")
	before, err := os.Stat(filepath.Join(dir, claudeFile))
	if err != nil {
		t.Fatal(err)
	}
	res := install(t, dir, Options{}, "claude")
	if a := res.Targets[0]; a.Action != Unchanged || a.From != v1 {
		t.Errorf("again: %+v", a)
	}
	after, err := os.Stat(filepath.Join(dir, claudeFile))
	if err != nil || !os.SameFile(before, after) {
		t.Error("an unchanged copy was replaced")
	}
	res = install(t, dir, Options{Version: v2}, "claude")
	if a := res.Targets[0]; a.Action != Updated || a.From != v1 {
		t.Errorf("new version: %+v", a)
	}
	if !bytes.Equal(read(t, dir, claudeFile), Text(v2)) {
		t.Error("not refreshed to the new version")
	}
	// Another version's text, stamped as metais stamps it, is metais's own
	// and is updated, even downward.
	old := append([]byte("# an older skill\n"), stamp("0.9.0", []byte("# an older skill\n"))...)
	put(t, dir, claudeFile, old)
	if a := install(t, dir, Options{}, "claude").Targets[0]; a.Action != Updated || a.From != "0.9.0" {
		t.Errorf("older text: %+v", a)
	}
	if got := files(t, dir); !slices.Equal(got, []string{claudeFile}) {
		t.Errorf("left %q", got)
	}
}

// TestHandEdits is item 5's hand-edit rule: anything but metais's own
// unedited copy is left as it is, unless Overwrite.
func TestHandEdits(t *testing.T) {
	text := Text(v1)
	edited := bytes.Replace(text, []byte("# metais"), []byte("# metais, ours"), 1)
	forged := regexp.MustCompile(`sha256 [0-9a-f]{64}`).ReplaceAll(text, []byte("sha256 "+strings.Repeat("0", 64)))
	big := append(bytes.Repeat([]byte("x"), MaxExisting), text...)
	exact := append(bytes.Repeat([]byte("y"), MaxExisting-len(stamp(v1, nil))), stamp(v1, []byte("other"))...)
	control := append([]byte("# x\n"), []byte("<!-- metais skill: version 1.0\x1b[31m, sha256 "+strings.Repeat("0", 64)+" -->\n")...)
	for name, b := range map[string][]byte{
		"an edited copy":               edited,
		"a copy whose stamp is forged": forged,
		"a file with no stamp":         []byte("# Our own skill\n"),
		"an empty file":                {},
		"a stamp not on the last line": append(slices.Clone(text), "more\n"...),
		"a file over the bound":        big,
		"a file at the bound, forged":  exact,
		"a stamp holding a control":    control,
	} {
		t.Run(name, func(t *testing.T) {
			dir := testenv.Dir(t)
			put(t, dir, claudeFile, b)
			a := install(t, dir, Options{}, "claude").Targets[0]
			if a.Action != Edited || a.From != "" || a.Err != nil {
				t.Fatalf("target %+v, want edited", a)
			}
			if !bytes.Equal(read(t, dir, claudeFile), b) {
				t.Fatal("an edited copy was changed")
			}
			if a := install(t, dir, Options{DryRun: true, Overwrite: true}, "claude").Targets[0]; a.Action != WouldReplace {
				t.Errorf("dry run with overwrite: %+v", a)
			}
			a = install(t, dir, Options{Overwrite: true}, "claude").Targets[0]
			if a.Action != Replaced || !bytes.Equal(read(t, dir, claudeFile), text) {
				t.Errorf("overwrite: %+v", a)
			}
		})
	}
}

// TestDryRun writes nothing, and reports what each target needs.
func TestDryRun(t *testing.T) {
	dir := testenv.Dir(t)
	if a := install(t, dir, Options{DryRun: true}, "claude").Targets[0]; a.Action != WouldInstall {
		t.Errorf("empty: %+v", a)
	}
	if got := files(t, dir); len(got) != 0 {
		t.Fatalf("wrote %q", got)
	}
	if entries, _ := os.ReadDir(dir); len(entries) != 0 {
		t.Fatalf("created %v", entries)
	}
	install(t, dir, Options{}, "claude")
	if a := install(t, dir, Options{DryRun: true, Version: v2}, "claude").Targets[0]; a.Action != WouldUpdate {
		t.Errorf("another version: %+v", a)
	}
	if !bytes.Equal(read(t, dir, claudeFile), Text(v1)) {
		t.Error("a dry run wrote")
	}
}

// TestHostile is ADR-009 A5 item 5's hostile cases: no link is written
// through, nothing but a regular file is replaced, and a refused target
// does not stop the other.
func TestHostile(t *testing.T) {
	type hostile struct {
		name   string
		posix  bool
		setup  func(t *testing.T, dir, outside string)
		reason hostfs.Reason
	}
	link := func(target, name string) func(t *testing.T, dir, outside string) {
		return func(t *testing.T, dir, outside string) {
			if target == "" {
				target = outside
			}
			p := filepath.Join(dir, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(target, p); err != nil {
				t.Fatal(err)
			}
		}
	}
	for _, c := range []hostile{
		{"a .claude link pointing out", false, link("", ".claude"), hostfs.Link},
		{"a .claude link pointing in", false, func(t *testing.T, dir, outside string) {
			mkdirs(t, dir, "elsewhere")
			link("elsewhere", ".claude")(t, dir, outside)
		}, hostfs.Link},
		{"a link at the skill's directory", false, link("", ".claude/skills/metais"), hostfs.Link},
		{"a file at the skill's directory", false, func(t *testing.T, dir, _ string) {
			put(t, dir, ".claude/skills/metais", []byte("x"))
		}, hostfs.NotDirectory},
		{"a link at SKILL.md", false, func(t *testing.T, dir, outside string) {
			put(t, outside, "SKILL.md", Text(v1))
			link(filepath.Join(outside, "SKILL.md"), claudeFile)(t, dir, outside)
		}, hostfs.Link},
		{"a directory at SKILL.md", false, func(t *testing.T, dir, _ string) {
			mkdirs(t, dir, claudeFile)
		}, hostfs.IsDirectory},
		{"a FIFO at SKILL.md", true, func(t *testing.T, dir, _ string) {
			mkdirs(t, dir, ".claude/skills/metais")
			mkfifo(t, filepath.Join(dir, filepath.FromSlash(claudeFile)))
		}, hostfs.NotRegularFile},
		{"a directory that cannot be written", true, func(t *testing.T, dir, _ string) {
			mkdirs(t, dir, ".claude/skills")
			locked := filepath.Join(dir, ".claude", "skills")
			if err := os.Chmod(locked, 0o555); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { os.Chmod(locked, 0o755) })
		}, hostfs.PermissionDenied},
	} {
		t.Run(c.name, func(t *testing.T) {
			if c.posix && (runtime.GOOS == "windows" || os.Geteuid() == 0) {
				t.Skip("needs a POSIX host and a user other than root")
			}
			dir, outside := testenv.Dir(t), testenv.Dir(t)
			c.setup(t, dir, outside)
			before := files(t, outside)
			res := install(t, dir, Options{Overwrite: true}, "claude", "codex")
			cl, ag := res.Targets[0], res.Targets[1]
			if r := hostfs.ReasonOf(cl.Err); r != c.reason || cl.Action != "" {
				t.Errorf("claude target: reason %q, action %q, want %q", r, cl.Action, c.reason)
			}
			if ag.Err != nil || ag.Action != Installed {
				t.Errorf("the other target: %+v", ag)
			}
			if got := files(t, outside); !slices.Equal(got, before) {
				t.Errorf("outside the directory: %q, was %q", got, before)
			}
			for _, f := range files(t, dir) {
				if strings.Contains(f, ".metais-tmp-") {
					t.Errorf("left %s", f)
				}
			}
		})
	}
	// A --dir that names a file, or nothing, is refused before anything.
	dir := testenv.Dir(t)
	put(t, dir, "file", []byte("x"))
	if _, err := Install(filepath.Join(dir, "file"), Options{Version: v1, Harnesses: []string{"claude"}}); hostfs.ReasonOf(err) != hostfs.NotDirectory {
		t.Errorf("a file: %v", err)
	}
	if _, err := Install(filepath.Join(dir, "absent"), Options{Version: v1, Harnesses: []string{"claude"}}); hostfs.ReasonOf(err) != hostfs.NotFound {
		t.Errorf("nothing: %v", err)
	}
}

// TestStamp holds the stamp to item 5: the last line, naming the version
// and the digest of every byte before it, which own recognises and
// nothing else does.
func TestStamp(t *testing.T) {
	text := Text(v1)
	if !bytes.HasPrefix(text, source) || !stampLine.Match(text[len(source):]) {
		t.Fatalf("the stamp is not the last line: %q", text[len(source):])
	}
	if v, ok := own(text); !ok || v != v1 {
		t.Errorf("own(Text) = %q, %v", v, ok)
	}
	if _, ok := own(text[:len(text)-1]); ok {
		t.Error("a stamp with no final LF is own")
	}
}

// TestFrontmatter holds the frontmatter to the rules every harness item 6
// names applies: name and description alone, the name the directory's,
// lowercase letters, digits and single hyphens, at most 64 characters, and
// a description of 1 to 1024 characters.
func TestFrontmatter(t *testing.T) {
	s := string(source)
	rest, ok := strings.CutPrefix(s, "---\n")
	if !ok {
		t.Fatal("SKILL.md does not open with frontmatter")
	}
	front, _, ok := strings.Cut(rest, "\n---\n")
	if !ok {
		t.Fatal("the frontmatter is not closed")
	}
	fields := map[string]string{}
	for _, line := range strings.Split(front, "\n") {
		k, v, ok := strings.Cut(line, ": ")
		if !ok {
			t.Fatalf("frontmatter line %q is not one key and value", line)
		}
		fields[k] = v
	}
	if len(fields) != 2 || fields["name"] != Name || fields["description"] == "" {
		t.Errorf("frontmatter %q holds other than name and description", fields)
	}
	if !regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`).MatchString(Name) || len(Name) > 64 {
		t.Errorf("name %q breaks the name rule", Name)
	}
	if n := len([]rune(fields["description"])); n < 1 || n > 1024 {
		t.Errorf("description of %d characters", n)
	}
	if !strings.Contains(PathOf(".claude"), "/"+Name+"/SKILL.md") {
		t.Error("the directory is not the name")
	}
}

// TestSize holds the skill under 16 KiB, so a harness that reads every
// skill whole is not taxed (ADR-008 A2 item 7).
func TestSize(t *testing.T) {
	if n := len(Text(v1)); n >= 16<<10 {
		t.Errorf("the skill is %d bytes", n)
	}
}
