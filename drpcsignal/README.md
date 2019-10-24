# package drpcsignal

`import "storj.io/drpc/drpcsignal"`

package drpcsignal holds a helper type to signal errors.

## Usage

#### type Signal

```go
type Signal struct {
}
```


#### func (*Signal) Err

```go
func (s *Signal) Err() error
```

#### func (*Signal) Get

```go
func (s *Signal) Get() (error, bool)
```

#### func (*Signal) IsSet

```go
func (s *Signal) IsSet() bool
```

#### func (*Signal) Set

```go
func (s *Signal) Set(err error) (ok bool)
```

#### func (*Signal) Signal

```go
func (s *Signal) Signal() chan struct{}
```
