package fileoperations

import (
	"crypto/ed25519"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"os"

	"github.com/bartossh/Computantis/wallet"
)

// Sealer offers behaviour to seal the bytes returning the signature on the data.
type Sealer interface {
	Encrypt(key, data []byte) ([]byte, error)
	Decrypt(key, data []byte) ([]byte, error)
}

// RereadWallet reads wallet from the file from GOB format.
// It uses decryption key to perform wallet decoding.
func (h Helper) ReadWallet() (wallet.Wallet, error) {
	raw, err := os.ReadFile(h.cfg.WalletPath)
	if err != nil {
		return wallet.Wallet{}, err
	}

	passwd, err := hex.DecodeString(h.cfg.WalletPasswd)
	if err != nil {
		return wallet.Wallet{}, err
	}

	opened, err := h.s.Decrypt(passwd, raw)
	if err != nil {
		return wallet.Wallet{}, err
	}

	w, err := wallet.DecodeGOBWallet(opened)
	if err != nil {
		return wallet.Wallet{}, err
	}
	return w, nil
}

// SaveWallet saves wallet to the file in GOB format.
// GOB file is secured cryptographically by the key,
// so it is safer option to move your wallet between machines
// in that format.
// This wallet can only be red by the Go wallet implementation.
// For transferring wallet to other implementations use PEM format.
func (h Helper) SaveWallet(w *wallet.Wallet) error {
	raw, err := w.EncodeGOB()
	if err != nil {
		return err
	}

	passwd, err := hex.DecodeString(h.cfg.WalletPasswd)
	if err != nil {
		return err
	}

	closed, err := h.s.Encrypt(passwd, raw)
	if err != nil {
		return err
	}

	return os.WriteFile(h.cfg.WalletPath, closed, 0644)
}

// SaveToPem saves wallet private and public key to the PEM format file.
// Saved files are like in the example:
// - PRIVATE: "your/path/name"
// - PUBLIC: "your/path/name.pub"
// Pem saved wallet is not sealed cryptographically and keys can be seen
// by anyone having access to the machine.
func (h Helper) SaveToPem(w *wallet.Wallet, filepath string) error {
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
func (h Helper) ReadFromPem(filepath string) (wallet.Wallet, error) {
	var w wallet.Wallet
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
