package fileoperations

import (
	"os"

	"github.com/bartossh/The-Accountant/wallet"
)

// RereadWallet reads wallet from the file.
func (h Helper) ReadWallet(path string) (wallet.Wallet, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return wallet.Wallet{}, err
	}
	w, err := wallet.DecodeGOBWallet(raw)
	if err != nil {
		return wallet.Wallet{}, err
	}
	return w, nil
}

// SaveWallet saves wallet to the file.
func (h Helper) SaveWallet(path string, w wallet.Wallet) error {
	raw, err := w.EncodeGOB()
	if err != nil {
		return err
	}
	return os.WriteFile(path, raw, 0644)
}
