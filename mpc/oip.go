package main

import (
	"log"
	"sync"

	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/dbfv"
	"github.com/ldsec/lattigo/v2/drlwe"
	"github.com/ldsec/lattigo/v2/rlwe"
	"github.com/ldsec/lattigo/v2/utils"
)

var LITERAL bfv.ParametersLiteral = bfv.PN12QP109

var CRS []byte = []byte{'C', 'R', 'S'}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func SetupParams() bfv.Parameters {
	params, err := bfv.NewParametersFromLiteral(LITERAL)
	if err != nil {
		panic(err)
	}
	return params
}

type OIP struct {
	me     int // player index
	n      int // number of parties
	log    bool
	params bfv.Parameters //

	// key material
	pk *rlwe.PublicKey // shared public key
	sk *rlwe.SecretKey // secret key share

	// key generation
	ckg *dbfv.CKGProtocol // distributed key generation protocol
	ss  *drlwe.CKGShare   // public key share (aggregated to generate the combined public key)
	crp drlwe.CKGCRP

	//
	encryptor sync.Pool // encryptor
	encoder   sync.Pool // encoder
	evaluator sync.Pool // evaluator
	conns     []*Connection
}

func NewOIP(params bfv.Parameters, me int, conns []*Connection) *OIP {
	// nil connection corresponds to myself
	if conns[me] != nil {
		panic("Self connection must be nil")
	}

	//
	return &OIP{
		params: params,
		me:     me,
		encoder: sync.Pool{
			New: func() interface{} {
				return bfv.NewEncoder(params)
			},
		},
		evaluator: sync.Pool{
			New: func() interface{} {
				return bfv.NewEvaluator(params, rlwe.EvaluationKey{
					Rlk:  nil,
					Rtks: nil,
				})
			},
		},
		n:     len(conns),
		conns: conns,
	}
}

func (o *OIP) P0() *Connection {
	return o.conns[0]
}

func (o *OIP) Pi(i int) *Connection {
	return o.conns[i]
}

func (o *OIP) Log(v ...interface{}) {
	if o.log {
		log.Println("OIP Player", o.me, ":", v)
	}
}

func (o *OIP) broadcast(v interface{}) error {
	if o.me != 0 {
		panic("Only player 0 can broadcast")
	}

	for p := 1; p < o.n; p++ {
		if err := o.Pi(p).Send(v); err != nil {
			return err
		}
	}

	return nil
}

func (o *OIP) IsP0() bool {
	return o.me == 0
}

func (o *OIP) Send0(v interface{}) error {
	if o.me == 0 {
		panic("Player 0 cannot send to self")
	}
	return o.P0().Send(v)
}

func (o *OIP) Recv0(v interface{}) error {
	if o.me == 0 {
		panic("Player 0 cannot receieve from self")
	}
	return o.P0().Recv(v)
}

func (o *OIP) CkgIRound1() error {
	return o.Send0(o.ss)
}

func (o *OIP) Ckg0Round1() error {
	// receieve from each player
	ss := new(drlwe.CKGShare)
	for i := 1; i < o.n; i++ {
		// receive share
		if err := o.Pi(i).Recv(ss); err != nil {
			return err
		}

		// aggregate
		o.ckg.AggregateShares(ss, o.ss, o.ss)
	}

	// extract public key
	pk := bfv.NewPublicKey(o.params)
	o.ckg.GenPublicKey(o.ss, o.crp, pk)
	o.pk = pk

	// broadcast public key to other players
	for i := 1; i < o.n; i++ {
		if err := o.Pi(i).Send(pk); err != nil {
			return err
		}
	}
	return nil
}

func (o *OIP) CkgIRound2() error {
	// receieve aggregated public key from player 0
	o.Log("Receive aggregated public key")
	o.pk = bfv.NewPublicKey(o.params)
	return o.Recv0(o.pk)
}

func (o *OIP) SetupI() error {
	if err := o.CkgIRound1(); err != nil {
		return err
	}

	if err := o.CkgIRound2(); err != nil {
		return err
	}

	return nil
}

func (o *OIP) Setup0() error {
	if err := o.Ckg0Round1(); err != nil {
		return err
	}

	return nil
}

// setup OIP: run distributed key generation
func (o *OIP) Setup() error {
	// expand CRS
	crs, err := utils.NewKeyedPRNG(CRS)
	if err != nil {
		return err
	}

	kgen := bfv.NewKeyGenerator(o.params)

	// prepare distributed key generation
	o.ckg = dbfv.NewCKGProtocol(o.params)
	o.ss = o.ckg.AllocateShares()
	o.sk = kgen.GenSecretKey()
	o.crp = o.ckg.SampleCRP(crs)
	o.ckg.GenShare(o.sk, o.crp, o.ss)

	// run protocol
	if o.IsP0() {
		if err := o.Setup0(); err != nil {
			return nil
		}
	} else {
		if err := o.SetupI(); err != nil {
			return nil
		}
	}

	// create encryptor pool
	o.encryptor =
		sync.Pool{
			New: func() interface{} {
				return bfv.NewEncryptor(o.params, o.pk)
			},
		}

	return nil
}

