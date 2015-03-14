package main

import (
	"bytes"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"hash"
	"math/big"
	"sync"
	"sync/atomic"

	pb "github.com/roasbeef/DuckDuckXor/protos"
	"github.com/roasbeef/perm-crypt"
	"golang.org/x/net/context"
)

// wordIndexCounter...
type wordIndexCounter struct {
	wordCounter map[string]uint32
	sync.RWMutex
}

// TODO(roasbeef): Move to diff file?
func NewWordIndexCounter() *wordIndexCounter {
	return &wordIndexCounter{wordCounter: make(map[string]uint32)}
}

// readThenIncrement....
// blinding counter is 1 behind the index counter
func (w *wordIndexCounter) readThenIncrement(term string) uint32 {
	w.Lock()
	defer w.Unlock()
	c, ok := w.wordCounter[term]
	if !ok {
		w.wordCounter[term] = 0
		c = 0
	}
	w.wordCounter[term]++
	return c
}

// TODO(roasbeef): Adding functions to this could make downstrema
// changes easier more testable. Move for xset.go?
type xTag []byte

// EncryptedIndexGenerator....
type EncryptedIndexGenerator struct {
	quit       chan struct{}
	started    int32
	shutdown   int32
	numWorkers int
	wg         sync.WaitGroup

	finishedXtags     chan []xTag
	incomingDocuments chan *InvIndexDocument

	keyMap map[KeyType]*[keySize]byte

	counter *wordIndexCounter
	bloom   *bloomMaster
	client  pb.EncryptedSearchClient
	curve   elliptic.Curve
}

// NewEncryptedIndexGenerator....
// TODO(roasbeef): Collapse inv index here?
// only actually needed it for the B bit.
func NewEncryptedIndexGenerator(invertedIndexes chan *InvIndexDocument, numWorkers int, keyMap map[KeyType]*[keySize]byte) (*EncryptedIndexGenerator, error) {
	return &EncryptedIndexGenerator{
		quit:    make(chan struct{}),
		counter: NewWordIndexCounter(),
		// TODO(roasbeef): buffer?
		finishedXtags:     make(chan []xTag),
		incomingDocuments: invertedIndexes,
		keyMap:            keyMap,
		curve:             elliptic.P224(),
		numWorkers:        numWorkers,
	}, nil
}

// Start...
func (e *EncryptedIndexGenerator) Start() error {
	if atomic.AddInt32(&e.started, 1) != 1 {
		return nil
	}
	// Set up chan splitter
	e.wg.Add(1)
	c1, c2 := e.chanSplitter(e.numWorkers * 2)

	for i := 0; i < e.numWorkers/2; i++ {
		e.wg.Add(1)
		go e.xSetWorker(c1)
	}

	for i := 0; i < e.numWorkers/2; i++ {
		e.wg.Add(1)
		go e.tSetWorker(c2)
	}
	return nil
}

// Stop....
func (e *EncryptedIndexGenerator) Stop() error {
	if atomic.AddInt32(&e.shutdown, 1) != 1 {
		return nil
	}
	close(e.quit)
	e.wg.Wait()
	return nil
}

// chanSplitter....
func (e *EncryptedIndexGenerator) chanSplitter(bufferSize int) (chan *InvIndexDocument, chan *InvIndexDocument) {
	xSetChan := make(chan *InvIndexDocument, bufferSize)
	tSetChan := make(chan *InvIndexDocument, bufferSize)
	go func() {
	out:
		for {
			select {
			case docIndex := <-e.incomingDocuments:
				xSetChan <- docIndex
				tSetChan <- docIndex
			case <-e.quit:
				break out
			}
		}
		e.wg.Done()
	}()

	return xSetChan, tSetChan
}

