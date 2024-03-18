package cache

import (
	"bytes"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/allegro/bigcache"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
)

const (
	longevity       = time.Minute * 5 // TODO: read from config
	cleanupInterval = time.Minute * 3 // TODO: read from config
)

const (
	shards = 1 << 10
)

const (
	prefixTrx     = "trx"
	prefixAddress = "address"
)

var (
	ErrNilTransaction                          = errors.New("cannot cache a nil transaction")
	ErrFailedDecodingTrxKeyWrongNumOfParts     = errors.New("failed decoding transaction key, wrong number of parts")
	ErrFailedDecodingAddressKeyWrongNumOfParts = errors.New("failed decoding address key, wrong number of parts")
	ErrTrxAlreadyExists                        = errors.New("transaction already exists")
	ErrTransactionNotFound                     = errors.New("transaction not found")
	ErrProcessingFailed                        = errors.New("processing unexpected failure")
	ErrUnauthorized                            = errors.New("unauthorized")
)

// Hippocampus is a short lived memory that can transmit memorized entities to other long lived memory.
type Hippocampus struct {
	mem *bigcache.BigCache
}

// New creates the new Hippocampus on success or returns an error otherwise.
func New(maxEntrySize, maxCacheSizeMB int) (*Hippocampus, error) {
	c, err := bigcache.NewBigCache(bigcache.Config{
		Shards:           shards,
		LifeWindow:       longevity,
		CleanWindow:      cleanupInterval,
		HardMaxCacheSize: maxCacheSizeMB,
		MaxEntrySize:     maxEntrySize,
	})
	if err != nil {
		return nil, err
	}

	return &Hippocampus{mem: c}, nil
}

// SaveAwaitedTrx caches awaited transaction and reference of issuer and receiver address to the transaction hash.
func (h *Hippocampus) SaveAwaitedTransaction(trx *transaction.Transaction) error {
	if trx == nil {
		return ErrNilTransaction
	}
	trxKey := encodeTrxKey(trx.Hash[:])
	_, err := h.mem.Get(trxKey)

	if err == nil {
		return ErrTrxAlreadyExists
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
				errG = errors.Join(err, errors.New("save trx, get address to trx mapping"))
			}
			continue
		}
		if err := h.mem.Set(addressKey, add(awaited, trx.Hash)); err != nil {
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
			return transaction.Transaction{}, errors.Join(ErrTransactionNotFound, errors.New("get found no entity"))
		default:
			return transaction.Transaction{}, errors.Join(ErrProcessingFailed, errors.New("getting transaction by the key failed"))
		}
	}

	trx, err := transaction.Decode(raw)
	if err != nil {
		return transaction.Transaction{}, errors.Join(ErrProcessingFailed, errors.New("decoding failed"))
	}

	if trx.ReceiverAddress != address {
		return transaction.Transaction{}, errors.Join(ErrUnauthorized, errors.New("provided address isn't matching receiver address"))
	}

	if err := h.mem.Delete(trxKey); err != nil {
		switch errors.Is(err, bigcache.ErrEntryNotFound) {
		case true:
			return transaction.Transaction{}, errors.Join(ErrTransactionNotFound, errors.New("delete found no entity"))
		default:
			return transaction.Transaction{}, errors.Join(ErrProcessingFailed, errors.New("deleting trx failed"))
		}
	}

	var errs error
	addresses := []string{encodeAddressKey(trx.IssuerAddress), encodeAddressKey(trx.ReceiverAddress)}
	for _, addressKey := range addresses {
		awaited, err := h.mem.Get(addressKey)
		if err != nil {
			if errors.Is(err, bigcache.ErrEntryNotFound) {
				errs = err
			}
			continue
		}
		if len(awaited) == 0 {
			h.mem.Delete(addressKey)
			continue
		}
		if err := h.mem.Set(addressKey, remove(awaited, hash)); err != nil {
			errs = err
		}
	}

	return trx, errs
}

func (h *Hippocampus) ReadTransactions(address string) ([]transaction.Transaction, error) {
	addressKey := encodeAddressKey(address)
	awaited, err := h.mem.Get(addressKey)
	if err != nil {
		if errors.Is(err, bigcache.ErrEntryNotFound) {
			return nil, ErrTransactionNotFound
		}
		return nil, ErrProcessingFailed
	}

	if len(awaited) == 0 {
		if err := h.mem.Delete(addressKey); err != nil {
			return nil, err
		}
		return nil, ErrTransactionNotFound
	}

	hashes, err := read(awaited)
	if err != nil {
		return nil, ErrProcessingFailed
	}

	var errs error
	var removeHashes [][32]byte
	trxs := make([]transaction.Transaction, 0, len(hashes))
	for _, hash := range hashes {
		raw, err := h.mem.Get(encodeTrxKey(hash[:]))
		if err != nil {
			if errors.Is(err, bigcache.ErrEntryNotFound) {
				removeHashes = append(removeHashes, hash)
				continue
			}
			return nil, err
		}
		trx, err := transaction.Decode(raw)
		if err != nil {
			errs = err
			continue
		}
		trxs = append(trxs, trx)
	}

	if len(removeHashes) != 0 {
		for _, hash := range removeHashes {
			awaited, err := h.mem.Get(addressKey)
			if err != nil || len(awaited) == 0 {
				break
			}
			if err := h.mem.Set(addressKey, remove(awaited, hash)); err != nil {
				errs = err
			}
		}
	}

	return trxs, errs
}

// SaveBalance saves balance in to the mem cache.
func (h *Hippocampus) SaveBalance(a string, s spice.Melange) error {
	buf, err := s.Encode()
	if err != nil {
		return err
	}
	return h.mem.Set(a, buf)
}

// ReadBalance reads balance from the mem cache.
func (h *Hippocampus) ReadBalance(a string) (spice.Melange, error) {
	b, err := h.mem.Get(a)
	if err != nil {
		return spice.Melange{}, err
	}
	return spice.Decode(b)
}

// RemoveBalance removes balance from the cache.
func (h *Hippocampus) RemoveBalance(a string) error {
	return h.mem.Delete(a)
}

// Close closes the cache in a safe way allowing all the goroutines to finish their jobs and cleaning the heap.
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
	if len(originalValues) == 0 {
		return hexEncodeBytes(newValue[:])
	}
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
