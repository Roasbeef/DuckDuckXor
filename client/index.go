package main

//index should handle high level indexing

type indexer struct {
	docNames    map[uint32]string
	nameChannel chan map[uint32]string
}

func NewIndexer() *indexer {

	return &indexer{

		docNames:    make(map[uint32]string),
		nameChannel: make(chan map[uint32]string),
	}
}

func (i *indexer) Index(homeIndex string) {
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
	//TODO make numWorkers a config
	tfCalc := NewTermFrequencyCalculator(100, preProcessor.TfOut, bloomMaster, eHandler.createAbortFunc())
	eHandler.stopChan <- tfCalc.Stop
	cReader.Start()
	preProcessor.Start()
	bloomMaster.Start()
	tfCalc.Start()
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
	//the reason I'm using this channel is so the client wont get the map until it is ready
	i.nameChannel <- i.docNames
	close(i.nameChannel)
}
