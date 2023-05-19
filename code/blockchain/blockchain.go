package blockchain

import (
	"database/sql"
	"fmt"
	"os"
	"time"

	"math/rand"

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
	DIFFICULTY uint8 = 21
)

const (
	KEY_SIZE       = 512
	DEBUG          = true
	TXS_LIMIT      = 2
	RAND_BYTES     = 32
	START_PERCENT  = 10
	STORAGE_REWARD = 1
)

const (
	GENESIS_BLOCK  = "GENESIS-BLOCK"
	STORAGE_VALUE  = 100
	GENESIS_REWARD = 100
	STORAGE_CHAIN  = "STORAGE-CHAIN"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func NewChain(filename, receiver string) *BlockChain {
	_, err := os.Create(filename)
	if err != nil {
		return nil
	}

	db, err := sql.Open("sqlite3", filename)
	if err != nil {
		return nil
	}

	_, err = db.Exec(CREATE_TABLE)
	if err != nil {
		return nil
	}

	chain := &BlockChain{
		DB: db,
	}

	genesis := &Block{
		CurrHash:  []byte(GENESIS_BLOCK),
		Mapping:   make(map[string]uint64),
		Miner:     receiver,
		TimeStamp: time.Now().Format(time.RFC3339),
	}
	genesis.Mapping[STORAGE_CHAIN] = STORAGE_VALUE
	genesis.Mapping[receiver] = GENESIS_REWARD
	chain.AddBlock(genesis)

	return chain
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
	bhash, err := Base64Decode(hash)
	if err != nil {
		fmt.Println(err)
	}
	return bhash
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

func (chain *BlockChain) AddBlock(block *Block) error {
	chain.index++
	_, err := chain.DB.Exec("INSERT INTO BlockChain (Hash, Block) VALUES (?, ?)",
		Base64Encode(block.CurrHash),
		SerializeBlock(block))

	return err
}
func (chain *BlockChain) HeadBlock() *Block {
	var sblock string
	row := chain.DB.QueryRow("SELECT Block FROM BlockChain ORDER BY Id DESC")
	row.Scan(&sblock)
	return DeserializeBlock(sblock)
}