// xSetWorker...
func (e *EncryptedIndexGenerator) xSetWorker(workChan chan *InvIndexDocument) {
	//xIndKey := e.keyMap[XIndKey]
	//xIndPRF := hmac.New(sha256.New, (*xIndKey)[:])
	xIndKey := e.keyMap[XIndKey]
	xIndPRF := hmac.New(sha1.New, (*xIndKey)[:16]) // TODO(roasbeef): extract slice

	// TODO(roasbeef): re-name everywhere, not xtag itself but half of it.
	xTagKey := e.keyMap[XTagKey]
	xTagPRF := hmac.New(sha1.New, (*xTagKey)[:16])

	indBuf := new(bytes.Buffer)
out:
	for {
		select {
		case index := <-workChan:
			// TODO(roasbeef): re-use buffer?
			xTags := make([]xTag, len(index.Words))
			for word, _ := range index.Words {
				// xind = F_p(K_i, ind)
				binary.Write(indBuf, binary.BigEndian, index.DocId)
				xIndPRF.Write(indBuf.Bytes())
				xind := xIndPRF.Sum(nil)

				// xtrap = F_p(K_x, w)
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

			xTagPRF.Reset()
			xIndPRF.Reset()
		case <-e.quit:
			break out
		}
	}
	e.wg.Done()
}

// bloomStreamer....
func (e *EncryptedIndexGenerator) bloomStreamer() {
	// Block until the X-Set bloom filter has been created.
	e.bloom.WaitForXSetInit()
out:
	for {
		select {
		case xtags := <-e.finishedXtags:
			e.bloom.QueueXSetAdd(xtags)
		case <-e.quit:
			break out
		}
	}
	e.wg.Done()
}

// tSetWorker....
// TODO(roasbeef): Can multiple workers send on a stream????
func (e *EncryptedIndexGenerator) tSetWorker(workChan chan *InvIndexDocument) {
	tSetStream, err := e.client.UploadTSet(context.Background())
	if err != nil {
		// TODO(roasbeef): Handle err
	}

	// TODO(roasbeef): Cache these values amongst workers?
	xIndKey := e.keyMap[XIndKey]
	xIndPRF := hmac.New(sha1.New, (*xIndKey)[:16])

	blindKey := e.keyMap[DHBlindKey]
	blindPRF := hmac.New(sha1.New, (*blindKey)[:16])

	wTrapKey := e.keyMap[WTrapKey]
	wTrapPRF := hmac.New(sha1.New, (*wTrapKey)[:16])

	sTagKey := e.keyMap[STagKey]
	sTagPRF := hmac.New(sha1.New, (*sTagKey)[:16])

	tweakVal := e.keyMap[PermuteTweak]
out:
	for {
		select {
		case index := <-workChan:
			for word, _ := range index.Words {
				blindCounter := e.counter.readThenIncrement(word)
				docCounter := blindCounter + 1

				// TODO(roasbeef): Buffer re-use??

				// z = F_p(K_z, w || c)
				z := computeBlindingValue(blindPRF, word, blindCounter)

				// xind = F_p(K_i, ind)
				xind := computeXind(xIndPRF, index.DocId)

				// y = xind * z^-1
				y := computeBlindedXind(z, xind, e.curve.Params().P)

				// Our tuple inverted index tuple element is
				// then (e, y)
				e := permuteDocId(word, wTrapPRF, tweakVal[:4], index.DocId)

				tSetShard := calcTsetTuple(sTagPRF, word, docCounter, e, y)
				if err := tSetStream.Send(tSetShard); err != nil {
					// log.Fatalf("Failed to send a doc: %v", err)
					// TODO(roasbeef): handle errs
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
	e.wg.Done()
}

// calcTsetVals...
func calcTsetVals(stag []byte, index uint32) ([1]byte, [16]byte, [37]byte) {
	prf := hmac.New(sha512.New, stag)
	binary.Write(prf, binary.BigEndian, index)
	m := prf.Sum(nil)

	// b, L, K = H(F(stag, index))
	sha := sha256.New()
	sha.Write(m)
	tSetSum := sha.Sum(nil)

	// 1 byte for bucket.
	var bucket [1]byte
	copy(bucket[:], tSetSum[0:1])

	// 16 bytes for the label.
	var label [16]byte
	copy(label[:], tSetSum[1:17])

	// end byte + 4 byte doc ID + 32 byte blind value
	// 37 bytes for the one-time pad.
	var otp [37]byte
	copy(otp[:], tSetSum[17:54])

	return bucket, label, otp
}

// xorBytes...
func xorBytes(x, y []byte) []byte {
	out := make([]byte, len(x))
	for i := 0; i < len(x); i++ {
		out[i] = x[i] ^ y[i]
	}
	return out
}

// computeBlindingValue....
func computeBlindingValue(prf hash.Hash, word string, blindCounter uint32) []byte {
	// z = F_p(K_z, w || c)
	var zBuffer bytes.Buffer
	zBuffer.Write([]byte(word))
	binary.Write(&zBuffer, binary.BigEndian, blindCounter)
	prf.Write(zBuffer.Bytes())
	return prf.Sum(nil)
}

// computeXind....
func computeXind(prf hash.Hash, ind uint32) []byte {
	// xind = F_p(K_i, ind)
	var indBuf bytes.Buffer
	binary.Write(&indBuf, binary.BigEndian, ind)
	prf.Write(indBuf.Bytes())
	return prf.Sum(nil)
}

// computeBlindedXind...
func computeBlindedXind(z, xind []byte, groupOrder *big.Int) []byte {
	// Turn into big ints for mod inverse calculation and mult.
	zBig := new(big.Int).SetBytes(z)
	xindBig := new(big.Int).SetBytes(xind)

	// y = xind * z^-1
	zInverse := new(big.Int).ModInverse(zBig, groupOrder)
	y := new(big.Int).Mul(xindBig, zInverse)
	return y.Bytes()
}

// permuteDocId...
func permuteDocId(word string, wPrf hash.Hash, tweak []byte, docId uint32) []byte {
	// K_e = F(k_s, w)
	wPrf.Write([]byte(word))
	wordPermKey := wPrf.Sum(nil)

	// e = Enc(Ke, ind)
	wordPerm, err := aesffx.NewCipher(16, wordPermKey, tweak)
	if err != nil {
		// TODO(roasbeef): handle err
	}
	var xindBuf bytes.Buffer
	binary.Write(&xindBuf, binary.BigEndian, docId)
	xindString := hex.EncodeToString(xindBuf.Bytes())
	permutedId, err := wordPerm.Encrypt(xindString)
	if err != nil {
		// TODO(roasbeef): handle err
	}

	permutedBytes, err := hex.DecodeString(permutedId)
	if err != nil {
		// TODO(roasbeef): handle err
	}
	return permutedBytes
}

// calcTsetTuple....
func calcTsetTuple(sprf hash.Hash, word string, tupleIndex uint32, permutedId, blindedXind []byte) *pb.TSetFragment {
	// stag = F(K_t, w)
	sprf.Write([]byte(word))
	sTag := sprf.Sum(nil)

	// TODO(roasbeef): the free set stuff?, cache also?
	// b, L, K = H(F(stag, index))
	bucket, label, otp := calcTsetVals(sTag, tupleIndex)

	// TODO(roasbeef): Scheme to get exact bit.
	// We actually write zero byte instead of bit here.
	var tsetTuple bytes.Buffer
	var tElement bytes.Buffer
	tElement.Write([]byte{0})
	tElement.Write(permutedId)
	tElement.Write(blindedXind) // TODO(roasbeef): pad out?
	tsetTuple.Write(xorBytes(tElement.Bytes(), otp[:]))

	return &pb.TSetFragment{
		Bucket: bucket[:],
		Label:  label[:],
		Data:   tsetTuple.Bytes(),
	}
}
