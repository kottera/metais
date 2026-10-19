// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package inputs

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
	"github.com/kottera/metais/internal/testenv"
)

// failingObject is a schema every value under hostileFailing's long name
// fails: each member's type, a member required and absent, and an array's
// two equal items.
const failingObject = `{"type": "object", "required": ["need"],
	"properties": {"list": {"type": "array", "uniqueItems": true}},
	"additionalProperties": {"type": "string"}}`

// failingPkg declares an object input, vendor, and a collection,
// articles, whose items a JSON and a YAML file may each be, both of an
// object whose members are each a failingObject.
var failingPkg = pkg{
	inputs: []map[string]any{
		slot("vendor", "inputs/vendor.json", "required", false, "forms", []any{"json"}),
		slot("articles", "inputs/list.json", "collection", true, "forms", []any{"json", "yaml"}, "required", false),
	},
	schemas: map[string]string{
		"inputs/vendor.json": `{"type": "object", "additionalProperties": ` + failingObject + `}`,
		"inputs/list.json":   `{"type": "array", "items": {"type": "object", "additionalProperties": ` + failingObject + `}}`,
	},
}

// hostileFailing returns a valid text of at least n bytes in form, JSON or
// YAML, about half of it one member name, under which one object holds
// many distinct numeric members, each a schema failure, and an array of
// two equal items. Each finding it gives has a pointer as long as half the
// text.
func hostileFailing(form string, n int) string {
	long := strings.Repeat("~/é", max(n/2, 4)/4)
	var b bytes.Buffer
	if form == "yaml" {
		b.WriteString("? \"" + long + "\"\n: {")
	} else {
		b.WriteString(`{"` + long + `": {`)
	}
	for i := 0; b.Len() < n; i++ {
		fmt.Fprintf(&b, "%q: %d, ", "é"+strconv.FormatInt(int64(i), 36), i%10)
	}
	b.WriteString(`"list": [1, 1]}`)
	if form == "yaml" {
		b.WriteString("\n")
	} else {
		b.WriteString("}")
	}
	return b.String()
}

