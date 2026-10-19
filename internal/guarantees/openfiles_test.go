// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// manyOutputs writes, below dir, a package of n template outputs spread
// over seven directories, and returns its path.
func manyOutputs(t *testing.T, dir string, n int) string {
	t.Helper()
	pkg := filepath.Join(dir, "package")
	if err := os.MkdirAll(filepath.Join(pkg, "templates", "d"), 0o755); err != nil {
		t.Fatal(err)
	}
	var m strings.Builder
	m.WriteString("metais: \"1.0\"\nname: many\nversion: 1.0.0\ntitle: Many\ndescription: Many outputs.\nguide: Render.\ninputs: []\noutputs:\n")
	for i := range n {
		fmt.Fprintf(&m, "  - id: o%d\n    template: d/t.txt.liquid\n    path: d%d/f%d.txt\n", i, i%7, i)
	}
	if err := os.WriteFile(filepath.Join(pkg, "metais.yaml"), []byte(m.String()), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkg, "templates", "d", "t.txt.liquid"), []byte("{{ output.path }}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return pkg
}

// nofileLimit is the open-file limit, soft and hard, the open-file stub
// runs its program under: well below darwin's default soft limit of 256,
// so a write that held a handle per file it wrote would meet it.
const nofileLimit = 64

// TestRenderWithinOpenFileLimit is the first of the Wave 4 brief's hostfs
// constraints for render (W4-3.10): a render within ADR-009's limits does
// not fail because of the host's open-file limit. The built tool renders
// 300 outputs into a new directory under an open-file limit of 64, soft and
// hard, far below the outputs it writes, and writes every one of them; a
// write that held a handle per file it created would run out.
func TestRenderWithinOpenFileLimit(t *testing.T) {
	testenv.Home(t)
	const n = 300
	bin := buildMetais(t)
	dir := testenv.Dir(t)
	pkg := manyOutputs(t, dir, n)
	cmd, ok := nofileCommand(bin, []string{"render", "--json", pkg, "--out", filepath.Join(dir, "out")})
	if !ok {
		t.Skip("this platform sets no open-file limit for a process")
	}
	cmd.Dir, cmd.Env = dir, []string{"HOME=" + dir}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("render of %d outputs under an open-file limit of %d: %v\n%s%s", n, nofileLimit, err, stdout.Bytes(), stderr.Bytes())
	}
	for i := range n {
		p := filepath.Join(dir, "out", fmt.Sprintf("d%d", i%7), fmt.Sprintf("f%d.txt", i))
		if b, err := os.ReadFile(p); err != nil || string(b) != fmt.Sprintf("d%d/f%d.txt\n", i%7, i) {
			t.Fatalf("output %d: %q, %v", i, b, err)
		}
	}
}

// TestRenderUnderAClosedUmask is the third of the brief's hostfs
// constraints for render (W4-3.10): nothing depends on the umask granting
// the owner a permission. Under a umask of 777, which grants nobody
// anything, the built tool renders outputs into nested directories it
// creates, and every one is written, with the modes the umask leaves:
// none.
func TestRenderUnderAClosedUmask(t *testing.T) {
	testenv.Home(t)
	const n = 20
	bin := buildMetais(t)
	dir := testenv.Dir(t)
	pkg := manyOutputs(t, dir, n)
	out := filepath.Join(dir, "out")
	cmd, ok := closedUmaskCommand(bin, []string{"render", "--json", pkg, "--out", out})
	if !ok {
		t.Skip("this platform has no umask")
	}
	cmd.Dir, cmd.Env = dir, []string{"HOME=" + dir}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("render under a umask of 777: %v\n%s%s", err, stdout.Bytes(), stderr.Bytes())
	}
	// The test reads what the umask left unreadable by giving the owner
	// its bits back first.
	if err := os.Chmod(out, 0o755); err != nil {
		t.Fatal(err)
	}
	for i := range n {
		d := filepath.Join(out, fmt.Sprintf("d%d", i%7))
		info, err := os.Stat(d)
		if err != nil || info.Mode().Perm() != 0 && info.Mode().Perm() != 0o755 {
			t.Fatalf("directory %s: %v, %v", d, info, err)
		}
		os.Chmod(d, 0o755)
		p := filepath.Join(d, fmt.Sprintf("f%d.txt", i))
		info, err = os.Stat(p)
		if err != nil || info.Mode().Perm() != 0 {
			t.Fatalf("output %s: %v, %v; want mode 0", p, info, err)
		}
	}
}
