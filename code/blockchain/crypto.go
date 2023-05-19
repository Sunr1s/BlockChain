package blockchain

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/binary"
	"fmt"
)

func Verify(pub *rsa.PublicKey, data, sign []byte) error {
	return rsa.VerifyPSS(pub, crypto.SHA256, data, sign, nil)
}

func ParsePublic(pubData string) (*rsa.PublicKey, error) {
	decodedData, decodeErr := Base64Decode(pubData)
	if decodeErr != nil {
		return nil, fmt.Errorf("unable to base64 decode public key: %w", decodeErr)
	}
	pub, parseErr := x509.ParsePKCS1PublicKey(decodedData)
	if parseErr != nil {
		return nil, fmt.Errorf("unable to parse public key: %w", parseErr)
	}
	return pub, nil
}

func ParsePrivate(privData string) (*rsa.PrivateKey, error) {
	decodedData, decodeErr := Base64Decode(privData)
	if decodeErr != nil {
		return nil, fmt.Errorf("unable to base64 decode private key: %w", decodeErr)
	}
	priv, parseErr := x509.ParsePKCS1PrivateKey(decodedData)
	if parseErr != nil {
		return nil, fmt.Errorf("unable to parse private key: %w", parseErr)
	}
	return priv, nil
}

func Base64Decode(data string) ([]byte, error) {
	result, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("unable to base64 decode: %w", err)
	}
	return result, nil
}

func GeneratePrivate(bits uint) (*rsa.PrivateKey, error) {
	priv, err := rsa.GenerateKey(rand.Reader, int(bits))
	if err != nil {
		return nil, fmt.Errorf("unable to generate private key: %w", err)
	}
	return priv, nil
}

func Sign(priv *rsa.PrivateKey, data []byte) ([]byte, error) {
	signdata, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, data, nil)
	if err != nil {
		return nil, fmt.Errorf("unable to sign data: %w", err)
	}
	return signdata, nil
}

func GenerateRandomBytes(max uint) []byte {
	var slice = make([]byte, max)
	_, err := rand.Read(slice)
	if err != nil {
		return nil
	}
	return slice
}

func ToBytes(num uint64) []byte {
	var data = new(bytes.Buffer)
	err := binary.Write(data, binary.BigEndian, num)
	if err != nil {
		return nil
	}
	return data.Bytes()
}

func StringPublic(pub *rsa.PublicKey) string {
	return Base64Encode(x509.MarshalPKCS1PublicKey(pub))
}

func StringPrivate(priv *rsa.PrivateKey) (string, error) {
	privBytes := x509.MarshalPKCS1PrivateKey(priv)
	return Base64Encode(privBytes), nil
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func HashSum(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
}
