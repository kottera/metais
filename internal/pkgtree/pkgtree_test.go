// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package pkgtree

import (
	"errors"
	"os/exec"
	"slices"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/testenv"
)

func TestPathError(t *testing.T) {
	err := error(&PathError{Op: "open", Path: "a\x00/\u202eb", Err: ErrNotExist})
	if got, want := err.Error(), `open "a\x00/\u202eb": no such entry`; got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
	if !errors.Is(err, ErrNotExist) {
		t.Error("errors.Is(err, ErrNotExist) = false")
	}
}

// hostPackages are the standard packages through which a program reaches
// the host's files, or that reach them themselves: time reads the zone
// database, and io/fs, which needs time, comes with it.
var hostPackages = []string{"os", "syscall", "time", "io/fs", "path/filepath", "net", "plugin", "runtime/cgo"}

// hostDeps returns every package of hostPackages, or below one of them,
// in the non-test dependency closure of the packages patterns name.
func hostDeps(t *testing.T, patterns ...string) []string {
	t.Helper()
	testenv.Home(t)
	cmd := goCommand(t, append([]string{"list", "-deps"}, patterns...)...)
	cmd.Dir = "../.."
	// With cgo on, a package that links runtime/cgo lists it, so the
	// closure read is the widest any build of these packages has.
	cmd.Env = append(cmd.Env, "CGO_ENABLED=1")
	out, err := cmd.Output()
	if err != nil {
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			t.Fatalf("go list -deps %s: %v\n%s", strings.Join(patterns, " "), err, ee.Stderr)
		}
		t.Fatalf("go list -deps %s: %v", strings.Join(patterns, " "), err)
	}
	var found []string
	for dep := range strings.FieldsSeq(string(out)) {
		name := strings.TrimPrefix(dep, "internal/")
		if slices.ContainsFunc(hostPackages, func(h string) bool { return name == h || strings.HasPrefix(name, h+"/") }) {
			found = append(found, dep)
		}
	}
	return found
}

// TestNoHostAccess holds internal/pkgtree, and the in-memory tree and check
// under it, to performing no host file access and importing nothing that
// does: nothing they link can open a file.
func TestNoHostAccess(t *testing.T) {
	if found := hostDeps(t, "./internal/pkgtree/..."); len(found) > 0 {
		t.Errorf("internal/pkgtree/... depends on packages that reach the host: %q", found)
	}
}

// TestNoHostAccessControl runs the same check over packages that do reach
// the host, so a check gone blind fails the build.
func TestNoHostAccessControl(t *testing.T) {
	for _, pkg := range []string{"io/fs", "github.com/kottera/metais/internal/testenv"} {
		if found := hostDeps(t, pkg); len(found) == 0 {
			t.Errorf("the check found no host package below %s", pkg)
		}
	}
}
