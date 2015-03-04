package main

import (
	"fmt"
	"sync"
	"sync/atomic"
)

type TermFrequencyCalculator struct {
	quit       chan struct{}
	numWorkers int
	TermFreq   chan map[string]int
	docIn      chan []string
	wg         sync.WaitGroup
	ResultMap  map[string]int
	started    int32
	shutDown   int32
	err        chan error
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
	fmt.Printf("before loop\n")
out:
	for {
		fmt.Printf("starting loop\n")
		select {
		case <-t.quit:
			break out
		case doc, ok := <-t.docIn:
			if !ok {
				break out
			}
			fmt.Printf("asgd\n")
			for _, token := range doc {
				m[token] = m[token] + 1
				fmt.Printf(token + "\n")
			}
			fmt.Printf("looping again\n")
		}
	}
	fmt.Printf("before channel\n")
	t.TermFreq <- m
	fmt.Printf("method closed")
	close(t.TermFreq)
	t.wg.Done()
}

func (t *TermFrequencyCalculator) literalMapReducer() map[string]int {
	t.wg.Wait()
	masterMap := <-t.TermFreq
	for i := 0; i < t.numWorkers-1; i++ {
		tempMap := <-t.TermFreq
		for k := range tempMap {
			masterMap[k] += tempMap[k]
		}
	}
	t.ResultMap = masterMap
	return t.ResultMap
}
