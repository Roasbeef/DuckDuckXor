package crypto

import (
	"crypto/sha512"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"hash"
	"strconv"

	"github.com/jacobsa/crypto/cmac"
	"github.com/roasbeef/perm-crypt"
)

// CalcTsetVals given an stag and document index, returns a triple containing
// the information required to look up, and decrypt document data from the t-set.
func CalcTsetVals(stag []byte, index uint32) ([1]byte, [16]byte, [37]byte) {
	// F(stag, index)
	prf, _ := cmac.New(stag)
	binary.Write(prf, binary.BigEndian, index)
	m := prf.Sum(nil)

	// b, L, K = H(F(stag, index))
	sha := sha512.New()
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

// XorBytes...
func XorBytes(x, y []byte) []byte {
	out := make([]byte, len(x))
	for i := 0; i < len(x); i++ {
		out[i] = x[i] ^ y[i]
	}
	return out
}

// permuteDocId computes the keyed permutation for the given document ID unique
// to each word. The keyed permutation is actually an encryption using AES-FFX
// a format preserving encryption scheme based on AES and a feisel network.
// The key for the AES-FFX scheme is derived from the word, making each
// encrypted document unique to each distinct word.
func PermuteDocId(word string, wPrf hash.Hash, docId uint32) []byte {
	// K_e = F(k_s, w)
	wPrf.Write([]byte(word))
	wordPermKey := wPrf.Sum(nil)

	wordPerm, err := aesffx.NewCipher(16, wordPermKey, nil)
	if err != nil {
		// TODO(roasbeef): handle err
	}

	// e = Enc(Ke, ind)
	xindString := strconv.FormatUint(uint64(docId), 16)
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

// UnpermuteDocId computes the keyed inverse-permutation for the given document
// ID unique to each word. The keyed permutation is actually an encryption
// using AES-FFX a format preserving encryption scheme based on AES and a
// feisel network. The key for the AES-FFX scheme is derived from the word,
// making each encrypted document unique to each distinct word.
func UnpermuteDocId(word string, wPrf hash.Hash, docId []byte) (uint32, error) {
	// K_e = F(k_s, w)
	wPrf.Write([]byte(word))
	wordPermKey := wPrf.Sum(nil)

	wordPerm, err := aesffx.NewCipher(16, wordPermKey, nil)
	if err != nil {
		return uint32(0), err
	}

	// e = Enc(Ke, ind)
	xindString := hex.EncodeToString(docId)
	permutedId, err := wordPerm.Decrypt(xindString)
	if err != nil {
		return uint32(0), err
	}

	dId, err := strconv.ParseUint(permutedId, 16, 32)
	if err != nil {
		return uint32(0), err
	}

	return uint32(dId), nil
}
