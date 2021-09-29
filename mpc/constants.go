package main

import (
	"crypto/rand"
	"encoding/binary"
	"math/bits"
)

const TCP_BUFFER int = 1 << 16
const PRIME uint64 = 65537

func init() {
	if bits.Len64(PRIME) > 31 {
		panic("prime too large")
	}
}

func inv(v uint64) uint64 {
	return PRIME - v
}

func add(v1, v2 uint64) uint64 {
	return (v1 + v2) % PRIME
}

func sub(v1, v2 uint64) uint64 {
	return add(v1, inv(v2))
}

func mul(v1, v2 uint64) uint64 {
	return (v1 * v2) % PRIME
}

func random(size int) []uint64 {
	// read random bytes
	bs := make([]byte, size*8, size*8)
	n, err := rand.Read(bs)
	if err != nil || n != size*8 {
		panic(err)
	}

	// convert to uint64
	nums := make([]uint64, 0, size)
	for i := 0; i < size; i++ {
		s := i * 8
		e := (i + 1) * 8
		nums = append(nums, binary.LittleEndian.Uint64(bs[s:e])%PRIME)
	}
	return nums
}
