// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

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

// hostile returns a CSV file of at least n bytes, about half of it one
// header name, under which every record but one in four has a quote
// finding: a bare quote, or text after a closing quote, in the column the
// long name names, so each finding's pointer is as long as half the file.
// The others have one field too many, a finding about the record, and a
// bare quote in the second column. The name escapes to ~0 and ~1 in a
// pointer, and holds a multi-byte character.
func hostile(n int, eol string) []byte {
	var b bytes.Buffer
	b.WriteString(strings.Repeat("~/é", max(n/2, 4)/4) + ",b" + eol)
	for i := 0; b.Len() < n; i++ {
		b.WriteString([]string{`a"é,1`, `"q"z,2`, `1,2,3`, `4,c"`}[i%4] + eol)
	}
	return b.Bytes()
}

// hostileReport reads src as an input and puts every finding it gives
// into a report capped at the findings limit of lim, as a command does.
func hostileReport(t *testing.T, src []byte, lim limits.Set) *finding.Report {
	t.Helper()
	if n := int64(len(src)); n > lim.Value(limits.InputSize) {
		t.Fatalf("the file is %d bytes, over input-size", n)
	}
	_, fs := Read(src, ',', testSource, lim)
	r := finding.NewReport(int(lim.Value(limits.Findings)))
	for _, f := range fs {
		r.Add(f)
	}
	return r
}

// loweredFindings returns the default limits with the findings limit
// lowered to n.
func loweredFindings(t *testing.T, n int) limits.Set {
	t.Helper()
	lim := limits.Defaults()
	if f, ok := lim.Lower("findings=" + strconv.Itoa(n)); !ok {
		t.Fatalf("lowering findings: %s", f.Message())
	}
	return lim
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

// TestHostileReportUnchanged pins the report a file of many findings under
// a long header name gives, under the default limits and with the
// findings limit lowered, byte for byte as the code before W3-9 wrote it:
// the same codes, parameters, locations, related locations, order and
// totals. The digests were taken from that code's encodeReport output.
func TestHostileReportUnchanged(t *testing.T) {
	testenv.Home(t)
	for _, tt := range []struct {
		name   string
		src    []byte
		lim    limits.Set
		digest string
	}{
		{"LF, default limits", hostile(4096, "\n"), limits.Defaults(), "884ac2a4fd9abcfcb6e3022d15aa53f2fac03b6c77c91c41de0d8684869673b5"},
		{"LF, findings=7", hostile(4096, "\n"), loweredFindings(t, 7), "4b336d37313e7a77e6eebe98d86f9c201de40fa1fcb25f8fa3f65c1271699c6b"},
		{"CRLF, default limits", hostile(4096, "\r\n"), limits.Defaults(), "fe5897bfe28100a998d294316887c32579974990a77a74f6912c20eb68ed529f"},
		{"CRLF, findings=7", hostile(4096, "\r\n"), loweredFindings(t, 7), "bbb668a7b91d670c318ee8c152cdc28c66062f9b564b58ca2545d97e37a1b297"},
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
// cost, start-up included. A linear reading of the largest file below
// needs a few seconds under the race detector; the quadratic code before
// W3-9 needed minutes.
const hostileCPU = 30 * time.Second

// hostileSize names the file size a child reads; the parent sets it.
const hostileSize = "METAIS_CSV_HOSTILE_SIZE"

// TestHostileLinear pins ADR-009 D10 and D18 for finding construction:
// from reading to the report, a file of many findings under a long header
// name costs CPU time linear in its size. The files are within every default limit and none is raised or
// bypassed. Each size runs in its own process, re-executing this test
// binary, so its cost is that process's CPU time (ADR-010 D14 item 7). A
// file eight times as long may cost at most three times as much per
// byte: a linear reading costs about the same per byte, and a quadratic
// one eight times as much.
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
			if !strings.Contains(string(out), "hostile file read: "+strconv.Itoa(n)) {
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

// runHostile is the child's half: it reads the file of the size named, and puts its findings into a report under the default limits
// and, with the findings limit lowered, writes the report out.
func runHostile(t *testing.T, size string) {
	n, err := strconv.Atoi(size)
	if err != nil {
		t.Fatal(err)
	}
	src := hostile(n, "\n")
	r := hostileReport(t, src, limits.Defaults())
	s := r.Summary()
	if s.Errors < n/16 {
		t.Fatalf("%d bytes gave %+v, want a finding in every record", n, s)
	}
	capped := hostileReport(t, src, loweredFindings(t, 10))
	if c := capped.Summary(); c.Errors != s.Errors || c.Listed != 10 {
		t.Fatalf("with findings=10, %+v; want %d errors, 10 listed", c, s.Errors)
	}
	if len(encodeReport(capped)) < 10*len(src)/2 {
		t.Fatal("the capped report does not hold its ten pointers")
	}
	fmt.Printf("hostile file read: %d\n", n)
}
