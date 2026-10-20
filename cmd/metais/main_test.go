// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/kottera/metais/internal/specdata"
	"github.com/kottera/metais/internal/testenv"
)

// noLoader refuses every URL: every schema is given to the compiler.
type noLoader struct{}

func (noLoader) Load(url string) (any, error) { return nil, errors.New("not loaded: " + url) }

// validateDocument validates b, a document the binary printed, against
// its format's committed schema under docs/cli/schemas (ADR-008 D16 item
// 3); a format with no schema fails.
func validateDocument(t *testing.T, b []byte) {
	t.Helper()
	var head struct{ Format string }
	if err := json.Unmarshal(b, &head); err != nil {
		t.Fatalf("not a document: %v\n%s", err, b)
	}
	c := jsonschema.NewCompiler()
	c.UseLoader(noLoader{})
	add := func(id string, src []byte) {
		doc, err := jsonschema.UnmarshalJSON(bytes.NewReader(src))
		if err != nil {
			t.Fatal(err)
		}
		if err := c.AddResource(id, doc); err != nil {
			t.Fatal(err)
		}
	}
	add("https://metais.org/schemas/1.0/finding.schema.json", specdata.FindingSchema)
	names, err := filepath.Glob("../../docs/cli/schemas/*.json")
	if err != nil {
		t.Fatal(err)
	}
	for _, n := range names {
		src, err := os.ReadFile(n)
		if err != nil {
			t.Fatal(err)
		}
		add("https://metais.org/cli/schemas/"+filepath.Base(n), src)
	}
	file := strings.Replace(head.Format, "/", ".", 1) + ".json"
	if _, err := os.Stat(filepath.Join("../../docs/cli/schemas", file)); err != nil {
		t.Fatalf("the format %q has no schema: %v", head.Format, err)
	}
	s, err := c.Compile("https://metais.org/cli/schemas/" + file)
	if err != nil {
		t.Fatal(err)
	}
	inst, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Validate(inst); err != nil {
		t.Errorf("%s does not validate: %v\n%s", head.Format, err, b)
	}
}

// TestBinary builds the tool and runs it as a subprocess, so dispatch,
// the exit mapping and the streams are proved through main as well as
// through cli.Run: a refusal exits 3, and
// with --json standard output is one valid document and standard error
// is empty.
func TestBinary(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := filepath.Join(testenv.Dir(t), "metais")
	build := goCommand(t, "build", "-trimpath", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, c := range []struct {
		args   []string
		status int
	}{
		{nil, 3},
		{[]string{"--json"}, 3},
		{[]string{"bogus"}, 3},
		{[]string{"bogus", "--json"}, 3},
		{[]string{"version", "--bogus", "--json"}, 3},
		{[]string{"version"}, 0},
		{[]string{"version", "--json"}, 0},
		{[]string{"--version"}, 0},
		{[]string{"limits", "--json"}, 0},
		{[]string{"help"}, 0},
		{[]string{"help", "--json"}, 0},
		{[]string{"limits", "--help", "--json"}, 0},
	} {
		cmd := exec.Command(bin, c.args...)
		cmd.Dir = testenv.Dir(t)
		cmd.Env = []string{"HOME=" + testenv.Dir(t), "NO_COLOR=1"}
		var stdout, stderr bytes.Buffer
		// A pipe on standard input is ignored (ADR-008 D11 item 6); that
		// nothing reads it is proved in-process, by internal/cli's tests.
		cmd.Stdin, cmd.Stdout, cmd.Stderr = bytes.NewReader([]byte("ignored")), &stdout, &stderr
		err := cmd.Run()
		status := 0
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			status = exit.ExitCode()
		} else if err != nil {
			t.Fatal(err)
		}
		if status != c.status {
			t.Errorf("metais %q exited %d, want %d\n%s", c.args, status, c.status, stderr.Bytes())
		}
		json := false
		for _, a := range c.args {
			json = json || a == "--json"
		}
		switch {
		case json:
			if stderr.Len() != 0 {
				t.Errorf("metais %q wrote standard error %q", c.args, stderr.Bytes())
			}
			validateDocument(t, stdout.Bytes())
		case c.status == 0:
			if stdout.Len() == 0 || stderr.Len() != 0 {
				t.Errorf("metais %q: stdout %q, stderr %q", c.args, stdout.Bytes(), stderr.Bytes())
			}
		default:
			if stdout.Len() != 0 || stderr.Len() == 0 {
				t.Errorf("metais %q was refused with stdout %q, stderr %q", c.args, stdout.Bytes(), stderr.Bytes())
			}
		}
		if bytes.Contains(stdout.Bytes(), []byte("\x1b[")) || bytes.Contains(stderr.Bytes(), []byte("\x1b[")) {
			t.Errorf("metais %q coloured its output under NO_COLOR", c.args)
		}
	}
}

// TestMemoryBackstop runs the memory canary's build, whose backstop
// ceiling is zero bytes, as a subprocess: every command passes the
// ceiling, and the process exits 2 through the backstop with
// internal.memory, reported as the command's own report would be
// (ADR-009 Amendment A6 item 6). It proves the stop at the process's own
// exit on the host it runs on.
func TestMemoryBackstop(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	bin := filepath.Join(testenv.Dir(t), "metais")
	build := goCommand(t, "build", "-trimpath", "-tags", "metais_canary_memory", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	for _, args := range [][]string{
		{"version"},
		{"version", "--json"},
		{"limits", "--json"},
		{"bogus", "--json"},
	} {
		cmd := exec.Command(bin, args...)
		cmd.Dir = testenv.Dir(t)
		cmd.Env = []string{"HOME=" + testenv.Dir(t), "NO_COLOR=1"}
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		var exit *exec.ExitError
		if err := cmd.Run(); !errors.As(err, &exit) || exit.ExitCode() != 2 {
			t.Errorf("metais %q ended with %v, want exit 2\n%s", args, err, stderr.Bytes())
			continue
		}
		if !slices.Contains(args, "--json") {
			if stdout.Len() != 0 || !bytes.Contains(stderr.Bytes(), []byte("internal.memory")) {
				t.Errorf("metais %q: stdout %q, stderr %q", args, stdout.Bytes(), stderr.Bytes())
			}
			continue
		}
		if stderr.Len() != 0 {
			t.Errorf("metais %q wrote standard error %q", args, stderr.Bytes())
		}
		validateDocument(t, stdout.Bytes())
		var doc struct {
			OK       bool
			Findings []struct {
				Code   string
				Params map[string]any
			}
		}
		if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
			t.Fatal(err)
		}
		if doc.OK || len(doc.Findings) != 1 || doc.Findings[0].Code != "internal.memory" || len(doc.Findings[0].Params) != 1 {
			t.Errorf("metais %q reported %s", args, stdout.Bytes())
		}
	}
}
