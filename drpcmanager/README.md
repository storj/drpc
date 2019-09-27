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

#### func (*Manager) Close

```go
func (m *Manager) Close() error
```

#### func (*Manager) NewClientStream

```go
func (m *Manager) NewClientStream(ctx context.Context) (stream *drpcstream.Stream, err error)
```

#### func (*Manager) NewServerStream

```go
func (m *Manager) NewServerStream(ctx context.Context) (stream *drpcstream.Stream, rpc string, err error)
```
