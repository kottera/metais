// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"runtime"
	"testing"

	"metais.org/metais/internal/testenv"
)

// rssHeld is what TestRSSIsTheProcessAlone makes this test process hold.
const rssHeld = 256 << 20

// TestRSSIsTheProcessAlone pins ADR-011 D13 item 2's peak memory to the
// measured process alone. Go starts a process on linux with
// CLONE_VFORK|CLONE_VM, so the child runs in this process's address space
// until it executes, and the kernel's exec folds that address space's
// high-water mark into the child's ru_maxrss: measured directly, a step
// would report this test process's peak, which a whole-package run drives
// above a hostile vector's bound. Here this process holds rssHeld while the
// clean fake answers version, which on its own needs a small fraction of
// it.
func TestRSSIsTheProcessAlone(t *testing.T) {
	held := make([]byte, rssHeld)
	for i := 0; i < len(held); i += 4096 {
		held[i] = 1
	}
	p, err := runProc(fakeProgram(t, testenv.Dir(t), "clean"), []string{"version"}, testenv.Dir(t), nil, nil, defaultWall)
	runtime.KeepAlive(held)
	if err != nil {
		t.Fatal(err)
	}
	if p.rss < 0 {
		t.Fatal("no peak memory was measured")
	}
	t.Logf("the fake's peak resident memory was %d MiB while this process held %d MiB", p.rss>>20, rssHeld>>20)
	if p.rss >= rssHeld/2 {
		t.Errorf("the fake's peak resident memory was %d MiB: it counts the %d MiB this process held", p.rss>>20, rssHeld>>20)
	}
}
