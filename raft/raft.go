package raft

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/RedactableBlockChain/data"
	"github.com/RedactableBlockChain/path"
	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"io/ioutil"
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	MIN_BLOCK_TX_NUM = 1
	MAX_BLOCK_TX_NUM = 100
)

// The raftd server is a combination of the Raft server and an HTTP
// server which acts as the transport.
type Server struct {
	name       string
	host       string
	port       int
	path       string
	epoch      int
	router     *mux.Router
	raftServer raft.Server
	httpServer *http.Server
	mutex      sync.RWMutex
}

// Creates a new server.
func New(path, host string, port, epoch int) *Server {
	s := &Server{
		host:   host,
		port:   port,
		path:   path,
		epoch:  epoch,
		router: mux.NewRouter(),
	}

	// Read existing name or generate a new one.
	if b, err := ioutil.ReadFile(filepath.Join(path, "name")); err == nil {
		s.name = string(b)
	} else {
		s.name = fmt.Sprintf("%07x", rand.Int())[0:7]
		if err = ioutil.WriteFile(filepath.Join(path, "name"), []byte(s.name), 0644); err != nil {
			panic(err)
		}
	}

	return s
}

// Returns the connection string.
func (s *Server) connectionString() string {
	return fmt.Sprintf("http://%s:%d", s.host, s.port)
}

// Starts the server.
func (s *Server) ListenAndServe(leader string) error {
	var err error

	log.Printf("Initializing Raft Server: %s", s.path)

	// Initialize and start Raft server.
	transporter := raft.NewHTTPTransporter("/raft", 200*time.Millisecond)
	s.raftServer, err = raft.NewServer(s.name, s.path, transporter, nil, nil, "")
	if err != nil {
		log.Fatal(err)
	}
	transporter.Install(s.raftServer, s)
	s.raftServer.Start()

	if leader != "" {
		// Join to leader if specified.

		log.Println("Attempting to join leader:", leader)

		if !s.raftServer.IsLogEmpty() {
			log.Fatal("Cannot join with an existing log")
		}
		if err := s.Join(leader); err != nil {
			log.Fatal(err)
		}

	} else if s.raftServer.IsLogEmpty() {
		// Initialize the server by joining itself.

		log.Println("Initializing new cluster")

		_, err := s.raftServer.Do(&raft.DefaultJoinCommand{
			Name:             s.raftServer.Name(),
			ConnectionString: s.connectionString(),
		})
		if err != nil {
			log.Fatal(err)
		}

	} else {
		log.Println("Recovered from log")
	}

	log.Println("Initializing HTTP server")

	// Initialize and start HTTP server.
	s.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: s.router,
	}

	s.router.HandleFunc("/get_transaction_by_hash/{hash}/{startHeight}", s.getTxByHashHandler).Methods("GET")
	s.router.HandleFunc("/get_transaction_by_index/{height}/{txId}", s.getTxByIndexHandler).Methods("GET")
	s.router.HandleFunc("/get_block_by_height/{height}", s.getBlockByHeightHandler).Methods("GET")
	s.router.HandleFunc("/get_current_height", s.getCurrentHeightHandler).Methods("GET")
	s.router.HandleFunc("/get_current_leader", s.getCurrentLeaderHandler).Methods("GET")
	s.router.HandleFunc("/modify/{height}/{txId}", s.modifyHandler).Methods("POST")
	s.router.HandleFunc("/new_block", s.newBlockHandler).Methods("POST")
	s.router.HandleFunc("/new_transaction", s.newTxHandler).Methods("POST")
	s.router.HandleFunc("/join", s.joinHandler).Methods("POST")

	log.Println("Listening at:", s.connectionString())

	go s.Mint()

	return s.httpServer.ListenAndServe()
}

