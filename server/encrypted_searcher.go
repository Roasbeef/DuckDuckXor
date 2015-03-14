package main

import (
	pb "github.com/roasbeef/DuckDuckXor/protos"
)

type encryptedIndexSearcher struct {
}

func (e *encryptedIndexSearcher) Start() error {
	return nil
}

func (e *encryptedIndexSearcher) Stop() error {
	return nil
}

func (e *encryptedIndexSearcher) LoadTSetMetaData(mData *pb.MetaData) {
}
