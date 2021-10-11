package main

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
)

func testOIPn(branches, length, players int) {

	fmt.Println("branches:", branches, "length:", length, "players:", players)

	conns1 := NDummies(players)
	conns2 := NDummies(players)

	var wg sync.WaitGroup

	m := make([][]int, branches)

	for b := 0; b < branches; b++ {
		for i := 0; i < length; i++ {
			m[b] = append(m[b], rand.Intn(length))
		}
	}

	v := make([][]uint64, players)
	b := make([][]uint64, players)

	for p := 0; p < players; p++ {
		v[p] = random(length)
		b[p] = random(branches)
	}

	for p := 0; p < players; p++ {
		wg.Add(1)
		go func(players, me int) {
			defer wg.Done()
			oip, err := NewOIP(
				[][]Connection{
					conns1[me],
					conns2[me],
				},
				me,
				players,
			)
			if err != nil {
				panic(err)
			}

			_, err = oip.OIPMapping(m, b[me], v[me])
			if err != nil {
				panic(err)
			}
		}(players, p)
	}

	wg.Wait()
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
	testOIPn(10, 1<<20, 10)
}