func (s *Server) Mint() {
	flag := 0
	for {
		time.Sleep(time.Duration(s.epoch) * time.Millisecond)
		if s.raftServer.State() == raft.Leader {
			minTxCount, maxTxCount := MIN_BLOCK_TX_NUM, MAX_BLOCK_TX_NUM
			para, _, _, err := data.GetGolbalChameleonParameter()
			if err != nil {
				log.Fatal(err)
				continue
			}
			block := data.NewBasicBlock(para)
			count := 0
			filepath.Walk(path.GetTxPoolPath(), func(path string, info os.FileInfo, e error) error {
				if count > maxTxCount {
					return nil
				}
				if info.IsDir() {
					return nil
				}
				t := &data.BasicTx{}
				er := data.Load(&t, path)
				if er != nil {
					return er
				}
				er = block.AppendTx(*t)
				if er != nil {
					return er
				}
				count += 1
				return nil
			})
			if count < minTxCount {
				if flag == 0 {
					log.Println("Skip block mint:no transaction in pool")
					flag += 1
				}
				continue
			}
			flag = 0
			top, err := data.GetCurrentBlockHeight()
			if err != nil {
				log.Fatal(err)
				continue
			}
			prvBlock := &data.BasicBlock{}
			err = data.Load(&prvBlock, path.GetBlockPath(top))
			if err != nil {
				log.Fatal(err)
				continue
			}
			err = block.Finalize(int(time.Now().Unix()), top+1, prvBlock.HeadB.HashRoot)
			if err != nil {
				log.Fatal(err)
				continue
			}
			_, err = s.raftServer.Do(NewPackCommand(*block))
			if err != nil {
				log.Fatal(err)
				continue
			}
		}
	}
}

// This is a hack around Gorilla mux not providing the correct net/http
// HandleFunc() interface.
func (s *Server) HandleFunc(pattern string, handler func(http.ResponseWriter, *http.Request)) {
	s.router.HandleFunc(pattern, handler)
}

// Joins to the leader of an existing cluster.
func (s *Server) Join(leader string) error {
	command := &raft.DefaultJoinCommand{
		Name:             s.raftServer.Name(),
		ConnectionString: s.connectionString(),
	}

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(command)
	resp, err := http.Post(fmt.Sprintf("http://%s/join", leader), "application/json", &b)
	if err != nil {
		return err
	}
	resp.Body.Close()

	return nil
}

