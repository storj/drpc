// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package main

import (
	"context"
	"fmt"
	"time"

	"storj.io/drpc/drpcconn"
	"storj.io/drpc/drpcmigrate"

	"storj.io/drpc/examples/drpc_and_http/pb"
)

func main() {
	err := Main(context.Background())
	if err != nil {
		panic(err)
	}
}

func Main(ctx context.Context) error {
	// dial the drpc server with the drpc connection header
	rawconn, err := drpcmigrate.DialWithHeader(ctx, "tcp", "localhost:8080", drpcmigrate.DRPCHeader)
	if err != nil {
		return err
	}
	// N.B.: If you want TLS, you need to wrap the net.Conn with TLS before
	// making a DRPC conn.

	// convert the net.Conn to a drpc.Conn
	conn := drpcconn.New(rawconn)
	defer conn.Close()

	// make a drpc proto-specific client
	client := pb.NewDRPCCookieMonsterClient(conn)

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
