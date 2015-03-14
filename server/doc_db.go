package main

import (
	"bytes"
	"encoding/binary"
	"errors"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
)

var (
	ErrDocDoesNotExist = errors.New("document does not exist")

	numReaders   = runtime.NumCPU() * 3
	chanBufSize  = numReaders * 2
	docBucketKey = []byte("docs")
)

// readDocRequest represents a request to read a document.
type readDocRequest struct {
	docId uint32
	err   chan error
	resp  chan *pb.CipherDoc
}

// putDocRequest represents a request to store a document.
type putDocRequest struct {
	doc *pb.CipherDoc
	err chan error
}

// documentDatabase is a service capable of responding to read/write requests
// for encrypted documents.
type documentDatabase struct {
	db *bolt.DB

	quit     chan struct{}
	started  int32
	shutdown int32

	readRequests  chan *readDocRequest
	writeRequests chan *putDocRequest

	wg sync.WaitGroup
}

// NewDocumentDatabas creates and returns a new instance of documentDatabase.
func NewDocumentDatabase(db *bolt.DB) (*documentDatabase, error) {
	return &documentDatabase{
		quit:          make(chan struct{}),
		readRequests:  make(chan *readDocRequest, chanBufSize),
		writeRequests: make(chan *putDocRequest, chanBufSize),
	}, nil
}

// Start initializes help goroutines, kicking the doc db into active mode ready
//to respond to requests.
func (d *documentDatabase) Start() error {
	if atomic.AddInt32(&d.started, 1) != 1 {
		return nil
	}

	for i := 0; i < numReaders; i++ {
		d.wg.Add(1)
		go d.readHandler()
	}

	d.wg.Add(1)
	go d.writeHandler()

	return nil
}

// Stops triggers graceful stoppage of the doc db.
func (d *documentDatabase) Stop() error {
	if atomic.AddInt32(&d.shutdown, 1) != 1 {
		return nil
	}
	close(d.quit)
	d.wg.Wait()
	return nil
}

// readHandler handles incoming read requests.
func (d *documentDatabase) readHandler() {
	docKeyBuf := make([]byte, 4)
	var docBuf bytes.Buffer
out:
	for {
		select {
		case <-d.quit:
			break out
		case req := <-d.readRequests:
			// Convert the uint32 doc id into bytes for retrieval by
			// key.
			binary.LittleEndian.PutUint32(docKeyBuf, req.docId)

			// Look up the document returning the doc if it exists.
			err := d.db.View(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucketIfNotExists(docBucketKey)
				if err != nil {
					return nil
				}

				doc := bucket.Get(docKeyBuf)
				if doc == nil {
					return ErrDocDoesNotExist
				}

				docBuf.Write(doc)
				return nil
			})

			if err != nil {
				req.err <- err
				req.resp <- nil
			} else {
				retBuf := make([]byte, docBuf.Len())
				_, err := docBuf.Read(retBuf)

				if err != nil {
					req.err <- err
					req.resp <- nil
				} else {
					req.err <- nil
					req.resp <- &pb.CipherDoc{
						DocId:        req.docId,
						EncryptedDoc: retBuf,
					}
				}
			}

			docBuf.Reset()
		}
	}
	d.wg.Done()
}

// writeHandler handles incoming write requests.
func (d *documentDatabase) writeHandler() {
	docKeyBuf := make([]byte, 4)
out:
	for {
		select {
		case <-d.quit:
			break out
		case req := <-d.writeRequests:
			// Convert the uint32 doc id into bytes for storage by key.
			binary.LittleEndian.PutUint32(docKeyBuf, req.doc.DocId)

			err := d.db.Update(func(tx *bolt.Tx) error {
				bucket, err := tx.CreateBucketIfNotExists(docBucketKey)
				if err != nil {
					return nil
				}

				err = bucket.Put(docKeyBuf, req.doc.EncryptedDoc)
				if err != nil {
					return nil
				}

				return nil
			})

			req.err <- err
		}
	}
	d.wg.Done()
}

// PutDoc queues an encrypted document for permanent storage in the DB.
func (d *documentDatabase) PutDoc(doc *pb.CipherDoc) error {
	err := make(chan error, 1)
	req := &putDocRequest{err: err, doc: doc}

	d.writeRequests <- req
	return <-req.err
}

// RetrieveDoc retrives an encrypted document by it's ID.
func (d *documentDatabase) RetrieveDoc(docId uint32) (*pb.CipherDoc, error) {
	err := make(chan error, 1)
	resp := make(chan *pb.CipherDoc, 1)
	req := &readDocRequest{err: err, docId: docId, resp: resp}

	return <-req.resp, <-req.err
}