func (o *OIP) E2S(cts []*bfv.Ciphertext) ([]*rlwe.AdditiveShare, error) {
	o.Log("Run distributed decryption")

	// generate descryption shares

	e2s := dbfv.NewE2SProtocol(o.params, 3.2)

	publicShares := make([]*drlwe.CKSShare, len(cts))
	remoteShares := make([]*drlwe.CKSShare, len(cts))
	secretShares := make([]*rlwe.AdditiveShare, len(cts))
	for i, ct := range cts {
		publicShares[i] = e2s.AllocateShare(ct.Level())
		remoteShares[i] = e2s.AllocateShare(ct.Level())
		secretShares[i] = rlwe.NewAdditiveShare(o.params.Parameters)
		e2s.GenShare(o.sk, ct, secretShares[i], publicShares[i])
	}

	// send / aggregate decryption shares

	if o.IsP0() {
		for p := 1; p < o.n; p++ {
			if err := o.Pi(p).Recv(&remoteShares); err != nil {
				return nil, err
			}
			for i, _ := range cts {
				e2s.AggregateShares(publicShares[i], remoteShares[i], publicShares[i])
			}
		}
	} else {
		// send share to player 0
		if err := o.Send0(publicShares); err != nil {
			return nil, err
		}
	}

	// player 0 generates correction share

	if o.IsP0() {
		for i, ct := range cts {
			e2s.GetShare(secretShares[i], publicShares[i], ct, secretShares[i])
		}
	}

	return secretShares, nil
}

func dup(v uint64, len int) []uint64 {
	arr := make([]uint64, len)
	for i := 0; i < len; i++ {
		arr[i] = v
	}
	return arr
}

func collect_errors(status chan error, num int) error {
	for i := 0; i < num; i++ {
		err := <-status
		if err != nil {
			return err
		}
	}
	return nil
}

func (o *OIP) getEncoder() bfv.Encoder {
	return o.encoder.Get().(bfv.Encoder)
}

func (o *OIP) putEncoder(e bfv.Encoder) {
	o.encoder.Put(e)
}

func (o *OIP) getEvaluator() bfv.Evaluator {
	return o.evaluator.Get().(bfv.Evaluator)
}

func (o *OIP) putEvaluator(e bfv.Evaluator) {
	o.evaluator.Put(e)
}

func (o *OIP) getEncryptor() bfv.Encryptor {
	return o.encryptor.Get().(bfv.Encryptor)
}

func (o *OIP) putEncryptor(e bfv.Encryptor) {
	o.encryptor.Put(e)
}

