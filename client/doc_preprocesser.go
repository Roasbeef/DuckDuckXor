package main

import (
	"bytes"
	"io"
	"strings"
	"sync"
	"unicode"
)

type DocPreprocessor struct {
	quit          chan struct{}
	TfOut         chan []string
	InvIndexOut   chan map[string]struct{}
	DocEncryptOut chan *document
	input         chan *document
	wg            sync.WaitGroup
}

func NewDocPreprocessor(inp chan *document) *DocPreprocessor {
	q := make(chan struct{})
	d := make(chan *document)
	i := make(chan map[string]struct{})
	t := make(chan []string)
	return &DocPreprocessor{quit: q, TfOut: t, InvIndexOut: i, DocEncryptOut: d, input: inp}
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
			d.TfOut <- parsedTFWords
			d.InvIndexOut <- invIndexMap
			d.DocEncryptOut <- doc
		}
	}
	d.wg.Done()
}

func ParseTokens(s string) []string {
	a := strings.ToLower(s)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	return strings.FieldsFunc(a, f)
}
