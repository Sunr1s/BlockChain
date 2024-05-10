package main

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"

	bc "github.com/Sunr1s/chain/blockchain"
)

type Wallet struct {
	Public  string
	Private string
}

var (
	Address []string
	User    *bc.User
)

const (
	SEPARATOR = "_SEPARATOR_"
)

const (
	ADD_BLOCK = iota + 1
	ADD_TRNSX
	GET_BLOCK
	GET_LHASH
	GET_BLNCE
	WAKEUP_MSG
)

func encode(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey) (string, string) {
	x509Encoded, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		panic(err)
	}
	pemEncoded := pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: x509Encoded})

	x509EncodedPub, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		panic(err)
	}
	pemEncodedPub := pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: x509EncodedPub})

	return string(pemEncoded), string(pemEncodedPub)
}

func decode(pemEncoded string, pemEncodedPub string) (ed25519.PrivateKey, ed25519.PublicKey) {
	block, _ := pem.Decode([]byte(pemEncoded))
	x509Encoded := block.Bytes
	privateKey, err := x509.ParsePKCS8PrivateKey(x509Encoded)
	if err != nil {
		fmt.Println(err)
	}

	blockPub, _ := pem.Decode([]byte(pemEncodedPub))
	x509EncodedPub := blockPub.Bytes
	genericPublicKey, err := x509.ParsePKIXPublicKey(x509EncodedPub)
	if err != nil {
		fmt.Println(err)
	}

	return privateKey.(ed25519.PrivateKey), genericPublicKey.(ed25519.PublicKey)
}

func userNew(filename string) *bc.User {
	user := bc.NewUser()
	if user == nil {
		return nil
	}
	err := writeFile(filename, user.Private(), user.Public())
	if err != nil {
		return nil
	}
	return user
}

func userLoad(filename string) *bc.User {
	priv, err := readKeys(filename, true)
	if err != nil {
		return nil
	}
	if priv == "" {
		return nil
	}
	user := bc.LoadUser(priv)
	if user == nil {
		return nil
	}
	return user
}

func writeFile(foldername string, priv ed25519.PrivateKey, pub ed25519.PublicKey) error {
	encPriv, encPub := encode(priv, pub)

	kdata := Wallet{
		Public:  encPub,
		Private: encPriv,
	}

	file, _ := json.MarshalIndent(kdata, "", " ")
	err := os.Mkdir(foldername, 0750)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return ioutil.WriteFile(foldername+"/wallet.dat", file, 0644)
}

func readFile(filename string) string {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return ""
	}
	return string(data)
}

func readKeys(filename string, key bool) (string, error) {
	data, err := os.ReadFile(filename + "/wallet.dat")
	if err != nil {
		return "", err
	}

	var keys Wallet
	err = json.Unmarshal(data, &keys)
	if err != nil {
		return "", err
	}

	priv, pub := decode(string(keys.Private), string(keys.Public))
	if err != nil {
		return "", err
	}

	if key {
		privkey, _ := bc.StringPrivate(priv)

		return privkey, nil
	} else {
		pubkey := bc.StringPublic(pub)
		if err != nil {
			return "", err
		}
		return pubkey, nil
	}
}
