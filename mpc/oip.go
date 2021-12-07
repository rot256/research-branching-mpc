package main

import (
	"log"
	"sync"

	"github.com/ldsec/lattigo/v2/bfv"
)

var LITERAL bfv.ParametersLiteral = bfv.PN12QP109
var DIMENSION int = 1 << LITERAL.LogN

func SetupParams() bfv.Parameters {
	params, err := bfv.NewParametersFromLiteral(LITERAL)
	if err != nil {
		panic(err)
	}
	return params
}

type OIP struct {
	me      int // my index
	parties int // number of parties

	// pipes
	conn []ConnectionPair

	// public keys of other parties
	recv []*Receiver
	send []*Sender
}

// used to send public key
func (oip *OIP) ConnSend(him int) *Connection {
	return oip.conn[him].send
}

// used to receieve public key
func (oip *OIP) ConnRecv(him int) *Connection {
	return oip.conn[him].recv
}

func NewOIP(
	conn []ConnectionPair,
	me,
	parties int,
) (*OIP, error) {
	oip := &OIP{}
	oip.me = me
	oip.parties = parties
	oip.conn = conn
	oip.send = make([]*Sender, oip.parties)
	oip.recv = make([]*Receiver, 0)

	// setup receivers
	params := SetupParams()
	recv := NewReceiver(params)
	for p := 0; p < parties; p++ {
		oip.recv = append(oip.recv, recv.Duplicate())
	}

	// send public key to every other party
	func() {
		log.Println("Broadcast own public key")

		setup := MsgSetup{
			Pk:  recv.pk,
			Rlk: recv.rlk,
		}

		// send to each party
		for party := 0; party < oip.parties; party++ {
			if party == me {
				continue
			}

			go func(party int, setup *MsgSetup) {
				c := oip.ConnSend(party)
				if err := c.Encode(setup); err != nil {
					panic(err)
				}
			}(party, &setup)
		}
	}()

	// receive public keys and setup senders
	var wg sync.WaitGroup
	defer wg.Wait()
	func() {
		log.Println("Receieve public keys")

		// receieve public keys from each party
		for party := 0; party < oip.parties; party++ {
			if party == me {
				continue
			}

			wg.Add(1)
			go func(party int) {
				defer wg.Done()
				c := oip.ConnRecv(party)
				msg, err := c.ReadMsgSetup()
				if err != nil {
					panic(err)
				}
				oip.send[party] = NewSender(params, msg)
			}(party)
		}
	}()

	log.Println("OIP, setup complete, player:", me)
	return oip, nil
}

func (oip *OIP) send_oip(
	conn *Connection,
	sender *Sender,
	share_mx *sync.Mutex,
	share []uint64,
	mapping [][]int,
	b []uint64,
	v []uint64,
) {
	size := len(share)
	blocks := (size + DIMENSION - 1) / DIMENSION
	branches := len(mapping)
	pad_size := blocks * DIMENSION

	log.Println("OIP Send,", "size:", size, "len(v):", len(v), "len(b):", b, "pad_size:", pad_size, "len(share):", len(share))

	// receieve message from receiever
	msg1, err := conn.ReadMsgReceiver()
	if err != nil {
		panic(err)
	}

	// subtract mask from own share
	x := random(pad_size)
	share_mx.Lock()
	for i := 0; i < len(share); i++ {
		share[i] = sub(share[i], x[i])
	}
	share_mx.Unlock()

	// create ciphertexts
	var ct_mx sync.Mutex
	msg2 := MsgSender{}
	msg2.Cts = make([]*bfv.Ciphertext, blocks)

	// wait for all parallel jobs to finish
	var wg sync.WaitGroup

	// add random masks (parallel)
	for i := 0; i < blocks; i++ {
		wg.Add(1)
		func(i int) {
			defer wg.Done()

			s := i * DIMENSION
			e := (i + 1) * DIMENSION
			pt_mask := bfv.NewPlaintext(oip.recv[0].params)
			sender.encoder.EncodeUint(x[s:e], pt_mask)

			msg2.Cts[i] = bfv.NewCiphertext(oip.recv[0].params, 1)
			sender.evaluator.Add(msg2.Cts[i], pt_mask, msg2.Cts[i])
		}(i)
	}

	// compute multiplication and additions (parallel)
	for branch := 0; branch < branches; branch++ {
		// apply permutation and masking
		m := mapping[branch]
		t := make([]uint64, pad_size)
		for i := 0; i < size; i++ {
			t[i] = v[m[i]]
		}

		// accumulate in ciphertext
		for i := 0; i < blocks; i++ {
			wg.Add(1)
			func(i int, branch int) {
				defer wg.Done()

				// start and end of block
				s := i * DIMENSION
				e := (i + 1) * DIMENSION

				// multiply choice by message

				ct := bfv.NewCiphertext(oip.recv[0].params, 1)
				pt_mul := bfv.NewPlaintextMul(oip.recv[0].params)
				sender.encoder.EncodeUintMul(t[s:e], pt_mul)
				sender.evaluator.Mul(msg1.Cts[branch], pt_mul, ct)

				// add to  accumulation
				ct_mx.Lock()
				sender.evaluator.Add(msg2.Cts[i], ct, msg2.Cts[i])
				ct_mx.Unlock()
			}(i, branch)
		}
	}

	// wait for ciphertext to be ready
	wg.Wait()

	// ship it!
	if err := conn.Encode(&msg2); err != nil {
		panic(err)
	}
}

