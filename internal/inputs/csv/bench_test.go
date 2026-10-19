// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package csv

import (
	"fmt"
	"strings"
	"testing"

	"github.com/kottera/metais/internal/limits"
	"github.com/kottera/metais/internal/testenv"
)

// benchSize is ADR-010 D15 item 1's input: 10 MiB.
const benchSize = 10 << 20

// benchInput is a CSV input of at least size bytes: a header record
// and records of plain fields and quoted ones, a quote doubled and a line
// break inside one.
func benchInput(size int) []byte {
	var s strings.Builder
	s.WriteString("id,name,amount,ok,note,where\n")
	for i := 0; s.Len() < size; i++ {
		fmt.Fprintf(&s, "%d,\"Record %d, \"\"quoted\"\"\",%d.%03d,%t,,\"data/%d.csv\nline %d\"\n", i, i, i*7, i%1000, i%2 == 0, i%100, i)
	}
	return []byte(s.String())
}

// BenchmarkRead is ADR-010 D15 item 1's CSV parse of a 10 MiB input, at
// the default limits.
func BenchmarkRead(b *testing.B) {
	testenv.Home(b)
	src := benchInput(benchSize)
	b.SetBytes(int64(len(src)))
	b.ReportAllocs()
	for b.Loop() {
		if _, fs := Read(src, Comma, testSource, limits.Defaults()); len(fs) > 0 {
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
	tbl, fs := Read(src, Comma, testSource, limits.Defaults())
	for _, f := range fs {
		t.Fatalf("the input reads with %d findings: %s", len(fs), f.Message())
	}
	if len(tbl.Header) != 6 || len(tbl.Records) < 500 {
		t.Errorf("the input reads as %d header fields and %d records", len(tbl.Header), len(tbl.Records))
	}
}
