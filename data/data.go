package data

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	ch "github.com/RedactableBlockChain/chameleon"
	"github.com/RedactableBlockChain/path"
	"os"
	"strconv"
)

type Block interface {
	Head() interface{}
	Transactions(index int) Tx
	TransactionCount() int
	Verify() bool
}

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
	HashVal() []byte

	// Vertify the validity of the tx
	// include the validity of payload,proof and Hash value
	Verify(ChameleonPara interface{}) bool

	// use new payload and proof to cover the original ones
	// use private key to generate collision
	Modify(Payload interface{}, Proof interface{}, PrivateKey interface{}, ChameleonPara interface{}) error
}

// Write to file system
func Write(t interface{}, path string) error {
	fw, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		return err
	}
	defer fw.Close()

	encoder := json.NewEncoder(fw)
	err = encoder.Encode(t)
	if err != nil {
		return err
	}

	return nil
}

// Load from file system
func Load(t interface{}, path string) error {
	fr, err := os.Open(path)
	if err != nil {
		return err
	}
	defer fr.Close()

	decoder := json.NewDecoder(fr)
	err = decoder.Decode(&t)
	if err != nil {
		return err
	}
	return nil
}

// Example Golbal Parameter
type GolbalParameter struct {
	CurHeight int    `json:"cur_height"`
	Bits      int    `json:"bit"`
	P         []byte `json:"p"`
	Q         []byte `json:"q"`
	G         []byte `json:"g"`
	Hk        []byte `json:"hk"`
	Tk        []byte `json:"tk"`
}

func GetGolbalChameleonParameter() ([][]byte, []byte, []byte, error) {
	local := &GolbalParameter{}
	err := Load(local, path.GetConfigPath())
	if err != nil {
		return nil, nil, nil, err
	}
	return [][]byte{local.P, local.Q, local.G}, local.Hk, local.Tk, nil
}

func CompareGolbalChameleonParameterWithLocal(para [][]byte) (bool, error) {
	local := &GolbalParameter{}
	err := Load(local, path.GetConfigPath())
	if err != nil {
		return false, err
	}
	return (bytes.Equal(local.P, para[0]) && bytes.Equal(local.Q, para[1]) && bytes.Equal(local.G, para[2])), nil
}

func GetCurrentBlockHeight() (int, error) {
	local := &GolbalParameter{}
	err := Load(local, path.GetConfigPath())
	if err != nil {
		return 0, err
	}
	return local.CurHeight, nil
}

func AddCurrentBlockHeight() error {
	local := &GolbalParameter{}
	err := Load(local, path.GetConfigPath())
	if err != nil {
		return err
	}
	local.CurHeight += 1
	err = Write(local, path.GetConfigPath())
	if err != nil {
		return err
	}
	return nil
}

type KeyStorage struct {
	Name    string `json:"name"`
	Key     string `json:"key"`
	Version int    `json:"version"`
}

// Example Tx Implementation
type BasicTx struct {
	PayloadB     []byte   `json:"payload"`
	ProofB       []byte   `json:"proof"`
	ChameleonPkB []byte   `json:"chameleon_public_key"`
	CheckStringB [][]byte `json:"check_string"`
	HashValB     []byte   `json:"hash"`
}

func NewBasicTx(payload []byte, proof []byte, pk []byte, para [][]byte) (*BasicTx, error) {
	if len(para) != 3 {
		return nil, errors.New("Invalid parameters,check your input!")
	}

	k := &KeyStorage{}
	err := json.Unmarshal(payload, k)
	if err != nil {
		return nil, errors.New("fail to unmarshal payload to struct KeyStorage:" + err.Error())
	}

	p := para[0]
	q := para[1]
	g := para[2]
	r := ch.Randgen(&q)
	s := ch.Randgen(&q)
	var hashout []byte
	ch.ChameleonHash(&pk, &p, &q, &g, &payload, &r, &s, &hashout)
	t := &BasicTx{
		PayloadB:     payload,
		ProofB:       proof,
		ChameleonPkB: pk,
		CheckStringB: [][]byte{r, s},
		HashValB:     hashout,
	}
	if !t.CheckProof() {
		return nil, errors.New("Error:invalid proof of transaction!")
	}

	return t, nil
}

func (t *BasicTx) Name() (string, error) {
	k := &KeyStorage{}
	err := json.Unmarshal(t.PayloadB, k)
	if err != nil {
		return "", errors.New("fail to unmarshal payload to struct KeyStorage:" + err.Error())
	}
	return k.Name, nil
}

func (t *BasicTx) Key() (string, error) {
	k := &KeyStorage{}
	err := json.Unmarshal(t.PayloadB, k)
	if err != nil {
		return "", errors.New("fail to unmarshal payload to struct KeyStorage:" + err.Error())
	}
	return k.Key, nil
}

