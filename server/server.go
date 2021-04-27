package main

import (
	"flag"
	"fmt"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
	raftc "github.com/RedactableBlockChain/raft"
	"github.com/goraft/raft"
	"log"
	"math/rand"
	"os"
	"time"
)

var verbose bool
var trace bool
var debug bool
var host string
var port int
var interval int
var join string
var configPath string
var txPoolPath string
var blockPath string

func init() {
	flag.BoolVar(&verbose, "v", false, "verbose logging")
	flag.BoolVar(&trace, "trace", false, "Raft trace debugging")
	flag.BoolVar(&debug, "debug", false, "Raft debugging")
	flag.StringVar(&host, "h", "localhost", "hostname")
	flag.StringVar(&configPath, "config", "./storage/config", "Config file path")
	flag.StringVar(&txPoolPath, "pool", "./storage/pool/", "Transaction pool dir")
	flag.StringVar(&blockPath, "blockdir", "./storage/block/", "Block storage dir")
	flag.IntVar(&port, "p", 6666, "port")
	flag.IntVar(&interval, "t", 10000, "block interval (uint ms)")
	flag.StringVar(&join, "join", "", "host:port of leader to join")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [arguments] <data-path> \n", os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {
	log.SetFlags(0)
	flag.Parse()
	if verbose {
		log.Print("Verbose logging enabled.")
	}
	if trace {
		raft.SetLogLevel(raft.Trace)
		log.Print("Raft trace debugging enabled.")
	} else if debug {
		raft.SetLogLevel(raft.Debug)
		log.Print("Raft debugging enabled.")
	}

	rand.Seed(time.Now().UnixNano())

	// Setup commands.
	raft.RegisterCommand(&raftc.ModifyCommand{})
	raft.RegisterCommand(&raftc.AddTxCommand{})
	raft.RegisterCommand(&raftc.PackCommand{})

	// Set up blockchain related dirs
	path.SetBlockDirPath(blockPath)
	path.SetConfigPath(configPath)
	path.SetTxPoolPath(txPoolPath)
	if !PathExists(path.GetBlockDirPath()) {
		os.Mkdir(path.GetBlockDirPath(), os.ModePerm)
	}
	if !PathExists(path.GetTxPoolPath()) {
		os.Mkdir(path.GetTxPoolPath(), os.ModePerm)
	}
	if !PathExists(path.GetConfigPath()) {
		log.Fatalf("Cannot find config file!")
	}
	if !PathExists(path.GetBlockPath(0)) {
		para, _, _, err := data.GetGolbalChameleonParameter()
		if err != nil {
			log.Fatalf("Error while create genesis block: %v", err)
		}
		block := data.NewBasicBlock(para)
		err = block.Finalize(0, 0, []byte(""))
		if err != nil {
			log.Fatalf("Error while create genesis block: %v", err)
		}
		err = data.Write(&block, path.GetBlockPath(0))
		if err != nil {
			log.Fatalf("Error while create genesis block: %v", err)
		}
	}

	// Set the data directory.
	if flag.NArg() == 0 {
		flag.Usage()
		log.Fatal("Data path argument required")
	}
	path := flag.Arg(0)
	if err := os.MkdirAll(path, 0744); err != nil {
		log.Fatalf("Unable to create path: %v", err)
	}

	log.SetFlags(log.LstdFlags)
	s := raftc.New(path, host, port, interval)
	log.Fatal(s.ListenAndServe(join))
}

func PathExists(path string) bool {
	_, err := os.Stat(path)
	if err == nil {
		return true
	}
	if os.IsNotExist(err) {
		return false
	}
	return false
}
