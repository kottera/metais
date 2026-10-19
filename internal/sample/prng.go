// Copyright 2026 Kottera
// SPDX-License-Identifier: Apache-2.0

package sample

import (
	"crypto/sha256"
	"encoding/binary"
	"math/big"
)

// SplitMix64 is the generator's PRNG (ADR-007 D9 item 4): a state x, and
// a Next that adds 0x9e3779b97f4a7c15 to x and returns it mixed. It is
// METAIS code, because the standard library promises its generators'
// outputs across no release (ADR-010 D11 item 6).
type SplitMix64 struct {
	x uint64
}

// NewSplitMix64 returns the generator whose state is x.
func NewSplitMix64(x uint64) *SplitMix64 { return &SplitMix64{x: x} }

// Next advances the state and returns its next output: z = x after the
// addition, then z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9,
// z = (z ^ (z >> 27)) * 0x94d049bb133111eb, and z ^ (z >> 31), all modulo
// 2^64.
func (s *SplitMix64) Next() uint64 {
	s.x += 0x9e3779b97f4a7c15
	z := s.x
	z = (z ^ (z >> 30)) * 0xbf58476d1ce4e5b9
	z = (z ^ (z >> 27)) * 0x94d049bb133111eb
	return z ^ (z >> 31)
}

// Below returns an integer in [0, n), n at least 1, drawn by rejection
// (ADR-007 D9 item 6): draw u, reject it when u ≥ 2^64 − (2^64 mod n),
// and return u mod n.
func (s *SplitMix64) Below(n uint64) uint64 {
	if n == 0 {
		panic("sample: Below(0)")
	}
	// 2^64 mod n, computed in 64 bits as (2^64 − n) mod n.
	limit := -((-n) % n) // 2^64 − (2^64 mod n), which is 0 for a power of two: no rejection
	for {
		u := s.Next()
		if limit == 0 || u < limit {
			return u % n
		}
	}
}

var two64 = new(big.Int).Lsh(big.NewInt(1), 64)

// BelowBig returns an integer in [0, n), n at least 1. A range of at most
// 2^64 takes one draw, as Below does; a wider one concatenates draws, most
// significant first, into u below 2^(64w) for the fewest words w that
// hold n−1, and rejects u ≥ 2^(64w) − (2^(64w) mod n) (ADR-007 D9 item 6).
func (s *SplitMix64) BelowBig(n *big.Int) *big.Int {
	if n.Sign() <= 0 {
		panic("sample: BelowBig of a range that is empty")
	}
	if n.Cmp(two64) < 0 {
		return new(big.Int).SetUint64(s.Below(n.Uint64()))
	}
	words := (new(big.Int).Sub(n, big.NewInt(1)).BitLen() + 63) / 64
	span := new(big.Int).Lsh(big.NewInt(1), uint(64*words))
	limit := new(big.Int).Sub(span, new(big.Int).Mod(span, n))
	u := new(big.Int)
	for {
		u.SetUint64(0)
		for range words {
			u.Lsh(u, 64)
			u.Or(u, new(big.Int).SetUint64(s.Next()))
		}
		if u.Cmp(limit) < 0 {
			return u.Mod(u, n)
		}
	}
}

// The purposes a stream is drawn for (ADR-007 D9 item 5). An attempt's
// purpose is attempt(k).
const (
	purposeValue   = "value"
	purposeInclude = "include"
	purposeType    = "type"
	purposeBranch  = "branch"
	purposeCount   = "count"
	purposeName    = "name"
)

// attempt is the purpose of the k-th regeneration, attempt:<k>.
func attempt(k int) string { return "attempt:" + itoa(k) }

// Stream returns the stream of one decision (ADR-007 D9 item 5): the
// generator whose initial state is the first 8 bytes, big-endian, of
// SHA-256("metais-sample" ‖ 0x00 ‖ seed as 8 bytes big-endian ‖ input ‖
// 0x00 ‖ pointer ‖ 0x00 ‖ purpose). pointer is the instance JSON Pointer
// the decision is made at, so a value depends only on where it sits.
func Stream(seed uint64, input, pointer, purpose string) *SplitMix64 {
	h := sha256.New()
	h.Write([]byte("metais-sample"))
	h.Write([]byte{0})
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], seed)
	h.Write(b[:])
	h.Write([]byte(input))
	h.Write([]byte{0})
	h.Write([]byte(pointer))
	h.Write([]byte{0})
	h.Write([]byte(purpose))
	return NewSplitMix64(binary.BigEndian.Uint64(h.Sum(nil)[:8]))
}