func (s *Server) joinHandler(w http.ResponseWriter, req *http.Request) {
	command := &raft.DefaultJoinCommand{}

	if err := json.NewDecoder(req.Body).Decode(&command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if _, err := s.raftServer.Do(command); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Client function
func SendNewTxReq(host, name, key string, proof, hk []byte) (returnData []byte, err error) {
	para, _, _, err := data.GetGolbalChameleonParameter()
	if err != nil {
		return nil, err
	}
	rawKey := &data.KeyStorage{
		Name:    name,
		Key:     key,
		Version: 1,
	}
	payload, err := json.Marshal(rawKey)
	if err != nil {
		return nil, err
	}
	tx, err := data.NewBasicTx(payload, proof, hk, para)
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(tx)
	_data := bytes.NewReader(content)
	resp, err := http.Post(host+"/new_transaction", "application/json", _data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func SendNewBlockReq(host string, minTxCount, maxTxCount int) (returnData []byte, err error) {
	para, _, _, err := data.GetGolbalChameleonParameter()
	if err != nil {
		return nil, err
	}
	block := data.NewBasicBlock(para)
	count := 0
	filepath.Walk(path.GetTxPoolPath(), func(path string, info os.FileInfo, e error) error {
		if count > maxTxCount {
			return nil
		}
		if info.IsDir() {
			return nil
		}
		t := &data.BasicTx{}
		er := data.Load(&t, path)
		if er != nil {
			return er
		}
		er = block.AppendTx(*t)
		if er != nil {
			return er
		}
		count += 1
		return nil
	})
	if count < minTxCount {
		return nil, errors.New("no enough transactions in pool")
	}
	top, err := data.GetCurrentBlockHeight()
	if err != nil {
		return nil, err
	}
	prvBlock := &data.BasicBlock{}
	err = data.Load(&prvBlock, path.GetBlockPath(top))
	if err != nil {
		return nil, err
	}
	err = block.Finalize(int(time.Now().Unix()), top+1, prvBlock.HeadB.HashRoot)
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(block)
	_data := bytes.NewReader(content)
	resp, err := http.Post(host+"/new_block", "application/json", _data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func SendModifyReq(host, nameNew, keyNew string, versionNew int, proofNew, tk []byte, height, txId int) (returnData []byte, err error) {
	para, _, _, err := data.GetGolbalChameleonParameter()
	if err != nil {
		return nil, err
	}
	tx, err := GetTxByIndex(host, height, txId)
	if err != nil {
		return nil, err
	}
	rawKey := &data.KeyStorage{
		Name:    nameNew,
		Key:     keyNew,
		Version: versionNew,
	}
	payloadNew, err := json.Marshal(rawKey)
	if err != nil {
		return nil, err
	}
	versionOld, err := tx.Version()
	if err != nil {
		return nil, err
	}
	if versionNew <= versionOld {
		return nil, errors.New(fmt.Sprintf("new tx version must be greater than the old one, want >%d, got %d", versionOld, versionNew))
	}
	err = tx.Modify(payloadNew, proofNew, tk, para)
	if err != nil {
		return nil, err
	}
	content, err := json.Marshal(tx)
	_data := bytes.NewReader(content)
	resp, err := http.Post(fmt.Sprintf("%s/modify/%d/%d", host, height, txId), "application/json", _data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return res, nil
}

func GetCurrentHeight(host string) (height int, err error) {
	resp, err := http.Get(host + "/get_current_height")
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, err
	}
	height, err = strconv.Atoi(string((res)))
	if err != nil {
		return 0, err
	}
	return height, nil
}

func GetBlockByHeight(host string, height int) (block *data.BasicBlock, err error) {
	resp, err := http.Get(host + "/get_block_by_height/" + strconv.Itoa(height))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	block = &data.BasicBlock{}
	err = json.Unmarshal(res, &block)
	if err != nil {
		return nil, err
	}
	return block, nil
}

func GetTxByIndex(host string, height, txId int) (tx *data.BasicTx, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/get_transaction_by_index/%d/%d", host, height, txId))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	tx = &data.BasicTx{}
	err = json.Unmarshal(res, &tx)
	if err != nil {
		return nil, err
	}
	return tx, nil
}

func GetTxByHash(host, hash string, startHeight int) (height, txId int, tx *data.BasicTx, err error) {
	resp, err := http.Get(fmt.Sprintf("%s/get_transaction_by_hash/%s/%d", host, hash, startHeight))
	if err != nil {
		return 0, 0, nil, err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, nil, err
	}
	index := strings.Split(string(res), "-")
	height, err = strconv.Atoi(index[0])
	if err != nil {
		return height, txId, nil, err
	}
	txId, err = strconv.Atoi(index[1])
	if err != nil {
		return height, txId, nil, err
	}
	tx, err = GetTxByIndex(host, height, txId)
	if err != nil {
		return height, txId, nil, err
	}
	return height, txId, tx, nil
}

func GetCurrentLeader(host string) (leader string, err error) {
	resp, err := http.Get(host + "/get_current_leader")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	res, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(res), nil
}

// Server handler
func (s *Server) newTxHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	var tx = &data.BasicTx{}
	err = json.Unmarshal(content, &tx)
	if err != nil {
		return
	}
	para, _, _, err := data.GetGolbalChameleonParameter()
	if err != nil {
		return
	}
	_, err = s.raftServer.Do(NewAddTxCommand(*tx, para))
	if err != nil {
		return
	}
	h, _ := data.GetCurrentBlockHeight()
	w.Write([]byte("Success:Trancasion " + fmt.Sprintf("%x", tx.HashVal()) + " is waitting for packing.Temporary block height: " + strconv.Itoa(h)))
}

func (s *Server) newBlockHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	var block = &data.BasicBlock{}
	err = json.Unmarshal(content, &block)
	if err != nil {
		return
	}
	_, err = s.raftServer.Do(NewPackCommand(*block))
	if err != nil {
		return
	}
	w.Write([]byte("Success:Block height: " + strconv.Itoa(block.HeadB.Height)))
}

func (s *Server) modifyHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	vars := mux.Vars(req)
	height, err := strconv.Atoi(vars["height"])
	if err != nil {
		return
	}
	txId, err := strconv.Atoi(vars["txId"])
	if err != nil {
		return
	}
	content, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return
	}
	var tx = &data.BasicTx{}
	err = json.Unmarshal(content, &tx)
	if err != nil {
		return
	}
	para, _, _, err := data.GetGolbalChameleonParameter()
	if err != nil {
		return
	}
	_, err = s.raftServer.Do(NewModifyCommand(height, txId, *tx, para))
	if err != nil {
		return
	}
	w.Write([]byte("Success:Transaction " + fmt.Sprintf("%x", tx.HashValB) + " has been modified"))
}

func (s *Server) getCurrentHeightHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	height, err := data.GetCurrentBlockHeight()
	if err != nil {
		return
	}
	w.Write([]byte(strconv.Itoa(height)))
}

func (s *Server) getBlockByHeightHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	vars := mux.Vars(req)
	height, err := strconv.Atoi(vars["height"])
	if err != nil {
		return
	}
	var block = &data.BasicBlock{}
	err = data.Load(&block, path.GetBlockPath(height))
	if err != nil {
		return
	}
	resp, err := json.Marshal(block)
	if err != nil {
		return
	}
	w.Write(resp)
}

