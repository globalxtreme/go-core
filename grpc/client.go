package xtremegrpc

import (
	"context"
	"github.com/globalxtreme/go-core/v2/pkg"
	"google.golang.org/grpc"
	"log"
	"time"
)

type GRPCClient struct {
	Ctx    context.Context
	Conn   *grpc.ClientConn
	Cancel context.CancelFunc
}

func (client *GRPCClient) RPCDialClient(host string, timeout ...time.Duration) context.CancelFunc {
	dialTimeout := xtremepkg.RPCDialTimeout
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
