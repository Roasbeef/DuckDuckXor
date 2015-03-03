package main

import (
	"bytes"
	"io"
	"sync"
	"sync/atomic"

	"github.com/conformal/btcwallet/snacl"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
)

type encryptedDoc struct {
	cipherText []byte
	DocId      int32
}

// EncryptedDocStreamer...
type EncryptedDocStreamer struct {
	key           []byte
	encryptedDocs chan *encryptedDoc
	docStream     chan *document

	numWorkers int32
	client     pb.EncryptedSearchClient

	keyRing *KeyManager

	started  int32
	shutdown int32
	quit     chan struct{}
	wg       sync.WaitGroup
}

// NewEncryptedDocStreamer...
func NewEncryptedDocStreamer(numWorkers int32, errChan chan error, aesKey []byte,
	docStream chan *document, client pb.EncryptedSearchClient, keyRing *KeyManager) *EncryptedDocStreamer {
	q := make(chan struct{})
	return &EncryptedDocStreamer{
		key:           aesKey,
		encryptedDocs: make(chan *encryptedDoc, numWorkers),
		docStream:     docStream,
		keyRing:       keyRing,
		numWorkers:    numWorkers,
		quit:          q,
		client:        client,
	}
}

// Coordinator...
func (e *EncryptedDocStreamer) Coordinator() {
}

// Start...
func (e *EncryptedDocStreamer) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return nil
	}

	for i := int32(0); i < e.numWorkers; i++ {
		e.wg.Add(1)
		go e.docEncrypter()
	}

	e.wg.Add(1)
	go e.docUploader()

	return nil
}

// Stop...
func (e *EncryptedDocStreamer) Stop() error {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		return nil
	}
	close(e.quit)
	e.wg.Wait()
	return nil
}

// docEncrypter...
func (e *EncryptedDocStreamer) docEncrypter() {
	// TODO(roasbeef0: Proper re-use of buffer
	var plainBuffer bytes.Buffer

	docKey := e.keyRing.FetchDocEncKey()
	snaclDocKey := snacl.CryptoKey(*docKey)
out:
	for {
		select {
		case doc := <-e.docStream:
			doc.Seek(1, 0)

			numWritten, err := io.Copy(&plainBuffer, doc)
			if err != nil {
				// TODO(roasbeef): Handle failure
			}

			err = doc.Close()
			if err != nil {
				// TODO(roasbeef): Handle failure
			}

			cipherDoc, err := snaclDocKey.Encrypt(plainBuffer.Bytes()[:numWritten])
			if err != nil {
				// TODO(roasbeef): Handle failure
			}

			e.encryptedDocs <- &encryptedDoc{
				cipherText: cipherDoc,
				DocId:      doc.DocId,
			}

			// TODO(roasbeef): Needed?
			//plainBuffer.Reset()
		case <-e.quit:
			break out
		}
	}
	e.wg.Done()
}

// docUploader...
func (e *EncryptedDocStreamer) docUploader() {
	cipherStream, err := e.client.UploadCipherDocs(context.Background())
	if err != nil {
		// TODO(roasbeef): Handle err
	}
out:
	for {
		select {
		case doc, ok := <-e.encryptedDocs:
			if !ok {
				cipherStream.CloseSend()
				break out
			}
			cipherDoc := &pb.CipherDoc{
				DocId:        doc.DocId,
				EncryptedDoc: doc.cipherText,
			}

			if err := cipherStream.Send(cipherDoc); err != nil {
				// log.Fatalf("Failed to send a doc: %v", err)
				// TODO(roasbeef): handle errs
			}
		case <-e.quit:
			break out
		}
	}
	e.wg.Done()
}
