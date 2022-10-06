# package drpcpool

`import "storj.io/drpc/drpcpool"`

Package drpcpool is a simple connection pool for clients.

It has the ability to maintain a cache of connections with a maximum size on
both the total and per key basis. It also can expire cached connections if they
have been inactive in the pool for long enough.

Implementation note: the cache has some methods that could potentially be
quadratic in the worst case in the number of per cache key connections.
Specifically, this worst case happens when there are many closed entries in the
list of values. While we could do a single pass filtering closed entries, the
logic is a bit harder to follow and ensure is correct. Instead we have a helper
to remove a single entry from a list without knowing where it came from. Since
we can possibly call that to remove every element from a list if they are all
closed, it's quadratic in the maximum size of that list. Since this cache is
intended to be used with small key capacities (like 5), the decision was made to
accept that quadratic worst case for the benefit of having as simple an
implementation as possible.

## Usage

#### type Options

```go
type Options struct {
	// Expiration will remove any values from the Pool after the
	// value passes. Zero means no expiration.
	Expiration time.Duration

	// Capacity is the maximum number of values the Pool can store.
	// Zero means unlimited. Negative means no values.
	Capacity int

	// KeyCapacity is like Capacity except it is per key. Zero means
	// the Pool holds unlimited for any single key. Negative means
	// no values for any single key.
	//
	// Implementation note: The cache is potentially quadratic in the
	// size of this parameter, so it is intended for small values, like
	// 5 or so.
	KeyCapacity int
}
```

Options contains the options to configure a pool.

#### type Pool

```go
type Pool[K comparable] struct {
}
```

Pool is a connection pool with key type K. It maintains a cache of connections
per key and ensures the total number of connections in the cache is bounded by
configurable values. It does not limit the maximum concurrency of the number of
connections either in total or per key.

#### func  New

```go
func New[K comparable](opts Options) *Pool[K]
```
New constructs a new Pool with the provided Options.

#### func (*Pool[K]) Close

```go
func (p *Pool[K]) Close() (err error)
```
Close evicts all entries from the Pool's cache, closing them and returning all
of the combined errors from closing.

#### func (*Pool[K]) Get

```go
func (p *Pool[K]) Get(ctx context.Context, key K, dial func(ctx context.Context, key K) (drpc.Conn, error)) drpc.Conn
```
Get returns a new drpc.Conn that will use the provided dial function to create
an underlying conn to be cached by the Pool when Conn methods are invoked. It
will share any cached connections with other conns that use the same key.
