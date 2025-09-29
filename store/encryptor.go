package store

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
)

type Encryptor struct {
	gcm cipher.AEAD
}

// NewEncryptor takes a 32-byte key (AES-256)
func NewEncryptor(key []byte) (*Encryptor, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	return &Encryptor{gcm}, nil
}

func (e *Encryptor) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := e.gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawStdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertextB64 string) (string, error) {
	ciphertext, err := base64.RawStdEncoding.DecodeString(ciphertextB64)
	if err != nil {
		return "", err
	}
	if len(ciphertext) < e.gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce := ciphertext[:e.gcm.NonceSize()]
	encrypted := ciphertext[e.gcm.NonceSize():]

	plaintext, err := e.gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}
