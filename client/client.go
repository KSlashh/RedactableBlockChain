package main

import (
	"flag"
	"fmt"
	ch "github.com/RedactableBlockChain/chameleon"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
	raftc "github.com/RedactableBlockChain/raft"
	"strconv"
)

var host string
var function int
var configPath string
var txPoolPath string
var blockPath string

func init() {
	flag.StringVar(&host, "h", "http://localhost:6666", "Restful url. default: http://localhost:6666")
	flag.StringVar(&configPath, "config", "./storage/config", "Config file path")
	flag.StringVar(&txPoolPath, "pool", "./storage/pool/", "Transaction pool dir")
	flag.StringVar(&blockPath, "block", "./storage/block/", "Block storage dir")
	flag.IntVar(&function, "func", 0,
		"Choose one function below:\n"+
			"0: get current height (args: nil)\n"+
			"1: get block by height and store it(args: height)\n"+
			"2: get transaction by index (args: height,transactionId)\n"+
			"3: get transaction by hash (args: hash,startHeight)\n"+
			"4: create a new transaction (args: payload,proof,hk)\n"+
			"5: modify a exisiting transaction (args: height,txId,payload,proof,tk)\n"+
			"6: generate chameleon key pair (args: flag)\n"+
			"  -- flag is 0 : default key pair\n"+
			"  -- else : new random key pair\n"+
			"7: get current leader of raft (args: nil)")

	flag.Parse()
}

func main() {

	path.SetBlockDirPath(blockPath)
	path.SetConfigPath(configPath)
	path.SetTxPoolPath(txPoolPath)

	switch function {
	case 0:
		{
			height, err := raftc.GetCurrentHeight(host)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Current height: %d", height)
		}
	case 1:
		{
			args := flag.Args()
			if len(args) != 1 {
				fmt.Printf("need %d args but get %d", 1, len(args))
				return
			}
			height, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			block, err := raftc.GetBlockByHeight(host, height)
			//err = data.Write(&block, path.GetBlockPath(height))
			//if err != nil {
			//	fmt.Println(err)
			//}
			fmt.Printf("Height: %d\nTimestamp: %d\nTransactions amount: %d\nHash root: %x\nPrevious root: %x\n",
				block.HeadB.Height, block.HeadB.Timestamp, block.HeadB.TxCount, block.HeadB.HashRoot, block.HeadB.PreviousRoot)
		}
	case 2:
		{
			args := flag.Args()
			if len(args) != 2 {
				fmt.Printf("need %d args but get %d", 2, len(args))
				return
			}
			height, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			txId, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println(err)
				return
			}
			tx, err := raftc.GetTxByIndex(host, height, txId)
			if err != nil {
				fmt.Println(err)
			}
			fmt.Printf("Payload: %s\nProof: %s\nHk: %s\nHash: %x\n",
				tx.Payload(), tx.Proof(), tx.ChameleonPk(), tx.HashVal())
		}
	case 3:
		{
			args := flag.Args()
			if len(args) != 2 {
				fmt.Printf("need %d args but get %d", 2, len(args))
				return
			}
			hash := args[0]
			start, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println(err)
				return
			}
			height, txId, tx, err := raftc.GetTxByHash(host, hash, start)
			fmt.Printf("height: %d, txId: %d\n", height, txId)
			fmt.Printf("Payload: %s\nProof: %s\nHk: %s", tx.Payload(), tx.Proof(), tx.ChameleonPk())
		}
	case 4:
		{
			leader, err := raftc.GetCurrentLeader(host)
			if err != nil {
				fmt.Println(err)
				return
			}
			args := flag.Args()
			if len(args) != 3 {
				fmt.Printf("need %d args but get %d", 3, len(args))
				return
			}
			payload := []byte(args[0])
			proof := []byte(args[1])
			hk := []byte(args[2])
			res, err := raftc.SendNewTxReq(leader, payload, proof, hk)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(res))
		}
	case 5:
		{
			leader, err := raftc.GetCurrentLeader(host)
			if err != nil {
				fmt.Println(err)
				return
			}
			args := flag.Args()
			if len(args) != 5 {
				fmt.Printf("need %d args but get %d", 5, len(args))
				return
			}
			height, err := strconv.Atoi(args[0])
			if err != nil {
				fmt.Println(err)
				return
			}
			txId, err := strconv.Atoi(args[1])
			if err != nil {
				fmt.Println(err)
				return
			}
			payload := []byte(args[2])
			proof := []byte(args[3])
			tk := []byte(args[4])
			res, err := raftc.SendModifyReq(leader, payload, proof, tk, height, txId)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(res))
		}
	case 6:
		{
			args := flag.Args()
			if len(args) != 1 {
				fmt.Printf("need %d args but get %d", 1, len(args))
				return
			}
			para := &data.GolbalParameter{}
			err := data.Load(&para, path.GetConfigPath())
			if err != nil {
				fmt.Println(err)
				return
			}
			if args[0] == "0" {
				fmt.Printf("PublicKey: %s\nPrivateKey: %s\n", para.Hk, para.Tk)
			} else {
				var hk, tk []byte
				ch.Keygen(para.Bits, para.P, para.Q, para.G, &hk, &tk)
				fmt.Printf("PublicKey: %s\nPrivateKey: %s\n", hk, tk)
			}
		}
	case 7:
		{
			leader, err := raftc.GetCurrentLeader(host)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(leader)
		}
	}

}
