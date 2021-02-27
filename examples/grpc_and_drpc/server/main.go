// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"net"

	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"storj.io/drpc/drpcmigrate"
	"storj.io/drpc/drpcmux"
	"storj.io/drpc/drpcserver"

	"storj.io/drpc/examples/grpc_and_drpc/pb"
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

	// listen on a tcp socket
	lis, err := net.Listen("tcp", ":8080")
	if err != nil {
		return err
	}

	// create a listen mux that evalutes enough bytes to recognize the DRPC header
	lisMux := drpcmigrate.NewListenMux(lis, len(drpcmigrate.DRPCHeader))

	// we're going to run the different protocol servers in parallel, so
	// make an errgroup
	var group errgroup.Group

	// grpc handling
	group.Go(func() error {
		// create a grpc server (without TLS)
		s := grpc.NewServer()

		// register the proto-specific methods on the server
		pb.RegisterCookieMonsterServer(s, cookieMonster)

		// run the server
		return s.Serve(lisMux.Default())
	})

	// drpc handling
	group.Go(func() error {
		// create a drpc RPC mux
		m := drpcmux.New()

		// register the proto-specific methods on the mux
		err := pb.DRPCRegisterCookieMonster(m, cookieMonster)
		if err != nil {
			return err
		}

		// create a drpc server
		s := drpcserver.New(m)

		// grap the listen mux route for the DRPC Header
		drpcLis := lisMux.Route(drpcmigrate.DRPCHeader)

		// run the server
		// N.B.: if you want TLS, you need to wrap the drpcLis net.Listener
		// with TLS before passing to Serve here.
		return s.Serve(ctx, drpcLis)
	})

	// run the listen mux
	group.Go(func() error {
		return lisMux.Run(ctx)
	})

	// wait
	return group.Wait()
}
