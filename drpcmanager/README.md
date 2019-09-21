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
func New(tr drpc.Transport, srv Server) *Manager
```

#### func (*Manager) Close

```go
func (m *Manager) Close() (err error)
```

#### func (*Manager) DoneSig

```go
func (m *Manager) DoneSig() *drpcsignal.Signal
```

#### func (*Manager) NewStream

```go
func (m *Manager) NewStream(ctx context.Context, sid uint64) (*drpcstream.Stream, error)
```

#### type Server

```go
type Server interface {
	HandleRPC(stream *drpcstream.Stream, rpc string)
}
```
