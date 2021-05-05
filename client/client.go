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
			"3: get transaction by hash (args: hash,#startHeight)\n"+
			"4: create a new transaction (args: name,key,chameleonHk)\n"+
			"5: modify a existing transaction (args: height,txId,newVersion,newName,newKey,chameleonTk)\n"+
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
			name, err := tx.Name()
			if err != nil {
				fmt.Println(err)
				return
			}
			key, err := tx.Key()
			if err != nil {
				fmt.Println(err)
				return
			}
			version, err := tx.Version()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Name: %s\nKey: %s\nVersion: %d\nHk: %s\nHash: %x\n",
				name, key, version, tx.ChameleonPk(), tx.HashVal())
		}
	case 3:
		{
			args := flag.Args()
			if len(args) == 0 {
				fmt.Printf("need at least one args but get none")
				return
			}
			hash := args[0]
			var start int
			var err error
			if len(args) >= 2 {
				start, err = strconv.Atoi(args[1])
				if err != nil {
					fmt.Println(err)
					return
				}
			} else {
				start = 0
			}
			height, txId, tx, err := raftc.GetTxByHash(host, hash, start)
			fmt.Printf("height: %d, txId: %d\n", height, txId)
			name, err := tx.Name()
			if err != nil {
				fmt.Println(err)
				return
			}
			key, err := tx.Key()
			if err != nil {
				fmt.Println(err)
				return
			}
			version, err := tx.Version()
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Printf("Name: %s\nKey: %s\nVersion: %d\nHk: %s\n",
				name, key, version, tx.ChameleonPk())
		}
	case 4:
		//  "4: create a new transaction (args: name,key,chameleonHk)\n"+
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
			name := args[0]
			key := args[1]
			hk := []byte(args[2])
			res, err := raftc.SendNewTxReq(leader, name, key, []byte(""), hk)
			if err != nil {
				fmt.Println(err)
				return
			}
			fmt.Println(string(res))
		}
	case 5:
		//	"5: modify a existing transaction (args: height,txId,newVersion,newName,newKey,chameleonTk)\n"+
		{
			leader, err := raftc.GetCurrentLeader(host)
			if err != nil {
				fmt.Println(err)
				return
			}
			args := flag.Args()
			if len(args) != 6 {
				fmt.Printf("need %d args but get %d", 6, len(args))
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
			version, err := strconv.Atoi(args[2])
			if err != nil {
				fmt.Println(err)
				return
			}
			name := args[3]
			key := args[4]
			tk := []byte(args[5])
			res, err := raftc.SendModifyReq(leader, name, key, version, []byte(""), tk, height, txId)
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
