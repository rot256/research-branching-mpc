package main

import (
	"fmt"

	"github.com/ldsec/lattigo/v2/ring"
	"github.com/ldsec/lattigo/v2/utils"
)

const BIT_SIZE = 60
const LOG_N = 12
const N = 1 << LOG_N
const NUM_PRIMES = 2

const PVW_N0 = 10
const PVW_N1 = 20
const PVW_K = 5
const PVW_M = 5

// page 19
const GSW_N0 = 2 // rows of B
const GSW_N1 = 3 // rows of P = [ -A \\ B ]
const GSW_N2 = 3 // ?
const GSW_K = 1  // rows of A

const GSW_M2 = 6

func gsw(ringQ *ring.Ring) {
	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}

	gaussian := ring.NewGaussianSampler(prng, ringQ, 3.2, 19)
	uniform := ring.NewUniformSampler(prng, ringQ)

	S := make([]*ring.Poly, GSW_N0)
	E := make([]*ring.Poly, GSW_N0)
	P := make([]*ring.Poly, GSW_N1)
	B := P[1:]

	// A <- Uniform()
	P[0] = uniform.ReadNew()
	P[0].IsNTT = true
	A := P[0]

	// B = A x S + E
	for i := 0; i < GSW_N0; i++ {
		// S <- Gaussian()
		S[i] = gaussian.ReadNew()
		ringQ.NTT(S[i], S[i])

		// E <- Gaussian()
		E[i] = gaussian.ReadNew()
		ringQ.NTT(E[i], E[i])

		// B = S x A + E
		B[i] = E[i].CopyNew()
		ringQ.MulCoeffsAndAdd(A, S[i], B[i])
	}

	// P = [ - A \\ B]
	ringQ.Neg(A, A)

	// sanity check
	func() {
		if len(P) != GSW_N1 {
			panic("P has wrong dimension")
		}

		e := make([]*ring.Poly, GSW_N0)
		e[0] = P[1].CopyNew()
		e[1] = P[2].CopyNew()

		ringQ.MulCoeffsAndAdd(S[0], P[0], e[0])
		ringQ.MulCoeffsAndAdd(S[1], P[0], e[1])

		for i := 0; i < GSW_N0; i++ {
			if !E[i].Equals(e[i]) {
				panic("not approximate eigenvector")
			}
		}

		fmt.Println(e[0])
	}()

	/*
		// sanity check

		r := make([]*ring.Poly, len(B))
		t_i := ringQ.NewPoly()
		ringQ.Neg(S, t_i)

		for i := 0; i < len(r); i++ {
			fmt.Println(b[i])
			r[i] = b[i].CopyNew()
			ringQ.MulCoeffsAndAdd(B[i], t_i, r[i])
			if !r[i].Equals(e[i]) {
				panic("equals noise")
			}
		}
	*/

	// test encryption

	// encode plaintext

	/*
		v := uint64(1)
		pt := ringQ.NewPoly()
		fmt.Println("len", len(pt.Coeffs[0]))
		fmt.Println("len", len(pt.Coeffs))
		pt.Coeffs[1][0] = v
	*/

	// encrypt zero
	C := make([]*ring.Poly, GSW_N1)
	X := make([]*ring.Poly, GSW_N1)
	for i := 0; i < GSW_N1; i++ {
		C[i] = ringQ.NewPoly()
		X[i] = gaussian.ReadNew() // X <- Gaussian()
		ringQ.MulCoeffs(P[i], X[i], C[i])
	}

	//

	pt := make([]*ring.Poly, GSW_N0)
	for i := 0; i < GSW_N0; i++ {
		pt[i] = ringQ.NewPoly()
		ringQ.MulCoeffsAndAdd(
			S[i],
			C[0],
			pt[i],
		)
		ringQ.Add(
			C[i+1],
			pt[i],
			pt[i],
		)
	}

	fmt.Println(pt[0])

	// gadget
	// pt.Coeffs[0][0] *= ringQ.ModulusBigint

	// R := make([]*ring.Poly, GSW_N1)
	// P := [ - A \\ B ]

	fmt.Println(1)

	// pt * G + P * X

}

func pvw_high(ringQ *ring.Ring) {
	// setup Gaussian sampler

	prng, err := utils.NewPRNG()
	if err != nil {
		panic(err)
	}

	// gaussianSamplerQ := ring.NewGaussianSampler(prng, ringQ, 3.2, 19)
	uniformSamplerQ := ring.NewUniformSampler(prng, ringQ)

	var Sp [PVW_N0][PVW_K]*ring.Poly
	var A [PVW_K][PVW_M]*ring.Poly

	for r := 0; r < len(Sp); r++ {
		for c := 0; c < len(Sp[0]); c++ {
			Sp[r][c] = uniformSamplerQ.ReadNew()
		}
	}

	for r := 0; r < len(A); r++ {
		for c := 0; c < len(A[0]); c++ {
			A[r][c] = uniformSamplerQ.ReadNew()
		}
	}

}

func main() {

	// setup ring

	primes := ring.GenerateNTTPrimes(
		BIT_SIZE,
		2*N,
		NUM_PRIMES,
	)

	ringQ, err := ring.NewRing(N, primes)
	if err != nil {
		panic(err)
	}

	pvw_high(ringQ)

	//

	gsw(ringQ)

}
