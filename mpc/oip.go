package main

import (
	"encoding/gob"
	"io"

	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

const DIMENSION int = 1 << 12

type MsgSetup struct {
	pk  *rlwe.PublicKey
	rlk *rlwe.RelinearizationKey
}

type MsgReceiver struct {
	cts []*bfv.Ciphertext
}

type MsgSender struct {
	cts []*bfv.Ciphertext
}

func SetupParams() bfv.Parameters {
	lit := bfv.PN12QP109
	params, err := bfv.NewParametersFromLiteral(lit)
	if err != nil {
		panic(err)
	}
	return params
}

type OIP struct {
	me      int // my index
	parties int // number of parties

	// pipes
	dec []*gob.Decoder
	enc []*gob.Encoder

	// public keys of other parties
	recv *Receiver
	send []*Sender
}

func NewIOP(
	r []io.Reader,
	w []io.Writer,
	me,
	parties int,
) (*OIP, error) {
	oip := &OIP{}
	oip.me = me
	oip.parties = parties

	// prepare encoders / decoders
	oip.dec = make([]*gob.Decoder, len(r))
	oip.enc = make([]*gob.Encoder, len(w))
	for party := 0; party < parties; party++ {
		if party == me {
			oip.dec = append(oip.dec, nil)
			oip.enc = append(oip.enc, nil)
		} else {
			oip.dec = append(oip.dec, gob.NewDecoder(r[party]))
			oip.enc = append(oip.enc, gob.NewEncoder(w[party]))
		}
	}

	// setup receivers
	params := SetupParams()
	oip.recv = NewReceiver(params)

	// send public key to every other party
	for party := 0; party < parties; party++ {
		if party == me {
			continue
		}
		err := oip.enc[party].Encode(
			MsgSetup{
				pk:  oip.recv.pk,
				rlk: oip.recv.rlk,
			},
		)
		if err != nil {
			return nil, err
		}
	}

	// receive public keys and setup senders
	for party := 0; party < parties; party++ {
		if party == me {
			oip.send = append(oip.send, nil)
			continue
		}
		var setup MsgSetup
		if err := oip.dec[party].Decode(&setup); err != nil {
			return nil, err
		}
		oip.send = append(oip.send, NewSender(params, setup))
	}

	return oip, nil
}

func (oip *OIP) IOPMapping(mapping [][]int, b []uint64, v []uint64) ([]uint64, error) {
	if len(mapping) != len(b) || len(mapping) != len(v) {
		panic("invalid dimension")
	}

	size := len(mapping[0])
	blocks := (size + DIMENSION - 1) / DIMENSION
	pad_size := blocks * DIMENSION
	branches := len(mapping)

	// send first message
	msg_recv := oip.recv.NewSelection(b)
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			continue
		}
		if err := oip.enc[party].Encode(msg_recv); err != nil {
			return nil, err
		}
	}

	// act as sender
	share := make([]uint64, size, size)
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			continue
		}

		// receive message from receiver
		var msg1 MsgReceiver
		if err := oip.dec[party].Decode(&msg1); err != nil {
			return nil, err
		}

		//
		var msg2 MsgSender
		x := random(size)
		for i := 0; i < len(x); i++ {
			share[i] = (share[i] + x[i]) % PRIME
		}

		// create compressed ciphertext
		msg2.cts = make([]*bfv.Ciphertext, blocks, blocks)
		for branch := 0; branch < branches; branch++ {
			// apply permutation and masking
			m := mapping[branch]
			t := make([]uint64, pad_size, pad_size)
			for i := 0; i < size; i++ {
				t[i] = (x[i] + v[m[i]]) % PRIME
			}

			// accumulate in ciphertext
			for i := 0; i < blocks; i++ {
				// start and end of block
				s := i * DIMENSION
				e := (i + 1) * DIMENSION

				// multiply choice by message
				ct := bfv.NewCiphertext(oip.recv.params, DIMENSION)
				pt_mul := bfv.NewPlaintextMul(oip.recv.params)
				oip.send[party].encoder.EncodeUintMul(t[s:e], pt_mul)
				oip.send[party].evaluator.Mul(msg1.cts[branch], pt_mul, ct)

				// add to  accumulation
				if msg2.cts[i] == nil {
					msg2.cts[i] = ct
				} else {
					oip.send[party].evaluator.Add(msg2.cts[i], ct, msg2.cts[i])
				}
			}
		}

		// send response to receiver
		if err := oip.enc[party].Encode(msg2); err != nil {
			return nil, err
		}
	}

	// act as receiver
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			continue
		}

		// receieve message from sender
		var msg_send MsgSender
		if err := oip.dec[party].Decode(&msg_send); err != nil {
			return nil, err
		}

		// decrypt message
		res := make([]uint64, pad_size)
		for i := 0; i < blocks; i++ {
			// start and end of block
			s := i * DIMENSION
			e := (i + 1) * DIMENSION

			// descrypt block
			pt_new := bfv.NewPlaintext(oip.recv.params)
			oip.recv.decryptor.Decrypt(msg_send.cts[i], pt_new)
			oip.recv.encoder.DecodeUint(pt_new, res[s:e])
		}
		res = res[:size]

		// accumulate into share
		for i := 0; i < size; i++ {
			share[i] = (share[i] + res[i]) % PRIME
		}
	}

	return share, nil
}

func (oip *OIP) TryIOPMapping(mapping [][]int, b []uint64, v []uint64) []uint64 {
	res, err := oip.IOPMapping(mapping, b, v)
	if err != nil {
		panic(err)
	}
	return res
}
