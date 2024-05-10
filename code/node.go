package main

import (
	"bytes"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	bc "github.com/Sunr1s/chain/blockchain"
	nt "github.com/Sunr1s/chain/network"
	color "github.com/fatih/color"

	_ "github.com/mattn/go-sqlite3"
)

var (
	info     = color.New(color.FgGreen).PrintlnFunc()
	warn     = color.New(color.FgYellow).PrintlnFunc()
	errColor = color.New(color.FgRed).PrintlnFunc()
	bcolor   = color.New(color.BgCyan).PrintFunc()
)
var (
	Filename      string
	Serve         string
	Chain         *bc.BlockChain
	Block         *bc.Block
	Mempool       = *bc.NewPool()
	Mutex         sync.Mutex
	IsMining      bool
	BreakMining   = make(chan bool)
	ProbablyLhash []byte
	Addresses     []string
	SleepMode     = make(chan bool, 2)
)

func init() {
	if len(os.Args) < 2 {
		errColor("Insufficient arguments.")
		os.Exit(1)
	}
	var (
		serveStr     = ""
		addrStr      = ""
		userNewStr   = ""
		userLoadStr  = ""
		chainNewStr  = ""
		chainLoadStr = ""
	)
	var (
		serveExist     = false
		addrExist      = false
		userNewExist   = false
		userLoadExist  = false
		chainLoadExist = false
		chainNewExist  = false
	)
	for i := 1; i < len(os.Args); i++ {
		arg := os.Args[i]
		switch {
		case strings.HasPrefix(arg, "-serve:"):
			serveStr = strings.Replace(arg, "-serve:", "", 1)
			serveExist = true
		case strings.HasPrefix(arg, "-newchain:"):
			chainNewStr = strings.Replace(arg, "-newchain:", "", 1)
			chainNewExist = true
		case strings.HasPrefix(arg, "-loadchain:"):
			chainLoadStr = strings.Replace(arg, "-loadchain:", "", 1)
			chainLoadExist = true
		case strings.HasPrefix(arg, "-loadaddr:"):
			addrStr = strings.Replace(arg, "-loadaddr:", "", 1)
			addrExist = true
		case strings.HasPrefix(arg, "-newuser:"):
			userNewStr = strings.Replace(arg, "-newuser:", "", 1)
			userNewExist = true
		case strings.HasPrefix(arg, "-loaduser:"):
			userLoadStr = strings.Replace(arg, "-loaduser:", "", 1)
			userLoadExist = true
		}
	}
	if !(userNewExist || userLoadExist || !addrExist || !serveExist || !chainNewExist || chainLoadExist) {
		errColor("Incorrect combination of flags.")
		os.Exit(1)
	}
	Serve = serveStr
	addrStr = "addr.json"
	err := json.Unmarshal([]byte(readFile(addrStr)), &Addresses)
	if err != nil {
		errColor("Failed to read addresses:", err)
		os.Exit(1)
	}
	var mapaddr = make(map[string]bool)
	for _, addr := range Addresses {
		if addr == Serve {
			continue
		}
		if _, ok := mapaddr[addr]; ok {
			continue
		}
		mapaddr[addr] = true
		Address = append(Address, addr)
	}
	if userNewExist {
		User = userNew(userNewStr)
	}
	if userLoadExist {
		User = userLoad(userLoadStr)
	}
	if User == nil {
		errColor("Failed to initialize user.")
		os.Exit(1)
	}
	if chainNewExist {
		Filename = chainNewStr
		Chain = chainNew(chainNewStr)
	}
	if chainLoadExist {
		Filename = chainLoadStr
		Chain = chainLoad(chainLoadStr)
	}
	if Chain == nil {
		errColor("Failed to initialize blockchain.")
		os.Exit(1)
	}
	Block = bc.NewBlock(User.Address(), Chain.LastHash())
	info("Initialization successful.")
}

func main() {
	nt.Listen(Serve, handleServer)
	go runMining()
	for {
		fmt.Scanln()
	}
}

func runMining() string {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for range ticker.C {
		shouldMine := processMempool()
		if shouldMine {
			performMining()
		}
	}
	return "ok"
}

func processMempool() bool {
	mempoolSize := Mempool.Size()
	if mempoolSize >= bc.TXS_LIMIT {
		handleMempoolTransactions(mempoolSize)
		return len(Block.Transactions) == bc.TXS_LIMIT
	}
	return false
}

