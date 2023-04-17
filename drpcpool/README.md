# package drpcpool

`import "storj.io/drpc/drpcpool"`

Package drpcpool is a simple connection pool for clients.

It has the ability to maintain a cache of connections with a maximum size on
both the total and per key basis. It also can expire cached connections if they
have been inactive in the pool for long enough.

## Usage

#### type Conn

```go
type Conn interface {
	drpc.Conn

	// Unblocked returns a channel that is closed when the conn is available
	// for an Invoke or NewStream call.
	Unblocked() <-chan struct{}
}
```

Conn is the type of connections that can be managed by the pool. Connections
need to provide an Unblocked function that can be used by the pool to skip
connections that are still blocked on canceling the last RPC.

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
	KeyCapacity int
}
```

Options contains the options to configure a pool.

#### type Pool

```go
type Pool[K comparable, V Conn] struct {
}
```

Pool is a connection pool with key type K. It maintains a cache of connections
per key and ensures the total number of connections in the cache is bounded by
configurable values. It does not limit the maximum concurrency of the number of
connections either in total or per key.

#### func  New

```go
func New[K comparable, V Conn](opts Options) *Pool[K, V]
```
New constructs a new Pool with the provided Options.

#### func (*Pool[K, V]) Close

```go
func (p *Pool[K, V]) Close() (err error)
```
Close evicts all entries from the Pool's cache, closing them and returning all
of the combined errors from closing.

#### func (*Pool[K, V]) Get

```go
func (p *Pool[K, V]) Get(ctx context.Context, key K,
	dial func(ctx context.Context, key K) (V, error)) Conn
```
Get returns a new Conn that will use the provided dial function to create an
underlying conn to be cached by the Pool when Conn methods are invoked. It will
share any cached connections with other conns that use the same key.

#### func (*Pool[K, V]) Put

```go
func (p *Pool[K, V]) Put(key K, val V)
```
Put places the connection in to the cache with the provided key, ensuring that
the size limits the Pool is configured with are respected.

#### func (*Pool[K, V]) Take

```go
func (p *Pool[K, V]) Take(key K) (V, bool)
```
Take acquires a value from the cache if one exists. It returns the zero value
for V and false if one does not.
