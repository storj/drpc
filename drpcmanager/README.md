# package drpcmanager

`import "storj.io/drpc/drpcmanager"`

package drpcmanager reads packets from a transport to make streams.

## Usage

#### type Manager

```go
type Manager struct {
}
```


#### func  New

```go
func New(tr drpc.Transport) *Manager
```

#### func  NewWithOptions

```go
func NewWithOptions(tr drpc.Transport, opts Options) *Manager
```

#### func (*Manager) Close

```go
func (m *Manager) Close() error
```

#### func (*Manager) Closed

```go
func (m *Manager) Closed() bool
```

#### func (*Manager) NewClientStream

```go
func (m *Manager) NewClientStream(ctx context.Context) (stream *drpcstream.Stream, err error)
```

#### func (*Manager) NewServerStream

```go
func (m *Manager) NewServerStream(ctx context.Context) (stream *drpcstream.Stream, rpc string, err error)
```

#### type Options

```go
type Options struct {
	// WriterBufferSize controls the size of the buffer that we will fill before
	// flushing. Normal writes to streams typically issue a flush explicitly.
	WriterBufferSize int

	// Stream are passed to any streams the manager creates.
	Stream drpcstream.Options
}
```

Options controls configuration settings for a manager.
