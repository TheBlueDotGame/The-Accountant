package walletmiddleware

import (
	"bytes"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/bartossh/Computantis/helperserver"
	"github.com/bartossh/Computantis/httpclient"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/spice"
	"github.com/bartossh/Computantis/token"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/wallet"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

// WalletReadSaver allows to read and save the wallet.
type WalletReadSaver interface {
	ReadWallet() (wallet.Wallet, error)
	SaveWallet(w *wallet.Wallet) error
}

// NewWalletCreator is a function that creates a new SignValidator.
type NewSignValidatorCreator func() (wallet.Wallet, error)

// Client is a rest client for the API.
// It provides methods to communicate with the API server
// and is designed to serve as a easy way of building client applications
// that uses the REST API of the central node.
type Client struct {
	verifier      transaction.Verifier
	wrs           WalletReadSaver
	walletCreator NewSignValidatorCreator
	apiRoot       string
	w             wallet.Wallet
	timeout       time.Duration
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
	var alive notaryserver.AliveResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.AliveURL)
	if err := httpclient.MakeGet(c.timeout, url, &alive); err != nil {
		return fmt.Errorf("check notary node alive on url: [ %s ], %w", url, err)
	}

	if alive.APIVersion != notaryserver.ApiVersion {
		return errors.Join(httpclient.ErrApiVersionMismatch, fmt.Errorf("expected %s but got %s", notaryserver.ApiVersion, alive.APIVersion))
	}

	if alive.APIHeader != notaryserver.Header {
		return errors.Join(httpclient.ErrApiHeaderMismatch, fmt.Errorf("expected %s but got %s", notaryserver.Header, alive.APIHeader))
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
			httpclient.ErrWalletChecksumMismatch,
			fmt.Errorf("checksum length mismatch, expected %d but got %d", checksumLength, w.ChecksumLength()))
	}
	if w.Version() != version {
		return errors.Join(
			httpclient.ErrWalletVersionMismatch,
			fmt.Errorf("version mismatch, expected %d but got %d", version, w.Version()))
	}

	c.w = w
	c.ready = true

	dataToSign, err := c.DataToSign(c.apiRoot)
	if err != nil {
		c.ready = false // reset wallet ready
		return errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := w.Sign(dataToSign.Data)

	reqCreateAddr := notaryserver.CreateAddressRequest{
		Address:   w.Address(),
		Token:     token,
		Data:      dataToSign.Data,
		Hash:      hash,
		Signature: signature,
	}
	var resCreateAddr notaryserver.CreateAddressResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.CreateAddressURL)
	if err := httpclient.MakePost(c.timeout, url, reqCreateAddr, &resCreateAddr); err != nil {
		c.ready = false // reset wallet ready
		return err
	}

	if !resCreateAddr.Success {
		c.ready = false // reset wallet ready
		return errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to create address"))
	}

	if resCreateAddr.Address != w.Address() {
		c.ready = false // reset wallet ready
		return errors.Join(httpclient.ErrServerReturnsInconsistentData, errors.New("failed to create address"))
	}

	return nil
}

// Address reads the wallet address.
// Address is a string representation of wallet public key.
func (c *Client) Address() (string, error) {
	if !c.ready {
		return "", httpclient.ErrWalletNotReady
	}

	return c.w.Address(), nil
}

// ProposeTransaction sends a Transaction proposal to the API server for provided receiver address.
// Subject describes how to read the data from the transaction. For example, if the subject is "json",
// then the data can by decoded to map[sting]any, when subject "pdf" than it should be decoded by proper pdf decoder,
// when "csv" then it should be decoded by proper csv decoder.
// Client is not responsible for decoding the data, it is only responsible for sending the data to the API server.
func (c *Client) ProposeTransaction(receiverAddr string, subject string, spc spice.Melange, data []byte) error {
	if !c.ready {
		return httpclient.ErrWalletNotReady
	}

	trx, err := transaction.New(subject, spc, data, receiverAddr, &c.w)
	if err != nil {
		return errors.Join(httpclient.ErrSigningFailed, err)
	}

	req := notaryserver.TransactionProposeRequest{
		ReceiverAddr: receiverAddr,
		Transaction:  trx,
	}
	var res notaryserver.TransactionConfirmProposeResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.ProposeTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return errors.Join(httpclient.ErrRejectedByServer, err)
	}

	if !res.Success {
		return errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to propose transaction"))
	}

	if !bytes.Equal(trx.Hash[:], res.TrxHash[:]) {
		return errors.Join(httpclient.ErrServerReturnsInconsistentData, errors.New("failed to propose transaction"))
	}

	return nil
}