func (o *OIP) Select(sel []uint64, v [][]uint64) ([]uint64, error) {
	if len(sel) != len(v) {
		log.Panicln("Dimensions does not match")
	}

	branches := len(sel)

	// check if one-time key generation setup required
	if o.pk == nil {
		o.Log("Running setup")
		if err := o.Setup(); err != nil {
			return nil, err
		}
	}

	// maximum length of any vector in v
	max_len := 0
	for i := 0; i < len(v); i++ {
		if len(v[i]) > max_len {
			max_len = len(v[i])
		}
	}

	block_size := 1 << o.params.LogN()
	blocks := (max_len + (block_size - 1)) / block_size
	padded_size := blocks * block_size

	o.Log("Max Len", max_len, "Blocks", blocks, "Padded Size", padded_size)

	// encode and encrypt selector shares

	o.Log("Generate encrypted selector shares")

	cts := make([]*bfv.Ciphertext, branches)

	func() {
		var wg sync.WaitGroup

		for i, s := range sel {
			wg.Add(1)
			go func(i int, s uint64) {
				// encode plaintext
				pt := bfv.NewPlaintext(o.params)
				enco := o.getEncoder()
				enco.EncodeUint(dup(s, block_size), pt)
				o.putEncoder(enco)

				// encrypt
				ct := bfv.NewCiphertext(o.params, 1)
				encr := o.getEncryptor()
				encr.Encrypt(pt, ct)
				o.putEncryptor(encr)

				// store share
				cts[i] = ct
				wg.Done()
			}(i, s)
		}

		wg.Wait()
	}()

	// send selector shares to player 0 and aggregate

	if o.IsP0() {

		o.Log("Aggregate encrypted selector shares")

		locks := make([]sync.Mutex, len(cts))
		status := make(chan error, o.n)

		for p := 1; p < o.n; p++ {
			go func(p int) {
				// allocate ciphertext
				ctp := make([]*bfv.Ciphertext, blocks)
				for i := 0; i < blocks; i++ {
					ctp[i] = bfv.NewCiphertext(o.params, 1)
				}

				// receieve from player
				if err := o.Pi(p).Recv(&ctp); err != nil {
					status <- err
					return
				}

				// add to accumulator
				evl := o.getEvaluator()
				for i := 0; i < branches; i++ {
					locks[i].Lock()
					evl.Add(ctp[i], cts[i], cts[i])
					locks[i].Unlock()
				}
				o.putEvaluator(evl)

				status <- nil
			}(p)
		}

		if err := collect_errors(status, o.n-1); err != nil {
			return nil, err
		}

	} else {

		o.Log("Send selector shares to player 0")

		if err := o.Send0(cts); err != nil {
			return nil, err
		}
	}

	// broadcast / receieve aggreated ciphertexts of selectors

	if o.IsP0() {
		o.Log("Broadcast aggregated encryptions of selectors")
		if err := o.broadcast(cts); err != nil {
			return nil, err
		}
	} else {
		o.Log("Receieve aggregated encryptions of selectors")
		if err := o.Recv0(&cts); err != nil {
			return nil, err
		}
	}

	// generate local share of inner product

	o.Log("Generate share of inner product")
	ct_acc := make([]*bfv.Ciphertext, blocks)

	func() {

		// execute every block in parallel (good for large branches)

		var wg1 sync.WaitGroup

		for b := 0; b < blocks; b++ {

			wg1.Add(1)

			go func(b int) {

				// execute every branch in each block in parallel (good for many branches)

				var wg2 sync.WaitGroup
				var lock sync.Mutex

				acc := bfv.NewCiphertext(o.params, 1)

				for i, vec := range v {
					wg2.Add(1)

					go func(i int, b int, vec []uint64) {
						t := bfv.NewCiphertext(o.params, 1)
						p := bfv.NewPlaintextMul(o.params)
						s := min(len(vec), b*block_size)
						e := min(len(vec), (b+1)*block_size)

						// obtain an encoder/evaluator
						enco := o.getEncoder()
						eval := o.getEvaluator()

						// encode and multiply (slow)
						enco.EncodeUintMul(vec[s:e], p)
						eval.Mul(cts[i], p, t)

						// take lock and add (fast)
						lock.Lock()
						eval.Add(t, acc, acc)
						lock.Unlock()

						// return resources to pool
						o.putEncoder(enco)
						o.putEvaluator(eval)
						wg2.Done()
					}(i, b, vec)
				}

				// save accumulated block

				ct_acc[b] = acc

				wg2.Wait()
				wg1.Done()
			}(b)
		}

		wg1.Wait()
	}()

	// aggregate the shares of the inner product

	if o.IsP0() {
		locks := make([]sync.Mutex, len(ct_acc))
		status := make(chan error, o.n)

		o.Log("Aggregating shares of inner product")

		for p := 1; p < o.n; p++ {

			// handle every player in parallel (good for many players)

			go func(p int, status chan error) {
				ctp := make([]*bfv.Ciphertext, blocks)
				for b := 0; b < blocks; b++ {
					ctp[b] = bfv.NewCiphertext(o.params, 1)
				}

				if err := o.Pi(p).Recv(&ctp); err != nil {
					status <- err
				}

				eval := o.getEvaluator()

				for i := range ct_acc {
					locks[i].Lock()
					eval.Add(ctp[i], ct_acc[i], ct_acc[i])
					locks[i].Unlock()
				}

				o.putEvaluator(eval)

				status <- nil
			}(p, status)

		}

		if err := collect_errors(status, o.n-1); err != nil {
			return nil, err
		}

	} else {

		o.Log("Send shares of inner product")

		if err := o.Send0(ct_acc); err != nil {
			return nil, err
		}
	}

	// broadcast / receieve aggregated ciphertext of inner product

	if o.IsP0() {
		o.Log("Broadcast aggregated encryptions of inner product")
		if err := o.broadcast(ct_acc); err != nil {
			return nil, err
		}
	} else {
		o.Log("Receieve aggregated encryptions of inner product")
		if err := o.Recv0(&ct_acc); err != nil {
			return nil, err
		}
	}

	// run distributed decryption

	shares, err := o.E2S(ct_acc)
	if err != nil {
		return nil, err
	}

	// convert shares back to array

	res := make([]uint64, len(shares)*block_size)

	func() {
		enc := o.getEncoder()
		defer o.putEncoder(enc)

		pt := bfv.NewPlaintextRingT(o.params)

		for i, share := range shares {
			s := i * block_size
			e := (i + 1) * block_size
			pt.Value.Copy(&share.Value)
			enc.DecodeUint(pt, res[s:e])
		}
	}()

	return res[:max_len], nil
}
