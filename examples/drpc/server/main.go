// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"net"

	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"

	"storj.io/drpc/examples/drpc/pb"
)

type CookieMonsterServer struct {
	pb.DRPCCookieMonsterUnimplementedServer
	// struct fields
}

// EatCookie turns a cookie into crumbs.
func (s *CookieMonsterServer) EatCookie(ctx context.Context, cookie *pb.Cookie) (*pb.Crumbs, error) {
	return &pb.Crumbs{
		Cookie: cookie,
	}, nil
}

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	// create an RPC server
	cookieMonster := &CookieMonsterServer{}

	// create a drpc RPC mux
	m := drpcmux.New()

	// register the proto-specific methods on the mux
	err := pb.DRPCRegisterCookieMonster(m, cookieMonster)
	if err != nil {
		return err
	}

	// create a drpc server
	s := drpcserver.New(m)

	// listen on a tcp socket
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	// run the server
	// N.B.: if you want TLS, you need to wrap the net.Listener with
	// TLS before passing to Serve here.
	return s.Serve(ctx, lis)
}
