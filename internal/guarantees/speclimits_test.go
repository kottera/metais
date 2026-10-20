// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package guarantees

import (
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
)

// limitsChapter is the chapter whose tables give every limit's default.
const limitsChapter = "14-security.md"

// TestLimitDefaults holds the defaults chapter 14's limit tables state to
// the ones internal/limits enforces (W7-NB2): every row names a limit the
// tool has, in the unit it counts, at its default. ADR-009 D3's table is
// the design the two follow; the specification and the code are what a
// user reads and what runs, so they are the two held together.
func TestLimitDefaults(t *testing.T) {
	d, err := readDoc(specRoot, limitsChapter)
	if err != nil {
		t.Fatal(err)
	}
	report(t, limitDefaultProblems(d))
}

// TestLimitDefaultsControl breaks chapter 14's tables once per way the
// tripwire must catch, and requires each break to be caught.
func TestLimitDefaultsControl(t *testing.T) {
	d, err := readDoc(specRoot, limitsChapter)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.Join(d.lines, "\n")
	if p := limitDefaultProblems(d); len(p) > 0 {
		t.Fatalf("the chapter does not pass: %q", p)
	}
	for _, c := range []struct {
		name, old, new string
		want           []string // empty: the edit must pass
	}{
		{"a size default changed", "| `package-size` | 256 MiB |", "| `package-size` | 255 MiB |",
			[]string{"package-size has the default 255 MiB, and internal/limits 268435456 bytes"}},
		{"a count default changed", "| `schema-work` | 450,000 units |", "| `schema-work` | 14,000,000 units |",
			[]string{"schema-work has the default 14,000,000 units, and internal/limits 450000"}},
		{"a power of two changed", "| `render-work` | 2³² units |", "| `render-work` | 2³³ units |",
			[]string{"render-work has the default 2³³ units"}},
		{"a size given as a count", "| `path-length` | 512 bytes |", "| `path-length` | 512 |",
			[]string{"path-length is a size, and its default 512 names no size unit"}},
		{"a count given as a size", "| `entries` | 10,000 |", "| `entries` | 10,000 bytes |",
			[]string{"entries is not a size, and its default 10,000 bytes is one"}},
		{"a limit the tool lacks", "| `findings` | 100,000 |", "| `findingz` | 100,000 |",
			[]string{"findingz, which internal/limits has no limit by"}},
		{"a limit stated twice", "| `outputs` | 100,000 |", "| `findings` | 100,000 |",
			[]string{"findings is in the limit tables twice"}},
		{"a default that is not a number", "| `entry-ratio` | 200 |", "| `entry-ratio` | two hundred |",
			[]string{"entry-ratio has the default \"two hundred\", which is not a number"}},
		{"a malformed grouping", "| `csv-rows` | 1,000,000 |", "| `csv-rows` | 1000,000 |",
			[]string{"csv-rows has the default \"1000,000\", which is not a number"}},
		{"no limit table", "| Limit | Default | Bounds |", "| Limit | Value | Bounds |",
			[]string{"holds no limit table"}},
		{"the same value written otherwise", "| `manifest-size` | 1 MiB |", "| `manifest-size` | 1,024 KiB |", nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			if !strings.Contains(text, c.old) {
				t.Fatalf("%s holds no %q to edit", limitsChapter, c.old)
			}
			got := limitDefaultProblems(scanDoc(limitsChapter, strings.ReplaceAll(text, c.old, c.new)))
			if len(c.want) == 0 {
				if len(got) > 0 {
					t.Errorf("reports %q, want nothing", got)
				}
				return
			}
			for _, w := range c.want {
				if !slices.ContainsFunc(got, func(p string) bool { return strings.Contains(p, w) }) {
					t.Errorf("reports %q, want one saying %q", got, w)
				}
			}
		})
	}
}

const limitTableHead = "| Limit | Default | Bounds |"

var (
	limitRowRe = regexp.MustCompile("^\\| `([a-z0-9-]+)` \\| ([^|]*?) \\|")
	// limitValueRe is a default as the chapter writes it: digits grouped
	// by commas, or a power of two; then the unit, if any.
	limitValueRe = regexp.MustCompile(`^(?:([1-9][0-9]{0,2}(?:,[0-9]{3})*|0)|2([⁰¹²³⁴⁵⁶⁷⁸⁹]+))(?: ([A-Za-z]+))?$`)
	sizeUnits    = map[string]int64{"bytes": 1, "KiB": 1 << 10, "MiB": 1 << 20, "GiB": 1 << 30}
)

// limitDefaultProblems is TestLimitDefaults' check of d, chapter 14: each
// row of a table headed limitTableHead names a limit of internal/limits
// once, and states its default in a size unit exactly when the limit
// counts bytes.
func limitDefaultProblems(d *mdDoc) []string {
	var problems []string
	seen := map[string]bool{}
	tables := 0
	for i := 0; i < len(d.lines); i++ {
		if d.fenced[i] || strings.TrimSpace(d.lines[i]) != limitTableHead {
			continue
		}
		tables++
		for i += 2; i < len(d.lines) && strings.HasPrefix(d.lines[i], "|"); i++ {
			where := fmt.Sprintf("%s:%d", d.path, i+1)
			m := limitRowRe.FindStringSubmatch(d.lines[i])
			if m == nil {
				problems = append(problems, fmt.Sprintf("%s: %q is not a row naming a limit and its default", where, d.lines[i]))
				continue
			}
			name, value := m[1], m[2]
			if seen[name] {
				problems = append(problems, fmt.Sprintf("%s: %s is in the limit tables twice", where, name))
			}
			seen[name] = true
			l, ok := limits.Lookup(name)
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: the row names %s, which internal/limits has no limit by", where, name))
				continue
			}
			n, unit, ok := limitValue(value)
			if !ok {
				problems = append(problems, fmt.Sprintf("%s: %s has the default %q, which is not a number", where, name, value))
				continue
			}
			mult, isSize := sizeUnits[unit]
			switch {
			case l.Unit == limits.Bytes && !isSize:
				problems = append(problems, fmt.Sprintf("%s: %s is a size, and its default %s names no size unit", where, name, value))
			case l.Unit != limits.Bytes && isSize:
				problems = append(problems, fmt.Sprintf("%s: %s is not a size, and its default %s is one", where, name, value))
			case n*max(mult, 1) != l.Default:
				problems = append(problems, fmt.Sprintf("%s: %s has the default %s, and internal/limits %d %s", where, name, value, l.Default, l.Unit))
			}
		}
	}
	if tables == 0 {
		problems = append(problems, fmt.Sprintf("%s holds no limit table headed %q", d.path, limitTableHead))
	}
	return problems
}

// limitValue reads a default: its number, and its unit word, if any.
func limitValue(v string) (int64, string, bool) {
	m := limitValueRe.FindStringSubmatch(v)
	if m == nil {
		return 0, "", false
	}
	if m[2] != "" {
		e := 0
		for _, r := range m[2] {
			e = e*10 + slices.Index([]rune("⁰¹²³⁴⁵⁶⁷⁸⁹"), r)
		}
		if e > 62 {
			return 0, "", false
		}
		return 1 << e, m[3], true
	}
	n, err := strconv.ParseInt(strings.ReplaceAll(m[1], ",", ""), 10, 64)
	return n, m[3], err == nil
}
