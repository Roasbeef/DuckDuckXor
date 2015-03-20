package main

import (
	"bytes"
	"encoding/gob"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	"github.com/roasbeef/DuckDuckXor/crypto"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"github.com/willf/bloom"
)

var (
	indexBucketKey  = []byte("index")
	tSetBucketKey   = []byte("tset")
	xSetKey         = []byte("xset")
	numTsetReaders  = runtime.NumCPU() * 3
	numTsetWriters  = 2
	searchChunkSize = uint32(20) // Proper # for chunk size?
)

// tSetWriteReq...
type tSetWriteReq struct {
	t *pb.TSetFragment
}

// tSetWriteReq...
type tSetReadReq struct {
	sTag   []byte
	resps  chan *tupleData // Should be buffered request side.
	cancel chan struct{}
}

type tupleData struct {
	eId         []byte
	blindingVal []byte
	workerIndex uint32
	isLast      bool
}

type tupleBatch []*tupleData

func (t tupleBatch) Len() int           { return len(t) }
func (t tupleBatch) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t tupleBatch) Less(i, j int) bool { return t[i].workerIndex < t[j].workerIndex }

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
	isMetaLoaded bool

	wg sync.WaitGroup
}

// NewEncryptedIndexSearcher...
func NewEncryptedIndexSearcher(db *bolt.DB) (*encryptedIndexSearcher, error) {
	// TODO(roasbeef): Check db for xfilter, if so load, signal to wait?
	return &encryptedIndexSearcher{
		db:              db,
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

	var indexIsPopulated bool
	err := e.db.View(func(tx *bolt.Tx) error {
		indexIsPopulated = tx.Bucket(indexBucketKey) != nil
		return nil
	})
	if err != nil {
		return err
	}

	// If the index has been created already. Then load the X-Set filter
	// from the DB. Otherwise, create tSet writers to handle the initial
	// indexing.
	if indexIsPopulated {
		if err := e.loadXSetFilterFromDb(); err != nil {
			return err
		}
	} else {
		for i := 0; i < numTsetWriters; i++ {
			go e.tSetWriter()
		}
	}

	for i := 0; i < numTsetReaders; i++ {
		go e.tSetReader()
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
func (e *encryptedIndexSearcher) PutXSetFilter(xf *pb.XSetFilter) {
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
	if !e.isMetaLoaded {
		close(e.metaDataRecived)
		e.isMetaLoaded = true
	}
}

// waitForMetaDataInit allows helper goroutines to wait, and the be signalled
// after the t-set meta data has been loaded.
func (e *encryptedIndexSearcher) waitForMetaDataInit() {
	<-e.metaDataRecived
}

// PutTsetFragment queues a tSet fragment to be written to the DB.
func (e *encryptedIndexSearcher) PutTsetFragment(tuple *pb.TSetFragment) {
	if tuple == nil {
		return
	}

	req := &tSetWriteReq{tuple}
	e.tWriteReqs <- req
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

			// TODO(roasbeef): Need to catch tset collision with prob 2^-32
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

// TSetSearch...
func (e *encryptedIndexSearcher) TSetSearch(stag []byte) (chan *tupleData, chan struct{}) {
	resp := make(chan *tupleData, searchChunkSize)
	cancel := make(chan struct{}, 1)
	req := &tSetReadReq{sTag: stag, resps: resp, cancel: cancel}
	e.tReadReqs <- req

	return resp, cancel
}

// tSetReader...
func (e *encryptedIndexSearcher) tSetReader() {
	e.waitForMetaDataInit()
out:
	for {
	top:
		select {
		case <-e.quit:
			break out
		case readReq := <-e.tReadReqs:
			stag := readReq.sTag
			respChan := readReq.resps

			i := uint32(0)
			outChan := make(chan *tupleData, searchChunkSize)
			batchResults := make(tupleBatch, searchChunkSize)
			for {
				select {
				case <-readReq.cancel:
					break top
				default:
				}

				// Launch a batch of workers to retrieve tSet chunks.
				for j := i; j < i+searchChunkSize; j++ {
					go e.tupleFetcher(stag, j, outChan)
				}

				// Collect the results of this batch.
				for m := uint32(0); m < searchChunkSize; m++ {
					batchResults[i] = <-outChan
				}

				// Sort by worker index.
				sort.Sort(batchResults)

				// With our odering enforced, send out results
				// until we reach the end of the inverted index
				// for this stag.
				for _, t := range batchResults {
					if !t.isLast {
						respChan <- t
					} else {
						close(respChan)
						goto top
					}
				}

				i += searchChunkSize
			}
		}
	}
	e.wg.Done()
}

// tupleFetcher...
func (e *encryptedIndexSearcher) tupleFetcher(stag []byte, index uint32, outChan chan *tupleData) {
	label, bucket, otp := crypto.CalcTsetVals(stag, index)
	t := &tupleData{}

	// TODO(roasbeef): erorr?
	e.db.View(func(tx *bolt.Tx) error {
		// Fetch the root t-set bucket.
		rootBucket := tx.Bucket(tSetBucketKey)

		// Grab the bucket for this tuple.
		fragmentBucket := rootBucket.Bucket(bucket[:])

		// Decrypt the tuple using it's one-time-pad.
		encryptedTuple := fragmentBucket.Get(label[:])
		decryptedTuple := crypto.XorBytes(otp[:], encryptedTuple)

		// Signal if this is the last element.
		finishedByte := decryptedTuple[0]

		eIdLength := e.tSetMetaData.NumEidBytes
		blindLength := e.tSetMetaData.NumBlindBytes

		encryptedId := decryptedTuple[1:eIdLength]
		blindingVal := decryptedTuple[eIdLength:blindLength]

		t.eId = encryptedId
		t.blindingVal = blindingVal
		t.workerIndex = index
		t.isLast = finishedByte == 0x01
		return nil
	})
	outChan <- t
}
