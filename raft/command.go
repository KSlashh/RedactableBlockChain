package raft

import (
	"bytes"
	"errors"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
	"github.com/goraft/raft"
	"log"
	"os"
	"strconv"
)

// This command Modifys a transaction.
type ModifyCommand struct {
	BlockHeight        int          `json:"block-height"`
	TxId               int          `json:"tx-id"`
	NewTx              data.BasicTx `json:"new_tx"`
	ChameleonParameter [][]byte     `json:"chameleon_parameter"`
}

// Creates a new Modify command.
func NewModifyCommand(height, txId int, newtx data.BasicTx, para [][]byte) *ModifyCommand {
	return &ModifyCommand{
		BlockHeight:        height,
		TxId:               txId,
		NewTx:              newtx,
		ChameleonParameter: para,
	}
}

// The name of the command in the log.
func (c *ModifyCommand) CommandName() string {
	return "Modify Transaction"
}

// Modify a transaction.
func (c *ModifyCommand) Apply(server raft.Server) (interface{}, error) {

	para := c.ChameleonParameter
	flag, err := data.CompareGolbalChameleonParameterWithLocal(para)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, errors.New("global chameleon parameter in Modify request diff from local")
	}

	block := &data.BasicBlock{}
	err = data.Load(block, path.GetBlockPath(c.BlockHeight))
	if err != nil {
		return nil, err
	}
	old := block.Transactions(c.TxId)
	tx := c.NewTx

	if !bytes.Equal(old.HashVal(), tx.HashVal()) {
		return nil, errors.New("new_tx and old_tx have different hash value")
	}

	if !tx.Verify(para) {
		return nil, errors.New("invalid tx transaction")
	}

	err = block.ReplaceTx(tx, c.TxId)
	if err != nil {
		return nil, err
	}

	err = data.Write(block, path.GetBlockPath(c.BlockHeight))
	if err != nil {
		return nil, err
	}

	log.Printf(
		"transaction %x at block /%d/%d has been modified.\n before: payload: %s\n proof: %s\n after: payload: %s\n proof: %s\n",
		old.HashVal(), c.BlockHeight, c.TxId,
		old.Payload(), old.Proof(),
		tx.Payload(), tx.Proof())

	return nil, nil
}

// This command adds a new tx.
type AddTxCommand struct {
	Transaction        data.BasicTx `json:"transaction"`
	ChameleonParameter [][]byte     `json:"chameleon_parameter"`
}

// Creates a new tx command.
func NewAddTxCommand(tx data.BasicTx, para [][]byte) *AddTxCommand {
	return &AddTxCommand{
		Transaction:        tx,
		ChameleonParameter: para,
	}
}

// The name of the command in the log.
func (c *AddTxCommand) CommandName() string {
	return "Create New Transaction"
}

// Writes a tx to Txpool.
func (c *AddTxCommand) Apply(server raft.Server) (interface{}, error) {

	para := c.ChameleonParameter
	flag, err := data.CompareGolbalChameleonParameterWithLocal(para)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, errors.New("global chameleon parameter in new Tx request diff from local")
	}

	tx := c.Transaction
	if !tx.Verify(para) {
		return nil, errors.New("invalid transaction")
	}
	err = data.Write(tx, path.GetPoolTxPath(tx.HashVal()))
	if err != nil {
		return nil, err
	}

	log.Printf("new transaction. payload: %s ;proof: %s ; hk: %s \n", tx.Payload(), tx.Proof(), tx.ChameleonPk())

	return nil, nil
}

// This command packs a new block.
type PackCommand struct {
	BlockContent data.BasicBlock `json:"block_content"`
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
	flag, err := data.CompareGolbalChameleonParameterWithLocal(para)
	if err != nil {
		return nil, err
	}
	if !flag {
		return nil, errors.New("global chameleon parameter in pack request diff from local")
	}

	top, err := data.GetCurrentBlockHeight()
	if err != nil {
		return nil, err
	}
	if c.BlockContent.HeadB.Height != top+1 {
		return nil, errors.New("New block height invalid!,Expect: " + strconv.Itoa(top+1) + " Get: " + strconv.Itoa(c.BlockContent.HeadB.Height))
	}

	prvBlock := &data.BasicBlock{}
	err = data.Load(&prvBlock, path.GetBlockPath(top))
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(prvBlock.HeadB.HashRoot, c.BlockContent.HeadB.PreviousRoot) {
		return nil, errors.New("unmatched previous block hash root")
	}

	for i := 0; i < c.BlockContent.HeadB.TxCount; i++ {
		hash := c.BlockContent.TransactionsB[i].HashVal()
		_, err = os.Stat(path.GetPoolTxPath(hash))
		if err != nil && os.IsNotExist(err) {
			return nil, errors.New("transaction " + string(hash) + " does not exisit in pool")
		}
		if !c.BlockContent.TransactionsB[i].Verify(para) {
			return nil, errors.New("transaction " + string(hash) + " invaild")
		}
	}

	if !c.BlockContent.Verify() {
		return nil, errors.New("invaild Block")
	}

	data.AddCurrentBlockHeight()
	data.Write(c.BlockContent, path.GetBlockPath(c.BlockContent.HeadB.Height))

	for i := 0; i < c.BlockContent.HeadB.TxCount; i++ {
		hash := c.BlockContent.TransactionsB[i].HashVal()
		os.Remove(path.GetPoolTxPath(hash))
		if err != nil {
			return nil, err
		}
	}

	log.Printf("new block generated. height: %d; timestamp: %d; hashRoot: %x ",
		c.BlockContent.HeadB.Height, c.BlockContent.HeadB.Timestamp, c.BlockContent.HeadB.HashRoot)

	return nil, nil
}
