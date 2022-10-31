package blockchain

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"math/big"
	"sort"
	"time"
)

func NewBlock(miner string, prevHash []byte) *Block {
	return &Block{
		Difficulty: DIFFICULTY,
		PrevHash:   prevHash,
		Miner:      miner,
		Mapping:    make(map[string]uint64),
	}
}

func (block *Block) AddTransaction(chain *BlockChain, tx *Transaction) error {
	if tx == nil {
		return errors.New("tx is null")
	}
	if tx.Value == 0 {
		return errors.New("tx value = 0")
	}
	if tx.Sender != STORAGE_CHAIN && len(block.Transactions) == TXS_LIMIT {
		return errors.New("len tx = limit")
	}
	if tx.Sender != STORAGE_CHAIN && tx.Value > START_PRECENT && tx.ToStorage != STORAGE_REWRD {
		return errors.New("storage reward pass")
	}
	if !bytes.Equal(tx.PrevBlock, chain.LastHash()) {
		return errors.New("prev block in tx != last hash in chain")
	}
	var balanceInChain uint64
	balanceInTX := tx.Value + tx.ToStorage
	if value, ok := block.Mapping[tx.Sender]; ok {
		balanceInChain = value
	} else {
		balanceInChain = chain.Balance(tx.Sender, chain.Size())
	}
	if balanceInTX > balanceInChain {
		return errors.New("insufficient funds")
	}
	block.Mapping[tx.Sender] = balanceInChain - balanceInTX
	block.addBalance(chain, tx.Reciver, tx.Value)
	block.addBalance(chain, STORAGE_CHAIN, tx.ToStorage)
	block.Transactions = append(block.Transactions, *tx)
	return nil
}

func (block *Block) Accept(chain *BlockChain, user *User) error {
	if !block.transactionsIsValid(chain, chain.Size()) {
		return errors.New("transaction is not valid")
	}
	block.AddTransaction(chain, &Transaction{
		RandBytes: GenerateRandomBytes(RAND_BYTES),
		Sender:    STORAGE_CHAIN,
		Reciver:   user.Address(),
		Value:     STORAGE_REWRD,
	})
	block.TimeStamp = time.Now().Format(time.RFC3339)
	block.CurrHash = block.hash()
	block.Signature = block.sign(user.PrivateKey)
	// block.Nonce, DIFFICULTY = block.proof(ch)
	// block.Difficulty = DIFFICULTY
	return nil
}

func (block *Block) Mining(ch chan bool) error {
	block.Nonce, DIFFICULTY = block.proof(ch)
	block.Difficulty = DIFFICULTY
	return nil
}

func (block *Block) transactionsIsValid(chain *BlockChain, size uint64) bool {
	lentxs := len(block.Transactions)
	plusStorage := 0
	for i := 0; i < lentxs; i++ {
		if block.Transactions[i].Sender == STORAGE_CHAIN {
			plusStorage = 1
			break
		}
	}
	if lentxs == 0 || lentxs > TXS_LIMIT+plusStorage {
		return false
	}
	for i := 0; i < lentxs-1; i++ {
		for j := i + 1; j < lentxs; j++ {
			if bytes.Equal(block.Transactions[i].RandBytes, block.Transactions[j].RandBytes) {
				return false
			}
			if block.Transactions[i].Sender == STORAGE_CHAIN &&
				block.Transactions[j].Sender == STORAGE_CHAIN {
				return false
			}
		}
	}
	for i := 0; i < lentxs; i++ {
		tx := block.Transactions[i]
		if tx.Sender == STORAGE_CHAIN {
			if tx.Reciver != block.Miner || tx.Value != STORAGE_REWRD {
				return false
			}
		} else {
			if !tx.hashIsValid() {
				return false
			}
			if !tx.signIsValid() {
				return false
			}
		}
		if !block.balanceIsValid(chain, tx.Sender, size) {
			return false
		}
		if !block.balanceIsValid(chain, tx.Reciver, size) {
			return false
		}
	}
	return true
}

func (block *Block) hash() []byte {
	var tempHash []byte
	for _, tx := range block.Transactions {
		tempHash = HashSum(bytes.Join(
			[][]byte{
				tempHash,
				tx.CurrHash,
			},
			[]byte{},
		))
	}
	var list []string
	for hash := range block.Mapping {
		list = append(list, hash)
	}
	sort.Strings(list)
	for _, hash := range list {
		tempHash = HashSum(bytes.Join(
			[][]byte{
				tempHash,
				[]byte(hash),
				ToBytes(block.Mapping[hash]),
			},
			[]byte{},
		))
	}
	return HashSum(bytes.Join(
		[][]byte{
			tempHash,
			ToBytes(uint64(block.Difficulty)),
			block.PrevHash,
			[]byte(block.Miner),
			[]byte(block.TimeStamp),
		},
		[]byte{},
	))
}

