package main

import "sync"

type TSetUpdateMessage struct {
	docID int32
	words map[string]struct{}
}

type InvertedIndexCalculator struct {
	quit                chan struct{}
	ResultInvIndex      map[string]int32
	finalIndexEntries   chan map[string]int32
	finalCounterEntries chan int
	ResultCount         int
	docIn               chan *InvIndexDocument
	wg                  sync.WaitGroup
	started             int32
	numWorkers          int
}

func (i *InvertedIndexCalculator) NewInvertedIndexCalculator(docs chan *InvIndexDocument, numWorkers int) InvertedIndexCalculator {
	q := make(chan struct{})

	finalIndexEntries := make(chan map[string]int32, numWorkers)
	return InvertedIndexCalculator{quit: q, finalIndexEntries: finalIndexEntries, docIn: docs, numWorkers: numWorkers}

}

func (i *InvertedIndexCalculator) finalIndexEntriesWorker() {
	mostRecentDoc := make(map[string]int32)
	counter := 0
	//this count assures that if mostRecentDoc gets too large we perform GC
out:
	for {
		select {
		case <-i.quit:
			break out
		case doc := <-i.docIn:
			currentID := doc.DocId
			for token := range doc.Words {
				mostRecentDoc[token] = maxInt(mostRecentDoc[token], currentID)
				counter++
			}
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
	//TODO create a counter that returns total number of word->docID pairs
	//also add logging
	masterMap := <-i.finalIndexEntries
	masterCounter := 0
	for j := 0; j < i.numWorkers-1; j++ {
		tempMap := <-i.finalIndexEntries
		tempCount := <-i.finalCounterEntries
		for k := range tempMap {
			masterMap[k] = maxInt(masterMap[k], tempMap[k])
		}
		masterCounter += tempCount
	}
	i.ResultInvIndex = masterMap
	i.ResultCount = masterCounter
	return i.ResultInvIndex
}
