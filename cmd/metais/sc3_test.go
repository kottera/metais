// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"metais.org/metais/internal/testenv"
)

// statusWidget is the package of PRD §4.3, as a source directory.
const statusWidget = "../../examples/status-widget"

// statusJSON is the object a scheduled workflow fetched: the status of a
// service, as the widget's one input.
const statusJSON = `{
  "service": "Payments <API>",
  "state": "degraded",
  "updated": "2026-10-20T08:15:00Z",
  "message": "Card payments are slower than usual & some retries fail.",
  "components": [
    {"name": "Card payments", "state": "degraded"},
    {"name": "Bank transfers", "state": "operational"}
  ]
}
`

// statusHTML is the one HTML fragment the widget renders from statusJSON,
// every value escaped.
const statusHTML = `<section class="status status-degraded" aria-label="Payments &lt;API&gt; status">
  <h2>Payments &lt;API&gt;</h2>
  <p class="status-state">Degraded</p>
  <p class="status-message">Card payments are slower than usual &amp; some retries fail.</p>
  <ul class="status-components">
    <li class="status-degraded">Card payments: Degraded</li>
    <li class="status-operational">Bank transfers: Operational</li>
  </ul>
  <p class="status-updated">Updated <time datetime="2026-10-20T08:15:00Z">2026-10-20T08:15:00Z</time></p>
</section>
`

// TestSC3StatusWidget is SC3 (PRD §26): a program renders the widget
// package of §4.3 from JSON with no harness, in a single validate and
// render. It runs the built binary as a scheduled workflow would — the
// fetched JSON written to a data directory, one metais validate, one
// metais render — and nothing else: no check, no test, no sample, and
// no flag beyond the data and the output.
func TestSC3StatusWidget(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	work := testenv.Dir(t)
	bin := filepath.Join(work, "metais")
	build := goCommand(t, "build", "-trimpath", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	pkg, err := filepath.Abs(statusWidget)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(work, "data")
	if err := os.Mkdir(data, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(data, "status.json"), []byte(statusJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command(bin, args...)
		cmd.Dir = work
		cmd.Env = []string{"HOME=" + testenv.Dir(t), "NO_COLOR=1"}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		err := cmd.Run()
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			t.Fatalf("metais %q exited %d\nstdout:\n%s\nstderr:\n%s", args, exit.ExitCode(), stdout.Bytes(), stderr.Bytes())
		} else if err != nil {
			t.Fatal(err)
		}
	}

	run("validate", pkg, "--data", data)
	out := filepath.Join(work, "out")
	run("render", pkg, "--data", data, "--out", out)

	ents, err := os.ReadDir(out)
	if err != nil {
		t.Fatal(err)
	}
	if len(ents) != 1 || ents[0].Name() != "status.html" {
		var names []string
		for _, e := range ents {
			names = append(names, e.Name())
		}
		t.Fatalf("the render wrote %q, not the one fragment status.html", names)
	}
	got, err := os.ReadFile(filepath.Join(out, "status.html"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != statusHTML {
		t.Errorf("status.html:\n%s\nwant:\n%s", got, statusHTML)
	}
}
