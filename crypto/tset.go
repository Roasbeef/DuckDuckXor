package crypto

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha512"
	"encoding/binary"
)

// CalcTsetVals given an stag and document index, returns a triple containing
// the information required to look up, and decrypt document data from the t-set.
func CalcTsetVals(stag []byte, index uint32) ([1]byte, [16]byte, [37]byte) {
	prf := hmac.New(sha1.New, stag)
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
