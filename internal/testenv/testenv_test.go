// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package testenv

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHome(t *testing.T) {
	outer := os.Getenv("HOME")
	var dir string
	t.Run("inside", func(t *testing.T) {
		dir = Home(t)
		if got := os.Getenv("HOME"); got != dir {
			t.Errorf("HOME = %q, want %q", got, dir)
		}
		if got := os.Getenv("USERPROFILE"); got != dir {
			t.Errorf("USERPROFILE = %q, want %q", got, dir)
		}
		if got, err := os.UserHomeDir(); err != nil || got != dir {
			t.Errorf("os.UserHomeDir() = %q, %v, want %q", got, err, dir)
		}
		assertEmptyDir(t, dir)
		if outer != "" && dir == outer {
			t.Errorf("Home returned the real home directory")
		}
	})
	if got := os.Getenv("HOME"); got != outer {
		t.Errorf("HOME after the test = %q, want it restored to %q", got, outer)
	}
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("home %q still exists after its test: %v", dir, err)
	}
}

func TestDir(t *testing.T) {
	var a, b string
	t.Run("inside", func(t *testing.T) {
		a, b = Dir(t), Dir(t)
		if a == b {
			t.Errorf("two calls returned the same directory %q", a)
		}
		assertEmptyDir(t, a)
		assertEmptyDir(t, b)
		if err := os.WriteFile(filepath.Join(a, "f"), []byte("x"), 0o600); err != nil {
			t.Fatal(err)
		}
	})
	for _, d := range []string{a, b} {
		if _, err := os.Stat(d); !os.IsNotExist(err) {
			t.Errorf("directory %q still exists after its test: %v", d, err)
		}
	}
}

func assertEmptyDir(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("%q: %v", dir, err)
	}
	if len(entries) != 0 {
		t.Errorf("%q holds %d entries, want none", dir, len(entries))
	}
}
