# package drpcstream

`import "storj.io/drpc/drpcstream"`

package drpcstream sends protobufs using the dprc wire protocol.

## Usage

#### type Options

```go
type Options struct {
	// SplitSize controls the default size we split packets into frames.
	SplitSize int
}
```

Options controls configuration settings for a stream.

#### type Stream

```go
type Stream struct {
}
```


#### func  New

```go
func New(ctx context.Context, sid uint64, wr *drpcwire.Writer) *Stream
```

#### func  NewWithOptions

```go
func NewWithOptions(ctx context.Context, sid uint64, wr *drpcwire.Writer, opts Options) *Stream
```

#### func (*Stream) Cancel

```go
func (s *Stream) Cancel(err error)
```
Cancel transitions the stream into a state where all writes to the transport
will return the provided error, and terminates the stream.

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

#### func (*Stream) Finished

```go
func (s *Stream) Finished() bool
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

#### func (*Stream) SendError

```go
func (s *Stream) SendError(serr error) error
```

#### func (*Stream) Terminated

```go
func (s *Stream) Terminated() <-chan struct{}
```
