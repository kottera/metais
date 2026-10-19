// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package liquidspec

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// reply is a response as a test reads it.
type reply struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  map[string]any  `json:"result"`
	Error   *struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	} `json:"error"`
}

// exchange serves lines, one message each, and returns every response.
func exchange(t *testing.T, lines ...string) []reply {
	t.Helper()
	var out bytes.Buffer
	if err := serve(strings.NewReader(strings.Join(lines, "\n")+"\n"), &out); err != nil {
		t.Fatalf("serve: %v", err)
	}
	var rs []reply
	for l := range strings.SplitSeq(strings.TrimSuffix(out.String(), "\n"), "\n") {
		if l == "" {
			continue
		}
		var r reply
		if err := json.Unmarshal([]byte(l), &r); err != nil {
			t.Fatalf("response %q is not JSON: %v", l, err)
		}
		if r.JSONRPC != "2.0" || (r.Result == nil) == (r.Error == nil) {
			t.Fatalf("response %q is not a JSON-RPC 2.0 result or error", l)
		}
		rs = append(rs, r)
	}
	return rs
}

// request is one JSON-RPC request line.
func request(id int, method string, params any) string {
	b, err := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params})
	if err != nil {
		panic(err)
	}
	return string(b)
}

// compileRender compiles src with fs and renders it with env, and returns
// the render's response, or the compile's when it refused.
func compileRender(t *testing.T, src string, fs map[string]string, env any) reply {
	t.Helper()
	params := map[string]any{"template": src, "options": map[string]any{"error_mode": "strict2", "line_numbers": true}}
	if fs != nil {
		params["filesystem"] = fs
	}
	rs := exchange(t, request(1, "compile", params),
		request(2, "render", map[string]any{"template_id": "tmpl_1", "environment": env,
			"options": map[string]any{"strict_errors": true, "registers": map[string]any{"current_time": "2024-01-01T00:01:58Z"}}}))
	if len(rs) != 2 {
		t.Fatalf("%d responses, want 2", len(rs))
	}
	if rs[0].Error != nil || rs[0].Result["template_id"] == nil {
		return rs[0]
	}
	return rs[1]
}

// TestInitialize: initialize answers the protocol's version, the gem
// release the dialect pins, and no features, since the adapter selects
// what runs; another protocol version is invalid params.
func TestInitialize(t *testing.T) {
	testenv.Home(t)
	rs := exchange(t, request(1, "initialize", map[string]any{"version": "1.0"}), request(2, "initialize", nil),
		request(3, "initialize", map[string]any{"version": "2.0"}))
	for _, r := range rs[:2] {
		if r.Result["version"] != "1.0" || r.Result["implementation"] != "metais" || r.Result["liquid_version"] != "5.14.0" {
			t.Errorf("initialize: %v", r.Result)
		}
		if fs, ok := r.Result["features"].([]any); !ok || len(fs) != 0 {
			t.Errorf("features %v, want []", r.Result["features"])
		}
	}
	if string(rs[1].ID) != "2" {
		t.Errorf("id %s, want 2", rs[1].ID)
	}
	if e := rs[2].Error; e == nil || e.Code != codeInvalidParams || e.Data["param"] != "version" {
		t.Errorf("initialize 2.0: %+v, want invalid params naming version", rs[2])
	}
}

