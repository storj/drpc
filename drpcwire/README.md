# package drpcwire

`import "storj.io/drpc/drpcwire"`

package drpcwire provides low level helpers for the drpc wire protocol.

## Usage

#### func  AppendFrame

```go
func AppendFrame(buf []byte, fr Frame) []byte
```

#### func  AppendVarint

```go
func AppendVarint(buf []byte, x uint64) []byte
```
AppendVarint appends the varint encoding of x to the buffer and returns it.

#### func  MarshalError

```go
func MarshalError(err error) []byte
```
MarshalError returns a byte form of the error with any error code incorporated.

#### func  ReadVarint

```go
func ReadVarint(buf []byte) (rem []byte, out uint64, ok bool, err error)
```
ReadVarint reads a varint encoded integer from the front of buf, returning the
remaining bytes, the value, and if there was a success. if ok is false, the
returned buffer is the same as the passed in buffer.

#### func  SplitFrame

```go
func SplitFrame(data []byte, atEOF bool) (int, []byte, error)
```

#### func  SplitN

```go
func SplitN(pkt Packet, n int, cb func(fr Frame) error) error
```

#### func  UnmarshalError

```go
func UnmarshalError(data []byte) error
```
UnmarshalError unmarshals the marshaled error to one with a code.

#### type Frame

```go
type Frame struct {
	Data []byte
	ID   ID
	Kind Kind
	Done bool
}
```


#### func  ParseFrame

```go
func ParseFrame(buf []byte) (rem []byte, fr Frame, ok bool, err error)
```

#### type ID

```go
type ID struct {
	Stream  uint64
	Message uint64
}
```


#### func (ID) Less

```go
func (i ID) Less(j ID) bool
```

#### func (ID) String

```go
func (i ID) String() string
```

#### type Kind

```go
type Kind uint8
```


```go
const (
	Kind_Reserved Kind = 0

	Kind_Invoke    Kind = 1 // body is rpc name
	Kind_Message   Kind = 2 // body is message data
	Kind_Error     Kind = 3 // body is error data
	Kind_Close     Kind = 5 // body must be empty
	Kind_CloseSend Kind = 6 // body must be empty

	Kind_Largest Kind = 7
)
```

#### func (Kind) String

```go
func (i Kind) String() string
```

#### type Packet

```go
type Packet struct {
	Data []byte
	ID   ID
	Kind Kind
}
```


#### func (Packet) String

```go
func (p Packet) String() string
```

#### type Reader

```go
type Reader struct {
}
```


#### func  NewReader

```go
func NewReader(r io.Reader) *Reader
```

#### func (*Reader) ReadPacket

```go
func (s *Reader) ReadPacket() (pkt Packet, err error)
```

#### type Writer

```go
type Writer struct {
}
```


#### func  NewWriter

```go
func NewWriter(w io.Writer, size int) *Writer
```

#### func (*Writer) Flush

```go
func (b *Writer) Flush() (err error)
```

#### func (*Writer) WriteFrame

```go
func (b *Writer) WriteFrame(fr Frame) (err error)
```

#### func (*Writer) WritePacket

```go
func (b *Writer) WritePacket(pkt Packet) (err error)
```
