// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcserver

import (
	"context"
	"net"

	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpccache"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcmanager"
	"storj.io/drpc/drpcstream"
)

// Options controls configuration settings for a server.
type Options struct {
	// Manager controls the options we pass to the managers this server creates.
	Manager drpcmanager.Options
}

// Server is an implementation of drpc.Server to serve drpc connections.
type Server struct {
	opts    Options
	handler drpc.Handler
}

// New constructs a new Server.
func New(handler drpc.Handler) *Server {
	return NewWithOptions(handler, Options{})
}

// NewWithOptions constructs a new Server using the provided options to tune
// how the drpc connections are handled.
func NewWithOptions(handler drpc.Handler, opts Options) *Server {
	return &Server{
		opts:    opts,
		handler: handler,
	}
}

// ServeOne serves a single set of rpcs on the provided transport.
func (s *Server) ServeOne(ctx context.Context, tr drpc.Transport) (err error) {
	man := drpcmanager.NewWithOptions(tr, s.opts.Manager)
	defer func() { err = errs.Combine(err, man.Close()) }()

	cache := drpccache.New()
	defer cache.Clear()

	ctx = drpccache.WithContext(ctx, cache)

	for {
		stream, rpc, err := man.NewServerStream(ctx)
		if err != nil {
			return errs.Wrap(err)
		}
		if err := s.handleRPC(stream, rpc); err != nil {
			return errs.Wrap(err)
		}
	}
}

// Serve listens for connections on the listener and serves the drpc request
// on new connections.
func (s *Server) Serve(ctx context.Context, lis net.Listener) (err error) {
	tracker := drpcctx.NewTracker(ctx)
	defer tracker.Cancel()

	tracker.Run(func(ctx context.Context) {
		<-ctx.Done()
		_ = lis.Close()
	})

	for {
		conn, err := lis.Accept()
		if err != nil {
			// TODO(jeff): temporary errors?
			select {
			case <-ctx.Done():
				tracker.Wait()
				return nil
			default:
				tracker.Cancel()
				tracker.Wait()
				return errs.Wrap(err)
			}
		}

		// TODO(jeff): connection limits?
		tracker.Run(func(ctx context.Context) {
			// TODO(jeff): handle this error?
			_ = s.ServeOne(ctx, conn)
		})
	}
}

// handleRPC handles the rpc that has been requested by the stream.
func (s *Server) handleRPC(stream *drpcstream.Stream, rpc string) (err error) {
	err = s.handler.HandleRPC(stream, rpc)
	if err != nil {
		return errs.Wrap(stream.SendError(err))
	}
	return errs.Wrap(stream.CloseSend())
}
