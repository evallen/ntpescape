package common

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
)

// Encrypt a plaintext using AES CTR-mode encryption. 
// The `key` and `nonce` must each be 16 bytes.
func Encrypt(plaintext []byte, nonce []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("Couldn't make cipher from key: " + err.Error())
	}

	ctr := cipher.NewCTR(block, nonce)

	ciphertext := make([]byte, len(plaintext))
	ctr.XORKeyStream(ciphertext, plaintext)

	return ciphertext, nil
}

// Decrypt a ciphertext using AES CTR-mode encryption. 
// The `key` and `nonce` must each be 16 bytes.
func Decrypt(ciphertext []byte, nonce []byte, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.New("Couldn't make cipher from key: " + err.Error())
	}

	ctr := cipher.NewCTR(block, nonce)

	plaintext := make([]byte, len(ciphertext))
	ctr.XORKeyStream(plaintext, ciphertext)

	return plaintext, nil
}