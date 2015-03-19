package main

import (
	"sync"

	"github.com/boltdb/bolt"
)

//index should handle high level indexing

type indexer struct {
	docNames    map[uint32]string
	nameChannel chan map[uint32]string
	mainWg      sync.WaitGroup
}

func NewIndexer() *indexer {

	return &indexer{

		docNames: make(map[uint32]string),
	}
}

func (i *indexer) Index(homeIndex string, db *bolt.DB, numWorkers int) (*bloomMaster, *KeyManager, map[uint32]string) {
	eHandler := NewErrorHandler()
	cReader := NewCorpusReader(homeIndex, eHandler.createAbortFunc())
	eHandler.stopChan <- cReader.Stop
	preProcessor := NewDocPreprocessor(cReader.DocOut, eHandler.createAbortFunc())
	eHandler.stopChan <- preProcessor.Stop
	//TODO every one of these classes should take in the main wg as a parameter
	bloomMaster, err := newBloomMaster(nil, numWorkers, &i.mainWg)
	if err != nil {
		//TODO handle error
	}
	keyManager, err := NewKeyManager(db, nil, &i.mainWg)
	if err != nil {
		//TODO handle error
	}
	tfCalc := NewTermFrequencyCalculator(uint32(numWorkers), preProcessor.TfOut, bloomMaster, eHandler.createAbortFunc(), &i.mainWg)
	eHandler.stopChan <- tfCalc.Stop

	go cReader.Start()
	go preProcessor.Start()
	go bloomMaster.Start()
	go keyManager.Start()
	go tfCalc.Start()

	i.mainWg.Wait()
	return bloomMaster, keyManager, i.docNames
}

func (i *indexer) gatherDocumentNames(dChan chan Doc) {
	for {
		select {
		case doc, more := <-dChan:
			if !more {
				break
			}
			i.docNames[doc.DocId] = doc.Name
		}
	}
}

func AddToWg(local sync.WaitGroup, main *sync.WaitGroup, val int) {
	local.Add(val)
	main.Add(val)
}

func WgDone(local sync.WaitGroup, main *sync.WaitGroup) {
	local.Done()
	main.Done()
}
