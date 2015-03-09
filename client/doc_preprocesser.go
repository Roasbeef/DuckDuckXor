package main

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

type InvIndexDocument struct {
	Words map[string]struct{}
	DocId int32
}

type DocPreprocessor struct {
	quit          chan struct{}
	started       int32
	shutdown      int32
	TfOut         chan []string
	InvIndexOut   chan *InvIndexDocument
	DocEncryptOut chan *document
	XsetGenOut    chan *InvIndexDocument
	input         chan *document
	wg            sync.WaitGroup
}

func NewDocPreprocessor(inp chan *document) *DocPreprocessor {
	q := make(chan struct{})
	//channels have buffer of size one in case of errors
	d := make(chan *document, 1)
	i := make(chan *InvIndexDocument, 1)
	x := make(chan *InvIndexDocument, 1)
	t := make(chan []string, 1)
	return &DocPreprocessor{quit: q, TfOut: t, InvIndexOut: i, DocEncryptOut: d, XsetGenOut: x, input: inp}
}

func (d *DocPreprocessor) Start() error {

	if atomic.AddInt32(&d.started, 1) != 1 {
		return nil
	}
	d.wg.Add(1)
	go d.partitionStreams()
	return nil

}

func (d *DocPreprocessor) Stop() error {
	if atomic.AddInt32(&d.shutdown, 1) != 1 {
		return nil
	}
	close(d.quit)
	d.wg.Wait()
	return nil
}

func (d *DocPreprocessor) partitionStreams() {
	var bit []byte
	b := bytes.NewBuffer(bit)
out:
	for {
		select {
		case <-d.quit:
			break out
		case doc, ok := <-d.input:
			if !ok {
				break out
			}
			invIndexMap := make(map[string]struct{})
			_, err := io.Copy(b, doc)
			if err != nil {
				//TODO error handling
			}
			parsedTFWords := ParseTokens(string(b.Bytes()))
			for _, token := range parsedTFWords {
				invIndexMap[token] = struct{}{}
			}
			InvIndDoc := &InvIndexDocument{invIndexMap, doc.DocId}
			d.TfOut <- parsedTFWords
			d.InvIndexOut <- InvIndDoc
			d.DocEncryptOut <- doc
			d.XsetGenOut <- InvIndDoc
			b.Reset()
		}
	}
	close(d.TfOut)
	close(d.InvIndexOut)
	close(d.DocEncryptOut)
	close(d.XsetGenOut)
	d.wg.Done()

}

func ParseTokens(s string) []string {
	a := strings.ToLower(s)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	return strings.FieldsFunc(a, f)
}
