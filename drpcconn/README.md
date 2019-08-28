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

#### func (*Conn) Close

```go
func (c *Conn) Close() (err error)
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
