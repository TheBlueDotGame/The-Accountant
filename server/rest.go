package server

import (
	"fmt"

	"github.com/bartossh/Computantis/transaction"
	"github.com/gofiber/fiber/v2"
)

// AliveResponse is a response for alive and version check.
type AliveResponse struct {
	Alive      bool   `json:"alive"`
	APIVersion string `json:"api_version"`
	APIHeader  string `json:"api_header"`
}

func (s *server) alive(c *fiber.Ctx) error {
	return c.JSON(
		AliveResponse{
			Alive:      true,
			APIVersion: ApiVersion,
			APIHeader:  Header,
		})
}

// SearchAddressRequest is a request to search for address.
type SearchAddressRequest struct {
	Address string `json:"address"`
}

// SearchAddressResponse is a response for address search.
type SearchAddressResponse struct {
	Addresses []string `json:"addresses"`
}

func (s *server) address(c *fiber.Ctx) error {
	var req SearchAddressRequest

	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("address endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	results, err := s.repo.FindAddress(c.Context(), req.Address, queryLimit)
	if err != nil {
		s.log.Error(fmt.Sprintf("address endpoint, failed to find address: %s", err.Error()))
		return fiber.ErrNotFound
	}

	return c.JSON(SearchAddressResponse{
		Addresses: results,
	})
}

// SearchBlockRequest is a request to search for block.
type SearchBlockRequest struct {
	RawTrxHash [32]byte `json:"raw_trx_hash"`
}

// SearchBlockResponse is a response for block search.
type SearchBlockResponse struct {
	RawBlockHash [32]byte `json:"raw_block_hash"`
}

func (s *server) trxInBlock(c *fiber.Ctx) error {
	var req SearchBlockRequest

	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("trx_in_block endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	res, err := s.repo.FindTransactionInBlockHash(c.Context(), req.RawTrxHash)
	if err != nil {
		s.log.Error(fmt.Sprintf("trx_in_block endpoint, failed to find transaction in block: %s", err.Error()))
		return fiber.ErrNotFound
	}

	return c.JSON(SearchBlockResponse{
		RawBlockHash: res,
	})
}

// TransactionProposeRequest is a request to propose a transaction.
type TransactionProposeRequest struct {
	ReceiverAddr string                  `json:"receiver_addr"`
	Transaction  transaction.Transaction `json:"transaction"`
}

// TransactionConfirmProposeResponse is a response for transaction propose.
type TransactionConfirmProposeResponse struct {
	Success bool     `json:"success"`
	TrxHash [32]byte `json:"trx_hash"`
}

func (s *server) propose(c *fiber.Ctx) error {
	var req TransactionProposeRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if len(req.Transaction.Data) == 0 || len(req.Transaction.Data) > s.dataSize {
		s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(req.Transaction.Data)))
		return fiber.ErrBadRequest
	}

	if err := s.bookkeeping.WriteIssuerSignedTransactionForReceiver(c.Context(), &req.Transaction); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, failed to write transaction: %s", err.Error()))
		return c.JSON(TransactionConfirmProposeResponse{
			Success: false,
			TrxHash: req.Transaction.Hash,
		})
	}

	return c.JSON(TransactionConfirmProposeResponse{
		Success: true,
		TrxHash: req.Transaction.Hash,
	})
}

func (s *server) confirm(c *fiber.Ctx) error {
	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if len(trx.Data) == 0 || len(trx.Data) > s.dataSize {
		s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(trx.Data)))
		return fiber.ErrBadRequest
	}

	if err := s.bookkeeping.WriteCandidateTransaction(c.Context(), &trx); err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, failed to write candidate transaction: %s", err.Error()))
		return c.JSON(TransactionConfirmProposeResponse{
			Success: false,
			TrxHash: trx.Hash,
		})
	}

	return c.JSON(TransactionConfirmProposeResponse{
		Success: true,
		TrxHash: trx.Hash,
	})
}

