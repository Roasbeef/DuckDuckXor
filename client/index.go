package main

//index should handle high level indexing

func Index(homeIndex string) {
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
