package aeswrapper

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"errors"
	"io"
)

var (
	ErrInvalidKeyLength   = errors.New("invalid key length, must be longer then 32 bytes")
	ErrCipherFailure      = errors.New("cipher creation failure")
	ErrGCMFailure         = errors.New("gcm creation failure")
	ErrRandomNonceFailure = errors.New("random nonce creation failure")
	ErrOpenDataFailure    = errors.New("open data failure, cannot decrypt data")
)

const (
	nonceSize = 12
)

// Helper wraps EAS encryption and decryption.
// Uses Galois Counter Mode (GCM) for encryption and decryption.
type Helper struct{}

// Creates a new Helper.
func New() Helper {
	return Helper{}
}

// Encrypt encrypts data with key.
// Key must be at least 32 bytes long.
func (h Helper) Encrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 && len(key) != 16 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Join(ErrCipherFailure, err)
	}

	nonce := make([]byte, nonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, errors.Join(ErrRandomNonceFailure, err)
	}

	aesgcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(ErrGCMFailure, err)
	}

	ciphertext := aesgcm.Seal(nonce, nonce, data, nil)

	return ciphertext, nil
}

// Decrypt decrypts data with key.
// Key must be at least 32 bytes long.
func (h Helper) Decrypt(key, data []byte) ([]byte, error) {
	if len(key) != 32 && len(key) != 16 {
		return nil, ErrInvalidKeyLength
	}
	nonce, cipherText := data[:nonceSize], data[nonceSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, errors.Join(ErrCipherFailure, err)
	}

	aesGcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, errors.Join(ErrGCMFailure, err)
	}

	plaintext, err := aesGcm.Open(nil, nonce, cipherText, nil)
	if err != nil {
		return nil, errors.Join(ErrOpenDataFailure, err)
	}

	return plaintext, nil
}
