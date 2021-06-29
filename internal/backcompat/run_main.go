// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package backcompat

import (
	"context"
	"errors"
	"log"
	"os"
)

// Main runs the service as either a client or server depending on os.Args.
func Main(ctx context.Context) {
	var err error
	switch os.Args[1] {
	case "server":
		err = runServer(ctx, os.Args[2])
	case "client":
		err = runClient(ctx, os.Args[2])
	default:
		err = errors.New("unknown mode")
	}
	if err != nil {
		log.Fatalf("%+v", err)
	} else if err = ctx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("%+v", err)
	}
}
