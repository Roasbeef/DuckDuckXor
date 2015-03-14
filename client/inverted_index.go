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
	numWorkers          uint32
	numReducers         uint32
	mappersDone         chan struct{}
	shufflersDone       chan struct{}
	shufflerQuit        chan struct{}
	reducerQuit         chan struct{}
	abort               func(chan struct{}, error)
	bloomMaster         *bloomMaster
}

func (i *InvertedIndexCalculator) NewInvertedIndexCalculator(docs chan *InvIndexDocument, numWorkers uint32, numReducers uint32, abort func(chan struct{}, error)) InvertedIndexCalculator {
	q := make(chan struct{})

	finalIndexEntries := make(chan map[string]uint32, numWorkers)
	return InvertedIndexCalculator{quit: q,
		finalIndexEntries: finalIndexEntries,
		docIn:             docs,
		mappersDone:       make(chan struct{}, numWorkers),
		shufflersDone:     make(chan struct{}, numReducers),
		shufflerQuit:      make(chan struct{}),
		reducerQuit:       make(chan struct{}),
		abort:             abort,
		numWorkers:        numWorkers,
		numReducers:       numReducers,
	}

}

func (i *InvertedIndexCalculator) initMappers() {
	for j := uint32(0); j < i.numWorkers; j++ {
		i.wg.Add(1)
		i.mappersDone <- struct{}{}
		go i.finalIndexEntriesWorker()
	}
}

func (i *InvertedIndexCalculator) initReducers() {
	for j := uint32(0); j < i.numReducers; j++ {
		i.wg.Add(1)
		go i.reducer(j)
	}
}

func (i *InvertedIndexCalculator) initShufflers() {
	for j := uint32(0); j < i.numReducers; j++ {
		i.wg.Add(1)
		i.shufflersDone <- struct{}{}
		go i.finalIndexShuffler()
	}
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

func (i *InvertedIndexCalculator) finalIndexShuffler() {

}

func (i *InvertedIndexCalculator) reducer(key uint32) {

}
