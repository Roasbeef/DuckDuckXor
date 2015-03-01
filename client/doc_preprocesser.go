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

func (d *DocPreprocessor) partitionStreams() error {
	var bit []byte
	b := bytes.NewBuffer(bit)
	m := make(map[string]struct{})
out:
	for {
		select {
		case <-d.quit:
			break out
		case doc, ok := <-d.input:
			if !ok {
				break out
			}
			_, err := io.Copy(b, doc)
			if err != nil {
				return err
			}
			s := string(b.Bytes())
			p := ParseTokens(s)
			d.TfOut <- p
			for _, token := range p {
				m[token] = struct{}{}
			}
			d.InvIndexOut <- m
			d.DocEncryptOut <- doc
			for k := range m {
				delete(m, k)
			}
		}
	}
	d.wg.Done()
	return nil
}

func ParseTokens(s string) []string {
	a := strings.ToLower(s)
	f := func(c rune) bool {
		return !unicode.IsLetter(c) && !unicode.IsNumber(c)
	}

	return strings.FieldsFunc(a, f)
}
