package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/Computantis/block"
	"github.com/bartossh/Computantis/server"
	"github.com/bartossh/Computantis/token"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/validator"
	"github.com/bartossh/Computantis/wallet"
	"github.com/valyala/fasthttp"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

var (
	ErrApiVersionMismatch            = fmt.Errorf("api version mismatch")
	ErrApiHeaderMismatch             = fmt.Errorf("api header mismatch")
	ErrStatusCodeMismatch            = fmt.Errorf("status code mismatch")
	ErrContentTypeMismatch           = fmt.Errorf("content type mismatch")
	ErrWalletChecksumMismatch        = fmt.Errorf("wallet checksum mismatch")
	ErrWalletVersionMismatch         = fmt.Errorf("wallet version mismatch")
	ErrServerReturnsInconsistentData = fmt.Errorf("server returns inconsistent data")
	ErrRejectedByServer              = fmt.Errorf("rejected by server")
	ErrWalletNotReady                = fmt.Errorf("wallet not ready, read wallet first")
	ErrSigningFailed                 = fmt.Errorf("signing failed")
)

// WalletReadSaver allows to read and save the wallet.
type WalletReadSaver interface {
	ReadWallet() (wallet.Wallet, error)
	SaveWallet(w wallet.Wallet) error
}

// NewWalletCreator is a function that creates a new SignValidator.
type NewSignValidatorCreator func() (wallet.Wallet, error)

// Client is a rest client for the API.
// It provides methods to communicate with the API server
// and is designed to serve as a easy way of building client applications
// that uses the REST API of the central node.
type Client struct {
	apiRoot       string
	timeout       time.Duration
	verifier      transaction.Verifier
	wrs           WalletReadSaver
	w             wallet.Wallet
	walletCreator NewSignValidatorCreator
	ready         bool
}

// NewClient creates a new rest client.
func NewClient(
	apiRoot string, timeout time.Duration, fw transaction.Verifier,
	wrs WalletReadSaver, walletCreator NewSignValidatorCreator,
) *Client {
	return &Client{apiRoot: apiRoot, timeout: timeout, verifier: fw, wrs: wrs, walletCreator: walletCreator}
}

// ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness.
// If API version not much it is returning an error as accessing the API server with different API version
// may lead to unexpected results.
func (c *Client) ValidateApiVersion() error {
	var alive server.AliveResponse
	if err := c.makeGet("alive", &alive); err != nil {
		return err
	}

	if alive.APIVersion != server.ApiVersion {
		return errors.Join(ErrApiVersionMismatch, fmt.Errorf("expected %s but got %s", server.ApiVersion, alive.APIVersion))
	}

	if alive.APIHeader != server.Header {
		return errors.Join(ErrApiHeaderMismatch, fmt.Errorf("expected %s but got %s", server.Header, alive.APIHeader))
	}

	return nil
}

// NewWallet creates a new wallet and sends a request to the API server to validate the wallet.
func (c *Client) NewWallet(token string) error {
	w, err := c.walletCreator()
	if err != nil {
		return err
	}
	if w.ChecksumLength() != checksumLength {
		return errors.Join(
			ErrWalletChecksumMismatch,
			fmt.Errorf("checksum length mismatch, expected %d but got %d", checksumLength, w.ChecksumLength()))
	}
	if w.Version() != version {
		return errors.Join(
			ErrWalletVersionMismatch,
			fmt.Errorf("version mismatch, expected %d but got %d", version, w.Version()))
	}

	dataToSign, err := c.DataToSign(w.Address())
	if err != nil {
		return errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := w.Sign(dataToSign.Data)

	reqCreateAddr := server.CreateAddressRequest{
		Address:   w.Address(),
		Token:     token,
		Data:      dataToSign.Data,
		Hash:      hash,
		Signature: signature,
	}
	var resCreateAddr server.CreateAddressResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.CreateAddressURL)
	if err := c.makePost(url, reqCreateAddr, &resCreateAddr); err != nil {
		return err
	}

	if !resCreateAddr.Success {
		return errors.Join(ErrRejectedByServer, errors.New("failed to create address"))
	}

	if resCreateAddr.Address != w.Address() {
		return errors.Join(ErrServerReturnsInconsistentData, errors.New("failed to create address"))
	}

	c.w = w
	c.ready = true

	return nil
}

