https://powcoder.com
代写代考加微信 powcoder
Assignment Project Exam Help
Add WeChat powcoder
// Library code for Diffie-Hellman key generation and exchange. This code uses
// the NIST standard P-256 curve as well as SHA-256 for key derivation.
//
// SECURITY WARNING: This code is meant for educational purposes and may
// contain vulnerabilities or other bugs. Please do not use it for
// security-critical applications.
//
// GRADING NOTES: your code will be evaluated using this code. If you modify
// this code remember that the default version will be used for grading. You
// should add functions as needed in chatter.go or other supplemental files,
// not here.
//
// Original version
// Joseph Bonneau February 2019

package chatterbox

import (
	"crypto/elliptic"
	"crypto/sha256"
	"fmt"
	"math/big"
)

const FINGERPRINT_LENGTH = 16 //128-bit key fingerprints

// Curve in use is NIST-P256
func curve() elliptic.Curve {
	return elliptic.P256()
}

// KeyPair represents a public and private key pair. In this application
// we are only doing Diffie-Hellman exchanges. The public key is g^x
// and the private key is the exponent x.
type KeyPair struct {
	PrivateKey PrivateKey
	PublicKey  PublicKey
}

// PrivateKey represents a "private key". This is simply a random buffer
// representing a secret exponent.
type PrivateKey struct {
	Key []byte
}

// PrivateKey represents a public key, which is an elliptic curve point.
// Represented by two integers X, Y.
type PublicKey struct {
	X *big.Int
	Y *big.Int
}

// NewKeyPair creates a new key pair. It panics in the case of
// randomness failure.
func NewKeyPair() *KeyPair {
	priv, x, y, err := elliptic.GenerateKey(curve(), RandomnessSource())

	if err != nil {
		panic(err)
	}

	return &KeyPair{
		PrivateKey: PrivateKey{Key: priv},
		PublicKey:  PublicKey{X: x, Y: y},
	}
}

// Zeroize overwrites the buffer storing a private key with 0 bytes.
func (k *PrivateKey) Zeroize() {
	for i := range k.Key {
		k.Key[i] = 0
	}
}

// Zeroize overwrites the buffer storing a private key with 0 bytes.
func (kp *KeyPair) Zeroize() {
	kp.PrivateKey.Zeroize()
}

// String representation of a public key.
func (kp *KeyPair) String() string {
	return fmt.Sprintf("Public key : % 0X\nPrivate key: % 0X\n", elliptic.Marshal(curve(), kp.PublicKey.X, kp.PublicKey.Y), kp.PrivateKey.Key)
}

// Fingerprint returns a hash representation of a public key.
// This is useful for a shorter value that uniquely identifies the key,
// but cannot be used to recover the key itself.
func (k *PublicKey) Fingerprint() []byte {
	h := sha256.New()
	h.Write(elliptic.Marshal(curve(), k.X, k.Y))
	return h.Sum(nil)[:FINGERPRINT_LENGTH]
}

// Fingerprint returns the fingerprint of the underlying public key.
func (kp *KeyPair) Fingerprint() []byte {
	return kp.PublicKey.Fingerprint()
}

// DHCombine performs a Diffie-Hellman exchange between a public key and a
// private key. For example, if the public key is g^a and the private key is
// b, this returns a key derived by hashing g^ab. This is immediatly converted
// to a SymmetricKey.
func DHCombine(publicKey *PublicKey, privateKey *PrivateKey) *SymmetricKey {
	x, y := curve().ScalarMult(publicKey.X, publicKey.Y, privateKey.Key)
	h := sha256.New()
	h.Write(elliptic.Marshal(curve(), x, y))

	return &SymmetricKey{
		h.Sum(nil)[:SYMMETRIC_KEY_LENGTH],
	}
}
