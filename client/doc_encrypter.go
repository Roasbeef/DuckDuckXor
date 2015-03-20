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

// encryptedDocs represents an encrypted document and its document ID.
type encryptedDoc struct {
	cipherText []byte
	DocId      uint32
}

// EncryptedDocStreamer is responsible for encrypting and sending encrypting
// documents to the server.
type EncryptedDocStreamer struct {
	docKey        snacl.CryptoKey
	encryptedDocs chan *encryptedDoc
	docStream     chan *document

	numWorkers int32
	client     pb.EncryptedSearchClient

	started  int32
	shutdown int32
	quit     chan struct{}
	wg       sync.WaitGroup

	abort func(chan struct{}, error)
}

// NewEncryptedDocStreamer creates a new EncryptedDocStreamer.
func NewEncryptedDocStreamer(numWorkers int32, docKey *[keySize]byte, docStream chan *document, client pb.EncryptedSearchClient, abort func(chan struct{}, error)) *EncryptedDocStreamer {
	q := make(chan struct{})
	return &EncryptedDocStreamer{
		docKey:        snacl.CryptoKey(*docKey),
		encryptedDocs: make(chan *encryptedDoc, numWorkers),
		docStream:     docStream,
		numWorkers:    numWorkers,
		quit:          q,
		client:        client,
		abort:         abort,
	}
}

// Start kicks off the EncryptedDocStreamer, creating all helper goroutines.
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

// Stop gracefully shuts down the encrypted doc streamer.
func (e *EncryptedDocStreamer) Stop() error {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		return nil
	}
	close(e.quit)
	e.wg.Wait()
	return nil
}

// docEncrypter handles encrypting passed documents from the docStream channel.
func (e *EncryptedDocStreamer) docEncrypter() {
	// TODO(roasbeef0: Proper re-use of buffer
	var plainBuffer bytes.Buffer
out:
	for {
		select {
		case doc, more := <-e.docStream:
			if !more {
				break out
			}
			doc.Seek(1, 0)

			_, err := io.Copy(&plainBuffer, doc)
			if err != nil {
				e.abort(e.quit, err)
			}

			err = doc.Close()
			if err != nil {
				e.abort(e.quit, err)
			}

			cipherDoc, err := e.docKey.Encrypt(plainBuffer.Bytes())
			if err != nil {
				e.abort(e.quit, err)
			}

			e.encryptedDocs <- &encryptedDoc{
				cipherText: cipherDoc,
				DocId:      doc.DocId,
			}

			// TODO(roasbeef): Needed?
			plainBuffer.Reset()
		case <-e.quit:
			break out
		}
	}
	close(e.encryptedDocs)
	e.wg.Done()
}

// docUploader is responsible for opening a gRPC stream to the document storage
// server, and streaming encrypted documents as they come in.
func (e *EncryptedDocStreamer) docUploader() {
	cipherStream, err := e.client.UploadCipherDocs(context.Background())
	if err != nil {
		e.abort(e.quit, err)
	}
out:
	for {
		select {
		case doc, more := <-e.encryptedDocs:
			if !more {
				cipherStream.CloseSend()
				break out
			}
			cipherDoc := &pb.CipherDoc{
				DocId:        doc.DocId,
				EncryptedDoc: doc.cipherText,
			}

			if err := cipherStream.Send(cipherDoc); err != nil {
				e.abort(e.quit, err)
			}
		case <-e.quit:
			break out
		}
	}
	e.wg.Done()
}
