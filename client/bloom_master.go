package main

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"github.com/willf/bloom"
	"golang.org/x/net/context"
)

// BloomFrequencyBucket represents induvidual frequency buckets that a
// particular term can fall into.
type BloomFrequencyBucket int

const (
	Below100 BloomFrequencyBucket = iota
	Below1000
	Below10000
	Below100000
	BucketUnknown
)

const (
	// Number of frequency buckets.
	numBuckets = 4
	// The false positive rate of each bloom filter.
	fpRate = float64(0.000001)
)

// The boltDB key that houses the bucket where we stored out bloom filters.
var bloomBucketKey = []byte("blooms")

// A helper map to quickly identify the key for a particular frequency bucket.
var bucketToKey = map[BloomFrequencyBucket][]byte{
	Below100:    []byte("b_100"),
	Below1000:   []byte("b_1000"),
	Below10000:  []byte("b_10000"),
	Below100000: []byte("b_100000"),
}

// finishedBloom is sent by a bloomWorker once a particular bloom filter frequency
// bucket becomes "full". Once this message is sent the bloomSaver can persist the
// completed filter to disk
type finishedBloom struct {
	whichBucket BloomFrequencyBucket
	filter      *bloom.BloomFilter
}

// xSetSizeInitMsg is sent by collaborators of the bloomMaster that wish to
// create the bloom filter for the X-Set.
type xSetSizeInitMsg struct {
	numElements uint
}

// xSetAddMsg sends a message to the bloomMaster to add a list of xTags into
// the X-Set.
type xSetAddMsg struct {
	xTags []xTag
	// TODO(roasbeef): Need error chans or???
	//errChan error
}

// freqBucketInitMsg is sent by collaborators of the bloomMaster that wish to
// create the frequency bucket bloom filters. The map should map a bucket name
// to the max number of elements in that bucket.
type freqBucketInitMsg struct {
	frequencyMap map[BloomFrequencyBucket]uint
}

// freqBucketAddMsg sends a message to the bloomMaster to add a list of terms
// to a particular bloom frequency bucket.
type freqBucketAddMsg struct {
	whichBucket BloomFrequencyBucket
	terms       []string
}

// wordQueryRequest represents a bloom filter query. The query response will be
// return via the passed 'resp' channel. A value of `BucketUnkown` means that
// the word isn't contained in *any* of the frequency buckets.
type wordQueryRequest struct {
	query string
	resp  chan BloomFrequencyBucket
}

// bloomMaster handles the initialization, creation, and coordination of all
// actions related to bloom filters. We use bloom filters for two things: to
// quickly identify the frequency bucket of a particular word, and to create a
// bloom filter of the X-Set which will be sent to the server.
type bloomMaster struct {
	numWorkers int32
	// TODO(roasbeef): Buffer, how large? Multiple of num workers?
	msgChan chan interface{}

	// We use these struct channels as notification variables. Waiters do
	// blocking read on the channel, and are woken up once close() is
	// called on the particular channel.
	xSetReady        chan struct{}
	freqBucketsReady chan struct{}
	xFilterFinished  chan struct{}

	// TODO(roasbeef): make buffered of bucket size
	finishedFreqBlooms chan *finishedBloom

	// Channel to send bloom filter queries over.
	wordQueries chan *wordQueryRequest

	// The bloom filter containing all xTags in the X-Set.
	xSetFilter        *bloom.BloomFilter
	xFinalSize        uint
	xNumAddedElements int32

	// The bloom filters which represent buckets of frequency ranges of
	// each word.
	bloomFreqBuckets  map[BloomFrequencyBucket]*bloom.BloomFilter
	bucketStatsMtx    sync.Mutex
	fullBucketSize    map[BloomFrequencyBucket]uint
	currentBucketSize map[BloomFrequencyBucket]uint

	// Internal variable to indicate if this is the first-time setup.
	isFirstTime bool
	db          *bolt.DB

	// Our gRPC client.
	client pb.EncryptedSearchClient

	wg       sync.WaitGroup
	quit     chan struct{}
	started  int32
	shutdown int32
}

