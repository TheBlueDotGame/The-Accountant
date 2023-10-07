package synchronizerserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"
)

const tickerTimer = time.Second

// Config holds configuration of the synchronizer server.
type Config struct {
	Token string `yaml:"token"`
	Port  int    `yaml:"port"`
}

type server struct {
	log   logger.Logger
	queue *queue
	token string
	protobufcompiled.UnsafeSynchronizerServer
}

func (s *server) AddToQueue(ctx context.Context, info *protobufcompiled.NodeInfo) (*emptypb.Empty, error) {
	if info.Token != s.token {
		return nil, errors.New("invalid token, access denied")
	}
	return &emptypb.Empty{}, s.queue.addNode(info)
}

func (s *server) RemoveFromQueue(ctx context.Context, info *protobufcompiled.NodeInfo) (*emptypb.Empty, error) {
	if info.Token != s.token {
		return nil, errors.New("invalid token, access denied")
	}
	s.queue.removeNode(info)
	return &emptypb.Empty{}, nil
}

// Run runs the synchronizer server.
// This function blocks until context is not canceled.
func Run(ctx context.Context, cfg Config, log logger.Logger) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	q := newQueue(log)
	q.run(ctx, tickerTimer)

	s := server{log: log, queue: q, token: cfg.Token}

	grpcServer := grpc.NewServer()
	protobufcompiled.RegisterSynchronizerServer(grpcServer, &s)

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

	if err != nil {
		return err
	}

	grpcServer.GracefulStop()

	return nil
}
