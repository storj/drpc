# package drpcserver

`import "storj.io/drpc/drpcserver"`

Package drpcserver allows one to execute registered rpcs.

## Usage

#### type Options

```go
type Options struct {
	// Manager controls the options we pass to the managers this server creates.
	Manager drpcmanager.Options
}
```

Options controls configuration settings for a server.

#### type Server

```go
type Server struct {
}
```

Server is an implementation of drpc.Server to serve drpc connections.

#### func  New

```go
func New(handler drpc.Handler) *Server
```
New constructs a new Server.

#### func  NewWithOptions

```go
func NewWithOptions(handler drpc.Handler, opts Options) *Server
```
NewWithOptions constructs a new Server using the provided options to tune how
the drpc connections are handled.

#### func (*Server) Serve

```go
func (s *Server) Serve(ctx context.Context, lis net.Listener) (err error)
```
Serve listens for connections on the listener and serves the drpc request on new
connections.

#### func (*Server) ServeOne

```go
func (s *Server) ServeOne(ctx context.Context, tr drpc.Transport) (err error)
```
ServeOne serves a single set of rpcs on the provided transport.
