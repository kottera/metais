// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

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

// hostilePkg declares an object input, vendor, and a collection, articles,
// that a JSON, a YAML and a CSV item may each be.
var hostilePkg = pkg{
	inputs: []map[string]any{
		slot("vendor", "inputs/vendor.json", "required", false, "forms", []any{"json"}),
		slot("articles", "inputs/list.json", "collection", true, "forms", []any{"json", "yaml", "csv"}, "required", false),
	},
	schemas: map[string]string{
		"inputs/vendor.json": `{"type": "object"}`,
		"inputs/list.json":   `{"type": "array", "items": {"type": "object"}}`,
	},
}

// hostileText returns a text of at least n bytes in form, about half of
// it one member name, under which one object holds distinct names each
// given twice, every first occurrence before every duplicate: for CSV,
// a header of that name over records each with a bare quote under it.
// Each finding it gives has a pointer as long as half the text.
func hostileText(form string, n int) string {
	long := strings.Repeat("~/é", max(n/2, 4)/4)
	var b bytes.Buffer
	switch form {
	case "csv":
		b.WriteString(long + "\n")
		for b.Len() < n {
			b.WriteString("a\"é\n")
		}
		return b.String()
	case "yaml":
		b.WriteString("? \"" + long + "\"\n: {")
	default:
		b.WriteString(`{"` + long + `": {`)
	}
	var names []string
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
	if form == "yaml" {
		b.WriteString("}\n")
	} else {
		b.WriteString("}}")
	}
	return b.String()
}

// hostileSupplies runs the inputs layer over hostile texts of n bytes, as
// a command supplies them: an input's file, a collection's JSON, YAML and
// CSV items, and an --inputs-json member. It returns each supply's
// findings put into a report capped at lim's findings limit.
func hostileSupplies(t testing.TB, n int, lim limits.Set) []*finding.Report {
	l := layer(t, hostilePkg)
	var reports []*finding.Report
	for _, a := range []args{
		{data: dataDir(t, map[string]string{
			"vendor.json":     hostileText("json", n),
			"articles/a.json": hostileText("json", n),
			"articles/b.yaml": hostileText("yaml", n),
			"articles/c.csv":  hostileText("csv", n),
			"articles/d.json": `{"ok": true}`,
		})},
		{flags: []string{`--inputs-json={"vendor": ` + hostileText("json", n) + `}`}},
	} {
		res := run(t, l, lim, a)
		r := finding.NewReport(int(lim.Value(limits.Findings)))
		for _, f := range res.Findings {
			r.Add(f)
		}
		reports = append(reports, r)
	}
	return reports
}

// hostileMeetingCPU bounds the CPU time, user and system, one child may
// cost, start-up included. A linear reading of the largest texts below
// needs a few seconds under the race detector.
const hostileMeetingCPU = 30 * time.Second

// hostileMeetingSize names the text size a child supplies; the parent
// sets it.
const hostileMeetingSize = "METAIS_INPUTS_HOSTILE_SIZE"

