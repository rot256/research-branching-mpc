package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"testing"
)

func reconstruct(shares [][]uint64) []uint64 {
	size := len(shares[0])
	res := make([]uint64, size)
	for _, s := range shares {
		if len(s) != size {
			panic("Invalid number of shares")
		}
		for i := 0; i < len(s); i++ {
			res[i] = add(res[i], s[i])
		}
	}
	return res
}

func testOIPn(branches, length, players int) {

	fmt.Println("branches:", branches, "length:", length, "players:", players)

	conns := NDummies(players)

	var wg sync.WaitGroup

	// sample random map
	m := make([][]int, branches)
	for b := 0; b < branches; b++ {
		for i := 0; i < length; i++ {
			m[b] = append(m[b], rand.Intn(length))
		}
	}

	// sample random samples for
	//   B = (b[0] + b[1] + ... + b[p-1])
	//   V = (v[0] + v[1] + ... + v[p-1])
	v := make([][]uint64, players)
	b := make([][]uint64, players)
	for p := 0; p < players; p++ {
		v[p] = random(length)
		b[p] = random(branches)
	}

	// compute
	//   R = (r[0] + r[1] + ... + r[p-1])
	//   R =
	//        B[0] * m[0](V)
	//      + B[1] * m[1](V)
	//      + ...
	//      + B[_] * m[_](V)
	oip_res := make([][]uint64, players)
	for p := 0; p < players; p++ {
		wg.Add(1)
		go func(players, me int) {
			defer wg.Done()
			oip, err := NewOIP(
				conns[me],
				me,
				players,
			)
			if err != nil {
				panic(err)
			}

			// run OIP and save result
			res, err := oip.OIPMapping(m, b[me], v[me])
			if err != nil {
				panic(err)
			}
			oip_res[me] = res
		}(players, p)
	}

	wg.Wait()

	// check correctness
	B := reconstruct(b)
	V := reconstruct(v)
	R := reconstruct(oip_res)
	r := make([]uint64, length)

	for i := 0; i < len(r); i++ {
		for branch := 0; branch < branches; branch++ {
			f := mul(B[branch], V[m[branch][i]])
			r[i] = add(r[i], f)
		}
	}

	for j := 0; j < len(r); j++ {
		if R[j] != r[j] {
			log.Fatal("At position: ", j, R[j], " != ", r[j])
		}
	}

}

func BenchmarkOIP_P2_B16_L20(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testOIPn(16, 1<<20, 2)
	}
}

func BenchmarkOIP_P2_B2_L20(b *testing.B) {
	for i := 0; i < b.N; i++ {
		testOIPn(2, 1<<20, 2)
	}
}

func TestOIP(t *testing.T) {
	// testOIPn(2, 1<<20, 2)
	// testOIPn(2, 100, 2)
	for i := 0; i < 32; i++ {
		testOIPn(2, 4097, 2)
	}
}
