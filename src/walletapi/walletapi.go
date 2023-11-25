package walletapi

import (
	"context"
	"fmt"
	"net"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/spice"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/transformers"
	"github.com/bartossh/Computantis/src/walletmiddleware"
)

// Config is the configuration for the notaryserver
type Config struct {
	Port            string `yaml:"port"`
	NotaryNodeURL   string `yaml:"notary_node_url"`
	WebhooksNodeURL string `yaml:"webhooks_node_url"`
}

type app struct {
	protobufcompiled.UnimplementedWalletClientAPIServer
	log               logger.Logger
	webhooksURL       string
	centralNodeClient walletmiddleware.Client
}

// Run runs the service application that exposes the GRPC API for creating, validating and signing transactions.
// This function blocks until the context is canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger, fw transaction.Verifier,
	wrs walletmiddleware.WalletReadSaver, walletCreator walletmiddleware.NewSignValidatorCreator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	c, err := walletmiddleware.NewClient(cfg.NotaryNodeURL, fw, wrs, walletCreator)
	if err != nil {
		return err
	}
	defer c.FlushWalletFromMemory()

	if err := c.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	s := app{log: log, centralNodeClient: *c, webhooksURL: cfg.WebhooksNodeURL}
	defer s.close()

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%s", cfg.Port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterWalletClientAPIServer(grpcServer, &s)

	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			cancel()
		}
	}()

	<-ctxx.Done()

	if err != nil {
		return err
	}

	grpcServer.GracefulStop()

	return nil
}

func (a *app) close() error {
	return a.centralNodeClient.Close()
}

// Alive implements wallet client API GRPC alive procedure.
// Procedure informs about notary node API version and header.
// This procedure allows to check is notary node alive, is client node alive
// and are the client and notary nodes using compatible API version.
func (a *app) Alive(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, a.centralNodeClient.ValidateApiVersion(ctx)
}

// WalletPublicAddress implements wallet client API GRPC address procedure.
// Procedure returns client public address if the client API version is valid.
// Address is in base58 format and contains wallet version and control sum.
func (a *app) WalletPublicAddress(ctx context.Context, _ *emptypb.Empty) (*protobufcompiled.Address, error) {
	address, err := a.centralNodeClient.Address()
	if err != nil {
		return nil, err
	}

	return &protobufcompiled.Address{Public: address}, nil
}

// Issue issues the transaction to the notary node.
func (a *app) Issue(ctx context.Context, in *protobufcompiled.IssueTrx) (*emptypb.Empty, error) {
	err := a.centralNodeClient.ProposeTransaction(
		ctx, in.ReceiverAddress, in.Subject, spice.New(in.Spice.Currency, in.Spice.SuplementaryCurrency), in.Data,
	)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Approve approves the transaction and sends it to the node that keeps awaiting transaction.
func (a *app) Approve(ctx context.Context, in *protobufcompiled.TransactionApproved) (*emptypb.Empty, error) {
	trx, err := transformers.ProtoTrxToTrx(in.Transaction)
	if err != nil {
		return nil, err
	}
	if err := a.centralNodeClient.ConfirmTransaction(ctx, in.Url, &trx); err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Reject rejects transaction removing it from the node that keeps awaiting transaction.
func (a *app) Reject(ctx context.Context, in *protobufcompiled.TrxHash) (*emptypb.Empty, error) {
	err := a.centralNodeClient.RejectTransactions(ctx, in.Url, [32]byte(in.Hash))
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Waiting pulls waiting transactions from the node that keeps awaiting transactions.
func (a *app) Waiting(ctx context.Context, in *protobufcompiled.NotaryNode) (*protobufcompiled.Transactions, error) {
	trxs, err := a.centralNodeClient.ReadWaitingTransactions(ctx, in.Url)
	if err != nil {
		return nil, err
	}
	result := &protobufcompiled.Transactions{Array: make([]*protobufcompiled.Transaction, 0, len(trxs)), Len: uint64(len(trxs))}
	for _, trx := range trxs {
		protoTrx, err := transformers.TrxToProtoTrx(trx)
		if err != nil {
			continue
		}
		result.Array = append(result.Array, protoTrx)
	}
	return result, nil
}

// Saved returns saved transaction in the DAG. Shall work on any Computantis node.
func (a *app) Saved(ctx context.Context, in *protobufcompiled.TrxHash) (*protobufcompiled.Transaction, error) {
	trx, err := a.centralNodeClient.ReadSavedTransaction(ctx, [32]byte(in.Hash))
	if err != nil {
		return nil, err
	}
	return transformers.TrxToProtoTrx(&trx)
}

// WebHook creates a web-hook on the WebHook Computantis node.
func (a *app) WebHook(ctx context.Context, in *protobufcompiled.CreateWebHook) (*emptypb.Empty, error) {
	err := a.centralNodeClient.CreateWebhook(ctx, a.webhooksURL, in.Url)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}
