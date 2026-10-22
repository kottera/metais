// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package score

import (
	"encoding/json"
	"testing"

	"github.com/kottera/metais/internal/tools/ux/streamjson"
)

// TestRenderToStandardOutput: a render to standard output (--out -) is a
// render when it wrote there, as one into a directory is when it lists
// what it wrote, and a refused one is not: metais writes nothing to
// standard output unless the render succeeds (MTS-RND-011), and reports
// the refusal on standard error, which a session often merges.
func TestRenderToStandardOutput(t *testing.T) {
	const render = "metais render r.metais --sample --out - --only invoice.html"
	for _, c := range []struct {
		name, command, output string
		isError, want         bool
	}{
		{"its bytes", render, "<html><body>Invoice</body></html>\n", false, true},
		{"piped", render + " 2>&1 | grep Total", "<td>Total</td>\n", false, true},
		{"redirected to a file", render + " > out/invoice.html", "", false, true},
		{"refused, and the call reports an error", render, "metais: error: Argument 7 names no file the render writes. [usage.unknown_output]\n", true, false},
		{"refused, piped", render + " 2>&1 | head", "metais: error: Argument 7 names no file the render writes. [usage.unknown_output]\n", false, false},
		{"refused, in JSON", render + " --json 2>&1 | head", `{"format":"metais.render/1","ok":false}` + "\n", false, false},
		{"nothing written", render + " | grep Nothing", "", false, false},
		{"into a directory, listing nothing", "metais render r.metais --sample --out /ux/host/o", "", false, false},
	} {
		in, _ := json.Marshal(map[string]string{"command": c.command})
		res, _ := json.Marshal(map[string]any{"type": "tool_result", "tool_use_id": "t1", "content": c.output, "is_error": c.isError})
		tr := `{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"t1","name":"Bash","input":` + string(in) + `}]}}` + "\n" +
			`{"type":"user","message":{"role":"user","content":[` + string(res) + `]}}` + "\n"
		r := &run{t: streamjson.Read([]byte(tr)), runDir: "/ux/run"}
		r.read()
		if got := len(r.renders()) == 1; got != c.want {
			t.Errorf("%s: %q, output %q: counted as a render %v, want %v", c.name, c.command, c.output, got, c.want)
		}
	}
}
