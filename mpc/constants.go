package main

import (
	"crypto/rand"
	"encoding/binary"
)

const PRIME uint64 = 65537

func random(size int) []uint64 {
	// read random bytes
	bs := make([]byte, size*8, size*8)
	n, err := rand.Read(bs)
	if err != nil || n != size*8 {
		panic(err)
	}

	// convert to uint64
	nums := make([]uint64, size)
	for i := 0; i < size; i++ {
		s := i * 8
		e := (i + 1) * 8
		nums = append(nums, binary.LittleEndian.Uint64(bs[s:e]))
	}
	return nums
}
