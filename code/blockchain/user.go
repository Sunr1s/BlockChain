package blockchain

import (
	"crypto/ed25519"
	"crypto/rand"
)

// NewUser создаёт нового пользователя с парой ключей ed25519.
func NewUser() *User {
	_, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil
	}
	return &User{
		PrivateKey: privKey,
	}
}

// LoadUser загружает пользователя из сериализованного приватного ключа.
func LoadUser(purse string) *User {
	priv, err := ParsePrivate(purse)
	if err != nil {
		return nil
	}
	return &User{
		PrivateKey: priv,
	}
}

// Purse возвращает приватный ключ пользователя в сериализованной форме.
func (user *User) Purse() string {
	privkey, err := StringPrivate(user.PrivateKey)
	if err != nil {
		return ""
	}
	return privkey
}

// Address возвращает публичный ключ пользователя в сериализованной форме.
func (user *User) Address() string {
	pub := user.Public()

	return StringPublic(pub)
}

// Address возвращает публичный ключ пользователя в сериализованной форме.
func (user *User) ViewAddress() string {
	// Генерируем адрес на основе публичного ключа пользователя
	address := PublicKeyToAddress(user.Public())
	return address
}

// Private возвращает приватный ключ пользователя.
func (user *User) Private() ed25519.PrivateKey {
	return user.PrivateKey
}

// Public возвращает публичный ключ пользователя.
func (user *User) Public() ed25519.PublicKey {
	return user.PrivateKey.Public().(ed25519.PublicKey)
}
