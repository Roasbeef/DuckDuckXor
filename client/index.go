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

func (i *indexer) Index(homeIndex string, db *bolt.DB) (*bloomMaster, *KeyManager, map[uint32]string) {
	eHandler := NewErrorHandler()
	cReader := NewCorpusReader(homeIndex, eHandler.createAbortFunc())
	eHandler.stopChan <- cReader.Stop
	preProcessor := NewDocPreprocessor(cReader.DocOut, eHandler.createAbortFunc())
	eHandler.stopChan <- preProcessor.Stop
	//TODO set up boltdb and make numWorkers a config
	bloomMaster, err := newBloomMaster(nil, 1)
	if err != nil {
		//TODO handle error
	}
	keyManager, err := NewKeyManager(db, nil)
	if err != nil {
		//TODO handle error
	}
	//TODO make numWorkers a config
	tfCalc := NewTermFrequencyCalculator(100, preProcessor.TfOut, bloomMaster, eHandler.createAbortFunc())
	eHandler.stopChan <- tfCalc.Stop
	cReader.Start()
	preProcessor.Start()
	bloomMaster.Start()
	tfCalc.Start()

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
