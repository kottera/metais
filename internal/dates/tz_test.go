// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package dates

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"metais.org/metais/internal/liquid/value"
	"metais.org/metais/internal/testenv"
)

// Go reads TZ once, when time.Local is first used, so a test cannot change
// the process's zone in place. TestSameUnderEveryZone runs this test binary
// again as a child under each TZ, and the child, TestZoneChild, writes every
// result of the package's corpus. The results must be byte-identical.
const zoneChildEnv = "METAIS_DATES_ZONE_CHILD"

// zones are the TZ values the children run under: UTC; two zones whose
// offset is not a whole number of hours, one each side of UTC; and
// Chatham's, which is neither whole nor half.
var zones = []string{"UTC", "Pacific/Chatham", "America/St_Johns", "Asia/Kathmandu"}

// zoneReport is what a child writes.
type zoneReport struct {
	// Results is every result of the corpus, in order.
	Results []string
	// LocalOffset is the child's own zone's offset at the corpus's instant,
	// from time.Local: the evidence that TZ really changed the zone.
	LocalOffset int
	// Naive is the corpus's instant formatted by time.Unix and Go's own
	// layout, which is what a formatter that reads the host's zone writes:
	// the control, which must differ from zone to zone.
	Naive string
}

// zoneCorpus is every input the children format: each format case, every
// --now and `date` input form, and texts whose offset is each child zone's
// own, which time.Parse gives that zone's location.
func zoneCorpus(t *testing.T) []string {
	var out []string
	add := func(s string, err error) {
		if err != nil {
			s = "error: " + err.Error()
		}
		out = append(out, s)
	}
	for _, tt := range formatCases {
		at, err := ParseNow(tt.at)
		if err != nil {
			t.Fatal(err)
		}
		add(FormatString(at, tt.format))
	}
	const all = "%a %A %b %B %c %C %d %D %e %F %G %H %I %j %k %l %L %m %M %N %3N %6N %9N %p %P %r %R %s %S %T %u %U %V %w %W %x %X %y %Y %z %:z %Z %%"
	texts := []string{
		"2026-10-18T09:30:00+13:45", // Chatham in October
		"2026-10-18T09:30:00+12:45", // Chatham's standard offset
		"2026-01-18T09:30:00-03:30", // St John's in January
		"2026-07-18T09:30:00-02:30", // St John's in July
		"2026-10-18T09:30:00+05:45", // Kathmandu
		"2026-10-18T09:30:00Z",
		"2026-10-18T09:30:00.25-00:00",
		"2026-10-18",
		"0000-01-01T00:00:00-23:59",
		"9999-12-31T23:59:59+23:59",
		"2026-10-18T09:30:00",
		"now",
	}
	for _, s := range texts {
		v, err := value.NewString(s)
		if err != nil {
			t.Fatal(err)
		}
		at, ok, err := FromValue(v)
		if err != nil || !ok {
			add("", err)
		} else {
			add(FormatString(at, all))
		}
		now, err := ParseNow(s)
		if err != nil {
			add("", err)
		} else {
			add(now.Normalized()+" "+fmt.Sprint(now.Offset()), nil)
		}
	}
	for _, lit := range []string{"0", "-1", "1792293603", "-62167219200", "253402300799", "1.5"} {
		n, err := value.ParseNumber(lit)
		if err != nil {
			t.Fatal(err)
		}
		at, ok, err := FromValue(n)
		if err != nil || !ok {
			add("", err)
		} else {
			add(FormatString(at, all))
		}
	}
	return out
}

// corpusInstant is the instant the child reads its own zone at.
const corpusInstant = 1792293603

func TestZoneChild(t *testing.T) {
	if os.Getenv(zoneChildEnv) != "1" {
		t.Skip("run only as TestSameUnderEveryZone's child")
	}
	local := time.Unix(corpusInstant, 0)
	_, off := local.Zone()
	r := zoneReport{
		Results:     zoneCorpus(t),
		LocalOffset: off,
		Naive:       local.Format("2006-01-02 15:04:05 -0700"),
	}
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("\nREPORT %s\n", b)
}

func runZoneChild(t *testing.T, tz string) zoneReport {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestZoneChild$", "-test.count=1")
	cmd.Env = []string{zoneChildEnv + "=1", "HOME=" + testenv.Dir(t), "TZ=" + tz}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("child under TZ=%s: %v\nstdout:\n%s\nstderr:\n%s", tz, err, stdout.Bytes(), stderr.Bytes())
	}
	_, line, ok := bytes.Cut(stdout.Bytes(), []byte("\nREPORT "))
	if !ok {
		t.Fatalf("child under TZ=%s reported nothing; did it run?\nstdout:\n%s", tz, stdout.Bytes())
	}
	line, _, _ = bytes.Cut(line, []byte("\n"))
	var r zoneReport
	if err := json.Unmarshal(line, &r); err != nil {
		t.Fatalf("child report %q: %v", line, err)
	}
	return r
}

// TestSameUnderEveryZone is ADR-010 D11 item 5's guarantee: no result
// depends on the host's time zone. Each child's own zone is checked to be
// the one TZ named, so the comparison is between truly different zones, and
// the control, a formatter that reads the zone, is checked to differ, so a
// leak of the zone into a result would be seen.
func TestSameUnderEveryZone(t *testing.T) {
	// The zones' offsets at corpusInstant, 2026-10-18T03:20:03Z: Chatham
	// and St John's are on daylight time.
	wantOffset := map[string]int{
		"UTC":              0,
		"Pacific/Chatham":  13*3600 + 45*60,
		"America/St_Johns": -(2*3600 + 30*60),
		"Asia/Kathmandu":   5*3600 + 45*60,
	}
	reports := map[string]zoneReport{}
	for _, tz := range zones {
		r := runZoneChild(t, tz)
		if r.LocalOffset != wantOffset[tz] {
			t.Fatalf("under TZ=%s the child's zone has offset %d, want %d: TZ did not change the zone", tz, r.LocalOffset, wantOffset[tz])
		}
		reports[tz] = r
	}
	base := reports[zones[0]]
	if len(base.Results) != len(zoneCorpus(t)) {
		t.Fatalf("the child wrote %d results, the corpus has %d", len(base.Results), len(zoneCorpus(t)))
	}
	naive := map[string]bool{}
	for _, tz := range zones {
		r := reports[tz]
		naive[r.Naive] = true
		if strings.Join(r.Results, "\x00") == strings.Join(base.Results, "\x00") {
			continue
		}
		for i := range base.Results {
			if i >= len(r.Results) || r.Results[i] != base.Results[i] {
				got := "<missing>"
				if i < len(r.Results) {
					got = r.Results[i]
				}
				t.Errorf("result %d under TZ=%s is %q, under TZ=%s %q", i, tz, got, zones[0], base.Results[i])
			}
		}
	}
	if len(naive) != len(zones) {
		t.Errorf("the control wrote %d distinct results under %d zones, want one each: %v", len(naive), len(zones), naive)
	}
}

// TestZoneCorpusInProcess runs the corpus in this process too, against the
// expected results, so a child that wrote the same wrong answer under every
// zone would still be caught here.
func TestZoneCorpusInProcess(t *testing.T) {
	results := zoneCorpus(t)
	for i, tt := range formatCases {
		if results[i] != tt.want {
			t.Errorf("corpus result %d = %q, want %q", i, results[i], tt.want)
		}
	}
}
