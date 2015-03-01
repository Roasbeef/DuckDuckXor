package main

import (
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
)

// document...
type document struct {
	*os.File
	DocId int32
}

// CorpusReader....
type CorpusReader struct {
	quit      chan struct{}
	rootDir   string
	started   int32
	shutDown  int32
	Wg        sync.WaitGroup
	DocOut    chan *document
	nextDocId int32
	sync.Mutex
}

// NewCorpusReader....
func NewCorpusReader(rootDir string) *CorpusReader {
	q := make(chan struct{})
	d := make(chan *document)
	return &CorpusReader{quit: q, DocOut: d, rootDir: rootDir}
}

// Stop...
func (c *CorpusReader) Stop() error {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return nil
	}
	close(c.quit)
	c.Wg.Wait()
	return nil
}

// Start...
func (c *CorpusReader) Start() error {
	if atomic.AddInt32(&c.started, 1) != 1 {
		return nil
	}
	c.Wg.Add(2)
	a := make(chan string)
	go c.DirectoryWalker(a, c.rootDir)
	// TOD
	go c.DocumentReader(a, c.DocOut)
	return nil
}

// NextDocId....
func (c *CorpusReader) NextDocId() int32 {
	c.Lock()
	defer c.Unlock()

	c.nextDocId += 1
	return c.nextDocId
}

// documentReader....
func (c *CorpusReader) DocumentReader(filePaths <-chan string, docOut chan *document) {
out:
	for {
		select {
		case <-c.quit:
			break out
		case filePath, ok := <-filePaths:
			if !ok {
				break out
			}
			f, err := os.Open(filePath)
			if err != nil {
				c.Stop()
				break out
			}
			d := &document{File: f, DocId: c.NextDocId()}
			docOut <- d
		}
	}

	// TODO(roasbeef): Re-think closing if want multiple reader workers
	close(docOut)
	c.Wg.Done()
}

// directoryWalker....
func (c *CorpusReader) DirectoryWalker(outPaths chan string, rootDir string) error {

	filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			outPaths <- path

		}
		return nil
	})

	close(outPaths)
	c.Wg.Done()
	return nil
}
