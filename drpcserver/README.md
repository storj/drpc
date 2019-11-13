# package drpcserver

`import "storj.io/drpc/drpcserver"`

package drpcserver allows one to execute registered rpcs.

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


#### func  New

```go
func New() *Server
```

#### func  NewWithOptions

```go
func NewWithOptions(opts Options) *Server
```

#### func (*Server) HandleRPC

```go
func (s *Server) HandleRPC(stream *drpcstream.Stream, rpc string) error
```

#### func (*Server) Register

```go
func (s *Server) Register(srv interface{}, desc drpc.Description)
```

#### func (*Server) Serve

```go
func (s *Server) Serve(ctx context.Context, lis net.Listener) error
```

#### func (*Server) ServeOne

```go
func (s *Server) ServeOne(ctx context.Context, tr drpc.Transport) (err error)
```
