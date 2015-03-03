package main

import (
	"crypto/sha512"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	"github.com/conformal/btcwallet/snacl"
)

type KeyType int

const (
	WTrapKey   KeyType = iota // K_s
	XTagKey                   // K_x
	XIndKey                   // K_i
	DHBlindKey                // K_z
	STagKey                   // K_t
	DocEncKey                 // Wrapped in snacl.CryptoKey
)
const (
	numKeys       = 6
	keySize       = 32
	keyBucketName = "cryptoKeys"
)

type keyRequestMessage struct {
	whichKey KeyType
	// TODO(roasbeef): BYOB everywhere?
	replyChan chan *[keySize]byte
	errChan   chan error // should be buffered
}

// TODO(roasbeef): Review validity of this struct
type KeyManager struct {
	wg          sync.WaitGroup
	keyMap      map[KeyType][keySize]byte
	keyRequests chan keyRequestMessage

	quit     chan struct{}
	started  int32
	shutdown int32
}

// NewKeyManager.....
func NewKeyManager(db bolt.DB, passphrase []byte) (*KeyManager, error) {
	k := &KeyManager{
		quit:        make(chan struct{}),
		keyMap:      make(map[KeyType][keySize]byte),
		keyRequests: make(chan keyRequestMessage),
	}

	// If our key bucket is present, then we've already done the initial set-up.
	// Otherwise, perform the derivation, and set-up steps for our keys.
	// TODO(roasbeef): Refactor this
	var found bool
	if err := db.View(func(tx *bolt.Tx) error {
		keyBucket := tx.Bucket([]byte(keyBucketName))
		found = !(keyBucket == nil)
		return nil
	}); err != nil {
		return nil, err
	}

	if !found {
		// Derive our master key using scrypt.
		masterKey, err := snacl.NewSecretKey(
			&passphrase, snacl.DefaultN, snacl.DefaultR, snacl.DefaultP,
		)
		if err != nil {
			return nil, err
		}

		// Derive child keys from our master key. These are the keys we
		// end up using for symmetric encryption and our PRFs.
		var mKey [keySize]byte
		copy(mKey[:], masterKey.Key[:])
		childKeys, err := inverseHashTreeKeyDerivation(mKey, numKeys)
		if err != nil {
			return nil, err
		}

		// We never store the master key, only paramters from which we
		// can re-derive it. This master key is used to encrypt the
		// derived child keys when they're stored in the DB.
		encryptedKeys, err := encryptChildKeys(masterKey, childKeys)
		if err != nil {
			return nil, err
		}

		serializedMasterKey := masterKey.Marshal()

		// Commit the encrypted keys to the DB.
		err = db.Update(func(tx *bolt.Tx) error {
			b, err := tx.CreateBucket([]byte(keyBucketName))
			if err != nil {
				return err
			}

			// Store the encrypted master key parameters.
			err = b.Put([]byte("master"), serializedMasterKey)

			// Store each individual key
			// TODO(roasbeef): Clean up with helper func, check each err?
			err = b.Put([]byte("k_s"), encryptedKeys[0])
			err = b.Put([]byte("k_x"), encryptedKeys[1])
			err = b.Put([]byte("k_i"), encryptedKeys[2])
			err = b.Put([]byte("k_z"), encryptedKeys[3])
			err = b.Put([]byte("k_t"), encryptedKeys[4])
			err = b.Put([]byte("doc_key"), encryptedKeys[5])
			if err != nil {
				return err
			}

			return nil
		})
		if err != nil {
			return nil, err
		}

		// zero out the master key, we don't need it anymore.
		masterKey.Zero()
		// Finally create our map, then we're all ready to go
		k.updateKeyMap(childKeys)
	} else {
		// else open bucket, decrypt, all, load into memory
		err := db.View(func(tx *bolt.Tx) error {
			b := tx.Bucket([]byte(keyBucketName))

			// Grab the marshalled master key paramaters.
			masterKeyParams := b.Get([]byte("master"))

			// Attempt to unmarshall bailing on failure.
			var masterKey snacl.SecretKey
			if err := masterKey.Unmarshal(masterKeyParams); err != nil {
				return err
			}

			// Try to re-derive the master key from the passed passphrase.
			// If the passphrase is incorrect, this should fail.
			err := masterKey.DeriveKey(&passphrase)
			if err != nil {
				return err
			}

			// TODO(roasbeef): Clean up
			// Decrypt the keys and load into memory.
			err = k.loadDBKey(b, WTrapKey, []byte("k_s"), &masterKey)
			err = k.loadDBKey(b, XTagKey, []byte("k_x"), &masterKey)
			err = k.loadDBKey(b, XIndKey, []byte("k_i"), &masterKey)
			err = k.loadDBKey(b, DHBlindKey, []byte("k_z"), &masterKey)
			err = k.loadDBKey(b, STagKey, []byte("k_t"), &masterKey)
			err = k.loadDBKey(b, DocEncKey, []byte("doc_key"), &masterKey)
			if err != nil {
				return err
			}

			// Zero out the master key, it's not longer needed.
			masterKey.Zero()
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	return k, nil
}

// updateKeyMap....
func (k *KeyManager) updateKeyMap(keys [][keySize]byte) {
	for i := 0; i < keySize; i++ {
		k.keyMap[KeyType(i)] = keys[i]
	}
}

// loadDBKey....
func (k *KeyManager) loadDBKey(b *bolt.Bucket, targetKey KeyType,
	boltKey []byte, masterKey *snacl.SecretKey) error {
	decryptedKey, err := masterKey.Decrypt(b.Get(boltKey))
	if err != nil {
		return nil
	}

	var a [keySize]byte
	copy(decryptedKey, a[:])
	k.keyMap[targetKey] = a

	return nil
}

// TODO(roasbeef): Generalize...
// inverseHashTreeKeyDerivation.....
// We use a derivation technique based on Merkle Trees. However, we instead
// add a twist building the tree from the top down. The master key serves as
// our root, and is used to dervie subsequent child keys. Children are
// generated in a breadth-first manner until a sufficient number have been
// generated.
//           MASTER_KEY
//               |
//            SHA512()
//          /        \
//    CHILD_KEY        CHILD_KEY  (Child keys are 256 bits each)
//    SHA512()
//   /        \
// CHILD_KEY  CHILD_KEY
func inverseHashTreeKeyDerivation(masterKey [keySize]byte, numTargetKids int) ([][keySize]byte, error) {
	childKeys := make([][keySize]byte, 0, numTargetKids)

	// TODO(roasbeef): Edge case for non-power of two?
	// Derive child keys in a breadth-first manner.
	var parentQueue [][keySize]byte
	parentQueue = append(parentQueue, masterKey)
	numDerived := 0
	for numDerived < numTargetKids {
		currentParent := parentQueue[0]
		parentQueue = parentQueue[1:]

		derivedKeys, err := deriveChildren(currentParent)
		if err != nil {
			return nil, err
		}

		childKeys = append(childKeys, derivedKeys...)
		numDerived += len(derivedKeys)

		parentQueue = append(parentQueue, derivedKeys...)
	}

	return childKeys, nil
}

// deriveChildren...
func deriveChildren(parent [keySize]byte) ([][keySize]byte, error) {
	children := make([][keySize]byte, 2)
	nextLevel := sha512.Sum512(parent[:])

	var childA [keySize]byte
	var childB [keySize]byte
	copy(nextLevel[snacl.KeySize:], childA[:])
	copy(nextLevel[:snacl.KeySize], childB[:])

	children[0] = childA
	children[1] = childB
	return children, nil
}

// encryptChildKeys....
func encryptChildKeys(masterKey *snacl.SecretKey, serializedKeys [][keySize]byte) ([][]byte, error) {
	encryptedKeys := make([][]byte, len(serializedKeys))
	for i, key := range serializedKeys {
		ec, err := masterKey.Encrypt(key[:])
		if err != nil {
			return nil, err
		}
		encryptedKeys[i] = ec
	}
	return encryptedKeys, nil
}

func (k *KeyManager) Start() error {
	if atomic.AddInt32(&k.started, 1) != 1 {
		return nil
	}

	k.wg.Add(1)
	go k.requestHandler()

	return nil
}

func (k *KeyManager) Stop() error {
	if atomic.AddInt32(&k.shutdown, 1) != 1 {
		return nil
	}
	close(k.quit)
	k.wg.Wait()
	return nil
}

func (k *KeyManager) requestHandler() {
out:
	for {
		select {
		case keyReq := <-k.keyRequests:
			targetKey, ok := k.keyMap[keyReq.whichKey]
			if !ok {
				keyReq.errChan <- fmt.Errorf("invalid key type")
				continue
			}

			keyReq.replyChan <- &targetKey
		case <-k.quit:
			break out
		}
	}
	k.wg.Done()
}

func (k *KeyManager) FetchDocEncKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: WTrapKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

func (k *KeyManager) FetchWTrapKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: WTrapKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

func (k *KeyManager) FetchXTagKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: XTagKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchXIndKey....
func (k *KeyManager) FetchXIndKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: XIndKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchDHBlindKey....
func (k *KeyManager) FetchDHBlindKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: DHBlindKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchSTagKey...
func (k *KeyManager) FetchSTagKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: STagKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}
