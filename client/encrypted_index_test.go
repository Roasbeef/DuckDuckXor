package main

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"os"
	"sync"
	"testing"

	pb "github.com/roasbeef/DuckDuckXor/protos"

	"github.com/willf/bloom"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func TestWordIndexCounter(t *testing.T) {
	counter := newWordIndexCounter()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for i := uint32(0); i < 5; i++ {
			counter.readThenIncrement("test", i)
		}
		wg.Done()
	}()
	wg.Add(1)
	go func() {
		for i := uint32(5); i < 10; i++ {
			counter.readThenIncrement("test", i)
		}
		wg.Done()
	}()

	wg.Wait()

	if counter.wordCounter["test"].count != 10 {
		t.Fatalf("Wrong counter value, should be %v, got %v",
			10, counter.wordCounter["test"].count)
	}

}

func TestChannelSplitter(t *testing.T) {
	indexChan := make(chan *InvIndexDocument)
	e := NewEncryptedIndexGenerator(indexChan, 5, nil, nil, nil)
	e.wg.Add(1)
	c1, c2 := e.chanSplitter()

	indexChan <- &InvIndexDocument{Words: make(map[string]struct{}), DocId: 10}

	// Verify two goroutines can pull from the channel and interact with
	// vals.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		for doc := range c1 {
			fmt.Println(doc.DocId)
		}
		wg.Done()
	}()

	wg.Add(1)
	go func() {
		for doc := range c2 {
			fmt.Println(doc.DocId)
		}
		wg.Done()
	}()

	// Workers should get nil read at this point.
	e.Stop()
	wg.Wait()
}

func TestXsetWorker(t *testing.T) {
	keyMap := make(map[KeyType][keySize]byte)
	for i := 0; i < numKeys; i++ {
		var b [32]byte
		_, err := rand.Read(b[:])
		if err != nil {
			t.Fatalf("unable to read rand num: %v", err)
		}

		keyMap[KeyType(i)] = b
	}

	indexChan := make(chan *InvIndexDocument)
	e := NewEncryptedIndexGenerator(indexChan, 5, keyMap, nil, nil)
	e.wg.Add(1)
	c1, c2 := e.chanSplitter()

	// Goroutine to eat up tSet chan.
	go func() {
		<-c2
		<-c2
		<-c2
	}()

	// Launch two xSet workers.
	e.wg.Add(2)
	go e.xSetWorker(c1)
	go e.xSetWorker(c1)

	var wg sync.WaitGroup

	// Send some fake docs to the xSetWorker.
	wg.Add(1)
	go func() {
		words := map[string]struct{}{
			"test": struct{}{},
			"ok":   struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 10}
		words = map[string]struct{}{
			"tester": struct{}{},
			"yeh":    struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 44}
		words = map[string]struct{}{
			"testin": struct{}{},
			"cool":   struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 40}
		wg.Done()
	}()

	// Verify we get all 6 tags back.
	tag1 := <-e.finishedXtags
	tag2 := <-e.finishedXtags
	tag3 := <-e.finishedXtags
	if bytes.Equal(tag1[0], tag2[0]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag1[0], tag2[0])
	}
	if bytes.Equal(tag1[1], tag2[1]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag1[1], tag2[1])
	}
	if bytes.Equal(tag2[0], tag3[0]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag2[0], tag3[0])
	}
	if bytes.Equal(tag2[1], tag3[1]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag2[1], tag3[1])
	}
	if bytes.Equal(tag1[0], tag3[0]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag1[0], tag3[0])
	}
	if bytes.Equal(tag1[1], tag3[1]) {
		t.Fatalf("Tags match, they shouldn't %v vs %v", tag1[1], tag3[1])
	}

	wg.Wait()
	e.Stop()
}

func TestBloomStreamer(t *testing.T) {
	db, path := createFakeDB(t)
	defer os.Remove(path)
	defer db.Close()

	// Create a bloom master, and a worker so our calls to add xtags will
	// be processed.
	b, err := newBloomMaster(db, 1)
	// Should only house 4 xtags.
	b.xFinalSize = 1
	b.xSetFilter = bloom.NewWithEstimates(100, 0.00001)
	b.wg.Add(1)
	go b.bloomWorker()

	if err != nil {
		t.Fatalf("Unable to create bloom master: %v", err)
	}

	e := NewEncryptedIndexGenerator(nil, 5, nil, b, nil)
	e.wg.Add(1)
	go e.bloomStreamer()

	// Signal that the x-set has been created.
	close(b.xSetReady)

	// Send over a fake xtag.
	var wg sync.WaitGroup
	go func() {
		wg.Add(1)
		e.finishedXtags <- []xTag{xTag([]byte("ok"))}
		close(e.finishedXtags)
		wg.Done()
	}()

	wg.Wait()
	<-b.xFilterFinished

	// Confirm the xtags have been added.
	if !b.xSetFilter.Test([]byte("ok")) {
		t.Fatalf("ok wasn't added to the bloom filter")
	}

	b.Stop()
	e.Stop()
}

