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
	abortFunc := eHandler.createAbortFunc()

	cReader := NewCorpusReader(homeIndex, abortFunc)
	eHandler.stopChan <- cReader.Stop

	preProcessor := NewDocPreprocessor(cReader.DocOut, abortFunc)
	eHandler.stopChan <- preProcessor.Stop

	bloomMaster, err := newBloomMaster(nil, numWorkers, &i.mainWg, abortFunc)
	if err != nil {
		//TODO handle error
	}
	eHandler.stopChan <- bloomMaster.Stop

	keyManager, err := NewKeyManager(db, nil, &i.mainWg, abortFunc)
	if err != nil {
		//TODO handle error
	}
	eHandler.stopChan <- keyManager.Stop

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

func AddToWg(local *sync.WaitGroup, main *sync.WaitGroup, val int) {
	local.Add(val)
	main.Add(val)
}

func WgDone(local *sync.WaitGroup, main *sync.WaitGroup) {
	local.Done()
	main.Done()
}
