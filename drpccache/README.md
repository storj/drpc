# package drpccache

`import "storj.io/drpc/drpccache"`

Package drpccache defines the per transport cache used by server.

## Usage

#### func  FromContext

```go
func FromContext(ctx context.Context) *drpccache.Cache
```

`FromContext` returns the cache associated with a context.

Example usage:

```
cache := drpccache.FromContext(stream.Context())
if cache != nil {
	value := cache.LoadOrCreate("initialized", func() (interface{}) {
		return 42
	})
}
```