// TestHostileSupplyLinear is where W3-9 meets W3-2 (ADR-009 D10 and D18):
// from parsing an input to the report, through the inputs layer, texts
// of many duplicate member names under a long pointer cost CPU time
// linear in their size, whether a file is an input's own, a collection's
// item, whose findings the layer places under its index, or an
// --inputs-json member, whose findings it places within the input. The
// texts are within every default limit and none is raised or bypassed.
// Each size runs in its own process, re-executing this test binary, so
// its cost is that process's CPU time (ADR-010 D14 item 7); texts eight
// times as long may cost at most three times as much per byte.
func TestHostileSupplyLinear(t *testing.T) {
	if n := os.Getenv(hostileMeetingSize); n != "" {
		runHostileSupply(t, n)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Home(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != hostileMeetingSize {
			env = append(env, kv)
		}
	}
	const small, large = 1 << 15, 1 << 18
	sizes := []int{small, large}
	cpu := make([]time.Duration, len(sizes))
	var wg sync.WaitGroup
	for i, n := range sizes {
		wg.Go(func() {
			cmd := exec.Command(exe, "-test.run=^TestHostileSupplyLinear$", "-test.count=1", "-test.v",
				"-test.timeout="+(4*hostileMeetingCPU).String())
			cmd.Env = slices.Concat(env, []string{hostileMeetingSize + "=" + strconv.Itoa(n)})
			out, err := cmd.CombinedOutput()
			cpu[i] = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Errorf("%d bytes: %v after %v of CPU time\n%s", n, err, cpu[i], out[max(0, len(out)-2048):])
				return
			}
			if !strings.Contains(string(out), "hostile supplies read: "+strconv.Itoa(n)) {
				t.Errorf("%d bytes: the child did not read them:\n%s", n, out)
			}
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	t.Logf("%d bytes: %v of CPU time; %d bytes: %v", small, cpu[0], large, cpu[1])
	if cpu[1] > hostileMeetingCPU {
		t.Errorf("%d bytes cost %v of CPU time, over the bound of %v", large, cpu[1], hostileMeetingCPU)
	}
	if ratio := float64(cpu[1]) / float64(cpu[0]); ratio > 3*large/small {
		t.Errorf("%d bytes cost %.1f times the CPU time of %d bytes: superlinear", large, ratio, small)
	}
}

// runHostileSupply is the child's half: it supplies the texts of the size
// named, under the default limits and with the findings limit lowered.
func runHostileSupply(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	full := hostileSupplies(t, n, limits.Defaults())
	capped := hostileSupplies(t, n, lowered(t, "findings=10"))
	for i, r := range full {
		s, c := r.Summary(), capped[i].Summary()
		if s.Errors < n/64 || c.Errors != s.Errors || c.Listed != 10 {
			t.Fatalf("supply %d of %d bytes: %+v, and capped %+v", i, n, s, c)
		}
	}
	fmt.Printf("hostile supplies read: %d\n", n)
}

// TestHostileSupplyFindings pins what the layer reports for small hostile
// texts: a finding for each duplicate and each bad record, each about its
// input at its place in it, a collection's under the item's index and an
// --inputs-json member's within the input.
func TestHostileSupplyFindings(t *testing.T) {
	testenv.Home(t)
	reports := hostileSupplies(t, 256, limits.Defaults())
	byCode := func(r *finding.Report) map[finding.Code]int {
		m := map[finding.Code]int{}
		for _, c := range r.Summary().ByCode {
			m[c.Code] = c.Count
		}
		return m
	}
	long := finding.Pointer(strings.Repeat("~/é", 32))
	prefixes := map[string]bool{}
	for _, f := range reports[0].Findings() {
		l, _ := f.Location()
		if f.Code() == "input.validation.partial" {
			continue
		}
		if i := strings.Index(l.Pointer, long); i < 0 {
			t.Errorf("%s about %s at %q: not under the long name", f.Code(), f.Subject().Name, l.Pointer)
		} else {
			// A CSV item's findings are under its records' indexes.
			at, _, _ := strings.Cut(strings.TrimPrefix(l.Pointer[:i], "/"), "/")
			prefixes[f.Subject().Name+" "+at] = true
		}
	}
	if want := map[string]bool{"vendor ": true, "articles 0": true, "articles 1": true, "articles 2": true}; !maps(prefixes, want) {
		t.Errorf("the findings are placed at %v, want %v", prefixes, want)
	}
	if got := byCode(reports[0]); got["json.duplicate_name"] == 0 || got["yaml.duplicate_key"] == 0 || got["csv.bare_quote"] == 0 {
		t.Errorf("the data directory's findings by code: %v", got)
	}
	for _, f := range reports[1].Findings() {
		if l, _ := f.Location(); f.Code() == "json.duplicate_name" && (f.Subject().Name != "vendor" || !strings.HasPrefix(l.Pointer, long+"/")) {
			t.Errorf("an --inputs-json finding about %q at %q, want vendor's, within it", f.Subject().Name, l.Pointer)
		}
	}
}

func maps(a, b map[string]bool) bool {
	if len(a) != len(b) {
		return false
	}
	for k := range a {
		if !b[k] {
			return false
		}
	}
	return true
}
