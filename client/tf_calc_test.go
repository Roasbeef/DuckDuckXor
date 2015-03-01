package main

import
//"fmt"

"testing"

// creating the doc list will assume that CorpusReader
// is already working correctly
func BuildDocumentChannel() chan *document {
	c := NewCorpusReader("./test_directory")
	c.Start()
	return c.DocOut
}

func TestFrequencyWorker(t *testing.T) {
	//b := BuildDocumentChannel()
	c := NewCorpusReader("./test_directory/testFile1.txt")
	c.Start()
	tf := NewTermFrequencyCalculator(2, c.DocOut)
	tf.wg.Add(1)
	err := tf.frequencyWorker()
	if err != nil {
		t.Error("worker returned error: ", err)
	}
	m1 := <-tf.TermFreq
	//m2 := <-tf.TermFreq
	if m1["yummy"] != 3 {
		//TODO ask lalu about actors hnadling multiple files
		t.Error("map1 did not return correct term frequencies, \n gave answer", m1["yummy"])
	}
}

func TestLiteralMapReducer(t *testing.T) {
	c := NewCorpusReader("./test_directory")
	c.Start()
	tf := NewTermFrequencyCalculator(2, c.DocOut)
	tf.wg.Add(1)
	err := tf.frequencyWorker()
	if err != nil {
		t.Error("something went wrong while mapping")
	}
	a := tf.literalMapReducer()
	if a["yummy"] != 4 {
		t.Error("map did not reduce to a sum of values. gave value:", a["yummy"])
	}
}
