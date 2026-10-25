// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package streamjson

import (
	"maps"
	"slices"
	"strings"
	"testing"
)

// TestRead: a transcript's opening, its tool calls with the results the
// harness returned and the turn each is part of, its text, and its result
// are read; a line that is not JSON is counted and skipped.
func TestRead(t *testing.T) {
	tr := Read([]byte(strings.Join([]string{
		`{"type":"system","subtype":"init","cwd":"/ux/run/work","model":"claude-haiku-5-5","claude_code_version":"2.1.299","mcp_servers":[],"skills":["debug"]}`,
		`{"type":"assistant","message":{"id":"m1","content":[{"type":"text","text":"I'll look."}]}}`,
		`{"type":"assistant","message":{"id":"m1","content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"metais inspect report.metais"}}]}}`,
		`not json`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t1","content":"Package dora"}]},"tool_use_result":{"stdout":"Package dora","returnCodeInterpretation":"No matches found"}}`,
		`{"type":"assistant","message":{"id":"m2","content":[{"type":"tool_use","id":"t2","name":"Write","input":{"file_path":"/ux/run/work/a.json","content":"{}"}}]}}`,
		`{"type":"user","message":{"content":[{"type":"tool_result","tool_use_id":"t2","is_error":true,"content":[{"type":"text","text":"Exit code 3"},{"type":"text","text":"no"}]}]}}`,
		`{"type":"result","subtype":"success","is_error":false,"num_turns":3,"total_cost_usd":0.01,"duration_ms":1200,"result":"Done?","modelUsage":{"claude-haiku-5-5":{"contextWindow":200000},"claude-opus-5-5":{}}}`,
	}, "\n")))
	if !tr.Init.Found || tr.Init.Line != 1 || tr.Init.Cwd != "/ux/run/work" || tr.Init.Version != "2.1.299" || tr.Init.MCPServers != 0 {
		t.Errorf("init = %+v", tr.Init)
	}
	if tr.Malformed != 1 {
		t.Errorf("malformed = %d", tr.Malformed)
	}
	if len(tr.ToolUses) != 2 {
		t.Fatalf("tool uses = %+v", tr.ToolUses)
	}
	u := tr.ToolUses[0]
	if u.Name != "Bash" || u.Command() != "metais inspect report.metais" || u.Line != 3 || u.Turn != 1 ||
		u.Result == nil || u.Result.IsError || u.Result.Text != "Package dora" || u.Result.Line != 5 || u.Result.Interpreted != "No matches found" {
		t.Errorf("first tool use = %+v, result %+v", u, u.Result)
	}
	u = tr.ToolUses[1]
	if u.Command() != "" || u.Field("file_path") != "/ux/run/work/a.json" || u.Turn != 2 || u.Result == nil || !u.Result.IsError || u.Result.Text != "Exit code 3\nno" || u.Result.Interpreted != "" {
		t.Errorf("second tool use = %+v, result %+v", u, u.Result)
	}
	if len(tr.Texts) != 1 || tr.Texts[0].Text != "I'll look." || tr.Texts[0].Turn != 1 {
		t.Errorf("texts = %+v", tr.Texts)
	}
	if !tr.Result.Found || tr.Result.Line != 8 || tr.Result.Text != "Done?" || tr.Result.Turns != 3 || !slices.Equal(tr.Result.Models, []string{"claude-haiku-5-5", "claude-opus-5-5"}) ||
		!maps.Equal(tr.Result.ContextWindows, map[string]int{"claude-haiku-5-5": 200000}) {
		t.Errorf("result = %+v", tr.Result)
	}
}

// TestReadCutOff: a transcript cut off mid-session is read up to where it
// stops, with no result and no tool result for the call left unanswered.
func TestReadCutOff(t *testing.T) {
	tr := Read([]byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","id":"t1","name":"Bash","input":{"command":"sleep 99"}}]}}` + "\n" + `{"type":"assi`))
	if tr.Result.Found || len(tr.ToolUses) != 1 || tr.ToolUses[0].Result != nil || tr.Malformed != 1 {
		t.Errorf("transcript = %+v", tr)
	}
}
