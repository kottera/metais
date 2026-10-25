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
		{"cd /Users/someone/src/metais && ls", "cd /ux/host && ls"},
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
		{`"No module named 'liquid'\n/var/lib/runner/toolchain/node/bin/node"`, `"No module named 'liquid'\n/ux/host"`},
		{"home/.claude/projects/-var-lib-runner-volumes-wo-scratch-set-runs-C1-render-real-haiku-work/memory",
			"home/.claude/projects/-ux-run-work/memory"},
		{".claude/projects/-Users-someone-src-metais/", ".claude/projects/-ux-host/"},
		{`total 8\ndrwxr-xr-x 2 tera-runner tera-runner 4096 Oct 21 14:06 .\n-rw-r--r--  1 someuser staff  41429 Oct 21 14:21 report.metais`,
			`total 8\ndrwxr-xr-x 2 user group 4096 Jan  1 00:00 .\n-rw-r--r--  1 user group  41429 Jan  1 00:00 report.metais`},
		{"lrwxr-xr-x@ 1 someone admin 12 Oct 21 x -> y", "lrwxr-xr-x@ 1 user group 12 Oct 21 x -> y"},
		// A time ls -l printed in the host's zone (W3-OOB5's 10:05 against
		// 14:05Z), in each form ls prints one; an older file's year is no time.
		{"-rw-r--r--@ 1 someone staff 41429 Oct 25 10:05 report.metais", "-rw-r--r--@ 1 user group 41429 Jan  1 00:00 report.metais"},
		{"drwxr-xr-x  3 someone staff 96 Oct  5 9:07 out", "drwxr-xr-x  3 user group 96 Jan  1 00:00 out"},
		{"-rw-r--r-- 1 someone staff 7 2026-10-25 10:05 a", "-rw-r--r-- 1 user group 7 2000-01-01 00:00 a"},
		{"-rw-r--r-- 1 someone staff 7 2026-10-25 10:05:31.123456789 -0400 a", "-rw-r--r-- 1 user group 7 2000-01-01 00:00 a"},
		{"-rw-r--r-- 1 someone staff 7 Oct 21  2025 old", "-rw-r--r-- 1 user group 7 Oct 21  2025 old"},
		{"the meeting is Oct 25 10:05", "the meeting is Oct 25 10:05"},
		// A PATH keeps the placeholders and the system's directories, and
		// every other entry, which describes the host, becomes one /ux/host:
		// W3-OOB5's Wireshark, /pkg/env/global/bin and macOS's cryptexd
		// paths, wherever the PATH is written.
		{"PATH=/ux/set/bin:/Applications/Wireshark.app/Contents/MacOS:/usr/bin",
			"PATH=/ux/set/bin:/ux/host:/usr/bin"},
		{"PATH=/pkg/env/global/bin", "PATH=/ux/host"},
		{`"PATH=/usr/local/bin:/System/Cryptexes/App/usr/bin:/usr/bin:/bin:/usr/sbin:/sbin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/local/bin:/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/bin:/pkg/env/global/bin"`,
			`"PATH=/usr/local/bin:/ux/host:/usr/bin:/bin:/usr/sbin:/sbin:/ux/host"`},
		{`"content":"/Users/someone/.local/bin:/opt/homebrew/bin:/Applications/Wireshark.app/Contents/MacOS:/usr/bin:/bin\n"`,
			`"content":"/ux/host:/usr/bin:/bin\n"`},
		{"echo $PATH: " + run + "/home/go/bin:/pkg/env/global/bin:/sbin/", "echo $PATH: /ux/run/home/go/bin:/ux/host:/sbin/"},
		{"see https://metais.org/install.sh and /ux/run/work/a.go:12:3", "see https://metais.org/install.sh and /ux/run/work/a.go:12:3"},
		{"/Applications/Wireshark.app is not a PATH", "/Applications/Wireshark.app is not a PATH"},
		{`sed -n '/^Strings:/,/^Encoding/p'`, `sed -n '/^Strings:/,/^Encoding/p'`},
		{"the file -rw-r--r-- is plain", "the file -rw-r--r-- is plain"},
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
		"/home/someone/report.metais",
		`C:\Users\someone\report.metais`,
		`"x\n/home/someone/y"`,
		"-var-lib-tera-runner-volumes-work-order-x",
		"-rw-r--r-- 1 someuser staff 41429 Oct 21 14:21 report.metais",
		"-rw-r--r-- 1 user group 41429 Oct 25 10:05 report.metais",
		"PATH=/ux/set/bin:/Applications/Wireshark.app/Contents/MacOS",
		"/usr/bin:/pkg/env/global/bin",
		"/var/run/com.apple.security.cryptexd/codex.system/bootstrap/usr/bin:/bin",
	} {
		if found := Check([]byte(in)); len(found) == 0 {
			t.Errorf("Check(%q) found nothing", in)
		}
	}
	clean := "/ux/run/work/report.metais /ux/set/bin/metais /ux/host /usr/bin/env ANTHROPIC_AUTH_TOKEN=[scrubbed] PATH=/ux/set/bin:/ux/host:/usr/local/bin:/usr/bin:/bin -rw-r--r-- 1 user group 7 Jan  1 00:00 a"
	if found := Check([]byte(clean)); len(found) > 0 {
		t.Errorf("Check(%q) = %q", clean, found)
	}
	if !strings.Contains(strings.Join(Check([]byte("x metais-ux-placeholder")), ""), "credential") {
		t.Error("the placeholder is not named a credential")
	}
}