// failingSupplies runs the inputs layer over hostileFailing's texts of n
// bytes, as a command supplies them: an input's file; a collection of a
// JSON and a YAML item, beside one that does not parse, which makes the
// collection validated in part (ADR-006 D7 item 4); and an --inputs-json
// member. It returns each supply's findings put into a report capped at
// lim's findings limit.
func failingSupplies(t testing.TB, n int, lim limits.Set) []*finding.Report {
	l := layer(t, failingPkg)
	var reports []*finding.Report
	for _, a := range []args{
		{data: dataDir(t, map[string]string{
			"vendor.json":     hostileFailing("json", n),
			"articles/a.json": hostileFailing("json", n),
			"articles/b.yaml": hostileFailing("yaml", n),
			"articles/c.json": `{`,
		})},
		{flags: []string{`--inputs-json={"vendor": ` + hostileFailing("json", n) + `}`}},
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

// encodeReport is a report as a command writes it: its findings in JSON
// and in the human form, and its summary.
func encodeReport(r *finding.Report) []byte {
	fs := r.Findings()
	b := finding.AppendJSON(nil, fs...)
	b = append(b, '\n')
	b = finding.AppendHuman(b, false, fs...)
	return r.Summary().AppendJSON(b)
}

// TestHostileFailuresReportUnchanged pins what the layer reports for small
// texts of many schema failures under a long member name, under the
// default limits and with the findings limit lowered, byte for byte as
// the code before W3-9F wrote it: the same codes, parameters, locations,
// related locations, order (ADR-006 D8) and totals. The digests were
// taken from that code's encodeReport output, with its messages bounded
// to 240 code points as W4-1 bounds them (ADR-006 D9 item 1): the bound
// changed the text of the messages that named the long pointer, and
// nothing else.
func TestHostileFailuresReportUnchanged(t *testing.T) {
	testenv.Home(t)
	for _, tt := range []struct {
		name    string
		lim     limits.Set
		digests []string
	}{
		{"default limits", limits.Defaults(), []string{
			"913560b488dab63d853f73c692a6bcf0a21d1a757f9fdfec313dfc0d9f62bc09",
			"9cf6f7c27b72c5fc0dac946279f6e1bb401ca2890d254c84cfcb4354ba06fb83",
		}},
		{"findings=7", lowered(t, "findings=7"), []string{
			"955aca680d0001f2a1aff2248a7896daa76294db0cecda2ac7c5b23648dfed7b",
			"a9a61384fe2c26b2115889a4d7a3752875b757dc0fd1b70b385a8ade44f7b010",
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for i, r := range failingSupplies(t, 1024, tt.lim) {
				out := encodeReport(r)
				d := sha256.Sum256(out)
				if got := hex.EncodeToString(d[:]); got != tt.digests[i] {
					t.Errorf("supply %d: the report's SHA-256 is %s, want %s; it begins\n%.600s", i, got, tt.digests[i], out)
				}
			}
		})
	}
}

// TestHostileFailuresPlaced says what the pinned reports hold: every
// finding is under the long name, an input's file's and an
// --inputs-json member's about vendor and a collection's under its
// item's original index, and each kind of failure is there.
func TestHostileFailuresPlaced(t *testing.T) {
	testenv.Home(t)
	long := finding.Pointer(strings.Repeat("~/é", 64))
	reports := failingSupplies(t, 512, limits.Defaults())
	codes := map[finding.Code]bool{}
	prefixes := map[string]bool{}
	for _, r := range reports {
		for _, f := range r.Findings() {
			codes[f.Code()] = true
			l, _ := f.Location()
			if f.Code() == "input.validation.partial" || f.Code() == "json.syntax" {
				continue
			}
			i := strings.Index(l.Pointer, long)
			if i < 0 {
				t.Errorf("%s about %s at %q: not under the long name", f.Code(), f.Subject().Name, l.Pointer)
				continue
			}
			prefixes[f.Subject().Name+" "+l.Pointer[:i]] = true
		}
	}
	if want := map[string]bool{"vendor ": true, "articles /0": true, "articles /1": true}; !maps(prefixes, want) {
		t.Errorf("the findings are placed at %v, want %v", prefixes, want)
	}
	for _, c := range []finding.Code{"input.type", "input.required", "input.unique_items", "input.validation.partial"} {
		if !codes[c] {
			t.Errorf("no %s among %v", c, codes)
		}
	}
}

// hostileFailuresSize names the text size a child supplies; the parent
// sets it.
const hostileFailuresSize = "METAIS_INPUTS_HOSTILE_FAILURES_SIZE"

// TestHostileFailuresLinear is ADR-009 D10 and D18 for schema validation:
// from parsing an input to the report, through the inputs layer, valid
// texts of many schema failures under one long member name cost CPU time
// linear in their size, whether a file is an input's own, a collection's
// item validated in part, or an --inputs-json member. The texts are within
// every default limit and none is raised or bypassed. Each size runs in
// its own process, re-executing this test binary, so its cost is that
// process's CPU time (ADR-010 D14 item 7); texts eight times as long may
// cost at most three times as much per byte.
func TestHostileFailuresLinear(t *testing.T) {
	if n := os.Getenv(hostileFailuresSize); n != "" {
		runHostileFailures(t, n)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Home(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != hostileFailuresSize {
			env = append(env, kv)
		}
	}
	const small, large = 1 << 15, 1 << 18
	sizes := []int{small, large}
	cpu := make([]time.Duration, len(sizes))
	var wg sync.WaitGroup
	for i, n := range sizes {
		wg.Go(func() {
			cmd := exec.Command(exe, "-test.run=^TestHostileFailuresLinear$", "-test.count=1", "-test.v",
				"-test.timeout="+(4*hostileMeetingCPU).String())
			cmd.Env = slices.Concat(env, []string{hostileFailuresSize + "=" + strconv.Itoa(n)})
			out, err := cmd.CombinedOutput()
			cpu[i] = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Errorf("%d bytes: %v after %v of CPU time\n%s", n, err, cpu[i], out[max(0, len(out)-2048):])
				return
			}
			if !strings.Contains(string(out), "hostile failures supplied: "+strconv.Itoa(n)) {
				t.Errorf("%d bytes: the child did not supply them:\n%s", n, out)
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

// runHostileFailures is the child's half: it supplies the texts of the
// size named, under the default limits and with the findings limit
// lowered.
func runHostileFailures(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	full := failingSupplies(t, n, limits.Defaults())
	capped := failingSupplies(t, n, lowered(t, "findings=10"))
	for i, r := range full {
		s, c := r.Summary(), capped[i].Summary()
		if s.Errors < n/64 || c.Errors != s.Errors || c.Listed != 10 {
			t.Fatalf("supply %d of %d bytes: %+v, and capped %+v", i, n, s, c)
		}
	}
	fmt.Printf("hostile failures supplied: %d\n", n)
}
