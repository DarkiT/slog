package xxhash

import (
	"unsafe"
)

// Implementation based on cespare/xxhash/v2 (MIT). Provides Sum64/Sum64String API
// used by cache optimizer while avoiding external dependency.

const (
	prime1 uint64 = 11400714785074694791
	prime2 uint64 = 14029467366897019727
	prime3 uint64 = 1609587929392839161
	prime4 uint64 = 9650029242287828579
	prime5 uint64 = 2870177450012600261
)

// Sum64 computes the 64-bit xxHash of the provided byte slice.
func Sum64(b []byte) uint64 {
	n := len(b)
	var h uint64

	if n >= 32 {
		v1 := prime1
		v1 += prime2
		v2 := prime2
		v3 := uint64(0)
		v4 := uint64(0)
		v4 -= prime1

		for len(b) >= 32 {
			v1 = round(v1, u64(b[0:]))
			v2 = round(v2, u64(b[8:]))
			v3 = round(v3, u64(b[16:]))
			v4 = round(v4, u64(b[24:]))
			b = b[32:]
		}

		h = rotl64(v1, 1) + rotl64(v2, 7) + rotl64(v3, 12) + rotl64(v4, 18)

		h = mergeRound(h, v1)
		h = mergeRound(h, v2)
		h = mergeRound(h, v3)
		h = mergeRound(h, v4)
	} else {
		h = prime5
	}

	h += uint64(n)

	// Process remaining 0..31 bytes.
	for len(b) >= 8 {
		k1 := round(0, u64(b))
		h ^= k1
		h = rotl64(h, 27)*prime1 + prime4
		b = b[8:]
	}

	if len(b) >= 4 {
		h ^= uint64(u32(b)) * prime1
		h = rotl64(h, 23)*prime2 + prime3
		b = b[4:]
	}

	for _, c := range b {
		h ^= uint64(c) * prime5
		h = rotl64(h, 11) * prime1
	}

	// Final mix to avalanche bits.
	h ^= h >> 33
	h *= prime2
	h ^= h >> 29
	h *= prime3
	h ^= h >> 32
	return h
}

// Sum64String computes the 64-bit xxHash of the provided string.
// Avoids an extra allocation by reinterpreting the string header as a byte slice.
func Sum64String(s string) uint64 {
	// Use unsafe.StringData to avoid extra allocation while keeping pointer safety rules.
	b := unsafe.Slice(unsafe.StringData(s), len(s))
	return Sum64(b)
}

func round(acc, input uint64) uint64 {
	acc += input * prime2
	acc = rotl64(acc, 31)
	acc *= prime1
	return acc
}

func mergeRound(acc, val uint64) uint64 {
	acc ^= round(0, val)
	acc = acc*prime1 + prime4
	return acc
}

func rotl64(x uint64, r uint) uint64 {
	return (x << r) | (x >> (64 - r))
}

// u64 reads a little-endian uint64 from b.
func u64(b []byte) uint64 {
	_ = b[7]
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

// u32 reads a little-endian uint32 from b.
func u32(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}
