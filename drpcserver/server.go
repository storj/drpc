// Copyright (C) 2019 Storj Labs, Inc.
// See LICENSE for copying information.

package drpcserver

import (
	"context"
	"fmt"
	"net"
	"reflect"

	"github.com/gogo/protobuf/proto"
	"github.com/zeebo/errs"

	"storj.io/drpc"
	"storj.io/drpc/drpcctx"
	"storj.io/drpc/drpcmanager"
	"storj.io/drpc/drpcstream"
	"storj.io/drpc/drpcwire"
)

type Server struct {
	rpcs map[string]rpcData
}

func New() *Server {
	return &Server{
		rpcs: make(map[string]rpcData),
	}
}

func panicf(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}

var (
	streamType  = reflect.TypeOf((*drpc.Stream)(nil)).Elem()
	messageType = reflect.TypeOf((*drpc.Message)(nil)).Elem()
)

type rpcData struct {
	srv     interface{}
	handler drpc.Handler
	in1     reflect.Type
	in2     reflect.Type
}

func (s *Server) Register(srv interface{}, desc drpc.Description) {
	n := desc.NumMethods()
	for i := 0; i < n; i++ {
		rpc, handler, method, ok := desc.Method(i)
		if !ok {
			panicf("description returned not ok for method %d", i)
		}
		s.registerOne(srv, rpc, handler, method)
	}
}

func (s *Server) registerOne(srv interface{}, rpc string, handler drpc.Handler, method interface{}) {
	if _, ok := s.rpcs[rpc]; ok {
		panicf("rpc already registered for %q", rpc)
	}
	data := rpcData{srv: srv, handler: handler}

	switch mt := reflect.TypeOf(method); {
	// unitary input, unitary output
	case mt.NumOut() == 2:
		data.in1 = mt.In(2)
		if !data.in1.Implements(messageType) {
			panicf("input argument not a drpc message: %v", data.in1)
		}

	// unitary input, stream output
	case mt.NumIn() == 3:
		data.in1 = mt.In(1)
		if !data.in1.Implements(messageType) {
			panicf("input argument not a drpc message: %v", data.in1)
		}
		data.in2 = streamType

	// stream input
	case mt.NumIn() == 2:
		data.in1 = streamType

	// code gen bug?
	default:
		panicf("unknown method type: %v", mt)
	}

	s.rpcs[rpc] = data
}

func (s *Server) ServeOne(ctx context.Context, tr drpc.Transport) (err error) {
	tracker := drpcctx.NewTracker(ctx)
	defer tracker.Cancel()

	man := drpcmanager.New(tr, s)

	errc := make(chan error, 1)
	tracker.Run(func(ctx context.Context) {
		<-ctx.Done()
		errc <- man.Close()
	})

	<-man.DoneSig().Signal()
	tracker.Cancel()
	tracker.Wait()

	var eg errs.Group
	eg.Add(<-errc)
	eg.Add(man.DoneSig().Err())
	return errs.Wrap(eg.Err())
}

func (s *Server) Serve(ctx context.Context, lis net.Listener) error {
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

func (s *Server) HandleRPC(stream *drpcstream.Stream, rpc string) {
	defer stream.CancelContext()

	err := s.doHandle(stream, rpc)
	if err != nil {
		_ = stream.SendError(err)
	}
	_ = stream.CloseSend()
}

func (s *Server) doHandle(stream *drpcstream.Stream, rpc string) error {
	data, ok := s.rpcs[rpc]
	if !ok {
		return drpc.ProtocolError.New("unknown rpc: %q", rpc)
	}

	in := interface{}(stream)
	if data.in1 != streamType {
		msg := reflect.New(data.in1.Elem()).Interface().(drpc.Message)
		if err := stream.MsgRecv(msg); err != nil {
			return errs.Wrap(err)
		}
		in = msg
	}

	out, err := data.handler(data.srv, stream.Context(), in, stream)
	switch {
	case err != nil:
		return errs.Wrap(err)
	case out != nil:
		data, err := proto.Marshal(out)
		if err != nil {
			return errs.Wrap(err)
		}
		return stream.RawWrite(drpcwire.Kind_Message, data)
	default:
		return nil
	}
}
