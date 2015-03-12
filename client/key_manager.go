package main

import (
	"crypto/rand"
	"crypto/sha512"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/boltdb/bolt"
	"github.com/conformal/btcwallet/snacl"
)

// An enum containing our various key types.
type KeyType int

const (
	WTrapKey   KeyType = iota // K_s
	XTagKey                   // K_x
	XIndKey                   // K_i
	DHBlindKey                // K_z
	STagKey                   // K_t
	DocEncKey                 // Wrapped in snacl.CryptoKey
	PermuteTweak
)
const (
	numKeys   = 6
	keySize   = 32
	tweakSize = 32 // Typically truncated to 4 bytes.
)

// A helper map to quickly identify the db key for a particular crypto key.
var keyToBoltKey = map[KeyType][]byte{
	WTrapKey:     []byte("k_s"), // Truncated to 16 bytes.
	XTagKey:      []byte("k_x"), // Truncated to 16 bytes.
	XIndKey:      []byte("k_i"), // Truncated to 16 bytes.
	DHBlindKey:   []byte("k_z"), // Truncated to 16 bytes.
	STagKey:      []byte("k_t"),
	DocEncKey:    []byte("doc_key"),
	PermuteTweak: []byte("tweak"),
}
var cryptoKeyBucket = []byte("cryptoKeys")

var masterKeyName = []byte("master")

// keyRequestMessage sends a message to the keyManager requesting a particular
// crypto key.
type keyRequestMessage struct {
	whichKey KeyType
	// TODO(roasbeef): BYOB everywhere?
	replyChan chan *[keySize]byte
	errChan   chan error // should be buffered
}

// KeyManager handles the storage, derivation, and querying of of various
// cryptographic keys.
type KeyManager struct {
	wg          sync.WaitGroup
	keyMap      map[KeyType][keySize]byte
	keyRequests chan keyRequestMessage

	db *bolt.DB

	quit     chan struct{}
	started  int32
	shutdown int32
}

// NewKeyManager creates a new KeyManager. The KeyManager is responsible for
// securely storing, deriving, and answering queries to retrieve our various
// cryptographic keys.
func NewKeyManager(db *bolt.DB, passphrase []byte) (*KeyManager, error) {
	var err error
	k := &KeyManager{
		db:          db,
		quit:        make(chan struct{}),
		keyMap:      make(map[KeyType][keySize]byte),
		keyRequests: make(chan keyRequestMessage),
	}

	// If our key bucket is present, then we've already done the initial set-up.
	// Otherwise, perform the derivation, and set-up steps for our keys.
	// TODO(roasbeef): Pass this in instead?
	var found bool
	err = db.View(func(tx *bolt.Tx) error {
		keyBucket := tx.Bucket(cryptoKeyBucket)
		found = !(keyBucket == nil)
		return nil
	})
	if err != nil {
		return nil, err
	}

	if !found {
		err = k.performInitialSetup(passphrase)
	} else {
		err = k.performRegularSetup(passphrase)
	}

	if err != nil {
		return nil, err
	}

	return k, nil
}

// performInitialSetup peforms regular setup of the KeyManager. In order to
// carry out this regular setup, we attempt to re-derive the master key from
// the passed passphrase. If this fails, then we have the incorrect passphrase.
// Otherwise, we retrieve the stored child keys, decrypt and store them.
func (k *KeyManager) performRegularSetup(passphrase []byte) error {
	// else open bucket, decrypt, all, load into memory
	return k.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(cryptoKeyBucket)

		// Grab the marshalled master key paramaters.
		masterKeyParams := b.Get(masterKeyName)

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

		// Decrypt the keys and load into memory.
		if err := k.loadDBKeys(b, &masterKey); err != nil {
			return err
		}

		// Zero out the master key, it's not longer needed.
		masterKey.Zero()
		return nil
	})
}

// performRegularSetup peforms regular initialization of the KeyManager.
// In order to initialize, we derive our initial master key from the passed
// passphrase. We then deterministicically generated related child keys for our
// PRFs and symmetric encryption schemes. The derived children are then
// encrypted using the master key before being stored in the database.
func (k *KeyManager) performInitialSetup(passphrase []byte) error {
	// Derive our master key using scrypt.
	masterKey, err := snacl.NewSecretKey(
		&passphrase, snacl.DefaultN, snacl.DefaultR, snacl.DefaultP,
	)
	if err != nil {
		return err
	}

	// Derive child keys from our master key. These are the keys we
	// end up using for symmetric encryption and our PRFs.
	var mKey [keySize]byte
	copy(mKey[:], masterKey.Key[:])
	childKeys, err := inverseHashTreeKeyDerivation(mKey, numKeys)
	if err != nil {
		return err
	}

	// We never store the master key, only paramters from which we
	// can re-derive it. This master key is used to encrypt the
	// derived child keys when they're stored in the DB.
	encryptedKeys, err := encryptChildKeys(masterKey, childKeys)
	if err != nil {
		return err
	}

	serializedMasterKey := masterKey.Marshal()

	// Generate a random tweak for use with AES-FFX.
	var permuteTweak [tweakSize]byte
	_, err = rand.Read(permuteTweak[:])
	if err != nil {
		return err
	}

	// Commit the encrypted keys to the DB.
	err = k.db.Update(func(tx *bolt.Tx) error {
		b, err := tx.CreateBucket(cryptoKeyBucket)
		if err != nil {
			return err
		}
		// Store the encrypted master key parameters.
		if err = b.Put(masterKeyName, serializedMasterKey); err != nil {
			return nil
		}

		// Store each individual key
		if err = k.storeEncryptedKeys(b, encryptedKeys); err != nil {
			return err
		}

		// Store our tweak.
		if err = b.Put(keyToBoltKey[PermuteTweak], permuteTweak[:]); err != nil {
			return nil
		}

		return nil
	})
	if err != nil {
		return err
	}

	// zero out the master key, we don't need it anymore.
	masterKey.Zero()
	// Finally create our map, then we're all ready to go
	k.updateKeyMap(childKeys)

	return nil
}

