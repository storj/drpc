# package drpcopts

`import "storj.io/drpc/internal/drpcopts"`

Package drpcopts contains internal options.

This package allows options to exist that are too sharp to provide to typical
users of the library that are not required to be backward compatible.

## Usage

#### func  GetManagerStatsCB

```go
func GetManagerStatsCB(opts *Manager) func(string) *drpcstats.Stats
```
GetManagerStatsCB returns the stats callback stored in the options.

#### func  GetStreamFin

```go
func GetStreamFin(opts *Stream) chan<- struct{}
```
GetStreamFin returns the chan<- struct{} stored in the options.

#### func  GetStreamKind

```go
func GetStreamKind(opts *Stream) string
```
GetStreamKind returns the kind debug string stored in the options.

#### func  GetStreamRPC

```go
func GetStreamRPC(opts *Stream) string
```
GetStreamRPC returns the RPC debug string stored in the options.

#### func  GetStreamStats

```go
func GetStreamStats(opts *Stream) *drpcstats.Stats
```
GetStreamStats returns the Stats stored in the options.

#### func  GetStreamTransport

```go
func GetStreamTransport(opts *Stream) drpc.Transport
```
GetStreamTransport returns the drpc.Transport stored in the options.

#### func  SetManagerStatsCB

```go
func SetManagerStatsCB(opts *Manager, statsCB func(string) *drpcstats.Stats)
```
SetManagerStatsCB sets the stats callback stored in the options.

#### func  SetStreamFin

```go
func SetStreamFin(opts *Stream, fin chan<- struct{})
```
SetStreamFin sets the chan<- struct{} stored in the options.

#### func  SetStreamKind

```go
func SetStreamKind(opts *Stream, kind string)
```
SetStreamKind sets the kind debug string stored in the options.

#### func  SetStreamRPC

```go
func SetStreamRPC(opts *Stream, rpc string)
```
SetStreamRPC sets the RPC debug string stored in the options.

#### func  SetStreamStats

```go
func SetStreamStats(opts *Stream, stats *drpcstats.Stats)
```
SetStreamStats sets the Stats stored in the options.

#### func  SetStreamTransport

```go
func SetStreamTransport(opts *Stream, tr drpc.Transport)
```
SetStreamTransport sets the drpc.Transport stored in the options.

#### type Manager

```go
type Manager struct {
}
```

Manager contains internal options for the drpcmanager package.

#### type Stream

```go
type Stream struct {
}
```

Stream contains internal options for the drpcstream package.
