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

func TestShuffler(t *testing.T) {
	fmt.Printf("starting shuffle test\n")
	a := make(map[string]int)
	a["hello"] = 5
	a["goodbye"] = 4
	a["golly"] = 3

	tf := NewTermFrequencyCalculator(1, nil, nil)
	tf.wg.Add(1)
	go tf.shuffler()
	tf.TermFreq <- a
	fmt.Printf("pulling value\n")
	fmt.Printf("hash of hello is: ", (Hash("hello") % tf.numReducers), "\n")
	word := <-tf.reduceMap[Hash("hello")%tf.numReducers]
	if word.word != "hello" || word.tf != 5 {
		t.Error("recieved incorrect value from map: expected hello but got " + word.word)
	}
	fmt.Printf("sasdgadsgads\n")
	word = <-tf.reduceMap[Hash("goodbye")%tf.numReducers]
	if word.word != "goodbye" || word.tf != 4 {
		t.Error("recieved incorrect value from map: expected goodbye but got " + word.word)
	}

}
