package walletmiddleware

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/bartossh/Computantis/src/httpclient"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/transformers"
	"github.com/bartossh/Computantis/src/versioning"
	"github.com/bartossh/Computantis/src/wallet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

const (
	checksumLength = 4
	version        = byte(0x00)
)

var (
	ErrEmptyMessage   = errors.New("unexpected empty message")
	ErrWalletNotReady = errors.New("wallet not ready")
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
	conn          *grpc.ClientConn
	client        protobufcompiled.NotaryAPIClient
	apiRoot       string
	w             wallet.Wallet
	ready         bool
}

// NewClient creates a new rest client.
func NewClient(
	apiRoot string, fw transaction.Verifier,
	wrs WalletReadSaver, walletCreator NewSignValidatorCreator,
) (*Client, error) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(apiRoot, opts)
	if err != nil {
		return nil, fmt.Errorf("dial failed, %s", err)
	}

	client := protobufcompiled.NewNotaryAPIClient(conn)

	return &Client{apiRoot: apiRoot, verifier: fw, wrs: wrs, walletCreator: walletCreator, conn: conn, client: client}, nil
}

// Close closes connection with the notary node.
func (c *Client) Close() error {
	return c.conn.Close()
}

// ValidateApiVersion makes a call to the API server and validates client and server API versions and header correctness.
// If API version not much it is returning an error as accessing the API server with different API version
// may lead to unexpected results.
func (c *Client) ValidateApiVersion(ctx context.Context) error {
	alive, err := c.client.Alive(ctx, &emptypb.Empty{})
	if err != nil {
		return err
	}
	if alive == nil {
		return ErrEmptyMessage
	}

	if alive.ApiVersion != versioning.ApiVersion {
		return errors.Join(httpclient.ErrApiVersionMismatch, fmt.Errorf("expected %s but got %s", versioning.ApiVersion, alive.ApiVersion))
	}

	if alive.ApiHeader != versioning.Header {
		return errors.Join(httpclient.ErrApiHeaderMismatch, fmt.Errorf("expected %s but got %s", versioning.Header, alive.ApiHeader))
	}

	return nil
}

// NewWallet creates a new wallet.
func (c *Client) NewWallet() error {
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

	return nil
}

// ProposeTransaction proposes transaction to the Computantis DAG.
func (c *Client) ProposeTransaction(ctx context.Context, receiverAddr string, subject string, spc spice.Melange, data []byte) error {
	if !c.ready {
		return ErrWalletNotReady
	}

	trx, err := transaction.New(subject, spc, data, receiverAddr, &c.w)
	if err != nil {
		return errors.Join(httpclient.ErrSigningFailed, err)
	}

	protoTrx, err := transformers.TrxToProtoTrx(&trx)
	if err != nil {
		return err
	}

	_, err = c.client.Propose(ctx, protoTrx)

	return err
}

// ConfirmTransaction confirms transaction by signing it with the wallet.
func (c *Client) ConfirmTransaction(ctx context.Context, notaryNodeURL string, trx *transaction.Transaction) error {
	if !c.ready {
		return httpclient.ErrWalletNotReady
	}

	if _, err := trx.Sign(&c.w, c.verifier); err != nil {
		return errors.Join(httpclient.ErrSigningFailed, err)
	}

	protoTrx, err := transformers.TrxToProtoTrx(trx)
	if err != nil {
		return err
	}

	client := c.client
	if notaryNodeURL != c.apiRoot {
		opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
		conn, err := grpc.Dial(notaryNodeURL, opts)
		if err != nil {
			return fmt.Errorf("dial failed, %s", err)
		}
		defer conn.Close()
		client = protobufcompiled.NewNotaryAPIClient(conn)
	}

	if _, err := client.Confirm(ctx, protoTrx); err != nil {
		return err
	}

	return nil
}

