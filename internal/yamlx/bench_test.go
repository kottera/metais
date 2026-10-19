// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package yamlx

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// benchSize is ADR-010 D15 item 1's input: 10 MiB.
const benchSize = 10 << 20

// benchInput is a YAML input of at least size bytes: a block
// sequence of records of every kind of scalar, plain and quoted, a flow
// sequence and a nested mapping, the shape a data file has.
func benchInput(size int) []byte {
	var s strings.Builder
	s.WriteString("# records\n")
	for i := 0; s.Len() < size; i++ {
		fmt.Fprintf(&s, "- id: %d\n  name: \"Record %d \\u00e9 \\\"quoted\\\"\"\n  plain: record %d of the file\n  amount: %d.%03d\n  ok: %t\n  note: null\n  tags: [t%d, t%d]\n  where:\n    line: %d\n    file: 'data/%d.yaml'\n",
			i, i, i, i*7, i%1000, i%2 == 0, i%13, i%17, i, i%100)
	}
	return []byte(s.String())
}

// BenchmarkParse is ADR-010 D15 item 1's YAML parse of a 10 MiB input,
// at the default limits, its position table included.
func BenchmarkParse(b *testing.B) {
	testenv.Home(b)
	src := benchInput(benchSize)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		if _, fs := Parse(src, inputSrc, limits.Defaults()); len(fs) > 0 {
			b.Fatal(fs[0])
		}
	}
}

// TestBenchInput holds the benchmark's input to its size, and its records
// to being read without a finding, in every go test. The records are read
// from 64 KiB of them, since every record is written alike, so that the
// race-detecting gate does not parse 10 MiB.
func TestBenchInput(t *testing.T) {
	testenv.Home(t)
	if n := len(benchInput(benchSize)); n < benchSize || n > benchSize+1024 {
		t.Errorf("the input is %d bytes, want 10 MiB", n)
	}
	src := benchInput(64 << 10)
	d, fs := Parse(src, inputSrc, limits.Defaults())
	for _, f := range fs {
		t.Fatalf("the input parses with %d findings: %s", len(fs), f.Message())
	}
	if d.Value == nil {
		t.Error("the input parses to no value")
	}
}
