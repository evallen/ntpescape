package common

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/hex"
	"errors"
	"fmt"
)

// DUMMY VALUE! Set at build time with
// -ldflags="-X 'github.com/evallen/ntpescape/common.KeyString=<put_your_key_here>'"
const KeyString = "00112233445566778899AABBCCDDEEFF"
const KeyLen = 16

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

// Get the KeyLen-byte key from the KeyString.
// We need to use a string so it can be set at compile time.
func GetKey() ([KeyLen]byte, error) {
	keySlice, err := hex.DecodeString(KeyString)
	if err != nil {
		return [KeyLen]byte{}, fmt.Errorf("could not get key from hex string %s: %v", KeyString, err)
	}

	if len(keySlice) != KeyLen {
		return [KeyLen]byte{}, fmt.Errorf("hex string %s has wrong length %d; need %d", 
			KeyString, len(keySlice), KeyLen)
	}

	keyArray := [16]byte{}
	copy(keyArray[:], keySlice)

	return keyArray, nil
}