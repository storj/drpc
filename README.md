# package drpc

`import "storj.io/drpc"`

drpc is a light replacement for [gRPC](https://grpc.io "An open-source universal RPC framework").

## Usage

```go
var (
	Error         = errs.Class("drpc")
	InternalError = errs.Class("internal error")
	ProtocolError = errs.Class("protocol error")
)
```

#### type Conn

```go
type Conn interface {
	Close() error
	Transport() Transport

	Invoke(ctx context.Context, rpc string, in, out Message) error
	NewStream(ctx context.Context, rpc string) (Stream, error)
}
```


#### type Description

```go
type Description interface {
	NumMethods() int
	Method(n int) (rpc string, handler Handler, method interface{}, ok bool)
}
```


#### type Handler

```go
type Handler = func(srv interface{}, ctx context.Context, in1, in2 interface{}) (out Message, err error)
```


#### type Message

```go
type Message interface {
	Reset()
	String() string
	ProtoMessage()
}
```


#### type Server

```go
type Server interface {
	Register(srv interface{}, desc Description)
}
```


#### type Stream

```go
type Stream interface {
	Context() context.Context

	MsgSend(msg Message) error
	MsgRecv(msg Message) error

	CloseSend() error
	Close() error
}
```


#### type Transport

```go
type Transport interface {
	io.Reader
	io.Writer
	io.Closer
}
```
