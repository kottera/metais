// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package scrub

import (
	"strings"
	"testing"
)

// TestBytes: each credential and each host path is taken out, the run's
// and the set's directories become their placeholders, and what names
// neither is kept as it is.
func TestBytes(t *testing.T) {
	const run = "/var/lib/runner/volumes/wo/scratch/set/runs/C1-render-real/haiku"
	s := New(run, "/var/lib/runner/volumes/wo/scratch/set")
	for _, c := range []struct{ in, want string }{
		{run + "/work/report.metais", "/ux/run/work/report.metais"},
		{`"cwd":"` + run + `/work"`, `"cwd":"/ux/run/work"`},
		{"/var/lib/runner/volumes/wo/scratch/set/bin/metais", "/ux/set/bin/metais"},
		{"PATH=/var/lib/runner/volumes/wo/scratch/set/bin:/var/lib/runner/toolchain/go/bin:/usr/bin",
			"PATH=/ux/set/bin:/ux/host:/usr/bin"},
		{"cd /Users/ed/src/metais && ls", "cd /ux/host && ls"},
		{"cat /tmp/x.json", "cat /ux/host"},
		{"/tmpfoo and report/data/x and /etc/bash.bashrc and /usr/bin/env", "/tmpfoo and report/data/x and /etc/bash.bashrc and /usr/bin/env"},
		{run + "/tmp/data.json", "/ux/run/tmp/data.json"},
		{`"ANTHROPIC_AUTH_TOKEN=metais-ux-placeholder"`, `"ANTHROPIC_AUTH_TOKEN=[scrubbed]"`},
		{"export ANTHROPIC_API_KEY='sk-ant-api03-abcdefghijkl'", "export ANTHROPIC_API_KEY='[scrubbed]'"},
		{"key sk-ant-oat01-abcdefghijklmnop end", "key [scrubbed] end"},
		{"the token metais-ux-placeholder", "the token [scrubbed]"},
		{"Authorization: Bearer abcdefghijklmnopqrstuvwxyz", "Authorization: Bearer [scrubbed]"},
		{"GH_TOKEN=ghp_abcdefghijklmnopqrstuvwxyz0123", "GH_TOKEN=[scrubbed]"},
		{"apiKeySource ANTHROPIC_AUTH_TOKEN", "apiKeySource ANTHROPIC_AUTH_TOKEN"},
	} {
		got := string(s.Bytes([]byte(c.in)))
		if got != c.want {
			t.Errorf("%q: scrubbed to %q, want %q", c.in, got, c.want)
		}
		if again := string(s.Bytes([]byte(got))); again != got {
			t.Errorf("%q: scrubbed again to %q", got, again)
		}
		if found := Check([]byte(got)); len(found) > 0 {
			t.Errorf("%q: a scrubbed text still holds %q", got, found)
		}
	}
}

// TestCheck: Check names each credential and host path, and passes a
// text that holds only placeholders.
func TestCheck(t *testing.T) {
	for _, in := range []string{
		"ANTHROPIC_AUTH_TOKEN=metais-ux-placeholder",
		"metais-ux-placeholder",
		"sk-ant-api03-abcdefghijkl",
		"/var/lib/tera-runner/volumes/work_order/x/scratch",
		"/home/ed/report.metais",
		`C:\Users\ed\report.metais`,
	} {
		if found := Check([]byte(in)); len(found) == 0 {
			t.Errorf("Check(%q) found nothing", in)
		}
	}
	clean := "/ux/run/work/report.metais /ux/set/bin/metais /ux/host /usr/bin/env ANTHROPIC_AUTH_TOKEN=[scrubbed]"
	if found := Check([]byte(clean)); len(found) > 0 {
		t.Errorf("Check(%q) = %q", clean, found)
	}
	if !strings.Contains(strings.Join(Check([]byte("x metais-ux-placeholder")), ""), "credential") {
		t.Error("the placeholder is not named a credential")
	}
}
