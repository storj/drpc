# package drpcstream

`import "storj.io/drpc/drpcstream"`

package drpcstream sends protobufs using the dprc wire protocol.

## Usage

#### type Stream

```go
type Stream struct {
}
```


#### func  New

```go
func New(ctx context.Context, sid uint64, wr *drpcwire.Writer) *Stream
```

#### func (*Stream) CancelContext

```go
func (s *Stream) CancelContext()
```

#### func (*Stream) Close

```go
func (s *Stream) Close() error
```

#### func (*Stream) CloseQueue

```go
func (s *Stream) CloseQueue()
```

#### func (*Stream) CloseSend

```go
func (s *Stream) CloseSend() error
```

#### func (*Stream) Context

```go
func (s *Stream) Context() context.Context
```

#### func (*Stream) DoneSig

```go
func (s *Stream) DoneSig() *drpcsignal.Signal
```

#### func (*Stream) ID

```go
func (s *Stream) ID() uint64
```

#### func (*Stream) MsgRecv

```go
func (s *Stream) MsgRecv(msg drpc.Message) error
```

#### func (*Stream) MsgSend

```go
func (s *Stream) MsgSend(msg drpc.Message) error
```

#### func (*Stream) Queue

```go
func (s *Stream) Queue() chan drpcwire.Packet
```

#### func (*Stream) QueueClosed

```go
func (s *Stream) QueueClosed() bool
```

#### func (*Stream) RawFlush

```go
func (s *Stream) RawFlush() (err error)
```

#### func (*Stream) RawRecv

```go
func (s *Stream) RawRecv() ([]byte, error)
```

#### func (*Stream) RawWrite

```go
func (s *Stream) RawWrite(kind drpcwire.Kind, data []byte) error
```

#### func (*Stream) RemoteErrSig

```go
func (s *Stream) RemoteErrSig() *drpcsignal.Signal
```

#### func (*Stream) SendCancel

```go
func (s *Stream) SendCancel() error
```

#### func (*Stream) SendError

```go
func (s *Stream) SendError(err error) error
```

#### func (*Stream) SendSig

```go
func (s *Stream) SendSig() *drpcsignal.Signal
```
