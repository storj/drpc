// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package twirpcompat

import (
	"context"
	errors "errors"
	"net/http"
	"testing"

	"github.com/twitchtv/twirp"
	"github.com/zeebo/assert"
	"google.golang.org/protobuf/proto"
)

func TestCompat_Noop(t *testing.T) {
	for _, code := range []twirp.ErrorCode{
		"",
		twirp.Canceled,
		twirp.Unknown,
		twirp.InvalidArgument,
		twirp.DeadlineExceeded,
		twirp.NotFound,
		twirp.BadRoute,
		twirp.AlreadyExists,
		twirp.PermissionDenied,
		twirp.Unauthenticated,
		twirp.ResourceExhausted,
		twirp.FailedPrecondition,
		twirp.Aborted,
		twirp.OutOfRange,
		twirp.Unimplemented,
		twirp.Internal,
		twirp.Unavailable,
		twirp.DataLoss,
	} {
		t.Run(string(code), func(t *testing.T) {
			cs, srv := newServer()
			defer srv.Close()

			cs.noop = func(context.Context, *Empty) (*Empty, error) {
				if code != "" {
					return nil, twirp.NewError(code, "oopsie")
				}
				return new(Empty), nil
			}

			client := NewCompatServiceProtobufClient(srv.URL, http.DefaultClient)
			_, err := client.NoopMethod(context.Background(), new(Empty))

			if code == "" {
				assert.NoError(t, err)
			} else {
				var te twirp.Error
				assert.That(t, errors.As(err, &te))
				assert.Equal(t, code, te.Code())
			}
		})
	}
}

func TestCompat_Method(t *testing.T) {
	cs, srv := newServer()
	defer srv.Close()

	cs.method = func(ctx context.Context, req *Req) (*Resp, error) {
		switch req.V {
		case "":
			return &Resp{}, nil
		case "error":
			return nil, twirp.InvalidArgumentError("V", "some error")
		default:
			return &Resp{V: int32(len(req.V))}, nil
		}
	}
	client := NewCompatServiceProtobufClient(srv.URL, http.DefaultClient)

	t.Run("empty", func(t *testing.T) {
		resp, err := client.Method(context.Background(), new(Req))
		assert.NoError(t, err)
		assert.That(t, proto.Equal(resp, new(Resp)))
	})

	t.Run("data", func(t *testing.T) {
		resp, err := client.Method(context.Background(), &Req{V: "hello world"})
		assert.NoError(t, err)
		assert.That(t, proto.Equal(resp, &Resp{V: 11}))
	})

	t.Run("error", func(t *testing.T) {
		resp, err := client.Method(context.Background(), &Req{V: "error"})
		var te twirp.Error
		assert.That(t, errors.As(err, &te))
		assert.Equal(t, twirp.InvalidArgument, te.Code())
		assert.Nil(t, resp)
	})
}
