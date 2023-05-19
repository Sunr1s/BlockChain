package blockchain

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"time"

	nt "github.com/Sunr1s/chain/network"
)

const timeThreshold = 1 * time.Second // duration to wait for

// Function to create a new block
func NewBlock(miner string, prevHash []byte) *Block {
	return &Block{
		Difficulty: DIFFICULTY,
		PrevHash:   prevHash,
		Miner:      miner,
		Mapping:    make(map[string]uint64),
	}
}

// Function to add transaction to the block
func (block *Block) AddTransaction(chain *BlockChain, tx *Transaction) error {
	if tx == nil {
		return errors.New("transaction is null")
	}
	if tx.Value == 0 {
		return errors.New("transaction value is zero")
	}
	if tx.Sender != STORAGE_CHAIN && len(block.Transactions) == TXS_LIMIT {
		return errors.New("transaction limit reached")
	}
	if tx.Sender != STORAGE_CHAIN && tx.Value > START_PERCENT && tx.ToStorage != STORAGE_REWARD {
		return errors.New("storage reward exceeded")
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
	block.IncrementBalance(chain, tx.Receiver, tx.Value)
	block.IncrementBalance(chain, STORAGE_CHAIN, tx.ToStorage)
	block.Transactions = append(block.Transactions, *tx)
	return nil
}

func (block *Block) Accept(chain *BlockChain, user *User) error {
	if !block.AreTransactionsValid(chain, chain.Size()) {
		return errors.New("transaction is not valid")
	}
	block.AddTransaction(chain, &Transaction{
		RandBytes: GenerateRandomBytes(RAND_BYTES),
		Sender:    STORAGE_CHAIN,
		Receiver:  user.Address(),
		Value:     STORAGE_REWARD,
	})
	block.TimeStamp = time.Now().Format(time.RFC3339)
	block.CurrHash = block.hash()
	block.Signature = block.sign(user.PrivateKey)
	// block.Nonce, DIFFICULTY = block.proof(ch)
	// block.Difficulty = DIFFICULTY
	return nil
}

// AreTransactionsValid checks if all transactions in the block are valid.
func (block *Block) AreTransactionsValid(chain *BlockChain, size uint64) bool {
	numTxs := len(block.Transactions)
	storageTransactionExists := false

	for _, tx := range block.Transactions {
		if tx.Sender == STORAGE_CHAIN {
			storageTransactionExists = true
			break
		}
	}

	if numTxs == 0 || numTxs > TXS_LIMIT+btoi(storageTransactionExists) {
		return false
	}

	for i := 0; i < numTxs-1; i++ {
		for j := i + 1; j < numTxs; j++ {
			if bytes.Equal(block.Transactions[i].RandBytes, block.Transactions[j].RandBytes) ||
				(block.Transactions[i].Sender == STORAGE_CHAIN && block.Transactions[j].Sender == STORAGE_CHAIN) {
				return false
			}
		}
	}

	for _, tx := range block.Transactions {
		if !tx.IsValid() || !block.IsBalanceValid(chain, tx.Sender, size) || !block.IsBalanceValid(chain, tx.Receiver, size) {
			return false
		}
	}

	return true
}

// CalculateHash computes the block's hash based on its contents.
func (block *Block) hash() []byte {
	var tempHash []byte

	for _, tx := range block.Transactions {
		tempHash = HashSum(bytes.Join(
			[][]byte{tempHash, tx.CurrHash},
			[]byte{},
		))
	}

	hashes := make([]string, 0, len(block.Mapping))
	for hash := range block.Mapping {
		hashes = append(hashes, hash)
	}
	sort.Strings(hashes)

	for _, hash := range hashes {
		tempHash = HashSum(bytes.Join(
			[][]byte{tempHash, []byte(hash), ToBytes(block.Mapping[hash])},
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

// SignBlock signs the block with the given private key.
func (block *Block) sign(priv *rsa.PrivateKey) []byte {
	signature, err := Sign(priv, block.CurrHash)
	if err != nil {
		return nil
	}
	return signature
}

// Helper function to convert bool to int
func btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

// StartMining starts the mining process on a block, using a hybrid PoET/PoW model.
// It sends a wake-up message to all other nodes and decides whether PoW is needed.
func (block *Block) StartMining(chain *BlockChain, stopChan, sleepModeChan chan bool, addresses []string) error {
	// Start PoET process and assign resulting nonce to block
	err, nonce := block.PoET(stopChan, sleepModeChan)
	if err != nil {
		return fmt.Errorf("error running PoET: %w", err)
	}
	block.Nonce = nonce

	// Wake up other nodes and decide whether PoW is needed
	isMiningNeeded := nt.SendWakeUpMsgToAll(addresses)

	if isMiningNeeded {
		// Start PoW process
		nonce, _ := ProofOfWork(block.CurrHash, block.Difficulty, stopChan)
		if err != nil {
			return fmt.Errorf("error running PoW: %w", err)
		}
		block.Nonce = nonce
	}

	return nil
}

func (block *Block) Mining(chain *BlockChain, ch chan bool, Addresses []string, sleepMode chan bool) error {
	return block.StartMining(chain, ch, sleepMode, Addresses)
}

// GetBlockFromChain fetches a specific block from the blockchain based on the given index.
func GetBlockFromChain(chain *BlockChain, index int) (string, error) {
	var block string
	err := chain.DB.QueryRow("SELECT Block FROM BlockChain WHERE Id=$1", index+1).Scan(&block)
	if err != nil {
		return "", fmt.Errorf("error fetching block: %w", err)
	}
	return block, nil
}

// IncrementBalance increments the balance of the specified receiver by the given value.
func (block *Block) IncrementBalance(chain *BlockChain, receiver string, value uint64) {
	var balanceInChain uint64
	if currentBalance, exists := block.Mapping[receiver]; exists {
		balanceInChain = currentBalance
	} else {
		balanceInChain = chain.Balance(receiver, chain.Size())
	}
	block.Mapping[receiver] = balanceInChain + value
}

// IsBlockValid checks whether the block is valid according to the blockchain.
func (block *Block) IsBlockValid(chain *BlockChain, size uint64) bool {
	if block == nil ||
		block.Difficulty != DIFFICULTY ||
		!block.IsHashValid(chain, chain.Size()) ||
		!block.IsSignatureValid() ||
		!block.IsProofValid() ||
		!block.IsMappingValid() ||
		!block.IsTimeValid(chain, chain.Size()) ||
		!block.AreTransactionsValid(chain, chain.Size()) {
		return false
	}
	return true
}

// IsHashValid validates the hash of the block.
func (block *Block) IsHashValid(chain *BlockChain, index uint64) bool {
	calculatedHash := block.hash()

	if !bytes.Equal(calculatedHash, block.CurrHash) {
		fmt.Printf("Hash mismatch. Calculated: %x, Current: %x\n", calculatedHash, block.CurrHash)
		return false
	}

	var id uint64
	err := chain.DB.QueryRow("SELECT Id FROM BlockChain WHERE Hash=$1", Base64Encode(block.PrevHash)).Scan(&id)
	if err != nil {
		fmt.Printf("Error fetching Id from database: %v\n", err)
		return false
	}

	if id != index {
		fmt.Printf("Block index mismatch. Expected: %d, Got: %d\n", index, id)
		return false
	}

	return true
}

// IsSignatureValid validates the block's signature.
func (block *Block) IsSignatureValid() bool {
	pubKey, err := ParsePublic(block.Miner)
	if err != nil {
		return false
	}
	return Verify(pubKey, block.CurrHash, block.Signature) == nil
}

// IsProofValid validates the proof of the block.
func (block *Block) IsProofValid() bool {
	intHash := big.NewInt(1)
	target := big.NewInt(1)
	hash := HashSum(bytes.Join([][]byte{block.CurrHash, ToBytes(block.Nonce)}, []byte{}))

	intHash.SetBytes(hash)
	target.Lsh(target, 256-uint(block.Difficulty))

	return intHash.Cmp(target) == -1
}

// IsMappingValid validates the block's address mapping.
func (block *Block) IsMappingValid() bool {
	for address := range block.Mapping {
		if address == STORAGE_CHAIN {
			continue
		}
		addressExists := false
		for _, tx := range block.Transactions {
			if tx.Sender == address || tx.Receiver == address {
				addressExists = true
				break
			}
		}
		if !addressExists {
			return false
		}
	}
	return true
}

// IsTimeValid validates the block's timestamp.
func (block *Block) IsTimeValid(chain *BlockChain, index uint64) bool {
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

// IsBalanceValid validates the balance of the block.
func (block *Block) IsBalanceValid(chain *BlockChain, address string, size uint64) bool {
	balance, exists := block.Mapping[address]
	if !exists {
		return false
	}

	balanceInChain := chain.Balance(address, size)
	var balanceAddBlock, balanceSubBlock uint64

	for _, tx := range block.Transactions {
		if tx.Sender == address {
			balanceSubBlock += tx.Value + tx.ToStorage
		}
		if tx.Receiver == address {
			balanceAddBlock += tx.Value
		}
		if STORAGE_CHAIN == address {
			balanceAddBlock += tx.ToStorage
		}
	}

	return (balanceInChain + balanceAddBlock - balanceSubBlock) == balance
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
