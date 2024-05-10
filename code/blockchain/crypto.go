package blockchain

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"fmt"

	"github.com/cosmos/btcutil/base58"
	"golang.org/x/crypto/ripemd160"
)

// Verify использует открытый ключ для проверки подписи данных
func Verify(pub ed25519.PublicKey, data, sig []byte) bool {
	return ed25519.Verify(pub, data, sig)
}

// ParsePublic преобразует строку с публичным ключом из Base64 в ed25519.PublicKey
func ParsePublic(pubData string) (ed25519.PublicKey, error) {
	decodedData, err := Base64Decode(pubData)
	if err != nil {
		return nil, fmt.Errorf("unable to base64 decode public key: %w", err)
	}
	if len(decodedData) != ed25519.PublicKeySize {
		return nil, errors.New("invalid public key length")
	}
	return ed25519.PublicKey(decodedData), nil
}

// ParsePrivate преобразует строку с приватным ключом из Base64 в ed25519.PrivateKey
func ParsePrivate(privData string) (ed25519.PrivateKey, error) {
	decodedData, err := Base64Decode(privData)
	if err != nil {
		return nil, fmt.Errorf("unable to base64 decode private key: %w", err)
	}
	if len(decodedData) != ed25519.PrivateKeySize {
		return nil, errors.New("invalid private key length")
	}
	return ed25519.PrivateKey(decodedData), nil
}

// Base64Decode декодирует данные из строки Base64
func Base64Decode(data string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("unable to base64 decode: %w", err)
	}
	return result, nil
}

// GenerateKeyPair генерирует пару ключей ed25519
func GenerateKeyPair() (ed25519.PublicKey, ed25519.PrivateKey, error) {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to generate key pair: %w", err)
	}
	return pubKey, privKey, nil
}

// Sign создает подпись для данных, используя приватный ключ
func Sign(priv ed25519.PrivateKey, data []byte) ([]byte, error) {
	if priv == nil {
		return nil, fmt.Errorf("private key is nil")
	}
	signature := ed25519.Sign(priv, data) // Разыменовываем указатель здесь
	return signature, nil
}

// GenerateRandomBytes генерирует случайный массив байтов заданной длины
func GenerateRandomBytes(max uint) []byte {
	slice := make([]byte, max)
	_, err := rand.Read(slice)
	if err != nil {
		return nil
	}
	return slice
}

// ToBytes преобразует число uint64 в массив байтов
func ToBytes(num uint64) []byte {
	buf := make([]byte, binary.Size(num))
	binary.BigEndian.PutUint64(buf, num)
	return buf
}

// StringPublic преобразует публичный ключ в строку Base64
func StringPublic(pub ed25519.PublicKey) string {
	return Base64Encode(pub)
}

// StringPrivate преобразует приватный ключ в строку Base64
func StringPrivate(priv ed25519.PrivateKey) (string, error) {
	return Base64Encode(priv), nil
}

// Base64Encode кодирует данные в строку Base64
func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// HashSum вычисляет хеш SHA-256 для данных
func HashSum(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}

func PublicKeyToAddress(pubKey ed25519.PublicKey) string {
	// Шаг 1: SHA-256 хеширование публичного ключа
	sha256Hash := sha256.New()
	sha256Hash.Write(pubKey)
	sha256Result := sha256Hash.Sum(nil)

	// Шаг 2: RIPEMD-160 хеширование результата SHA-256
	ripemd160Hash := ripemd160.New()
	ripemd160Hash.Write(sha256Result)
	ripemdResult := ripemd160Hash.Sum(nil)

	// Добавление байта версии (например, 0x00 для Bitcoin)
	versionedPayload := append([]byte{0x00}, ripemdResult...)

	// Шаг 3: Двойное SHA-256 хеширование и получение контрольной суммы
	firstSha256 := sha256.Sum256(versionedPayload)
	checksumFull := sha256.Sum256(firstSha256[:])
	checksum := checksumFull[:4]

	// Шаг 4: Добавление контрольной суммы к payload
	finalPayload := append(versionedPayload, checksum...)

	// Шаг 5: Кодирование в Base58Check
	address := base58.CheckEncode(finalPayload, 0x00)

	return address
}
