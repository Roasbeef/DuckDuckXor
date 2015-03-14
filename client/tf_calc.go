package main

import (
	"fmt"
	"hash/fnv"
	"sync"
	"sync/atomic"
)

type wordPair struct {
	key string
	tf  int
}

type bucketVals struct {
	ltHunredbucketSize   uint
	ltOneKbucketSize     uint
	ltTenKbucketSize     uint
	ltHundredKBucketSize uint
}
type TermFrequencyCalculator struct {
	quit               chan struct{}
	numWorkers         uint32
	reduceMap          map[uint32]chan wordPair
	bloomSizeChan      chan bucketVals
	bloomPopulateChan  chan bucketVals
	bloomInitChan      chan int
	TermFreq           chan map[string]int
	docIn              chan []string
	wg                 sync.WaitGroup
	ResultMap          map[string]int
	started            int32
	shutDown           int32
	err                chan error
	mappersDone        chan struct{}
	shufflersDone      chan struct{}
	shufflerQuit       chan struct{}
	reducerQuit        chan struct{}
	ltHunredbucketSize uint
	ltOneKbucketSize   uint
	ltTenKbucketSize   uint
	abort              func(chan struct{}, error)
	numReducers        uint32
	sync.Mutex
	ltHundredKBucketSize uint
	bloomFilterManager   *bloomMaster
}

//TermFreq shoud have as many buffers as workers
func NewTermFrequencyCalculator(numWorkers uint32, d chan []string, bm *bloomMaster, abort func(chan struct{}, error)) TermFrequencyCalculator {
	size := make(chan bucketVals)
	populate := make(chan bucketVals)
	r := make(map[uint32]chan wordPair)
	var i uint32
	numReducers := uint32(26)
	for i = 0; i < 26; i++ {
		r[i] = make(chan wordPair)
	}
	return TermFrequencyCalculator{
		quit:               make(chan struct{}),
		numWorkers:         numWorkers,
		reduceMap:          r,
		bloomSizeChan:      size,
		bloomPopulateChan:  populate,
		bloomInitChan:      make(chan int),
		TermFreq:           make(chan map[string]int),
		numReducers:        numReducers,
		mappersDone:        make(chan struct{}, numWorkers),
		shufflersDone:      make(chan struct{}, numReducers),
		shufflerQuit:       make(chan struct{}),
		reducerQuit:        make(chan struct{}),
		docIn:              d,
		abort:              abort,
		bloomFilterManager: bm,
	}
}

func (t *TermFrequencyCalculator) Start() error {
	if atomic.AddInt32(&t.started, 1) != 1 {
		return nil
	}
	t.initMappers()
	t.initShufflers()
	t.initReducers()
	//TODO after initializing bloom filters, wait for info
	//from lalu stating that the buckets are created
	go t.populateBloomFilters()

	return nil
}

func (t *TermFrequencyCalculator) Stop() error {

	if atomic.AddInt32(&t.started, 1) != 1 {
		return nil
	}
	close(t.quit)
	t.wg.Wait()
	return nil

}

func (t *TermFrequencyCalculator) initMappers() {
	for i := uint32(0); i < t.numWorkers; i++ {
		t.wg.Add(1)
		t.mappersDone <- struct{}{}
		go t.frequencyWorker()
	}
}

func (t *TermFrequencyCalculator) initReducers() {
	for i := uint32(0); i < t.numReducers; i++ {
		t.wg.Add(1)
		go t.reducer(i)
	}
}

func (t *TermFrequencyCalculator) initShufflers() {
	for i := uint32(0); i < t.numReducers; i++ {
		t.wg.Add(1)
		t.shufflersDone <- struct{}{}
		go t.shuffler()
	}

}

func (t *TermFrequencyCalculator) waitForBloomFilter() {

}

func (t *TermFrequencyCalculator) frequencyWorker() {
out:
	for {
		select {
		case <-t.quit:
			break out
		case doc, ok := <-t.docIn:
			m := make(map[string]int)
			if !ok {
				break out
			}
			for _, token := range doc {
				m[token] = m[token] + 1
			}
			t.TermFreq <- m
		}
	}
	<-t.mappersDone
	if len(t.mappersDone) == 0 {
		close(t.shufflerQuit)
	}
	t.wg.Done()
}

func (t *TermFrequencyCalculator) shuffler() {
	fmt.Printf("starting shuffler\n")
out:
	for {
		select {
		case <-t.quit:
			break out
		case doc := <-t.TermFreq:
			for key, val := range doc {
				hashValue := Hash(key)
				hashValue = hashValue % t.numReducers
				t.reduceMap[hashValue] <- wordPair{key, val}
			}
		case <-t.shufflerQuit:
			break out
		}

	}
	fmt.Printf("terminating shuffler\n")
	<-t.shufflersDone
	if len(t.shufflersDone) == 0 {
		close(t.reducerQuit)
	}
	t.wg.Done()
}

