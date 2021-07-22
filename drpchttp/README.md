# package drpchttp

`import "storj.io/drpc/drpchttp"`

Package drpchttp implements a net/http handler for unitary RPCs.

## Usage

#### func  Context

```go
func Context(req *http.Request) (context.Context, error)
```
Context returns the context.Context from the http.Request with any metadata sent
using the X-Drpc-Metadata header set as values.

#### func  JSONMarshal

```go
func JSONMarshal(msg drpc.Message, enc drpc.Encoding) ([]byte, error)
```
JSONMarshal looks for a JSONMarshal method on the encoding and calls that if it
exists. Otherwise, it does a normal message marshal before doing a JSON marshal.

#### func  JSONUnmarshal

```go
func JSONUnmarshal(buf []byte, msg drpc.Message, enc drpc.Encoding) error
```
JSONUnmarshal looks for a JSONUnmarshal method on the encoding and calls that if
it exists. Otherwise, it JSON unmarshals the buf before doing a normal message
unmarshal.

#### func  New

```go
func New(handler drpc.Handler) http.Handler
```
New returns a net/http.Handler that dispatches to the passed in drpc.Handler.
See NewWithOptions for more details.

#### func  NewWithOptions

```go
func NewWithOptions(handler drpc.Handler, os ...Option) http.Handler
```
NewWithOptions returns a net/http.Handler that dispatches to the passed in
drpc.Handler. The RPCs are hosted at a path based on their name, like
`/service.Server/Method`.

Metadata can be attached by adding the "X-Drpc-Metadata" header to the request
possibly multiple times. The format is

    X-Drpc-Metadata: percentEncode(key)=percentEncode(value)

where percentEncode is the encoding used for query strings. Only the '%' and '='
characters are necessary to be escaped.

The specific protocol for the request and response used is chosen by the
request's Content-Type. By default the content types "application/json" and
"application/protobuf" correspond to unitary-only RPCs that respond with the
same Content-Type as the incoming request upon success. Upon failure, the
response code will not be 200 OK, the response content type will always be
"application/json", and the body will look something like

    {
      "code": "...",
      "msg": "..."
    }

where msg is a textual description of the error, and code is a short string that
describes the kind of error that happened, if possible. If nothing could be
detected, then the string "unknown" is used for the code.

The content types "application/grpc-web+proto", "application/grpc-web+json",
"application/grpc-web-text+proto", and "application/grpc-web-text+json" will
serve unitary and server-streaming RPCs using the protocol described by the
grpc-web project. Informally, messages are framed with a 5 byte header where the
first byte is some flags, and the second through fourth are the message length
in big endian. Response codes and status messages are sent as HTTP Trailers. The
"-text" series of content types mean that the whole request and response bodies
are base64 encoded.

#### type Option

```go
type Option struct {
}
```

Option configures some aspect of the handler.

#### func  WithProtocol

```go
func WithProtocol(contentType string, pr Protocol) Option
```
WithProtocol associates the given Protocol with some content type. The match is
exact, with the special case that the content type "*" is the fallback Protocol
used when nothing matches.

#### type Protocol

```go
type Protocol interface {
	// NewStream takes an incoming request and response writer and returns
	// a drpc.Stream that should be used for the RPC.
	NewStream(rw http.ResponseWriter, req *http.Request) Stream
}
```

Protocol is used by the handler to create drpc.Streams from incoming requests
and format responses.

#### type Stream

```go
type Stream interface {
	drpc.Stream

	// Finish is passed the possibly-nil error that was generated handling
	// the RPC and is expected to write any error reporting or otherwise
	// finalize the request.
	Finish(err error)
}
```

Stream wraps a drpc.Stream type with a Finish method that knows how to send and
format the error/response to an http request.
