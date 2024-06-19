package blockchain

import (
	"bytes"
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/noot/ring-go"
	"golang.org/x/crypto/sha3"
)

func HashToBytes(data []byte) [32]byte {
	var result [32]byte
	copy(result[:], data)
	return result
}

func NewTransaction(user *User, lastHash []byte, to string, value uint64, chain *BlockChain) (*Transaction, error) {
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

	// Подготовка данных для кольцевой подписи
	publicKeys, err := chain.GetLatestPublicKeys(2) // Получаем 3 последних публичных ключа
	if err != nil {
		return nil, err
	}

	// Проверяем, достаточно ли публичных ключей
	if len(publicKeys) < 3 {
		// Генерируем дополнительные ключи
		for i := len(publicKeys); i < 3; i++ {
			pubKey, _, err := GenerateKeyPair()
			if err != nil {
				return nil, err
			}
			publicKeys = append(publicKeys, pubKey)
		}
	}

	// Создание кольцевой подписи
  // ringSignature, err := tx.SignTransaction(ring.Ed25519(), publicKeys)
	// if err != nil {
	//	return nil, err
	// }
	//tx.RingSignature = &ringSignature

	return tx, nil
}

func ConvertPublicKeysToString(keys []ed25519.PublicKey) string {
	var sb strings.Builder

	for _, key := range keys {
		// Конвертируем каждый ключ в hex-строку
		encodedKey := hex.EncodeToString(key)
		// Добавляем ключ в строковый буфер, разделяя ключи пробелом для удобства
		if sb.Len() > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(encodedKey)
	}

	return sb.String()
}

func (tx *Transaction) SignTransaction(curve ring.Curve, msg []ed25519.PublicKey) (ring.RingSig, error) {

	privkey := curve.NewRandomScalar()
	msgHash := sha3.Sum256([]byte(ConvertPublicKeysToString(msg)))

	// size of the public key ring (anonymity set)
	const size = 2
	const idx = 0

	keyring, err := ring.NewKeyRing(curve, size, privkey, idx)
	if err != nil {
		panic(err)
	}

	sig, err := keyring.Sign(msgHash, privkey)
	if err != nil {
		panic(err)
	}
	return *sig, nil
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

func (tx *Transaction) Sign(priv ed25519.PrivateKey) []byte {
	if priv == nil {
		return nil
	}
	sign, err := Sign(priv, tx.CurrHash)
	if err != nil {
		return nil
	}
	return sign
}

func (tx *Transaction) IsValid(chain BlockChain) bool {
	if !bytes.Equal(tx.Hash(), tx.CurrHash) {
		return false
	}

	pubkey, err := ParsePublic(tx.Sender)
	if err != nil {
		return false
	}

	if Verify(pubkey, tx.CurrHash, tx.Signature); err != nil {
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
