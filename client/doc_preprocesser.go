package main

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"unicode"
)

type InvIndexDocument struct {
	Words map[string]struct{}
	DocId uint32
}

type Doc struct {
	Name  string
	DocId uint32
}

type DocPreprocessor struct {
	quit             chan struct{}
	started          int32
	shutdown         int32
	abort            func(chan struct{}, error)
	docNames         chan Doc
	TfOut            chan []string
	InvIndexOut      chan *InvIndexDocument
	KeyWordSummerOut chan *InvIndexDocument
	DocEncryptOut    chan *document
	input            chan *document
	wg               sync.WaitGroup
}

func NewDocPreprocessor(inp chan *document, e func(chan struct{}, error)) *DocPreprocessor {
	q := make(chan struct{})
	//channels have buffer of size one in case of errors
	d := make(chan *document, 1)
	i := make(chan *InvIndexDocument, 1)
	k := make(chan *InvIndexDocument, 1)
	t := make(chan []string, 1)
	return &DocPreprocessor{
		quit:             q,
		TfOut:            t,
		InvIndexOut:      i,
		DocEncryptOut:    d,
		KeyWordSummerOut: k,
		docNames:         make(chan Doc),
		input:            inp,
		abort:            e,
	}
}

func (d *DocPreprocessor) Start() error {
	fmt.Printf("PreProcessing Documents\n")
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
			fmt.Println("PRE: got doc", doc)
			if !ok {
				break out
			}
			//d.docNames <- Doc{doc.Name(), doc.DocId}
			invIndexMap := make(map[string]struct{})
			_, err := io.Copy(b, doc)
			if err != nil {
				d.abort(d.quit, err)
			}
			parsedTFWords := ParseTokens(string(b.Bytes()))
			for _, token := range parsedTFWords {
				invIndexMap[token] = struct{}{}
			}

			InvIndDoc := &InvIndexDocument{invIndexMap, doc.DocId}
			d.TfOut <- parsedTFWords
			fmt.Println("sent to term freq")
			d.InvIndexOut <- InvIndDoc
			d.KeyWordSummerOut <- InvIndDoc
			fmt.Println("sent to inv index")
			d.DocEncryptOut <- doc
			fmt.Println("sent to doc enc")
			b.Reset()
		}
	}
	close(d.TfOut)
	close(d.InvIndexOut)
	close(d.DocEncryptOut)
	close(d.KeyWordSummerOut)
	d.wg.Done()

}

func ParseTokens(s string) []string {
	a := strings.ToLower(s)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	return strings.FieldsFunc(a, f)
}
