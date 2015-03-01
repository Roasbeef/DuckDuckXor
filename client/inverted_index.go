package main

import (
	"bytes"
	"io"
	"strings"
	"sync"
)

type TSetUpdateMessage struct {
	docID int32
	words map[string]struct{}
}

type InvertedIndexCalculator struct {
	quit              chan struct{}
	ResultInvIndex    map[string]int32
	finalIndexEntries chan map[string]int32

	docIn      chan *document
	wg         sync.WaitGroup
	started    int32
	numWorkers int
}

func (i *InvertedIndexCalculator) NewInvertedIndexCalculator(docs chan *document, numWorkers int) InvertedIndexCalculator {

	q := make(chan struct{})

	finalIndexEntries := make(chan map[string]int32, numWorkers)
	return InvertedIndexCalculator{quit: q, finalIndexEntries: finalIndexEntries, docIn: docs, numWorkers: numWorkers}

}

func (i *InvertedIndexCalculator) finalIndexEntriesWorker() {
	var bit []byte
	b := bytes.NewBuffer(bit)
	mostRecentDoc := make(map[string]int32)
	//this count assures that if mostRecentDoc gets too large we perform GC
out:
	for doc := range i.docIn {
		select {
		case <-i.quit:
			break out
		default:
			io.Copy(b, doc)
			currentID := doc.DocId
			contents := string(b.Bytes())
			tokens := strings.Split(contents, " ")
			for _, token := range tokens {
				mostRecentDoc[token] = maxInt(mostRecentDoc[token], currentID)
			}
			b.Reset()
		}
	}
	i.finalIndexEntries <- mostRecentDoc
	i.wg.Done()
}

func maxInt(a int32, b int32) int32 {

	if a > b {
		return a
	}
	return b
}

func (i *InvertedIndexCalculator) invIndexMapReduce() map[string]int32 {
	i.wg.Wait()
	masterMap := <-i.finalIndexEntries
	for j := 0; j < i.numWorkers-1; j++ {
		tempMap := <-i.finalIndexEntries
		for k := range tempMap {
			masterMap[k] = maxInt(masterMap[k], tempMap[k])
		}
	}
	i.ResultInvIndex = masterMap
	return i.ResultInvIndex
}