// ConfirmTransaction confirms transaction by signing it with the wallet
// and then sending it to the API server.
func (c *Client) ConfirmTransaction(notaryNodeURL string, trx *transaction.Transaction) error {
	if !c.ready {
		return httpclient.ErrWalletNotReady
	}

	if _, err := trx.Sign(&c.w, c.verifier); err != nil {
		return errors.Join(httpclient.ErrSigningFailed, err)
	}

	rootURL := c.apiRoot
	if notaryNodeURL != "" {
		_, err := url.Parse(notaryNodeURL)
		if err != nil {
			rootURL = notaryNodeURL
		}
	}

	var res notaryserver.TransactionConfirmProposeResponse
	url := fmt.Sprintf("%s%s", rootURL, notaryserver.ConfirmTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, trx, &res); err != nil {
		return errors.Join(httpclient.ErrRejectedByServer, err)
	}

	if !res.Success {
		return errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to confirm transaction"))
	}

	if !bytes.Equal(trx.Hash[:], res.TrxHash[:]) {
		return errors.Join(httpclient.ErrServerReturnsInconsistentData, errors.New("failed to confirm transaction"))
	}

	return nil
}

// RejectTransactions rejects given transactions.
// Transaction will be rejected if the transaction receiver is a given wellet public address.
// Returns hashes of all the rejected transactions or error otherwise.
func (c *Client) RejectTransactions(notaryNodeURL string, trxs []transaction.Transaction) ([][32]byte, error) {
	addr, err := c.Address()
	if err != nil {
		return nil, err
	}

	rootURL := c.apiRoot
	if notaryNodeURL != "" {
		_, err := url.Parse(notaryNodeURL)
		if err != nil {
			rootURL = notaryNodeURL
		}
	}

	data, err := c.DataToSign(rootURL)
	if err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.TransactionsRejectRequest{
		Address:      addr,
		Data:         data.Data,
		Hash:         hash,
		Signature:    signature,
		Transactions: trxs,
	}

	var res notaryserver.TransactionsRejectResponse
	url := fmt.Sprintf("%s%s", rootURL, notaryserver.RejectTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to reject transactions"))
	}

	return res.TrxHashes, nil
}

// ReadWaitingTransactions reads all waiting transactions belonging to current wallet from the API server.
func (c *Client) ReadWaitingTransactions(notaryNodeURL string) ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, httpclient.ErrWalletNotReady
	}

	rootURL := c.apiRoot
	if notaryNodeURL != "" {
		_, err := url.Parse(notaryNodeURL)
		if err != nil {
			rootURL = notaryNodeURL
		}
	}

	data, err := c.DataToSign(rootURL)
	if err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.TransactionsRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	url := fmt.Sprintf("%s%s", rootURL, notaryserver.AwaitedTransactionURL)

	var res notaryserver.AwaitedTransactionsResponse
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to read waiting transactions"))
	}

	return res.AwaitedTransactions, nil
}

// ReadIssuedTransactions reads all issued transactions belonging to current wallet from the API server.
func (c *Client) ReadIssuedTransactions(notaryNodeURL string) ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, httpclient.ErrWalletNotReady
	}

	rootURL := c.apiRoot
	if notaryNodeURL != "" {
		_, err := url.Parse(notaryNodeURL)
		if err != nil {
			rootURL = notaryNodeURL
		}
	}

	data, err := c.DataToSign(rootURL)
	if err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.TransactionsRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
	}
	var res notaryserver.IssuedTransactionsResponse
	url := fmt.Sprintf("%s%s", rootURL, notaryserver.IssuedTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to read issued transactions"))
	}

	return res.IssuedTransactions, nil
}

// ReadRejectedTransactions reads rejected transactions belonging to current wallet from the API server.
// Method allows for paggination with offset and limit.
func (c *Client) ReadRejectedTransactions(offset, limit int) ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, httpclient.ErrWalletNotReady
	}

	data, err := c.DataToSign(c.apiRoot)
	if err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.TransactionsRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
		Offset:    offset,
		Limit:     limit,
	}
	var res notaryserver.RejectedTransactionsResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.RejectedTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to read rejected transactions"))
	}

	return res.RejectedTransactions, nil
}

