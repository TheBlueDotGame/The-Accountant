package cache

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/allegro/bigcache"
	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/transaction"
)

const (
	longevity       = time.Minute * 30
	cleanupInterval = time.Minute * 10
)

const (
	shards = 1024
)

const (
	prefixTrx     = "trx"
	prefixAddress = "address"
)

var (
	ErrNilTransaction                          = errors.New("cannot cache a nil transaction")
	ErrFailedDecodingTrxKeyWrongNumOfParts     = errors.New("failed decoding transaction key, wrong number of parts")
	ErrFailedDecodingAddressKeyWrongNumOfParts = errors.New("failed decoding address key, wrong number of parts")
	ErrTrxAlredyExists                         = errors.New("transaction already exists")
	ErrTransactionNotFound                     = errors.New("transaction not found")
	ErrProccessingFailed                       = errors.New("processing unexpected failure")
	ErrUnauthorized                            = errors.New("unauthorized")
)

// Hippocampus is a short lived memory that can transmit memorized entities to other long lived memory.
type Hippocampus struct {
	mem *bigcache.BigCache
	log logger.Logger
}

// New creates the new Hippocampus on success or returns an error otherwise.
func New(log logger.Logger, maxEntrySize, maxCacheSizeMB int) (*Hippocampus, error) {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           shards,
		LifeWindow:       longevity,
		CleanWindow:      cleanupInterval,
		HardMaxCacheSize: maxCacheSizeMB, // 1GB
		MaxEntrySize:     maxEntrySize,   // B
	})
	if err != nil {
		return nil, err
	}

	return &Hippocampus{mem: c, log: log}, nil
}

// SaveAwaitedTrx caches awaited transaction and reference of issuer and receiver address to the transaction hash.
func (h *Hippocampus) SaveAwaitedTransaction(trx *transaction.Transaction) error {
	if trx == nil {
		return ErrNilTransaction
	}

	trxKey := encodeTrxKey(trx.Hash[:])

	_, err := h.mem.Get(trxKey)
	if err == nil {
		return ErrTrxAlredyExists
	}
	if !errors.Is(err, bigcache.ErrEntryNotFound) {
		return err
	}

	raw, err := trx.Encode()
	if err != nil {
		return errors.Join(err, errors.New("save trx encoding"))
	}

	if err := h.mem.Set(trxKey, raw); err != nil {
		return errors.Join(err, errors.New("save trx set"))
	}

	addresses := []string{encodeAddressKey(trx.IssuerAddress), encodeAddressKey(trx.ReceiverAddress)}
	if trx.IssuerAddress == trx.ReceiverAddress {
		addresses = addresses[1:]
	}

	var errG error
	for _, addressKey := range addresses {
		if addressKey == "" {
			continue
		}
		awaited, err := h.mem.Get(addressKey)
		if err != nil {
			switch errors.Is(err, bigcache.ErrEntryNotFound) {
			case true:
				h.mem.Set(addressKey, set(trx.Hash))
			default:
				h.log.Error(fmt.Sprintf("failed to add trx to address cache for trx %v, %s", trx.Hash, err))
				fmt.Printf("awaited %v\n", awaited)
				errG = errors.Join(err, errors.New("save trx, get address to trx mapping"))
			}
			continue
		}
		if err := h.mem.Set(addressKey, add(awaited, trx.Hash)); err != nil {
			h.log.Error(fmt.Sprintf("failed to add trx to address cache for trx %v, %s", trx.Hash, err))
			errG = errors.Join(err, errors.New("save trx, set address to trx mapping"))
		}
	}

	return errG
}

// RemoveAwaitedTransaction removes awaited transaction and the issuer and receiver address reference to the transaction.
func (h *Hippocampus) RemoveAwaitedTransaction(hash [32]byte, address string) (transaction.Transaction, error) {
	trxKey := encodeTrxKey(hash[:])

	raw, err := h.mem.Get(trxKey)
	if err != nil {
		switch errors.Is(err, bigcache.ErrEntryNotFound) {
		case true:
			return transaction.Transaction{}, ErrTransactionNotFound
		default:
			h.log.Error(fmt.Sprintf("error removing transaction with hash %v, lookup failed, %s", hash, err))
			return transaction.Transaction{}, errors.Join(ErrProccessingFailed, errors.New("getting transaction by the key failed"))
		}
	}

	trx, err := transaction.Decode(raw)
	if err != nil {
		h.log.Error(fmt.Sprintf("error removing transaction with hash %v, decoding failed, %s", hash, err))
		return transaction.Transaction{}, errors.Join(ErrProccessingFailed, errors.New("decoding failed"))
	}

	if trx.ReceiverAddress != address {
		h.log.Error(fmt.Sprintf("error removing transaction with hash %v, wrong receiver address", hash))
		return transaction.Transaction{}, ErrUnauthorized
	}

	if err := h.mem.Delete(trxKey); err != nil {
		switch errors.Is(err, bigcache.ErrEntryNotFound) {
		case true:
			return transaction.Transaction{}, ErrTransactionNotFound
		default:
			h.log.Error(fmt.Sprintf("error removing transaction with hash %v, deletion failed, %s", hash, err))
			return transaction.Transaction{}, errors.Join(ErrProccessingFailed, errors.New("deleteing trx failed"))
		}
	}
	addresses := []string{encodeAddressKey(trx.IssuerAddress), encodeAddressKey(trx.ReceiverAddress)}
	for _, addressKey := range addresses {
		awaited, err := h.mem.Get(addressKey)
		if err != nil {
			h.log.Error(fmt.Sprintf("error removing address reference to transaction with hash %v, lookup failed, %s", hash, err))
			continue
		}
		if len(awaited) == 0 {
			if err := h.mem.Delete(addressKey); err != nil {
				h.log.Error(fmt.Sprintf("error removing address reference to transaction with hash %v, deletion failed, %s", hash, err))
			}
			continue
		}
		if err := h.mem.Set(addressKey, remove(awaited, hash)); err != nil {
			h.log.Error(fmt.Sprintf("error removing address reference to transaction with hash %v, remove failed, %s", hash, err))
		}
	}

	return trx, nil
}

