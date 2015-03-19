package main

import "testing"

func TestPartitionStreams(t *testing.T) {
	c := NewCorpusReader("./test_directory", nil)
	if c == nil {
		t.Error("corpus not instantiated")
	}
	c.Start()

	preprocesser := NewDocPreprocessor(c.DocOut, nil)
	preprocesser.Start()
	name := <-preprocesser.docNames
	if name.Name != "test_directory/testFile1.txt" {
		t.Error("gave incorrect doc name. expected test_directory/testfile1 but got: ", name.Name)
	}
	d := <-preprocesser.TfOut
	testFile1 := []string{"i", "like", "banana", "sandwich", "cats", "are", "fun", "yummy", "yummy", "yummy"}
	for i, token := range d {
		if token != testFile1[i] {
			t.Error("incorrect input: expected %s but got %s", testFile1[i], token)
		}
	}
	<-preprocesser.InvIndexOut
	<-preprocesser.DocEncryptOut
	<-preprocesser.XsetGenOut
	<-preprocesser.docNames
	d = <-preprocesser.TfOut
	testFile2 := []string{"the", "cat", "in", "the", "hat", "ate", "a", "yummy", "banana", "and", "didnt", "like", "it"}
	for i, token := range d {
		if token != testFile2[i] {
			t.Error("incorrect input: expected " + testFile2[i] + " but got " + token)
		}
	}

	<-preprocesser.InvIndexOut
	<-preprocesser.DocEncryptOut
	<-preprocesser.XsetGenOut
	preprocesser.Stop()
}
