package main

import (
	"fmt"

	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

const DIMENSION int = 1 << 12

type Msg1 struct {
	cts []*bfv.Ciphertext
	pk  *rlwe.PublicKey
	rlk *rlwe.RelinearizationKey
}

type Sender struct {
	params    bfv.Parameters
	pk        *rlwe.PublicKey
	cts       []*bfv.Ciphertext
	encoder   bfv.Encoder
	encryptor bfv.Encryptor
	evaluator bfv.Evaluator
}

type Receiver struct {
	params    bfv.Parameters
	pk        *rlwe.PublicKey
	sk        *rlwe.SecretKey
	rlk       *rlwe.RelinearizationKey
	encoder   bfv.Encoder
	encryptor bfv.Encryptor
	decryptor bfv.Decryptor
}

func Setup() bfv.Parameters {
	lit := bfv.PN12QP109
	params, err := bfv.NewParametersFromLiteral(lit)
	if err != nil {
		panic(err)
	}
	return params
}

func NewReceiver(params bfv.Parameters) Receiver {
	keygen := bfv.NewKeyGenerator(params)
	sk, pk := keygen.GenKeyPair()
	rlk := keygen.GenRelinearizationKey(sk, 1)
	encoder := bfv.NewEncoder(params)
	encryptor := bfv.NewEncryptor(params, pk)
	decryptor := bfv.NewDecryptor(params, sk)

	return Receiver{
		params:    params,
		pk:        pk,
		sk:        sk,
		rlk:       rlk,
		encryptor: encryptor,
		decryptor: decryptor,
		encoder:   encoder,
	}
}

func (recv *Receiver) Select(s []uint64) Msg1 {
	pt := bfv.NewPlaintext(recv.params)
	cts := make([]*bfv.Ciphertext, len(s))
	org := make([]uint64, DIMENSION)
	for i := 0; i < len(s); i++ {
		for j := 0; j < DIMENSION; j++ {
			org[j] = s[i]
		}
		cts[i] = bfv.NewCiphertext(recv.params, DIMENSION)
		recv.encoder.EncodeUint(org, pt)
		recv.encryptor.Encrypt(pt, cts[i])
	}

	return Msg1{
		pk:  recv.pk,
		rlk: recv.rlk,
		cts: cts,
	}
}

func NewSender(params bfv.Parameters, msg Msg1) Sender {
	encoder := bfv.NewEncoder(params)
	encryptor := bfv.NewEncryptor(params, msg.pk)
	evaluator := bfv.NewEvaluator(params, rlwe.EvaluationKey{Rlk: msg.rlk})
	return Sender{
		cts:       msg.cts,
		encoder:   encoder,
		encryptor: encryptor,
		evaluator: evaluator,
	}
}

func (send *Sender) Send(w [][]uint64) []*bfv.Ciphertext {
	if len(w) != len(send.cts) {
		panic("dimension does not match")
	}

	blks := (len(w[0]) + DIMENSION - 1) / DIMENSION
	res := make([]*bfv.Ciphertext, blks)
	tmp := bfv.NewCiphertext(send.params, DIMENSION)

	for i := 0; i < blks; i++ {
		s := i * DIMENSION
		e := (i + 1) * DIMENSION
		res[i] = bfv.NewCiphertext(send.params, DIMENSION)
		for j := 0; j < len(w); j++ {
			pt_mul := bfv.NewPlaintextMul(send.params)
			fmt.Println(s, e)
			fmt.Println(w[j][s:e], pt_mul)
			send.encoder.EncodeUintMul(w[j][s:e], pt_mul)
			send.evaluator.Mul(send.cts[i], pt_mul, tmp)
			send.evaluator.Add(res[i], tmp, res[i])
		}
	}

	return res
}

func main() {

	params := Setup()

	recv := NewReceiver(params)

	msg1 := recv.Select([]uint64{0x1, 0x0})

	send := NewSender(params, msg1)

	v1 := make([]uint64, DIMENSION)
	v2 := make([]uint64, DIMENSION)

	send.Send([][]uint64{v1, v2})

	fmt.Println(msg1)

	/*

		lit := bfv.PN12QP109
		params, err := bfv.NewParametersFromLiteral(lit)
		if err != nil {
			panic(err)
		}
		keygen := bfv.NewKeyGenerator(params)
		sk, pk := keygen.GenKeyPair()

		encryptor := bfv.NewEncryptor(params, pk)
		decryptor := bfv.NewDecryptor(params, sk)

		fmt.Println(sk, pk, keygen, encryptor, decryptor)

		s := uint64(1)

		org := make([]uint64, DIMENSION)
		for i := 0; i < DIMENSION; i++ {
			org[i] = s
		}

		pt := bfv.NewPlaintext(params)

		ct := bfv.NewCiphertext(params, len(org))

		encoder := bfv.NewEncoder(params)

		encoder.EncodeUint(org, pt)

		encryptor.Encrypt(pt, ct)

		// multiply
		rlk := keygen.GenRelinearizationKey(sk, 1)
		evaluator := bfv.NewEvaluator(params, rlwe.EvaluationKey{Rlk: rlk})

		send := make([]uint64, DIMENSION)
		for i := 0; i < DIMENSION; i++ {
			send[i] = 0x33
		}

		ct_new := bfv.NewCiphertext(params, DIMENSION)
		pt_mul := bfv.NewPlaintextMul(params)
		encoder.EncodeUintMul(send, pt_mul)
		evaluator.Mul(ct, pt_mul, ct_new)

		// decrypt

		pt_new := bfv.NewPlaintext(params)

		decryptor.Decrypt(ct_new, pt_new)
		fmt.Println(pt, pt_new)

		res := make([]uint64, 1<<12)

		encoder.DecodeUint(pt_new, res)
		fmt.Println(len(res))

		fmt.Println(res)
	*/

}
