package wallet

import (
	"bytes"
	"crypto/ed25519"
	"crypto/sha256"
	"errors"

	"github.com/bartossh/Computantis/serializer"
)

// Helper provides wallet helper functionalities without knowing about wallet private and public keys.
type Helper struct{}

// NewVerifier creates new wallet Helper verifier.
func NewVerifier() Helper {
	return Helper{}
}

// AddressToPubKey creates ED25519 public key from address, or returns error otherwise.
func (h Helper) AddressToPubKey(address string) (ed25519.PublicKey, error) {
	pubKey, err := serializer.Base58Decode([]byte(address))
	if err != nil {
		return []byte{}, err
	}
	if len(pubKey)-checksumLength < 1 {
		return []byte{}, errors.New("address of invalid length")
	}
	actualChecksum := pubKey[len(pubKey)-checksumLength:]
	version := pubKey[0]
	pubKey = pubKey[1 : len(pubKey)-checksumLength]
	targetChecksum := checksum(append([]byte{version}, pubKey...))

	if !bytes.Equal(actualChecksum, targetChecksum) {
		return []byte{}, errors.New("address checksum is not equal")
	}

	return pubKey, nil
}

// Verify verifies if message is signed by given key and hash is equal.
func (h Helper) Verify(message, signature []byte, hash [32]byte, address string) error {
	digest := sha256.Sum256(message)
	if !bytes.Equal(hash[:], digest[:]) {
		return errors.New("hash is corrupted")
	}

	pubKey, err := h.AddressToPubKey(address)
	if err != nil {
		return err
	}

	if !ed25519.Verify(pubKey, digest[:], signature) {
		return errors.New("message signature isn't valid")
	}
	return nil
}
