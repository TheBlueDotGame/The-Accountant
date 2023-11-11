package notaryserver

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/dgraph-io/badger/v4"
	"github.com/gofiber/fiber/v2"

	"github.com/bartossh/Computantis/accountant"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/versioning"
)

// AliveResponse is a response for alive and version check.
type AliveResponse struct {
	APIVersion string `json:"api_version"`
	APIHeader  string `json:"api_header"`
	Alive      bool   `json:"alive"`
}

func (s *server) alive(c *fiber.Ctx) error {
	return c.JSON(
		AliveResponse{
			Alive:      true,
			APIVersion: versioning.ApiVersion,
			APIHeader:  versioning.Header,
		})
}

// TransactionConfirmProposeResponse is a response for transaction propose.
type TransactionConfirmProposeResponse struct {
	TrxHash [32]byte `json:"trx_hash"`
	Success bool     `json:"success"`
}

func (s *server) propose(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(proposeTrxTelemetryHistogram, time.Since(t))

	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if err := checkNotEmpty(&trx); err != nil {
		return err
	}

	if err := trx.VerifyIssuer(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, verification failed: %s", err))
		return fiber.ErrBadRequest
	}

	switch trx.IsContract() {
	case true:
		if err := s.saveAwaitedTrx(c.Context(), &trx); err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, saving awaited trx for issuer [ %s ], %s", trx.IssuerAddress, err))
			return fiber.ErrConflict
		}
		addresses := []string{trx.IssuerAddress, trx.ReceiverAddress}
		if err := s.pub.PublishAddressesAwaitingTrxs(addresses, s.nodePublicURL); err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, publishing awaited trx for addresses %v, failed, %s", addresses, err))
		}
	default:
		if len(trx.Data) > s.dataSize {
			s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(trx.Data)))
			return fiber.ErrBadRequest
		}
		vrx, err := s.acc.CreateLeaf(c.Context(), &trx)
		if err != nil {
			s.log.Error(fmt.Sprintf("propose endpoint, creating leaf: %s", err))
			return fiber.ErrBadRequest
		}
		go func(v *accountant.Vertex) {
			s.vrxGossipCh <- v
		}(&vrx)
	}

	return c.JSON(TransactionConfirmProposeResponse{
		Success: true,
		TrxHash: trx.Hash,
	})
}

func (s *server) confirm(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(confirmTrxTelemetryHistogram, time.Since(t))

	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, failed to parse request body, %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if err := checkNotEmpty(&trx); err != nil {
		return err
	}

	if err := trx.VerifyIssuerReceiver(s.verifier); err != nil {
		s.log.Error(fmt.Sprintf(
			"confirm endpoint, failed to verify trx hash %v from receiver [ %s ], %s", trx.Hash, trx.ReceiverAddress, err.Error(),
		))
		return fiber.ErrBadRequest
	}

	if err := s.removeAwaitedTrx(&trx); err != nil {
		s.log.Error(
			fmt.Sprintf(
				"confirm endpoint, failed to remove awaited trx hash %v from receiver [ %s ] , %s", trx.Hash, trx.ReceiverAddress, err.Error(),
			))
		return fiber.ErrBadRequest
	}

	vrx, err := s.acc.CreateLeaf(c.Context(), &trx)
	if err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, creating leaf: %s", err))
		return fiber.ErrBadRequest
	}

	go func(v *accountant.Vertex) {
		s.vrxGossipCh <- v
	}(&vrx)

	return c.JSON(TransactionConfirmProposeResponse{
		Success: true,
		TrxHash: trx.Hash,
	})
}

// TransactionsRejectRequest is a request to reject a transactions.
type TransactionsRejectRequest struct {
	Address      string                    `json:"address"`
	Transactions []transaction.Transaction `json:"transaction"`
	Data         []byte                    `json:"data"`
	Signature    []byte                    `json:"signature"`
	Hash         [32]byte                  `json:"hash"`
}