func (h *Hippocampus) ReadTransactions(address string) ([]transaction.Transaction, error) {
	awaited, err := h.mem.Get(encodeAddressKey(address))
	if err != nil {
		h.log.Error(fmt.Sprintf("error reading address [ %s ] reference to transaction, lookup failed, %s", address, err))
		return nil, ErrTransactionNotFound
	}
	if len(awaited) == 0 || awaited == nil {
		if err := h.mem.Delete(encodeAddressKey(address)); err != nil {
			h.log.Error(fmt.Sprintf("error reading address [ %s ] reference to transaction, removing empty reference failed, %s", address, err))
		}
		return nil, ErrTransactionNotFound
	}

	hashes, err := read(awaited)
	if err != nil {
		h.log.Error(fmt.Sprintf("error read transaction from bytes slice %s", err))
		return nil, ErrProccessingFailed
	}

	trxs := make([]transaction.Transaction, 0, len(hashes))
	for _, hash := range hashes {
		raw, err := h.mem.Get(encodeTrxKey(hash[:]))
		if err != nil {
			h.log.Error(fmt.Sprintf("error reading transaction with hash %v, %s", hash, err))
			continue
		}
		trx, err := transaction.Decode(raw)
		if err != nil {
			h.log.Error(fmt.Sprintf("error decoding transaction with hash %v, %s", hash, err))
			continue
		}
		trxs = append(trxs, trx)
	}

	return trxs, nil
}

// Close closes the cache in a safe way allowing all the goroutines to finish their jobs.
func (h *Hippocampus) Close() error {
	return h.mem.Close()
}

func hexEncodeBytes(src []byte) []byte {
	dst := make([]byte, hex.EncodedLen(len(src)))
	hex.Encode(dst, src)
	return dst
}

func hexDecodeBytes(src []byte) ([]byte, error) {
	dst := make([]byte, hex.DecodedLen(len(src)))
	_, err := hex.Decode(dst, src)
	return dst, err
}

func encodeAddressKey(src string) string {
	return fmt.Sprintf("%s-%s", prefixAddress, src)
}

func decodeAddressKey(src string) (string, error) {
	parts := strings.Split(src, "-")
	if len(parts) != 2 {
		return "", ErrFailedDecodingAddressKeyWrongNumOfParts
	}
	return parts[1], nil
}

func encodeTrxKey(src []byte) string {
	return fmt.Sprintf("%s-%s", prefixTrx, hex.EncodeToString(src))
}

func decodeTrxKey(src string) ([]byte, error) {
	parts := strings.Split(src, "-")
	if len(parts) != 2 {
		return nil, ErrFailedDecodingTrxKeyWrongNumOfParts
	}
	return hex.DecodeString(parts[1])
}

func set(newValue [32]byte) []byte {
	return hexEncodeBytes(newValue[:])
}

func add(originalValues []byte, newValue [32]byte) []byte {
	return append(originalValues, append([]byte{','}, hexEncodeBytes(newValue[:])...)...)
}

func read(val []byte) ([][32]byte, error) {
	var err error
	hexValues := bytes.Split(val, []byte{','})
	hashes := make([][32]byte, 0, len(hexValues))
	for _, hexBytes := range hexValues {
		if len(hexBytes) == 0 {
			continue
		}
		dec, errL := hexDecodeBytes(hexBytes)
		if errL != nil {
			err = errL
			continue
		}
		hashes = append(hashes, [32]byte(dec))
	}
	return hashes, err
}

func remove(values []byte, removeValue [32]byte) []byte {
	enc := hexEncodeBytes(removeValue[:])
	newValues := make([]byte, 0, len(values))
	for _, val := range bytes.Split(values, []byte{','}) {
		if bytes.Equal(val, enc) {
			continue
		}
		newValues = append(newValues, append([]byte{','}, val...)...)
	}
	return newValues
}
