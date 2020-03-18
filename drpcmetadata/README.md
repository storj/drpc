# package drpcmetadata

`import "storj.io/drpc/drpcmetadata"`

Package drpcmetadata define the structure of the metadata supported by drpc
library.

## Usage

#### func  Add

```go
func Add(ctx context.Context, key, value string) context.Context
```
Add associates a key/value pair on the context.

#### func  Decode

```go
func Decode(data []byte) (*ppb.InvokeMetadata, error)
```
Decode translate byte form of metadata into metadata struct defined by protobuf.

#### type Metadata

```go
type Metadata map[string]string
```

Metadata is a mapping from metadata key to value.

#### func  Get

```go
func Get(ctx context.Context) (Metadata, bool)
```
Get returns all key/value pairs on the given context.

#### func  New

```go
func New(data map[string]string) Metadata
```
New generates a new Metadata instance with keys set to lowercase.

#### func (Metadata) AddPairs

```go
func (md Metadata) AddPairs(ctx context.Context) context.Context
```
AddPairs attaches metadata onto a context and return the context.

#### func (Metadata) Encode

```go
func (md Metadata) Encode(buffer []byte) ([]byte, error)
```
Encode generates byte form of the metadata and appends it onto the passed in
buffer.
