package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
)

func TestDirectoryWalker(t *testing.T) {
	c := NewCorpusReader("./testDirectory")
	if c == nil {
		t.Error("class not instantiated")
	}
	a := make(chan string)
	//	c.Start()
	c.Wg.Add(1)
	go c.DirectoryWalker(a, "./testDirectory")
	counter := 0
	for s := range a {
		fmt.Printf("%s\n", s)
		if !strings.Contains(s, "testDirectory") {
			t.Error("test returned incorrect classPath: %s", s)
		}
		counter++
	}
	if counter != 2 {
		t.Error("incorrect number of files returned: %i", counter)
	}
}

func TestDocumentReader(t *testing.T) {
	c := NewCorpusReader("./testDirectory")
	a := make(chan string)
	c.Wg.Add(1)
	go c.DocumentReader(a, c.DocOut)
	b, _ := filepath.Abs("./testDirectory/testfile1.txt")
	a <- b
	d := <-c.DocOut
	b, _ = filepath.Abs("./testDirectory/testfile2.txt")
	a <- b
	fmt.Printf("%s", d.DocId)
	d = <-c.DocOut
	fmt.Printf("%s", d.DocId)
	t.Error("stub")

}
