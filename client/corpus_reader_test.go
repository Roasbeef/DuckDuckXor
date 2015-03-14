package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryWalker(t *testing.T) {
	c := NewCorpusReader("./test_directory", nil)
	if c == nil {
		t.Error("class not instantiated")
	}
	a := make(chan string)
	//	c.Start()
	c.Wg.Add(1)
	go c.DirectoryWalker(a, "./test_directory")
	counter := 0
	for s := range a {
		if !strings.Contains(s, "test_directory") {
			t.Error("test returned incorrect classPath: %s", s)
		}
		counter++
	}
	if counter != 2 {
		t.Error("incorrect number of files returned: %i", counter)
	}
}

func TestDocumentReader(t *testing.T) {
	c := NewCorpusReader("./test_directory", nil)
	a := make(chan string)
	c.Wg.Add(1)
	go c.DocumentReader(a, c.DocOut)
	b, _ := filepath.Abs("./test_directory/testfile1.txt")
	a <- b
	d := <-c.DocOut
	if d.DocId != 1 {
		t.Error("incorrect DocId assignment: %i", d.DocId)
	}
	if !strings.Contains(d.Name(), "testfile1") {
		t.Error("did not return correct file")
	}
	b, _ = filepath.Abs("./test_directory/testfile2.txt")
	a <- b
	d = <-c.DocOut
	if d.DocId != 2 {
		t.Error("incorrect DocId assignment: ", d.DocId)
	}
	if !strings.Contains(d.Name(), "testfile2") {
		t.Error("did not return correct file")
	}

}
