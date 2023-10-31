package walletapi

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/notaryserver"
	"github.com/bartossh/Computantis/protobufcompiled"
	"github.com/bartossh/Computantis/transaction"
	"github.com/bartossh/Computantis/walletmiddleware"
)

// RunGRPC runs the service application that exposes the GRPC API for creating, validating and signing transactions.
// This blocks until the context is canceled.
func RunGRPC(ctx context.Context, cfg Config, log logger.Logger, timeout time.Duration, fw transaction.Verifier,
	wrs walletmiddleware.WalletReadSaver, walletCreator walletmiddleware.NewSignValidatorCreator,
) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	c := walletmiddleware.NewClient(cfg.NotaryNodeURL, timeout, fw, wrs, walletCreator)
	defer c.FlushWalletFromMemory()

	if err := c.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	v := walletmiddleware.NewClient(cfg.HelperNodeURL, timeout, fw, wrs, walletCreator)
	defer v.FlushWalletFromMemory()

	if err := v.ReadWalletFromFile(); err != nil {
		log.Info(fmt.Sprintf("error with reading wallet from file: %s", err))
	}

	s := app{log: log, centralNodeClient: *c, validatorNodeClient: *v}

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

// Alive implements wallet client API GRPC alive procedure.
// Procedure informs about notary node API version and header.
// This procedure allows to check is notary node alive, is client node alive
// and are the client and notary nodes using compatible API version.
func (a *app) Alive(context.Context, *emptypb.Empty) (*protobufcompiled.AliveInfo, error) {
	var aliveResp protobufcompiled.AliveInfo
	if err := a.centralNodeClient.ValidateApiVersion(); err != nil {
		return &aliveResp, err
	}
	aliveResp.Alive = true
	aliveResp.ApiVersion = notaryserver.ApiVersion
	aliveResp.ApiHeader = notaryserver.Header
	return &aliveResp, nil
}

// Address implements wallet client API GRPC address procedure.
// Procedure returns client public address if the client API version is valid.
// Address is in base58 format and contains wallet version and control sum.
func (a *app) Address(context.Context, *emptypb.Empty) (*protobufcompiled.WalletPublicAddress, error) {
	var walletResp protobufcompiled.WalletPublicAddress
	if err := a.centralNodeClient.ValidateApiVersion(); err != nil {
		walletResp.Info.Err = err.Error()
		return &walletResp, err
	}

	addr, err := a.centralNodeClient.Address()
	if err != nil {
		walletResp.Info.Err = err.Error()
		return &walletResp, err
	}
	walletResp.Info.Ok = true
	walletResp.Address = addr
	return &walletResp, nil
}

