# package drpcconn

`import "storj.io/drpc/drpcconn"`

package drpcconn creates a drpc client connection from a transport.

## Usage

#### type Conn

```go
type Conn struct {
}
```


#### func  New

```go
func New(tr drpc.Transport) *Conn
```

#### func  NewWithOptions

```go
func NewWithOptions(tr drpc.Transport, opts Options) *Conn
```

#### func (*Conn) Close

```go
func (c *Conn) Close() (err error)
```

#### func (*Conn) Closed

```go
func (c *Conn) Closed() bool
```

#### func (*Conn) Invoke

```go
func (c *Conn) Invoke(ctx context.Context, rpc string, in, out drpc.Message) (err error)
```

#### func (*Conn) NewStream

```go
func (c *Conn) NewStream(ctx context.Context, rpc string) (_ drpc.Stream, err error)
```

#### func (*Conn) Transport

```go
func (c *Conn) Transport() drpc.Transport
```

#### type Options

```go
type Options struct {
	// Manager controls the options we pass to the manager of this conn.
	Manager drpcmanager.Options
}
```

Options controls configuration settings for a conn.
