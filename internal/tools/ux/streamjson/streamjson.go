// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

// Package streamjson reads a Claude Code session's stream-json
// transcript: one JSON object per line, as claude -p --output-format
// stream-json --verbose writes it. It is the one reading of a transcript
// the UX runner and its scorer share (PLAN-2026-10-metais-ux W1-1, W1-2):
// the runner's isolation checks read it, and so does every pass rule of
// the scorer.
//
// The package is handed the transcript's bytes and refers to no host
// symbol, so the scorer may use it with none of the runner's allowance
// (ADR-009 Amendment A8 item 4).
package streamjson

import (
	"encoding/json"
	"slices"
	"strings"
)

// Transcript is what a session's transcript says.
type Transcript struct {
	// Init is the session's opening system/init event, the first one.
	Init struct {
		Found      bool
		Line       int
		Cwd        string
		Model      string
		Version    string
		MCPServers int
		Skills     []string
	}
	// Result is the session's closing result event.
	Result struct {
		Found      bool
		Line       int
		IsError    bool
		Subtype    string
		APIStatus  int
		Text       string
		Turns      int
		CostUSD    float64
		DurationMS int64
		Models     []string
		// ContextWindows is the context window, in tokens, the session's
		// usage reports for each model it names, where it reports one.
		ContextWindows map[string]int
	}
	// ToolUses are the session's tool calls, in order.
	ToolUses []ToolUse
	// Texts are the text the session wrote to the user, in order, each
	// part as one entry.
	Texts []Text
	// Malformed counts lines that are not JSON objects.
	Malformed int
}

// ToolUse is one tool call: its tool's name and its input as written, and
// the result the harness returned to the session, when one is recorded.
type ToolUse struct {
	ID    string
	Name  string
	Input json.RawMessage
	// Line is the transcript's line, from 1, and Turn the session's turn:
	// the assistant message it is part of, counted from 1.
	Line, Turn int
	Result     *ToolResult
}

// ToolResult is the harness's answer to a tool call.
type ToolResult struct {
	Line    int
	IsError bool
	// Text is the result's content, its text parts joined by newlines.
	Text string
	// Interpreted is the harness's reading of a Bash call's exit status
	// that was not 0 and that it did not report as an error, such as
	// grep's "No matches found"; "" when it exited 0 or was reported so.
	Interpreted string
}

// Text is one part of text the session wrote.
type Text struct {
	Line, Turn int
	Text       string
}

// Command is a Bash tool call's command, or "" for any other tool.
func (u ToolUse) Command() string {
	if u.Name != "Bash" {
		return ""
	}
	var in struct {
		Command string `json:"command"`
	}
	_ = json.Unmarshal(u.Input, &in)
	return in.Command
}

// Field is the string a tool call's input holds under key, or "".
func (u ToolUse) Field(key string) string {
	var in map[string]any
	if json.Unmarshal(u.Input, &in) != nil {
		return ""
	}
	s, _ := in[key].(string)
	return s
}

