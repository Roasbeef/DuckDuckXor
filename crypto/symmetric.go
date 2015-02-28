package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

// TODO(roasbeef): Re-use buffers?

// AesEncrypt encrypts the passed plain text using AES in CTR mode with the
// given key.
func AesEncrypt(key []byte, plainText []byte) ([]byte, error) {
	// Create a new cipher block.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a buffer for our cipher text, leaving room for our random IV.
	cipherText := make([]byte, aes.BlockSize+len(plainText))
	iv := cipherText[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	// Encrypt our plain text using AES in CTR mode.
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(cipherText[aes.BlockSize:], plainText)

	return cipherText, nil
}

// AesDecrypt decrypts the passed cipher text using AES in CTR mode with the
// given key.
func AesDecrypt(key []byte, cipherText []byte) ([]byte, error) {
	// Create a new cipher block.
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	plainText := make([]byte, len(cipherText)-aes.BlockSize)

	iv := cipherText[:aes.BlockSize]
	stream := cipher.NewCTR(block, iv)
	stream.XORKeyStream(plainText, cipherText[aes.BlockSize:])

	return plainText, nil
}
