package main

import (
	"fmt"

	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

const DIMENSION int = 1 << 12

type MsgSetup struct {
	Pk  *rlwe.PublicKey
	Rlk *rlwe.RelinearizationKey
}

type MsgReceiver struct {
	Cts []*bfv.Ciphertext
}

type MsgSender struct {
	Cts []*bfv.Ciphertext
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
	conn []Connection

	// public keys of other parties
	recv *Receiver
	send []*Sender
}

func NewIOP(
	conn []Connection,
	me,
	parties int,
) (*OIP, error) {
	oip := &OIP{}
	oip.me = me
	oip.parties = parties

	// setup receivers
	params := SetupParams()
	oip.recv = NewReceiver(params)
	oip.conn = conn

	// send public key to every other party
	for party := 0; party < parties; party++ {
		if party == me {
			continue
		}
		err := oip.conn[party].Send(
			MsgSetup{
				Pk:  oip.recv.pk,
				Rlk: oip.recv.rlk,
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
		if err := oip.conn[party].Recv(&setup); err != nil {
			return nil, err
		}
		oip.send = append(oip.send, NewSender(params, setup))
	}

	return oip, nil
}

func (oip *OIP) OIPMapping(mapping [][]int, b []uint64, v []uint64) ([]uint64, error) {
	fmt.Println("OIPMapping")

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
		if err := oip.conn[party].Send(msg_recv); err != nil {
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
		if err := oip.conn[party].Recv(&msg1); err != nil {
			return nil, err
		}

		//
		var msg2 MsgSender
		x := random(size)
		for i := 0; i < len(x); i++ {
			share[i] = (share[i] + x[i]) % PRIME
		}

		// create compressed ciphertext
		msg2.Cts = make([]*bfv.Ciphertext, blocks, blocks)
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
				oip.send[party].evaluator.Mul(msg1.Cts[branch], pt_mul, ct)

				// add to  accumulation
				if msg2.Cts[i] == nil {
					msg2.Cts[i] = ct
				} else {
					oip.send[party].evaluator.Add(msg2.Cts[i], ct, msg2.Cts[i])
				}
			}
		}

		// send response to receiver
		if err := oip.conn[party].Send(msg2); err != nil {
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
		if err := oip.conn[party].Recv(&msg_send); err != nil {
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
			oip.recv.decryptor.Decrypt(msg_send.Cts[i], pt_new)
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

func (oip *OIP) TryOIPMapping(mapping [][]int, b []uint64, v []uint64) []uint64 {
	res, err := oip.OIPMapping(mapping, b, v)
	if err != nil {
		panic(err)
	}
	return res
}
