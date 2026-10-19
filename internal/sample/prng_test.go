// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
	"testing"
)

// TestSplitMix64 is ADR-007 D9 item 4: the published outputs of the
// SplitMix64 reference implementation, from state 0 and from state
// 1234567.
func TestSplitMix64(t *testing.T) {
	for _, tc := range []struct {
		state uint64
		want  []uint64
	}{
		{0, []uint64{0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4, 0x06c45d188009454f, 0xf88bb8a8724c81ec}},
		{1234567, []uint64{6457827717110365317, 3203168211198807973, 9817491932198370423}},
	} {
		g := NewSplitMix64(tc.state)
		for i, w := range tc.want {
			if got := g.Next(); got != w {
				t.Errorf("state %d, output %d: %#x, want %#x", tc.state, i, got, w)
			}
		}
	}
}

// TestStream is item 5: a decision's stream starts at the first 8 bytes,
// big-endian, of SHA-256("metais-sample" 0x00 seed input 0x00 pointer 0x00
// purpose), so each of seed, input, pointer and purpose gives another
// stream.
func TestStream(t *testing.T) {
	want := func(seed uint64, input, ptr, purpose string) uint64 {
		var b []byte
		b = append(b, "metais-sample"...)
		b = append(b, 0)
		b = binary.BigEndian.AppendUint64(b, seed)
		b = append(b, input...)
		b = append(b, 0)
		b = append(b, ptr...)
		b = append(b, 0)
		b = append(b, purpose...)
		sum := sha256.Sum256(b)
		return NewSplitMix64(binary.BigEndian.Uint64(sum[:8])).Next()
	}
	seen := map[uint64]string{}
	for _, tc := range []struct {
		seed                 uint64
		input, ptr, purpose string
	}{
		{0, "vendor", "", "value"},
		{1, "vendor", "", "value"},
		{0, "vendors", "", "value"},
		{0, "vendor", "/name", "value"},
		{0, "vendor", "/name", "include"},
		{0, "vendor", "/name", "attempt:3"},
		{1 << 63, "vendor", "/a/0", "count"},
	} {
		got := Stream(tc.seed, tc.input, tc.ptr, tc.purpose).Next()
		if w := want(tc.seed, tc.input, tc.ptr, tc.purpose); got != w {
			t.Errorf("%+v: %#x, want %#x", tc, got, w)
		}
		if prior, dup := seen[got]; dup {
			t.Errorf("%+v gives the stream of %s", tc, prior)
		}
		seen[got] = tc.input + tc.ptr + tc.purpose
	}
	if attempt(3) != "attempt:3" {
		t.Errorf("attempt(3) is %q", attempt(3))
	}
}

// TestBelow is item 6: an integer in [0, n) drawn by rejection — draw u,
// reject it when u ≥ 2^64 − (2^64 mod n), return u mod n.
func TestBelow(t *testing.T) {
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	rejected := 0
	for _, n := range []uint64{1, 2, 3, 7, 100, 1 << 32, 1<<63 + 1, 1<<64 - 1, 1 << 63} {
		limit := new(big.Int).Sub(two64, new(big.Int).Mod(two64, new(big.Int).SetUint64(n)))
		g, ref := NewSplitMix64(n), NewSplitMix64(n)
		for range 50 {
			got := g.Below(n)
			var want uint64
			for {
				u := ref.Next()
				if new(big.Int).SetUint64(u).Cmp(limit) < 0 {
					want = u % n
					break
				}
				if n == 1<<63+1 {
					rejected++
				}
			}
			if got != want || got >= n {
				t.Fatalf("Below(%d) = %d, want %d", n, got, want)
			}
		}
	}
	if rejected == 0 {
		t.Error("no draw was rejected for n = 2^63+1, where almost half are")
	}
}

// TestBelowBig is item 6's wide range: up to 2^64 it is one draw, as
// Below; wider, draws are concatenated, most significant first.
func TestBelowBig(t *testing.T) {
	two64 := new(big.Int).Lsh(big.NewInt(1), 64)
	g, ref := NewSplitMix64(5), NewSplitMix64(5)
	if got, want := g.BelowBig(two64), new(big.Int).SetUint64(ref.Next()); got.Cmp(want) != 0 {
		t.Errorf("BelowBig(2^64) = %v, want the draw %v", got, want)
	}
	g, ref = NewSplitMix64(5), NewSplitMix64(5)
	if got, want := g.BelowBig(big.NewInt(1000)), ref.Below(1000); got.Uint64() != want {
		t.Errorf("BelowBig(1000) = %v, want %d", got, want)
	}
	// 2^128: two draws, concatenated, and no rejection.
	n := new(big.Int).Lsh(big.NewInt(1), 128)
	g, ref = NewSplitMix64(6), NewSplitMix64(6)
	hi, lo := ref.Next(), ref.Next()
	want := new(big.Int).Lsh(new(big.Int).SetUint64(hi), 64)
	want.Or(want, new(big.Int).SetUint64(lo))
	if got := g.BelowBig(n); got.Cmp(want) != 0 {
		t.Errorf("BelowBig(2^128) = %v, want %v", got, want)
	}
	// 2^64+1: two draws, reduced modulo n, always below it.
	n = new(big.Int).Add(two64, big.NewInt(1))
	for range 20 {
		if got := g.BelowBig(n); got.Sign() < 0 || got.Cmp(n) >= 0 {
			t.Fatalf("BelowBig(2^64+1) = %v", got)
		}
	}
}