// TestCompileRender: a template renders through the reference engine in
// the dialect, its environment the globals with numbers kept as written,
// and its file system the templates a render tag names exactly.
func TestCompileRender(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src  string
		fs   map[string]string
		env  string
		want string
	}{
		{`Hello {{ name | upcase }}!`, nil, `{"name": "World"}`, "Hello WORLD!"},
		{`{{ i }} {{ f }} {{ i | plus: 1 }} {{ f | times: 2 }}`, nil, `{"i": 3, "f": 2.5}`, "3 2.5 4 5.0"},
		{`{% for x in xs %}{{ x }}{% endfor %}`, nil, `{"xs": [1, "a", true, null]}`, "1atrue"},
		{`{% render "p.liquid", who: name %}|{% render "dir/q.liquid" %}`, map[string]string{"p.liquid": "hi {{ who }}", "dir/q.liquid": "q"}, `{"name": "you"}`, "hi you|q"},
		{`{% for p in (1..2) %}{{ p }}{% endfor %}`, map[string]string{"template": "the file system's own template"}, `null`, "12"},
		{`plain`, nil, ``, "plain"},
	} {
		var env any
		if c.env != "" {
			if err := json.Unmarshal([]byte(c.env), &env); err != nil {
				t.Fatal(err)
			}
		}
		r := compileRender(t, c.src, c.fs, env)
		if r.Error != nil || r.Result["output"] != c.want {
			t.Errorf("%s: %+v, want output %q", c.src, r, c.want)
			continue
		}
		if es, ok := r.Result["errors"].([]any); !ok || len(es) != 0 {
			t.Errorf("%s: errors %v, want []", c.src, r.Result["errors"])
		}
	}
}

// TestParseErrors: a template the dialect refuses at parse is a compile
// result with a null template_id and a parse_error naming the code and
// its line, never a protocol error. A render tag naming a file without
// its extension names no template (MTS-TPL-041), and a partial's own
// parse error refuses the compile.
func TestParseErrors(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src  string
		fs   map[string]string
		code string
		line float64
	}{
		{"a\n{% nosuchtag %}", nil, "template.syntax", 2},
		{`{% include "p" %}`, nil, "template.include", 1},
		{`{{ x | nosuch }}`, nil, "template.unknown_filter", 1},
		{`{{ ["x"] }}`, nil, "template.syntax", 1},
		{`{% render "p" %}`, map[string]string{"p.liquid": "p"}, "template.partial_not_found", 1},
		{`{% render "p.liquid" %}`, map[string]string{"p.liquid": "\n\n{% if %}"}, "template.syntax", 3},
		{`{% render "a" %}`, map[string]string{"a": "x", "a/b": "y"}, "template.partial_not_found", 1},
	} {
		r := compileRender(t, c.src, c.fs, nil)
		e, _ := r.Result["error"].(map[string]any)
		if r.Error != nil || r.Result["template_id"] != nil || e == nil {
			t.Errorf("%q: %+v, want a parse error result", c.src, r)
			continue
		}
		if e["type"] != "parse_error" || !strings.HasPrefix(e["message"].(string), c.code+": ") || e["line"] != c.line {
			t.Errorf("%q: error %v, want a parse_error %s at line %v", c.src, e, c.code, c.line)
		}
	}
}

// TestRenderErrors: an error met while rendering is raised, never written
// into the output (MTS-TPL-047): the protocol's render error, code
// -32001, with its type, message and line.
func TestRenderErrors(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src, env, code string
		line           float64
	}{
		{"a\nb{{ nope }}", `{}`, "undefined_variable", 2},
		{`{{ o }}`, `{"o": {"a": 1}}`, "object_output", 1},
		{`{{ o.b.c }}`, `{"o": {"b": null}}`, "nil_property", 1},
		{`{% if 1 < "2" %}{% endif %}`, `{}`, "comparison", 1},
	} {
		var env any
		if err := json.Unmarshal([]byte(c.env), &env); err != nil {
			t.Fatal(err)
		}
		r := compileRender(t, c.src, nil, env)
		if r.Error == nil || r.Error.Code != codeRender || r.Error.Data["type"] != "render_error" ||
			!strings.Contains(r.Error.Data["message"].(string), c.code) || r.Error.Data["line"] != c.line {
			t.Errorf("%q: %+v, want a render error %s at line %v", c.src, r, c.code, c.line)
		}
	}
}

