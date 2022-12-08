# package drpctest

`import "storj.io/drpc/drpctest"`

Package drpctest provides test related helpers.

## Usage

#### type Tracker

```go
type Tracker struct {
	context.Context
}
```

Tracker keeps track of launched goroutines with a context.

#### func  NewTracker

```go
func NewTracker(tb testing.TB) *Tracker
```
NewTracker creates a new tracker that inspects the provided TB to see if tests
have failed in any of its launched goroutines.

#### func (*Tracker) Cancel

```go
func (t *Tracker) Cancel()
```
Cancel cancels the tracker's context.

#### func (*Tracker) Close

```go
func (t *Tracker) Close()
```
Close cancels the context and waits for all of the goroutines started by Run to
finish.

#### func (*Tracker) Run

```go
func (t *Tracker) Run(cb func(ctx context.Context))
```
Run starts a goroutine running the callback with the tracker as the context.

#### func (*Tracker) Wait

```go
func (t *Tracker) Wait()
```
Wait blocks until all callbacks started with Run have exited.
