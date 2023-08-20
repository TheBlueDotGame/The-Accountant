package fileoperations

import (
	"encoding/hex"
	"os"

	"github.com/bartossh/Computantis/wallet"
)

// Sealer offers behaviour to seal the bytes returning the signature on the data.
type Sealer interface {
	Encrypt(key, data []byte) ([]byte, error)
	Decrypt(key, data []byte) ([]byte, error)
}

// RereadWallet reads wallet from the file.
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

// SaveWallet saves wallet to the file.
func (h Helper) SaveWallet(w wallet.Wallet) error {
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