func Hash(s string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(s))
	return h.Sum32()

}

func (t *TermFrequencyCalculator) reducer(key uint32) {
	count := 0
	input := t.reduceMap[key]
	subSet := make(map[string]int)
out:
	for {
		select {
		case <-t.quit:
			break out
		case val := <-input:
			//TODO why am I doing this count?
			if subSet[val.key] == 0 {
				count++
			}
			if val.key == "golly" {
				fmt.Printf("gosh golly what a cool input!\n")
			}
			subSet[val.key] += val.tf
		case <-t.reducerQuit:
			break out
		}
	}

	bloomFilterVals := t.calculateBucketSizes(subSet)
	t.bloomSizeChan <- bloomFilterVals

}

func (t *TermFrequencyCalculator) bloomFilterInitializer() {
	sem := t.numReducers
	b := make(map[BloomFrequencyBucket]uint)
out:
	for {
		select {
		case a := <-t.bloomSizeChan:
			sem--
			b[Below100] += a.ltHunredbucketSize
			b[Below1000] += a.ltOneKbucketSize
			b[Below10000] += a.ltTenKbucketSize
			b[Below100000] += a.ltHundredKBucketSize
		default:
			if sem == 0 {
				break out
			}

		}
	}
	close(t.bloomSizeChan)
	t.bloomFilterManager.InitFreqBuckets(b)

}

func (t *TermFrequencyCalculator) calculateBucketSizes(resultMap map[string]int) bucketVals {
	//this function does not need parallelization, since the number of words in the english language is constant
	var b bucketVals
	for _, size := range resultMap {
		switch {
		case size < 100:
			b.ltHunredbucketSize++
		case 100 < size && size < 1000:
			b.ltOneKbucketSize++
		case 1000 < size && size < 10000:
			b.ltTenKbucketSize++
		case 10000 < size && size < 100000:
			b.ltHundredKBucketSize++
		default:
		}
	}
	return b
}

func (t *TermFrequencyCalculator) populateBloomFilters() {
	// Block and wait until the bloom filters have been created.
	t.bloomFilterManager.WaitForBloomFreqInit()

	//while this approach is kind of verbose, it avoids the expense
	//of millions of allocations
	ltHundredSlice := make([]string, t.ltHunredbucketSize)
	ltOneKSlice := make([]string, t.ltOneKbucketSize)
	ltTenKSlice := make([]string, t.ltTenKbucketSize)
	ltHundredKSlice := make([]string, t.ltHundredKBucketSize)
	ltHundredIndex := 0
	ltOneKIndex := 0
	ltTenKSIndex := 0
	ltHundredKIndex := 0
	for word, size := range t.ResultMap {
		switch {

		case size < 100:
			ltHundredSlice[ltHundredIndex] = word
			ltHundredIndex++
		case 100 < size && size < 1000:
			ltOneKSlice[ltOneKIndex] = word
			ltOneKIndex++
		case 1000 < size && size < 10000:
			ltTenKSlice[ltTenKSIndex] = word
			ltTenKSIndex++
		case 10000 < size && size < 100000:
			ltHundredKSlice[ltHundredIndex] = word
			ltHundredKIndex++
		}

		if ltHundredIndex == 10000 {
			z := ltHundredSlice[:ltHundredIndex]
			ltHundredSlice = ltHundredSlice[ltHundredIndex:]
			ltHundredIndex = 0
			t.bloomFilterManager.QueueFreqBucketAdd(Below100, z)
		}
		if ltOneKIndex == 10000 {
			z := ltOneKSlice[:ltOneKIndex]
			ltOneKSlice = ltOneKSlice[ltOneKIndex:]
			ltOneKIndex = 0
			t.bloomFilterManager.QueueFreqBucketAdd(Below1000, z)
		}
		if ltTenKSIndex == 10000 {
			z := ltTenKSlice[:ltTenKSIndex]
			ltTenKSlice = ltTenKSlice[ltTenKSIndex:]
			ltTenKSIndex = 0
			t.bloomFilterManager.QueueFreqBucketAdd(Below10000, z)
		}
		if ltHundredKIndex == 10000 {
			z := ltHundredKSlice[:ltHundredKIndex]
			ltHundredKSlice = ltHundredKSlice[:ltHundredKIndex]
			ltHundredKIndex = 0
			t.bloomFilterManager.QueueFreqBucketAdd(Below100000, z)
		}
	}
	//handle leftovers
	t.bloomFilterManager.QueueFreqBucketAdd(Below100, ltHundredSlice)
	t.bloomFilterManager.QueueFreqBucketAdd(Below1000, ltOneKSlice)
	t.bloomFilterManager.QueueFreqBucketAdd(Below10000, ltTenKSlice)
	t.bloomFilterManager.QueueFreqBucketAdd(Below100000, ltHundredKSlice)

}
