package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"

	"sync/atomic"
)

type TermFrequencyCalculator struct {
	quit       chan struct{}
	numWorkers int
	TermFreq   chan map[string]int
	docIn      chan *document
	wg         sync.WaitGroup
	ResultMap  map[string]int
	started    int32
	shutDown   int32
	err        chan error
}

//TermFreq shoud have as many buffers as workers
func NewTermFrequencyCalculator(numWorkers int, d chan *document) TermFrequencyCalculator {
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

func (t *TermFrequencyCalculator) frequencyWorker() error {
	var bit []byte
	b := bytes.NewBuffer(bit)
	m := make(map[string]int)
out:
	for {
		select {
		case <-t.quit:
			break out
		case doc := <-t.docIn:
			if doc == nil {
				//TODO ask lalu if this is kosher
				break out
			}

			fmt.Printf("asgsdfa\n")
			_, err := io.Copy(b, doc)
			if err != nil {
				fmt.Printf("issue with copy\n")
				return err
			}
			contents := string(b.Bytes())
			contents = strings.Replace(contents, "\n", " ", -1)
			tokens := strings.Split(contents, " ")
			for _, token := range tokens {
				m[token] = m[token] + 1
				//			fmt.Printf("value for %s is now %d \n", token, m[token])
			}
			b.Reset()
		}
	}
	t.TermFreq <- m
	close(t.TermFreq)
	t.wg.Done()
	return nil
}

func (t *TermFrequencyCalculator) literalMapReducer() map[string]int {
	t.wg.Wait()
	masterMap := <-t.TermFreq
	fmt.Printf("hooray i'm starting")
	for i := 0; i < t.numWorkers-1; i++ {
		tempMap := <-t.TermFreq
		fmt.Printf("ouch that hurts")
		for k := range tempMap {
			masterMap[k] += tempMap[k]
		}
	}
	t.ResultMap = masterMap
	fmt.Printf("master map created\n")
	return t.ResultMap
}
