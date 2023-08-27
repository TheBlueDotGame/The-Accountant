package wallet

import (
	"bytes"
	"crypto/ed25519"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/gob"
	"encoding/pem"
	"errors"
	"os"

	"github.com/bartossh/Computantis/serializer"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// Wallet holds public and private key of the wallet owner.
type Wallet struct {
	Private ed25519.PrivateKey `json:"private" bson:"private"`
	Public  ed25519.PublicKey  `json:"public" bson:"public"`
}

// New tries to creates a new Wallet or returns error otherwise.
func New() (Wallet, error) {
	private, public, err := newPair()
	if err != nil {
		return Wallet{}, err
	}
	return Wallet{Private: private, Public: public}, nil
}

// SaveToPem saves wallet private and public key to the PEM format file.
// Saved files are like in the example:
// - PRIVATE: "your/path/name"
// - PUBLIC: "your/path/name.pub"
func (w *Wallet) SaveToPem(filepath string) error {
	prv, err := x509.MarshalPKCS8PrivateKey(w.Private)
	if err != nil {
		return err
	}
	pub, err := x509.MarshalPKIXPublicKey(w.Public)
	if err != nil {
		return err
	}
	blockPrv := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: prv,
	}
	blockPub := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: pub,
	}
	if err := os.WriteFile(filepath, pem.EncodeToMemory(blockPrv), 0644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath+".pub", pem.EncodeToMemory(blockPub), 0644); err != nil {
		return err
	}
	return nil
}

// ReadFromPem creates Wallet from PEM format file.
// Uses both private and public key.
// Provide the path to a file without specifying the extension : <your/path/name".
func ReadFromPem(filepath string) (Wallet, error) {
	var w Wallet
	rawPub, err := os.ReadFile(filepath + ".pub")
	if err != nil {
		return w, err
	}
	rawPrv, err := os.ReadFile(filepath)
	if err != nil {
		return w, err
	}

	blockPub, _ := pem.Decode(rawPub)
	if blockPub == nil || blockPub.Type != "PUBLIC KEY" {
		return w, errors.New("cannot decode public key from PEM format")
	}
	pub, err := x509.ParsePKIXPublicKey(blockPub.Bytes)
	if err != nil {
		return w, err
	}
	blockPrv, _ := pem.Decode(rawPrv)
	if blockPrv == nil || blockPrv.Type != "PRIVATE KEY" {
		return w, errors.New("cannot decode private key from PEM format")
	}
	prv, err := x509.ParsePKCS8PrivateKey(blockPrv.Bytes)
	if err != nil {
		return w, err
	}
	var ok bool
	w.Public, ok = pub.(ed25519.PublicKey)
	if !ok {
		return w, errors.New("cannot cast x509 decoded parsed key to ed25519 public key")
	}
	w.Private, ok = prv.(ed25519.PrivateKey)
	if !ok {
		return w, errors.New("cannot cast x509 decoded parsed key to ed25519 private key")
	}
	return w, nil
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
	vers := append([]byte{version}, w.Public...)
	cs := checksum(vers)

	full := append(vers, cs...)
	address := serializer.Base58Encode(full)

	return string(address)
}

// Sign signs the message with Ed25519 signature.
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
	if !bytes.Equal(hash[:], digest[:]) {
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
