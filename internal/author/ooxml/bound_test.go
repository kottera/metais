// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package ooxml

import (
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/kottera/metais/internal/testenv"
)

// The two sizes each hostile template is scanned at, eight times apart. A
// scan linear in the template's length costs about eight times as much at
// the larger; a quadratic one, sixty-four.
const (
	smallSize = 512 << 10
	largeSize = 4 << 20
)

// linearRatio is the most the larger scan may cost over the smaller: twice
// what linear work costs, so a garbage collection's timing does not fail
// it, and a quarter of what quadratic work costs. linearSlack is added to
// it for a child process's start-up, which both sizes pay.
const (
	linearRatio = 16
	linearSlack = 300 * time.Millisecond
)

// graphCPU is the CPU time a hostileGraphs case may cost its process,
// start-up included. A propagation bounded by the prefixes times the edges
// needs a fraction of a second on these packages, even under the race
// detector; one that copies bindings along every path needs minutes.
const graphCPU = 5 * time.Second

// boundAtOnce is how many children run at once; each is CPU-bound.
const boundAtOnce = 4

// childBound names the case a child process runs; the parent sets it.
const childBound = "METAIS_OOXML_BOUND_CASE"

// TestScanIsLinear holds the scan to ADR-009 D10: its CPU time is linear in
// the template's length, measured by running each of hostileCases at two
// sizes, and each of hostileGraphs once, in its own process, re-executing
// this test binary, so that its cost is the process's CPU time (ADR-010
// D14 item 7) without a platform-specific call.
func TestScanIsLinear(t *testing.T) {
	if name := os.Getenv(childBound); name != "" {
		runBound(t, name)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Dir(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != childBound {
			env = append(env, kv)
		}
	}
	slots := make(chan struct{}, boundAtOnce)
	cpu := func(t *testing.T, name string) time.Duration {
		slots <- struct{}{}
		defer func() { <-slots }()
		cmd := exec.Command(exe, "-test.run=^TestScanIsLinear$", "-test.count=1", "-test.v", "-test.timeout=10m")
		cmd.Env = slices.Concat(env, []string{childBound + "=" + name})
		out, err := cmd.CombinedOutput()
		d := cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
		if err != nil {
			t.Fatalf("case %q: %v after %v of CPU time\n%s", name, err, d, out[max(0, len(out)-2048):])
		}
		if !strings.Contains(string(out), "bound case ran: "+strconv.Quote(name)) {
			t.Fatalf("case %q: the child did not run it:\n%s", name, out)
		}
		return d
	}
	for name := range hostileCases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			small := cpu(t, fmt.Sprintf("%s@%d", name, smallSize))
			large := cpu(t, fmt.Sprintf("%s@%d", name, largeSize))
			t.Logf("%s: %v at %d bytes, %v at %d", name, small, smallSize, large, largeSize)
			if large > linearRatio*small+linearSlack {
				t.Errorf("%s cost %v of CPU time at %d bytes and %v at %d, more than %d times as much", name, small, smallSize, large, largeSize, linearRatio)
			}
		})
	}
	for name := range hostileGraphs {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			d := cpu(t, name)
			t.Logf("%s: %v", name, d)
			if d > graphCPU {
				t.Errorf("%s cost %v of CPU time, over the bound of %v", name, d, graphCPU)
			}
		})
	}
}

// runBound is the child's half: it scans the one case named, a hostileCases
// name and size, or a hostileGraphs name.
func runBound(t *testing.T, name string) {
	var ts []Template
	if base, size, ok := strings.Cut(name, "@"); ok {
		n, err := strconv.Atoi(size)
		if gen := hostileCases[base]; err == nil && gen != nil {
			ts = gen(n)
		}
	} else if gen := hostileGraphs[name]; gen != nil {
		ts = gen()
	}
	if ts == nil {
		t.Fatalf("no bound case %q", name)
	}
	Scan(ts)
	t.Logf("bound case ran: %q", name)
}

// TestHostileCasesAreHostile: each case is the size it is run at, so a
// case that builds too little cannot pass the bound by doing nothing, and
// each graph is a package of thousands of templates.
func TestHostileCasesAreHostile(t *testing.T) {
	testenv.Home(t)
	for name, gen := range hostileCases {
		n := 0
		for _, tm := range gen(smallSize) {
			n += len(tm.Text)
		}
		if n < smallSize*3/4 || n > smallSize*5/4 {
			t.Errorf("%s builds %d bytes at size %d", name, n, smallSize)
		}
	}
	for name, gen := range hostileGraphs {
		if n := len(gen()); n < 1000 {
			t.Errorf("%s builds %d templates", name, n)
		}
	}
}
