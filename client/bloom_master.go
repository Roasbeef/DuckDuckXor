package main

import (
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	"github.com/willf/bloom"
)

type BloomFrequencyBucket int

const (
	Below100 BloomFrequencyBucket = iota
	Below1000
	Below10000
	Below100000
	BucketUnkown
)

const (
	numBuckets = 4
	fpRate     = float64(.000001)
)

var bloomBucketKey = []byte("blooms")

var bucketToKey = map[BloomFrequencyBucket][]byte{
	Below100:    []byte("b_100"),
	Below1000:   []byte("b_1000"),
	Below10000:  []byte("b_10000"),
	Below100000: []byte("b_100000"),
}

type finishedBloom struct {
	whichBucket BloomFrequencyBucket
	filter      *bloom.BloomFilter
}

// XSetSizeInit....
type xSetSizeInitMsg struct {
	numElements uint
}

// TODO(roasbeef): Adding functions to this could make downstrema
// changes easier more testable. Move for xset.go?
type xTag []byte

// XSetAdd...
type xSetAddMsg struct {
	xTags []xTag
	// TODO(roasbeef): Need error chans or???
	//errChan error
}

// FreqBucketInit....
type freqBucketInitMsg struct {
	frequencyMap map[BloomFrequencyBucket]uint
}

// FreqBucketAdd...
type freqBucketAddMsg struct {
	whichBucket BloomFrequencyBucket
	terms       []string
}

type wordQueryRequest struct {
	query string
	resp  chan BloomFrequencyBucket
}

// bloomMaster...
type bloomMaster struct {
	numWorkers int32
	// TODO(roasbeef): Buffer, how large? Multiple of num workers?
	msgChan chan interface{}

	// Close after init message
	xSetReady        chan struct{}
	freqBucketsReady chan struct{}

	// TODO(roasbeef): make buffered of bucket size
	finishedFreqBlooms chan finishedBloom

	wordQueries chan wordQueryRequest

	xSetFilter *bloom.BloomFilter

	bloomFreqBuckets map[BloomFrequencyBucket]*bloom.BloomFilter

	isFirstTime bool
	db          *bolt.DB

	wg       sync.WaitGroup
	quit     chan struct{}
	started  int32
	shutdown int32
}

// newBloomMaster....
func newBloomMaster(db *bolt.DB) (*bloomMaster, error) {
	return nil, nil
}

// Start...
func (b *bloomMaster) Start() error {
	if atomic.AddInt32(&b.started, 1) != 1 {
		return nil
	}

	b.wg.Add(1)
	// if isFirstTime...
	//go k.streamWorker()

	return nil
}

// Stop....
func (b *bloomMaster) Stop() error {
	if atomic.AddInt32(&b.shutdown, 1) != 1 {
		return nil
	}
	close(b.quit)
	b.wg.Wait()
	return nil
}

// WaitForXSetInit allows workers who would like to send x-set bloom filter
// updates to wait until the bloom filter has been initialized.
func (b *bloomMaster) WaitForXSetInit() {
	<-b.xSetReady
}

// WaitForBloomFreqInit...
func (b *bloomMaster) WaitForBloomFreqInit() {
	<-b.freqBucketsReady
}

// bloomSaver...
func (b *bloomMaster) bloomSaver() {
	numSaved := 0
out:
	for numSaved < numBuckets {
		select {
		case fBloom := <-b.finishedFreqBlooms:
			err := b.db.View(func(tx *bolt.Tx) error {
				// Grab our bucket, creating if it doesn't already exist.
				bloomBucket, err := tx.CreateBucketIfNotExists(bloomBucketKey)
				if err != nil {
					return err
				}

				// Serialize and save our filter to disk.
				serializedFilter, err := fBloom.filter.GobEncode()
				if err != nil {
					return err
				}

				err = bloomBucket.Put(bucketToKey[fBloom.whichBucket],
					serializedFilter)
				if err != nil {
					return err
				}

				return nil
			})
			if err != nil {
				// TODO(roasbeef): handle errs
			}
			numSaved++
		case <-b.quit:
			break out
		}
	}
	b.wg.Done()
}

