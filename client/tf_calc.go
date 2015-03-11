package main

import (
	"strconv"
	"sync"
	"sync/atomic"
)

var letters = [26]string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l", "m", "n", "o", "p", "q", "r", "s", "t", "u", "v", "w", "x", "y", "z"}

type wordPair struct {
	word string
	tf   int
}

type bucketVals struct {
	ltHunredbucketSize   int
	ltOneKbucketSize     int
	ltTenKbucketSize     int
	ltHundredKBucketSize int
}
type TermFrequencyCalculator struct {
	quit               chan struct{}
	numWorkers         int
	shuffleMap         map[string]chan string
	reduceMap          map[int]chan wordPair
	bloomSizeChan      chan bucketVals
	bloomPopulateChan  chan bucketVals
	bloomInitChan      chan int
	numActiveWorkers   int32
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
	numReducers        int
	sync.Mutex
	ltHundredKBucketSize uint
	bloomFilterManager   *bloomMaster
}

//TermFreq shoud have as many buffers as workers
func NewTermFrequencyCalculator(numWorkers int, d chan []string, bm *bloomMaster) TermFrequencyCalculator {
	q := make(chan struct{})
	termFreq := make(chan map[string]int, numWorkers)
	s := make(map[string]chan string)
	r := make(map[int]chan wordPair)
	size := make(chan bucketVals)
	populate := make(chan bucketVals)
	for i := 0; i < 26; i++ {
		s[letters[i]] = make(chan string)
		r[i] = make(chan wordPair)
	}
	return TermFrequencyCalculator{quit: q, numWorkers: numWorkers, shuffleMap: s, reduceMap: r, bloomSizeChan: size, bloomPopulateChan: populate, bloomInitChan: make(chan int), TermFreq: termFreq, docIn: d, bloomFilterManager: bm}
}

func (t *TermFrequencyCalculator) Start() error {
	if atomic.AddInt32(&t.started, 1) != 1 {
		return nil
	}
	t.wg.Add(4)
	t.numReducers = 26
	go t.initReducers()
	for i := 0; i < t.numWorkers; i++ {
		t.numActiveWorkers++
		go t.frequencyWorker()
	}
	//go t.calculateBucketSizes()
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

func (t *TermFrequencyCalculator) initReducers() {
	for i := 0; i < t.numReducers; i++ {
		go t.reducer(i)
	}
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
	atomic.AddInt32(&t.numActiveWorkers, -1)
	t.wg.Done()
}

func (t *TermFrequencyCalculator) shuffler() {

out:
	for {
		select {
		case <-t.quit:
			break out
		case doc := <-t.TermFreq:
			for key, val := range doc {
				hashValue, err := strconv.Atoi(key)
				if err != nil {
					//TODO error handling
				}
				hashValue = hashValue % t.numReducers
				t.reduceMap[hashValue] <- wordPair{key, val}
			}
		default:
			if t.numActiveWorkers == 0 {
				break out
			}
		}

	}
	close(t.TermFreq)
}

func (t *TermFrequencyCalculator) reducer(key int) {
	count := 0
	input := t.reduceMap[key]
	subSet := make(map[string]int)
out:
	for {
		select {
		case <-t.quit:
			break out
		case val := <-input:
			if subSet[val.word] == 0 {
				count++
			}
			subSet[val.word] += val.tf
		}
	}

	bloomFilterVals := t.calculateBucketSizes(subSet)
	t.bloomSizeChan <- bloomFilterVals

}

func (t *TermFrequencyCalculator) bloomFilterInitializer() {
	sem := 26
	var b bucketVals
out:
	for {
		select {
		case a := <-t.bloomSizeChan:
			sem--
			b.ltHunredbucketSize += a.ltHunredbucketSize
			b.ltOneKbucketSize += a.ltOneKbucketSize
			b.ltTenKbucketSize += a.ltTenKbucketSize
			b.ltHundredKBucketSize += a.ltHundredKBucketSize
		default:
			if sem == 0 {
				break out
			}

		}
	}
	close(t.bloomSizeChan)
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