func (t *BasicTx) Version() (int, error) {
	k := &KeyStorage{}
	err := json.Unmarshal(t.PayloadB, k)
	if err != nil {
		return 0, errors.New("fail to unmarshal payload to struct Key:" + err.Error())
	}
	return k.Version, nil
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

func (t *BasicTx) HashVal() []byte {
	return t.HashValB
}

func (t *BasicTx) Verify(pa interface{}) bool {
	para, ok := pa.([][]byte)
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

	if bytes.Equal(hashout, t.HashValB) {
		return t.CheckProof()
	}
	return false
}

func (t *BasicTx) Modify(payldNew interface{}, prfNew interface{}, privateKey interface{}, parameter interface{}) error {
	para, ok1 := parameter.([][]byte)
	payloadNew, ok2 := payldNew.([]byte)
	proofNew, ok3 := prfNew.([]byte)
	sk, ok4 := privateKey.([]byte)
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
	var r2, s2 []byte
	ch.GenerateCollision(&pk, &sk, &p, &q, &g, &old, &payloadNew, &r1, &s1, &r2, &s2)
	tNew := &BasicTx{
		PayloadB:     payloadNew,
		ProofB:       proofNew,
		ChameleonPkB: pk,
		CheckStringB: [][]byte{r2, s2},
		HashValB:     t.HashValB,
	}
	if !tNew.Verify(para) {
		return errors.New("Error:something in the payloadNew transaction is invalid!Collsion generating failed.")
	}
	t.PayloadB = payloadNew
	t.ProofB = proofNew
	t.CheckStringB = [][]byte{r2, s2}
	return nil
}

// Example Block Implementation
type BasicHead struct {
	Height             int      `json:"height"`
	Timestamp          int      `json:"timestamp"`
	TxCount            int      `json:"transactionCount"`
	HashRoot           []byte   `json:"hashRoot"`
	PreviousRoot       []byte   `json:"previous_root"`
	ChameleonParameter [][]byte `json:"chameleonParameter"`
}

type BasicBlock struct {
	HeadB         BasicHead `json:"head"`
	TransactionsB []BasicTx `json:"transactions"`
}

func NewBasicBlock(ChameleonParameter [][]byte) *BasicBlock {
	b := &BasicBlock{}
	b.HeadB.TxCount = 0
	b.HeadB.ChameleonParameter = ChameleonParameter
	return b
}

func (b *BasicBlock) Head() interface{} {
	return b.HeadB
}

func (b *BasicBlock) Transactions(index int) BasicTx {
	if index >= b.HeadB.TxCount {
		return BasicTx{}
	}
	return b.TransactionsB[index]
}

func (b *BasicBlock) GetTxIndexByHash(hash string) (bool, int) {
	for i := 0; i < b.HeadB.TxCount; i++ {
		if fmt.Sprintf("%x", b.Transactions(i).HashValB) == hash {
			return true, i
		}
	}
	return false, b.HeadB.TxCount
}

func (b *BasicBlock) TransactionCount() int {
	return b.HeadB.TxCount
}

func (b *BasicBlock) Verify() bool {
	var tree [][]byte
	for i := 0; i < b.HeadB.TxCount; i++ {
		if !b.TransactionsB[i].Verify(b.HeadB.ChameleonParameter) {
			return false
		}
		tree = append(tree, b.TransactionsB[i].HashVal())
	}
	for len(tree) > 1 {
		var tmp [][]byte
		for i := 1; i < len(tree); i += 2 {
			parent := sha256.Sum256(bytes.Join([][]byte{tree[i-1], tree[i]}, []byte("")))
			tmp = append(tmp, parent[:])
		}
		if len(tree)%2 == 1 {
			tmp = append(tmp, tree[len(tree)-1])
		}
		tree = tmp
	}
	root := sha256.Sum256(bytes.Join([][]byte{tree[0], b.HeadB.PreviousRoot}, []byte("")))
	return bytes.Equal(b.HeadB.HashRoot, root[:])
}

func (b *BasicBlock) AppendTx(t BasicTx) error {
	if !t.Verify(b.HeadB.ChameleonParameter) {
		return errors.New("Verify transaction failed!")
	}
	b.TransactionsB = append(b.TransactionsB, t)
	b.HeadB.TxCount += 1
	return nil
}

func (b *BasicBlock) Finalize(timestamp, height int, prvRoot []byte) error {
	var tree [][]byte
	for i := 0; i < b.HeadB.TxCount; i++ {
		if !b.TransactionsB[i].Verify(b.HeadB.ChameleonParameter) {
			return errors.New("Verify transaction " + strconv.Itoa(i) + " failed!")
		}
		tree = append(tree, b.TransactionsB[i].HashVal())
	}
	for len(tree) > 1 {
		var tmp [][]byte
		for i := 1; i < len(tree); i += 2 {
			parent := sha256.Sum256(bytes.Join([][]byte{tree[i-1], tree[i]}, []byte("")))
			tmp = append(tmp, parent[:])
		}
		if len(tree)%2 == 1 {
			tmp = append(tmp, tree[len(tree)-1])
		}
		tree = tmp
	}
	b.HeadB.Height = height
	b.HeadB.Timestamp = timestamp
	b.HeadB.PreviousRoot = prvRoot
	var root [32]byte
	if len(tree) != 0 {
		root = sha256.Sum256(bytes.Join([][]byte{tree[0], b.HeadB.PreviousRoot}, []byte("")))
	} else {
		root = sha256.Sum256(bytes.Join([][]byte{[]byte("default"), b.HeadB.PreviousRoot}, []byte("")))
	}
	b.HeadB.HashRoot = root[:]
	return nil
}

func (b *BasicBlock) ReplaceTx(t BasicTx, index int) error {
	old := b.Transactions(index)
	if len(old.HashVal()) == 0 {
		return errors.New("index ovweflow")
	}
	if !t.Verify(b.HeadB.ChameleonParameter) {
		return errors.New("invalid new transaction")
	}
	if !bytes.Equal(t.HashVal(), old.HashVal()) {
		return errors.New("new transaction hash different from old one")
	}
	b.TransactionsB[index] = t
	return nil
}
