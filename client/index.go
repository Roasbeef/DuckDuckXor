package main

//index should handle high level indexing

func Index(homeIndex string) {
	eHandler := NewErrorHandler()
	cReader := NewCorpusReader(homeIndex, eHandler.createAbortFunc())
	eHandler.stopChan <- cReader.Stop
	preProcessor := NewDocPreprocessor(cReader.DocOut, eHandler.createAbortFunc())
	eHandler.stopChan <- preProcessor.Stop
}