func handleMempoolTransactions(mempoolSize int) {
	for _, txs := range Mempool.Get(bc.TXS_LIMIT - len(Block.Transactions)) {
		if ProbablyLhash != nil {
			Mempool.UpdateMempoolTransactions(ProbablyLhash)
		} else {
			Mempool.UpdateMempoolTransactions(Chain.LastHash())
		}

		err := Block.AddTransaction(Chain, &txs)
		if err != nil {
			fmt.Println(err.Error())
		}
		if len(Block.Transactions) == bc.TXS_LIMIT {
			break
		}
	}
}

func performMining() {
	Mutex.Lock()
	block := *Block
	IsMining = true
	Mutex.Unlock()
	res := (&block).Accept(Chain, User)
	ProbablyLhash = (&block).CurrHash
	(&block).Mining(Chain, BreakMining, Addresses, SleepMode)
	Mutex.Lock()
	IsMining = false
	if res == nil && bytes.Equal(block.PrevHash, Block.PrevHash) {
		Chain.AddBlock(&block)
		pushBlockToNet(&block)
		fmt.Println("Hash good")
	} else {
		fmt.Println("Hash err ", res)
	}
	Block = bc.NewBlock(User.Address(), Chain.LastHash())
	Mutex.Unlock()
}

func handleServer(conn nt.Conn, pack *nt.Package) {
	nt.Handle(ADD_BLOCK, conn, pack, addBlock)
	nt.Handle(ADD_TRNSX, conn, pack, handleTransaction)
	nt.Handle(GET_BLOCK, conn, pack, getBlock)
	nt.Handle(GET_LHASH, conn, pack, getLastHash)
	nt.Handle(GET_BLNCE, conn, pack, getBalance)
	nt.Handle(WAKEUP_MSG, conn, pack, wakeUpHandler)
}

func wakeUpHandler(pack *nt.Package) string {
	select {
	case status := <-SleepMode:
		SleepMode <- status
		if status {
			return "asleep"
		} else {
			return "awake"
		}
	default:
		return "unknown" // this will be returned if nothing is in the channel
	}
}
func handleTransaction(pack *nt.Package) string {
	tx, err := bc.DeserializeTx(pack.Data)
	fmt.Println(tx)
	if err != nil || tx == nil {
		return "failed to deserialize transaction"
	}

	if Mempool.GetByID(tx.CurrHash) == nil {
		Mempool.Add(tx)
		fmt.Printf("Mempool size: %d\n", Mempool.Size())
	}

	return "ok"
}

// getBlock fetches a block from the blockchain based on a provided package.
func getBlock(pack *nt.Package) string {
	num, err := strconv.Atoi(pack.Data)
	if err != nil {
		fmt.Println("Error converting string to integer: ", err)
		return ""
	}

	if uint64(num) < Chain.Size() {
		return fetchBlockFromChain(Chain, num)
	}

	return ""
}

// fetchBlockFromChain retrieves a block from the chain.
func fetchBlockFromChain(chain *bc.BlockChain, i int) string {
	var block string
	row := chain.DB.QueryRow("SELECT Block FROM BlockChain WHERE Id=$1", i+1)
	row.Scan(&block)
	return block
}

// getLastHash retrieves the last hash from the chain or the ProbablyLhash.
func getLastHash(pack *nt.Package) string {
	var hash []byte

	if ProbablyLhash != nil {
		hash = ProbablyLhash
	} else {
		hash = Chain.LastHash()
	}

	return bc.Base64Encode(hash)
}

// getBalance gets the balance from the chain.
func getBalance(pack *nt.Package) string {
	return fmt.Sprintf("%d", Chain.Balance(pack.Data, Chain.Size()))
}

func selectBlock(chain *bc.BlockChain, i int) string {
	var block string
	row := chain.DB.QueryRow("SELECT Block FROM BlockChain WHERE Id=$1", i+1)
	row.Scan(&block)
	return block
}

// compareChains checks and compares two different chains.
func compareChains(address string, num uint64) {
	tempFile, tempFileName := createTemporaryFile()
	defer os.Remove(tempFile.Name())

	genesis := fetchGenesisBlock(address)
	if genesis == nil {
		return
	}

	chain := initializeBlockchain(tempFile.Name())
	defer chain.DB.Close()

	chain.AddBlock(genesis)

	populateChainWithBlocks(address, num, chain)

	replaceChain(chain, tempFileName)

	if IsMining {
		BreakMining <- true
		IsMining = false
	}
}

