package main

import (
	"fmt"
	"sync"
)

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
	reduceMap           map[uint32]chan wordPair
	mostRecentDoc       map[string]int
	ResultCount         int
	docIn               chan *InvIndexDocument
	wg                  sync.WaitGroup
	started             int32
	numWorkers          uint32
	numReducers         uint32
	beta_indeces        chan wordPair
	mappersDone         chan struct{}
	shufflersDone       chan struct{}
	shufflerQuit        chan struct{}
	reducerQuit         chan struct{}
	abort               func(chan struct{}, error)
	bloomMaster         *bloomMaster
}

func (i *InvertedIndexCalculator) NewInvertedIndexCalculator(docs chan *InvIndexDocument, numWorkers uint32, numReducers uint32, abort func(chan struct{}, error)) InvertedIndexCalculator {
	q := make(chan struct{})
	r := make(map[uint32]chan wordPair)
	var j uint32
	for j = 0; j < 26; j++ {
		r[j] = make(chan wordPair)
	}
	finalIndexEntries := make(chan map[string]uint32, numWorkers)
	return InvertedIndexCalculator{quit: q,
		finalIndexEntries: finalIndexEntries,
		docIn:             docs,
		reduceMap:         r,
		mappersDone:       make(chan struct{}, numWorkers),
		shufflersDone:     make(chan struct{}, numReducers),
		shufflerQuit:      make(chan struct{}),
		reducerQuit:       make(chan struct{}),
		abort:             abort,
		beta_indeces:      make(chan wordPair),
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
	<-i.mappersDone
	if len(i.mappersDone) == 0 {
		close(i.shufflerQuit)
	}
	i.wg.Done()
}

func maxInt(a uint32, b uint32) uint32 {

	if a > b {
		return a
	}
	return b
}

func (i *InvertedIndexCalculator) finalIndexShuffler() {
out:
	for {
		select {
		case <-i.quit:
			break out
		case a := <-i.finalIndexEntries:
			for key, val := range a {
				hashValue := Hash(key)
				hashValue = hashValue % i.numReducers
				i.reduceMap[hashValue] <- wordPair{key, int(val)}
			}
		case <-i.shufflerQuit:
			break out
		}
	}
	<-i.shufflersDone
	if len(i.shufflersDone) == 0 {
		close(i.reducerQuit)
	}
	i.wg.Done()
}

func (i *InvertedIndexCalculator) reducer(key uint32) {

	count := 0
	input := i.reduceMap[key]
	subSet := make(map[string]int)
out:
	for {
		select {
		case <-i.quit:
			break out
		case val := <-input:
			//TODO why am I doing this count?
			if subSet[val.key] == 0 {
				count++
			}
			if val.key == "golly" {
				fmt.Printf("gosh golly what a cool input!\n")
			}

			//TODO:I realize that this is HORRIBLE. I am keeping this here until after yang
			//functionally it will work, but this wont stay here
			subSet[val.key] = int(maxInt(uint32(subSet[val.key]), uint32(val.tf)))
		case <-i.reducerQuit:
			break out
		}
	}

}
