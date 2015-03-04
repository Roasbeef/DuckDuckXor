package main

import (
	"sync"
	"sync/atomic"
)

type TermFrequencyCalculator struct {
	quit                 chan struct{}
	numWorkers           int
	TermFreq             chan map[string]int
	docIn                chan []string
	wg                   sync.WaitGroup
	ResultMap            map[string]int
	started              int32
	shutDown             int32
	err                  chan error
	ltHunredbucketSize   int32
	ltOneKbucketSize     int32
	ltTenKbucketSize     int32
	ltHundredKBucketSize int32
}

//TermFreq shoud have as many buffers as workers
func NewTermFrequencyCalculator(numWorkers int, d chan []string) TermFrequencyCalculator {
	q := make(chan struct{})

	termFreq := make(chan map[string]int, numWorkers)
	return TermFrequencyCalculator{quit: q, numWorkers: numWorkers, TermFreq: termFreq, docIn: d}
}

func (t *TermFrequencyCalculator) Start() error {
	if atomic.AddInt32(&t.started, 1) != 1 {
		return nil
	}
	t.wg.Add(2)

	go t.frequencyWorker()
	go t.literalMapReducer()
	go t.bucketSorter()
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
	//TODO scale horizontally (find log answer)
	//keep track of how many are <100, 100-1000, 1000-10000, 10000-100000, 100k+
	masterMap := <-t.TermFreq
	for i := 0; i < t.numWorkers-1; i++ {
		tempMap := <-t.TermFreq
		for k := range tempMap {
			masterMap[k] += tempMap[k]
		}
	}
	t.ResultMap = masterMap
	for _, size := range masterMap {
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
}

func (t *TermFrequencyCalculator) bucketSorter() {

	//TODO after sending bucket sizes, wait for info
	//from lalu stating that the buckets are created
	//at which point you can

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

		}
		if ltOneKIndex == 10000 {
			z := ltOneKSlice[:ltOneKIndex]
			ltOneKSlice = ltOneKSlice[ltOneKIndex:]
			ltOneKIndex = 0
		}
		if ltTenKSIndex == 10000 {
			z := ltTenKSlice[:ltTenKSIndex]
			ltTenKSlice = ltTenKSlice[ltTenKSIndex:]
			ltTenKSIndex = 0
		}
		if ltHundredKIndex == 10000 {
			z := ltHundredKSlice[:ltHundredKIndex]
			ltHundredKSlice = ltHundredKSlice[:ltHundredKIndex]
			ltHundredKIndex = 0
		}
	}

}
