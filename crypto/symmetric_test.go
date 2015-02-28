package crypto_test

import (
	"bytes"
	"testing"

	"github.com/roasbeef/DuckDuckXor/crypto"
)

func TestCipherCorrectness(t *testing.T) {
	key := []byte("hellohellohelloo")
	originalText := []byte("testing")

	cipherText, err := crypto.AesEncrypt(key, originalText)
	if err != nil {
		t.Fatalf("Unable to encrypt plaintext: %v", err)
	}

	plainText, err := crypto.AesDecrypt(key, cipherText)
	if err != nil {
		t.Fatalf("Unable to decrypt ciphertext: %v", err)
	}

	if bytes.Compare(originalText, plainText) != 0 {
		t.Fatalf("Decrypted plaintext doesn't match original %v vs %v",
			originalText, plainText)
	}
}