// newBloomMaster....
func newBloomMaster(db *bolt.DB, numWorkers int) (*bloomMaster, error) {
	// Do we already have all the filters saved?
	// TODO(roasbeef): Determine this at an upper layer?
	firstTime := false
	err := db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(bloomBucketKey))
		firstTime = (b == nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return &bloomMaster{
		db:                 db,
		isFirstTime:        firstTime,
		msgChan:            make(chan interface{}, numWorkers*3),
		quit:               make(chan struct{}),
		xSetReady:          make(chan struct{}),
		freqBucketsReady:   make(chan struct{}),
		xFilterFinished:    make(chan struct{}),
		finishedFreqBlooms: make(chan *finishedBloom, numBuckets),
		wordQueries:        make(chan *wordQueryRequest, numBuckets),
		bloomFreqBuckets:   make(map[BloomFrequencyBucket]*bloom.BloomFilter),
		fullBucketSize:     make(map[BloomFrequencyBucket]uint),
		currentBucketSize:  make(map[BloomFrequencyBucket]uint),
	}, nil
}

// Start kicks off the bloomMaster and all it's helper goroutines.
func (b *bloomMaster) Start() error {
	if atomic.AddInt32(&b.started, 1) != 1 {
		return nil
	}

	if b.isFirstTime {
		b.wg.Add(1)
		go b.xFilterUploader()

		for i := int32(0); i < b.numWorkers; i++ {
			b.wg.Add(1)
			go b.bloomWorker()
		}

		b.wg.Add(1)
		go b.freqBloomSaver()
	} else {

		// Load the buckets into memory for aide with conjunctive queries.
		err := b.db.View(func(tx *bolt.Tx) error {
			fBucket := tx.Bucket(bloomBucketKey)
			return b.loadFiltersFromDb(fBucket)
		})
		if err != nil {
			return err
		}
	}

	b.wg.Add(1)
	go b.queryHandler()

	return nil
}

// Stop shuts down the bloomMaster.
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

// WaitForBloomFreqInit allows workers who would like to send frequency
// bloom filter bucket upadtes to wait until the bloom filter has been created.
func (b *bloomMaster) WaitForBloomFreqInit() {
	<-b.freqBucketsReady
}

