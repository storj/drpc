// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"net"

	"google.golang.org/grpc"

	"storj.io/drpc/examples/grpc/pb"
)

type CookieMonsterServer struct {
	pb.UnimplementedCookieMonsterServer
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

	// create a grpc server (without TLS)
	s := grpc.NewServer()

	// register the proto-specific methods on the server
	pb.RegisterCookieMonsterServer(s, cookieMonster)

	// listen on a tcp socket
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	// run the server
	return s.Serve(lis)
}
