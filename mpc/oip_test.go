package main

import (
	"fmt"
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

func reconstruct_branches(shares [][][]uint64) [][]uint64 {
	players := len(shares)
	branches := len(shares[0])

	output := make([][]uint64, branches)

	for b := 0; b < branches; b++ {
		s := make([][]uint64, players)
		for p := range shares {
			s[p] = shares[p][b]
		}
		output[b] = reconstruct(s)
	}

	return output
}

func testOIPn(branches, length, players, repetitions int) {

	fmt.Println("Branches", branches, "Length", length, "Players", players)

	s := make([][]uint64, players)
	v := make([][][]uint64, players)

	for p := 0; p < players; p++ {
		s[p] = random(branches)
	}

	for p := 0; p < players; p++ {
		v[p] = make([][]uint64, branches)
		for b := 0; b < branches; b++ {
			v[p][b] = random(length)
		}
	}

	// reconstruct

	sel := reconstruct(s)
	bra := reconstruct_branches(v)

	correct := make([]uint64, length)

	for b := 0; b < branches; b++ {

		for i := 0; i < length; i++ {
			correct[i] = add(correct[i], mul(sel[b], bra[b][i]))
		}
	}

	params := SetupParams()

	var oips []*OIP
	for p, c := range StarDummies(players) {
		oips = append(oips, NewOIP(params, p, c))
	}

	res_shares := make([][]uint64, players)

	// repetions

	fmt.Println("Running...")

	for r := 0; r < repetitions; r++ {

		var wg sync.WaitGroup

		for p, oip := range oips {
			wg.Add(1)

			go func(p int, oip *OIP) {
				res, err := oip.Select(
					s[p],
					v[p],
				)

				res_shares[p] = res

				if err != nil {
					panic(err)
				}

				wg.Done()
			}(p, oip)
		}

		wg.Wait()
	}

	//

	fmt.Println("Check output")

	result := reconstruct(res_shares)

	if len(result) != len(correct) {
		panic("Wrong size")
	}

	for i := range result {
		if result[i] != correct[i] {
			panic("Does not match")
		}
	}

}

func TestOIP(t *testing.T) {

	for p := 1; p < 10; p++ {
		branches := rand.Intn(100)
		length := rand.Intn(1 << 16)

		testOIPn(branches, length, p, 1)
	}

	for r := 1; r < 10; r++ {
		branches := rand.Intn(1 << 8)
		length := rand.Intn(1 << 16)
		players := rand.Intn(100) + 1
		testOIPn(branches, length, players, 1)
	}

}

func BenchmarkOIP_P2_B2_L20(b *testing.B) {
	testOIPn(2, 1<<20, 2, b.N)
}

func BenchmarkOIP_P2_B32_L20(b *testing.B) {
	testOIPn(2, 1<<20, 32, b.N)
}