package synchronizerserver

import (
	"context"
	"fmt"
	"net"

	"github.com/bartossh/Computantis/logger"
	"github.com/bartossh/Computantis/protobufcompiled"
	"google.golang.org/grpc"
)

// Config holds configuration of the synchronizer server.
type Config struct {
	Port int `yaml:"port"`
}

type server struct {
	log logger.Logger
	protobufcompiled.UnsafeSynchronizerServer
}

func (s *server) AddToQueue(ctx context.Context, info *protobufcompiled.NodeInfo) (*protobufcompiled.QueueStatus, error) {
	return nil, nil
}

func (s *server) RemoveFromQueue(ctx context.Context, info *protobufcompiled.NodeInfo) (*protobufcompiled.QueueStatus, error) {
	return nil, nil
}

// Run runs the synchronizer server.
func Run(ctx context.Context, cfg Config, log logger.Logger) error {
	ctxx, cancel := context.WithCancel(ctx)
	defer cancel()

	s := server{log: log}

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
