package raft

import (
	"github.com/goraft/raft"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
	"strconv"
	"bytes"
	"errors"
	"os"
)


// This command Modifys a transaction.
type ModifyCommand struct {
	BlockHeight  int `json:"block-height"`
	TxId     int `json:"tx-id"`
	Payload []byte `json:"payload"`
	Proof   []byte `json:"proof"`
	P   []byte `json:"p"`
	Q   []byte `json:"q"`
	G   []byte `json:"g"`
	Hk  []byte `json:"hk"`
}

// Creates a new Modify command.
func NewModifyCommand(block,tx int, payload,proof,p,q,g,hk []byte) *ModifyCommand {
	return &ModifyCommand{
		BlockHeight:  block,
		TxId:  tx,
		Payload: payload,
		Proof: proof,
		P:   p,
		Q:   q,
		G:   g,
		Hk:  hk,
	}
}

// The name of the command in the log.
func (c *ModifyCommand) CommandName() string {
	return "Modify Transaction"
}

// Modify a transaction.
func (c *ModifyCommand) Apply(server raft.Server) (interface{}, error) {

	flag,err := data.CompareGolbalChameleonParameterWithLocal(c.P,c.Q,c.G)
	if err != nil {
		return nil,err
	}
	if !flag {
		return nil,errors.New("global chameleon parameter in Modify request diff from local")
	}

	block := &data.BasicBlock{}
	err = data.Load(block, path.GetBlockPath(c.BlockHeight))
	if err != nil {
		return nil,err
	}
	old := block.Transactions(c.TxId)
	new,err := data.NewBasicTx(c.Payload,c.Proof,c.Hk,[][]byte{c.P,c.Q,c.G})
	if err != nil {
		return nil,err
	}

	if !bytes.Equal(old.HashVal(),new.HashVal()) {
		return nil,errors.New("new_tx and old_tx have different hash value")
	}

	if !new.Verify([][]byte{c.P,c.Q,c.G}) {
		return nil,errors.New("invalid new transaction")
	}

	err = block.ReplaceTx(new, c.TxId)
	if err != nil {
		return nil,err
	}

	err = data.Write(block, path.GetBlockPath(c.BlockHeight))
	if err != nil {
		return nil,err
	}

	return nil, nil
}




// This command adds a new tx.
type AddTxCommand struct {
	Payload []byte `json:"payload"`
	Proof   []byte `json:"proof"`
	P   []byte `json:"p"`
	Q   []byte `json:"q"`
	G   []byte `json:"g"`
	Hk  []byte `json:"hk"`
}

// Creates a new tx command.
func NewAddTxCommand(payload,proof,p,q,g,hk []byte) *AddTxCommand {
	return &AddTxCommand{
		Payload: payload,
		Proof: proof,
		P: p,
		Q: q,
		G: g,
		Hk: hk,
	}
}

// The name of the command in the log.
func (c *AddTxCommand) CommandName() string {
	return "Create New Transaction"
}

// Writes a tx to Txpool.
func (c *AddTxCommand) Apply(server raft.Server) (interface{}, error) {

	flag,err := data.CompareGolbalChameleonParameterWithLocal(c.P,c.Q,c.G)
	if err != nil {
		return nil,err
	}
	if !flag {
		return nil,errors.New("global chameleon parameter in Modify request diff from local")
	}

	tx,err := data.NewBasicTx(c.Payload,c.Proof,c.Hk,[][]byte{c.P,c.Q,c.G})
	if err != nil {
		return nil,err
	}

	err = data.Write(tx, path.GetPoolTxPath(tx.HashVal()))
	if err != nil {
		return nil,err
	}

	return nil, nil
}


// This command packs a new block.
type PackCommand struct {
	BlockContent   data.BasicBlock `json:"block_content"`
}

// Creates a new block command.
func NewPackCommand(block data.BasicBlock) *PackCommand {
	return &PackCommand{
		BlockContent: block,
	}
}

// The name of the command in the log.
func (c *PackCommand) CommandName() string {
	return "Generate New Block"
}

//Pack some tx to a block.
func (c *PackCommand) Apply(server raft.Server) (interface{}, error) {
	para := c.BlockContent.HeadB.ChameleonParameter
	flag,err := data.CompareGolbalChameleonParameterWithLocal(para[0], para[1], para[2])
	if err != nil {
		return nil,err
	}
	if !flag {
		return nil,errors.New("global chameleon parameter in Modify request diff from local")
	}

	top,err := data.GetCurrentBlockHeight()
	if err != nil {
		return nil,err
	}
	if c.BlockContent.HeadB.Height!=top+1 {
		return nil,errors.New("New block height invalid!,Expect: "+strconv.Itoa(top+1)+" Get: "+strconv.Itoa(c.BlockContent.HeadB.Height))
	}

	for i:=0;i<c.BlockContent.HeadB.TxCount;i++ {
		hash := c.BlockContent.TransactionsB[i].HashVal()
		_,err = os.Stat(path.GetPoolTxPath(hash))
		if err != nil && os.IsNotExist(err)  {
			return nil,errors.New("transaction "+string(hash)+" does not exisit in pool")
		}
		if !c.BlockContent.TransactionsB[i].Verify(para) {
			return nil,errors.New("transaction "+string(hash)+" invaild")
		}
	}

	data.AddCurrentBlockHeight()

	if !c.BlockContent.Verify() {
		return nil,errors.New("invaild Block")
	}

	for i:=0;i<c.BlockContent.HeadB.TxCount;i++ {
		hash := c.BlockContent.TransactionsB[i].HashVal()
		os.Remove(path.GetPoolTxPath(hash))
		if err != nil {
			return nil,err
		}
	}

	return nil,nil
}