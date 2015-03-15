package main

import (
	"bytes"
	"encoding/gob"
	"runtime"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	"github.com/roasbeef/DuckDuckXor/crypto"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"github.com/willf/bloom"
)

var (
	indexBucketKey = []byte("index")
	tSetBucketKey  = []byte("tset")
	xSetKey        = []byte("xset")
	numTsetReaders = runtime.NumCPU() * 3
	numTsetWriters = 2
)

// tSetWriteReq...
type tSetWriteReq struct {
	t *pb.TSetFragment
}

// tSetWriteReq...
type tSetReadReq struct {
	sTag  []byte
	resps chan *tupleData // Should be buffered request side.
}

type tupleData struct {
	eId         []byte
	blindingVal []byte
}

// encryptedIndexSearcher....
type encryptedIndexSearcher struct {
	// TODO(roasbeef): Add the WAL
	db *bolt.DB

	quit     chan struct{}
	started  int32
	shutdown int32

	// TODO(roasbeef): need meta-data send?
	metaDataRecived chan struct{}

	tWriteReqs chan *tSetWriteReq
	tReadReqs  chan *tSetReadReq

	xSet *bloom.BloomFilter

	tSetMetaData *pb.MetaData

	wg sync.WaitGroup
}

// NewEncryptedIndexSearcher...
func NewEncryptedIndexSearcher(db *bolt.DB) (*encryptedIndexSearcher, error) {
	// TODO(roasbeef): Check db for xfilter, if so load, signal to wait?
	return &encryptedIndexSearcher{
		quit:            make(chan struct{}),
		metaDataRecived: make(chan struct{}),
		tWriteReqs:      make(chan *tSetWriteReq, numTsetWriters*5),
		tReadReqs:       make(chan *tSetReadReq, numTsetReaders*10),
	}, nil
}

// Start...
func (e *encryptedIndexSearcher) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return nil
	}
	return nil
}

// Stop...
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

// waitForMetaDataInit allows helper goroutines to wait, and the be signalled
// after the t-set meta data has been loaded.
func (e *encryptedIndexSearcher) waitForMetaDataInit() {
	<-e.metaDataRecived
}

// tSetWriter accepts incoming requests to write the t-set to the DB.
func (e *encryptedIndexSearcher) tSetWriter() {
	//TODO(roasbeef): Kill this guy after client init finished?
out:
	for {
		select {
		case <-e.quit:
			break out
		case tFragment := <-e.tWriteReqs:
			targetBucket := tFragment.t.Bucket
			targetLabel := tFragment.t.Label
			tupleData := tFragment.t.Data

			err := e.db.Update(func(tx *bolt.Tx) error {
				// Fetch the root t-set bucket.
				rootBucket, err := tx.CreateBucketIfNotExists(tSetBucketKey)
				if err != nil {
					return err
				}

				// Grab the bucket for this tuple, creating if
				// it doesn't yet exist.
				fragmentBucket, err := rootBucket.CreateBucketIfNotExists(targetBucket)
				if err != nil {
					return err
				}

				// Finally put the t-set data at the proper lable
				// in the bucket.
				return fragmentBucket.Put(targetLabel, tupleData)
			})
			if err != nil {
				//TODO(roasbeef): error
			}
		}
	}
	e.wg.Done()
}

// PutTsetFragment queues a tSet fragment to be written to the DB.
func (e *encryptedIndexSearcher) PutTsetFragment(tuple *pb.TSetFragment) {
	req := &tSetWriteReq{tuple}
	e.tWriteReqs <- req
}

// tSetReader...
func (e *encryptedIndexSearcher) tSetReader() {
out:
	for {
	top:
		select {
		case <-e.quit:
			break out
		case readReq := <-e.tReadReqs:
			stag := readReq.sTag
			respChan := readReq.resps

			workerQuit := make(chan struct{})
			lastTupleFound := make(chan struct{}, 1)
			i := uint32(0)
			for {
				select {
				case <-lastTupleFound:
					close(respChan)
					break top
				default:
				}

				label, bucket, otp := crypto.CalcTsetVals(stag, i)

				go e.tupleFetcher(label[:], bucket[:], otp[:],
					lastTupleFound, workerQuit, respChan)

				i++
			}
		}
	}
	e.wg.Done()
}

// tupleFetcher...
func (e *encryptedIndexSearcher) tupleFetcher(l, b, k []byte, workerQuit chan struct{}, lastTupleFound chan struct{}, respChan chan *tupleData) {
	select {
	case <-workerQuit:
		return
	default:
	}

	err := e.db.View(func(tx *bolt.Tx) error {
		// Fetch the root t-set bucket.
		rootBucket := tx.Bucket(tSetBucketKey)

		// Grab the bucket for this tuple.
		fragmentBucket := rootBucket.Bucket(b)

		// Decrypt the tuple using it's one-time-pad.
		encryptedTuple := fragmentBucket.Get(l)
		decryptedTuple := crypto.XorBytes(k, encryptedTuple)

		// Signal if this is the last element.
		finishedByte := decryptedTuple[0]
		if finishedByte == 0x01 {
			lastTupleFound <- struct{}{}
			close(workerQuit)
		}

		eIdLength := e.tSetMetaData.NumEidBytes
		blindLength := e.tSetMetaData.NumBlindBytes

		encryptedId := decryptedTuple[1:eIdLength]
		blindingVal := decryptedTuple[eIdLength:blindLength]

		respChan <- &tupleData{
			eId:         encryptedId,
			blindingVal: blindingVal,
		}
		return nil
	})

	if err != nil {
		//TODO(roasbeef): Error

	}
}