type MockSearchClient struct {
	grpc.ClientStream
}

type MockTSetClient struct {
	grpc.ClientStream
}

func (m MockTSetClient) Send(*pb.TSetFragment) error {
	return nil
}

func (m MockTSetClient) CloseAndRecv() (*pb.TSetAck, error) {
	return nil, nil
}

func (m MockTSetClient) CloseSend() error {
	return nil
}

func (m MockSearchClient) UploadMetaData(ctx context.Context, in *pb.MetaData, opts ...grpc.CallOption) (*pb.MetaDataAck, error) {
	return nil, nil
}
func (m MockSearchClient) UploadTSet(ctx context.Context, opts ...grpc.CallOption) (pb.EncryptedSearch_UploadTSetClient, error) {
	return &MockTSetClient{}, nil
}
func (m MockSearchClient) UploadXSetFilter(ctx context.Context, in *pb.XSetFilter, opts ...grpc.CallOption) (*pb.FilterAck, error) {
	return nil, nil
}
func (m MockSearchClient) UploadCipherDocs(ctx context.Context, opts ...grpc.CallOption) (pb.EncryptedSearch_UploadCipherDocsClient, error) {
	return nil, nil
}
func (m MockSearchClient) KeywordSearch(ctx context.Context, in *pb.KeywordQuery, opts ...grpc.CallOption) (pb.EncryptedSearch_KeywordSearchClient, error) {
	return nil, nil
}
func (m MockSearchClient) FetchDocuments(ctx context.Context, opts ...grpc.CallOption) (pb.EncryptedSearch_FetchDocumentsClient, error) {
	return nil, nil
}
func (m MockSearchClient) ConjunctiveSearchRequest(ctx context.Context, opts ...grpc.CallOption) (pb.EncryptedSearch_ConjunctiveSearchRequestClient, error) {
	return nil, nil
}
func (m MockSearchClient) XTokenExchange(ctx context.Context, opts ...grpc.CallOption) (pb.EncryptedSearch_XTokenExchangeClient, error) {
	return nil, nil
}

func TestTsetWorker(t *testing.T) {
	// send doc down channel.
	// verify tset gets sent
	// send two, make sure PRF reset works?
	// Create some fake keys for testing purposes.
	keyMap := make(map[KeyType][keySize]byte)
	for i := 0; i < numKeys; i++ {
		var b [32]byte
		_, err := rand.Read(b[:])
		if err != nil {
			t.Fatalf("unable to read rand num: %v", err)
		}

		keyMap[KeyType(i)] = b
	}

	indexChan := make(chan *InvIndexDocument)
	e := NewEncryptedIndexGenerator(indexChan, 5, keyMap, nil, MockSearchClient{})
	e.wg.Add(1)
	c1, c2 := e.chanSplitter()

	// Goroutine to eat up xSet chan.
	go func() {
		<-c1
		<-c1
		<-c1
	}()

	// Launch two xSet workers.
	e.wg.Add(2)
	e.janitorWG.Add(2)
	go e.tSetWorker(c2)
	go e.tSetWorker(c2)

	// Start the janitor.
	e.wg.Add(1)
	go e.tSetJanitor()

	var wg sync.WaitGroup

	// Send some fake docs to the tSetWorker.
	wg.Add(1)
	go func() {
		words := map[string]struct{}{
			"test": struct{}{},
			"ok":   struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 10}
		words = map[string]struct{}{
			"test": struct{}{},
			"yeh":  struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 44}
		words = map[string]struct{}{
			"test": struct{}{},
			"cool": struct{}{},
		}
		indexChan <- &InvIndexDocument{Words: words, DocId: 40}
		wg.Done()
	}()

	wg.Wait()

	// Signal end of incoming documents.
	close(indexChan)
	//e.Stop()

	// Verify last docID for test is 40??
}
