package blockchain

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"fmt"
)

func NewTransaction(user *User, lastHash []byte, to string, value uint64) (*Transaction, error) {
	if user == nil {
		return nil, fmt.Errorf("invalid user")
	}

	tx := &Transaction{
		RandBytes: GenerateRandomBytes(RAND_BYTES),
		PrevBlock: lastHash,
		Sender:    user.Address(),
		Receiver:  to,
		Value:     value,
	}
	if value > START_PERCENT {
		tx.ToStorage = STORAGE_REWARD
	}
	tx.CurrHash = tx.Hash()
	tx.Signature = tx.Sign(user.Private())
	return tx, nil
}

func (tx *Transaction) Hash() []byte {
	return HashSum(bytes.Join(
		[][]byte{
			tx.RandBytes,
			tx.PrevBlock,
			[]byte(tx.Sender),
			[]byte(tx.Receiver),
			ToBytes(tx.Value),
			ToBytes(tx.ToStorage),
		},
		[]byte{},
	))
}

func (tx *Transaction) Sign(priv *rsa.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	sign, err := Sign(priv, tx.CurrHash)
	if err != nil {
		return nil
	}
	return sign
}

func (tx *Transaction) IsValid() bool {
	if !bytes.Equal(tx.Hash(), tx.CurrHash) {
		return false
	}

	pubkey, err := ParsePublic(tx.Sender)
	if err != nil {
		return false
	}

	if err := Verify(pubkey, tx.CurrHash, tx.Signature); err != nil {
		return false
	}

	return true
}

func SerializeTx(tx *Transaction) (string, error) {
	data, err := json.MarshalIndent(*tx, "", "\t")
	if err != nil {
		return "", fmt.Errorf("serialization failed: %w", err)
	}
	return string(data), nil
}

func DeserializeTx(data string) (*Transaction, error) {
	var tx Transaction
	err := json.Unmarshal([]byte(data), &tx)
	if err != nil {
		return nil, fmt.Errorf("deserialization failed: %w", err)
	}
	return &tx, nil
}
