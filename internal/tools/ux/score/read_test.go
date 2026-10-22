// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"testing/fstest"

	"metais.org/metais/internal/tools/ux/scenario"
)

// step is one Bash call of a made-up transcript: its command line, the
// output the harness returned, and how it reported the exit status.
type step struct {
	command, output string
	// isError is a call the harness reports as an error; interpreted is
	// its reading of an exit status other than 0 it did not report so.
	isError     bool
	interpreted string
}

// synthetic reads a made-up run of scenario id, with files in its
// directory, as the scorer reads a recorded one.
func synthetic(t *testing.T, id string, files map[string]string, steps ...step) *run {
	t.Helper()
	set, err := scenario.Load(os.DirFS("../scenarios"))
	if err != nil {
		t.Fatal(err)
	}
	s := set[slices.IndexFunc(set, func(s *scenario.Scenario) bool { return s.ID == id })]
	var tr bytes.Buffer
	for i, st := range steps {
		use, _ := json.Marshal(map[string]any{"type": "assistant", "message": map[string]any{"id": fmt.Sprint("m", i), "content": []any{
			map[string]any{"type": "tool_use", "id": fmt.Sprint("t", i), "name": "Bash", "input": map[string]any{"command": st.command}}}}})
		res := map[string]any{"type": "user", "message": map[string]any{"content": []any{
			map[string]any{"type": "tool_result", "tool_use_id": fmt.Sprint("t", i), "is_error": st.isError, "content": st.output}}}}
		if st.interpreted != "" {
			res["tool_use_result"] = map[string]any{"stdout": st.output, "returnCodeInterpretation": st.interpreted}
		}
		rb, _ := json.Marshal(res)
		tr.Write(use)
		tr.WriteString("\n")
		tr.Write(rb)
		tr.WriteString("\n")
	}
	m := fstest.MapFS{
		"record.json":      {Data: []byte(`{"format":"metais-ux-run/1","scenario":{"id":"` + id + `"},"status":"complete","valid":true,"invocation":{"dir":"/ux/run/work"}}`)},
		"transcript.jsonl": {Data: tr.Bytes()},
	}
	for p, b := range files {
		m[p] = &fstest.MapFile{Data: []byte(b)}
	}
	r, err := load(Input{Scenario: s, Run: m})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// cwdOf is the directory the scorer reads the first metais command of
// the given subcommand as running in.
func cwdOf(t *testing.T, r *run, sub string) string {
	t.Helper()
	return cwdAt(t, r, sub, 0)
}

// cwdAt is cwdOf for the command at a line of the transcript, or the
// first at any line when line is 0.
func cwdAt(t *testing.T, r *run, sub string, line int) string {
	t.Helper()
	for _, c := range r.calls {
		if c.sub == sub && (line == 0 || c.use.Line == line) {
			return c.cwd
		}
	}
	t.Fatalf("no metais %s at line %d", sub, line)
	return ""
}

// TestPWD: $PWD is the directory the command runs in, so cd "$PWD" stays
// where it is and cd "$PWD/pkg" moves into pkg; a variable the call
// never assigns is still not known.
func TestPWD(t *testing.T) {
	for _, c := range []struct{ line, want string }{
		{`cd "$PWD"; metais check .`, "work"},
		{`cd "${PWD}/pkg" && metais check .`, "work/pkg"},
		{`cd pkg; cd "$PWD/sub"; metais check .`, "work/pkg/sub"},
		{`D=$PWD/pkg; cd $D; metais check .`, "work/pkg"},
		{`cd "$ELSEWHERE"; metais check .`, "work/$ELSEWHERE"},
	} {
		if got := cwdOf(t, synthetic(t, "A1", nil, step{command: c.line}), "check"); got != c.want {
			t.Errorf("%s: metais check runs in %q, want %q", c.line, got, c.want)
		}
	}
}

// TestCallDirectory: the harness records the shell's directory for the
// next call with pwd -P after the call and &&, so a cd carries to the
// next call only when its call exits 0. Within the call it moves the
// commands after it either way.
func TestCallDirectory(t *testing.T) {
	for _, c := range []struct {
		first step
		want  string
	}{
		{step{command: "cd pkg && ls"}, "work/pkg"},
		{step{command: "cd pkg && grep -q x y", interpreted: "No matches found"}, "work"},
		{step{command: "cd pkg && false", isError: true, output: "Exit code 1"}, "work"},
	} {
		r := synthetic(t, "A1", nil, c.first, step{command: "metais check ."})
		if got := cwdOf(t, r, "check"); got != c.want {
			t.Errorf("after %q (error %v, %q): metais check runs in %q, want %q", c.first.command, c.first.isError, c.first.interpreted, got, c.want)
		}
	}
	r := synthetic(t, "A1", nil, step{command: "cd pkg && metais check . && grep -q x y", interpreted: "No matches found"})
	if got := cwdOf(t, r, "check"); got != "work/pkg" {
		t.Errorf("within its own call, metais check runs in %q, want work/pkg", got)
	}

	// The recorded run: its cd numchart at line 21 is in a call that exits
	// 1, so its render at line 32 runs where that call's next cd numchart
	// takes it, not one directory deeper. Read as a call that exits 0,
	// the same transcript puts the render inside numchart/numchart.
	dir := fixturesDir + "/a3-sonnet-failed-call-keeps-directory"
	for _, c := range []struct {
		edit func([]byte) []byte
		want string
	}{
		{func(b []byte) []byte { return b }, "work/numchart"},
		{func(b []byte) []byte {
			return bytes.Replace(b, []byte(`"returnCodeInterpretation":"No matches found"`), []byte(`"returnCodeInterpretation":null`), 1)
		}, "work/numchart/numchart"},
	} {
		m := fstest.MapFS{}
		for _, p := range []string{"record.json", "transcript.jsonl"} {
			b, err := os.ReadFile(dir + "/" + p)
			if err != nil {
				t.Fatal(err)
			}
			m[p] = &fstest.MapFile{Data: c.edit(b)}
		}
		set, err := scenario.Load(os.DirFS("../scenarios"))
		if err != nil {
			t.Fatal(err)
		}
		r, err := load(Input{Scenario: set[slices.IndexFunc(set, func(s *scenario.Scenario) bool { return s.ID == "A3" })], Run: m})
		if err != nil {
			t.Fatal(err)
		}
		if got := cwdAt(t, r, "render", 32); got != c.want {
			t.Errorf("the render at line 32 runs in %q, want %q", got, c.want)
		}
	}
}

// TestUnroll: a for loop's body is read once for each of its words, with
// the loop's variable expanded; a loop whose words the line does not hold
// is read once, as it is.
func TestUnroll(t *testing.T) {
	for _, c := range []struct {
		line string
		want []string
	}{
		{"for s in 1 2; do metais render . --out ../.s$s; done; cat ../.s2/x", []string{"do metais render . --out ../.s1", "do metais render . --out ../.s2", "cat ../.s2/x"}},
		{"for f in a b\ndo\n  metais validate r --data $f\ndone", []string{"metais validate r --data a", "metais validate r --data b"}},
		{"for a in x y; do for b in 1 2; do echo $a$b; done; done", []string{"do echo x1", "do echo x2", "do echo y1", "do echo y2"}},
		{"for f in $(ls); do metais validate r --data $f; done", []string{"for f in", "ls", "do metais validate r --data $f"}},
		{"for s in 1 2; do echo $s", []string{"for s in 1 2", "do echo $s"}},
	} {
		var got []string
		for _, s := range unroll(parseShell(c.line)) {
			if w := strings.Join(s.words, " "); w != "do" && w != "done" && !strings.HasPrefix(w, "for b") {
				got = append(got, w)
			}
		}
		if !slices.Equal(got, c.want) {
			t.Errorf("%q:\n\tgot  %q\n\twant %q", c.line, got, c.want)
		}
	}
}

// TestReadBack: a render whose listing went elsewhere and whose files are
// gone has written when a later command of its own call reads a file of
// its output directory back, and the call shows no refusal and no
// missing file; nothing less is a render.
func TestReadBack(t *testing.T) {
	const render = "metais render . --sample --out ../o >/dev/null; "
	for _, c := range []struct {
		line, output string
		want         bool
	}{
		{render + "sed -n 1,5p ../o/page.html; rm -rf ../o", "<p>x</p>", true},
		{render + "head -3 < ../o/page.html; rm -rf ../o", "<p>x</p>", true},
		{render + "rm -rf ../o", "", false},
		{render + "ls ../o; rm -rf ../o", "page.html", false},
		{render + "cat ../other/page.html", "<p>x</p>", false},
		// A refusal, though the read printed a file a render before it
		// left there.
		{render + "cat ../o/page.html", "metais: error: The package cannot be rendered. [x]\n<p>x</p>", false},
		{render + "cat ../o/page.html", `{"ok":false}` + "\n<p>x</p>", false},
		{render + "cat ../o/page.html", "cat: ../o/page.html: No such file or directory", false},
		{"cat ../o/page.html; " + render, "<p>x</p>", false},
	} {
		r := synthetic(t, "A1", nil, step{command: "cd pkg"}, step{command: c.line, output: c.output})
		if got := len(r.renders()) > 0; got != c.want {
			t.Errorf("%q, output %q: rendered %v, want %v", c.line, c.output, got, c.want)
		}
	}
	// A read in a later call is not the render's: the files it reads may
	// be any command's in between.
	r := synthetic(t, "A1", nil, step{command: "cd pkg"}, step{command: render + "true"}, step{command: "cat ../o/page.html", output: "<p>x</p>"})
	if len(r.renders()) > 0 {
		t.Error("a read in the next call is seen as the render's")
	}
}
