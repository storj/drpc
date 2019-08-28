# package drpcctx

`import "storj.io/drpc/drpcctx"`

package drpcctx has helpers to interact with context.Context.

## Usage

#### func  Transport

```go
func Transport(ctx context.Context) (drpc.Transport, bool)
```

#### func  WithTransport

```go
func WithTransport(ctx context.Context, tr drpc.Transport) context.Context
```
