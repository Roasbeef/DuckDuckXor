package main

import (
	"bytes"
	"fmt"
	"io"
	"sync"
	"testing"
)

// creating the doc list will assume that CorpusReader
// is already working correctly

func TestFrequencyWorker(t *testing.T) {
	corpusreader := NewCorpusReader("./test_directory/", nil)
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
	tf := NewTermFrequencyCalculator(1, terms, nil, nil)
	tf.wg.Add(1)
	go tf.frequencyWorker()
	m1 := <-tf.TermFreq
	if m1["yummy"] != 3 {
		t.Error("map1 did not return correct term frequencies, \n gave answer", m1["yummy"])
	}
	m2 := <-tf.TermFreq
	if m2["yummy"] != 1 {
		t.Error("map2 did not return correct term frequencies, \n gave answer", m2["yummy"])
	}
}

func TestShuffler(t *testing.T) {

	var wg sync.WaitGroup
	fmt.Printf("testing shuffler\n")
	a := make(map[string]int)
	a["hello"] = 5
	a["goodbye"] = 4
	a["golly"] = 3
	tf := NewTermFrequencyCalculator(1, nil, nil, nil)
	tf.initShufflers()
	fmt.Printf("shufflers initialized\n")
	tf.TermFreq <- a

	fmt.Printf("reading shuffle values\n")
	wg.Add(3)
	go func() {
		word := <-tf.reduceMap[Hash("hello")%tf.numReducers]
		if word.key != "hello" || word.tf != 5 {
			t.Error("recieved incorrect value from map: expected hello but got " + word.key)
		}
		wg.Done()
	}()
	go func() {
		word := <-tf.reduceMap[Hash("goodbye")%tf.numReducers]
		if word.key != "goodbye" || word.tf != 4 {
			t.Error("recieved incorrect value from map: expected goodbye but got " + word.key)
		}
		wg.Done()
	}()

	go func() {
		word := <-tf.reduceMap[Hash("golly")%tf.numReducers]
		if word.key != "golly" || word.tf != 3 {
			t.Error("recieved incorrect value from map: expected golly  but got " + word.key)
		}
		wg.Done()
	}()
	wg.Wait()
	close(tf.TermFreq)

}

func TestReducer(t *testing.T) {
	fmt.Printf("starting reducer test\n")
	a := make(map[string]int)
	b := make(map[string]int)
	a["hello"] = 500
	a["goodbye"] = 40000
	a["golly"] = 3
	b["golly"] = 6
	b["camel"] = 54
	tf := NewTermFrequencyCalculator(1, nil, nil, nil)
	tf.initShufflers()
	tf.initReducers()
	tf.TermFreq <- a
	tf.TermFreq <- b
	close(tf.TermFreq)
	b100 := uint(0)
	for i := 0; i < 26; i++ {
		a := <-tf.bloomSizeChan
		b100 += a.ltHunredbucketSize
	}
	if b100 != 2 {
		t.Error("incorrect below100 size: expected 2 but got", b100)
	}
}