// updateKeyMap upadtes our internal keymap with each of the derived keys.
func (k *KeyManager) updateKeyMap(keys [][keySize]byte) {
	for i := 0; i < numKeys; i++ {
		k.keyMap[KeyType(i)] = keys[i]
	}
}

// loadDBKeys loads all of our cryptographic keys from the passed boltDB bucket.
// Each key is decrypted using the master key before its loaded into our key map.
func (k *KeyManager) loadDBKeys(b *bolt.Bucket, masterKey *snacl.SecretKey) error {
	for i := 0; i < numKeys; i++ {
		targetKey := KeyType(i)
		dbKey := keyToBoltKey[targetKey]
		decryptedKey, err := masterKey.Decrypt(b.Get(dbKey))
		if err != nil {
			return nil
		}

		var a [keySize]byte
		copy(a[:], decryptedKey)
		k.keyMap[targetKey] = a
	}

	// Recover our permutation tweak value.
	tweak := b.Get(keyToBoltKey[PermuteTweak])
	var c [keySize]byte
	copy(c[:], tweak)
	k.keyMap[PermuteTweak] = c

	return nil
}

// storeEncryptedKeys stores each encrypted child key in the passed botlDB
// bucket.
func (k *KeyManager) storeEncryptedKeys(b *bolt.Bucket, keys [][]byte) error {
	for i := 0; i < numKeys; i++ {
		if err := b.Put(keyToBoltKey[KeyType(i)], keys[i]); err != nil {
			return err
		}
	}

	return nil
}

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

		derivedKeys := deriveChildren(currentParent)

		childKeys = append(childKeys, derivedKeys...)
		numDerived += len(derivedKeys)

		parentQueue = append(parentQueue, derivedKeys...)
	}

	return childKeys, nil
}

// deriveChildren derives two child keys from a parent key.
// Derivation is performed by interpreting the digest of SHA-512(parent) as
// two 256-bit keys.
func deriveChildren(parent [keySize]byte) [][keySize]byte {
	children := make([][keySize]byte, 2)
	nextLevel := sha512.Sum512(parent[:])

	var childA [keySize]byte
	var childB [keySize]byte
	copy(childA[:], nextLevel[snacl.KeySize:])
	copy(childB[:], nextLevel[:snacl.KeySize])

	children[0] = childA
	children[1] = childB
	return children
}

// encryptChildKeys encrypts each of the passed serialized child keys using the
// masterKey. Each child key should have been deterministicically generated from
// the masterKey.
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

// Start kicks off the KeyManager starting any helper goroutines.
func (k *KeyManager) Start() error {
	if atomic.AddInt32(&k.started, 1) != 1 {
		return nil
	}

	k.wg.Add(1)
	go k.requestHandler()

	return nil
}

// Stop gracefully stops the KeyManager and its related helper goroutines.
func (k *KeyManager) Stop() error {
	if atomic.AddInt32(&k.shutdown, 1) != 1 {
		return nil
	}
	close(k.quit)
	k.wg.Wait()
	return nil
}

// requestHandler handles incoming requests for cryptographic keys.
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

// FetchDocEncKey retrieves they key designated for encrypting our corpus of
// documents.
func (k *KeyManager) FetchDocEncKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: WTrapKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchWTrapKey retrieves the key designated for generating wtraps via aPRF.
// These wtraps are then used to uniquely encrypt a document ID for a
// particular word.
func (k *KeyManager) FetchWTrapKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: WTrapKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchXTagKey retrieves the key designated for generating an xtrap via a PRF.
func (k *KeyManager) FetchXTagKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: XTagKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchXIndKey retrieves the key designated for generating xind's via a PRF.
func (k *KeyManager) FetchXIndKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: XIndKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchDHBlindKey retrieves the key designated for generating tokens used
// for diffie-helman blind exponentation in the OXT protocol.
func (k *KeyManager) FetchDHBlindKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: DHBlindKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}

// FetchSTagKey retrieves the key designated for creating s-tags.
func (k *KeyManager) FetchSTagKey() *[keySize]byte {
	resp := make(chan *[keySize]byte)
	req := keyRequestMessage{whichKey: STagKey, replyChan: resp}
	k.keyRequests <- req

	return <-resp
}
