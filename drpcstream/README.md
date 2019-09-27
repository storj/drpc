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

#### func (*Stream) Close

```go
func (s *Stream) Close() error
```

#### func (*Stream) CloseSend

```go
func (s *Stream) CloseSend() error
```

#### func (*Stream) Context

```go
func (s *Stream) Context() context.Context
```

#### func (*Stream) HandlePacket

```go
func (s *Stream) HandlePacket(pkt drpcwire.Packet) (error, bool)
```
HandlePacket advances the stream state machine by inspecting the packet. It
returns any major errors that should terminate the transport the stream is
operating on as well as a boolean indicating if the stream expects more packets.

#### func (*Stream) MsgRecv

```go
func (s *Stream) MsgRecv(msg drpc.Message) error
```

#### func (*Stream) MsgSend

```go
func (s *Stream) MsgSend(msg drpc.Message) error
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

#### func (*Stream) SendCancel

```go
func (s *Stream) SendCancel(err error) error
```

#### func (*Stream) SendError

```go
func (s *Stream) SendError(err error) error
```

#### func (*Stream) Terminated

```go
func (s *Stream) Terminated() <-chan struct{}
```