// TransactionsRejectResponse is a response for transaction reject.
type TransactionsRejectResponse struct {
	TrxHashes [][32]byte `json:"trx_hash"`
	Success   bool       `json:"success"`
}

func (s *server) reject(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(rejectTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRejectRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("reject endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.verifier.Verify(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("reject endpoint, failed to verify signature for address: %s, %s", req.Address, err))
		return fiber.ErrForbidden
	}

	for _, trx := range req.Transactions {
		err := checkNotEmpty(&trx)
		if err != nil {
			s.log.Error(fmt.Sprintf("reject endpoint, received empty transaction from address [ %v ]", req.Address))
			return fiber.ErrBadRequest
		}
		if trx.ReceiverAddress != req.Address {
			s.log.Error(
				fmt.Sprintf("reject endpoint, received transaction for address [ %v ] from address [ %v ]", trx.ReceiverAddress, req.Address),
			)
			return fiber.ErrBadRequest
		}
		if len(trx.ReceiverSignature) != 0 {
			s.log.Error(
				fmt.Sprintf(
					"reject endpoint, received transaction with receiver signature [ %v ] from address [ %v ]",
					trx.ReceiverSignature, req.Address),
			)
			return fiber.ErrBadRequest
		}
	}
	hashes := make([][32]byte, 0, len(req.Transactions))
	for _, trx := range req.Transactions {
		if err := s.removeAwaitedTrx(&trx); err != nil {
			s.log.Error(fmt.Sprintf("reject endpoint, failed removing transaction %v for address [ %s ]", trx.Hash, trx.ReceiverAddress))
			continue
		}
		hashes = append(hashes, trx.Hash)
	}

	return c.JSON(TransactionsRejectResponse{Success: true, TrxHashes: hashes})
}

// TransactionsRequest is a request to get awaited and issued to the DAG transactions.
// Request contains of Address for which Transactions are requested, Data in binary format,
// Hash of Data and Signature of the Data to prove that entity doing the request is an Address owner.
type TransactionsRequest struct {
	Address   string   `json:"address"`
	Data      []byte   `json:"data"`
	Signature []byte   `json:"signature"`
	Hash      [32]byte `json:"hash"`
}

// TransactionsResponse is a response for awaited or issued transactions request.
type TransactionsResponse struct {
	Transactions []transaction.Transaction `json:"awaited_transactions"`
	Success      bool                      `json:"success"`
}

func (s *server) awaited(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(awaitedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("awaited transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("awaited transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.verifier.Verify(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("awaited endpoint, failed to verify signature for address: %s, %s", req.Address, err))
		return fiber.ErrForbidden
	}

	trxs, err := s.readAwaitedTrx(req.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("awaited endpoint, failed to read awaited transactions for address: %s, %s", req.Address, err))
		return fiber.ErrBadRequest
	}

	return c.JSON(TransactionsResponse{
		Success:      true,
		Transactions: trxs,
	})
}

// TransactionsByHashRequest  is a request to get approved transactions that are part of the DAG.
type TransactionsByHashRequest struct {
	Address   string     `json:"address"`
	Hashes    [][32]byte `json:"hashes"`
	Signature []byte     `json:"signature"`
	Hash      [32]byte   `json:"hash"`
}

func (s *server) approved(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(approvedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsByHashRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	data := make([]byte, 0, len(req.Hashes)*32)
	for _, h := range req.Hashes {
		data = append(data, h[:]...)
	}

	if err := s.verifier.Verify(data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to verify signature for address: %s, %s", req.Address, err))
		return fiber.ErrForbidden
	}

	trxs, err := s.acc.ReadTransactionsByHashes(c.Context(), req.Hashes)
	if err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to read hashes for address: %s, %s", req.Address, err))
		return fiber.ErrForbidden
	}

	return c.JSON(TransactionsResponse{
		Success:      true,
		Transactions: trxs,
	})
}

// DataToSignRequest is a request to get data to sign for proving identity.
type DataToSignRequest struct {
	Address string `json:"address"`
}

// DataToSignRequest is a response containing data to sign for proving identity.
type DataToSignResponse struct {
	Data []byte `json:"message"`
}

func (s *server) data(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(dataToSignTelemetryHistogram, time.Since(t))

	var req DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("data endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if req.Address == "" {
		s.log.Error("wrong JSON format when requesting data to sign")
		return fiber.ErrBadRequest
	}

	d := s.randDataProv.ProvideData(req.Address)
	return c.JSON(DataToSignResponse{Data: d})
}

func (s *server) saveAwaitedTrx(ctx context.Context, trx *transaction.Transaction) error {
	if trx == nil {
		return nil
	}
	buf, err := trx.Encode()
	if err != nil {
		return err
	}
	err = s.trxsAwaitedDB.Update(func(txn *badger.Txn) error {
		if _, err := txn.Get(trx.Hash[:]); err == nil {
			return ErrTrxAlreadyExists
		}
		return txn.SetEntry(badger.NewEntry(trx.Hash[:], buf))
	})
	if err != nil {
		if errors.Is(err, ErrTrxAlreadyExists) {
			return nil
		}
		return err
	}

	hashHex := hexEncode(trx.Hash[:])

	for _, address := range []string{trx.IssuerAddress, trx.ReceiverAddress} {
		m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), add, time.Nanosecond)
		if err := m.Add(hashHex); err != nil {
			s.log.Error(fmt.Sprintf("saving address awaited failed for [ %s ], hex %v", address, hashHex))
		}
		m.Stop()
	}

	return nil
}

