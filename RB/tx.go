package main

import (
	ch "./chameleon"
	"errors"
	"bytes"
)

type Tx interface {
	// Payload contains the main data
	// scrpits,certificates,texts,smart contracts...
	// anything you wanna put on chain
	Payload() interface{}

	// Proof proves the validity of this tx
	// could be signature or set of sigs,declaration,proof of knowledge...
	// even empty (normally in permissioned chain,accept tx unconditionally)
	// you can assign some bits to distinguish between initial tx and modified tx
	Proof() interface{}

	// ChameleonPk is used to compute the hash value of the tx
	ChameleonPk() interface{}

	// CheckString is used to verify the hash value
	CheckString() interface{}

	// Hash value of the tx,
	HashVal() interface{}

	// Vertify the validity of the tx
	// include the validity of payload,proof and Hash value
	Verify(ChameleonPara interface{}) bool

	// use new payload and proof to cover the original ones
	// use private key to generate collision
	Modify(Payload interface{},Proof interface{},PrivateKey interface{},ChameleonPara interface{}) error
}



// Example Tx Implementation
type BasicTx struct{
	PayloadB     []byte    `json:"payload"`
	ProofB       string    `json:"proof"`
	ChameleonPkB []byte    `json:"chameleon_public_key"`
	CheckStringB [][]byte  `json:"check_string"`
	HashValB     []byte    `json:"hash"`
}

func NewBasicTx(payload []byte,proof string,pk []byte,para [][]byte) (*BasicTx,error ){
	if len(para) != 3 {
		return nil,errors.New("Invalid parameters,check your input!")
	}

	p := para[0]
	q := para[1]
	g := para[2]
	r := ch.Randgen(&q)
	s := ch.Randgen(&q)
	var hashout []byte
	ch.ChameleonHash(&pk, &p, &q, &g, &payload, &r, &s, &hashout)
	t := &BasicTx{
		PayloadB:       payload,
		ProofB:         proof,
		ChameleonPkB:   pk ,
		CheckStringB:   [][]byte{r,s},
		HashValB:       hashout,
	}
	if !t.CheckProof() {
		return nil,errors.New("Error:invalid proof of transaction!")
	}

	return t,nil
}

func (t *BasicTx) CheckProof() bool {
	return true
}

func (t *BasicTx) Payload() interface{} {
	return t.PayloadB
}

func (t *BasicTx) Proof() interface{} {
	return t.ProofB
}

func (t *BasicTx) ChameleonPk() interface{} {
	return t.ChameleonPkB
}

func (t *BasicTx) CheckString() interface{} {
	return t.CheckStringB
}

func (t *BasicTx) HashVal() interface{} {
	return t.HashValB
}

func (t *BasicTx) Verify(pa interface{}) bool {
	para,ok := pa.([][]byte)
	if ok && len(para) == 3 {
	} else {
		return false
	}

	p := para[0]
	q := para[1]
	g := para[2]
	r := t.CheckStringB[0]
	s := t.CheckStringB[1]
	pk := t.ChameleonPkB
	payload := t.PayloadB
	var hashout []byte
	ch.ChameleonHash(&pk, &p, &q, &g, &payload, &r, &s, &hashout)

	if bytes.Equal(hashout,t.HashValB) {
		return t.CheckProof()
	}
	return false
}

func (t *BasicTx) Modify(payld_new interface{},prf_new interface{},private interface{},pa interface{}) error {
	para,ok1 := pa.([][]byte)
	new,ok2 := payld_new.([]byte)
	proof_new,ok3 := prf_new.(string)
	sk,ok4 := private.([]byte)
	if ok1 && ok2 && ok3 && ok4 && len(para) == 3 {
	} else {
		return errors.New("Invalid parameters,check your input!")
	}

	p := para[0]
	q := para[1]
	g := para[2]
	r1 := t.CheckStringB[0]
	s1 := t.CheckStringB[1]
	pk := t.ChameleonPkB
	old := t.PayloadB
	var r2,s2 []byte
	ch.GenerateCollision(&pk, &sk, &p, &q, &g, &old, &new, &r1, &s1, &r2, &s2)
	t_new := &BasicTx{
		PayloadB:       new,
		ProofB:         proof_new,
		ChameleonPkB:   pk,
		CheckStringB:   [][]byte{r2,s2},
		HashValB:       t.HashValB,
	}
	if !t_new.Verify(para) {
		return errors.New("Error:something in the new transaction is invalid!Collsion generating failed.")
	}
	t = t_new
	return nil
}

