// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.3.0
// - protoc             v4.25.1
// source: webhooks.proto

package protobufcompiled

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

const (
	WebhooksAPI_Alive_FullMethodName    = "/computantis.WebhooksAPI/Alive"
	WebhooksAPI_Webhooks_FullMethodName = "/computantis.WebhooksAPI/Webhooks"
)

// WebhooksAPIClient is the client API for WebhooksAPI service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type WebhooksAPIClient interface {
	Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error)
	Webhooks(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*emptypb.Empty, error)
}

type webhooksAPIClient struct {
	cc grpc.ClientConnInterface
}

func NewWebhooksAPIClient(cc grpc.ClientConnInterface) WebhooksAPIClient {
	return &webhooksAPIClient{cc}
}

func (c *webhooksAPIClient) Alive(ctx context.Context, in *emptypb.Empty, opts ...grpc.CallOption) (*AliveData, error) {
	out := new(AliveData)
	err := c.cc.Invoke(ctx, WebhooksAPI_Alive_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *webhooksAPIClient) Webhooks(ctx context.Context, in *SignedHash, opts ...grpc.CallOption) (*emptypb.Empty, error) {
	out := new(emptypb.Empty)
	err := c.cc.Invoke(ctx, WebhooksAPI_Webhooks_FullMethodName, in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// WebhooksAPIServer is the server API for WebhooksAPI service.
// All implementations must embed UnimplementedWebhooksAPIServer
// for forward compatibility
type WebhooksAPIServer interface {
	Alive(context.Context, *emptypb.Empty) (*AliveData, error)
	Webhooks(context.Context, *SignedHash) (*emptypb.Empty, error)
	mustEmbedUnimplementedWebhooksAPIServer()
}

// UnimplementedWebhooksAPIServer must be embedded to have forward compatible implementations.
type UnimplementedWebhooksAPIServer struct {
}

func (UnimplementedWebhooksAPIServer) Alive(context.Context, *emptypb.Empty) (*AliveData, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Alive not implemented")
}
func (UnimplementedWebhooksAPIServer) Webhooks(context.Context, *SignedHash) (*emptypb.Empty, error) {
	return nil, status.Errorf(codes.Unimplemented, "method Webhooks not implemented")
}
func (UnimplementedWebhooksAPIServer) mustEmbedUnimplementedWebhooksAPIServer() {}

// UnsafeWebhooksAPIServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to WebhooksAPIServer will
// result in compilation errors.
type UnsafeWebhooksAPIServer interface {
	mustEmbedUnimplementedWebhooksAPIServer()
}

func RegisterWebhooksAPIServer(s grpc.ServiceRegistrar, srv WebhooksAPIServer) {
	s.RegisterService(&WebhooksAPI_ServiceDesc, srv)
}

func _WebhooksAPI_Alive_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(emptypb.Empty)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebhooksAPIServer).Alive(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WebhooksAPI_Alive_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebhooksAPIServer).Alive(ctx, req.(*emptypb.Empty))
	}
	return interceptor(ctx, in, info, handler)
}

func _WebhooksAPI_Webhooks_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(SignedHash)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(WebhooksAPIServer).Webhooks(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: WebhooksAPI_Webhooks_FullMethodName,
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(WebhooksAPIServer).Webhooks(ctx, req.(*SignedHash))
	}
	return interceptor(ctx, in, info, handler)
}

// WebhooksAPI_ServiceDesc is the grpc.ServiceDesc for WebhooksAPI service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var WebhooksAPI_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "computantis.WebhooksAPI",
	HandlerType: (*WebhooksAPIServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "Alive",
			Handler:    _WebhooksAPI_Alive_Handler,
		},
		{
			MethodName: "Webhooks",
			Handler:    _WebhooksAPI_Webhooks_Handler,
		},
	},
	Streams:  []grpc.StreamDesc{},
	Metadata: "webhooks.proto",
}
