package notaryserver

import (
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"

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

	if trx.Subject == "" || trx.Data == nil || trx.IssuerAddress == "" ||
		trx.ReceiverAddress == "" || trx.Hash == [32]byte{} ||
		trx.CreatedAt.IsZero() || trx.IssuerSignature == nil {
		s.log.Error("wrong JSON format for propose trx")
		return fiber.ErrBadRequest
	}

	if len(trx.Data) == 0 || len(trx.Data) > s.dataSize {
		s.log.Error(fmt.Sprintf("propose endpoint, invalid transaction data size: %d", len(trx.Data)))
		return fiber.ErrBadRequest
	}

	// TODO: validate if spice transfer or contract and use accountant or store in cache

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
		s.log.Error(fmt.Sprintf("confirm endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if trx.Subject == "" || trx.Data == nil || trx.IssuerAddress == "" || trx.ReceiverAddress == "" ||
		trx.Hash == [32]byte{} || trx.CreatedAt.IsZero() || trx.IssuerSignature == nil ||
		trx.ReceiverSignature == nil {
		s.log.Error("wrong address JSON format to confirm trx")
		return fiber.ErrBadRequest
	}

	// TODO: send to accountant

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

	if req.Address == "" || req.Transactions == nil || req.Data == nil || req.Signature == nil || req.Hash == [32]byte{} {
		s.log.Error("wrong JSON format when rejecting transactions")
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("issued endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	// TODO: validate request and send remove from temporary do per trx in loop

	trxsReject := make([]transaction.Transaction, 0, len(req.Transactions))
	for _, trx := range req.Transactions {
		if trx.ReceiverAddress == req.Address {
			trxsReject = append(trxsReject, trx)
		}
	}

	hashes := make([][32]byte, 0, len(trxsReject))
	for _, trx := range trxsReject {
		hashes = append(hashes, trx.Hash)
	}

	return c.JSON(TransactionsRejectResponse{Success: true, TrxHashes: hashes})
}

// TransactionsRequest is a request to get awaited, issued or added to the DAG transactions.
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

	if req.Address == "" || req.Hash == [32]byte{} || req.Signature == nil || req.Data == nil {
		s.log.Error("wrong JSON format when reading awaited transactions")
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("awaited transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	// TODO: validate signature then look in awaited trx

	return c.JSON(TransactionsResponse{
		Success: true,
	})
}

func (s *server) approved(c *fiber.Ctx) error {
	t := time.Now()
	defer s.tele.RecordHistogramTime(approvedTrxTelemetryHistogram, time.Since(t))

	var req TransactionsRequest
	if err := c.BodyParser(&req); err != nil {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to parse request body: %s", err.Error()))
		return fiber.ErrBadRequest
	}

	if req.Address == "" || req.Hash == [32]byte{} || req.Signature == nil || req.Data == nil {
		s.log.Error("wrong JSON format when reading approved transactions")
		return fiber.ErrBadRequest
	}

	if ok := s.randDataProv.ValidateData(req.Address, req.Data); !ok {
		s.log.Error(fmt.Sprintf("approved transactions endpoint, failed to validate data for address: %s", req.Address))
		return fiber.ErrForbidden
	}

	// TODO: look in to accountant DAG for the transactions.

	return c.JSON(TransactionsResponse{
		Success: true,
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