func (block *Block) sign(priv *rsa.PrivateKey) []byte {
	return Sign(priv, block.CurrHash)
}

func (block *Block) proof(ch chan bool) (uint64, uint8) {
	nonce, hashTime := ProofOfWork(block.CurrHash, block.Difficulty, ch)
	// DIFFICULTY = uint8(float64(block.Difficulty) / (hashTime / 3))
	hashTime++
	// fmt.Println("slozhnost ", block.Difficulty)
	// fmt.Println("Now ", DIFFICULTY)
	return nonce, DIFFICULTY
}

func (block *Block) addBalance(chain *BlockChain, reciver string, value uint64) {
	var balanceInChain uint64
	if v, ok := block.Mapping[reciver]; ok {
		balanceInChain = v
	} else {
		balanceInChain = chain.Balance(reciver, chain.Size())
	}
	block.Mapping[reciver] = balanceInChain + value
}

func (block *Block) IsValid(chain *BlockChain, size uint64) bool {
	switch {
	case block == nil:
		return false
	case block.Difficulty != DIFFICULTY:
		return false
	case !block.hashIsValid(chain, chain.Size()):
		return false
	case !block.signIsValid():
		return false
	case !block.proofIsValid():
		return false
	case !block.mappingIsValid():
		return false
	case !block.timeIsValid(chain, chain.Size()):
		return false
	case !block.transactionsIsValid(chain, chain.Size()):
		return false
	}
	return true
}

func (block *Block) hashIsValid(chain *BlockChain, index uint64) bool {
	if !bytes.Equal(block.hash(), block.CurrHash) {
		return false
	}
	var id uint64
	row := chain.DB.QueryRow("SELECT Id FROM BlockChain WHERE Hash=$1", Base64Encode(block.PrevHash))
	row.Scan(&id)
	return id == index
}

func (block *Block) signIsValid() bool {
	return Verify(ParsePublic(block.Miner), block.CurrHash, block.Signature) == nil
}

func (block *Block) proofIsValid() bool {
	intHash := big.NewInt(1)
	Target := big.NewInt(1)
	hash := HashSum(bytes.Join(
		[][]byte{
			block.CurrHash,
			ToBytes(block.Nonce),
		},
		[]byte{},
	))
	intHash.SetBytes(hash)
	Target.Lsh(Target, 256-uint(block.Difficulty))
	if intHash.Cmp(Target) == -1 {
		return true
	}
	return false
}

func (block *Block) mappingIsValid() bool {
	for addr := range block.Mapping {
		if addr == STORAGE_CHAIN {
			continue
		}
		flag := false
		for _, tx := range block.Transactions {
			if tx.Sender == addr || tx.Reciver == addr {
				flag = true
				break
			}
		}
		if !flag {
			return false
		}
	}
	return true
}

func (block *Block) timeIsValid(chain *BlockChain, indedx uint64) bool {
	btime, err := time.Parse(time.RFC3339, block.TimeStamp)
	if err != nil {
		return false
	}
	diff := time.Now().Sub(btime)
	if diff < 0 {
		return false
	}
	var sblock string
	row := chain.DB.QueryRow("SELECT Block FROM BlockChain WHERE Hash=$1",
		Base64Encode(block.PrevHash))
	row.Scan(&sblock)
	lblock := DeserializeBlock(sblock)
	if lblock == nil {
		return false
	}
	ltime, err := time.Parse(time.RFC3339, lblock.TimeStamp)
	if err != nil {
		return false
	}
	diff = btime.Sub(ltime)
	return diff > 0
}

func (block *Block) balanceIsValid(chain *BlockChain, address string, size uint64) bool {
	if _, ok := block.Mapping[address]; !ok {
		return false
	}
	lentxs := len(block.Transactions)
	balanceInChain := chain.Balance(address, size)
	balanceSubBlock := uint64(0)
	balanceAddBlock := uint64(0)
	for j := 0; j < lentxs; j++ {
		tx := block.Transactions[j]
		if tx.Sender == address {
			balanceSubBlock += tx.Value + tx.ToStorage
		}
		if tx.Reciver == address {
			balanceAddBlock += tx.Value
		}
		if STORAGE_CHAIN == address {
			balanceAddBlock += tx.ToStorage
		}
	}
	if (balanceInChain + balanceAddBlock - balanceSubBlock) != block.Mapping[address] {
		return false
	}
	return true
}

func DeserializeBlock(data string) *Block {
	var block Block
	err := json.Unmarshal([]byte(data), &block)
	if err != nil {
		return nil
	}
	return &block
}

func SerializeBlock(block *Block) string {
	jsonData, err := json.MarshalIndent(*block, "", "\t")
	if err != nil {
		return ""
	}
	return string(jsonData)
}
