package addons

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bartossh/Computantis/src/protobufcompiled"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrConnectionClosed = errors.New("connection has been closed")
	ErrServerFailure    = errors.New("server failure")
	ErrEmptyData        = errors.New("empty data")
)

// APICaller calls the add-ons api.
type APICaller struct {
	conn   *grpc.ClientConn
	client protobufcompiled.AddonsAPIClient
	token  string
}

// New creates a new APICaller if connection to the URL can be created or error otherwise.
// Canceling given context closes the connection.
func New(ctx context.Context, url, token string) (*APICaller, error) {
	opts := grpc.WithTransportCredentials(insecure.NewCredentials()) // TODO: remove when credentials are set
	conn, err := grpc.Dial(url, opts)
	if err != nil {
		return nil, fmt.Errorf("dial failed, %s", err)
	}

	client := protobufcompiled.NewAddonsAPIClient(conn)

	api := APICaller{
		conn:   conn,
		client: client,
		token:  token,
	}

	go api.runCtxCancelListener(ctx)

	return &api, nil
}

func (a *APICaller) runCtxCancelListener(ctx context.Context) {
	<-ctx.Done()
	a.conn.Close()
}

// AnalyzeTransaction implements immunity TransactionAntibodyProvider
func (a *APICaller) AnalyzeTransaction(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return ErrEmptyData
	}
	errorMsg, err := a.client.AnalyzeTransaction(ctx, &protobufcompiled.AddonsMessage{Data: data})
	if err != nil {
		if strings.Contains(err.Error(), "connection closed") {
			return ErrConnectionClosed
		}
		return ErrServerFailure
	}
	if errorMsg.Error != "" {
		return errors.New(errorMsg.Error)
	}

	return nil
}
