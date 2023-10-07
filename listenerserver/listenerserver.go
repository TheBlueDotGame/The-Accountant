package listenerserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/emptypb"
)

type server struct {
	protobufcompiled.UnimplementedQueueListenerServer
	client  protobufcompiled.SynchronizerClient
	log     logger.Logger
	done    chan struct{}
	id      string
	token   string
	selfURL string
}

// Config holds configuration of the synchronizer server.
type Config struct {
	Token         string `yaml:"token"`
	SyncServerURL string `yaml:"sync_server_url"`
	SelfServerURL string `yaml:"self_server_url"`
	Port          int    `yaml:"port"`
}

// Run runs the queue listener server.
// This function blocks until context is not canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger, id string) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	_, err := url.Parse(cfg.SyncServerURL)
	if err != nil {
		return err
	}
	_, err = url.Parse(cfg.SelfServerURL)
	if err != nil {
		return err
	}

	conn, err := grpc.Dial(cfg.SyncServerURL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	client := protobufcompiled.NewSynchronizerClient(conn)

	s := server{log: log, done: make(chan struct{}, 1), id: id, token: cfg.Token, client: client, selfURL: cfg.SelfServerURL}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterQueueListenerServer(grpcServer, &s)

	lis, err := net.Listen("tcp", fmt.Sprintf("0.0.0.0:%v", cfg.Port))
	if err != nil {
		cancel()
		return err
	}
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			cancel()
		}
	}()

	<-ctxx.Done()

	close(s.done)

	if err != nil {
		return err
	}

	grpcServer.GracefulStop()

	return nil
}

func (s *server) RegisterInQueue(ctx context.Context, transactionsPerSecond uint64) error {
	_, err := s.client.AddToQueue(ctx, &protobufcompiled.NodeInfo{
		Id:                    s.id,
		Token:                 s.token,
		Address:               s.selfURL,
		TransactionsPerSecond: transactionsPerSecond,
	})
	return err
}

func (s *server) RemoveFromQueue(ctx context.Context) error {
	_, err := s.client.RemoveFromQueue(ctx, &protobufcompiled.NodeInfo{
		Id:    s.id,
		Token: s.token,
	})
	return err
}

func (s *server) Done() <-chan struct{} {
	return s.done
}

func (s *server) QueueUpdate(ctx context.Context, status *protobufcompiled.QueueStatus) (*emptypb.Empty, error) {
	if status.Token != s.token || status.Id != s.id {
		return nil, errors.New("invalid token, access denied")
	}
	s.done <- struct{}{}
	return &emptypb.Empty{}, nil
}

func (s *server) Ping(ctx context.Context, status *protobufcompiled.QueueStatus) (*emptypb.Empty, error) {
	if status.Token != s.token || status.Id != s.id {
		return nil, errors.New("invalid token, access denied")
	}
	return &emptypb.Empty{}, nil
}
