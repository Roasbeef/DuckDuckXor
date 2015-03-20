package main

import (
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/boltdb/bolt"
	"github.com/willf/bloom"
)

func createFakeDB(t *testing.T) (*bolt.DB, string) {
	// Create a new database to run tests against.
	dbPath := "fakeTest.db"
	db, err := bolt.Open(dbPath, 0600, nil)
	if err != nil {
		t.Fatalf("Failed to create test database %v", err)
	}
	return db, dbPath
}

func TestQueryWordFrequency(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Create a frequency bucket, add some random words.
	b.bloomFreqBuckets[Below100] = bloom.NewWithEstimates(100, 0.00001)
	b.bloomFreqBuckets[Below100].Add([]byte("testing"))
	b.bloomFreqBuckets[Below100].Add([]byte("is"))
	b.bloomFreqBuckets[Below100].Add([]byte("cool"))

	// Launch our query handler.
	AddToWg(&b.wg, &mainWg, 1)
	go b.queryHandler()

	// Test some queries.
	resp := b.QueryWordFrequency("testing")
	if resp != Below100 {
		t.Fatalf("Query failed, should have %v, got %v", Below100, resp)
	}

	resp = b.QueryWordFrequency("is")
	if resp != Below100 {
		t.Fatalf("Query failed, should have %v, got %v", Below100, resp)
	}

	resp = b.QueryWordFrequency("cool")
	if resp != Below100 {
		t.Fatalf("Query failed, should have %v, got %v", Below100, resp)
	}

	b.Stop()
}

func TestFreqBucketAdd(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Create a frequency bucket.
	b.bloomFreqBuckets[Below100] = bloom.NewWithEstimates(100, 0.00001)
	// This bucekt is full after 5 elements have been added.
	b.fullBucketSize[Below100] = 5

	// Launch two bloom workers.
	AddToWg(&b.wg, &mainWg, 2)
	go b.bloomWorker()
	go b.bloomWorker()

	AddToWg(&b.wg, &mainWg, 1)
	go b.queryHandler()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		// Grab the finished filter.
		<-b.finishedFreqBlooms
		wg.Done()
	}()

	// Add some words to a filter bucket.
	b.QueueFreqBucketAdd(Below100, []string{"testing", "is", "nice"})
	b.QueueFreqBucketAdd(Below100, []string{"filters", "ok"})

	// Ensure the finished filter gets sent.
	wg.Wait()

	// Bucket should now be full.
	if b.currentBucketSize[Below100] != 5 {
		t.Fatalf("Bucket size is incorrect, got %v, should be %v",
			b.currentBucketSize[Below100], 5)
	}

	// Test word was properly added.
	resp := b.QueryWordFrequency("testing")
	if resp != Below100 {
		t.Fatalf("Query failed, should have %v, got %v", Below100, resp)
	}

	b.Stop()
}

func TestFreqBucketInitAndSigal(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Launch two bloom workers.
	AddToWg(&b.wg, &mainWg, 2)
	go b.bloomWorker()
	go b.bloomWorker()

	// Create a fake frequency bucket map.
	freqMap := make(map[BloomFrequencyBucket]uint)
	bucketSizes := []uint{4, 5, 6, 7}
	freqMap[Below100] = bucketSizes[0]
	freqMap[Below1000] = bucketSizes[1]
	freqMap[Below10000] = bucketSizes[2]
	freqMap[Below100000] = bucketSizes[3]

	// Send off the map for init
	b.InitFreqBuckets(freqMap)

	var wg sync.WaitGroup
	wg.Add(2)
	// Simulate helper goroutines waiting for the bucket to be init'd.
	go func() {
		b.WaitForBloomFreqInit()
		wg.Done()
	}()
	go func() {
		b.WaitForBloomFreqInit()
		wg.Done()
	}()

	// Goroutine above should have exited.
	wg.Wait()

	// Ensure buckets were created with proper max size.
	for i := 0; i < numBuckets; i++ {
		if b.bloomFreqBuckets[BloomFrequencyBucket(i)] == nil {
			t.Fatalf("Bucket %v wasn't created", i)
		}

		if b.fullBucketSize[BloomFrequencyBucket(i)] != bucketSizes[i] {
			t.Fatalf("Incorrect bucket size set. Got %v, should be %v",
				b.fullBucketSize[BloomFrequencyBucket(i)], bucketSizes[i])
		}
	}

	b.Stop()
}

