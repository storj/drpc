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
func JSONUnmarshal(msg drpc.Message, enc drpc.Encoding, buf []byte) error
```
JSONUnmarshal looks for a JSONUnmarshal method on the encoding and calls that if
it exists. Otherwise, it JSON unmarshals the buf before doing a normal message
unmarshal.

#### func  New

```go
func New(handler drpc.Handler) http.Handler
```
New returns a net/http.Handler that dispatches to the passed in drpc.Handler.

The returned value handles unitary RPCs over an http request. The RPCs are
hosted at a path based on their name, like `/service.Server/Method` and accept
the request message in JSON. The response will either be of the form

    {
      "status": "ok",
      "response": ...
    }

if the request was successful, or

    {
      "status": "error",
      "error": ...,
      "code": ...
    }

where error is a textual description of the error, and code is the numeric code
that was set with drpcerr, if any.

Metadata can be attached by adding the "X-Drpc-Metadata" header to the request
possibly multiple times. The format is

    X-Drpc-Metadata: percentEncode(key)=percentEncode(value)

where percentEncode is the encoding used for query strings. Only the '%' and '='
characters are necessary to be escaped.