// Read reads a stream-json transcript. A line that is not a JSON object is
// counted, not fatal: a transcript cut off by a timeout is still read up
// to where it stops.
func Read(b []byte) *Transcript {
	t := &Transcript{}
	byID := map[string]*ToolResult{}
	turn, lastMessage := 0, ""
	for i, line := range strings.Split(string(b), "\n") {
		n := i + 1
		if strings.TrimSpace(line) == "" {
			continue
		}
		var ev struct {
			Type    string `json:"type"`
			Subtype string `json:"subtype"`
			// system/init
			Cwd        string            `json:"cwd"`
			Model      string            `json:"model"`
			Version    string            `json:"claude_code_version"`
			MCPServers []json.RawMessage `json:"mcp_servers"`
			Skills     []string          `json:"skills"`
			// assistant and user
			Message struct {
				ID      string          `json:"id"`
				Content json.RawMessage `json:"content"`
			} `json:"message"`
			// result
			IsError    bool                       `json:"is_error"`
			APIStatus  *int                       `json:"api_error_status"`
			Result     string                     `json:"result"`
			NumTurns   int                        `json:"num_turns"`
			CostUSD    float64                    `json:"total_cost_usd"`
			DurationMS int64                      `json:"duration_ms"`
			ModelUsage map[string]json.RawMessage `json:"modelUsage"`
			// user: the harness's own record of a tool call's result
			ToolUseResult json.RawMessage `json:"tool_use_result"`
		}
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			t.Malformed++
			continue
		}
		switch {
		case ev.Type == "system" && ev.Subtype == "init" && !t.Init.Found:
			t.Init.Found, t.Init.Line = true, n
			t.Init.Cwd, t.Init.Model, t.Init.Version = ev.Cwd, ev.Model, ev.Version
			t.Init.MCPServers = len(ev.MCPServers)
			t.Init.Skills = ev.Skills
		case ev.Type == "assistant":
			// One assistant message is streamed as one event per part,
			// each carrying the message's id.
			if id := ev.Message.ID; id == "" || id != lastMessage {
				turn++
				lastMessage = id
			}
			for _, part := range parts(ev.Message.Content) {
				switch part.Type {
				case "tool_use":
					t.ToolUses = append(t.ToolUses, ToolUse{ID: part.ID, Name: part.Name, Input: part.Input, Line: n, Turn: turn})
				case "text":
					t.Texts = append(t.Texts, Text{Line: n, Turn: turn, Text: part.Text})
				}
			}
		case ev.Type == "user":
			for _, part := range parts(ev.Message.Content) {
				if part.Type == "tool_result" && part.ToolUseID != "" {
					tr := &ToolResult{Line: n, IsError: part.IsError, Text: contentText(part.Content)}
					var rec struct {
						Interpretation string `json:"returnCodeInterpretation"`
					}
					if json.Unmarshal(ev.ToolUseResult, &rec) == nil {
						tr.Interpreted = rec.Interpretation
					}
					byID[part.ToolUseID] = tr
				}
			}
		case ev.Type == "result":
			r := &t.Result
			r.Found, r.Line = true, n
			r.IsError, r.Subtype, r.Text = ev.IsError, ev.Subtype, ev.Result
			if ev.APIStatus != nil {
				r.APIStatus = *ev.APIStatus
			}
			r.Turns, r.CostUSD, r.DurationMS = ev.NumTurns, ev.CostUSD, ev.DurationMS
			r.Models = slices.Sorted(func(yield func(string) bool) {
				for m := range ev.ModelUsage {
					if !yield(m) {
						return
					}
				}
			})
			r.ContextWindows = nil
			for m, raw := range ev.ModelUsage {
				var u struct {
					ContextWindow int `json:"contextWindow"`
				}
				if json.Unmarshal(raw, &u) == nil && u.ContextWindow > 0 {
					if r.ContextWindows == nil {
						r.ContextWindows = map[string]int{}
					}
					r.ContextWindows[m] = u.ContextWindow
				}
			}
		}
	}
	for i := range t.ToolUses {
		if r, ok := byID[t.ToolUses[i].ID]; ok && t.ToolUses[i].ID != "" {
			t.ToolUses[i].Result = r
		}
	}
	return t
}

// part is one part of a message's content.
type part struct {
	Type      string          `json:"type"`
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Input     json.RawMessage `json:"input"`
	Text      string          `json:"text"`
	ToolUseID string          `json:"tool_use_id"`
	IsError   bool            `json:"is_error"`
	Content   json.RawMessage `json:"content"`
}

// parts is a message's content: a list of parts, or a string, which is
// one text part.
func parts(raw json.RawMessage) []part {
	var ps []part
	if json.Unmarshal(raw, &ps) == nil {
		return ps
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []part{{Type: "text", Text: s}}
	}
	return nil
}

// contentText is a tool result's content as text: a string, or its text
// parts joined by newlines.
func contentText(raw json.RawMessage) string {
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return s
	}
	var texts []string
	for _, p := range parts(raw) {
		if p.Type == "text" {
			texts = append(texts, p.Text)
		}
	}
	return strings.Join(texts, "\n")
}
