package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"testing"

	"github.com/conformal/btcwallet/snacl"
)

func TestInitialKeyDerivation(t *testing.T) {
}

func TestRegularSetup(t *testing.T) {
}

func TestUpdateKeyMap(t *testing.T) {
}

func TestLoadDbKeys(t *testing.T) {
}

func TestStoreEncryptedKeys(t *testing.T) {
}

func TestInverseHashTreeKeyDerivation(t *testing.T) {
	var parentKey [keySize]byte
	if _, err := rand.Read(parentKey[:]); err != nil {
		t.Fatalf("Unable to read random number: %v", err)
	}

	numChildren := 2
	derivedChildren1, err := inverseHashTreeKeyDerivation(parentKey, numChildren)
	if err != nil {
		t.Fatalf("Unable to perform inverse hash tree derivation: %v", err)
	}
	derivedChildren2, err := inverseHashTreeKeyDerivation(parentKey, numChildren)
	if err != nil {
		t.Fatalf("Unable to perform inverse hash tree derivation: %v", err)
	}

	// Should produce the correct number of child keys.
	if len(derivedChildren1) != numChildren || len(derivedChildren2) != numChildren {
		t.Fatalf("Derivation produced incorrect number of children: ",
			"got %v need %v", len(derivedChildren1), numChildren)
	}

	// Ensure that we get the same child nodes.
	for i := 0; i < numChildren; i++ {
		if !bytes.Equal(derivedChildren1[i][:], derivedChildren2[i][:]) {
			t.Fatalf("Derivation is not deterministic %v vs %v",
				derivedChildren1[i], derivedChildren2[i])
		}
	}

	fmt.Println(derivedChildren1, derivedChildren2)
}

func TestDeriveChildren(t *testing.T) {
	var parentKey [keySize]byte
	if _, err := rand.Read(parentKey[:]); err != nil {
		t.Fatalf("Unable to read random number: %v", err)
	}

	children := deriveChildren(parentKey)
	fmt.Println(children)
}

func TestEncryptChildKeys(t *testing.T) {
	var mKey snacl.CryptoKey
	if _, err := rand.Read(mKey[:]); err != nil {
		t.Fatalf("Unable to read random number: %v", err)
	}

	// TODO(roasbeef): Have marshalled key in file?
	masterKey := snacl.SecretKey{Key: &mKey}

	// Read some random data for each child key.
	childKeys := make([][keySize]byte, 6)
	for _, k := range childKeys {
		if _, err := rand.Read(k[:]); err != nil {
			t.Fatalf("Unable to read random number: %v", err)
		}
	}

	// Encrypt our child keys.
	encryptedChildKeys, err := encryptChildKeys(&masterKey, childKeys)
	if err != nil {
		t.Fatalf("Unable to encrypt child keys: %v", err)
	}

	// Ensure that we can properly decrypt them.
	for i, ec := range encryptedChildKeys {
		regKey, err := masterKey.Decrypt(ec)
		if err != nil {
			t.Fatalf("Unable to decrypt child key: %v", err)
		}

		if !bytes.Equal(regKey[:], childKeys[i][:]) {
			t.Fatalf("Child key was not properly decrypted got %v need %v",
				regKey, childKeys[i])
		}
	}
}

func TestRequestHandler(t *testing.T) {
}