// RejectTransactions rejects given transaction by the hash. Can by performed only by the receiver.
func (c *Client) RejectTransactions(ctx context.Context, notaryNodeURL string, hash [32]byte) error {
	if !c.ready {
		return httpclient.ErrWalletNotReady
	}

	digest, signature := c.w.Sign(hash[:])

	client := c.client
	if notaryNodeURL != c.apiRoot {
		opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
		conn, err := grpc.Dial(notaryNodeURL, opts)
		if err != nil {
			return fmt.Errorf("dial failed, %s", err)
		}
		defer conn.Close()
		client = protobufcompiled.NewNotaryAPIClient(conn)
	}

	if _, err := client.Reject(ctx, &protobufcompiled.SignedHash{
		Address:   c.w.Address(),
		Data:      hash[:],
		Signature: signature,
		Hash:      digest[:],
	}); err != nil {
		return err
	}

	return nil
}

// ReadWaitingTransactions reads all waiting transactions belonging to current wallet.
func (c *Client) ReadWaitingTransactions(ctx context.Context, notaryNodeURL string) ([]transaction.Transaction, error) {
	if !c.ready {
		return nil, httpclient.ErrWalletNotReady
	}

	client := c.client
	if notaryNodeURL != c.apiRoot {
		opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
		conn, err := grpc.Dial(notaryNodeURL, opts)
		if err != nil {
			return nil, fmt.Errorf("dial failed, %s", err)
		}
		defer conn.Close()
		client = protobufcompiled.NewNotaryAPIClient(conn)
	}

	data, err := client.Data(ctx, &protobufcompiled.Address{Public: c.w.Address()})
	if err != nil {
		return nil, err
	}

	digest, signature := c.w.Sign(data.Blob)
	proto, err := client.Waiting(ctx, &protobufcompiled.SignedHash{
		Address:   c.w.Address(),
		Data:      data.Blob,
		Signature: signature,
		Hash:      digest[:],
	})
	if err != nil {
		return nil, err
	}

	trxs := make([]transaction.Transaction, 0, len(proto.Array))
	for _, protoTrx := range proto.Array {
		trx, err := transformers.ProtoTrxToTrx(protoTrx)
		if err != nil {
			continue
		}
		trxs = append(trxs, trx)
	}

	return trxs, nil
}

// ReadSavedTransaction reads saved transaction from connected node.
func (c *Client) ReadSavedTransaction(ctx context.Context, hash [32]byte) (transaction.Transaction, error) {
	if !c.ready {
		return transaction.Transaction{}, httpclient.ErrWalletNotReady
	}

	digest, signature := c.w.Sign(hash[:])

	protoTrx, err := c.client.Saved(ctx, &protobufcompiled.SignedHash{
		Address:   c.w.Address(),
		Data:      hash[:],
		Signature: signature,
		Hash:      digest[:],
	})
	if err != nil {
		return transaction.Transaction{}, err
	}

	return transformers.ProtoTrxToTrx(protoTrx)
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

// CreateWebhook creates webhook in the webhooks server.
func (c *Client) CreateWebhook(ctx context.Context, webHookURL, clientURL string) error {
	if _, err := url.Parse(webHookURL); err != nil {
		return err
	}
	if _, err := url.Parse(clientURL); err != nil {
		return err
	}
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(webHookURL, opts)
	if err != nil {
		return fmt.Errorf("dial failed, %s", err)
	}
	defer conn.Close()
	client := protobufcompiled.NewWebhooksAPIClient(conn)

	digest, signature := c.w.Sign([]byte(clientURL))

	_, err = client.Webhooks(ctx, &protobufcompiled.SignedHash{
		Address:   c.w.Address(),
		Data:      []byte(clientURL),
		Signature: signature,
		Hash:      digest[:],
	})
	if err != nil {
		return err
	}

	return nil
}

// Address returns public address of the wallet.
func (c *Client) Address() (string, error) {
	if !c.ready {
		return "", ErrWalletNotReady
	}
	return c.w.Address(), nil
}

func (c *Client) URL() string {
	return c.apiRoot
}

// FlushWalletFromMemory flushes the wallet from the memory.
// Do it after you have saved the wallet to the file.
// It is recommended to use this just before logging out from the UI
// or closing the front end app that.
func (c *Client) FlushWalletFromMemory() {
	c.w = wallet.Wallet{}
	c.ready = false
}