func (s *server) readAwaitedTrx(address string) ([]transaction.Transaction, error) {
	hashesHex := make([][]byte, 0)
	err := s.addressAwaitedTrxsDB.View(func(txn *badger.Txn) error {
		item, err := txn.Get([]byte(address))
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return nil
			}
			return err
		}

		if err := item.Value(func(val []byte) error {
			hashesHex = bytes.Split(val, []byte{','})
			return nil
		}); err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(hashesHex) == 0 {
		return nil, nil
	}

	cleanup := make([][]byte, 0)
	trxs := make([]transaction.Transaction, 0)
	err = s.trxsAwaitedDB.View(func(txn *badger.Txn) error {
		for _, h := range hashesHex {
			hash, err := hexDecode(h)
			if err != nil {
				s.log.Error(fmt.Sprintf("read awaited trxs failed to decode hex hash %v for address [ %s ]", h, address))
				continue
			}
			item, err := txn.Get(hash)
			if err != nil {
				if errors.Is(err, badger.ErrKeyNotFound) {
					cleanup = append(cleanup, h)
					continue
				}
				return err
			}
			if err := item.Value(func(val []byte) error {
				trx, err := transaction.Decode(val)
				if err != nil {
					return err
				}
				trxs = append(trxs, trx)
				return nil
			}); err != nil {
				return err
			}
			return nil
		}
		return nil
	})

	if len(cleanup) == 0 {
		return trxs, err
	}

	m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), remove, time.Nanosecond)
	defer m.Stop()
	for _, h := range cleanup {
		err := m.Add(h)
		if err != nil {
			s.log.Error(fmt.Sprintf("cleanup for hex hash %v failed, %s", h, err))
		}
	}

	return trxs, err
}

func (s *server) removeAwaitedTrx(trx *transaction.Transaction) error {
	if trx == nil {
		return nil
	}

	err := s.trxsAwaitedDB.Update(func(txn *badger.Txn) error {
		return txn.Delete(trx.Hash[:])
	})
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return nil
		}
	}

	hashHex := hexEncode(trx.Hash[:])
	for _, address := range []string{trx.IssuerAddress, trx.ReceiverAddress} {
		m := s.addressAwaitedTrxsDB.GetMergeOperator([]byte(address), remove, time.Nanosecond)
		if err := m.Add(hashHex); err != nil {
			s.log.Error(fmt.Sprintf("removing address of  awaited failed for [ %s ], hex %v", address, hashHex))
		}
		m.Stop()
	}

	return err
}