// bloomSaver handles saving our populated frequency bloom filter buckets to
// disk.
func (b *bloomMaster) freqBloomSaver() {
	numSaved := 0
out:
	for numSaved < numBuckets {
		select {
		case fBloom := <-b.finishedFreqBlooms:
			err := b.db.Update(func(tx *bolt.Tx) error {
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

				// Write the serializedFilter to disk.
				return bloomBucket.Put(
					bucketToKey[fBloom.whichBucket],
					serializedFilter,
				)
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
// bloomWorker handles creating and populating our two categories of
// bloom filters.
func (b *bloomMaster) bloomWorker() {
out:
	for {
		select {
		// TODO(roasbeef): Proper stoppage condition...
		case m, more := <-b.msgChan:
			if !more {
				break out
			}
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

// xFilterUploader waits until it has been signaled that the X-Set has been
// fully initialized. After getting this signal it opens a stream to the server
// and sends the entire X-Set bloom filter.
func (b *bloomMaster) xFilterUploader() {
out:
	// TODO(roasbeef): for loop needed?
	for {
		select {
		case <-b.quit:
			break out
		case <-b.xFilterFinished:
			xFilterBytes, err := b.xSetFilter.GobEncode()
			if err != nil {
				// TODO(roasbeef): Handle error
			}

			_, err = b.client.UploadXSetFilter(context.Background(),
				&pb.XSetFilter{BloomFilter: xFilterBytes})
			if err != nil {
				// TODO(roasbeef): Handle error
			}
			break out
		}
	}
	b.wg.Done()
}

// queryHandler handles bloom filter word frequency queries.
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

			req.resp <- BucketUnknown
		case <-b.quit:
			break out
		}
	}
	b.wg.Done()
}

// InitXSet sends a message indicating that the xSet filter should be created.
func (b *bloomMaster) InitXSet(numElements uint) error {
	// TODO(roasbeef): Do the same thing for KeyManager.
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return fmt.Errorf("Request ignored. Shutting Down.")
	}
	req := &xSetSizeInitMsg{numElements: numElements}
	b.msgChan <- req
	return nil
}

// QueueXSetAdd sends a message requesting for the passed xTags should be
// added to the xSet bloom filter.
func (b *bloomMaster) QueueXSetAdd(newXtags []xTag) error {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return fmt.Errorf("Request ignored. Shutting Down.")
	}
	req := &xSetAddMsg{xTags: newXtags}
	b.msgChan <- req
	return nil
}

// InitFreqBuckets sends a message indicating that the passed frequency buckets
// be initialized with the following size.
func (b *bloomMaster) InitFreqBuckets(freqs map[BloomFrequencyBucket]uint) error {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return fmt.Errorf("Request ignored. Shutting Down.")
	}
	req := &freqBucketInitMsg{frequencyMap: freqs}
	b.msgChan <- req
	return nil
}

// QueueFreqBucketAdd sends a message to the bloomMaster to add the list of
// words to the proper frequency bucket.
func (b *bloomMaster) QueueFreqBucketAdd(targetBucket BloomFrequencyBucket, words []string) error {
	// Don't send a message if we're already shutting down.
	if atomic.LoadInt32(&b.shutdown) != 0 {
		return fmt.Errorf("Request ignored. Shutting Down.")
	}
	req := &freqBucketAddMsg{whichBucket: targetBucket, terms: words}
	b.msgChan <- req
	return nil
}

// TODO(roasbeef): Handle re-init fail
// handleXSetInit creates the bloom filter for the X-Set.
func (b *bloomMaster) handleXSetInit(msg *xSetSizeInitMsg) {
	// Create the x-set bloom filter now that we have the proper parameters.
	b.xSetFilter = bloom.NewWithEstimates(msg.numElements, fpRate)
	b.xFinalSize = msg.numElements

	// Notify any upstream workers that are waiting for the initialization of
	// the bloom filter. We're essentially using the chan struct{} as a
	// condition variable.
	close(b.xSetReady)
}

// handleXSetAdd adds a list of xTags to the X-Set filter.
func (b *bloomMaster) handleXSetAdd(msg *xSetAddMsg) {
	for _, xTag := range msg.xTags {
		b.xSetFilter.Add(xTag)
		// TODO(Roasbeef): channel scheme instead??
		atomic.AddInt32(&b.xNumAddedElements, 1)
	}

	// Final element has been added, signal the streamer to begin.
	if uint(b.xNumAddedElements) == b.xFinalSize {
		close(b.xFilterFinished)
	}
}

// handleFreqBucketInit create a bloom filter for each frequency bucket.
func (b *bloomMaster) handleFreqBucketInit(msg *freqBucketInitMsg) {
	for bucketRange, targetSize := range msg.frequencyMap {
		filter := bloom.NewWithEstimates(targetSize, fpRate)
		b.bloomFreqBuckets[bucketRange] = filter
		b.fullBucketSize[bucketRange] = targetSize
	}

	// Notify any upstream workers that are waiting for the initialization of
	// the bloom filter. We're essentially using the chan struct{} as a
	// condition variable.
	close(b.freqBucketsReady)
}

// handleFreqBucketAdd adds a list of words to a particular frequency bucket.
// If after adding all the passed strings to the bloomfilter to bucket is full,
// then we also send the finished bucket off so it can be written to disk.
func (b *bloomMaster) handleFreqBucketAdd(msg *freqBucketAddMsg) {
	numAdded := uint(0)
	targetBucket := msg.whichBucket
	bucket, ok := b.bloomFreqBuckets[targetBucket]
	if !ok {
		return
	}

	for _, term := range msg.terms {
		bucket.Add([]byte(term))
		numAdded++
	}

	// Send off the finished bloom filter if the bucket is now full.
	b.bucketStatsMtx.Lock()
	b.currentBucketSize[targetBucket] += numAdded
	if b.fullBucketSize[targetBucket] == b.currentBucketSize[targetBucket] {
		b.finishedFreqBlooms <- &finishedBloom{filter: bucket,
			whichBucket: targetBucket}
	}
	b.bucketStatsMtx.Unlock()
}

// QueryWordFrequency sends a query to determine which frequency bucket a given
// word is a member of.
func (b *bloomMaster) QueryWordFrequency(word string) BloomFrequencyBucket {
	query := &wordQueryRequest{query: word, resp: make(chan BloomFrequencyBucket)}
	b.wordQueries <- query
	return <-query.resp
}

// loadFiltersFromDb retrives and deserializes each frequency bucket bloom
// filter form the DB.
func (b *bloomMaster) loadFiltersFromDb(bloomBucket *bolt.Bucket) error {
	for i := 0; i < numBuckets; i++ {
		filterBucket := BloomFrequencyBucket(i)
		key := bucketToKey[filterBucket]

		serializedFilter := bloomBucket.Get(key)
		if serializedFilter == nil {
			return fmt.Errorf("Frequency bloom filter %v not found", i)
		}

		var filter bloom.BloomFilter
		if err := filter.GobDecode(serializedFilter); err != nil {
			return fmt.Errorf("Unable to deserialize filter: %v", err)
		}

		b.bloomFreqBuckets[filterBucket] = &filter

	}
	return nil
}
