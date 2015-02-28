package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

type document struct {
	*os.File
	DocId int32
}

type CorpusReader struct {
	quit     chan struct{}
	rootDir  string
	started  int32
	shutDown int32
	wg       sync.WaitGroup
	DocOut   chan *document

	nextDocId int32

	sync.Mutex
}

func NewCorpusReader(rootDir string) *CorpusReader {
	q := make(chan struct{})
	d := make(chan *document)
	return &CorpusReader{quit: q, DocOut: d, rootDir: rootDir}
}

func (c *CorpusReader) Stop() error {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return nil
	}
	close(c.quit)
	c.wg.Wait()
	return nil
}

func (c *CorpusReader) Start() error {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return nil
	}
	c.wg.Add(2)
	a := make(chan string)
	go c.directoryWalker(a)
	// TODO(daniel): Re-visit if want to scale out num workers
	go c.documentReader(a)
	return nil
}

func (c *CorpusReader) NextDocId() int32 {
	c.Lock()
	defer c.Unlock()

	c.nextDocId += 1
	return c.nextDocId
}

//documentReader ...
func (c *CorpusReader) documentReader(filePaths <-chan string) {
out:
	for filePath := range filePaths {
		select {
		case <-c.quit:
			break out
		default:
		}

		f, err := os.Open(filePath)
		if err != nil {
			c.Stop()
			break out
		}
		d := &document{File: f, DocId: c.NextDocId()}
		c.DocOut <- d
	}
	// TODO(roasbeef): Re-think closing if want multiple reader workers
	close(c.DocOut)
	c.wg.Done()
}

func (c *CorpusReader) directoryWalker(outPaths chan string) {
	filepath.Walk(c.rootDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			outPaths <- path

		}
		return nil
	})

	close(outPaths)
	c.wg.Done()
}

type TermFrequencyCalculator struct {
}

func (t *TermFrequencyCalculator) createFrequencyMap() {

}
