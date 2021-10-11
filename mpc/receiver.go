package main

import (
	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

type Receiver struct {
	params    bfv.Parameters
	pk        *rlwe.PublicKey
	sk        *rlwe.SecretKey
	rlk       *rlwe.RelinearizationKey
	encoder   bfv.Encoder
	encryptor bfv.Encryptor
	decryptor bfv.Decryptor
}

func NewReceiver(params bfv.Parameters) *Receiver {
	keygen := bfv.NewKeyGenerator(params)
	sk, pk := keygen.GenKeyPair()
	rlk := keygen.GenRelinearizationKey(sk, 1)
	encoder := bfv.NewEncoder(params)
	encryptor := bfv.NewEncryptor(params, pk)
	decryptor := bfv.NewDecryptor(params, sk)

	return &Receiver{
		params:    params,
		pk:        pk,
		sk:        sk,
		rlk:       rlk,
		encryptor: encryptor,
		decryptor: decryptor,
		encoder:   encoder,
	}
}

func (old *Receiver) Duplicate() *Receiver {
	encoder := bfv.NewEncoder(old.params)
	encryptor := bfv.NewEncryptor(old.params, old.pk)
	decryptor := bfv.NewDecryptor(old.params, old.sk)
	return &Receiver{
		params:    old.params,
		pk:        old.pk,
		sk:        old.sk,
		rlk:       old.rlk,
		encryptor: encryptor,
		decryptor: decryptor,
		encoder:   encoder,
	}
}

func (recv *Receiver) NewSelection(s []uint64) *MsgReceiver {
	cts := make([]*bfv.Ciphertext, len(s))
	org := make([]uint64, DIMENSION)

	for i := 0; i < len(s); i++ {
		// duplicate value
		for j := 0; j < DIMENSION; j++ {
			org[j] = s[i]
		}
		pt := bfv.NewPlaintext(recv.params)
		ct := bfv.NewCiphertext(recv.params, 1)
		recv.encoder.EncodeUint(org, pt)
		recv.encryptor.Encrypt(pt, ct)
		cts[i] = ct
	}

	return &MsgReceiver{
		Cts: cts,
	}
}
