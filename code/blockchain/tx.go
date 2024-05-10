package blockchain

import (
	"bytes"
	"crypto/ed25519"
	"encoding/json"
	"fmt"

	"github.com/athanorlabs/go-dleq/types"
	"github.com/noot/ring-go"
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

	secretIndex := 0 // Индекс секретного ключа пользователя

	// Создание кольцевой подписи
	ringSignature, err := tx.SignTransaction(tx.CurrHash, user.Private(), publicKeys, secretIndex)
	if err != nil {
		return nil, err
	}
	tx.RingSignature = ringSignature

	return tx, nil
}

func (tx *Transaction) SignTransaction(message []byte, secretKey ed25519.PrivateKey, publicKeys []ed25519.PublicKey, secretIndex int) (*ring.RingSig, error) {
	curve := ring.Ed25519()

	// Преобразование publicKeys в []types.Point
	pubKeys := make([]types.Point, len(publicKeys))
	for i, pubKey := range publicKeys {
		point, err := curve.DecodeToPoint(pubKey)
		if err != nil {
			return nil, err
		}
		pubKeys[i] = point
	}

	fmt.Println([32]byte(secretKey.Seed()))
	privkey := curve.NewRandomScalar()
	fmt.Println(privkey)
	// Преобразование secretKey в types.Scalar
	privKey := curve.ScalarFromBytes([32]byte(secretKey.Seed()))

	// Создание кольца из публичных ключей
	r, err := ring.NewKeyRingFromPublicKeys(curve, pubKeys, privKey, secretIndex)
	if err != nil {
		return nil, err
	}

	// Создание кольцевой подписи
	signature, err := r.Sign(HashToBytes(message), privKey)
	if err != nil {
		return nil, err
	}

	return signature, nil
}

func (tx *Transaction) VerifyTransaction(message []byte, ringSignature *ring.RingSig) bool {
	// Проверка подписи
	return ringSignature.Verify(HashToBytes(message))
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
	// Проверка кольцевой подписи
	if !tx.VerifyTransaction(tx.CurrHash, tx.RingSignature) {
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