// TestProtocolErrors: each fault of a message is answered with the
// protocol's error for it, and the connection goes on.
func TestProtocolErrors(t *testing.T) {
	testenv.Home(t)
	compile := request(1, "compile", map[string]any{"template": "x"})
	for _, c := range []struct {
		name  string
		lines []string
		code  int
		param string
		id    string
	}{
		{"not JSON", []string{`{"jsonrpc": "2.0", "id": 1,`}, codeJSON, "", "null"},
		{"not an object", []string{`[1, 2]`}, codeInvalidRequest, "", "null"},
		{"an id that is an object", []string{`{"jsonrpc": "2.0", "id": {}, "method": "initialize"}`}, codeInvalidRequest, "", "null"},
		{"no jsonrpc", []string{`{"id": 7, "method": "initialize"}`}, codeInvalidRequest, "", "7"},
		{"no method", []string{`{"jsonrpc": "2.0", "id": "a"}`}, codeInvalidRequest, "", `"a"`},
		{"an unknown method", []string{request(1, "eval", nil)}, codeNoMethod, "", "1"},
		{"params not an object", []string{`{"jsonrpc": "2.0", "id": 1, "method": "compile", "params": [1]}`}, codeInvalidParams, "params", "1"},
		{"compile without a template", []string{request(1, "compile", map[string]any{})}, codeInvalidParams, "template", "1"},
		{"a template that is not a string", []string{request(1, "compile", map[string]any{"template": 3})}, codeInvalidParams, "params", "1"},
		{"lax", []string{request(1, "compile", map[string]any{"template": "x", "options": map[string]any{"error_mode": "lax"}})}, codeInvalidParams, "options.error_mode", "1"},
		{"strict", []string{request(1, "compile", map[string]any{"template": "x", "options": map[string]any{"error_mode": "strict"}})}, codeInvalidParams, "options.error_mode", "1"},
		{"render without a template_id", []string{request(1, "render", map[string]any{})}, codeInvalidParams, "template_id", "1"},
		{"an unknown template_id", []string{request(1, "render", map[string]any{"template_id": "tmpl_9"})}, codeInvalidParams, "template_id", "1"},
		{"inline errors", []string{compile, request(2, "render", map[string]any{"template_id": "tmpl_1", "options": map[string]any{"strict_errors": false}})}, codeInvalidParams, "options.strict_errors", "2"},
		{"resource limits", []string{compile, request(2, "render", map[string]any{"template_id": "tmpl_1", "options": map[string]any{"resource_limits": map[string]any{"render_score_limit": 10}}})}, codeInvalidParams, "options.resource_limits", "2"},
		{"registers not an object", []string{compile, request(2, "render", map[string]any{"template_id": "tmpl_1", "options": map[string]any{"registers": 3}})}, codeInvalidParams, "options.registers", "2"},
		{"an environment that is not an object", []string{compile, request(2, "render", map[string]any{"template_id": "tmpl_1", "environment": []int{1}})}, codeInvalidParams, "environment", "2"},
		{"a member named twice", []string{compile, `{"jsonrpc": "2.0", "id": 2, "method": "render", "params": {"template_id": "tmpl_1", "environment": {"a": 1, "a": 2}}}`}, codeInvalidParams, "environment", "2"},
	} {
		rs := exchange(t, append(c.lines, request(99, "initialize", nil))...)
		r := rs[len(rs)-2]
		if r.Error == nil || r.Error.Code != c.code || string(r.ID) != c.id {
			t.Errorf("%s: %+v, want error %d with id %s", c.name, r, c.code, c.id)
			continue
		}
		if c.param != "" && r.Error.Data["param"] != c.param {
			t.Errorf("%s: data %v, want param %s", c.name, r.Error.Data, c.param)
		}
		if last := rs[len(rs)-1]; last.Result == nil || string(last.ID) != "99" {
			t.Errorf("%s: the connection did not go on: %+v", c.name, last)
		}
	}
}

