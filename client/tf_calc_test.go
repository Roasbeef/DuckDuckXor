package main

import
//"fmt"

(
	"fmt"
	"testing"
)

// creating the doc list will assume that CorpusReader
// is already working correctly

func TestFrequencyWorker(t *testing.T) {
	fmt.Printf("abcd\n")
	corpusreader := NewCorpusReader("./test_directory/")
	corpusreader.Start()
	preprocess := NewDocPreprocessor(corpusreader.DocOut)
	preprocess.Start()
	tf := NewTermFrequencyCalculator(1, preprocess.TfOut, nil)
	tf.wg.Add(1)
	fmt.Printf("start TfWorker")
	go tf.frequencyWorker()
	<-preprocess.InvIndexOut
	<-preprocess.DocEncryptOut
	<-preprocess.InvIndexOut
	<-preprocess.DocEncryptOut
	m1 := <-tf.TermFreq
	if m1["yummy"] != 4 {
		// 	//TODO ask lalu about actors hnadling multiple files
		t.Error("map1 did not return correct term frequencies, \n gave answer", m1["yummy"])
	}
}

func TestLiteralMapReducer(t *testing.T) {

	corpusreader := NewCorpusReader("./test_directory/testFile1.txt")
	corpusreader.Start()
	preprocess := NewDocPreprocessor(corpusreader.DocOut)
	preprocess.Start()

	// c := NewCorpusReader("./test_directory")
	// c.Start()
	// p := NewDocPreprocessor(c.DocOut)
	// p.Start()
	// tf := NewTermFrequencyCalculator(2, p.TfOut)
	// tf.wg.Add(1)
	// tf.frequencyWorker()
	// a := tf.literalMapReducer()
	// if a["yummy"] != 4 {
	// 	t.Error("map did not reduce to a sum of values. gave value:", a["yummy"])
	// }
}

func TestBucketSorter(t *testing.T) {
	m := make(map[string]int)
	m["a"] = 30
	m["b"] = 40
	m["c"] = 50

	m["aa"] = 300
	m["bb"] = 400

	m["aaa"] = 2001
	m["bbb"] = 2002
	m["ccc"] = 2003
	m["ddd"] = 2004

	m["aaaa"] = 23001
	m["bbbb"] = 22002
	m["cccc"] = 24003

	m["dddd"] = 32004
	m["eeee"] = 32004

	tf := NewTermFrequencyCalculator(4, nil, nil)
	tf.bucketSorter()

}
