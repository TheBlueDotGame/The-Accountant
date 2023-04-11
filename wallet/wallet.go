package wallet

import (
	"bytes"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/gob"

	"github.com/bartossh/The-Accountant/serializer"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// Wallet holds public and private key of the wallet owner.
type Wallet struct {
	Private ed25519.PrivateKey
	Public  ed25519.PublicKey
}

// New tries to creates a new Wallet or returns error otherwise.
func New() (Wallet, error) {
	private, public, err := newPair()
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{Private: private, Public: public}, nil
}

// DecodeGOBWallet tries to decode Wallet from gob representation or returns error otherwise.
func DecodeGOBWallet(data []byte) (Wallet, error) {
	var wallet Wallet
	gob.Register(elliptic.P256())
	decoder := gob.NewDecoder(bytes.NewReader(data))
	err := decoder.Decode(&wallet)
	if err != nil {
		return Wallet{}, err
	}
	return wallet, nil
}

// EncodeGOB tries to encodes Wallet in to the gob representation or returns error otherwise.
func (w *Wallet) EncodeGOB() ([]byte, error) {
	var content bytes.Buffer

	gob.Register(elliptic.P256())
	encoder := gob.NewEncoder(&content)
	err := encoder.Encode(w)
	if err != nil {
		return nil, err
	}
	return content.Bytes(), nil
}

// ChecksumLength returns checksum length.
func (w *Wallet) ChecksumLength() int {
	return checksumLength
}

// Version returns wallet version.
func (w *Wallet) Version() byte {
	return version
}

// Address creates address from the public key that contains wallet version and checksum.
func (w *Wallet) Address() string {
	versionedHash := append([]byte{version}, w.Public...)
	checksum := checksum(versionedHash)

	fullHash := append(versionedHash, checksum...)
	address := serializer.Base58Encode(fullHash)

	return string(address)
}

// Signe signs the message with Ed25519 signature.
// Returns digest hash sha256 and signature.
func (w *Wallet) Sign(message []byte) (digest [32]byte, signature []byte) {
	digest = sha256.Sum256(message)
	signature = ed25519.Sign(w.Private, digest[:])
	return digest, signature
}

// Verify verifies message ED25519 signature and hash.
// Uses hashing sha256.
func (w *Wallet) Verify(message, signature []byte, hash [32]byte) bool {
	digest := sha256.Sum256(message)
	if hash != digest {
		return false
	}
	return ed25519.Verify(w.Public, digest[:], signature)
}

func newPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return private, public, err
}

func checksum(payload []byte) []byte {
	firstHash := sha256.Sum256(payload)
	secondHash := sha256.Sum256(firstHash[:])

	return secondHash[:checksumLength]
}
