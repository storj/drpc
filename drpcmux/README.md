# package drpcmux

`import "storj.io/drpc/drpcmux"`

Package drpcmux is a handler to dispatch rpcs to implementations.

## Usage

#### type Mux

```go
type Mux struct {
}
```

Mux is an implementation of Handler to serve drpc connections to the appropriate
Receivers registered by Descriptions.

#### func  New

```go
func New() *Mux
```
New constructs a new Mux.

#### func (*Mux) HandleRPC

```go
func (m *Mux) HandleRPC(stream drpc.Stream, rpc string) (err error)
```
HandleRPC handles the rpc that has been requested by the stream.

#### func (*Mux) Register

```go
func (m *Mux) Register(srv interface{}, desc drpc.Description) error
```
Register associates the RPCs described by the description in the server. It
returns an error if there was a problem registering it.

#### func (*Mux) ServeHTTP

```go
func (m *Mux) ServeHTTP(w http.ResponseWriter, req *http.Request)
```
ServeHTTP handles unitary RPCs over an http request. The RPCs are hosted at a
path based on their name, like `/service.Server/Method` and accept the request
protobuf in json. The response will either be of the form

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
