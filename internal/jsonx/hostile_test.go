// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package jsonx

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/rand/v2"
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

// hostile returns a JSON text of at least n bytes, about half of it one
// member name, under which one object holds distinct names each given
// twice: every first occurrence, then every duplicate in reverse order,
// joined by sep. Each json.duplicate_name finding it gives has a pointer
// as long as half the text, and, with sep a comma, a column as far along
// one line, with its first occurrence's far behind it. Three in four of
// the first occurrences hold a lone surrogate, a number out of range or a
// noncharacter, whose findings' messages name the field by its pointer.
// The name escapes to ~0 and ~1 in a pointer, and the duplicated names
// hold a multi-byte character, so columns count code points.
func hostile(n int, sep string) []byte {
	long := strings.Repeat("~/é", max(n/2, 4)/4)
	var names []string
	size := len(long)
	for i := 0; size < n; i++ {
		name := "é" + strconv.FormatInt(int64(i), 36)
		names = append(names, name)
		size += 2 * (len(name) + 4 + len(sep))
	}
	var b bytes.Buffer
	b.WriteString(`{"` + long + `": {`)
	for i, name := range names {
		if i > 0 {
			b.WriteString(sep)
		}
		fmt.Fprintf(&b, "%q:%s", name, []string{`"\ud800"`, "1e400", `"\uFFFE"`, "7"}[i%4])
	}
	for _, name := range slices.Backward(names) {
		fmt.Fprintf(&b, "%s%q:%d", sep, name, len(name)%10)
	}
	b.WriteString("}}")
	return b.Bytes()
}

// hostileReport parses src and puts every finding it gives into a report
// capped at the findings limit of lim, as a command does.
func hostileReport(t *testing.T, src []byte, lim limits.Set) *finding.Report {
	t.Helper()
	if n := int64(len(src)); n > lim.Value(limits.InputSize) {
		t.Fatalf("the text is %d bytes, over input-size", n)
	}
	_, fs := Parse(src, vendor, lim)
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

// TestHostileReportUnchanged pins the report a text of many duplicate
// names under a long pointer gives, under the default limits and with the
// findings limit lowered, byte for byte as the code before W3-9 wrote it:
// the same codes, parameters, locations, related locations, order and
// totals. The digests were taken from that code's encodeReport output,
// with its messages bounded to 240 code points as W4-1 bounds them
// (ADR-006 D9 item 1): the bound changed the text of the messages that
// named the long pointer, and nothing else.
func TestHostileReportUnchanged(t *testing.T) {
	testenv.Home(t)
	for _, tt := range []struct {
		name   string
		src    []byte
		lim    limits.Set
		digest string
	}{
		{"one line, default limits", hostile(4096, ","), limits.Defaults(), "95bdc074d3ae528502b93c7378879be5558c30a0a19e19a3434b8183447b688d"},
		{"one line, findings=7", hostile(4096, ","), lowered(t, "findings=7"), "b3efa4275b6b53f260b2226ab36cef2b499580c6b6f9665097881b9771193baf"},
		{"a line each, default limits", hostile(4096, ",\n"), limits.Defaults(), "0f6f734b0c7f371dd14b2f6d8ecf9a1c34c7989c6047e36ec268709f17467cef"},
		{"a line each, findings=7", hostile(4096, ",\r\n "), lowered(t, "findings=7"), "3ef5007f288d63b88e6d8de83dcfca0c396c6ed73cfaabcc7976348625a1ee20"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			out := encodeReport(hostileReport(t, tt.src, tt.lim))
			d := sha256.Sum256(out)
			if got := hex.EncodeToString(d[:]); got != tt.digest {
				t.Errorf("the report's SHA-256 is %s, want %s; it begins\n%.600s", got, tt.digest, out)
			}
		})
	}
}

// hostileCPU bounds the CPU time, user and system, one hostile child may
// cost, start-up included. A linear reading of the largest text below
// needs a few seconds under the race detector; the quadratic code before
// W3-9 needed minutes.
const hostileCPU = 30 * time.Second

// hostileSize names the text size a child reads; the parent sets it.
const hostileSize = "METAIS_JSONX_HOSTILE_SIZE"

