package main

import (
	"github.com/ldsec/lattigo/v2/bfv"
	"github.com/ldsec/lattigo/v2/rlwe"
)

type Sender struct {
	params    bfv.Parameters
	pk        *rlwe.PublicKey
	encoder   bfv.Encoder
	encryptor bfv.Encryptor
	evaluator bfv.Evaluator
}

func NewSender(params bfv.Parameters, msg MsgSetup) *Sender {
	encoder := bfv.NewEncoder(params)
	encryptor := bfv.NewEncryptor(params, msg.Pk)
	evaluator := bfv.NewEvaluator(params, rlwe.EvaluationKey{Rlk: msg.Rlk})
	return &Sender{
		encoder:   encoder,
		encryptor: encryptor,
		evaluator: evaluator,
	}
}