// TODO(roasbeef): Have each stage of pipeline take WG group.
// bloomWorker...
func (b *bloomMaster) bloomWorker() {
out:
	for {
		select {
		case m := <-b.msgChan:
			switch msg := m.(type) {
			case *xSetSizeInitMsg:
				b.handleXSetInit(msg)
			case *xSetAddMsg:
				b.handleXSetAdd(msg)
			case *freqBucketInitMsg:
				b.handleFreqBucketInit(msg)
			case *freqBucketAddMsg:
				b.handleFreqBucketAdd(msg)
			default:
				// Invalid message type do something
			}
		case <-b.quit:
			break out
		}
	}

	b.wg.Done()
}

// queryHandler...
func (b *bloomMaster) queryHandler() {
out:
	for {
	top:
		select {
		case req := <-b.wordQueries:
			for bucketRange, filter := range b.bloomFreqBuckets {
				if filter.Test([]byte(req.query)) {
					req.resp <- bucketRange
					break top
				}
			}

			req.resp <- BucketUnkown
		case <-b.quit:
			break out
		}
	}
	b.wg.Done()
}

// InitXSet....
func (b *bloomMaster) InitXSet(numElements uint) {
	// TODO(roasbeef): Do the same thing for KeyManager.
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}
	req := &xSetSizeInitMsg{numElements: numElements}
	b.msgChan <- req
}

// QueueXSetAdd....
func (b *bloomMaster) QueueXSetAdd(newXtags []xTag) {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}
	req := &xSetAddMsg{xTags: newXtags}
	b.msgChan <- req
}

// InitFreqBuckets...
func (b *bloomMaster) InitFreqBuckets(freqs map[BloomFrequencyBucket]uint) {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}
	req := &freqBucketInitMsg{frequencyMap: freqs}
	b.msgChan <- req
}

// QueueFreqBucketAdd....
func (b *bloomMaster) QueueFreqBucketAdd(targetBucket BloomFrequencyBucket, words []string) {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return
	}
	req := &freqBucketAddMsg{whichBucket: targetBucket, terms: words}
	b.msgChan <- req
}

// handleXSetInit
// TODO(roasbeef): Handle re-init fail
func (b *bloomMaster) handleXSetInit(msg *xSetSizeInitMsg) {
	// Create the x-set bloom filter now that we have the proper parameters.
	b.xSetFilter = bloom.NewWithEstimates(msg.numElements, fpRate)

	// Notify any upstream workers that are waiting for the initialization of
	// the bloom filter. We're essentially using the chan struct{} as a
	// condition variable.
	close(b.xSetReady)
}

// handleXSetAdd....
func (b *bloomMaster) handleXSetAdd(msg *xSetAddMsg) {
	for _, xTag := range msg.xTags {
		b.xSetFilter.Add(xTag)
	}
}

// handleFreqBucketInit...
func (b *bloomMaster) handleFreqBucketInit(msg *freqBucketInitMsg) {
	for bucketRange, targetSize := range msg.frequencyMap {
		filter := bloom.NewWithEstimates(targetSize, fpRate)
		b.bloomFreqBuckets[bucketRange] = filter
	}

	// Notify any upstream workers that are waiting for the initialization of
	// the bloom filter. We're essentially using the chan struct{} as a
	// condition variable.
	close(b.freqBucketsReady)
}

// handleFreqBucketAdd....
func (b *bloomMaster) handleFreqBucketAdd(msg *freqBucketAddMsg) {
	bucket, ok := b.bloomFreqBuckets[msg.whichBucket]
	if !ok {
		return
	}

	for _, term := range msg.terms {
		bucket.Add([]byte(term))
	}

}

// QueryWordFrequency....
func (b *bloomMaster) QueryWordFrequency(word string) BloomFrequencyBucket {
	query := wordQueryRequest{query: word, resp: make(chan BloomFrequencyBucket)}

	b.wordQueries <- query

	return <-query.resp
}
