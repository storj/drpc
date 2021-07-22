// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package twirpcompat

import (
	"context"
	"net/http/httptest"

	"github.com/zeebo/hmux"

	"storj.io/drpc/drpchttp"
	"storj.io/drpc/drpcmux"
)

type compatService struct {
	method func(context.Context, *Req) (*Resp, error)
	noop   func(context.Context, *Empty) (*Empty, error)
}

func (cs *compatService) Method(ctx context.Context, req *Req) (*Resp, error) {
	return cs.method(ctx, req)
}

func (cs *compatService) NoopMethod(ctx context.Context, req *Empty) (*Empty, error) {
	return cs.noop(ctx, req)
}

func newServer() (*compatService, *httptest.Server) {
	cs := new(compatService)
	mux := drpcmux.New()
	_ = DRPCRegisterCompatService(mux, cs)
	return cs, httptest.NewServer(hmux.Dir{"/twirp": drpchttp.New(mux)})
}