// AwaitedIssuedTransactionRequest is a request to get awaited or issued transactions for given address.
// Request contains of Address for which Transactions are requested, Data in binary format,
// Hash of Data and Signature of the Data to prove that entity doing the request is an Address owner.
type AwaitedIssuedTransactionRequest struct {
	Address   string   `json:"address"`
	Data      []byte   `json:"data"`
	Hash      [32]byte `json:"hash"`
	Signature []byte   `json:"signature"`
}

// AwaitedTransactionResponse is a response for awaited transactions request.
type AwaitedTransactionResponse struct {
	Success             bool                      `json:"success"`
	AwaitedTransactions []transaction.Transaction `json:"awaited_transactions"`
}

func (s *server) awaited(c *fiber.Ctx) error {
	var req AwaitedIssuedTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("awaited endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("awaited endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("awaited endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.repo.ReadAwaitingTransactionsByReceiver(c.Context(), req.Address)
	if err != nil {
		s.log.Error(
			fmt.Sprintf("awaited endpoint, failed to read awaiting transactions for address: %s, %s",
				req.Address, err.Error()))
		return c.JSON(AwaitedTransactionResponse{
			Success:             false,
			AwaitedTransactions: nil,
		})
	}

	return c.JSON(AwaitedTransactionResponse{
		Success:             true,
		AwaitedTransactions: trxs,
	})
}

// AwaitedTransactionResponse is a response for issued transactions request.
type IssuedTransactionResponse struct {
	Success            bool                      `json:"success"`
	IssuedTransactions []transaction.Transaction `json:"issued_transactions"`
}

func (s *server) issued(c *fiber.Ctx) error {
	var req AwaitedIssuedTransactionRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("issued endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("issued endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("issued endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.repo.ReadAwaitingTransactionsByIssuer(c.Context(), req.Address)
	if err != nil {
		s.log.Error(fmt.Sprintf("issued endpoint, failed to read issued transactions for address: %s, %s",
			req.Address, err.Error()))
		return c.JSON(IssuedTransactionResponse{
			Success:            false,
			IssuedTransactions: nil,
		})
	}

	return c.JSON(IssuedTransactionResponse{
		Success:            true,
		IssuedTransactions: trxs,
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
	var req DataToSignRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("data endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	d := s.randDataProv.ProvideData(req.Address)
	return c.JSON(DataToSignResponse{Data: d})
}

// CreateAddressRequest is a request to create an address.
type CreateAddressRequest struct {
	Address   string   `json:"address"`
	Token     string   `json:"token"`
	Data      []byte   `json:"data"`
	Hash      [32]byte `json:"hash"`
	Signature []byte   `json:"signature"`
}

// Response for address creation request.
// If Success is true, Address contains created address in base58 format.
type CreateAddressResponse struct {
	Success bool   `json:"success"`
	Address string `json:"address"`
}

func (s *server) addressCreate(c *fiber.Ctx) error {
	var req CreateAddressRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.repo.CheckToken(c.Context(), req.Token); !ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("address create endpoint, address: %s, failed to check token: %s", req.Address, err.Error()))
			return fiber.ErrGone
		}
		s.log.Error(fmt.Sprintf("address create endpoint, token: %s not found in the repository", req.Token))
		return fiber.ErrForbidden
	}

	if err := s.repo.InvalidateToken(c.Context(), req.Token); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to invalidate token: %s, %s", req.Token, err.Error()))
		return fiber.ErrGone
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to verify signature for address: %s, %s",
			req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	if ok, err := s.repo.CheckAddressExists(c.Context(), req.Address); ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("address create endpoint, failed to check address: %s,%s", req.Address, err.Error()))
			return fiber.ErrGone
		}
		s.log.Error(fmt.Sprintf("address create endpoint, address already exists: %s, %s", req.Address, err.Error()))
		return fiber.ErrConflict
	}

	if err := s.repo.WriteAddress(c.Context(), req.Address); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to write address: %s, %s", req.Address, err.Error()))
		return fiber.ErrConflict
	}

	return c.JSON(&CreateAddressResponse{Success: true, Address: req.Address})
}
