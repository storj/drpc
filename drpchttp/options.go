// Copyright (C) 2021 Storj Labs, Inc.
// See LICENSE for copying information.

package drpchttp

import (
	"net/http"

	"storj.io/drpc"
)

// Option configures some aspect of the handler.
type Option struct{ apply func(*options) }

type options struct {
	protocols map[string]Protocol
}

// Protocol is used by the handler to create drpc.Streams from incoming
// requests and format responses.
type Protocol interface {
	// NewStream takes an incoming request and response writer and returns
	// a drpc.Stream that should be used for the RPC.
	NewStream(rw http.ResponseWriter, req *http.Request) Stream
}

// Stream wraps a drpc.Stream type with a Finish method that knows how to
// send and format the error/response to an http request.
type Stream interface {
	drpc.Stream

	// Finish is passed the possibly-nil error that was generated handling
	// the RPC and is expected to write any error reporting or otherwise
	// finalize the request.
	Finish(err error)
}

// WithProtocol associates the given Protocol with some content type. The
// match is exact, with the special case that the content type "*" is the
// fallback Protocol used when nothing matches.
func WithProtocol(contentType string, pr Protocol) Option {
	return Option{apply: func(opts *options) {
		opts.protocols[contentType] = pr
	}}
}

func defaultProtocols() map[string]Protocol {
	return map[string]Protocol{
		"*": twirpProtocol{
			ct:        "application/proto",
			marshal:   protoMarshal,
			unmarshal: protoUnmarshal,
		},

		"application/proto": twirpProtocol{
			ct:        "application/proto",
			marshal:   protoMarshal,
			unmarshal: protoUnmarshal,
		},

		"application/json": twirpProtocol{
			ct:        "application/json",
			marshal:   JSONMarshal,
			unmarshal: JSONUnmarshal,
		},

		"application/grpc-web+proto": grpcWebProtocol{
			ct:        "application/grpc-web+proto",
			read:      grpcRead,
			write:     normalWrite,
			marshal:   protoMarshal,
			unmarshal: protoUnmarshal,
		},

		"application/grpc-web+json": grpcWebProtocol{
			ct:        "application/grpc-web+json",
			read:      grpcRead,
			write:     normalWrite,
			marshal:   JSONMarshal,
			unmarshal: JSONUnmarshal,
		},

		"application/grpc-web-text+proto": grpcWebProtocol{
			ct:        "application/grpc-web-text+proto",
			read:      base64Read(grpcRead),
			write:     base64Write(normalWrite),
			marshal:   protoMarshal,
			unmarshal: protoUnmarshal,
		},

		"application/grpc-web-text+json": grpcWebProtocol{
			ct:        "application/grpc-web-text+json",
			read:      base64Read(grpcRead),
			write:     base64Write(normalWrite),
			marshal:   JSONMarshal,
			unmarshal: JSONUnmarshal,
		},
	}
}
