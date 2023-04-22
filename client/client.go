package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/bartossh/The-Accountant/server"
	"github.com/bartossh/The-Accountant/transaction"
	"github.com/bartossh/The-Accountant/wallet"
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
	ReadWallet(path string) (wallet.Wallet, error)
	SaveWallet(path string, w wallet.Wallet) error
}

// NewWalletCreator is a function that creates a new SignValidator.
type NewSignValidatorCreator func() (wallet.Wallet, error)

// Rest is a rest client for the API.
type Rest struct {
	apiRoot       string
	timeout       time.Duration
	verifier      transaction.Verifier
	wrs           WalletReadSaver
	w             wallet.Wallet
	walletCreator NewSignValidatorCreator
	ready         bool
}

// NewRest creates a new rest client.
func NewRest(
	apiRoot string, timeout time.Duration, fw transaction.Verifier,
	wrs WalletReadSaver, walletCreator NewSignValidatorCreator,
) *Rest {
	return &Rest{apiRoot: apiRoot, timeout: timeout, verifier: fw, wrs: wrs, walletCreator: walletCreator}
}

// ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness.
func (r *Rest) ValidateApiVersion() error {
	var alive server.AliveResponse
	if err := r.makeGet("alive", &alive); err != nil {
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
func (r *Rest) NewWallet(token string) error {
	w, err := r.walletCreator()
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

	dataToSign, err := r.dataToSign(w.Address())
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
	if err := r.makePost(server.CreateAddressURL, reqCreateAddr, &resCreateAddr); err != nil {
		return err
	}

	if !resCreateAddr.Success {
		return errors.Join(ErrRejectedByServer, errors.New("failed to create address"))
	}

	if resCreateAddr.Address != w.Address() {
		return errors.Join(ErrServerReturnsInconsistentData, errors.New("failed to create address"))
	}

	r.w = w
	r.ready = true

	return nil
}

// Address reads the wallet address.
// Address is a string representation of wallet public key.
func (r *Rest) Address() (string, error) {
	if !r.ready {
		return "", ErrWalletNotReady
	}

	return r.w.Address(), nil
}

// ProposeTransaction sends a Transaction proposal to the API server for provided receiver address.
// Subject describes how to read the data from the transaction. For example, if the subject is "json",
// then the data can by decoded to map[sting]any, when subject "pdf" than it should be decoded by proper pdf decoder,
// when "csv" then it should be decoded by proper csv decoder.
func (r *Rest) ProposeTransaction(receiverAddr string, subject string, data []byte) error {
	if !r.ready {
		return ErrWalletNotReady
	}

	trx, err := transaction.New(subject, data, &r.w)
	if err != nil {
		return errors.Join(ErrSigningFailed, err)
	}

	req := server.TransactionProposeRequest{
		ReceiverAddr: receiverAddr,
		Transaction:  trx,
	}
	var res server.TransactionConfirmProposeResponse
	if err := r.makePost(server.ProposeTransactionURL, req, &res); err != nil {
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
func (r *Rest) ConfirmTransaction(trx transaction.Transaction) error {
	if !r.ready {
		return ErrWalletNotReady
	}

	if _, err := trx.Sign(&r.w, r.verifier); err != nil {
		return errors.Join(ErrSigningFailed, err)
	}

	var res server.TransactionConfirmProposeResponse
	if err := r.makePost(server.ConfirmTransactionURL, trx, &res); err != nil {
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
func (r *Rest) ReadWaitingTransactions() ([]transaction.Transaction, error) {
	if !r.ready {
		return nil, ErrWalletNotReady
	}

	data, err := r.dataToSign(r.w.Address())
	if err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := r.w.Sign(data.Data)
	req := server.AwaitedIssuedTransactionRequest{
		Address:   r.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	var res server.AwaitedTransactionResponse
	if err := r.makePost(server.AwaitedTransactionURL, req, &res); err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(ErrRejectedByServer, errors.New("failed to read waiting transactions"))
	}

	return res.AwaitedTransactions, nil

}

// ReadIssuedTransactions reads all issued transactions belonging to current wallet from the API server.
func (r *Rest) ReadIssuedTransactions() ([]transaction.Transaction, error) {
	if !r.ready {
		return nil, ErrWalletNotReady
	}

	data, err := r.dataToSign(r.w.Address())
	if err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}

	hash, signature := r.w.Sign(data.Data)
	req := server.AwaitedIssuedTransactionRequest{
		Address:   r.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	var res server.IssuedTransactionResponse
	if err := r.makePost(server.IssuedTransactionURL, req, &res); err != nil {
		return nil, errors.Join(ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(ErrRejectedByServer, errors.New("failed to read issued transactions"))
	}

	return res.IssuedTransactions, nil
}

// SaveWalletToFile saves the wallet to the file in the path.
func (r *Rest) SaveWalletToFile(path string) error {
	if !r.ready {
		return ErrWalletNotReady
	}

	return r.wrs.SaveWallet(path, r.w)
}

// ReadWalletFromFile reads the wallet from the file in the path.
func (r *Rest) ReadWalletFromFile(path string) error {
	w, err := r.wrs.ReadWallet(path)
	if err != nil {
		return err
	}
	r.w = w
	r.ready = true
	return nil
}

// FlushWalletFromMemory flushes the wallet from the memory.
// Do it after you have saved the wallet to the file.
// It is recommended to use this just before logging out from the UI
// or closing the front end app that.
func (r *Rest) FlushWalletFromMemory() {
	r.w = wallet.Wallet{}
	r.ready = false
}

func (r *Rest) makePost(path string, out, in any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", r.apiRoot, path))
	req.Header.SetMethod("POST")
	req.Header.SetContentType("application/json")
	raw, err := json.Marshal(out)
	if err != nil {
		return err
	}
	req.SetBody(raw)

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, r.timeout); err != nil {
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

func (r *Rest) makeGet(path string, out any) error {
	req := fasthttp.AcquireRequest()
	defer fasthttp.ReleaseRequest(req)

	req.SetRequestURI(fmt.Sprintf("%s/%s", r.apiRoot, path))
	req.Header.SetMethod("GET")

	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := fasthttp.DoTimeout(req, resp, r.timeout); err != nil {
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

func (r *Rest) dataToSign(address string) (server.DataToSignResponse, error) {
	req := server.DataToSignRequest{
		Address: address,
	}
	var resp server.DataToSignResponse
	if err := r.makePost(server.DataToValidateURL, req, &resp); err != nil {
		return server.DataToSignResponse{}, err
	}
	return resp, nil
}
