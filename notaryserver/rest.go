package notaryserver

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/bartossh/Computantis/token"
	"github.com/bartossh/Computantis/transaction"
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
	t := time.Now()
	defer s.tele.RecordHistogramTime(addressURLTelemetryHistogram, time.Since(t))

	var req SearchAddressRequest

	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("address endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	results, err := s.addressProv.FindAddress(c.Context(), req.Address, queryLimit)
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
	Address    string   `json:"address"`
	RawTrxHash [32]byte `json:"raw_trx_hash"`
}

// SearchBlockResponse is a response for block search.
type SearchBlockResponse struct {
	RawBlockHash [32]byte `json:"raw_block_hash"`
}

func (s *server) trxInBlock(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(trxInBlockTelemetryHistogram, time.Since(t))

	var req SearchBlockRequest

	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("trx_in_block endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	res, err := s.trxProv.FindTransactionInBlockHash(c.Context(), req.RawTrxHash)
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
	TrxHash [32]byte `json:"trx_hash"`
	Success bool     `json:"success"`
}

func (s *server) propose(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(proposeTrxTelemetryHistogram, time.Since(t))

	var req TransactionProposeRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("propose endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if len(req.Transaction.Data) == 0 || len(req.Transaction.Data) > s.dataSize {
		s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(req.Transaction.Data)))
		return fiber.ErrBadRequest
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Transaction.IssuerAddress); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("issuer address %s is suspended", req.Transaction.IssuerAddress))
		return fiber.ErrForbidden
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
	t := time.Now()
	defer s.tele.RecordHistogramTime(confirmTrxTelemetryHistogram, time.Since(t))

	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		s.log.Error(fmt.Sprintf("confirm endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), trx.IssuerAddress); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("issuer address %s is suspended", trx.IssuerAddress))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), trx.ReceiverAddress); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("receiver address %s is suspended", trx.ReceiverAddress))
		return fiber.ErrForbidden
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
		s.log.Error(fmt.Sprintf("issued endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("issued endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxsReject := make([]transaction.Transaction, 0, len(req.Transactions))
	for _, trx := range req.Transactions {
		if trx.ReceiverAddress == req.Address {
			trxsReject = append(trxsReject, trx)
		}
	}

	if err := s.trxProv.RejectTransactions(c.Context(), req.Address, trxsReject); err != nil {
		return c.JSON(TransactionsRejectResponse{Success: false, TrxHashes: nil})
	}

	hashes := make([][32]byte, 0, len(trxsReject))
	for _, trx := range trxsReject {
		hashes = append(hashes, trx.Hash)
	}

	return c.JSON(TransactionsRejectResponse{Success: true, TrxHashes: hashes})
}

// TransactionsRequest is a request to get awaited, issued or rejected transactions for given address.
// Request contains of Address for which Transactions are requested, Data in binary format,
// Hash of Data and Signature of the Data to prove that entity doing the request is an Address owner.
type TransactionsRequest struct {
	Address   string   `json:"address"`
	Data      []byte   `json:"data"`
	Signature []byte   `json:"signature"`
	Hash      [32]byte `json:"hash"`
	Offset    int      `json:"offset,omitempty"`
	Limit     int      `json:"limit,omitempty"`
}

// AwaitedTransactionsResponse is a response for awaited transactions request.
type AwaitedTransactionsResponse struct {
	AwaitedTransactions []transaction.Transaction `json:"awaited_transactions"`
	Success             bool                      `json:"success"`
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

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("awaited transactions endpoint, failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("awaited transactions endpoint, address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("awaited transactions endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.trxProv.ReadAwaitingTransactionsByReceiver(c.Context(), req.Address)
	if err != nil {
		s.log.Error(
			fmt.Sprintf("awaited transactions endpoint, failed to read awaiting transactions for address: %s, %s",
				req.Address, err.Error()))
		return c.JSON(AwaitedTransactionsResponse{
			Success:             false,
			AwaitedTransactions: nil,
		})
	}

	return c.JSON(AwaitedTransactionsResponse{
		Success:             true,
		AwaitedTransactions: trxs,
	})
}

// IssuedTransactionsResponse is a response for issued transactions request.
type IssuedTransactionsResponse struct {
	IssuedTransactions []transaction.Transaction `json:"issued_transactions"`
	Success            bool                      `json:"success"`
}

func (s *server) issued(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(issuedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("issued transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("issued transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("issued transactions endpoint, failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("issued transactions endpoint, address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("issued transactions endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.trxProv.ReadRejectedTransactionsPagginate(c.Context(), req.Address, req.Offset, req.Limit)
	if err != nil {
		s.log.Error(fmt.Sprintf("issued transactions endpoint, failed to read issued transactions for address: %s, %s",
			req.Address, err.Error()))
		return c.JSON(IssuedTransactionsResponse{
			Success:            false,
			IssuedTransactions: nil,
		})
	}

	return c.JSON(IssuedTransactionsResponse{
		Success:            true,
		IssuedTransactions: trxs,
	})
}

// RejectedTransactionsResponse is a response for rejected transactions request.
type RejectedTransactionsResponse struct {
	RejectedTransactions []transaction.Transaction `json:"rejected_transactions"`
	Success              bool                      `json:"success"`
}

func (s *server) rejected(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(rejectedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("rejected transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("rejected transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("rejected transactions endpoint, failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("rejected transactions endpoint, address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("rejected transactions endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.trxProv.ReadRejectedTransactionsPagginate(c.Context(), req.Address, req.Offset, req.Limit)
	if err != nil {
		s.log.Error(fmt.Sprintf("rejected transactions endpoint, failed to read issued transactions for address: %s, %s",
			req.Address, err.Error()))
		return c.JSON(RejectedTransactionsResponse{
			Success:              false,
			RejectedTransactions: nil,
		})
	}

	return c.JSON(RejectedTransactionsResponse{
		Success:              true,
		RejectedTransactions: trxs,
	})
}

// ApprovedTransactionsResponse is a response for approved transactions request.
type ApprovedTransactionsResponse struct {
	ApprovedTransactions []transaction.Transaction `json:"approved_transactions"`
	Success              bool                      `json:"success"`
}

func (s *server) approved(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(approvedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressSuspended(c.Context(), req.Address); err != nil || ok {
		if err != nil {
			s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to check address: %s", err.Error()))
			return fiber.ErrForbidden
		}
		s.log.Error(fmt.Sprintf("approved transactions endpoint, address %s is suspended", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(
			fmt.Sprintf("approved transactions endpoint, failed to verify signature for address: %s, %s", req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	trxs, err := s.trxProv.ReadApprovedTransactions(c.Context(), req.Address, req.Offset, req.Limit)
	if err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to read issued transactions for address: %s, %s",
			req.Address, err.Error()))
		return c.JSON(ApprovedTransactionsResponse{
			Success:              false,
			ApprovedTransactions: nil,
		})
	}

	return c.JSON(ApprovedTransactionsResponse{
		Success:              true,
		ApprovedTransactions: trxs,
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

	d := s.randDataProv.ProvideData(req.Address)
	return c.JSON(DataToSignResponse{Data: d})
}

// CreateAddressRequest is a request to create an address.
type CreateAddressRequest struct {
	Address   string   `json:"address"`
	Token     string   `json:"token"`
	Data      []byte   `json:"data"`
	Signature []byte   `json:"signature"`
	Hash      [32]byte `json:"hash"`
}

// Response for address creation request.
// If Success is true, Address contains created address in base58 format.
type CreateAddressResponse struct {
	Address string `json:"address"`
	Success bool   `json:"success"`
}

func (s *server) addressCreate(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(addressCreateTelemetryHistogram, time.Since(t))

	var req CreateAddressRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if ok, err := s.tokenProv.CheckToken(c.Context(), req.Token); !ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("address create endpoint, address: %s, failed to check token: %s", req.Address, err.Error()))
			return fiber.ErrInternalServerError
		}
		s.log.Error(fmt.Sprintf("address create endpoint, token: %s not found in the repository", req.Token))
		return fiber.ErrForbidden
	}

	if err := s.tokenProv.InvalidateToken(c.Context(), req.Token); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to invalidate token: %s, %s", req.Token, err.Error()))
		return fiber.ErrInternalServerError
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to verify signature for address: %s, %s",
			req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.CheckAddressExists(c.Context(), req.Address); ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("address create endpoint, failed to check if address exists: %s,%s", req.Address, err.Error()))
			return fiber.ErrInternalServerError
		}
		s.log.Error(fmt.Sprintf("address create endpoint, address already exists: %s", req.Address))
		return fiber.ErrConflict
	}

	if err := s.addressProv.WriteAddress(c.Context(), req.Address); err != nil {
		s.log.Error(fmt.Sprintf("address create endpoint, failed to write address: %s, %s", req.Address, err.Error()))
		return fiber.ErrConflict
	}

	return c.JSON(&CreateAddressResponse{Success: true, Address: req.Address})
}

// GenerateTokenRequest is a request for token generation.
type GenerateTokenRequest struct {
	Address    string   `json:"address"`
	Data       []byte   `json:"data"`
	Signature  []byte   `json:"signature"`
	Hash       [32]byte `json:"hash"`
	Expiration int64    `json:"expiration"`
}

// GenerateTokenResponse is a response containing generated token.
type GenerateTokenResponse = token.Token

func (s *server) tokenGenerate(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(tokenGenerateTelemetryHistogram, time.Since(t))

	var req GenerateTokenRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("token generate, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}
	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("token generate, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	if err := s.bookkeeping.VerifySignature(req.Data, req.Signature, req.Hash, req.Address); err != nil {
		s.log.Error(fmt.Sprintf("token generate, failed to verify signature for address: %s, %s",
			req.Address, err.Error()))
		return fiber.ErrForbidden
	}

	if ok, err := s.addressProv.IsAddressAdmin(c.Context(), req.Address); !ok || err != nil {
		if err != nil {
			s.log.Error(fmt.Sprintf("token generate, failed to check address: %s,%s", req.Address, err.Error()))
			return fiber.ErrGone
		}
		s.log.Error(fmt.Sprintf("token generate, address is not admin: %s", req.Address))
		return fiber.ErrForbidden
	}

	token, err := token.New(req.Expiration)
	if err != nil {
		s.log.Error(fmt.Sprintf("token generate, failed to create token: %s", err.Error()))
		return fiber.ErrInternalServerError
	}

	if err := s.tokenProv.WriteToken(c.Context(), req.Address, req.Expiration); err != nil {
		s.log.Error(fmt.Sprintf("token generate, failed to write token: %s, %s", req.Address, err.Error()))
		return fiber.ErrConflict
	}

	return c.JSON(token)
}