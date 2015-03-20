package main

import (
	"bytes"
	"crypto/elliptic"
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"math/big"
	"sync"
	"sync/atomic"

	"github.com/jacobsa/crypto/cmac"
	"github.com/roasbeef/DuckDuckXor/crypto"
	pb "github.com/roasbeef/DuckDuckXor/protos"
	"golang.org/x/net/context"
)

// wordIndexCounter is a simple wrapper around a map to create a thread safe
// multi-counter.
type wordIndexCounter struct {
	wordCounter map[string]*indexDocPair
	sync.Mutex
}

type indexDocPair struct {
	count  uint32
	lastId uint32
}

// newWordIndexCounter creates and returns a new instance of the counter.
func newWordIndexCounter() *wordIndexCounter {
	return &wordIndexCounter{wordCounter: make(map[string]*indexDocPair)}
}

// readThenIncrement reads the current stored counter value for the given word,
// then incrementing the counter before returning.
// NOTE: the blinding counter is 1 behind the word-level doc index counter
func (w *wordIndexCounter) readThenIncrement(term string, docId uint32) uint32 {
	w.Lock()
	defer w.Unlock()

	pair, ok := w.wordCounter[term]
	if !ok {
		pair = &indexDocPair{count: 0, lastId: docId}
		w.wordCounter[term] = pair
	}

	c := pair.count

	pair.count++
	pair.lastId = docId

	return c
}

// xTag represents an xTag for a particular (word, docId) combination.
// i.e: g^(xind * wtag). This value is passed around as a serialized ECC point.
type xTag []byte

// EncryptedIndexGenerator is responsible for generating and storing the
// client side encrypted index. This entails generating and sending off t-set
// fragments to the search server, and computing xtags for conjunctive queries.
type EncryptedIndexGenerator struct {
	quit       chan struct{}
	started    int32
	shutdown   int32
	numWorkers int
	wg         sync.WaitGroup

	finishedXtags     chan []xTag
	incomingDocuments chan *InvIndexDocument

	closeOnce     sync.Once
	closeXtagChan func()
	mainWg        *sync.WaitGroup
	janitorWG     sync.WaitGroup

	keyMap map[KeyType][keySize]byte

	counter *wordIndexCounter
	bloom   *bloomMaster
	client  pb.EncryptedSearchClient
	curve   elliptic.Curve
}

// NewEncryptedIndexGenerator creates and returns a new instance of the
// EncryptedIndexGenerator.
func NewEncryptedIndexGenerator(invertedIndexes chan *InvIndexDocument, numWorkers int, keyMap map[KeyType][keySize]byte, bloom *bloomMaster, client pb.EncryptedSearchClient, mainWg *sync.WaitGroup) *EncryptedIndexGenerator {
	e := &EncryptedIndexGenerator{
		quit:    make(chan struct{}),
		counter: newWordIndexCounter(),
		// TODO(roasbeef): buffer?
		finishedXtags:     make(chan []xTag),
		incomingDocuments: invertedIndexes,
		keyMap:            keyMap,
		curve:             elliptic.P224(),
		numWorkers:        numWorkers,
		bloom:             bloom,
		mainWg:            mainWg,
		client:            client,
	}
	var once sync.Once
	e.closeOnce = once
	e.closeXtagChan = func() {
		close(e.finishedXtags)
	}

	return e
}

// Start kicks off the generator, spawning helper goroutines before returning.
func (e *EncryptedIndexGenerator) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return nil
	}
	// Set up chan splitter
	AddToWg(&e.wg, e.mainWg, 1)
	c1, c2 := e.chanSplitter()

	for i := 0; i < e.numWorkers/2; i++ {
		AddToWg(&e.wg, e.mainWg, 1)
		go e.xSetWorker(c1)
	}

	for i := 0; i < e.numWorkers/2; i++ {
		AddToWg(&e.wg, e.mainWg, 1)
		AddToWg(&e.janitorWG, e.mainWg, 1)
		go e.tSetWorker(c2)
	}

	AddToWg(&e.wg, e.mainWg, 1)
	go e.tSetJanitor()

	return nil
}

// Stop gracefully stops the index generator and all related helper goroutines.
func (e *EncryptedIndexGenerator) Stop() error {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		return nil
	}
	close(e.quit)
	e.wg.Wait()
	return nil
}

// chanSplitter is a helper goroutines that copies incoming documents into
// channels to both the tSet and xSet workers.
func (e *EncryptedIndexGenerator) chanSplitter() (chan *InvIndexDocument, chan *InvIndexDocument) {
	xSetChan := make(chan *InvIndexDocument)
	tSetChan := make(chan *InvIndexDocument)
	go func() {
	out:
		for {
			select {
			case docIndex, more := <-e.incomingDocuments:
				if !more {
					break out
				}
				xSetChan <- docIndex
				tSetChan <- docIndex
			case <-e.quit:
				break out
			}
		}
		close(xSetChan)
		close(tSetChan)
		WgDone(&e.wg, e.mainWg)
	}()

	return xSetChan, tSetChan
}

