package blockchain

import (
	"database/sql"
	"os"
	"time"

	mrand "math/rand"

	_ "github.com/mattn/go-sqlite3"
)

const (
	CREATE_TABLE = `
CREATE TABLE BlockChain (
    Id INTEGER PRIMARY KEY AUTOINCREMENT,
    Hash VARCHAR(44) UNIQUE,
    Block TEXT
);
`
)

var (
	hashTime   time.Time
	DIFFICULTY uint8 = 24
)

const (
	KEY_SIZE      = 512
	DEBUG         = false
	TXS_LIMIT     = 2
	RAND_BYTES    = 32
	START_PRECENT = 10
	STORAGE_REWRD = 1
)

const (
	GENESIS_BLOCK  = "GENESIS-BLOCK"
	STORAGE_VALUE  = 100
	GENESIS_REWARD = 100
	STORAGE_CHAIN  = "STORAGE-CHAIN"
)

func init() {
	mrand.Seed(time.Now().UnixNano())
}

func NewChain(filename, reciver string) error {
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	file.Close()
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return err
	}
	defer db.Close()
	_, err = db.Exec(CREATE_TABLE)
	chain := &BlockChain{
		DB: db,
	}
	genesis := &Block{
		CurrHash:  []byte(GENESIS_BLOCK),
		Mapping:   make(map[string]uint64),
		Miner:     reciver,
		TimeStamp: time.Now().Format(time.RFC3339),
	}
	genesis.Mapping[STORAGE_CHAIN] = STORAGE_VALUE
	genesis.Mapping[reciver] = GENESIS_REWARD
	chain.AddBlock(genesis)
	return nil
}

func LoadChain(filename string) *BlockChain {
	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil
	}
	chain := BlockChain{
		DB: db,
	}
	chain.index = chain.Size()
	return &chain
}

func (chain *BlockChain) LastHash() []byte {
	var hash string
	row := chain.DB.QueryRow("SELECT Hash FROM BlockChain ORDER BY Id DESC")
	row.Scan(&hash)
	return Base64Decode(hash)
}

func (chain *BlockChain) Balance(address string, size uint64) uint64 {
	var (
		sblock  string
		block   *Block
		balance uint64
	)
	rows, err := chain.DB.Query("SELECT Block FROM BlockChain WHERE Id <= $1 ORDER BY Id DESC", size)
	if err != nil {
		return balance
	}
	defer rows.Close()
	for rows.Next() {
		rows.Scan(&sblock)
		block = DeserializeBlock(sblock)
		if value, ok := block.Mapping[address]; ok {
			balance = value
			break
		}
	}
	return balance
}

func (chain *BlockChain) Size() uint64 {
	var index uint64
	row := chain.DB.QueryRow("SELECT Id FROM BlockChain ORDER BY Id DESC")
	row.Scan(&index)
	return index
}

func (chain *BlockChain) AddBlock(block *Block) {
	chain.index += 1
	chain.DB.Exec("INSERT INTO BlockChain (Hash, Block) VALUES ($1, $2)",
		Base64Encode(block.CurrHash),
		SerializeBlock(block))
}
