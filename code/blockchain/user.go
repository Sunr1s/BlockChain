package blockchain

import "crypto/rsa"

func NewUser() *User {
	key, err := GeneratePrivate(KEY_SIZE)
	if err == nil {
		return &User{
			PrivateKey: key,
		}
	}
	return &User{
		PrivateKey: key,
	}
}

func LoadUser(purse string) *User {
	priv, err := ParsePrivate(purse)
	if err != nil {
		return nil
	}
	if priv == nil {
		return nil
	}
	return &User{
		PrivateKey: priv,
	}
}

func (user *User) Purse() string {
	privkey, err := StringPrivate(user.Private())
	if err != nil {
		return ""
	}
	return privkey
}

func (user *User) Address() string {
	return StringPublic(user.Public())
}

func (user *User) Private() *rsa.PrivateKey {
	return user.PrivateKey
}

func (user *User) Public() *rsa.PublicKey {
	return &(user.PrivateKey).PublicKey
}
