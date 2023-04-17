package server

import (
	"github.com/bartossh/The-Accountant/transaction"
	"github.com/gofiber/fiber/v2"
)

func (s *server) alive(c *fiber.Ctx) error {
	return c.JSON(map[string]interface{}{"alive": true})
}

// SearchAddressRquest is a request to search for address.
type SearchAddressRquest struct {
	Address string `json:"address"`
}

// SearchAddressResponse is a response for address search.
type SearchAddressResponse struct {
	Addresses []string `json:"addresses"`
}

func (s *server) address(c *fiber.Ctx) error {
	var req SearchAddressRquest

	if err := c.BodyParser(&req); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}
	results, err := s.repo.FindAddress(c.Context(), req.Address, queryLimit)
	if err != nil {
		// TODO: log error
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

// searchBlockResponse is a response for block search.
type SearchBlockResponse struct {
	RawBlockHash [32]byte `json:"raw_block_hash"`
}

func (s *server) trxInBlock(c *fiber.Ctx) error {
	var req SearchBlockRequest

	if err := c.BodyParser(&req); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}

	res, err := s.repo.FindTransactionInBlockHash(c.Context(), req.RawTrxHash)
	if err != nil {
		// TODO: log error
		return fiber.ErrNotFound
	}

	return c.JSON(SearchBlockResponse{
		RawBlockHash: res,
	})
}

type TransactionConfirmResponse struct {
	Succes  bool     `json:"success"`
	TrxHash [32]byte `json:"trx_hash"`
}

func (s *server) confirm(c *fiber.Ctx) error {
	var trx transaction.Transaction
	if err := c.BodyParser(&trx); err != nil {
		// TODO: log err
		return fiber.ErrBadRequest
	}

	if err := s.bookkeeping.WriteCandidateTransaction(c.Context(), &trx); err != nil {
		// TODO: log err
		return c.JSON(TransactionConfirmResponse{
			Succes:  false,
			TrxHash: trx.Hash,
		})
	}

	return c.JSON(TransactionConfirmResponse{
		Succes:  true,
		TrxHash: trx.Hash,
	})
}
