# package fuzz

`import "storj.io/drpc/internal/fuzz-drpcwire"`

Package fuzz is used to fuzz drpcwire frame parsing.

## Usage

#### func  Fuzz

```go
func Fuzz(data []byte) int
```
Fuzz takes in some data and attempts to parse it.