// TestNotificationsAndQuit: a notification, which has no id, is never
// answered; quit ends the connection, and nothing after it is read; a
// quit sent as a request is answered first.
func TestNotificationsAndQuit(t *testing.T) {
	testenv.Home(t)
	rs := exchange(t, `{"jsonrpc": "2.0", "method": "initialize"}`, "", request(1, "compile", map[string]any{"template": "x"}),
		`{"jsonrpc": "2.0", "method": "quit"}`, request(2, "initialize", nil))
	if len(rs) != 1 || string(rs[0].ID) != "1" || rs[0].Result["template_id"] != "tmpl_1" {
		t.Errorf("responses %+v, want only compile's", rs)
	}
	rs = exchange(t, request(1, "quit", nil), request(2, "initialize", nil))
	if len(rs) != 1 || string(rs[0].ID) != "1" || rs[0].Error != nil {
		t.Errorf("responses %+v, want quit's alone", rs)
	}
}

// TestTemplateIDs: each compile has its own id, and each id renders its
// own template, however many follow it.
func TestTemplateIDs(t *testing.T) {
	testenv.Home(t)
	rs := exchange(t, request(1, "compile", map[string]any{"template": "a"}), request(2, "compile", map[string]any{"template": "{{ "}),
		request(3, "compile", map[string]any{"template": "b"}),
		request(4, "render", map[string]any{"template_id": "tmpl_1"}), request(5, "render", map[string]any{"template_id": "tmpl_2"}))
	if rs[0].Result["template_id"] != "tmpl_1" || rs[1].Result["template_id"] != nil || rs[2].Result["template_id"] != "tmpl_2" {
		t.Errorf("ids %v %v %v, want tmpl_1, none and tmpl_2", rs[0].Result, rs[1].Result, rs[2].Result)
	}
	if rs[3].Result["output"] != "a" || rs[4].Result["output"] != "b" {
		t.Errorf("outputs %v and %v, want a and b", rs[3].Result, rs[4].Result)
	}
}

// TestHeldOut: a template that applies date, or whose file system holds
// one that does, is held out of the run with HeldOut as its parse error;
// one that names date in text is not, and nor is one that does not parse
// even with date, which its parse refuses.
func TestHeldOut(t *testing.T) {
	testenv.Home(t)
	for _, c := range []struct {
		src  string
		fs   map[string]string
		held bool
	}{
		{`{{ "now" | date: "%Y" }}`, nil, true},
		{`{% if %}{{ t | date: "%Y" }}`, nil, false},
		{`{% render "p.liquid" %}`, map[string]string{"p.liquid": `{{ t | date: "%Y" }}`}, true},
		{`date {{ "date" }} {% assign date = 1 %}{{ date }}`, nil, false},
	} {
		r := compileRender(t, c.src, c.fs, map[string]any{})
		e, _ := r.Result["error"].(map[string]any)
		if held := e != nil && e["message"] == HeldOut; held != c.held {
			t.Errorf("%q: %+v, held out %v, want %v", c.src, r, held, c.held)
		}
	}
}

// TestDateNotRegistered fails by design once date is registered: its
// cases are then in scope, and the hold-out must go (heldout.go).
func TestDateNotRegistered(t *testing.T) {
	testenv.Home(t)
	if _, err := newServer(); err != nil {
		t.Fatal(err)
	}
}

// TestTree: the compiled template is named apart from every file of the
// file system, and a name that cannot be an entry path is no template.
func TestTree(t *testing.T) {
	testenv.Home(t)
	tr, name := newTree(map[string]string{"template": "a", "_template/x": "b", "ok.liquid": "c", "/abs": "d", "a//b": "e", `a\b`: "f", "c:d": "g", "./e": "h"}, "main")
	if name != "__template" {
		t.Errorf("name %q, want __template", name)
	}
	want := []string{"templates/__template", "templates/_template/x", "templates/ok.liquid", "templates/template"}
	if got := tr.List(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("entries %q, want %q", got, want)
	}
	if info, err := tr.Stat("templates/__template"); err != nil || info.Size != 4 {
		t.Errorf("stat: %v %v", info, err)
	}
	if _, err := tr.Open("templates/nope"); err == nil {
		t.Error("open of an absent entry succeeded")
	}
}
