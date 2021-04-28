package main

import (
	"github.com/RedactableBlockChain/path"
	"log"

	//"encoding/json"
	"flag"
	"fmt"
	"github.com/RedactableBlockChain/data"
	//raftc "github.com/RedactableBlockChain/raft"
)

func init() {
	flag.Parse()
}

func main() {
	para, _, _, _ := data.GetGolbalChameleonParameter()
	height, txId := 2, 0

	block := &data.BasicBlock{}
	err := data.Load(block, path.GetBlockPath(height))
	if err != nil {
		fmt.Println(err)
		return
	}
	old := block.Transactions(txId)
	tx := old
	tk := []byte("6fb2bbde90050d39a1d916bbd259fc73")
	p1 := []byte("modified")
	p2 := []byte("by-me")
	tx.Modify(p1, p2, tk, para)
	fmt.Printf("Payload: %s\nProof: %s\nHk: %s", tx.Payload(), tx.Proof(), tx.ChameleonPk())

	if !tx.Verify(para) {
		fmt.Println("invalid tx transaction")
		return
	}

	err = block.ReplaceTx(tx, txId)
	if err != nil {
		fmt.Println(err)
		return
	}

	err = data.Write(block, path.GetBlockPath(height))
	if err != nil {
		fmt.Println(err)
		return
	}

	log.Printf(
		"transaction %x at block /%d/%d has been modified.\n before: payload: %s\n proof: %s\n after: payload: %s\n proof: %s\n",
		old.HashVal(), height, txId,
		old.Payload(), old.Proof(),
		tx.Payload(), tx.Proof())

	return
}
