// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/kottera/metais/internal/finding"
	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/schema"
	"github.com/kottera/metais/internal/testenv"
)

// failingPackage returns the minimal package whose schema, about n bytes,
// fails its meta-schema many times under one long pointer: a property
// named by a long name, about half the schema, whose `required` lists
// many numbers where member names belong. Each schema.invalid finding it
// gives has a pointer as long as half the schema.
//
// The manifest gives no such case: its schema closes every object, so a
// long member name is one finding and nothing beneath it is validated.
func failingPackage(n int) map[string][]byte {
	long := strings.Repeat("~/é", max(n/2, 4)/4)
	var b bytes.Buffer
	b.WriteString(`{"type": "object", "properties": {"` + long + `": {"required": [`)
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "%d, ", i)
	}
	b.WriteString(`-1]}}}`)
	files := minimal()
	files["inputs/report.schema.json"] = b.Bytes()
	return files
}

// failingOpen opens failingPackage(n) as a source directory, as inspect
// does, and puts its findings into a report capped at lim's findings
// limit.
func failingOpen(t *testing.T, n int, lim limits.Set) *finding.Report {
	p, fs := Open(writeSource(t, failingPackage(n)), Options{Limits: lim}, 0)
	if p != nil {
		p.Close()
		t.Fatalf("a package of %d bytes whose schema is invalid opened", n)
	}
	r := finding.NewReport(int(lim.Value(limits.Findings)))
	for _, f := range fs {
		r.Add(f)
	}
	return r
}

// encodeReport is a report as a command writes it: its findings in JSON
// and in the human form, and its summary.
func encodeReport(r *finding.Report) []byte {
	fs := r.Findings()
	b := finding.AppendJSON(nil, fs...)
	b = append(b, '\n')
	b = finding.AppendHuman(b, false, fs...)
	return r.Summary().AppendJSON(b)
}

// TestFailingSchemaReportUnchanged pins what the open reports for a small
// schema of many meta-schema failures under a long pointer, under the
// default limits and with the findings limit lowered, byte for byte as
// the code before W3-9F wrote it: the same codes, parameters, locations,
// order (ADR-006 D8) and totals. The digests were taken from that code's
// encodeReport output.
func TestFailingSchemaReportUnchanged(t *testing.T) {
	testenv.Home(t)
	lowered := limits.Defaults()
	if f, ok := lowered.Lower("findings=7"); !ok {
		t.Fatal(f.Message())
	}
	for _, tt := range []struct {
		name   string
		lim    limits.Set
		digest string
	}{
		{"default limits", limits.Defaults(), "62bb787e303f1b8443e1d2abcbd9a6c726feb2b62faadedc23b54013605f8a00"},
		{"findings=7", lowered, "3d15c306600b5d46d4355bbca6e50d9db6a35ec575255c161daf887df7c3c249"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := encodeReport(failingOpen(t, 1024, tt.lim))
			d := sha256.Sum256(out)
			if got := hex.EncodeToString(d[:]); got != tt.digest {
				t.Errorf("the report's SHA-256 is %s, want %s; it begins\n%.600s", got, tt.digest, out)
			}
		})
	}
}

// TestFailingSchemaPlaced says what the pinned report holds: a
// schema.invalid finding for each number, each in the schema file under
// the long name.
func TestFailingSchemaPlaced(t *testing.T) {
	testenv.Home(t)
	long := "/properties/" + finding.Pointer(strings.Repeat("~/é", 64))[1:] + "/required/"
	r := failingOpen(t, 512, limits.Defaults())
	n := 0
	for _, f := range r.Findings() {
		l, _ := f.Location()
		if f.Code() != schema.CodeInvalid || l.File != "inputs/report.schema.json" || !strings.HasPrefix(l.Pointer, long) {
			t.Errorf("%s at %s %q: not a schema.invalid under the long name", f.Code(), l.File, l.Pointer)
			continue
		}
		n++
	}
	if s := r.Summary(); n == 0 || s.Errors < 32 {
		t.Errorf("%d findings under the long name, of %+v", n, s)
	}
}

// failingOpenSize names the size a child opens; the parent sets it.
const failingOpenSize = "METAIS_LOADER_FAILING_SIZE"

// TestFailingSchemaLinear is ADR-009 D10 and D18 for the meta-schema's
// failures: the loader's open path, which inspect runs, from parsing to
// the report, on a package whose schema fails its meta-schema many times
// under one long pointer, costs CPU time linear in its size. The packages
// are within every default limit and none is raised or bypassed. Each
// size runs in its own process, re-executing this test binary, so its
// cost is that process's CPU time (ADR-010 D14 item 7); packages eight
// times as large may cost at most three times as much per byte.
func TestFailingSchemaLinear(t *testing.T) {
	if n := os.Getenv(failingOpenSize); n != "" {
		runFailingOpen(t, n)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Home(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != failingOpenSize {
			env = append(env, kv)
		}
	}
	const small, large = 1 << 15, 1 << 18
	sizes := []int{small, large}
	cpu := make([]time.Duration, len(sizes))
	var wg sync.WaitGroup
	for i, n := range sizes {
		wg.Go(func() {
			cmd := exec.Command(exe, "-test.run=^TestFailingSchemaLinear$", "-test.count=1", "-test.v",
				"-test.timeout="+(4*hostileOpenCPU).String())
			cmd.Env = slices.Concat(env, []string{failingOpenSize + "=" + strconv.Itoa(n)})
			out, err := cmd.CombinedOutput()
			cpu[i] = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Errorf("%d bytes: %v after %v of CPU time\n%s", n, err, cpu[i], out[max(0, len(out)-2048):])
				return
			}
			if !strings.Contains(string(out), "failing packages opened: "+strconv.Itoa(n)) {
				t.Errorf("%d bytes: the child did not open them:\n%s", n, out)
			}
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	t.Logf("%d bytes: %v of CPU time; %d bytes: %v", small, cpu[0], large, cpu[1])
	if cpu[1] > hostileOpenCPU {
		t.Errorf("%d bytes cost %v of CPU time, over the bound of %v", large, cpu[1], hostileOpenCPU)
	}
	if ratio := float64(cpu[1]) / float64(cpu[0]); ratio > 3*large/small {
		t.Errorf("%d bytes cost %.1f times the CPU time of %d bytes: superlinear", large, ratio, small)
	}
}

// runFailingOpen is the child's half: it opens the package of the size
// named, under the default limits and with the findings limit lowered.
func runFailingOpen(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	lowered := limits.Defaults()
	if f, ok := lowered.Lower("findings=10"); !ok {
		t.Fatal(f.Message())
	}
	full, capped := failingOpen(t, n, limits.Defaults()), failingOpen(t, n, lowered)
	if s, c := full.Summary(), capped.Summary(); s.Errors < n/64 || c.Errors != s.Errors || c.Listed != 10 {
		t.Fatalf("a package of %d bytes: %+v, and capped %+v", n, s, c)
	}
	fmt.Printf("failing packages opened: %d\n", n)
}
