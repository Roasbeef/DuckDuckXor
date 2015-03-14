package main

import (
	"bytes"
	"encoding/gob"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"github.com/willf/bloom"
)

var indexBucketKey = []byte("index")
var xSetKey = []byte("xset")

type encryptedIndexSearcher struct {
	db *bolt.DB

	quit     chan struct{}
	started  int32
	shutdown int32

	metaDataRecived chan struct{}

	xSet *bloom.BloomFilter

	tSetMetaData *pb.MetaData

	wg sync.WaitGroup
}

func NewEncryptedIndexSearcher(db *bolt.DB) (*encryptedIndexSearcher, error) {
	// TODO(roasbeef): Check db for xfilter, if so load, signal to wait?
	return &encryptedIndexSearcher{
		quit: make(chan struct{}),
	}, nil
}

func (e *encryptedIndexSearcher) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return nil
	}
	return nil
}

func (e *encryptedIndexSearcher) Stop() error {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		return nil
	}
	close(e.quit)
	e.wg.Wait()
	return nil
}

// LoadXSetFilter...
func (e *encryptedIndexSearcher) LoadXSetFilter(xf *pb.XSetFilter) {
	// Deserialize the xSet bloom filter and load it into memory.
	filterbuf := bytes.NewBuffer(xf.BloomFilter)
	err := gob.NewDecoder(filterbuf).Decode(e.xSet)
	if err != nil {
		// TODO(roasbeef): handle err
	}

	// Store the recieved xSet in the DB.
	err = e.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(indexBucketKey)
		if err != nil {
			return err
		}

		// TODO(roasbeef): Need copy here?
		err = b.Put(xSetKey, xf.BloomFilter)
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
	}
	// TODO(roasbeef): Does anyone need to be notified by this?
}

// loadXSetFilterFromDb retrieves and deserializes a stored x-set bloom filter
// before loading it into memory.
func (e *encryptedIndexSearcher) loadXSetFilterFromDb() error {
	return e.db.View(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucketIfNotExists(indexBucketKey)
		if err != nil {
			return err
		}

		// TODO(roasbeef): Need copy here?
		filter := b.Get(xSetKey)
		if err != nil {
			return err
		}

		filterBuf := bytes.NewBuffer(filter)
		err = gob.NewDecoder(filterBuf).Decode(e.xSet)
		if err != nil {
			return nil
		}

		return nil
	})
}

// LoadTSetMetaData loads metadata about the tSet into memory.
func (e *encryptedIndexSearcher) LoadTSetMetaData(mData *pb.MetaData) {
	e.tSetMetaData = mData
	close(e.metaDataRecived)
}

// waitForMetaDataInit... allows helper goroutines to wait, and the be signalled
// after the t-set meta data has been loaded.
func (e *encryptedIndexSearcher) waitForMetaDataInit() {
	<-e.metaDataRecived
}
