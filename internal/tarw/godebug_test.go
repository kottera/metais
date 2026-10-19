// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package tarw

import (
	"bytes"
	"encoding/hex"
	"os"
	"os/exec"
	"slices"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

const godebugChildEnv = "METAIS_TARW_GODEBUG_CHILD"

// godebugFiles are the samples behind two paths archive/tar's reader calls
// insecure, which tarinsecurepath decides on reading, so writing would show
// the setting on them first if it ever consulted it. A render never plans
// such a path; Add does not judge paths, so it writes them.
var godebugFiles = append([]file{{"../up", false, "up"}, {"/abs", false, "abs"}}, samples...)

// TestGODEBUGChild writes godebugFiles and prints them, hex-encoded. It runs
// only as TestGODEBUG's child, so GODEBUG is set before the process starts
// and archive/tar reads it as it would in the tool.
func TestGODEBUGChild(t *testing.T) {
	if os.Getenv(godebugChildEnv) != "1" {
		t.Skip("run only as TestGODEBUG's child")
	}
	os.Stdout.WriteString("\nARCHIVE " + hex.EncodeToString(write(t, godebugFiles...)) + "\n")
}

// writeUnder runs TestGODEBUGChild with GODEBUG set to godebug and returns
// the archive it wrote.
func writeUnder(t *testing.T, godebug string) []byte {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestGODEBUGChild$", "-test.count=1")
	cmd.Env = []string{godebugChildEnv + "=1", "HOME=" + testenv.Dir(t), "GODEBUG=" + godebug}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("child under GODEBUG=%s: %v\nstdout:\n%s\nstderr:\n%s", godebug, err, stdout.Bytes(), stderr.Bytes())
	}
	_, line, ok := bytes.Cut(stdout.Bytes(), []byte("\nARCHIVE "))
	if !ok {
		t.Fatalf("child under GODEBUG=%s reported nothing; did it run?\nstdout:\n%s", godebug, stdout.Bytes())
	}
	line, _, _ = bytes.Cut(line, []byte("\n"))
	b, err := hex.DecodeString(string(line))
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// TestGODEBUG is W2-13.4 and ADR-010 D11 item 4: the archive is the same
// bytes under GODEBUG=tarinsecurepath=0 and =1, and the same as this
// process writes.
func TestGODEBUG(t *testing.T) {
	strict := writeUnder(t, "tarinsecurepath=0")
	loose := writeUnder(t, "tarinsecurepath=1")
	here := write(t, godebugFiles...)
	if err := sameBytes(strict, loose); err != nil {
		t.Errorf("tarinsecurepath=0 against =1: %v", err)
	}
	if err := sameBytes(here, strict); err != nil {
		t.Errorf("this process against tarinsecurepath=0: %v", err)
	}
	// The child really wrote the samples, so equal bytes are not two empty
	// reports agreeing.
	if len(strict) <= 2*blockSize {
		t.Fatalf("the child wrote %d bytes", len(strict))
	}
	// The comparison is red on one changed byte.
	changed := slices.Clone(loose)
	changed[len(changed)/2] ^= 0x01
	if sameBytes(strict, changed) == nil {
		t.Error("the comparison passes archives that differ in a byte")
	}
}