func (oip *OIP) recv_oip(
	conn *Connection,
	recv *Receiver,
	share_mx *sync.Mutex,
	share []uint64,
	b []uint64,
) {
	size := len(share)
	blocks := (size + DIMENSION - 1) / DIMENSION
	pad_size := blocks * DIMENSION

	log.Println("OIP Recv, blocks:", blocks, "pad_size:", pad_size, "size:", size, "len(b):", len(b))

	// send first message
	msg1 := recv.NewSelection(b)
	if err := conn.Encode(msg1); err != nil {
		panic(err)
	}

	// receieve message from sender
	msg2, err := conn.ReadMsgSender()
	if err != nil {
		panic(err)
	}

	// decryption result slice
	var res_mx sync.Mutex
	res := make([]uint64, pad_size)

	// decrypt message in parallel
	var wg sync.WaitGroup
	for i := 0; i < blocks; i++ {
		wg.Add(1)
		func(i int) {
			defer wg.Done()

			// start and end of block
			s := i * DIMENSION
			e := (i + 1) * DIMENSION

			// descrypt block
			pt_new := bfv.NewPlaintext(recv.params)
			recv.decryptor.Decrypt(msg2.Cts[i], pt_new)
			res_mx.Lock()
			recv.encoder.DecodeUint(pt_new, res[s:e])
			res_mx.Unlock()
		}(i)
	}

	// wait for work to complete
	wg.Wait()

	// accumulate into share
	share_mx.Lock()
	for i := 0; i < size; i++ {
		share[i] = add(share[i], res[i])
	}
	share_mx.Unlock()
}

func (oip *OIP) OIPMapping(mapping [][]int, b []uint64, v []uint64) ([]uint64, error) {
	// debug
	// b = []uint64{1, 0}
	// v = []uint64{1, 2, 3, 4, 5, 6}
	// fmt.Println("OIPMapping, v:", v, "b:", b)

	if len(mapping) != len(b) {
		log.Fatalln(
			"len(mapping) =", len(mapping),
			"len(mapping[0]) =", len(mapping[0]),
			"len(b) =", len(b),
			"len(v) =", len(v),
		)
	}

	size := len(mapping[0])
	branches := len(mapping)

	var wg sync.WaitGroup
	var share_mx sync.Mutex // lock protecting the share
	share := make([]uint64, size)

	// calculate local cross terms
	wg.Add(1)
	func() {
		share_mx.Lock()
		for branch := 0; branch < branches; branch++ {
			m := mapping[branch]
			for i := 0; i < size; i++ {
				h := mul(b[branch], v[m[i]])
				share[i] = add(share[i], h)
			}
		}
		share_mx.Unlock()
		wg.Done()
	}()

	// act as receiever
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			continue
		}
		wg.Add(1)
		go func(party int) {
			oip.recv_oip(
				oip.ConnRecv(party),
				oip.recv[party],
				&share_mx,
				share,
				b,
			)
			wg.Done()
		}(party)
	}

	// act as sender
	for party := 0; party < oip.parties; party++ {
		if party == oip.me {
			continue
		}
		wg.Add(1)
		go func(party int) {
			oip.send_oip(
				oip.ConnSend(party),
				oip.send[party],
				&share_mx,
				share,
				mapping,
				b,
				v,
			)
			wg.Done()
		}(party)
	}

	// wait for senders/receivers to finish

	wg.Wait()

	return share, nil
}

func (oip *OIP) TryOIPMapping(mapping [][]int, b []uint64, v []uint64) []uint64 {
	res, err := oip.OIPMapping(mapping, b, v)
	if err != nil {
		panic(err)
	}
	return res
}