// ReadApprovedTransactions reads approved transactions belonging to current wallet from the API server.
// Method allows for paggination with offset and limit.
func (c *Client) ReadApprovedTransactions(offset, limit int) ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, httpclient.ErrWalletNotReady
	}

	data, err := c.DataToSign(c.apiRoot)
	if err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.TransactionsRequest{
		Address:   c.w.Address(),
		Data:      data.Data,
		Hash:      hash,
		Signature: signature,
		Offset:    offset,
		Limit:     limit,
	}
	var res notaryserver.ApprovedTransactionsResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.RejectedTransactionURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return nil, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Success {
		return nil, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to read approved transactions"))
	}

	return res.ApprovedTransactions, nil
}

// GenerateToken generates a token for the given time in the central node repository.
// It is only permitted to generate a token if wallet has admin permissions in the central node.
func (c *Client) GenerateToken(t time.Time) (token.Token, error) {
	if !c.ready {
		return token.Token{}, httpclient.ErrWalletNotReady
	}

	data, err := c.DataToSign(c.apiRoot)
	if err != nil {
		return token.Token{}, errors.Join(httpclient.ErrRejectedByServer, err)
	}

	hash, signature := c.w.Sign(data.Data)
	req := notaryserver.GenerateTokenRequest{
		Address:    c.w.Address(),
		Data:       data.Data,
		Hash:       hash,
		Signature:  signature,
		Expiration: t.UnixMicro(),
	}

	var res notaryserver.GenerateTokenResponse
	url := fmt.Sprintf("%s%s", c.apiRoot, notaryserver.GenerateTokenURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return token.Token{}, errors.Join(httpclient.ErrRejectedByServer, err)
	}
	if !res.Valid {
		return token.Token{}, errors.Join(httpclient.ErrRejectedByServer, errors.New("failed to generate token"))
	}

	return res, nil
}

// SaveWalletToFile saves the wallet to the file in the path.
func (c *Client) SaveWalletToFile() error {
	if !c.ready {
		return httpclient.ErrWalletNotReady
	}

	return c.wrs.SaveWallet(&c.w)
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

// DataToSign returns data to sign for the current wallet.
// Data to sign are randomly generated bytes by the server and stored in pair with the address.
// Signing this data is a proof that the signing public address is the owner of the wallet a making request.
func (c *Client) DataToSign(notaryNodeURL string) (notaryserver.DataToSignResponse, error) {
	addr, err := c.Address()
	if err != nil {
		return notaryserver.DataToSignResponse{}, err
	}

	req := notaryserver.DataToSignRequest{
		Address: addr,
	}
	var resp notaryserver.DataToSignResponse
	url := fmt.Sprintf("%s%s", notaryNodeURL, notaryserver.DataToValidateURL)
	if err := httpclient.MakePost(c.timeout, url, req, &resp); err != nil {
		return notaryserver.DataToSignResponse{}, err
	}
	return resp, nil
}

// Sign signs the given data with the wallet and returns digest and signature or error otherwise.
// This process creates a proof for the API server that requesting client is the owner of the wallet.
func (c *Client) Sign(d []byte) (digest [32]byte, signature []byte, err error) {
	if !c.ready {
		return digest, signature, httpclient.ErrWalletNotReady
	}
	digest, signature = c.w.Sign(d)
	return
}

func (c *Client) CreateWebhook(webHookURL string) error {
	data, err := c.DataToSign(c.apiRoot)
	if err != nil {
		return err
	}

	buf := make([]byte, 0, len(data.Data)+len(webHookURL))
	buf = append(buf, append(data.Data, []byte(webHookURL)...)...)

	digest, signature, err := c.Sign(buf)
	if err != nil {
		return err
	}

	addr, err := c.Address()
	if err != nil {
		return err
	}

	req := helperserver.CreateRemoveUpdateHookRequest{
		URL:       webHookURL,
		Address:   addr,
		Data:      data.Data,
		Digest:    digest,
		Signature: signature,
	}

	var res helperserver.CreateRemoveUpdateHookResponse

	url := fmt.Sprintf("%s%s", c.apiRoot, helperserver.TransactionHookURL)
	if err := httpclient.MakePost(c.timeout, url, req, &res); err != nil {
		return err
	}

	if !res.Ok {
		if res.Err != "" {
			return errors.New(res.Err)
		}
		return errors.New("failed to create webhook, something went wrong")
	}

	return nil
}

// FlushWalletFromMemory flushes the wallet from the memory.
// Do it after you have saved the wallet to the file.
// It is recommended to use this just before logging out from the UI
// or closing the front end app that.
func (c *Client) FlushWalletFromMemory() {
	c.w = wallet.Wallet{}
	c.ready = false
}
