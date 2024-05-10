package blockchain

import (
	"crypto/ed25519"
	"database/sql"
	"sync"

	"github.com/noot/ring-go"
)

type BlockChain struct {
	DB    *sql.DB
	index uint64
}

type Block struct {
	CurrHash     []byte
	PrevHash     []byte
	Nonce        uint64
	Difficulty   uint8
	Miner        string
	Signature    []byte
	TimeStamp    string
	Transactions []Transaction
	Mapping      map[string]uint64
}

type Transaction struct {
	RandBytes     []byte
	PrevBlock     []byte
	Sender        string
	Receiver      string
	Value         uint64
	ToStorage     uint64
	CurrHash      []byte
	Signature     []byte
	RingSignature *ring.RingSig
}

type User struct {
	PrivateKey ed25519.PrivateKey
}

type MemPool struct {
	pool map[string]Transaction
	l    sync.RWMutex
}