// createTemporaryFile creates a temporary file and returns it.
func createTemporaryFile() (*os.File, string) {
	tempFileName := "temp_" + hex.EncodeToString(bc.GenerateRandomBytes(8))
	tempFile, err := os.Create(tempFileName)
	if err != nil {
		log.Fatalf("Failed to create temporary file: %v", err)
	}
	return tempFile, tempFileName
}

// fetchGenesisBlock retrieves the genesis block from a given address.
func fetchGenesisBlock(address string) *bc.Block {
	response := nt.Send(address, &nt.Package{
		Option: GET_BLOCK,
		Data:   "0",
	})

	if response == nil {
		return nil
	}

	return bc.DeserializeBlock(response.Data)
}

// initializeBlockchain creates and returns a new blockchain.
func initializeBlockchain(filename string) *bc.BlockChain {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		log.Fatalf("Failed to open SQLite DB: %v", err)
	}

	_, err = db.Exec(bc.CREATE_TABLE)
	if err != nil {
		log.Fatalf("Failed to create table: %v", err)
	}

	return &bc.BlockChain{DB: db}
}

// populateChainWithBlocks populates a given chain with blocks from a given address up to a given number.
func populateChainWithBlocks(address string, num uint64, chain *bc.BlockChain) {
	for i := uint64(1); i < num; i++ {
		response := nt.Send(address, &nt.Package{
			Option: GET_BLOCK,
			Data:   strconv.FormatUint(i, 10),
		})

		if response == nil {
			return
		}

		block := bc.DeserializeBlock(response.Data)
		if block == nil {
			return
		}

		chain.AddBlock(block)
	}
}

// copyFile copies the contents of one file to another.
func copyFile(src, dst string) error {
	inputFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer inputFile.Close()

	outputFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer outputFile.Close()

	_, err = io.Copy(outputFile, inputFile)
	if err != nil {
		return err
	}
	return outputFile.Close()
}

// addBlock adds a block to the chain.
func addBlock(pack *nt.Package) string {
	splitted := strings.Split(pack.Data, SEPARATOR)
	if len(splitted) != 3 {
		return "fail"
	}

	block := bc.DeserializeBlock(splitted[2])
	if !block.IsBlockValid(Chain, Chain.Size()) {
		currentSize := Chain.Size()
		num, err := strconv.Atoi(splitted[1])
		if err != nil {
			return "fail"
		}
		if currentSize < uint64(num) {
			bc.DIFFICULTY = block.Difficulty
			go compareChains(splitted[0], uint64(num))
			return "ok"
		}
		return "fail"
	}

	Mutex.Lock()
	defer Mutex.Unlock()

	Chain.AddBlock(block)
	Block = bc.NewBlock(User.Address(), Chain.LastHash())

	if IsMining {
		BreakMining <- true
		IsMining = false
	}

	return "ok"
}

// chainNew creates a new blockchain and returns it.
func chainNew(filename string) *bc.BlockChain {
	chain := bc.NewChain(filename, User.Address())
	if chain == nil {
		log.Fatal("Failed to create new chain")
	}
	return bc.LoadChain(filename)
}

// chainLoad loads an existing blockchain and returns it.
func chainLoad(filename string) *bc.BlockChain {
	chain := bc.LoadChain(filename)
	if chain == nil {
		log.Fatal("Failed to load chain")
	}
	return chain
}

// replaceChain replaces the current chain with the provided one.
func replaceChain(chain *bc.BlockChain, tempFile string) {
	Mutex.Lock()
	defer Mutex.Unlock()

	Chain.DB.Close()
	os.Remove(Filename)

	err := copyFile(tempFile, Filename)
	if err != nil {
		log.Fatalf("Failed to copy file: %v", err)
	}
	Chain = bc.LoadChain(Filename)
	Block = bc.NewBlock(User.Address(), Chain.LastHash())
}

// pushBlockToNet pushes the provided block to the network.
func pushBlockToNet(block *bc.Block) {
	serializedBlock := bc.SerializeBlock(block)
	message := fmt.Sprintf("%s%s%d%s%s", Serve, SEPARATOR, Chain.Size(), SEPARATOR, serializedBlock)

	for _, address := range Address {
		go nt.Send(address, &nt.Package{
			Option: ADD_BLOCK,
			Data:   message,
		})
	}
}
