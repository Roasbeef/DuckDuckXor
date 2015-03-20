package main

import (
	"fmt"
	"log"
	"sync"

	"github.com/boltdb/bolt"
	pb "github.com/roasbeef/DuckDuckXor/protos"
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

func (i *indexer) Index(homeIndex string, db *bolt.DB, numWorkers int, client pb.EncryptedSearchClient, pass []byte) (*bloomMaster, *KeyManager, map[uint32]string) {
	eHandler := NewErrorHandler()
	eHandler.Start()
	abortFunc := eHandler.createAbortFunc()

	cReader := NewCorpusReader(homeIndex, abortFunc)
	eHandler.stopChan <- cReader.Stop

	preProcessor := NewDocPreprocessor(cReader.DocOut, abortFunc)
	eHandler.stopChan <- preProcessor.Stop

	bloomMaster, err := newBloomMaster(db, numWorkers, &i.mainWg, abortFunc, client)
	if err != nil {
		log.Fatal(err)
		//TODO handle error
	}
	eHandler.stopChan <- bloomMaster.Stop

	keyManager, err := NewKeyManager(db, pass, &i.mainWg, abortFunc)
	if err != nil {
		//TODO handle error
		log.Fatal(err)
	}
	eHandler.stopChan <- keyManager.Stop
	keyManager.Start()

	tfCalc := NewTermFrequencyCalculator(uint32(numWorkers), preProcessor.TfOut, bloomMaster, eHandler.createAbortFunc(), &i.mainWg)
	eHandler.stopChan <- tfCalc.Stop

	docKey := keyManager.FetchDocEncKey()
	eDocStreamer := NewEncryptedDocStreamer(numWorkers, docKey, preProcessor.DocEncryptOut, client, abortFunc)

	encryptedIndexer := NewEncryptedIndexGenerator(preProcessor.InvIndexOut, numWorkers, keyManager.keyMap, bloomMaster, client, &i.mainWg, abortFunc)

	go totalKeyWordSummer(&i.mainWg, bloomMaster, preProcessor.KeyWordSummerOut)
	cReader.Start()
	preProcessor.Start()
	bloomMaster.Start()
	tfCalc.Start()
	eDocStreamer.Start()
	encryptedIndexer.Start()
	i.mainWg.Wait()
	return bloomMaster, keyManager, i.docNames
}

func totalKeyWordSummer(wg *sync.WaitGroup, bloom *bloomMaster, keyWordsIn chan *InvIndexDocument) {
	count := 0
	fmt.Println("summing keywords")
out:
	for {
		select {
		case doc, more := <-keyWordsIn:
			if !more {
				break out
			}
			count += len(doc.Words)
		}
	}
	bloom.InitXSet(uint(count))
	fmt.Println("total keywords found:", count)
	wg.Done()
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
