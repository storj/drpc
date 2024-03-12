# package drpcstats

`import "storj.io/drpc/drpcstats"`

Package drpcstats contatins types for stat collection.

## Usage

#### type Stats

```go
type Stats struct {
	Read    uint64
	Written uint64
}
```

Stats keeps counters of read and written bytes.

#### func (*Stats) AddRead

```go
func (s *Stats) AddRead(n uint64)
```
AddRead atomically adds n bytes to the Read counter.

#### func (*Stats) AddWritten

```go
func (s *Stats) AddWritten(n uint64)
```
AddWritten atomically adds n bytes to the Written counter.

#### func (*Stats) AtomicClone

```go
func (s *Stats) AtomicClone() Stats
```
AtomicClone returns a copy of the stats that is safe to use concurrently with
Add methods.