// IssuedTransactions implements wallet client API GRPC read issued transactions.
// Procedure returns client issued transactions if exists in the system.
func (a *app) IssuedTransactions(ctx context.Context, notaryNode *protobufcompiled.NotaryNode) (*protobufcompiled.Transactions, error) {
	if notaryNode.Url == "" {
		a.log.Error("notary node URL is empty in the message")
		err := errors.New("empty notary node URL")
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}
	if _, err := url.Parse(notaryNode.Url); err != nil {
		a.log.Error(fmt.Sprintf("wrong URL format, notary node URL cannot be parsed, %s", err))
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	transactions, err := a.centralNodeClient.ReadIssuedTransactions(notaryNode.Url)
	if err != nil {
		err := fmt.Errorf("error getting issued transactions: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	trxs := make([]*protobufcompiled.Transaction, 0, len(transactions))
	for _, transaction := range transactions {
		trxs = append(trxs, &protobufcompiled.Transaction{
			Subject:           transaction.Subject,
			Data:              transaction.Data,
			Hash:              transaction.Hash[:],
			CreaterdAt:        uint64(transaction.CreatedAt.UnixNano()),
			IssuerAddress:     transaction.IssuerAddress,
			ReceiverAddress:   transaction.ReceiverAddress,
			IssuerSignature:   transaction.IssuerSignature,
			ReceiverSignature: transaction.ReceiverSignature,
		})
	}

	return &protobufcompiled.Transactions{
		Transactions: trxs,
		Info: &protobufcompiled.ServerInfo{
			Ok: true,
		},
	}, nil
}

// ReceivedTransactions implements wallet client API GRPC read received transactions.
// Procedure returns client awaiting transactions if exists in the system.
func (a *app) ReceivedTransactions(ctx context.Context, notaryNode *protobufcompiled.NotaryNode) (*protobufcompiled.Transactions, error) {
	if notaryNode.Url == "" {
		a.log.Error("notary node URL is empty in the message")
		err := errors.New("empty notary node URL")
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}
	if _, err := url.Parse(notaryNode.Url); err != nil {
		a.log.Error(fmt.Sprintf("wrong URL format, notary node URL cannot be parsed, %s", err))
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	transactions, err := a.centralNodeClient.ReadWaitingTransactions(notaryNode.Url)
	if err != nil {
		err := fmt.Errorf("error getting waiting transactions: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	trxs := make([]*protobufcompiled.Transaction, 0, len(transactions))
	for _, transaction := range transactions {
		trxs = append(trxs, &protobufcompiled.Transaction{
			Subject:           transaction.Subject,
			Data:              transaction.Data,
			Hash:              transaction.Hash[:],
			CreaterdAt:        uint64(transaction.CreatedAt.UnixNano()),
			IssuerAddress:     transaction.IssuerAddress,
			ReceiverAddress:   transaction.ReceiverAddress,
			IssuerSignature:   transaction.IssuerSignature,
			ReceiverSignature: transaction.ReceiverSignature,
		})
	}

	return &protobufcompiled.Transactions{
		Transactions: trxs,
		Info: &protobufcompiled.ServerInfo{
			Ok: true,
		},
	}, nil
}

// ApprovedTransactions implements wallet client API GRPC read approved transactions.
// Procedure returns client approved transactions if exists in the system.
func (a *app) ApprovedTransactions(ctx context.Context, paggination *protobufcompiled.Paggination) (*protobufcompiled.Transactions, error) {
	transactions, err := a.centralNodeClient.ReadApprovedTransactions(int(paggination.Offset), int(paggination.Limit))
	if err != nil {
		err := fmt.Errorf("error getting approved transactions: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	trxs := make([]*protobufcompiled.Transaction, 0, len(transactions))
	for _, transaction := range transactions {
		trxs = append(trxs, &protobufcompiled.Transaction{
			Subject:           transaction.Subject,
			Data:              transaction.Data,
			Hash:              transaction.Hash[:],
			CreaterdAt:        uint64(transaction.CreatedAt.UnixNano()),
			IssuerAddress:     transaction.IssuerAddress,
			ReceiverAddress:   transaction.ReceiverAddress,
			IssuerSignature:   transaction.IssuerSignature,
			ReceiverSignature: transaction.ReceiverSignature,
		})
	}

	return &protobufcompiled.Transactions{
		Transactions: trxs,
		Info: &protobufcompiled.ServerInfo{
			Ok: true,
		},
	}, nil
}

// RejectedTransactions implements wallet client API GRPC read rejected transactions.
// Procedure returns client rejected transactions if exists in the system.
func (a *app) RejectedTransactions(ctx context.Context, paggination *protobufcompiled.Paggination) (*protobufcompiled.Transactions, error) {
	transactions, err := a.centralNodeClient.ReadRejectedTransactions(int(paggination.Offset), int(paggination.Limit))
	if err != nil {
		err := fmt.Errorf("error getting rejected transactions: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.Transactions{
			Transactions: nil,
			Info: &protobufcompiled.ServerInfo{
				Ok:  false,
				Err: err.Error(),
			},
		}, err
	}

	trxs := make([]*protobufcompiled.Transaction, 0, len(transactions))
	for _, transaction := range transactions {
		trxs = append(trxs, &protobufcompiled.Transaction{
			Subject:           transaction.Subject,
			Data:              transaction.Data,
			Hash:              transaction.Hash[:],
			CreaterdAt:        uint64(transaction.CreatedAt.UnixNano()),
			IssuerAddress:     transaction.IssuerAddress,
			ReceiverAddress:   transaction.ReceiverAddress,
			IssuerSignature:   transaction.IssuerSignature,
			ReceiverSignature: transaction.ReceiverSignature,
		})
	}

	return &protobufcompiled.Transactions{
		Transactions: trxs,
		Info: &protobufcompiled.ServerInfo{
			Ok: true,
		},
	}, nil
}

// Wallet implements wallet client API GRPC create wallet.
// Procedure returns server info containing error message.
func (a *app) Wallet(ctx context.Context, wallet *protobufcompiled.CreateWallet) (*protobufcompiled.ServerInfo, error) {
	if wallet.Token == "" {
		err := errors.New("token is empty")
		a.log.Error(err.Error())
		return &protobufcompiled.ServerInfo{
			Ok:  false,
			Err: err.Error(),
		}, err
	}

	if err := a.centralNodeClient.NewWallet(wallet.Token); err != nil {
		err := fmt.Errorf("error creating wallet: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.ServerInfo{
			Ok:  false,
			Err: err.Error(),
		}, err
	}

	if err := a.centralNodeClient.SaveWalletToFile(); err != nil {
		err := fmt.Errorf("error saving wallet to file: %v", err)
		a.log.Error(err.Error())
		return &protobufcompiled.ServerInfo{
			Ok:  false,
			Err: err.Error(),
		}, err
	}

	return &protobufcompiled.ServerInfo{Ok: true}, nil
}

// WebHook implements wallet client API GRPC create web hook.
// Procedure returns server info containing error message.
func (a *app) WebHook(ctx context.Context, webHook *protobufcompiled.CreateWebHook) (*protobufcompiled.ServerInfo, error) {
	if webHook.Url == "" {
		err := errors.New("wrong JSON format when creating a web hook")
		a.log.Error(err.Error())
		return &protobufcompiled.ServerInfo{
			Ok:  false,
			Err: err.Error(),
		}, err
	}

	if err := a.validatorNodeClient.CreateWebhook(webHook.Url); err != nil {
		a.log.Error(err.Error())
		return &protobufcompiled.ServerInfo{
			Ok:  false,
			Err: err.Error(),
		}, err
	}

	return &protobufcompiled.ServerInfo{Ok: true}, nil
}
