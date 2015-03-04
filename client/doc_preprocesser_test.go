package main

import (
	"fmt"
	"testing"
)

func TestPartitionStreams(t *testing.T) {
	c := NewCorpusReader("./test_directory")
	if c == nil {
		t.Error("corpus not instantiated")
	}
	c.Start()

	preprocesser := NewDocPreprocessor(c.DocOut)
	preprocesser.wg.Add(2)
	go preprocesser.partitionStreams()
	d := <-preprocesser.TfOut
	testFile1 := []string{"i", "like", "banana", "sandwich", "cats", "are", "fun", "yummy", "yummy", "yummy"}
	for i, token := range d {
		if token != testFile1[i] {
			t.Error("incorrect input: expected %s but got %s", testFile1[i], token)
		}
	}
	b := <-preprocesser.InvIndexOut
	x := <-preprocesser.DocEncryptOut
	d = <-preprocesser.TfOut
	testFile2 := []string{"the", "cat", "in", "the", "hat", "ate", "a", "yummy", "banana", "and", "didnt", "like", "it"}
	for i, token := range d {
		if token != testFile2[i] {
			t.Error("incorrect input: expected " + testFile2[i] + " but got " + token)
		}
	}

	if b != nil && x != nil {
		fmt.Printf("asdgasg\n")
	}

}
