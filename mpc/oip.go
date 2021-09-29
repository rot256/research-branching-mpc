package main

import (
	"encoding/gob"
	"fmt"

	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/ring"
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

func init() {
	gob.Register(bfv.Ciphertext{})
	gob.Register(rlwe.PublicKey{})
	gob.Register(rlwe.RelinearizationKey{})
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

func broadcast(me int, msgs []interface{}, conn []Connection) chan error {

	if len(msgs) != len(conn) {
		panic("len(msgs) != len(conn)")
	}

	sig_send := make(chan error)
	go func() {
		for party := 0; party < len(conn); party++ {
			if party == me {
				continue
			}
			if err := conn[party].Send(msgs[party]); err != nil {
				sig_send <- err
				return
			}
		}
		sig_send <- nil
	}()

	return sig_send
}

/*
func (c *Connection) SendMsgReceiver(msg *MsgReceiver) error {
	value := msg.Cts.Ciphertext.Value
	for i := 0; i < len(value); i++ {
		if err := c.Send(value[i]); err != nil {
			return err
		}
	}
}

func (c *Connection) RecvMsgReceiver() (*MsgReceiver, error) {
	msg := MsgReceiver{}
	for i := 0; i < DIMENSION+1; i++ {
		poly := ring.Poly{}
		if err := c.Recv(&poly); err != nil {
			return nil, err
		}
		msg.Ciphertext.Value = append(msg.Ciphertext.Value, poly)
	}
	return msg, nil
}
*/

func send_ct(c *Connection, ct *bfv.Ciphertext) error {
	v := ct.Ciphertext.Value
	if err := c.Send(len(v)); err != nil {
		return err
	}
	for i := 0; i < len(v); i++ {
		if err := c.Send(v[i]); err != nil {
			return err
		}
	}
	return nil
}

func recv_ct(c *Connection) (*bfv.Ciphertext, error) {
	var size int
	if err := c.Recv(&size); err != nil {
		return nil, err
	}
	ct := bfv.Ciphertext{}
	ct.Ciphertext.Value = make([]*ring.Poly, size)
	v := ct.Ciphertext.Value

	for i := 0; i < len(v); i++ {
		if err := c.Recv(&v[i]); err != nil {
			return nil, err
		}
	}

	return &ct, nil
}

/*
func fix_ct(fill *ring.Poly, ct *bfv.Ciphertext) {
	pad := (DIMENSION + 1) - len(ct.Ciphertext.Value)
	for i := 0; i < len(ct.Ciphertext.Value); i++ {
		fmt.Println("new-entry:", i, ct.Ciphertext.Value[i])
	}
	for i := 0; i < pad; i++ {
		ct.Ciphertext.Value = append(ct.Ciphertext.Value, fill.CopyNew())
	}
}
*/

func (oip *OIP) OIPMapping(mapping [][]int, b []uint64, v []uint64) ([]uint64, error) {
	// debug
	// b = []uint64{1, 0}
	// v = []uint64{1, 2, 3, 4, 5, 6}
	fmt.Println("OIPMapping, v:", v, "b:", b)

	if len(mapping) != len(b) || len(mapping[0]) != len(v) {
		fmt.Println(mapping, b, v)
		panic("invalid dimension")
	}

	size := len(mapping[0])
	blocks := (size + DIMENSION - 1) / DIMENSION
	pad_size := blocks * DIMENSION
	branches := len(mapping)

	// send first message
	msg_recv := oip.recv.NewSelection(b)
	sig_send := broadcast(
		oip.me,
		func() []interface{} {
			msgs := make([]interface{}, oip.parties)
			for party := 0; party < oip.parties; party++ {
				msgs[party] = msg_recv
			}
			return msgs
		}(),
		oip.conn,
	)

	// calculate local cross terms
	share := make([]uint64, size)
	fmt.Println("Cross terms")
	for branch := 0; branch < branches; branch++ {
		m := mapping[branch]
		for i := 0; i < size; i++ {
			h := mul(b[branch], v[m[i]])
			share[i] = add(share[i], h)
		}
	}

	// act as sender
	fmt.Println("Act as sender")
	msgs2 := make([]MsgSender, 0, oip.parties)
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			msgs2 = append(msgs2, MsgSender{})
			continue
		}

		// receive message from receiver
		var msg1 MsgReceiver
		if err := oip.conn[party].Recv(&msg1); err != nil {
			return nil, err
		}

		//
		var msg2 MsgSender
		// x := random(pad_size)
		x := make([]uint64, pad_size)
		for i := 0; i < len(share); i++ {
			share[i] = sub(share[i], x[i])
		}

		fmt.Println("Party:", party, "x:", x[:size])

		// create compressed ciphertext
		msg2.Cts = make([]*bfv.Ciphertext, blocks, blocks)

		// add random mask
		for i := 0; i < blocks; i++ {
			s := i * DIMENSION
			e := (i + 1) * DIMENSION
			pt_mask := bfv.NewPlaintext(oip.recv.params)
			msg2.Cts[i] = bfv.NewCiphertext(oip.recv.params, DIMENSION)
			oip.send[party].encoder.EncodeUint(x[s:e], pt_mask)
			oip.send[party].evaluator.Add(msg2.Cts[i], pt_mask, msg2.Cts[i])
		}

		//
		for branch := 0; branch < branches; branch++ {
			// apply permutation and masking
			m := mapping[branch]
			t := make([]uint64, pad_size)
			for i := 0; i < size; i++ {
				t[i] = v[m[i]]
			}

			fmt.Println("Branch:", branch, "t:", t[:size])

			// accumulate in ciphertext
			for i := 0; i < blocks; i++ {
				// start and end of block
				s := i * DIMENSION
				e := (i + 1) * DIMENSION

				// multiply choice by message
				ct := bfv.NewCiphertext(oip.recv.params, DIMENSION)
				pt_mul := bfv.NewPlaintextMul(oip.recv.params)
				oip.send[party].encoder.EncodeUintMul(t[s:e], pt_mul)

				// add to  accumulation
				oip.send[party].evaluator.Mul(msg1.Cts[branch], pt_mul, ct)
				oip.send[party].evaluator.Add(msg2.Cts[i], ct, msg2.Cts[i])
			}
		}

		msgs2 = append(msgs2, msg2)
	}

	// send response to receiver
	if err := <-sig_send; err != nil {
		return nil, err
	}
	sig_send = broadcast(
		oip.me,
		func() []interface{} {
			msgs := make([]interface{}, oip.parties)
			for party := 0; party < oip.parties; party++ {
				msgs[party] = msgs2[party]
			}
			return msgs
		}(),
		oip.conn,
	)

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

		// accumulate into share
		fmt.Println("Preshare:", share)
		fmt.Println("Res:", res[:size])
		for i := 0; i < size; i++ {
			share[i] = add(share[i], res[i])
		}
	}

	if err := <-sig_send; err != nil {
		return nil, err
	}
	fmt.Println("Shares:", share)
	return share, nil
}

func (oip *OIP) TryOIPMapping(mapping [][]int, b []uint64, v []uint64) []uint64 {
	res, err := oip.OIPMapping(mapping, b, v)
	if err != nil {
		panic(err)
	}
	return res
}
