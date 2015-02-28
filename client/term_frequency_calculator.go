package main

import (
	"bytes"
	"io"
	"strings"
	"sync"

	"sync/atomic"
)

type TermFrequencyCalculator struct {
	quit       chan struct{}
	numWorkers int
	termFreq   chan map[string]int
	docIn      chan *document
	wg         sync.WaitGroup
	started    int32
	shutDown   int32
}

//termFreq shoud have as many buffers as workers
func (t *TermFrequencyCalculator) NewTermFrequencyCalculator(numWorkers int, d chan *document) TermFrequencyCalculator {
	q := make(chan struct{})

	termFreq := make(chan map[string]int, numWorkers)
	return TermFrequencyCalculator{quit: q, numWorkers: numWorkers, termFreq: termFreq, docIn: d}
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
	var bit []byte
	b := bytes.NewBuffer(bit)

	m := make(map[string]int)
out:
	for doc := range t.docIn {
		select {
		case <-t.quit:
			break out
		default:
			io.Copy(b, doc)
			contents := string(b.Bytes())
			tokens := strings.Split(contents, " ")
			for _, token := range tokens {
				m[token] = m[token] + 1
			}
			b.Reset()
		}
	}
	t.termFreq <- m
	t.wg.Done()
}

func (t *TermFrequencyCalculator) literalMapReducer() {
	t.wg.Wait()
	masterMap := <-t.termFreq
	for i := 0; i < t.numWorkers-1; i++ {
		tempMap := <-t.termFreq
		for k := range tempMap {
			masterMap[k] += tempMap[k]
		}
	}
}
