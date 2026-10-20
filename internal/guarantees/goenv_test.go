// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

// TestGoCommandTakesNoHostGoEnv holds the go command a test runs to taking
// nothing from the environment the test runs in (ADR-010 D14 item 1). The
// test points HOME at a temporary directory, as every test that builds the
// tool does, and empties GOMODCACHE, GOPATH and GOFLAGS, as a developer host
// that sets none of them has them. A go command that read its environment
// would keep its module cache under that HOME, fetch the module's
// dependencies into it and leave them read only, where RemoveAll cannot
// delete them; with GOPROXY=off it cannot fetch them, and fails.
//
// buildMetais must build anyway, writing nothing under HOME's go
// directory, and go env in goCommand's environment must report the host's
// module cache, no proxy, no checksum database, a read-only module graph
// and the local toolchain.
func TestGoCommandTakesNoHostGoEnv(t *testing.T) {
	if testing.Short() {
		t.Skip("builds the binary")
	}
	home := testenv.Home(t)
	for k, v := range map[string]string{
		"GOMODCACHE": "", "GOPATH": "", "GOFLAGS": "-mod=mod",
		"GOPROXY": "off", "GOSUMDB": "sum.golang.org", "GOENV": "",
	} {
		t.Setenv(k, v)
	}
	h, err := goHost()
	if err != nil {
		t.Fatal(err)
	}

	cmd := goCommand(t, "env", "-json", "GOMODCACHE", "GOPROXY", "GOSUMDB", "GOFLAGS", "GOTOOLCHAIN")
	cmd.Dir = moduleRootAbs(t)
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("go env: %v", err)
	}
	var got map[string]string
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("go env -json: %v\n%s", err, out)
	}
	for k, want := range map[string]string{
		"GOMODCACHE":  h.GOMODCACHE,
		"GOPROXY":     "off",
		"GOSUMDB":     "off",
		"GOFLAGS":     "-mod=readonly -buildvcs=false",
		"GOTOOLCHAIN": "local",
	} {
		if got[k] != want {
			t.Errorf("go env %s = %q in goCommand's environment, want %q", k, got[k], want)
		}
	}

	bin := buildMetais(t)
	if _, err := os.Stat(bin); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Lstat(filepath.Join(home, "go")); !errors.Is(err, fs.ErrNotExist) {
		t.Errorf("building the tool wrote %s (%v): the go command kept its module cache under the test's HOME", filepath.Join(home, "go"), err)
	}
}