func (s *Server) getTxByIndexHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	vars := mux.Vars(req)
	height, err := strconv.Atoi(vars["height"])
	if err != nil {
		return
	}
	txId, err := strconv.Atoi(vars["txId"])
	if err != nil {
		return
	}
	var block = &data.BasicBlock{}
	err = data.Load(&block, path.GetBlockPath(height))
	if err != nil {
		return
	}
	tx := block.Transactions(txId)
	if len(tx.HashVal()) == 0 {
		err = errors.New("transaction index overflow")
		return
	}
	resp, err := json.Marshal(tx)
	if err != nil {
		return
	}
	w.Write(resp)
}

func (s *Server) getTxByHashHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	vars := mux.Vars(req)
	hash := vars["hash"]
	startHeight, err := strconv.Atoi(vars["startHeight"])
	if err != nil {
		return
	}
	currentHeight, err := data.GetCurrentBlockHeight()
	if err != nil {
		return
	}
	for i := startHeight; i <= currentHeight; i++ {
		var block = &data.BasicBlock{}
		err = data.Load(block, path.GetBlockPath(i))
		if err != nil {
			return
		}
		flag, index := block.GetTxIndexByHash(hash)
		if flag {
			resp := strings.Join([]string{strconv.Itoa(i), strconv.Itoa(index)}, "-")
			if err != nil {
				return
			}
			w.Write([]byte(resp))
			return
		}
	}
	log.Printf("not found %s", hash)
	http.Error(w, errors.New("transaction not found").Error(), http.StatusNotFound)
}

func (s *Server) getCurrentLeaderHandler(w http.ResponseWriter, req *http.Request) {
	var err error
	defer func() {
		if err != nil {
			log.Println(err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}()
	if s.raftServer.State() == raft.Leader {
		w.Write([]byte(s.connectionString()))
	} else {
		log.Printf("leader now:%s\n", s.raftServer.Peers()[s.raftServer.Leader()].ConnectionString)
		w.Write([]byte(s.raftServer.Peers()[s.raftServer.Leader()].ConnectionString))
	}
}
