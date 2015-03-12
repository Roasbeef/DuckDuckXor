package main

import "sync"

type TSetUpdateMessage struct {
	docID int32
	words map[string]struct{}
}

type InvertedIndexCalculator struct {
	quit                chan struct{}
	ResultInvIndex      map[string]uint32
	finalIndexEntries   chan map[string]uint32
	finalCounterEntries chan int
	TsetIndex           map[string]int
	mostRecentDoc       map[string]int
	ResultCount         int
	docIn               chan *InvIndexDocument
	wg                  sync.WaitGroup
	started             int32
	numWorkers          int
	bloomMaster         *bloomMaster
}

func (i *InvertedIndexCalculator) NewInvertedIndexCalculator(docs chan *InvIndexDocument, numWorkers int) InvertedIndexCalculator {
	q := make(chan struct{})

	finalIndexEntries := make(chan map[string]uint32, numWorkers)
	return InvertedIndexCalculator{quit: q, finalIndexEntries: finalIndexEntries, docIn: docs, numWorkers: numWorkers}

}

func (i *InvertedIndexCalculator) finalIndexEntriesWorker() {
	lastTermInDocument := make(map[string]uint32)
	counter := 0
	//this count assures that if mostRecentDoc gets too large we perform GC
out:
	for {
		select {
		case <-i.quit:
			break out
		case doc, more := <-i.docIn:
			if !more {
				break out
			}
			currentID := doc.DocId
			for token := range doc.Words {
				lastTermInDocument[token] = maxInt(lastTermInDocument[token], currentID)
				counter++
			}
		}
	}
	i.finalCounterEntries <- counter
	i.finalIndexEntries <- lastTermInDocument
	i.wg.Done()
}

func maxInt(a uint32, b uint32) uint32 {

	if a > b {
		return a
	}
	return b
}

func (i *InvertedIndexCalculator) invIndexMapReduce() {
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
	//TODO rename ResultInvIndex

	i.ResultInvIndex = masterMap
	i.bloomMaster.InitXSet(uint(masterCounter))
}
