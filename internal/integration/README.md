# package integration

`import "storj.io/drpc/internal/integration"`

Package integration holds integration tests for drpc.

## Usage

#### func  DRPCRegisterService

```go
func DRPCRegisterService(mux drpc.Mux, impl DRPCServiceServer) error
```

#### type DRPCServiceClient

```go
type DRPCServiceClient interface {
	DRPCConn() drpc.Conn

	Method1(ctx context.Context, in *In) (*Out, error)
	Method2(ctx context.Context) (DRPCService_Method2Client, error)
	Method3(ctx context.Context, in *In) (DRPCService_Method3Client, error)
	Method4(ctx context.Context) (DRPCService_Method4Client, error)
}
```


#### func  NewDRPCServiceClient

```go
func NewDRPCServiceClient(cc drpc.Conn) DRPCServiceClient
```

#### type DRPCServiceDescription

```go
type DRPCServiceDescription struct{}
```


#### func (DRPCServiceDescription) Method

```go
func (DRPCServiceDescription) Method(n int) (string, drpc.Receiver, interface{}, bool)
```

#### func (DRPCServiceDescription) NumMethods

```go
func (DRPCServiceDescription) NumMethods() int
```

#### type DRPCServiceServer

```go
type DRPCServiceServer interface {
	Method1(context.Context, *In) (*Out, error)
	Method2(DRPCService_Method2Stream) error
	Method3(*In, DRPCService_Method3Stream) error
	Method4(DRPCService_Method4Stream) error
}
```


#### type DRPCServiceUnimplementedServer

```go
type DRPCServiceUnimplementedServer struct{}
```


#### func (*DRPCServiceUnimplementedServer) Method1

```go
func (s *DRPCServiceUnimplementedServer) Method1(context.Context, *In) (*Out, error)
```

#### func (*DRPCServiceUnimplementedServer) Method2

```go
func (s *DRPCServiceUnimplementedServer) Method2(DRPCService_Method2Stream) error
```

#### func (*DRPCServiceUnimplementedServer) Method3

```go
func (s *DRPCServiceUnimplementedServer) Method3(*In, DRPCService_Method3Stream) error
```

#### func (*DRPCServiceUnimplementedServer) Method4

```go
func (s *DRPCServiceUnimplementedServer) Method4(DRPCService_Method4Stream) error
```

#### type DRPCService_Method1Stream

```go
type DRPCService_Method1Stream interface {
	drpc.Stream
	SendAndClose(*Out) error
}
```


#### type DRPCService_Method2Client

```go
type DRPCService_Method2Client interface {
	drpc.Stream
	Send(*In) error
	CloseAndRecv() (*Out, error)
}
```


#### type DRPCService_Method2Stream

```go
type DRPCService_Method2Stream interface {
	drpc.Stream
	SendAndClose(*Out) error
	Recv() (*In, error)
}
```


#### type DRPCService_Method3Client

```go
type DRPCService_Method3Client interface {
	drpc.Stream
	Recv() (*Out, error)
}
```


#### type DRPCService_Method3Stream

```go
type DRPCService_Method3Stream interface {
	drpc.Stream
	Send(*Out) error
}
```


#### type DRPCService_Method4Client

```go
type DRPCService_Method4Client interface {
	drpc.Stream
	Send(*In) error
	Recv() (*Out, error)
}
```


#### type DRPCService_Method4Stream

```go
type DRPCService_Method4Stream interface {
	drpc.Stream
	Send(*Out) error
	Recv() (*In, error)
}
```


#### type In

```go
type In struct {
	In                   int64    `protobuf:"varint,1,opt,name=in,proto3" json:"in,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}
```


#### func (*In) Descriptor

```go
func (*In) Descriptor() ([]byte, []int)
```

#### func (*In) GetIn

```go
func (m *In) GetIn() int64
```

#### func (*In) ProtoMessage

```go
func (*In) ProtoMessage()
```

#### func (*In) Reset

```go
func (m *In) Reset()
```

#### func (*In) String

```go
func (m *In) String() string
```

#### func (*In) XXX_DiscardUnknown

```go
func (m *In) XXX_DiscardUnknown()
```

#### func (*In) XXX_Marshal

```go
func (m *In) XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
```

#### func (*In) XXX_Merge

```go
func (m *In) XXX_Merge(src proto.Message)
```

#### func (*In) XXX_Size

```go
func (m *In) XXX_Size() int
```

#### func (*In) XXX_Unmarshal

```go
func (m *In) XXX_Unmarshal(b []byte) error
```

#### type Out

```go
type Out struct {
	Out                  int64    `protobuf:"varint,1,opt,name=out,proto3" json:"out,omitempty"`
	XXX_NoUnkeyedLiteral struct{} `json:"-"`
	XXX_unrecognized     []byte   `json:"-"`
	XXX_sizecache        int32    `json:"-"`
}
```


#### func (*Out) Descriptor

```go
func (*Out) Descriptor() ([]byte, []int)
```

#### func (*Out) GetOut

```go
func (m *Out) GetOut() int64
```

#### func (*Out) ProtoMessage

```go
func (*Out) ProtoMessage()
```

#### func (*Out) Reset

```go
func (m *Out) Reset()
```

#### func (*Out) String

```go
func (m *Out) String() string
```

#### func (*Out) XXX_DiscardUnknown

```go
func (m *Out) XXX_DiscardUnknown()
```

#### func (*Out) XXX_Marshal

```go
func (m *Out) XXX_Marshal(b []byte, deterministic bool) ([]byte, error)
```

#### func (*Out) XXX_Merge

```go
func (m *Out) XXX_Merge(src proto.Message)
```

#### func (*Out) XXX_Size

```go
func (m *Out) XXX_Size() int
```

#### func (*Out) XXX_Unmarshal

```go
func (m *Out) XXX_Unmarshal(b []byte) error
```
