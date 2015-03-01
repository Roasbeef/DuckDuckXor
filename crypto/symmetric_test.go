package crypto_test

import (
	"bytes"
	"crypto/aes"
	"testing"

	"github.com/roasbeef/DuckDuckXor/crypto"
)

func TestCipherCorrectness(t *testing.T) {
	key := []byte("hellohellohelloo")
	originalText := []byte("testing")

	cipher, err := aes.NewCipher(key)
	if err != nil {
		t.Fatalf("Unable to create cipher: %v", err)
	}

	cipherText, err := crypto.AesEncrypt(cipher, originalText)
	if err != nil {
		t.Fatalf("Unable to encrypt plaintext: %v", err)
	}

	plainText, err := crypto.AesDecrypt(cipher, cipherText)
	if err != nil {
		t.Fatalf("Unable to decrypt ciphertext: %v", err)
	}

	if bytes.Compare(originalText, plainText) != 0 {
		t.Fatalf("Decrypted plaintext doesn't match original %v vs %v",
			originalText, plainText)
	}
}
