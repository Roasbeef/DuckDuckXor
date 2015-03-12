package main

import
//"fmt"

(
	"bytes"
	"fmt"
	"io"
	"testing"
)

// creating the doc list will assume that CorpusReader
// is already working correctly

func TestFrequencyWorker(t *testing.T) {
	corpusreader := NewCorpusReader("./test_directory/")
	corpusreader.Start()
	terms := make(chan []string, 2)
	var bit []byte
	b := bytes.NewBuffer(bit)
out:
	for {
		select {
		case doc, ok := <-corpusreader.DocOut:
			if !ok {
				break out
			}
			_, err := io.Copy(b, doc)
			if err != nil {
				//TODO error handling
			}
			fmt.Println(string(b.Bytes()))
			fmt.Println("\n")
			fmt.Println("\n")
			fmt.Println("\n")
			terms <- ParseTokens(string(b.Bytes()))
			b.Reset()
		}
	}
	tf := NewTermFrequencyCalculator(1, terms, nil)
	tf.wg.Add(1)
	go tf.frequencyWorker()
	m1 := <-tf.TermFreq
	if m1["yummy"] != 3 {
		// 	//TODO ask lalu about actors hnadling multiple files
		t.Error("map1 did not return correct term frequencies, \n gave answer", m1["yummy"])
	}
	m2 := <-tf.TermFreq
	if m2["yummy"] != 1 {
		t.Error("map2 did not return correct term frequencies, \n gave answer", m2["yummy"])
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
