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
	"math"
	"math/big"
	mrand "math/rand"
	"time"
)

func ProofOfWork(blockHash []byte, difficulty uint8, ch chan bool) (uint64, float64) {
	var (
		Target  = big.NewInt(1)
		intHash = big.NewInt(1)
		nonce   = uint64(mrand.Intn(math.MaxUint32))
		hash    []byte
		count   float64
	)
	Target.Lsh(Target, 256-uint(difficulty))
	start := time.Now()
	for nonce < math.MaxUint64 {
		count++
		select {
		case <-ch:
			if DEBUG {
				fmt.Println()
				fmt.Printf("time: %v: hash: %v\n", time.Since(start).Seconds(), count)
			}
			return nonce, time.Since(start).Seconds()
		default:
			hash = HashSum(bytes.Join(
				[][]byte{
					blockHash,
					ToBytes(nonce),
				},
				[]byte{},
			))
			if DEBUG {
				fmt.Printf("\rMining: %s", Base64Encode(hash))
			}
			intHash.SetBytes(hash)
			if intHash.Cmp(Target) == -1 {
				if DEBUG {
					fmt.Println()
				}
				fmt.Printf("%v: %v\n", time.Since(start).Seconds(), count)
				return nonce, time.Since(start).Seconds()
			}
			nonce++
		}
	}
	return nonce, time.Since(start).Seconds()
}

func Verify(pub *rsa.PublicKey, data, sign []byte) error {
	return rsa.VerifyPSS(pub, crypto.SHA256, data, sign, nil)
}

func ParsePublic(pubData string) *rsa.PublicKey {
	pub, err := x509.ParsePKCS1PublicKey([]byte(Base64Decode(pubData)))
	if err != nil {
		return nil
	}
	return pub
}

func ParsePrivate(privData string) *rsa.PrivateKey {
	priv, err := x509.ParsePKCS1PrivateKey(Base64Decode(privData))
	if err != nil {
		return nil
	}
	return priv
}

func Base64Decode(data string) []byte {
	result, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil
	}
	return result
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func GeneratePrivate(bits uint) *rsa.PrivateKey {
	priv, err := rsa.GenerateKey(rand.Reader, int(bits))
	if err != nil {
		return nil
	}
	return priv
}

func StringPrivate(priv *rsa.PrivateKey) string {
	return Base64Encode(x509.MarshalPKCS1PrivateKey(priv))
}

func Sign(priv *rsa.PrivateKey, data []byte) []byte {
	signdata, err := rsa.SignPSS(rand.Reader, priv, crypto.SHA256, data, nil)
	if err != nil {
		return nil
	}
	return signdata
}

func GenerateRandomBytes(max uint) []byte {
	var slice = make([]byte, max)
	_, err := rand.Read(slice)
	if err != nil {
		return nil
	}
	return slice
}

func HashSum(data []byte) []byte {
	hash := sha256.Sum256(data)
	return hash[:]
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
