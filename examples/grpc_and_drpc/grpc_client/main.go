// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"time"

	"google.golang.org/grpc"

	"storj.io/drpc/examples/grpc_and_drpc/pb"
)

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	// dial the grpc server (without TLS or a connection header)
	conn, err := grpc.Dial("localhost:8080", grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		return err
	}
	defer conn.Close()

	// make a grpc proto-specific client
	client := pb.NewCookieMonsterClient(conn)

	// set a deadline for the operation
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()

	// run the RPC
	crumbs, err := client.EatCookie(ctx, &pb.Cookie{
		Type: pb.Cookie_Oatmeal,
	})
	if err != nil {
		return err
	}

	// check the results
	_, err = fmt.Println(crumbs.Cookie.Type.String())
	return err
}
