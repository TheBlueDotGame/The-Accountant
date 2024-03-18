package grpcsecured

import (
	"crypto/tls"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// NewTLSServer creates a new grpc server secured with TLS.
func NewTLSServer(cert, key string) (*grpc.Server, error) {
	certificate, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, err
	}
	opts := []grpc.ServerOption{
		grpc.Creds(credentials.NewServerTLSFromCert(&certificate)),
	}
	return grpc.NewServer(opts...), nil
}

// NewClientOptions is a helper function creating TLS dial options.
func NewTLSClientOptions(caCert, serverNameOverride string) ([]grpc.DialOption, error) {
	creds, err := credentials.NewClientTLSFromFile(caCert, serverNameOverride)
	if err != nil {
		return nil, err
	}
	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
	}

	return opts, nil
}
