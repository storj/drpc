# package drpcconn

`import "storj.io/drpc/drpcconn"`

Package drpcconn creates a drpc client connection from a transport.

## Usage

#### type Conn

```go
type Conn struct {
}
```

Conn is a drpc client connection.

#### func  New

```go
func New(tr drpc.Transport) *Conn
```
New returns a conn that uses the transport for reads and writes.

#### func  NewWithOptions

```go
func NewWithOptions(tr drpc.Transport, opts Options) *Conn
```
NewWithOptions returns a conn that uses the transport for reads and writes. The
Options control details of how the conn operates.

#### func (*Conn) Close

```go
func (c *Conn) Close() (err error)
```
Close closes the connection.

#### func (*Conn) Closed

```go
func (c *Conn) Closed() <-chan struct{}
```
Closed returns a channel that is closed once the connection is closed.

#### func (*Conn) Invoke

```go
func (c *Conn) Invoke(ctx context.Context, rpc string, enc drpc.Encoding, in, out drpc.Message) (err error)
```
Invoke issues the rpc on the transport serializing in, waits for a response, and
deserializes it into out. Only one Invoke or Stream may be open at a time.

#### func (*Conn) NewStream

```go
func (c *Conn) NewStream(ctx context.Context, rpc string, enc drpc.Encoding) (_ drpc.Stream, err error)
```
NewStream begins a streaming rpc on the connection. Only one Invoke or Stream
may be open at a time.

#### func (*Conn) Stats

```go
func (c *Conn) Stats() map[string]drpcstats.Stats
```
Stats returns the collected stats grouped by rpc.

#### func (*Conn) Transport

```go
func (c *Conn) Transport() drpc.Transport
```
Transport returns the transport the conn is using.

#### func (*Conn) Unblocked

```go
func (c *Conn) Unblocked() <-chan struct{}
```
Unblocked returns a channel that is closed once the connection is no longer
blocked by a previously canceled Invoke or NewStream call. It should not be
called concurrently with Invoke or NewStream.

#### type Options

```go
type Options struct {
	// Manager controls the options we pass to the manager of this conn.
	Manager drpcmanager.Options

	// CollectStats controls whether the server should collect stats on the
	// rpcs it creates.
	CollectStats bool
}
```

Options controls configuration settings for a conn.
