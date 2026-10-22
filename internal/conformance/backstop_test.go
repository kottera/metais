// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package conformance

import (
	"slices"
	"strings"
	"testing"

	"metais.org/metais/internal/testenv"
)

// TestHarnessesFailTheBackstop is ADR-009 D18 item 2 for Amendment A6
// item 6: a case whose command exits through the memory backstop fails,
// in the runner the hostile corpus runs under and in the calibration
// suite, whatever the case expected. The case is forced over a lowered
// ceiling by the memory canary's build, whose backstop ceiling is zero
// bytes, so every command it runs stops with internal.memory.
func TestHarnessesFailTheBackstop(t *testing.T) {
	if builtTool == "" {
		t.Skip("builds the memory canary beside the binary built from ./cmd/metais")
	}
	bin := build(t, "canary-memory", "-tags", "metais_canary_memory")
	t.Run("corpus", func(t *testing.T) {
		suite := defaultSuite()
		a, err := newAdapter("cli", bin, suite)
		if err != nil {
			t.Fatal(err)
		}
		ps := runVector(newRunner(t, suite, a), suite, "dat/required-input-missing", testenv.Dir(t))
		if !slices.ContainsFunc(ps, func(p problem) bool { return p.check == checkBackstop && !p.harness }) {
			t.Errorf("the runner did not fail the case on the %s check; it reported %q", checkBackstop, ps)
		}
	})
	t.Run("calibration", func(t *testing.T) {
		defaults, err := toolDefaults(builtTool, testenv.Dir(t))
		if err != nil {
			t.Fatal(err)
		}
		c := calCases[0]
		run, err := runCase(bin, c, defaults, defaults.with(c.lowered), testenv.Dir(t), defaultWall)
		if err == nil || !strings.Contains(err.Error(), "memory backstop") {
			t.Fatalf("the suite's run of %s through the backstop ended with %v", c.name, err)
		}
		if r := judgeCase(c, defaults, run, err); len(r.failures()) == 0 {
			t.Errorf("the suite judged %s, which exited through the backstop, a pass", c.name)
		}
	})
}