// Address reads the wallet address.
// Address is a string representation of wallet public key.
func (c *Client) Address() (string, error) {
	if !c.ready {
		return "", ErrWalletNotReady
	}

	return c.w.Address(), nil
}

// ProposeTransaction sends a Transaction proposal to the API server for provided receiver address.
// Subject describes how to read the data from the transaction. For example, if the subject is "json",
// then the data can by decoded to map[sting]any, when subject "pdf" than it should be decoded by proper pdf decoder,
// when "csv" then it should be decoded by proper csv decoder.
// Client is not responsible for decoding the data, it is only responsible for sending the data to the API server.
func (c *Client) ProposeTransaction(receiverAddr string, subject string, data []byte) error {
	if !c.ready {
		return ErrWalletNotReady
	}

	trx, err := transaction.New(subject, data, receiverAddr, &c.w)
	if err != nil {
		return errors.Join(ErrSigningFailed, err)
	}

	req := server.TransactionProposeRequest{
		ReceiverAddr: receiverAddr,
		Transaction:  trx,
	}
	var res server.TransactionConfirmProposeResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.ProposeTransactionURL)
	if err := c.makePost(url, req, &res); err != nil {
		return errors.Join(ErrRejectedByServer, err)
	}

	if !res.Success {
		return errors.Join(ErrRejectedByServer, errors.New("failed to propose transaction"))
	}

	if !bytes.Equal(trx.Hash[:], res.TrxHash[:]) {
		return errors.Join(ErrServerReturnsInconsistentData, errors.New("failed to propose transaction"))
	}

	return nil
}

// ConfirmTransaction confirms transaction by signing it with the wallet
// and then sending it to the API server.
func (c *Client) ConfirmTransaction(trx *transaction.Transaction) error {
	if !c.ready {
		return ErrWalletNotReady
	}

	if _, err := trx.Sign(&c.w, c.verifier); err != nil {
		return errors.Join(ErrSigningFailed, err)
	}

	var res server.TransactionConfirmProposeResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.ConfirmTransactionURL)
	if err := c.makePost(url, trx, &res); err != nil {
		return errors.Join(ErrRejectedByServer, err)
	}

	if !res.Success {
		return errors.Join(ErrRejectedByServer, errors.New("failed to confirm transaction"))
	}

	if !bytes.Equal(trx.Hash[:], res.TrxHash[:]) {
		return errors.Join(ErrServerReturnsInconsistentData, errors.New("failed to confirm transaction"))
	}

	return nil
}

// ReadWaitingTransactions reads all waiting transactions belonging to current wallet from the API server.
func (c *Client) ReadWaitingTransactions() ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, ErrWalletNotReady
	}

	data, err := c.DataToSign(c.w.Address())
	if err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := server.AwaitedIssuedTransactionRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	var res server.AwaitedTransactionResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.AwaitedTransactionURL)
	if err := c.makePost(url, req, &res); err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(ErrRejectedByServer, errors.New("failed to read waiting transactions"))
	}

	return res.AwaitedTransactions, nil

}

// ReadIssuedTransactions reads all issued transactions belonging to current wallet from the API server.
func (c *Client) ReadIssuedTransactions() ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, ErrWalletNotReady
	}

	data, err := c.DataToSign(c.w.Address())
	if err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := server.AwaitedIssuedTransactionRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	var res server.IssuedTransactionResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.IssuedTransactionURL)
	if err := c.makePost(url, req, &res); err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(ErrRejectedByServer, errors.New("failed to read issued transactions"))
	}

	return res.IssuedTransactions, nil
}

