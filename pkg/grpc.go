package xtremepkg

import (
	"context"
	"google.golang.org/grpc"
	"log"
	"net"
	"time"
)

type GRPCClient struct {
	Ctx    context.Context
	Conn   *grpc.ClientConn
	Cancel context.CancelFunc
}

func (client *GRPCClient) RPCDialClient(host string, timeout ...time.Duration) context.CancelFunc {
	dialTimeout := RPCDialTimeout
	if len(timeout) > 0 {
		dialTimeout = timeout[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), dialTimeout)

	conn, err := grpc.DialContext(ctx, host, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Panicf("Did not connect to %s: %v", host, err)
	}

	client.Ctx = ctx
	client.Conn = conn
	client.Cancel = cancel

	cleanup := func() {
		client.Cancel()
		client.Conn.Close()
	}

	return cleanup
}

type GRPCServer struct {
	listener net.Listener
	server   *grpc.Server
}

type GRPCServerRegister interface {
	Register(*grpc.Server)
}

func (srv *GRPCServer) NewServer(address string) *GRPCServer {
	lis, err := net.Listen("tcp", address)
	if err != nil {
		log.Panicf("Failed to listen: %v", err)
	}
	s := grpc.NewServer()

	srv.listener = lis
	srv.server = s

	return srv
}

func (srv *GRPCServer) Register(interfaces ...GRPCServerRegister) *GRPCServer {
	for _, itf := range interfaces {
		itf.Register(srv.server)
	}

	return srv
}

func (srv *GRPCServer) Serve() error {
	return srv.server.Serve(srv.listener)
}