// xSetWorker is responsible for generating the resulting xTags for each
// unique word in incoming document. These xTags are then so they can be sent
// off downstream to be added to the final xSet bloom filter.
func (e *EncryptedIndexGenerator) xSetWorker(workChan chan *InvIndexDocument) {
	xIndKey := e.keyMap[XIndKey]
	xIndPRF, _ := cmac.New(xIndKey[:])

	// TODO(roasbeef): re-name everywhere, not xtag itself but half of it (wtag?)
	xTagKey := e.keyMap[XTagKey]
	xTagPRF, _ := cmac.New(xTagKey[:])

	indBytes := make([]byte, 4)
	indBuf := bytes.NewBuffer(indBytes)
out:
	for {
		select {
		case index, more := <-workChan:
			if !more {
				break out
			}
			// TODO(roasbeef): re-use buffer?
			xTags := make([]xTag, 0, len(index.Words))
			for word, _ := range index.Words {
				// xind = F_p(K_i, ind)
				binary.Write(indBuf, binary.BigEndian, index.DocId)
				_, err := io.Copy(xIndPRF, indBuf)
				if err != nil {
					// TODO(roasbeef): hook up errs
					fmt.Println("ERROR: %v", err)
				}
				xind := xIndPRF.Sum(nil)

				// wtag = F_p(K_x, w)
				xTagPRF.Write([]byte(word))
				wtag := xTagPRF.Sum(nil)

				// xtag = g^(xind * wtag)
				// xtag = (g^xind)^wtag
				x, y := e.curve.ScalarBaseMult(xind)
				xTagX, xTagY := e.curve.ScalarMult(x, y, wtag)
				serialziedPoint := elliptic.Marshal(e.curve, xTagX, xTagY)

				xTags = append(xTags, serialziedPoint)
			}

			go func() {
				e.finishedXtags <- xTags
			}()

			indBuf.Reset()
			xTagPRF.Reset()
			xIndPRF.Reset()
		case <-e.quit:
			break out
		}
	}

	// Signal the streamer that there aren't any more xTags, but do this
	// AT MOST once.
	e.closeOnce.Do(e.closeXtagChan)
	WgDone(&e.wg, e.mainWg)
}

// bloomStreamer is responsible for sending computed xTags off to the
// bloomMaster so they can be added to the xSet bloom filter and finally be set
// to the search server.
func (e *EncryptedIndexGenerator) bloomStreamer() {
	// Block until the X-Set bloom filter has been created.
	e.bloom.WaitForXSetInit()
out:
	for {
		select {
		case xtags, more := <-e.finishedXtags:
			if !more {
				break out
			}
			e.bloom.QueueXSetAdd(xtags)
		case <-e.quit:
			break out
		}
	}
	WgDone(&e.wg, e.mainWg)
}

// tSetWorker is responsible computing and sending off t-set tuples for each
// unique word per document recieved. This entails computing the proper t-set // bucket and label for a tuple, it's blinding value for conjunctive queries,
// and permuting the document ID, unique for each word.
func (e *EncryptedIndexGenerator) tSetWorker(workChan chan *InvIndexDocument) {
	tSetStream, err := e.client.UploadTSet(context.Background())
	if err != nil {
		// TODO(roasbeef): Handle err
		fmt.Println("TSET ERROR: %v", err)
	}

	// TODO(roasbeef): Cache these values amongst workers?
	xIndKey := e.keyMap[XIndKey]
	xIndPRF, _ := cmac.New(xIndKey[:])

	blindKey := e.keyMap[DHBlindKey]
	blindPRF, _ := cmac.New(blindKey[:])

	wTrapKey := e.keyMap[WTrapKey]
	wTrapPRF, _ := cmac.New(wTrapKey[:])

	sTagKey := e.keyMap[STagKey]
	sTagPRF, _ := cmac.New(sTagKey[:])
out:
	for {
		select {
		case index, more := <-workChan:
			if !more {
				tSetStream.CloseAndRecv()
				break out
			}

			for word, _ := range index.Words {
				blindCounter := e.counter.readThenIncrement(word, index.DocId)
				docCounter := blindCounter + 1
				// TODO(roasbeef): Buffer re-use??

				// z = F_p(K_z, w || c)
				z := computeBlindingValue(blindPRF, word, blindCounter)

				// xind = F_p(K_i, ind)
				xind := computeXind(xIndPRF, index.DocId)

				// y = xind * z^-1
				y := computeBlindedXind(z, xind, e.curve.Params().P)

				// Permute the document ID using a format
				// preserving encryption scheme.
				e := crypto.PermuteDocId(word, wTrapPRF, index.DocId)

				// Our tuple inverted index tuple element is
				// then (e, y)
				tSetShard := calcTsetTuple(sTagPRF, word, docCounter, e, y, false)

				if err := tSetStream.Send(tSetShard); err != nil {
					// log.Fatalf("Failed to send a doc: %v", err)
					// TODO(roasbeef): handle errs
					fmt.Println("TSET ERROR: %v", err)
				}
			}

			wTrapPRF.Reset()
			sTagPRF.Reset()
			blindPRF.Reset()
			xIndPRF.Reset()
		case <-e.quit:
			break out
		}
	}
	WgDone(&e.janitorWG, e.mainWg)
	WgDone(&e.wg, e.mainWg)
}