// TestHostileLinear pins ADR-009 D10 and D18 for finding construction:
// from parsing to the report, a text of many duplicate member names under
// a long pointer, on one line, costs CPU time linear in the text's size.
// The texts are within every default limit and none is raised or
// bypassed. Each size runs in its own process, re-executing this test
// binary, so its cost is that process's CPU time (ADR-010 D14 item 7). A
// text eight times as long may cost at most three times as much per byte:
// a linear reading costs about the same per byte, and a quadratic one
// eight times as much.
func TestHostileLinear(t *testing.T) {
	if n := os.Getenv(hostileSize); n != "" {
		runHostile(t, n)
		return
	}
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := testenv.Home(t)
	env := []string{"HOME=" + home, "USERPROFILE=" + home}
	for _, kv := range os.Environ() {
		if k, _, _ := strings.Cut(kv, "="); k != "HOME" && k != "USERPROFILE" && k != hostileSize {
			env = append(env, kv)
		}
	}
	const small, large = 1 << 17, 1 << 20
	sizes := []int{small, large}
	cpu := make([]time.Duration, len(sizes))
	var wg sync.WaitGroup
	for i, n := range sizes {
		wg.Go(func() {
			cmd := exec.Command(exe, "-test.run=^TestHostileLinear$", "-test.count=1", "-test.v",
				"-test.timeout="+(4*hostileCPU).String())
			cmd.Env = slices.Concat(env, []string{hostileSize + "=" + strconv.Itoa(n)})
			out, err := cmd.CombinedOutput()
			cpu[i] = cmd.ProcessState.UserTime() + cmd.ProcessState.SystemTime()
			if err != nil {
				t.Errorf("%d bytes: %v after %v of CPU time\n%s", n, err, cpu[i], out[max(0, len(out)-2048):])
				return
			}
			if !strings.Contains(string(out), "hostile text read: "+strconv.Itoa(n)) {
				t.Errorf("%d bytes: the child did not read it:\n%s", n, out)
			}
		})
	}
	wg.Wait()
	if t.Failed() {
		return
	}
	t.Logf("%d bytes: %v of CPU time; %d bytes: %v", small, cpu[0], large, cpu[1])
	if cpu[1] > hostileCPU {
		t.Errorf("%d bytes cost %v of CPU time, over the bound of %v", large, cpu[1], hostileCPU)
	}
	if ratio := float64(cpu[1]) / float64(cpu[0]); ratio > 3*large/small {
		t.Errorf("%d bytes cost %.1f times the CPU time of %d bytes: superlinear", large, ratio, small)
	}
}

// runHostile is the child's half: it reads the text of the size named,
// and puts its findings into a report under the default limits and, with
// the findings limit lowered, writes the report out.
func runHostile(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	src := hostile(n, ",")
	r := hostileReport(t, src, limits.Defaults())
	s := r.Summary()
	if s.Errors < n/64 || s.Omitted != 0 {
		t.Fatalf("%d bytes gave %+v, want every duplicate listed", n, s)
	}
	capped := hostileReport(t, src, lowered(t, "findings=10"))
	if c := capped.Summary(); c.Errors != s.Errors || c.Listed != 10 {
		t.Fatalf("with findings=10, %+v; want %d errors, 10 listed", c, s.Errors)
	}
	if len(encodeReport(capped)) < 10*len(src)/2 {
		t.Fatal("the capped report does not hold its ten pointers")
	}
	fmt.Printf("hostile text read: %d\n", n)
}

// TestRepeatedNameIsOneFinding: every repeat of one member name in one
// object is reported, each at its own name with the same pointer and
// params, so by ADR-006 D8 item 3's key they are duplicates, and a report
// keeps one: the repeat that sorts first, the second occurrence. The
// report neither panics nor depends on the order the findings come in,
// and its totals count the name once.
func TestRepeatedNameIsOneFinding(t *testing.T) {
	testenv.Home(t)
	many := `{"a": 0` + strings.Repeat(`, "a": 0`, 60) + "}"
	for _, tt := range []struct {
		name, src string
		line, col int
	}{
		{"three", `{"a": 1, "a": 2, "a": 3}`, 1, 10},
		{"many", many, 1, 10},
		{"many, a line each", strings.ReplaceAll(many, ", ", ",\n "), 2, 2},
	} {
		t.Run(tt.name, func(t *testing.T) {
			_, fs := Parse([]byte(tt.src), vendor, limits.Defaults())
			if len(fs) < 2 {
				t.Fatalf("%d findings, want one per repeat", len(fs))
			}
			rng := rand.New(rand.NewPCG(1, uint64(len(fs))))
			for _, cap := range []int{100_000, 1, 0} {
				var want string
				for range 20 {
					rng.Shuffle(len(fs), func(a, b int) { fs[a], fs[b] = fs[b], fs[a] })
					r := finding.NewReport(cap)
					for _, f := range fs {
						r.Add(f)
					}
					s := r.Summary()
					if s.Errors != 1 || len(s.ByCode) == 0 || s.ByCode[0] != (finding.CodeCount{Code: CodeDuplicateName, Count: 1}) {
						t.Fatalf("cap %d: summary %+v, want one json.duplicate_name", cap, s)
					}
					if cap > 0 {
						l, _ := r.Findings()[0].Location()
						if l.Line != tt.line || l.Column != tt.col || l.Pointer != "/a" {
							t.Fatalf("cap %d: kept the repeat at %d:%d %q, want the first at %d:%d", cap, l.Line, l.Column, l.Pointer, tt.line, tt.col)
						}
					}
					got := string(encodeReport(r))
					if want == "" {
						want = got
					} else if got != want {
						t.Fatalf("cap %d: another order gave another report", cap)
					}
				}
			}
		})
	}
}