// GenerateToken generates a token for the given time in the central node repository.
// It is only permitted to generate a token if wallet has admin permissions in the central node.
func (c *Client) GenerateToken(t time.Time) (token.Token, error) {
	if !c.ready {
		return token.Token{}, ErrWalletNotReady
	}

	data, err := c.DataToSign(c.w.Address())
	if err != nil {
		return token.Token{}, errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := server.GenerateTokenRequest{
		Address:    c.w.Address(),
		Data:       data.Data,
		Hash:       hash,
		Signature:  signature,
		Expiration: t.UnixMicro(),
	}

	var res server.GenerateTokenResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.GenerateTokenURL)
	if err := c.makePost(url, req, &res); err != nil {
		return token.Token{}, errors.Join(ErrRejectedByServer, err)
	}
	if !res.Valid {
		return token.Token{}, errors.Join(ErrRejectedByServer, errors.New("failed to generate token"))
	}

	return res, nil
}

// SaveWalletToFile saves the wallet to the file in the path.
func (c *Client) SaveWalletToFile() error {
	if !c.ready {
		return ErrWalletNotReady
	}

	return c.wrs.SaveWallet(c.w)
}

// ReadWalletFromFile reads the wallet from the file in the path.
func (c *Client) ReadWalletFromFile() error {
	w, err := c.wrs.ReadWallet()
	if err != nil {
		return err
	}
	c.w = w
	c.ready = true
	return nil
}

// DataToSign returns data to sign for the given address.
// Data to sign are randomly generated bytes by the server and stored in pair with the address.
// Signing this data is a proof that the signing public address is the owner of the wallet a making request.
func (c *Client) DataToSign(address string) (server.DataToSignResponse, error) {
	req := server.DataToSignRequest{
		Address: address,
	}
	var resp server.DataToSignResponse
	url := fmt.Sprintf("%s/%s", c.apiRoot, server.DataToValidateURL)
	if err := c.makePost(url, req, &resp); err != nil {
		return server.DataToSignResponse{}, err
	}
	return resp, nil
}

// Sign signs the given data with the wallet and returns digest and signature or error otherwise.
// This process creates a proof for the API server that requesting client is the owner of the wallet.
func (c *Client) Sign(d []byte) (digest [32]byte, signature []byte, err error) {
	if !c.ready {
		return digest, signature, ErrWalletNotReady
	}
	digest, signature = c.w.Sign(d)
	return
}

// PostWebhookBlock posts validator.WebHookNewBlockMessage to given url.
func (c *Client) PostWebhookBlock(url, token string, block *block.Block) error {
	req := validator.WebHookNewBlockMessage{
		Token: token,
		Block: *block,
	}
	res := make(map[string]any)
	return c.makePost(url, req, &res)
}

// PostWebhookNewTransaction posts validator.WebHookNewTransaction to given url.
func (c *Client) PostWebhookNewTransaction(url, token string) error {
	req := validator.NewTransactionMessage{
		State: validator.StateIssued,
		Time:  time.Now(),
		Token: token,
	}

	res := make(map[string]any)
	return c.makePost(url, req, &res)
}

// FlushWalletFromMemory flushes the wallet from the memory.
// Do it after you have saved the wallet to the file.
// It is recommended to use this just before logging out from the UI
// or closing the front end app that.
func (c *Client) FlushWalletFromMemory() {
	c.w = wallet.Wallet{}
	c.ready = false
}

func (c *Client) makePost(path string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(path)
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, c.timeout); err != nil {
		return err
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK, fasthttp.StatusCreated, fasthttp.StatusAccepted:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return errors.Join(
			ErrStatusCodeMismatch,
			fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode()))
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return errors.Join(
			ErrContentTypeMismatch,
			fmt.Errorf("expected content type application/json but got %s", contentType))
	}

	return json.Unmarshal(resp.Body(), in)
}

func (c *Client) makeGet(path string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", c.apiRoot, path))
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, c.timeout); err != nil {
		return err
	}

	switch resp.StatusCode() {
	case fasthttp.StatusOK:
	case fasthttp.StatusNoContent:
		return nil
	default:
		return errors.Join(
			ErrStatusCodeMismatch,
			fmt.Errorf("expected status code %d but got %d", fasthttp.StatusOK, resp.StatusCode()))
	}

	contentType := resp.Header.Peek("Content-Type")
	if bytes.Index(contentType, []byte("application/json")) != 0 {
		return errors.Join(
			ErrContentTypeMismatch,
			fmt.Errorf("expected content type application/json but got %s", contentType))
	}

	return json.Unmarshal(resp.Body(), out)
}