// tSetJanitor...
func (e *EncryptedIndexGenerator) tSetJanitor() {
	e.janitorWG.Wait()
	tSetStream, err := e.client.UploadTSet(context.Background())
	if err != nil {
		// TODO(roasbeef): Handle err
	}

	xIndKey := e.keyMap[XIndKey]
	xIndPRF, _ := cmac.New(xIndKey[:])

	blindKey := e.keyMap[DHBlindKey]
	blindPRF, _ := cmac.New(blindKey[:])

	wTrapKey := e.keyMap[WTrapKey]
	wTrapPRF, _ := cmac.New(wTrapKey[:])

	sTagKey := e.keyMap[STagKey]
	sTagPRF, _ := cmac.New(sTagKey[:])

	// TODO(roasbeef): WAYY to redundant need to clean up.
	for word, pair := range e.counter.wordCounter {
		blindCounter := pair.count
		docIndex := pair.count + 1
		docId := pair.lastId

		// z = F_p(K_z, w || c)
		z := computeBlindingValue(blindPRF, word, blindCounter)

		// xind = F_p(K_i, ind)
		xind := computeXind(xIndPRF, docId)

		// y = xind * z^-1
		y := computeBlindedXind(z, xind, e.curve.Params().P)

		// Permute the document ID using a format
		// preserving encryption scheme.
		e := crypto.PermuteDocId(word, wTrapPRF, docId)

		// Our tuple inverted index tuple element is
		// then (e, y)
		tSetShard := calcTsetTuple(sTagPRF, word, docIndex, e, y, true)

		if err := tSetStream.Send(tSetShard); err != nil {
			// log.Fatalf("Failed to send a doc: %v", err)
			// TODO(roasbeef): handle errs
		}
		wTrapPRF.Reset()
		sTagPRF.Reset()
		blindPRF.Reset()
		xIndPRF.Reset()
	}

	tSetStream.CloseAndRecv()
	WgDone(&e.wg, e.mainWg)
}

// computeBlindingValue computes the blinding value used for blinded
// Diffie-Helman exponentation for use in the oblivious computatino protocol for
// conjunctive searches.
func computeBlindingValue(prf hash.Hash, word string, blindCounter uint32) []byte {
	// z = F_p(K_z, w || c)
	var zBuffer bytes.Buffer
	zBuffer.Write([]byte(word))
	binary.Write(&zBuffer, binary.BigEndian, blindCounter)
	prf.Write(zBuffer.Bytes())
	return prf.Sum(nil)
}

// computeXind computes the xind, which is the result of running the actual
// document ID through a psuedo-random function.
func computeXind(prf hash.Hash, ind uint32) []byte {
	// xind = F_p(K_i, ind)
	var indBuf bytes.Buffer
	binary.Write(&indBuf, binary.BigEndian, ind)
	prf.Write(indBuf.Bytes())
	return prf.Sum(nil)
}

// computeBlindedXind pre-computes xind*z^-1 for storage server-side for the
// OXT protocol. z^-1 indicates the multiplicative inverse of z mod the order
// of our chosen curve.
func computeBlindedXind(z, xind []byte, groupOrder *big.Int) []byte {
	// Turn into big ints for mod inverse calculation and mult.
	zBig := new(big.Int).SetBytes(z)
	xindBig := new(big.Int).SetBytes(xind)

	// y = xind * z^-1
	zInverse := new(big.Int).ModInverse(zBig, groupOrder)
	y := new(big.Int).Mod(
		new(big.Int).Mul(xindBig, zInverse),
		groupOrder,
	)
	return y.Bytes()
}

// calcTsetTuple calculates indexing information for the a given word and tuple
// data pair.
func calcTsetTuple(sprf hash.Hash, word string, tupleIndex uint32, permutedId, blindedXind []byte, isLast bool) *pb.TSetFragment {
	// stag = F(K_t, w)
	sprf.Write([]byte(word))
	sTag := sprf.Sum(nil)

	// TODO(roasbeef): the free set stuff?, cache also?
	// b, L, K = H(F(stag, index))
	bucket, label, otp := crypto.CalcTsetVals(sTag, tupleIndex)

	// We actually write zero byte instead of bit here.
	var tsetTuple bytes.Buffer
	var tElement bytes.Buffer
	if isLast {
		tElement.Write([]byte{0x01})
	} else {
		tElement.Write([]byte{0x00})
	}
	tElement.Write(permutedId)
	tElement.Write(blindedXind) // TODO(roasbeef): pad out?
	tsetTuple.Write(crypto.XorBytes(tElement.Bytes(), otp[:]))

	return &pb.TSetFragment{
		Bucket: bucket[:],
		Label:  label[:],
		Data:   tsetTuple.Bytes(),
	}
}
