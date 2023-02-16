// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcserver

import (
	"context"
	"net"
	"time"

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

	// Log is called when errors happen that can not be returned up, like
	// temporary network errors when accepting connections, or errors
	// handling individual clients. It is not called if nil.
	Log func(error)
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

var temporarySleep = 500 * time.Millisecond

// Serve listens for connections on the listener and serves the drpc request
// on new connections.
func (s *Server) Serve(ctx context.Context, lis net.Listener) (err error) {
	tracker := drpcctx.NewTracker(ctx)
	defer tracker.Wait()
	defer tracker.Cancel()

	tracker.Run(func(ctx context.Context) {
		<-ctx.Done()
		_ = lis.Close()
	})

	for {
		conn, err := lis.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}

			if isTemporary(err) {
				if s.opts.Log != nil {
					s.opts.Log(err)
				}

				t := time.NewTimer(temporarySleep)
				select {
				case <-t.C:
				case <-ctx.Done():
					t.Stop()
					return nil
				}

				continue
			}

			return errs.Wrap(err)
		}

		// TODO(jeff): connection limits?
		tracker.Run(func(ctx context.Context) {
			err := s.ServeOne(ctx, conn)
			if err != nil && s.opts.Log != nil {
				s.opts.Log(err)
			}
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
