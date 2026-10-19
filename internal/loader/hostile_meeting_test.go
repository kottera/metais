// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package loader

import (
	"bytes"
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
	"github.com/kottera/metais/internal/testenv"
)

// hostileMembers returns at least n bytes of distinct names each given
// twice, every first occurrence before every duplicate, as the members of
// a flow mapping or a JSON object.
func hostileMembers(n int) string {
	var names []string
	var b bytes.Buffer
	for i := 0; b.Len()+12*len(names) < n; i++ {
		names = append(names, "é"+strconv.FormatInt(int64(i), 36))
		fmt.Fprintf(&b, "%q: 0, ", names[i])
	}
	for i, name := range slices.Backward(names) {
		fmt.Fprintf(&b, "%q: 1", name)
		if i > 0 {
			b.WriteString(", ")
		}
	}
	return b.String()
}

// hostilePackages returns the minimal package twice made hostile at about
// n bytes: its schema a JSON object whose one long member holds many
// duplicate names, and its manifest a mapping with one long explicit key
// whose value does.
func hostilePackages(n int) []map[string][]byte {
	long := strings.Repeat("~/é", max(n/2, 4)/4)
	schema := minimal()
	schema["inputs/report.schema.json"] = []byte(`{"type": "object", "` + long + `": {` + hostileMembers(n/2) + `}}`)
	manifest := minimal()
	manifest["metais.yaml"] = append(manifest["metais.yaml"], "? \""+long+"\"\n: {"+hostileMembers(n/2)+"}\n"...)
	return []map[string][]byte{schema, manifest}
}

// hostileOpens opens each hostile package of about n bytes as a source
// directory, as inspect does, and puts its findings into a report capped
// at lim's findings limit.
func hostileOpens(t testing.TB, n int, lim limits.Set) []*finding.Report {
	var out []*finding.Report
	for _, files := range hostilePackages(n) {
		p, fs := Open(writeSource(t.(*testing.T), files), Options{Limits: lim}, 0)
		if p != nil {
			p.Close()
			t.Fatalf("a hostile package of %d bytes opened", n)
		}
		r := finding.NewReport(int(lim.Value(limits.Findings)))
		for _, f := range fs {
			r.Add(f)
		}
		out = append(out, r)
	}
	return out
}

// hostileOpenCPU bounds the CPU time, user and system, one child may
// cost, start-up included.
const hostileOpenCPU = 30 * time.Second

// hostileOpenSize names the size a child opens; the parent sets it.
const hostileOpenSize = "METAIS_LOADER_HOSTILE_SIZE"

// TestHostileOpenLinear is where W3-9 meets W3-1 (ADR-009 D10 and D18):
// the loader's open path, which inspect runs, from parsing to the report,
// on a package whose schema or manifest holds many duplicate member names
// under a long pointer, costs CPU time linear in its size. The packages
// are within every default limit and none is raised or bypassed. Each
// size runs in its own process, re-executing this test binary, so its
// cost is that process's CPU time (ADR-010 D14 item 7); packages eight
// times as large may cost at most three times as much per byte.
func TestHostileOpenLinear(t *testing.T) {
	if n := os.Getenv(hostileOpenSize); n != "" {
		runHostileOpen(t, n)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Home(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != hostileOpenSize {
			env = append(env, kv)
		}
	}
	const small, large = 1 << 15, 1 << 18
	sizes := []int{small, large}
	cpu := make([]time.Duration, len(sizes))
	var wg sync.WaitGroup
	for i, n := range sizes {
		wg.Go(func() {
			cmd := exec.Command(exe, "-test.run=^TestHostileOpenLinear$", "-test.count=1", "-test.v",
				"-test.timeout="+(4*hostileOpenCPU).String())
			cmd.Env = slices.Concat(env, []string{hostileOpenSize + "=" + strconv.Itoa(n)})
			out, err := cmd.CombinedOutput()
			cpu[i] = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Errorf("%d bytes: %v after %v of CPU time\n%s", n, err, cpu[i], out[max(0, len(out)-2048):])
				return
			}
			if !strings.Contains(string(out), "hostile packages opened: "+strconv.Itoa(n)) {
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

// runHostileOpen is the child's half: it opens the packages of the size
// named, under the default limits and with the findings limit lowered.
func runHostileOpen(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	lowered := limits.Defaults()
	if f, ok := lowered.Lower("findings=10"); !ok {
		t.Fatal(f.Message())
	}
	full, capped := hostileOpens(t, n, limits.Defaults()), hostileOpens(t, n, lowered)
	for i, r := range full {
		s, c := r.Summary(), capped[i].Summary()
		if s.Errors < n/64 || c.Errors != s.Errors || c.Listed != 10 {
			t.Fatalf("package %d of %d bytes: %+v, and capped %+v", i, n, s, c)
		}
	}
	fmt.Printf("hostile packages opened: %d\n", n)
}

// TestHostileOpenFindings pins what the open reports for small hostile
// packages: the schema's duplicates as schema.syntax and the
// manifest's as manifest.yaml.duplicate_key, each under the long name.
func TestHostileOpenFindings(t *testing.T) {
	testenv.Home(t)
	long := finding.Pointer(strings.Repeat("~/é", 64))
	for i, r := range hostileOpens(t, 512, limits.Defaults()) {
		want := []finding.Code{CodeSchemaSyntax, "manifest.yaml.duplicate_key"}[i]
		n := 0
		for _, f := range r.Findings() {
			if l, _ := f.Location(); f.Code() == want && strings.HasPrefix(l.Pointer, long+"/") {
				n++
			}
		}
		if n == 0 {
			t.Errorf("package %d: no %s under the long name in %+v", i, want, r.Summary())
		}
	}
}
