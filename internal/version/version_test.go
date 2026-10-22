// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package version

import (
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// TestResolve covers each branch of ADR-010 D12 item 3's rule.
func TestResolve(t *testing.T) {
	for _, tc := range []struct {
		name, release, module, want string
	}{
		{"release wins over a module version", "1.2.3", "v9.9.9", "1.2.3"},
		{"release wins over no module version", "1.0.0-rc.1", "", "1.0.0-rc.1"},
		{"release wins over a dirty module version", "1.0.0", "v1.0.0+dirty", "1.0.0"},
		{"tagged module version", "", "v1.0.0", "1.0.0"},
		{"module pre-release", "", "v1.2.0-rc.1", "1.2.0-rc.1"},
		{"module pseudo-version", "", "v0.0.0-20261018120000-abcdef123456", "0.0.0-20261018120000-abcdef123456"},
		{"module build metadata other than dirty", "", "v1.0.0+incompatible", "1.0.0+incompatible"},
		{"dirty module version", "", "v1.0.0+dirty", devel},
		{"dirty pseudo-version", "", "v0.0.0-20261018120000-abcdef123456+dirty", devel},
		{"devel build", "", "(devel)", devel},
		{"no build information", "", "", devel},
		{"no leading v", "", "1.0.0", devel},
		{"two parts", "", "v1.0", devel},
		{"leading zero", "", "v1.02.0", devel},
		{"leading zero in a numeric pre-release", "", "v1.0.0-01", devel},
		{"empty pre-release", "", "v1.0.0-", devel},
		{"empty build metadata", "", "v1.0.0+", devel},
		{"empty identifier", "", "v1.0.0-rc..1", devel},
		{"invalid character", "", "v1.0.0-rc_1", devel},
		{"not a number", "", "vX.Y.Z", devel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := resolve(tc.release, tc.module); got != tc.want {
				t.Errorf("resolve(%q, %q) = %q, want %q", tc.release, tc.module, got, tc.want)
			}
		})
	}
}

// TestStringUnreleased proves String applies resolve to this binary's own
// build information when no release variable is set, and so never returns
// a version with a leading v.
func TestStringUnreleased(t *testing.T) {
	got := String()
	if strings.HasPrefix(got, "v") || (got != devel && !isSemver(got)) {
		t.Errorf("String() = %q, want a semantic version without a leading v", got)
	}
}

// TestOverride proves Override fixes String for the test that calls it and
// restores the previous version when that test ends.
func TestOverride(t *testing.T) {
	before := String()
	t.Run("overridden", func(t *testing.T) {
		Override(t, "test")
		if got := String(); got != "test" {
			t.Errorf("String() = %q under Override, want %q", got, "test")
		}
	})
	if got := String(); got != before {
		t.Errorf("String() = %q after the overriding test ended, want %q", got, before)
	}
}

// TestReleaseFlag builds a program with the release builder's -X flag and
// proves the flag reaches the release variable: a rename of the variable or
// the package would leave a release build reporting a development version.
func TestReleaseFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("builds a binary")
	}
	bin := filepath.Join(testenv.Dir(t), "printversion")
	build := goCommand(t, "build", "-trimpath", "-o", bin,
		"-ldflags", "-X metais.org/metais/internal/version.release=1.2.3-rc.1",
		"./testdata/printversion")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build: %v\n%s", err, out)
	}
	cmd := exec.Command(bin)
	cmd.Dir = testenv.Dir(t)
	cmd.Env = []string{"HOME=" + testenv.Dir(t)}
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("printversion: %v", err)
	}
	if got := string(out); got != "1.2.3-rc.1" {
		t.Errorf("release build reports %q, want %q", got, "1.2.3-rc.1")
	}
}

// specVersion is ADR-002 D7 item 1's grammar of a specification version.
var specVersion = regexp.MustCompile(`^(0|[1-9][0-9]{0,3})\.(0|[1-9][0-9]{0,3})$`)

// TestImplements proves the implemented list is ADR-002 D8 item 1's: this
// tool implements 1.0 alone, each entry is a specification version, the
// list ascends by major then minor, and within each major it names every
// minor from .0 to the highest (D7 item 2).
func TestImplements(t *testing.T) {
	got := Implements()
	if want := []string{"1.0"}; !slices.Equal(got, want) {
		t.Errorf("Implements() = %q, want %q", got, want)
	}
	prevMajor, prevMinor := -1, -1
	for _, v := range got {
		m := specVersion.FindStringSubmatch(v)
		if m == nil {
			t.Fatalf("Implements() lists %q, which is not a specification version", v)
		}
		major, _ := strconv.Atoi(m[1])
		minor, _ := strconv.Atoi(m[2])
		switch {
		case major > prevMajor && minor != 0:
			t.Errorf("Implements() starts major %d at %q, want %d.0", major, v, major)
		case major == prevMajor && minor != prevMinor+1:
			t.Errorf("Implements() lists %q after %d.%d, want %d.%d", v, prevMajor, prevMinor, major, prevMinor+1)
		case major < prevMajor:
			t.Errorf("Implements() lists %q after major %d, want ascending order", v, prevMajor)
		}
		prevMajor, prevMinor = major, minor
	}
}

// TestImplementsIsACopy proves a caller cannot change what the tool says it
// implements by changing the slice it was given.
func TestImplementsIsACopy(t *testing.T) {
	Implements()[0] = "9.9"
	if got := Implements(); got[0] != "1.0" {
		t.Errorf("Implements() = %q after a caller changed its copy, want [\"1.0\"]", got)
	}
}
