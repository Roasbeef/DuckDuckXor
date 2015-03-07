package main

import (
	"sync"
	"sync/atomic"
)

type TermFrequencyCalculator struct {
	quit               chan struct{}
	numWorkers         int
	TermFreq           chan map[string]int
	docIn              chan []string
	wg                 sync.WaitGroup
	ResultMap          map[string]int
	started            int32
	shutDown           int32
	err                chan error
	ltHunredbucketSize uint
	ltOneKbucketSize   uint
	ltTenKbucketSize   uint
	sync.Mutex
	ltHundredKBucketSize uint
	bloomFilterManager   *bloomMaster
}

//TermFreq shoud have as many buffers as workers
func NewTermFrequencyCalculator(numWorkers int, d chan []string, bm *bloomMaster) TermFrequencyCalculator {
	q := make(chan struct{})
	termFreq := make(chan map[string]int, numWorkers)
	return TermFrequencyCalculator{quit: q, numWorkers: numWorkers, TermFreq: termFreq, docIn: d, bloomFilterManager: bm}
}

func (t *TermFrequencyCalculator) Start() error {
	if atomic.AddInt32(&t.started, 1) != 1 {
		return nil
	}
	t.wg.Add(4)
	go t.frequencyWorker()
	go t.literalMapReducer()
	go t.initBloomFilterBuckets()
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

func (t *TermFrequencyCalculator) frequencyWorker() {
	m := make(map[string]int)
out:
	for {
		select {
		case <-t.quit:
			break out
		case doc, ok := <-t.docIn:
			if !ok {
				break out
			}
			for _, token := range doc {
				m[token] = m[token] + 1
			}
			doc = nil
		}
	}
	t.TermFreq <- m
	close(t.TermFreq)
	t.wg.Done()
}

func (t *TermFrequencyCalculator) literalMapReducer() {
	t.wg.Wait()
	var wg sync.WaitGroup
	//TODO scale horizontally (find log answer)
	//keep track of how many are <100, 100-1000, 1000-10000, 10000-100000, 100k+
	masterMap := <-t.TermFreq
	for i := 0; i < t.numWorkers-1; i++ {
		wg.Add(1)
		go func() {
			tempMap := <-t.TermFreq
			for k := range tempMap {
				t.Lock()
				masterMap[k] += tempMap[k]
				t.Unlock()
			}
			wg.Done()
		}()
	}
	wg.Wait()
	t.ResultMap = masterMap
}

func (t *TermFrequencyCalculator) initBloomFilterBuckets() {
	//this function does not need parallelization, since the number of words in the english language is constant
	for _, size := range t.ResultMap {
		switch {
		case size < 100:
			t.ltHunredbucketSize++
		case 100 < size && size < 1000:
			t.ltOneKbucketSize++
		case 1000 < size && size < 10000:
			t.ltTenKbucketSize++
		case 10000 < size && size < 100000:
			t.ltHundredKBucketSize++
		default:
		}
	}
	bfMap := make(map[BloomFrequencyBucket]uint)
	bfMap[Below100] = t.ltHunredbucketSize
	bfMap[Below1000] = t.ltOneKbucketSize
	bfMap[Below10000] = t.ltTenKbucketSize
	bfMap[Below100000] = t.ltHundredKBucketSize
	t.bloomFilterManager.InitFreqBuckets(bfMap)
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
