package main

import (
	"testing"
)

// creating the doc list will assume that CorpusReader
// is already working correctly
func BuildDocumentChannel() chan *document {
	c := NewCorpusReader("./testDirectory")
	c.Start()
	return c.DocOut
}

func TestFrequencyWorker(t *testing.T) {
	//	doc := BuildDocumentChannel()
	//	freq := NewTermFrequencyCalculator(1, doc)

	t.Error("stub")
}

func TestLiteralMapReducer(t *testing.T) {
	t.Error("stub")
}
