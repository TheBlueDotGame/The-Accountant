package webhooksserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/bartossh/Computantis/src/logger"
	"github.com/bartossh/Computantis/src/protobufcompiled"
	"github.com/bartossh/Computantis/src/transaction"
	"github.com/bartossh/Computantis/src/versioning"
	"github.com/bartossh/Computantis/src/webhooks"
)

const (
	Header = "Computantis-Web-Hooks"
)

var (
	ErrNotAuthorized     = errors.New("not authorized")
	ErrProcessingFailure = errors.New("processing failed")
)

// Config contains configuration of the validator.
type Config struct {
	Port int `yaml:"port"` // port on which validator will listen for http requests
}

// WebhookCreateRemovePoster provides methods to create, remove webhooks and post messages to webhooks.
type WebhookCreateRemovePoster interface {
	CreateWebhook(trigger byte, address string, h webhooks.Hook) error
	RemoveWebhook(trigger byte, address string, h webhooks.Hook) error
	PostWebhookNewTransaction(publicAddresses []string, storingNodeURL string)
}

// NodesCommunicationSubscriber provides facade access to communication between nodes publisher endpoint.
type NodesCommunicationSubscriber interface {
	SubscribeNewTransactionsForAddresses(call transaction.TrxAddressesSubscriberCallback, log logger.Logger) error
}

type verifier interface {
	Verify(message, signature []byte, hash [32]byte, address string) error
}

type app struct {
	protobufcompiled.UnimplementedWebhooksAPIServer
	ver verifier
	wh  WebhookCreateRemovePoster
	log logger.Logger
}

// Run initializes webhooks server and GRPC API server.
// It will block until the context is canceled.
func Run(
	ctx context.Context, cfg Config, sub NodesCommunicationSubscriber, log logger.Logger, ver verifier, wh WebhookCreateRemovePoster,
) error {
	a := &app{
		log: log,
		ver: ver,
		wh:  wh,
	}

	if cfg.Port < 0 || cfg.Port > 65535 {
		return errors.New("port out of range 0 - 65535")
	}

	if err := sub.SubscribeNewTransactionsForAddresses(a.processNewTrxIssuedByAddresses, log); err != nil {
		log.Error(fmt.Sprintf("webhooks server failed, %s", err))
		return err
	}

	return a.runServer(ctx, cfg.Port)
}

func (a *app) runServer(ctx context.Context, port int) error {
	ctxx, cancel := context.WithCancel(ctx)
	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", port))
	if err != nil {
		cancel()
		return err
	}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterWebhooksAPIServer(grpcServer, a)

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

	return err
}

func (a *app) processNewTrxIssuedByAddresses(receivers []string, storingNodeURL string) {
	go a.wh.PostWebhookNewTransaction(receivers, storingNodeURL) // post concurrently
}

// Alive returns alive data such as API Version and API Header.
func (a *app) Alive(_ context.Context, _ *emptypb.Empty) (*protobufcompiled.AliveData, error) {
	return &protobufcompiled.AliveData{ApiVersion: versioning.ApiVersion, ApiHeader: versioning.WebhooksHeader, PublicAddress: ""}, nil
}

// Webhooks creates a webhook for the client.
func (a *app) Webhooks(ctx context.Context, in *protobufcompiled.SignedHash) (*emptypb.Empty, error) {
	if err := a.ver.Verify(in.Data, in.Signature, [32]byte(in.Hash), in.Address); err != nil {
		a.log.Error(fmt.Sprintf("create webhook invalid signature: %s", err.Error()))
		return nil, ErrNotAuthorized
	}

	_, err := url.Parse(string(in.Data))
	if err != nil {
		return nil, ErrProcessingFailure
	}

	h := webhooks.Hook{
		URL: string(in.Data),
	}
	if err := a.wh.CreateWebhook(webhooks.TriggerNewTransaction, in.Address, h); err != nil {
		a.log.Error(fmt.Sprintf("create webhook failed, %s", err.Error()))
		return nil, ErrProcessingFailure
	}

	return &emptypb.Empty{}, nil
}
