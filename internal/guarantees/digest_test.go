// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"metais.org/metais/internal/cli"
	"metais.org/metais/internal/fault"
	"metais.org/metais/internal/testenv"
)

// digestMembers are the hostile corpus's digest-mismatch members: every
// committed package the suite expects refused with a package.digest code
// (ADR-010 D13 item 4's digest detector).
func digestMembers(t *testing.T) []string {
	t.Helper()
	root := filepath.Join(moduleRootAbs(t), "spec", "conformance", "vectors")
	files, err := filepath.Glob(filepath.Join(root, "*", "*", "expected", "findings.json"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, f := range files {
		b, err := os.ReadFile(f)
		if err != nil {
			t.Fatal(err)
		}
		pkg := filepath.Join(filepath.Dir(filepath.Dir(f)), "package.metais")
		if _, err := os.Stat(pkg); err == nil && bytes.Contains(b, []byte(`"code": "package.digest.`)) {
			out = append(out, pkg)
		}
	}
	if len(out) == 0 {
		t.Fatal("the suite holds no digest-mismatch package")
	}
	return out
}

// detectDigests is the detector: every digest-mismatch member is refused
// by inspect under faults. It returns the members that opened.
func detectDigests(t *testing.T, members []string, faults fault.Set) []string {
	t.Helper()
	var opened []string
	for _, pkg := range members {
		var stdout, stderr bytes.Buffer
		cli.Run(context.Background(), cli.Invocation{
			Args:   []string{"inspect", "--json", pkg},
			Stdin:  strings.NewReader(""),
			Stdout: &stdout, Stderr: &stderr,
			Options: cli.Options{Faults: faults},
		})
		var doc struct {
			OK bool `json:"ok"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &doc); err != nil {
			t.Fatalf("%s: %v: %q", pkg, err, stdout.String())
		}
		if doc.OK {
			opened = append(opened, filepath.Base(filepath.Dir(pkg)))
		}
	}
	return opened
}

// TestDigestDetector is ADR-010 D13's digest guarantee (R1 item 5,
// W3-1.4): every digest-mismatch member of the hostile corpus is refused.
// Its control, in the same run, is the SkipDigest fault: with verification
// made to return success, the detector must see a member through, so a
// detector gone blind fails here.
func TestDigestDetector(t *testing.T) {
	testenv.Home(t)
	members := digestMembers(t)
	if opened := detectDigests(t, members, 0); len(opened) > 0 {
		t.Errorf("digest-mismatch packages opened: %q", opened)
	}
	if opened := detectDigests(t, members, fault.SkipDigest); len(opened) == 0 {
		t.Error("with SkipDigest, every digest-mismatch package was still refused: the detector cannot tell verification from its absence")
	}
}
