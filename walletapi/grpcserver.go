package walletapi

import (
	"context"
	"fmt"
	"net"
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
