// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

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

// TestGoCommandKeepsHostCToolchain holds the go command a test runs to
// keeping the C toolchain the host configured (ADR-010 D14 item 1, W7-C8).
// A cgo build, the race detector's on linux, runs the host's C compiler in
// the go command's environment. A compiler that is a wrapper needs
// variables of its own there, as the cc toolchain's shim needs
// CC_TOOLCHAIN_TARGET_GLIBC, and refuses without them.
//
// The test sets CC to fakecc, which refuses unless CC_TOOLCHAIN_TARGET_FAKE
// is set and otherwise runs the host's own compiler, and builds a cgo
// program, whose C preamble no build cache holds, in GoEnv's environment
// over that host. The build must succeed, and fakecc must have run in it.
// The go env query that reads the host runs CC too, so the marker it
// leaves is removed before the build: only the build can make it again.
func TestGoCommandKeepsHostCToolchain(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a cgo program")
	}
	host, err := goHost()
	if err != nil {
		t.Fatal(err)
	}
	var hostCC string
	for _, kv := range host.CToolchain {
		if v, ok := strings.CutPrefix(kv, "CC="); ok {
			hostCC = v
		}
	}
	realCC, err := exec.LookPath(hostCC)
	if hostCC == "" || err != nil {
		t.Skipf("the host has no C compiler (go env CC = %q: %v)", hostCC, err)
	}

	dir := testenv.Dir(t)
	fake := filepath.Join(dir, "fakecc")
	if runtime.GOOS == "windows" {
		fake += ".exe"
	}
	build := goCommand(t, "build", "-trimpath", "-o", fake, "./internal/guarantees/testdata/fakecc")
	build.Dir = moduleRootAbs(t)
	if b, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build fakecc: %v\n%s", err, b)
	}
	if err := os.WriteFile(fake+".real", []byte(realCC), 0o600); err != nil {
		t.Fatal(err)
	}

	environ := append(slices.Clone(hostEnviron), "CC="+fake, "CC_TOOLCHAIN_TARGET_FAKE=metais-test")
	query := exec.Command(host.Go(), testenv.GoHostQuery...)
	query.Env = append(slices.Clone(environ), "GOTOOLCHAIN=local")
	out, err := query.Output()
	if err != nil {
		t.Fatalf("go %s: %v", strings.Join(testenv.GoHostQuery, " "), err)
	}
	h, err := testenv.ParseGoHost(environ, out)
	if err != nil {
		t.Fatal(err)
	}

	if err := os.Remove(fake + ".ran"); err != nil && !errors.Is(err, fs.ErrNotExist) {
		t.Fatal(err)
	}

	probe := filepath.Join(dir, "probe")
	nonce := strconv.FormatInt(time.Now().UnixNano(), 10)
	if err := os.Mkdir(probe, 0o700); err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string]string{
		"go.mod":  "module cgoprobe\n\ngo 1.21\n",
		"main.go": "package main\n\n// int probe(void) { return 0; } // " + nonce + "\nimport \"C\"\n\nfunc main() { _ = C.probe() }\n",
	} {
		if err := os.WriteFile(filepath.Join(probe, name), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(h.Go(), "build", "-o", filepath.Join(dir, "probe.bin"), ".")
	cmd.Dir = probe
	cmd.Env = append(testenv.GoEnv(h, testenv.Dir(t)), "CGO_ENABLED=1")
	if b, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("a cgo build in GoEnv's environment, over a host whose CC needs its own variable: %v\n%s", err, b)
	}
	if _, err := os.Stat(fake + ".ran"); err != nil {
		t.Errorf("the cgo build never ran the host's CC, fakecc: %v", err)
	}
}

// TestGoCommandStartsNoTelemetrySidecar holds the go command a test runs to
// leaving nothing running behind it (W7-NB1). In a HOME whose telemetry
// directory is fresh, as every GoEnv HOME is, the go command takes the
// day's upload token and forks a detached telemetry sidecar, which writes
// under HOME after the command exits and races the test's removal of the
// directory: TestReleaseTakesNoHostGoConfiguration's cleanup failed with
// "directory not empty" over the sidecar's telemetry files. The token is
// taken synchronously, before the fork, so a go command that leaves no
// upload.token under its HOME started no sidecar.
func TestGoCommandStartsNoTelemetrySidecar(t *testing.T) {
	h, err := goHost()
	if err != nil {
		t.Fatal(err)
	}
	dir := testenv.Dir(t)
	cmd := exec.Command(h.Go(), "env", "GOVERSION")
	cmd.Dir = dir
	cmd.Env = testenv.GoEnv(h, dir)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("go env: %v\n%s", err, out)
	}
	err = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Name() == "upload.token" {
			t.Errorf("the go command took the telemetry upload token %s, and so started a sidecar that outlives it", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
