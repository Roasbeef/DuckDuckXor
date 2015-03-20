package main

import (
	"crypto/rand"
	"os"
	"testing"
)

/*
func TestDocUploader(t *testing.T) {
	var fakeCipher1 [60]byte
	if _, err := rand.Read(fakeCipher1[:]); err != nil {
		t.Fatalf("Couldn't read random number for fake doc")
	}
	fakeEncDoc1 := &encryptedDoc{fakeCipher, 1}

	// Generate a key for the encrypter
	var docKey [keySize]byte
	rand.Read(docKey[:])

	// Create a new EncryptedDocStreamer and launch only the docEncrypter
	// goroutine. docStream chen not needed, so pass nil.
	streamChan := make(chan *document)
	encrypter := NewEncryptedDocStreamer(1, &docKey, nil, nil)
}
*/

func TestDocEncrypter(t *testing.T) {
	// Open up our test files.
	testFile1, err := os.Open("test_directory/testFile1.txt")
	if err != nil {
		t.Fatalf("Unable to open test file: %v", testFile1)
	}
	testDoc1 := &document{testFile1, 1}

	testFile2, err := os.Open("test_directory/testFile2.txt")
	if err != nil {
		t.Fatalf("Unable to open test file: %v", testFile2)
	}
	testDoc2 := &document{testFile2, 2}

	// Generate a key for the encrypter
	var docKey [keySize]byte
	rand.Read(docKey[:])

	// Create a new EncryptedDocStreamer and launch only the docEncrypter
	// goroutine.
	streamChan := make(chan *document)
	encrypter := NewEncryptedDocStreamer(1, &docKey, streamChan, nil, nil)
	encrypter.wg.Add(1)
	go encrypter.docEncrypter()

	go func() {
		streamChan <- testDoc1
		streamChan <- testDoc2
	}()

	// Encryption should hold correctness.
	encryptedDoc1 := <-encrypter.encryptedDocs
	_, err = encrypter.docKey.Decrypt(encryptedDoc1.cipherText)
	if err != nil {
		t.Fatalf("Unable to decrypt document1: %v", err)
	}

	// Encrypter should have closed the file.
	if err := testDoc1.Close(); err == nil {
		t.Fatalf("testFile1 should be closed")
	}

	// Encryption should hold correctness.
	encryptedDoc2 := <-encrypter.encryptedDocs
	_, err = encrypter.docKey.Decrypt(encryptedDoc2.cipherText)
	if err != nil {
		t.Fatalf("Unable to decrypt document2: %v", err)
	}

	// Encrypter should have closed the file.
	if err := testDoc2.Close(); err == nil {
		t.Fatalf("testFile2 should be closed")
	}

	// Close the channel, worker should exit.
	close(streamChan)
}