func TestXSetAdd(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	b.xSetFilter = bloom.NewWithEstimates(100, 0.00001)
	b.xFinalSize = 4

	// Create some bloom workers.
	AddToWg(&b.wg, &mainWg, 2)
	go b.bloomWorker()
	go b.bloomWorker()

	// Send off some fake xtags
	b.QueueXSetAdd([]xTag{xTag([]byte("random")), xTag([]byte("bytes"))})
	b.QueueXSetAdd([]xTag{xTag([]byte("kjsalf")), xTag([]byte("kmlkfk"))})

	// Create some waiter goroutines.
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		<-b.xFilterFinished
		wg.Done()
	}()
	go func() {
		<-b.xFilterFinished
		wg.Done()
	}()

	// Workers should be signalled.
	wg.Wait()

	// Max elements should be reached.
	if b.xNumAddedElements != 4 {
		t.Fatalf("Number of added elements incorrect, got %v, should be %v",
			b.xNumAddedElements, 4)
	}

	b.Stop()
}

func TestXSetInit(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Launch two bloom workers.
	AddToWg(&b.wg, &mainWg, 2)
	go b.bloomWorker()
	go b.bloomWorker()

	// Init xSet filter with chosen size.
	b.InitXSet(100)

	var wg sync.WaitGroup
	wg.Add(2)
	// Simulate helper goroutines waiting for the xset filter to be init'd.
	go func() {
		b.WaitForXSetInit()
		wg.Done()
	}()
	go func() {
		b.WaitForXSetInit()
		wg.Done()
	}()

	// Goroutine above should have exited.
	wg.Wait()

	if b.xSetFilter == nil {
		t.Fatalf("xset filter wasn't properly created")
	}

	// Ensure buckets were created with proper max size.
	if b.xFinalSize != 100 {
		t.Fatalf("Max size of filter wasn't set. Got %v, should be %v",
			b.xFinalSize, 100)
	}

	b.Stop()
}

func TestFilterUploader(t *testing.T) {
	// make fake filter
	// launch goroutine
	// mock out client call
	// verify can still use filter??
}

func TestFreqBloomSaver(t *testing.T) {
	db, path := createFakeDB(t)
	var mainWg sync.WaitGroup
	defer os.Remove(path)
	defer db.Close()

	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Launch the bloom saver.
	AddToWg(&b.wg, &mainWg, 1)
	go b.freqBloomSaver()

	bloomSizes := []uint{100, 50, 90, 10}
	for i := 0; i < numBuckets; i++ {
		f := bloom.NewWithEstimates(bloomSizes[i], .000001)
		f.Add([]byte("test"))
		go func(bf *bloom.BloomFilter, bucketNum int) {
			b.finishedFreqBlooms <- &finishedBloom{
				whichBucket: BloomFrequencyBucket(bucketNum),
				filter:      bf,
			}
		}(f, i)
	}

	// Saver should have exited.
	b.wg.Wait()

	// Ensure buckets have been saved.
	for i := 0; i < numBuckets; i++ {
		key := bucketToKey[BloomFrequencyBucket(i)]
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket(bloomBucketKey)

			filter := b.Get(key)
			if filter == nil {
				return fmt.Errorf("Filter %v wasn't written", key)
			}

			var f bloom.BloomFilter
			if err := f.GobDecode(filter); err != nil {
				return fmt.Errorf("Unable to deserialize filter: %v", err)
			}

			if !f.Test([]byte("test")) {
				return fmt.Errorf("Inserted item 'test' not present.")
			}

			return nil
		})
		if err != nil {
			t.Fatalf("Error retrieving filter: %v", err)
		}
	}

	b.Stop()
}

func TestShutDownIgnoreRequests(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()
	var mainWg sync.WaitGroup

	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	b.Stop()

	// All requests now should return an error.
	if err := b.QueueXSetAdd(nil); err == nil {
		t.Fatalf("BM has shutdown, request didn't return an error")
	}
}

func TestStartUpShutDown(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	var mainWg sync.WaitGroup
	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	b.Start()
	fmt.Printf("got here\n")
	b.Stop()
}

func TestLoadFiltersFromDb(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()
	var mainWg sync.WaitGroup

	// Create our bloom master.
	b, err := newBloomMaster(db, 2, &mainWg)
	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	// Load some fake filters into the DB.
	err = b.db.Update(func(tx *bolt.Tx) error {
		// Grab our bucket, creating if it doesn't already exist.
		bloomBucket, err := tx.CreateBucketIfNotExists(bloomBucketKey)
		if err != nil {
			return err
		}
		for i := 0; i < numBuckets; i++ {
			filterBucket := BloomFrequencyBucket(i)
			key := bucketToKey[filterBucket]

			f, err := bloom.NewWithEstimates(100, 0.00001).GobEncode()
			if err != nil {
				return err
			}

			err = bloomBucket.Put(key, f)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		t.Fatalf("Unable to store filters: %v", err)
	}

	// Load the filters from the DB. Should return without error.
	err = b.db.View(func(tx *bolt.Tx) error {
		fBucket := tx.Bucket(bloomBucketKey)
		return b.loadFiltersFromDb(fBucket)
	})
	if err != nil {
		t.Fatalf("Unable to load filters: %v", err)
	}
}
