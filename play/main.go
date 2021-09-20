package main

import (
	"fmt"
	"math/big"

	"github.com/ldsec/lattigo/v2/ring"
	"github.com/ldsec/lattigo/v2/utils"
)

const BIT_SIZE_P = 48
const BIT_SIZE_Q = 60
const LOG_N = 12
const N = 1 << LOG_N
const NUM_PRIMES = 2

const PVW_N0 = 10
const PVW_N1 = 20
const PVW_K = 5
const PVW_M = 5

// p, q with p < q
//

// page 19
const GSW_N0 = 2 // rows of B
const GSW_N1 = 3 // rows of P = [ -A \\ B ]
const GSW_N2 = 3 // ?
const GSW_K = 1  // rows of A

const GSW_M2 = 6

func encode(ringQ *ring.Ring, v []uint64) *ring.Poly {
	poly := ringQ.NewPoly()

	// check
	if len(v) > len(poly.Coeffs) {
		panic("invalid size during conversion")
	}
	if len(ringQ.Modulus) != 2 || ringQ.Modulus[0] >= ringQ.Modulus[1] {
		panic("invalid ring")
	}

	// each coefficient is v[i] * q
	p := ringQ.Modulus[0]
	q := ringQ.Modulus[1]
	q_mp := big.NewInt(0)
	q_mp.SetUint64(q % p)
	p_b := big.NewInt(0)
	p_b.SetUint64(p)

	for i := 0; i < len(v); i++ {
		// w = (v[i] * (q mod p)) mod p
		// w = v[i] * q mod p
		w := big.NewInt(0)
		w.SetUint64(v[i])
		w.Mul(w, q_mp)
		w.Mod(w, p_b)
		poly.Coeffs[i][0] = w.Uint64()
		poly.Coeffs[i][1] = 0
	}
	return poly
}

func decode(ringQ *ring.Ring, poly *ring.Poly) []uint64 {
	v := make([]uint64, len(poly.Coeffs))

	if poly.IsNTT {
		panic("providing an NTT polynomial")

	}

	p := ringQ.Modulus[0]
	q := ringQ.Modulus[1]

	for i := 0; i < len(poly.Coeffs); i++ {
		v[i] =
	}

	return v
}

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
	}()

	// test encryption

	// encode plaintext
	if len(ringQ.Modulus) != 2 {
		panic("not supported")
	}

	q := ringQ.Modulus[0]
	q_p := ringQ.Modulus[1]

	v := 0

	// compute v * q' \mod q
	w := func() uint64 {
		q_int := big.NewInt(0)
		q_int.SetUint64(q)

		w := big.NewInt(0)
		w.SetUint64(q_p)
		w.Mul(w, big.NewInt(int64(v)))
		w.Mod(w, q_int)
		return w.Uint64()
	}()

	C := make([]*ring.Poly, GSW_N1)
	for i := 0; i < GSW_N1; i++ {
		C[i] = ringQ.NewPoly()
		C[i].Coeffs[0][0] = w
	}

	// encrypt zero
	X := gaussian.ReadNew()
	ringQ.NTT(X, X)
	for i := 0; i < GSW_N1; i++ {
		ringQ.NTT(C[i], C[i])
		ringQ.MulCoeffsAndAdd(P[i], X, C[i])
	}

	// decrypt
	pt := make([]*ring.Poly, GSW_N0)
	for i := 0; i < GSW_N0; i++ {
		pt[i] = C[i+1].CopyNew()
		ringQ.MulCoeffsAndAdd(
			S[i],
			C[0],
			pt[i],
		)
		ringQ.InvNTT(pt[i], pt[i])
	}

	// x := ringQ.NewPoly()
	// ringQ.DivRoundByLastModulusNTT(pt[0], x)
	// ringQ.InvNTTLvl(1, x, x)

	r := make([]*big.Int, len(pt[0].Coeffs[0]))
	ringQ.PolyToBigintLvl(1, pt[0], r)

	x := big.NewInt(0)
	x.SetUint64(q_p)

	d := big.NewInt(0)

	d.Mod(r[0], x)

	fmt.Println(r[0], d)

	/*
		for i := 0; i < len(r); i++ {

			x := big.NewInt(0)
			x.SetUint64(q_p)

			d := big.NewInt(0)

			d.Mod(r[i], x)

			fmt.Println(r[i], d)
		}
	*/

	/*

		r := uint64(4000) // todo

		for i := 0; i < len(pt[0].Coeffs); i++ {
			c := pt[0].Coeffs[i]
			m := ringQ.Modulus[i]
			l := m / 2
			for j := 0; j < len(c); j++ {

				// map to absolute value
				var v uint64
				if c[j] > l {
					v = m - c[j]
				} else {
					v = c[j]
				}
				fmt.Println("line", i, v, v/r)
			}
		}
	*/
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

	q := ring.GenerateNTTPrimes(
		BIT_SIZE_Q,
		2*N,
		1,
	)

	p := ring.GenerateNTTPrimes(
		BIT_SIZE_P,
		2*N,
		1,
	)

	primes := []uint64{p[0], q[0]}

	fmt.Println("primes: ", primes)

	ringQ, err := ring.NewRing(N, primes)
	if err != nil {
		panic(err)
	}

	pvw_high(ringQ)

	//

	gsw(ringQ)

}
